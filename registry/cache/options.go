package cache

import "time"

type Options struct {
	// TTL is the cache TTL
	TTL time.Duration
}

type Option func(o *Options)

// WithTTL sets the cache TTL
func WithTTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}
