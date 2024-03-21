package redis

import (
	"context"
	"errors"

	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/codec"
	"github.com/realmicro/realmicro/codec/json"
	"github.com/realmicro/realmicro/common/stores/redis"
	red "github.com/redis/go-redis/v9"
)

// publication is an internal publication for the Redis broker.
type publication struct {
	topic   string
	message *broker.Message
	err     error
}

// Topic returns the topic this publication applies to.
func (p *publication) Topic() string {
	return p.topic
}

// Message returns the broker message of the publication.
func (p *publication) Message() *broker.Message {
	return p.message
}

// Ack sends an acknowledgement to the broker. However this is not supported
// is Redis and therefore this is a no-op.
func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

// subscriber proxies and handles Redis messages as broker publications.
type subscriber struct {
	codec   codec.Marshaler
	sub     *red.PubSub
	topic   string
	handler broker.Handler
	opts    broker.SubscribeOptions
}

// recv loops to receive new messages from Redis and handler them
// as publications.
func (s *subscriber) recv() {
	defer s.sub.Close()

	_, err := s.sub.Receive(context.Background())
	if err != nil {
		return
	}

	ch := s.sub.Channel()
	for msg := range ch {
		var m broker.Message
		if err = s.codec.Unmarshal([]byte(msg.Payload), &m); err != nil {
			break
		}
		p := &publication{
			topic:   msg.Channel,
			message: &m,
		}
		if p.err = s.handler(p); p.err != nil {
			break
		}

		if s.opts.AutoAck {
			if err = p.Ack(); err != nil {
				break
			}
		}
	}
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
	return s.sub.Unsubscribe(context.Background())
}

type redisBroker struct {
	client *redis.Redis
	opts   broker.Options
	bOpts  *brokerOptions
}

// String returns the name of the broker implementation.
func (b *redisBroker) String() string {
	return "redis"
}

// Options returns the options defined for the broker.
func (b *redisBroker) Options() broker.Options {
	return b.opts
}

// Address returns the address the broker will use to create new connections.
// This will be set only after Connect is called.
func (b *redisBroker) Address() string {
	return ""
}

// Init sets or overrides broker options.
func (b *redisBroker) Init(opts ...broker.Option) error {
	if b.client != nil {
		return errors.New("redis: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

// Connect establishes a connection to Redis which provides the
// pub/sub implementation.
func (b *redisBroker) Connect() error {
	if b.client != nil {
		return nil
	}

	if len(b.opts.Addrs) == 0 {
		b.opts.Addrs = []string{"127.0.0.1:6379"}
	}

	b.client = redis.New(
		redis.Addrs(b.opts.Addrs),
		redis.Pass(b.bOpts.password),
		redis.DB(b.bOpts.db),
	)

	return nil
}

func (b *redisBroker) Disconnect() error {
	b.client = nil
	return nil
}

// Publish publishes a message.
func (b *redisBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	return b.client.Publish(context.Background(), topic, v)
}

// Subscribe returns a subscriber for the topic and handler.
func (b *redisBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	sub, err := b.client.Subscribe(context.Background(), topic)
	if err != nil {
		return nil, err
	}

	s := subscriber{
		codec:   b.opts.Codec,
		sub:     sub,
		topic:   topic,
		handler: handler,
		opts:    options,
	}

	// Run the receiver routine.
	go s.recv()

	return &s, nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	bOpts := &brokerOptions{}

	// Initialize with empty broker options.
	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bOpts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &redisBroker{
		opts:  options,
		bOpts: bOpts,
	}
}
