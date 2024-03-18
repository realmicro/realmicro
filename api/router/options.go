package router

import (
	"github.com/realmicro/realmicro/api/resolver"
	"github.com/realmicro/realmicro/api/resolver/vpath"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/registry"
)

// Options is a struct of options available.
type Options struct {
	Registry registry.Registry
	Resolver resolver.Resolver
	Logger   logger.Logger
	Handler  string
}

// Option is a helper for a single options.
type Option func(o *Options)

// NewOptions wires options together.
func NewOptions(opts ...Option) Options {
	options := Options{
		Handler:  "meta",
		Registry: registry.DefaultRegistry,
		Logger:   logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Resolver == nil {
		options.Resolver = vpath.NewResolver(
			resolver.WithHandler(options.Handler),
		)
	}

	return options
}

func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

func WithRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func WithResolver(r resolver.Resolver) Option {
	return func(o *Options) {
		o.Resolver = r
	}
}

// WithLogger sets the underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
