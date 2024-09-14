package main

import (
	"context"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/config"
	cetcd "github.com/realmicro/realmicro/config/source/etcd"
	"github.com/realmicro/realmicro/debug/health/http"
	"github.com/realmicro/realmicro/errors"
	"github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/logger"
	mlogrus "github.com/realmicro/realmicro/logger/logrus"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/realmicro/realmicro/wrapper/validator"
	"github.com/sirupsen/logrus"
)

var (
	cfg config.Config
)

type TestInfo struct {
	Test string `json:"test"`
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
	logger.Infof("Received: %v", req.Name)
	if req.Name == "BreakerError" {
		return errors.New("", "breaker tripped", 502)
	}
	rsp.Greeting = "Hello " + req.Name
	if cfg != nil {
		logger.Info("config data:", cfg.Map())
		// config in etcd:
		// key: helloworld/test
		// value: {"test": "test"}
		var t1, t2 TestInfo
		cfg.Get("test").Scan(&t1)
		cfg.Get("1", "t").Scan(&t2)
		logger.Info("test : ", t1)
		logger.Info("test : ", t2)
	}
	return nil
}

func main() {
	serviceName := "realmicro.helloworld"

	logger.DefaultLogger = mlogrus.NewLogger(mlogrus.WithJSONFormatter(&logrus.JSONFormatter{}))
	logger.Init(logger.WithLevel(logger.InfoLevel))

	logger.Logf(logger.InfoLevel, "Example Name: %s", serviceName)

	etcdAddress := "127.0.0.1:2379"

	var err error
	cfg, err = config.NewConfig(config.WithSource(
		cetcd.NewSource(
			cetcd.WithAddress(etcdAddress),
			cetcd.WithPrefix(serviceName),
			cetcd.StripPrefix(true),
			cetcd.WithPrefixCreate(true),
		),
	))
	if err != nil {
		logger.Fatal(err)
		return
	}

	service := realmicro.NewService(
		realmicro.Name(serviceName),
		realmicro.Registry(etcd.NewRegistry(registry.Addrs([]string{etcdAddress}...))),
		realmicro.Health(http.NewHealth()),
		realmicro.WrapHandler(validator.NewHandlerWrapper()),
	)

	service.Init()

	greeter.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}
