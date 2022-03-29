package server

import (
	"crypto/tls"
	"net/http"

	"github.com/realmicro/realmicro/api/server/cors"
)

type Option func(o *Options)

type Options struct {
	EnableACME bool
	EnableCORS bool
	CORSConfig *cors.Config
	EnableTLS  bool
	ACMEHosts  []string
	TLSConfig  *tls.Config
	Wrappers   []Wrapper
}

type Wrapper func(h http.Handler) http.Handler

func WrapHandler(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

func EnableCORS(b bool) Option {
	return func(o *Options) {
		o.EnableCORS = b
	}
}

func CORSConfig(c *cors.Config) Option {
	return func(o *Options) {
		o.CORSConfig = c
	}
}

func EnableACME(b bool) Option {
	return func(o *Options) {
		o.EnableACME = b
	}
}

func ACMEHosts(hosts ...string) Option {
	return func(o *Options) {
		o.ACMEHosts = hosts
	}
}

func EnableTLS(b bool) Option {
	return func(o *Options) {
		o.EnableTLS = b
	}
}

func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}
