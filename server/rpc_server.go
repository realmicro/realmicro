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
	raw "github.com/realmicro/realmicro/codec/bytes"
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
	router *router
	exit   chan chan error

	sync.RWMutex
	opts        Options
	handlers    map[string]Handler
	subscribers map[Subscriber][]broker.Subscriber
	// marks the serve as started
	started bool
	// used for first registration
	registered bool
	// subscribe to service name
	subscriber broker.Subscriber
	// graceful exit
	wg *sync.WaitGroup

	rsvc *registry.Service
}

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

// HandleEvent handles inbound messages to the service directly
// TODO: handle requests from an event. We won't send a response.
func (s *rpcServer) HandleEvent(e broker.Event) error {
	// formatting horrible cruft
	msg := e.Message()

	if msg.Header == nil {
		// create empty map in case of headers empty to avoid panic later
		msg.Header = make(map[string]string)
	}

	// get codec
	ct := msg.Header["Content-Type"]

	// default content type
	if len(ct) == 0 {
		msg.Header["Content-Type"] = DefaultContentType
		ct = DefaultContentType
	}

	// get codec
	cf, err := s.newCodec(ct)
	if err != nil {
		return err
	}

	// copy headers
	hdr := make(map[string]string, len(msg.Header))
	for k, v := range msg.Header {
		hdr[k] = v
	}

	// create context
	ctx := metadata.NewContext(context.Background(), hdr)

	// TODO: inspect message header
	// Micro-Service means a request
	// Micro-Topic means a message

	rpcMsg := &rpcMessage{
		topic:       msg.Header[headers.Message],
		contentType: ct,
		payload:     &raw.Frame{Data: msg.Body},
		codec:       cf,
		header:      msg.Header,
		body:        msg.Body,
	}

	// existing router
	r := Router(s.router)

	// if the router is present then execute it
	if s.opts.Router != nil {
		// create a wrapped function
		handler := s.opts.Router.ProcessMessage

		// execute the wrapper for it
		for i := len(s.opts.SubWrappers); i > 0; i-- {
			handler = s.opts.SubWrappers[i-1](handler)
		}

		// set the router
		r = rpcRouter{m: handler}
	}

	return r.ProcessMessage(ctx, rpcMsg)
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
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error("panic recovered: ", r)
				logger.Error(string(debug.Stack()))
			}
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
			if err := sock.Send(&transport.Message{
				Header: msg.Header,
			}); err != nil {
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

		// check if we have an existing socket
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

		// no socket was found so its new
		// set the local and remote values
		psock.SetLocal(sock.Local())
		psock.SetRemote(sock.Remote())

		// load the socket with the current message
		if err := psock.Accept(&msg); err != nil {
			logger.Logf(logger.ErrorLevel, "Socket failed to accept message: %v", err)
		}

		// now walk the usual path

		// we use this Timeout header to set a server deadline
		to := msg.Header["Timeout"]
		// we use this Content-Type header to identify the codec needed
		contentType := msg.Header["Content-Type"]

		// copy the message headers
		hdr := make(map[string]string, len(msg.Header))
		for k, v := range msg.Header {
			hdr[k] = v
		}

		// set local/remote ips
		hdr["Local"] = sock.Local()
		hdr["Remote"] = sock.Remote()

		// create new context with the metadata
		ctx := metadata.NewContext(context.Background(), hdr)

		// set the timeout from the header if we have it
		if len(to) > 0 {
			if n, err := strconv.ParseUint(to, 10, 64); err == nil {
				var cancel context.CancelFunc

				ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
				defer cancel()
			}
		}

		// if there's no content type default it
		if len(contentType) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			contentType = DefaultContentType
		}

		// setup old protocol
		cf := setupProtocol(&msg)

		// no legacy codec needed
		if cf == nil {
			var err error
			// try to get a new codec
			if cf, err = s.newCodec(contentType); err != nil {
				// no codec found so send back an error
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

		// create a new rpc codec based on the pseudo socket and codec
		rcodec := newRpcCodec(&msg, psock, cf)
		// check the protocol as well
		protocol := rcodec.String()

		// internal request
		request := &rpcRequest{
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
		response := &rpcResponse{
			header: make(map[string]string),
			socket: psock,
			codec:  rcodec,
		}

		// set router
		r := Router(s.router)

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
			r = rpcRouter{h: handler}
		}

		// wait for two coroutines to exit
		// serve the request and process the outbound messages
		wg.Add(2)

		// process the outbound messages from the socket
		go func(psock *socket.Socket) {
			defer func() {
				// TODO: don't hack this but if its grpc just break out of the stream
				// We do this because the underlying connection is h2 and its a stream
				switch protocol {
				case "grpc":
					if err := sock.Close(); err != nil {
						logger.Logf(logger.ErrorLevel, "Failed to close socket: %v", err)
					}
				}
				// release the socket
				pool.Release(psock)
				// signal we're done
				wg.Done()

				// recover any panics for outbound process
				if r := recover(); r != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Error("panic recovered: ", r)
						logger.Error(string(debug.Stack()))
					}
				}
			}()

			for {
				// get the message from our internal handler/stream
				m := new(transport.Message)
				if err := psock.Process(m); err != nil {
					return
				}

				// send the message back over the socket
				if err := sock.Send(m); err != nil {
					return
				}
			}
		}(psock)

		// serve the request in a go routine as this may be a stream
		go func(psock *socket.Socket) {
			defer func() {
				// release the socket
				pool.Release(psock)
				// signal we're done
				wg.Done()

				// recover any panics for call handler
				if r := recover(); r != nil {
					logger.Error("panic recovered: ", r)
					logger.Error(string(debug.Stack()))
				}
			}()

			// serve the actual request using the request router
			if serveRequestError := r.ServeRequest(ctx, request, response); serveRequestError != nil {
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
		}(psock)
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

func (s *rpcServer) NewSubscriber(topic string, sb interface{}, opts ...SubscriberOption) Subscriber {
	return s.router.NewSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Subscribe(sb); err != nil {
		return err
	}

	s.subscribers[sb] = nil
	return nil
}

func (s *rpcServer) Register() error {
	config := s.Options()

	rsvc := s.getCachedService()

	regFunc := func(service *registry.Service) error {
		// create registry options
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

	// Directly register if service was cached
	if rsvc != nil {
		if err := regFunc(rsvc); err != nil {
			return errors.Wrap(err, "failed to register service")
		}

		return nil
	}

	var err error
	var advt, host, port string
	var cacheService bool

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	if ip := net.ParseIP(host); ip != nil {
		cacheService = true
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	// register service
	node := &registry.Node{
		// TODO: node id should be set better. Add native option to specify
		// host id through either config or ENV. Also look at logging of name.
		Id:       config.Name + "-" + config.Id,
		Address:  addr,
		Metadata: s.newNodeMetadata(config),
	}

	s.RLock()

	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, e := range s.handlers {
		// Only advertise non-internal handlers
		if !e.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}

	sort.Strings(handlerList)

	var subscriberList []Subscriber
	for e := range s.subscribers {
		// Only advertise non-internal subscribers
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

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	// get registered value
	registered := s.registered

	s.RUnlock()

	if !registered {
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("Registry [%s] Registering node: %s", config.Registry.String(), node.Id)
		}
	}

	// register the service
	if err := regFunc(service); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	// set what we're advertising
	s.opts.Advertise = addr

	// router can exchange messages
	if s.opts.Router != nil {
		// subscribe to the topic with own name
		sub, err := s.opts.Broker.Subscribe(config.Name, s.HandleEvent)
		if err != nil {
			return err
		}

		// save the subscriber
		s.subscriber = sub
	}

	// subscribe for all the subscribers
	for sb := range s.subscribers {
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if cx := sb.Options().Context; cx != nil {
			opts = append(opts, broker.SubscribeContext(cx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("Subscribing to topic: %s", sb.Topic())
		}
		sub, err := config.Broker.Subscribe(sb.Topic(), s.HandleEvent, opts...)
		if err != nil {
			return err
		}

		s.subscribers[sb] = []broker.Subscriber{sub}
	}
	if cacheService {
		s.rsvc = service
	}
	s.registered = true

	return nil
}

func (s *rpcServer) Deregister() error {
	var err error
	var advt, host, port string

	config := s.Options()

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
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
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Broker [%s] connect error: %v", brokerName, err)
		}
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		logger.Infof("Broker [%s] Connected to %s", brokerName, config.Broker.Address())
	}

	// use RegisterCheck func before register
	if err = s.opts.RegisterCheck(s.opts.Context); err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, err)
		}
	} else {
		// announce self to the world
		if err = s.Register(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
			}
		}
	}

	exit := make(chan bool)

	go func() {
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
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("Accept error: %v", err)
					}
					time.Sleep(time.Second)

					continue
				}
			}

			// no error just exit
			return
		}
	}()

	go func() {
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
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("Server %s-%s register check error: %s, deregister it", config.Name, config.Id, err)
					}
					// deregister self in case of error
					if err := s.Deregister(); err != nil {
						if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
							logger.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
						}
					}
				} else if rerr != nil && !registered {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, err)
					}
					continue
				}

				if err := s.Register(); err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
					}
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
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
				}
			}
		}

		// wait for requests to finish
		if swg := s.getWg(); swg != nil {
			swg.Wait()
		}

		// close transport listener
		ch <- listener.Close()

		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("Broker [%s] Disconnected from %s", brokerName, config.Broker.Address())
		}
		// disconnect the broker
		if err := config.Broker.Disconnect(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("Broker [%s] Disconnect error: %v", brokerName, err)
			}
		}

		// Swap back address
		s.setOptsAddr(addr)
	}()

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
