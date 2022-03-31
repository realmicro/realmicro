package etcd

import (
	"context"
	"time"

	"github.com/realmicro/realmicro/config/source"
)

type addressKey struct{}
type prefixKey struct{}
type stripPrefixKey struct{}
type authKey struct{}
type dialTimeoutKey struct{}
type prefixCreateKey struct{}

type authCredit struct {
	Username string
	Password string
}

// WithAddress sets the etcd address
func WithAddress(a ...string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, a)
	}
}

// WithPrefix sets the key prefix to use
func WithPrefix(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, prefixKey{}, p)
	}
}

// StripPrefix indicates whether to remove the prefix from config entries, or leave it in place.
func StripPrefix(strip bool) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, stripPrefixKey{}, strip)
	}
}

// WithPrefixCreate create prefix if source not found
func WithPrefixCreate(ifCreated bool) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, prefixCreateKey{}, ifCreated)
	}
}

// Auth allows you to specify username/password
func Auth(username, password string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, authKey{}, &authCredit{Username: username, Password: password})
	}
}

// WithDialTimeout set the time out for dialing to etcd
func WithDialTimeout(timeout time.Duration) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dialTimeoutKey{}, timeout)
	}
}
