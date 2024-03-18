package autocert

import (
	"crypto/tls"
	"net"
	"os"

	"github.com/realmicro/realmicro/api/server/acme"
	log "github.com/realmicro/realmicro/logger"
	"golang.org/x/crypto/acme/autocert"
)

// autoCertACME is the ACME provider from golang.org/x/crypto/acme/autocert.
type autocertProvider struct {
	logger log.Logger
}

// Listen implements acme.Provider.
func (a *autocertProvider) Listen(hosts ...string) (net.Listener, error) {
	return autocert.NewListener(hosts...), nil
}

// TLSConfig returns a new tls config.
func (a *autocertProvider) TLSConfig(hosts ...string) (*tls.Config, error) {
	logger := log.LoggerOrDefault(a.logger)
	// create a new manager
	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}
	if len(hosts) > 0 {
		m.HostPolicy = autocert.HostWhitelist(hosts...)
	}
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		logger.Logf(log.InfoLevel, "warning: autocert not using a cache: %v", err)
	} else {
		m.Cache = autocert.DirCache(dir)
	}
	return m.TLSConfig(), nil
}

// NewProvider returns an autocert acme.Provider.
func NewProvider() acme.Provider {
	return &autocertProvider{}
}
