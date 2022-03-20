package asynq

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/codec/json"
	"github.com/realmicro/realmicro/logger"
)

type asynqBroker struct {
	opts   broker.Options
	bOpts  *brokerOptions
	client *asynq.Client
	server *asynq.Server
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
	if b.bOpts.Type == NodeType {
		redisOpt := &asynq.RedisClientOpt{
			Addr:      b.opts.Addrs[0],
			Password:  b.bOpts.Pass,
			DB:        b.bOpts.DB,
			TLSConfig: tlsConfig,
		}
		// redis node mode
		b.client = asynq.NewClient(redisOpt)
		b.server = asynq.NewServer(redisOpt, asynq.Config{})
	} else {
		redisOpt := &asynq.RedisClusterClientOpt{
			Addrs:     b.opts.Addrs,
			Password:  b.bOpts.Pass,
			TLSConfig: tlsConfig,
		}
		// redis cluster node
		b.client = asynq.NewClient(redisOpt)
		b.server = asynq.NewServer(redisOpt, asynq.Config{})
	}
	return nil
}

func (b *asynqBroker) Disconnect() error {
	err := b.client.Close()
	b.client = nil
	return err
}

func (b *asynqBroker) Publish(topic string, m *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(m)
	if err != nil {
		return err
	}

	pOpts := &publishOptions{}
	options := broker.PublishOptions{
		Context: context.WithValue(context.Background(), optionsKey, pOpts),
	}
	for _, o := range opts {
		o(&options)
	}

	t := fmt.Sprintf("%s:%s", topic, pOpts.Opr)
	task := asynq.NewTask(t, v)
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
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Asynq Broker Publish task result: %+v", info)
	}

	return nil
}

func (b *asynqBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	return nil, nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	bOpts := &brokerOptions{
		Type: NodeType,
		DB:   DefaultDB,
	}

	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bOpts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &asynqBroker{
		opts:  options,
		bOpts: bOpts,
	}
}
