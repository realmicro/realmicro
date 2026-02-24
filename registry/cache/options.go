package cache

import "time"

type Options struct {
	// TTL is the cache TTL
	TTL time.Duration
	// MinimumRetryInterval is the minimum time to wait before retrying a failed service lookup
	// This prevents cache penetration when registry is failing and there's no stale cache
	MinimumRetryInterval time.Duration
}

type Option func(o *Options)

// WithTTL sets the cache TTL
func WithTTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}

// WithMinimumRetryInterval sets the minimum retry interval for failed lookups.
// This prevents cache penetration when registry is failing and there's no stale cache.
func WithMinimumRetryInterval(d time.Duration) Option {
	return func(o *Options) {
		o.MinimumRetryInterval = d
	}
}
