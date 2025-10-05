// Package realmicro is a pluggable framework for microservices
package realmicro

import (
	"context"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/server"
)

type serviceKey struct{}

// Service is an interface that wraps the lower level libraries
// within realmicro. It's a convenience method for building
// and initialising services.
type Service interface {
	// Name The service name
	Name() string
	// Init initialises options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// Handle Register the handler
	Handle(v interface{}) error
	// Client is used to call services
	Client() client.Client
	// Server is for handling requests and events
	Server() server.Server
	// Run the service
	Run() error
	// String The service implementation
	String() string
}

// Function is a one time executing Service
type Function interface {
	// Service Inherits Service interface
	Service
	// Done signals to complete execution
	Done() error
	// Handle registers an RPC handler
	Handle(v interface{}) error
	// Subscribe registers a subscriber
	Subscribe(topic string, v interface{}) error
}

// Event is used to publish messages to a topic
type Event interface {
	// Publish publishes a message to the event topic
	Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error
}

// Publisher Type alias to satisfy the deprecation
type Publisher = Event

type Option func(*Options)

// NewService creates and returns a new Service based on the packages within.
func NewService(opts ...Option) Service {
	return newService(opts...)
}

// FromContext retrieves a Service from the Context.
func FromContext(ctx context.Context) (Service, bool) {
	s, ok := ctx.Value(serviceKey{}).(Service)
	return s, ok
}

// NewContext returns a new Context with the Service embedded within it.
func NewContext(ctx context.Context, s Service) context.Context {
	return context.WithValue(ctx, serviceKey{}, s)
}

// NewFunction returns a new Function for a one time executing Service
func NewFunction(opts ...Option) Function {
	return newFunction(opts...)
}

// NewEvent creates a new event publisher
func NewEvent(topic string, c client.Client) Event {
	if c == nil {
		c = client.NewClient()
	}
	return &event{c, topic}
}

// RegisterHandler is syntactic sugar for registering a handler
func RegisterHandler(s server.Server, h interface{}, opts ...server.HandlerOption) error {
	return s.Handle(s.NewHandler(h, opts...))
}

// RegisterSubscriber is syntactic sugar for registering a subscriber
func RegisterSubscriber(topic string, s server.Server, h interface{}, opts ...server.SubscriberOption) error {
	return s.Subscribe(s.NewSubscriber(topic, h, opts...))
}
