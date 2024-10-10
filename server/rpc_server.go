package server

import (
	"context"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/codec"
	"github.com/realmicro/realmicro/common/util/addr"
	"github.com/realmicro/realmicro/common/util/backoff"
	mnet "github.com/realmicro/realmicro/common/util/net"
	"github.com/realmicro/realmicro/common/util/socket"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/transport"
	"github.com/realmicro/realmicro/transport/headers"
)

type rpcServer struct {
	opts Options
	// subscribe to service name
	subscriber broker.Subscriber
	router     *router
	exit       chan chan error

	handlers    map[string]Handler
	subscribers map[Subscriber][]broker.Subscriber
	// graceful exit
	wg *sync.WaitGroup
	// Cached service
	rsvc *registry.Service

	sync.RWMutex
	// marks the serve as started
	started bool
	// used for first registration
	registered bool
}

// newRpcServer will create a new default RPC server.
func newRpcServer(opts ...Option) Server {
	options := NewOptions(opts...)
	r := newRpcRouter()
	r.hdlrWrappers = options.HdlrWrappers
	r.subWrappers = options.SubWrappers

	return &rpcServer{
		opts:        options,
		router:      r,
		handlers:    make(map[string]Handler),
		subscribers: make(map[Subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
		wg:          wait(options.Context),
	}
}

func (s *rpcServer) Init(opts ...Option) error {
	s.Lock()
	defer s.Unlock()

	for _, opt := range opts {
		opt(&s.opts)
	}

	// update router if it's the default
	if s.opts.Router == nil {
		r := newRpcRouter()
		r.hdlrWrappers = s.opts.HdlrWrappers
		r.serviceMap = s.router.serviceMap
		r.subWrappers = s.opts.SubWrappers
		s.router = r
	}

	s.rsvc = nil

	return nil
}

// ServeConn serves a single connection
func (s *rpcServer) ServeConn(sock transport.Socket) {
	// global error tracking
	var gerr error

	// Keep track of Connection: close header
	var closeConn bool

	// streams are multiplexed on Micro-Stream or Micro-Id header
	pool := socket.NewPool()

	// Waitgroup to wait for processing to finish
	// A double waitgroup is used to block the global waitgroup incase it is
	// empty, but only wait for the local routines to finish with the local waitgroup.
	wg := NewWaitGroup(s.getWg())

	defer func() {
		// Only wait if there's no error
		if gerr != nil {
			select {
			case <-s.exit:
			default:
				// EOF is expected if the client closes the connection
				if !errors.Is(gerr, io.EOF) {
					logger.Logf(logger.ErrorLevel, "error while serving connection: %v", gerr)
				}
			}
		} else {
			wg.Wait()
		}

		// close all the sockets for this connection
		pool.Close()

		// close underlying socket
		if err := sock.Close(); err != nil {
			logger.Logf(logger.ErrorLevel, "failed to close socket: %v", err)
		}

		// recover any panics
		if r := recover(); r != nil {
			logger.Error("panic recovered: ", r)
			logger.Error(string(debug.Stack()))
		}
	}()

	for {
		msg := transport.Message{
			Header: make(map[string]string),
		}

		// Close connection if Connection: close header was set
		if closeConn {
			return
		}

		// process inbound messages one at a time
		if err := sock.Recv(&msg); err != nil {
			// set a global error and return
			// we're saying we essentially can't
			// use the socket anymore
			gerr = errors.Wrapf(err, "%s-%s | %s", s.opts.Name, s.opts.Id, sock.Remote())

			return
		}

		// Keep track of when to close the connection
		if c := msg.Header["Connection"]; c == "close" {
			closeConn = true
		}

		// Check the message header for micro message header, if so handle
		// as micro event
		if t := msg.Header[headers.Message]; len(t) > 0 {
			// process the event
			ev := newEvent(msg)

			if err := s.HandleEvent(ev); err != nil {
				msg.Header[headers.Error] = err.Error()
				logger.Logf(logger.ErrorLevel, "failed to handle event: %v", err)
			}
			// write back some 200
			if err := sock.Send(&transport.Message{Header: msg.Header}); err != nil {
				gerr = err
				break
			}

			continue
		}

		// business as usual

		// use Micro-Stream as the stream identifier
		// in the event its blank we'll always process
		// on the same socket
		var (
			stream bool
			id     string
		)

		if v := getHeader(headers.Stream, msg.Header); len(v) > 0 {
			id = v
			stream = true
		} else {
			// if there's no stream id then it's a standard request
			// use the Micro-Id
			id = msg.Header[headers.ID]
		}

		// Check if we have an existing socket
		psock, ok := pool.Get(id)

		// if we don't have a socket, and it's a stream
		// check if it's a last stream EOS error
		if !ok && stream && msg.Header[headers.Error] == errLastStreamResponse.Error() {
			closeConn = true
			pool.Release(psock)

			continue
		}

		// got an existing socket already
		if ok {
			// we're starting processing
			wg.Add(1)

			// pass the message to that existing socket
			if err := psock.Accept(&msg); err != nil {
				// release the socket if there's an error
				pool.Release(psock)
			}

			wg.Done()

			continue
		}

		// No socket was found so its new
		// set the local and remote values
		psock.SetLocal(sock.Local())
		psock.SetRemote(sock.Remote())

		// Load the socket with the current message
		if err := psock.Accept(&msg); err != nil {
			logger.Logf(logger.ErrorLevel, "Socket failed to accept message: %v", err)
		}

		// Now walk the usual path

		// We use this Timeout header to set a server deadline
		to := msg.Header["Timeout"]
		// We use this Content-Type header to identify the codec needed
		contentType := msg.Header["Content-Type"]

		// Copy the message headers
		header := make(map[string]string, len(msg.Header))
		for k, v := range msg.Header {
			header[k] = v
		}

		// Set local/remote ips
		header["Local"] = sock.Local()
		header["Remote"] = sock.Remote()

		// Create new context with the metadata
		ctx := metadata.NewContext(context.Background(), header)

		// Set the timeout from the header if we have it
		if len(to) > 0 {
			if n, err := strconv.ParseUint(to, 10, 64); err == nil {
				var cancel context.CancelFunc

				ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
				defer cancel()
			}
		}

		// If there's no content type default it
		if len(contentType) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			contentType = DefaultContentType
		}

		// Setup old protocol
		cf := setupProtocol(&msg)

		// No legacy codec needed
		if cf == nil {
			var err error
			// Try to get a new codec
			if cf, err = s.newCodec(contentType); err != nil {
				// No codec found so send back an error
				if err := sock.Send(&transport.Message{
					Header: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: []byte(err.Error()),
				}); err != nil {
					gerr = err
				}

				pool.Release(psock)

				continue
			}
		}

		// Create a new rpc codec based on the pseudo socket and codec
		rcodec := newRpcCodec(&msg, psock, cf)
		// Check the protocol as well
		protocol := rcodec.String()

		// Internal request
		request := rpcRequest{
			service:     getHeader(headers.Request, msg.Header),
			method:      getHeader(headers.Method, msg.Header),
			endpoint:    getHeader(headers.Endpoint, msg.Header),
			contentType: contentType,
			codec:       rcodec,
			header:      msg.Header,
			body:        msg.Body,
			socket:      psock,
			stream:      stream,
		}

		// internal response
		response := rpcResponse{
			header: make(map[string]string),
			socket: psock,
			codec:  rcodec,
		}

		// Wait for two coroutines to exit
		// Serve the request and process the outbound messages
		wg.Add(2)

		// Process the outbound messages from the socket
		go func(psock *socket.Socket) {
			defer func() {
				// TODO: don't hack this but if its grpc just break out of the stream
				// We do this because the underlying connection is h2 and its a stream
				if protocol == "grpc" {
					if err := sock.Close(); err != nil {
						logger.Logf(logger.ErrorLevel, "Failed to close socket: %v", err)
					}
				}

				s.deferer(pool, psock, wg)
			}()

			for {
				// Get the message from our internal handler/stream
				m := new(transport.Message)
				if err := psock.Process(m); err != nil {
					return
				}

				// Send the message back over the socket
				if err := sock.Send(m); err != nil {
					return
				}
			}
		}(psock)

		// serve the request in a go routine as this may be a stream
		go func(psock *socket.Socket) {
			defer s.deferer(pool, psock, wg)

			s.serveReq(ctx, msg, &request, &response, rcodec)
		}(psock)
	}
}

func (s *rpcServer) NewHandler(h interface{}, opts ...HandlerOption) Handler {
	return s.router.NewHandler(h, opts...)
}

func (s *rpcServer) Handle(h Handler) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Handle(h); err != nil {
		return err
	}

	s.handlers[h.Name()] = h

	return nil
}

func (s *rpcServer) Register() error {
	config := s.Options()

	// Registry function used to register the service
	regFunc := s.newRegFuc(config)

	// Directly register if service was cached
	rsvc := s.getCachedService()
	if rsvc != nil {
		if err := regFunc(rsvc); err != nil {
			return errors.Wrap(err, "failed to register service")
		}

		return nil
	}

	// Only cache service if host IP valid
	addr, cacheService, err := s.getAddr(config)
	if err != nil {
		return err
	}

	// register service
	node := &registry.Node{
		// TODO: node id should be set better. Add native option to specify
		// host id through either config or ENV. Also look at logging of name.
		Id:       config.Name + "-" + config.Id,
		Address:  addr,
		Metadata: s.newNodeMetadata(config),
	}

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: s.getEndpoints(),
	}

	// get registered value
	registered := s.isRegistered()
	if !registered {
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("Registry [%s] Registering node: %s", config.Registry.String(), node.Id)
		}
	}

	// register the service
	if err := regFunc(service); err != nil {
		return errors.Wrap(err, "failed to register service")
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	s.registered = true

	// Cache service
	if cacheService {
		s.rsvc = service
	}

	// set what we're advertising
	s.opts.Advertise = addr

	// Router can exchange messages on broker
	// Subscribe to the topic with its own name
	if err := s.subscribeServer(config); err != nil {
		return errors.Wrap(err, "failed to subscribe to service name topic")
	}

	// Subscribe for all the subscribers
	if err := s.reSubscribe(config); err != nil {
		return errors.Wrap(err, "failed to resubscribe")
	}

	return nil
}

func (s *rpcServer) Deregister() error {
	var err error

	config := s.Options()

	addr, _, err := s.getAddr(config)
	if err != nil {
		return err
	}

	node := &registry.Node{
		Id:      config.Name + "-" + config.Id,
		Address: addr,
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		logger.Infof("Registry [%s] Deregistering node: %s", config.Registry.String(), node.Id)
	}
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	s.rsvc = nil

	if !s.registered {
		return nil
	}

	s.registered = false

	// close the subscriber
	if s.subscriber != nil {
		if err := s.subscriber.Unsubscribe(); err != nil {
			logger.Logf(logger.ErrorLevel, "Failed to unsubscribe service from service name topic: %v", err)
		}

		s.subscriber = nil
	}

	for sb, subs := range s.subscribers {
		for i, sub := range subs {
			if logger.V(logger.InfoLevel, logger.DefaultLogger) {
				logger.Infof("Unsubscribing %s from topic: %s", node.Id, sub.Topic())
			}
			if err := sub.Unsubscribe(); err != nil {
				logger.Logf(logger.ErrorLevel, "Failed to unsubscribe subscriber nr. %d from topic %s: %v", i+1, sub.Topic(), err)
			}
		}

		s.subscribers[sb] = nil
	}

	return nil
}

func (s *rpcServer) Start() error {
	if s.isStarted() {
		return nil
	}

	config := s.Options()

	// start listening on the listener
	listener, err := config.Transport.Listen(config.Address)
	if err != nil {
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		logger.Infof("Transport [%s] Listening on %s", config.Transport.String(), listener.Addr())
	}

	// swap address
	addr := s.swapAddr(config, listener.Addr())

	// connect to the broker
	brokerName := config.Broker.String()
	if err := config.Broker.Connect(); err != nil {
		logger.Errorf("Broker [%s] connect error: %v", brokerName, err)
		return err
	}

	logger.Infof("Broker [%s] Connected to %s", brokerName, config.Broker.Address())

	// use RegisterCheck func before register
	if err = s.opts.RegisterCheck(s.opts.Context); err != nil {
		logger.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, err)
	} else {
		// announce self to the world
		if err = s.Register(); err != nil {
			logger.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
		}
	}

	exit := make(chan bool)

	go s.listen(listener, exit)

	go s.registrar(listener, addr, config, exit)

	s.setStarted(true)

	return nil
}

func (s *rpcServer) Stop() error {
	if !s.isStarted() {
		return nil
	}

	ch := make(chan error)
	s.exit <- ch

	err := <-ch

	s.setStarted(false)

	return err
}

func (s *rpcServer) String() string {
	return "mucp"
}

// newRegFuc will create a new registry function used to register the service.
func (s *rpcServer) newRegFuc(config Options) func(service *registry.Service) error {
	return func(service *registry.Service) error {
		rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}

		var regErr error

		// Attempt to register. If registration fails, back off and try again.
		// TODO: see if we can improve the retry mechanism. Maybe retry lib, maybe config values
		for i := 0; i < 3; i++ {
			if err := config.Registry.Register(service, rOpts...); err != nil {
				regErr = err

				time.Sleep(backoff.Do(i + 1))

				continue
			}

			return nil
		}

		return regErr
	}
}

// getAddr will take the advertise or service address, and return it.
func (s *rpcServer) getAddr(config Options) (string, bool, error) {
	// Use advertise address if provided, else use service address
	advt := config.Address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	}

	// Use explicit host and port if possible
	host, port := advt, ""

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		h, p, err := net.SplitHostPort(advt)
		if err != nil {
			return "", false, err
		}

		host, port = h, p
	}

	validHost := net.ParseIP(host) != nil

	addr, err := addr.Extract(host)
	if err != nil {
		return "", false, err
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	return addr, validHost, nil
}

// newNodeMetadata creates a new metadata map with default values.
func (s *rpcServer) newNodeMetadata(config Options) metadata.Metadata {
	md := metadata.Copy(config.Metadata)

	md["transport"] = config.Transport.String()
	md["broker"] = config.Broker.String()
	md["server"] = s.String()
	md["registry"] = config.Registry.String()
	md["protocol"] = "mucp"
	md["startup"] = time.Now().Format("2006-01-02 15:04:05")

	return md
}

// getEndpoints takes the list of handlers and subscribers and adds them to
// a single endpoints list.
func (s *rpcServer) getEndpoints() []*registry.Endpoint {
	s.RLock()
	defer s.RUnlock()

	var handlerList []string

	for n, e := range s.handlers {
		// Only advertise non-internal handlers
		if !e.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}

	// Maps are ordered randomly, sort the keys for consistency
	// TODO: replace with generic version
	sort.Strings(handlerList)

	var subscriberList []Subscriber

	for e := range s.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			subscriberList = append(subscriberList, e)
		}
	}

	sort.Slice(subscriberList, func(i, j int) bool {
		return subscriberList[i].Topic() > subscriberList[j].Topic()
	})

	endpoints := make([]*registry.Endpoint, 0, len(handlerList)+len(subscriberList))

	for _, n := range handlerList {
		endpoints = append(endpoints, s.handlers[n].Endpoints()...)
	}

	for _, e := range subscriberList {
		endpoints = append(endpoints, e.Endpoints()...)
	}

	return endpoints
}

func (s *rpcServer) listen(listener transport.Listener, exit chan bool) {
	for {
		// listen for connections
		err := listener.Accept(s.ServeConn)

		// TODO: listen for messages
		// msg := broker.Exchange(service).Consume()

		select {
		// check if we're supposed to exit
		case <-exit:
			return
		// check the error and backoff
		default:
			if err != nil {
				logger.Errorf("Accept error: %v", err)
				time.Sleep(time.Second)

				continue
			}
		}

		return
	}
}

// registrar is responsible for keeping the service registered to the registry.
func (s *rpcServer) registrar(listener transport.Listener, addr string, config Options, exit chan bool) {
	// only process if it exists
	ticker := new(time.Ticker)
	if s.opts.RegisterInterval > time.Duration(0) {
		ticker = time.NewTicker(s.opts.RegisterInterval)
	}

	// return error chan
	var ch chan error

Loop:
	for {
		select {
		// register self on interval
		case <-ticker.C:
			registered := s.isRegistered()

			rerr := s.opts.RegisterCheck(s.opts.Context)
			if rerr != nil && registered {
				logger.Errorf("Server %s-%s register check error: %s, deregister it", config.Name, config.Id, rerr)
				// deregister self in case of error
				if err := s.Deregister(); err != nil {
					logger.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
				}
			} else if rerr != nil && !registered {
				logger.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, rerr)
				continue
			}

			if err := s.Register(); err != nil {
				logger.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
			}

		// wait for exit
		case ch = <-s.exit:
			ticker.Stop()
			close(exit)

			break Loop
		}
	}

	// Shutting down, deregister
	if s.isRegistered() {
		if err := s.Deregister(); err != nil {
			logger.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
		}
	}

	// wait for requests to finish
	if swg := s.getWg(); swg != nil {
		swg.Wait()
	}

	// close transport listener
	ch <- listener.Close()

	brokerName := config.Broker.String()
	logger.Infof("Broker [%s] Disconnected from %s", brokerName, config.Broker.Address())

	// disconnect the broker
	if err := config.Broker.Disconnect(); err != nil {
		logger.Errorf("Broker [%s] Disconnect error: %v", brokerName, err)
	}

	// Swap back address
	s.setOptsAddr(addr)
}

func (s *rpcServer) serveReq(ctx context.Context,
	msg transport.Message,
	req *rpcRequest,
	resp *rpcResponse,
	rcodec codec.Codec,
) {
	router := s.getRouter()

	// serve the actual request using the request router
	if serveRequestError := router.ServeRequest(ctx, req, resp); serveRequestError != nil {
		// write an error response
		writeError := rcodec.Write(&codec.Message{
			Header: msg.Header,
			Error:  serveRequestError.Error(),
			Type:   codec.Error,
		}, nil)

		// if the server request is an EOS error we let the socket know
		// sometimes the socket is already closed on the other side, so we can ignore that error
		alreadyClosed := errors.Is(serveRequestError, errLastStreamResponse) && errors.Is(writeError, io.EOF)

		// could not write error response
		if writeError != nil && !alreadyClosed {
			logger.Debugf("rpc: unable to write error response: %v", writeError)
		}
	}
}

func (s *rpcServer) deferer(pool *socket.Pool, psock *socket.Socket, wg *waitGroup) {
	// release the socket
	pool.Release(psock)
	// signal we're done
	wg.Done()

	// recover any panics for call handler
	if r := recover(); r != nil {
		logger.Error("panic recovered: ", r)
		logger.Error(string(debug.Stack()))
	}
}

func (s *rpcServer) getRouter() Router {
	router := Router(s.router)

	// if not nil use the router specified
	if s.opts.Router != nil {
		// create a wrapped function
		handler := func(ctx context.Context, req Request, rsp interface{}) error {
			return s.opts.Router.ServeRequest(ctx, req, rsp.(Response))
		}

		// execute the wrapper for it
		for i := len(s.opts.HdlrWrappers); i > 0; i-- {
			handler = s.opts.HdlrWrappers[i-1](handler)
		}

		// set the router
		router = rpcRouter{h: handler}
	}

	return router
}
