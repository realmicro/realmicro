package realmicro

import (
	"os"
	"os/signal"
	"runtime"
	"sync"

	"github.com/realmicro/realmicro/client"
	msignal "github.com/realmicro/realmicro/common/util/signal"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/server"
)

type service struct {
	opts Options

	once sync.Once
}

func newService(opts ...Option) Service {
	return &service{
		opts: newOptions(opts...),
	}
}

func (s *service) Name() string {
	return s.opts.Server.Options().Name
}

// Init initialises options. Additionally, it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *service) Init(opts ...Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *service) Options() Options {
	return s.opts
}

func (s *service) Client() client.Client {
	return s.opts.Client
}

func (s *service) Server() server.Server {
	return s.opts.Server
}

func (s *service) String() string {
	return "realmicro"
}

func (s *service) Start() error {
	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	if err := s.opts.Server.Start(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) Stop() error {
	var err error

	for _, fn := range s.opts.BeforeStop {
		err = fn()
	}

	logger.Logf(logger.InfoLevel, "Stopping [service] %s", s.Name())

	if err = s.opts.Server.Stop(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStop {
		err = fn()
	}

	return err
}

func (s *service) Run() (err error) {
	// exit when help flag is provided
	for _, v := range os.Args[1:] {
		if v == "-h" || v == "--help" {
			os.Exit(0)
		}
	}

	// start the profiler
	if s.opts.Profile != nil {
		// to view mutex contention
		runtime.SetMutexProfileFraction(5)
		// to view blocking profile
		runtime.SetBlockProfileRate(1)

		if err = s.opts.Profile.Start(); err != nil {
			return err
		}
		defer func() {
			err = s.opts.Profile.Stop()
			if err != nil {
				logger.Error(err)
			}
		}()
	}

	// start the health
	if s.opts.Health != nil {
		if err = s.opts.Health.Start(); err != nil {
			return err
		}
		defer func() {
			err = s.opts.Health.Stop()
			if err != nil {
				logger.Error(err)
			}
		}()
	}

	logger.Logf(logger.InfoLevel, "Starting [service] %s", s.Name())

	if err = s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	if s.opts.Signal {
		signal.Notify(ch, msignal.Shutdown()...)
	}

	select {
	// wait on kill signal
	case <-ch:
	// wait on context cancel
	case <-s.opts.Context.Done():
	}

	return s.Stop()
}
