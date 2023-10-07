package cache

import (
	"context"
	"time"
)

// Options represents the options for the cache.
type Options struct {
	// Context should contain all implementation specific options, using context.WithValue.
	Context context.Context
	Items   map[string]Item
	// Address represents the address or other connection information of the cache service.
	Address    string
	Expiration time.Duration
}

// Option manipulates the Options passed.
type Option func(o *Options)

// Expiration sets the duration for items stored in the cache to expire.
func Expiration(d time.Duration) Option {
	return func(o *Options) {
		o.Expiration = d
	}
}

// Items initializes the cache with preconfigured items.
func Items(i map[string]Item) Option {
	return func(o *Options) {
		o.Items = i
	}
}

// WithAddress sets the cache service address or connection information
func WithAddress(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

// WithContext sets the cache context, for any extra configuration
func WithContext(c context.Context) Option {
	return func(o *Options) {
		o.Context = c
	}
}

// NewOptions returns a new Options struct.
func NewOptions(opts ...Option) Options {
	options := Options{
		Expiration: DefaultExpiration,
		Items:      make(map[string]Item),
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
