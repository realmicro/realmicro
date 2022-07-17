package asynq

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/codec/json"
	"github.com/realmicro/realmicro/logger"
)

var (
	DefaultPath = "mbroker:"
)

// publication is an internal publication for the Asynq broker.
type publication struct {
	topic   string
	opr     string
	message *broker.Message
	err     error
}

// Topic returns the topic this publication applies to.
func (p *publication) Topic() string {
	if len(p.opr) == 0 {
		return p.topic
	}
	return p.topic + ":" + p.opr
}

// Message returns the broker message of the publication.
func (p *publication) Message() *broker.Message {
	return p.message
}

// Ack sends an acknowledgement to the broker. However, this is not supported
// in Asynq and therefore this is a no-op.
func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

type subscriber struct {
	opts    broker.SubscribeOptions
	topic   string
	opr     string
	handler broker.Handler
	b       *asynqBroker
}

// Options returns the subscriber options.
func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes the subscriber and frees the connection.
func (s *subscriber) Unsubscribe() error {
	return s.b.unsubscribe(s)
}

type asynqBroker struct {
	opts      broker.Options
	bOpts     *brokerOptions
	client    *asynq.Client
	server    *asynq.Server
	inspector *asynq.Inspector

	sync.RWMutex
	subscribers map[string]map[string]*subscriber
}

// String returns the name of the broker implementation.
func (b *asynqBroker) String() string {
	return "asynq"
}

// Options returns the options defined for the broker.
func (b *asynqBroker) Options() broker.Options {
	return b.opts
}

// Address returns the address the broker will use to create new connections.
// This will be set only after Connect is called.
func (b *asynqBroker) Address() string {
	return strings.Join(b.opts.Addrs, ",")
}

// Init sets or overrides broker options.
func (b *asynqBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

func (b *asynqBroker) Connect() error {
	if b.client != nil {
		return nil
	}

	if len(b.opts.Addrs) == 0 {
		b.opts.Addrs = []string{"127.0.0.1:6379"}
	}

	var tlsConfig *tls.Config
	if b.bOpts.tls {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	if b.bOpts.nodeType == NodeType {
		redisOpt := &asynq.RedisClientOpt{
			Addr:      b.opts.Addrs[0],
			Password:  b.bOpts.pass,
			DB:        b.bOpts.db,
			TLSConfig: tlsConfig,
		}
		// redis node mode
		b.client = asynq.NewClient(redisOpt)
		b.inspector = asynq.NewInspector(redisOpt)
	} else {
		redisOpt := &asynq.RedisClusterClientOpt{
			Addrs:     b.opts.Addrs,
			Password:  b.bOpts.pass,
			TLSConfig: tlsConfig,
		}
		// redis cluster node
		b.client = asynq.NewClient(redisOpt)
		b.inspector = asynq.NewInspector(redisOpt)
	}
	b.opts.Inspector = b

	return nil
}

func (b *asynqBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	err := b.client.Close()
	b.client = nil
	if b.server != nil {
		b.server.Shutdown()
		b.server = nil
	}
	err = b.inspector.Close()
	b.inspector = nil
	return err
}

func (b *asynqBroker) Publish(topic string, m *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(m)
	if err != nil {
		return err
	}

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	pOpts := &publishOptions{}
	if options.Context != nil {
		if pv, ok := options.Context.Value(publishKey).(*publishOptions); ok {
			pOpts = pv
		}
	}

	typename := fmt.Sprintf("%s%s:%s", DefaultPath, topic, pOpts.Opr)
	task := asynq.NewTask(typename, v)
	var taskOpts []asynq.Option
	if len(pOpts.Queue) > 0 {
		taskOpts = append(taskOpts, asynq.Queue(pOpts.Queue))
	}
	if pOpts.ProcessIn > 0 {
		taskOpts = append(taskOpts, asynq.ProcessIn(pOpts.ProcessIn))
	}
	if pOpts.Retention > 0 {
		taskOpts = append(taskOpts, asynq.Retention(pOpts.Retention))
	}
	info, err := b.client.Enqueue(task, taskOpts...)
	if err != nil {
		return err
	}
	m.MsgId = info.ID
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Asynq Broker Publish task result: %+v", info)
	}

	return nil
}

func (b *asynqBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	sOpts := &subscribeOptions{}
	options := broker.SubscribeOptions{
		Context: context.WithValue(context.Background(), subscribeKey, sOpts),
	}
	for _, o := range opts {
		o(&options)
	}

	if err := b.startServer(); err != nil {
		return nil, err
	}

	s := &subscriber{
		opts:    options,
		topic:   topic,
		opr:     sOpts.Opr,
		handler: h,
		b:       b,
	}

	// subscribe now
	if err := b.subscribe(s); err != nil {
		return nil, err
	}

	// return the subscriber
	return s, nil
}

func (b *asynqBroker) DeleteTask(queue, id string) error {
	if b.inspector != nil {
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("Asynq Broker DeleteTask: %s, %s", queue, id)
		}
		return b.inspector.DeleteTask(queue, id)
	}
	return nil
}

func (b *asynqBroker) startServer() error {
	b.Lock()
	defer b.Unlock()

	if b.server != nil {
		return nil
	}

	var tlsConfig *tls.Config
	if b.bOpts.tls {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	asynqCfg := asynq.Config{
		LogLevel: asynq.ErrorLevel,
	}
	if b.bOpts.queues != nil && len(b.bOpts.queues) > 0 {
		asynqCfg.Queues = b.bOpts.queues
	}
	if b.bOpts.nodeType == NodeType {
		redisOpt := &asynq.RedisClientOpt{
			Addr:      b.opts.Addrs[0],
			Password:  b.bOpts.pass,
			DB:        b.bOpts.db,
			TLSConfig: tlsConfig,
		}
		// redis node mode
		b.server = asynq.NewServer(redisOpt, asynqCfg)
	} else {
		redisOpt := &asynq.RedisClusterClientOpt{
			Addrs:     b.opts.Addrs,
			Password:  b.bOpts.pass,
			TLSConfig: tlsConfig,
		}
		// redis cluster node
		b.server = asynq.NewServer(redisOpt, asynqCfg)
	}

	mux := asynq.NewServeMux()
	mux.Handle(DefaultPath, asynq.HandlerFunc(b.processTask))

	go func() {
		if err := b.server.Run(mux); err != nil {
			logger.Fatal(err)
		}
	}()

	return nil
}

func (b *asynqBroker) subscribe(s *subscriber) error {
	b.Lock()
	defer b.Unlock()

	if ts, ok := b.subscribers[s.topic]; ok {
		ts[s.opr] = s
	} else {
		b.subscribers[s.topic] = make(map[string]*subscriber)
		b.subscribers[s.topic][s.opr] = s
	}
	return nil
}

func (b *asynqBroker) unsubscribe(s *subscriber) error {
	b.Lock()
	defer b.Unlock()

	if ts, ok := b.subscribers[s.topic]; ok {
		delete(ts, s.opr)
		if len(ts) == 0 {
			delete(b.subscribers, s.topic)
		}
	}
	return nil
}

func (b *asynqBroker) processTask(ctx context.Context, t *asynq.Task) error {
	ts := strings.Split(t.Type(), ":")
	if len(ts) != 3 {
		return fmt.Errorf("error type: %v", t.Type())
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("async task topic: %s, opr: %s, payload: %s", ts[1], ts[2], t.Payload())
	}

	var m broker.Message
	if err := b.opts.Codec.Unmarshal(t.Payload(), &m); err != nil {
		logger.Errorf("Asynq Broker Process Task Codec.Unmarshal error: %v", err)
		return err
	}
	p := &publication{
		topic:   ts[1],
		opr:     ts[2],
		message: &m,
	}

	var s *subscriber
	b.RLock()
	if topicSub, ok := b.subscribers[ts[1]]; ok {
		s = topicSub[ts[2]]
	}
	b.RUnlock()

	if s != nil {
		if p.err = s.handler(p); p.err != nil {
			return p.err
		}
	}

	return nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	bOpts := &brokerOptions{
		nodeType: NodeType,
		db:       DefaultDB,
	}

	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bOpts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &asynqBroker{
		opts:        options,
		bOpts:       bOpts,
		subscribers: make(map[string]map[string]*subscriber),
	}
}
