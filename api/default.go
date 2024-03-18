package api

import (
	"context"

	"github.com/realmicro/realmicro/api/handler"
	"github.com/realmicro/realmicro/api/handler/rpc"
	"github.com/realmicro/realmicro/api/router/registry"
	"github.com/realmicro/realmicro/api/server"
	"github.com/realmicro/realmicro/api/server/http"
)

type api struct {
	options Options

	server server.Server
}

func newApi(opts ...Option) Api {
	options := NewOptions(opts...)

	rtr := options.Router

	if rtr == nil {
		// TODO: make configurable
		rtr = registry.NewRouter()
	}

	// TODO: make configurable
	hdlr := rpc.NewHandler(
		handler.WithRouter(rtr),
	)

	// TODO: make configurable
	// create a new server
	srv := http.NewServer(options.Address)

	// TODO: allow multiple handlers
	// define the handler
	srv.Handle("/", hdlr)

	return &api{
		options: options,
		server:  srv,
	}
}

func (a *api) Init(opts ...Option) error {
	for _, o := range opts {
		o(&a.options)
	}
	return nil
}

// Options Get the options.
func (a *api) Options() Options {
	return a.options
}

// Register a http handler.
func (a *api) Register(*Endpoint) error {
	return nil
}

// Deregister a route.
func (a *api) Deregister(*Endpoint) error {
	return nil
}

func (a *api) Run(ctx context.Context) error {
	if err := a.server.Start(); err != nil {
		return err
	}

	// wait to finish
	<-ctx.Done()

	if err := a.server.Stop(); err != nil {
		return err
	}

	return nil
}

func (a *api) String() string {
	return "http"
}
