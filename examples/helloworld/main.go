package main

import (
	"context"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/config"
	cetcd "github.com/realmicro/realmicro/config/source/etcd"
	greeter "github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
)

var (
	cfg config.Config
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
	rsp.Greeting = "Hello " + req.Name
	if cfg != nil {
		logger.Info("config data:", cfg.Map())
		// config in etcd:
		// key: helloworld/test
		// value: {"test": "test"}
		logger.Info("test:", cfg.Get("test").String("test"))
	}
	return nil
}

func main() {
	logger.Init(logger.WithLevel(logger.TraceLevel))

	serviceName := "helloworld"
	etcdAddress := "127.0.0.1:2379"

	var err error
	cfg, err = config.NewConfig(config.WithSource(
		cetcd.NewSource(
			cetcd.WithAddress(etcdAddress),
			cetcd.WithPrefix(serviceName+"/"),
			cetcd.StripPrefix(true),
		),
	))
	if err != nil {
		logger.Fatal(err)
		return
	}

	service := realmicro.NewService(
		realmicro.Name(serviceName),
		realmicro.Registry(etcd.NewRegistry(registry.Addrs([]string{etcdAddress}...))),
	)

	service.Init()

	greeter.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}
