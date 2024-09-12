package http

import (
	"context"
	"net/http"
	"sync"

	"github.com/realmicro/realmicro/debug/health"
	"github.com/realmicro/realmicro/logger"
)

type httpHealth struct {
	opts health.Options

	sync.Mutex
	running bool
	server  *http.Server
}

// Start the health
func (h *httpHealth) Start() error {
	h.Lock()
	defer h.Unlock()

	if h.running {
		return nil
	}

	go func() {
		if err := h.server.ListenAndServe(); err != nil {
			h.Lock()
			h.running = false
			h.Unlock()
		}
	}()

	h.running = true

	logger.Logf(logger.InfoLevel, "Starting health[%s] at: %s", h.String(), h.opts.Address)

	return nil
}

// Stop the health
func (h *httpHealth) Stop() error {
	h.Lock()
	defer h.Unlock()

	if !h.running {
		return nil
	}

	h.running = false

	return h.server.Shutdown(context.TODO())
}

func (h *httpHealth) String() string {
	return "http"
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func NewHealth(opts ...health.Option) health.Health {
	options := health.Options{
		Address: health.DefaultAddress,
	}
	for _, o := range opts {
		o(&options)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthCheck)

	h := &httpHealth{
		opts: options,
		server: &http.Server{
			Addr:    options.Address,
			Handler: mux,
		},
	}

	return h
}
