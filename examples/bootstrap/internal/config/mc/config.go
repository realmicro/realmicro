package mc

import (
	"github.com/realmicro/realmicro/config"
	"github.com/realmicro/realmicro/config/source/etcd"
)

var (
	DefaultConfig = newConfig()
)

type Config struct {
	cfg  config.Config
	opts Options
}

func newConfig(opts ...Option) *Config {
	var c Config

	for _, o := range opts {
		o(&c.opts)
	}

	return &c
}

func (c *Config) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

	var err error
	c.cfg, err = config.NewConfig(config.WithSource(
		etcd.NewSource(
			etcd.WithAddress(c.opts.Address...),
			etcd.WithPrefix(c.opts.ServiceName),
			etcd.StripPrefix(true),
			etcd.WithPrefixCreate(true),
		),
	))
	if err != nil {
		return err
	}
	return nil
}
