package asynq

import (
	"context"
	"strings"

	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/codec/json"
)

type asynqBroker struct {
	opts  broker.Options
	bOpts *brokerOptions
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
	panic("implement me")
}

func (b *asynqBroker) Disconnect() error {
	panic("implement me")
}

func (b *asynqBroker) Publish(topic string, m *broker.Message, opts ...broker.PublishOption) error {
	panic("implement me")
}

func (b *asynqBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	panic("implement me")
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
