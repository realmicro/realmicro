package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/config"
	cetcd "github.com/realmicro/realmicro/config/source/etcd"
	"github.com/realmicro/realmicro/debug/health/http"
	"github.com/realmicro/realmicro/errors"
	"github.com/realmicro/realmicro/examples/common"
	"github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/logger"
	mlogrus "github.com/realmicro/realmicro/logger/logrus"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/realmicro/realmicro/wrapper/trace/opentelemetry"
	"github.com/realmicro/realmicro/wrapper/validator"
	"github.com/sirupsen/logrus"
)

const (
	exampleName  = "realmicro.helloworld"
	exampleName1 = "realmicro.helloworld1"
	exampleName2 = "realmicro.helloworld2"
	exampleName3 = "realmicro.helloworld3"
)

var serviceName = flag.String("s", "", "service name")
var endpoint = flag.String("e", "", "trace endpoint")
var token = flag.String("t", "", "trace token")

var (
	cfg config.Config
)

type TestInfo struct {
	Test string `json:"test"`
}

type Greeter struct {
	ServiceName string
	Client      client.Client
}

func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
	logger.Infof("Received: %d %v", req.Id, req.Name)

	if req.Name == "BreakerError" {
		return errors.New("", "breaker tripped", 502)
	}
	rsp.Greeting = "Hello " + req.Name

	if g.ServiceName == exampleName {
		g.call(ctx, req, exampleName1)
		g.call(ctx, req, exampleName2)
	} else if g.ServiceName == exampleName1 {
		g.call(ctx, req, exampleName3)
	}

	//if cfg != nil {
	//	logger.Info("config data:", cfg.Map())
	//	// config in etcd:
	//	// key: helloworld/test
	//	// value: {"test": "test"}
	//	var t1, t2 TestInfo
	//	cfg.Get("test").Scan(&t1)
	//	cfg.Get("1", "t").Scan(&t2)
	//	logger.Info("test : ", t1)
	//	logger.Info("test : ", t2)
	//}
	return nil
}

func (g *Greeter) call(ctx context.Context, req *greeter.Request, sn string) {
	req.Name += ""
	r := g.Client.NewRequest(sn, "Greeter.Hello", req)

	rsp := &greeter.Response{}
	// Call service
	if err := g.Client.Call(ctx, r, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("recall rsp:", rsp)
}

func main() {
	flag.Parse()

	logger.DefaultLogger = mlogrus.NewLogger(mlogrus.WithJSONFormatter(&logrus.JSONFormatter{}))
	logger.Init(logger.WithLevel(logger.DebugLevel))

	sn := *serviceName
	if len(sn) == 0 {
		sn = exampleName
	}
	logger.Logf(logger.InfoLevel, "trace:[%s][%s] Service Name: %s", *endpoint, *token, sn)

	etcdAddress := "127.0.0.1:2379"

	var err error
	cfg, err = config.NewConfig(config.WithSource(
		cetcd.NewSource(
			cetcd.WithAddress(etcdAddress),
			cetcd.WithPrefix(sn),
			cetcd.StripPrefix(true),
			cetcd.WithPrefixCreate(true),
		),
	))
	if err != nil {
		logger.Fatal(err)
		return
	}

	tp, err := common.NewTraceProvider(context.Background(), *endpoint, *token, sn)
	if err != nil {
		logger.Fatal(err)
	}
	service := realmicro.NewService(
		realmicro.Name(sn),
		realmicro.Registry(etcd.NewRegistry(registry.Addrs([]string{etcdAddress}...))),
		realmicro.Health(http.NewHealth()),
		realmicro.WrapHandler(
			validator.NewHandlerWrapper(),
			opentelemetry.NewHandlerWrapper(opentelemetry.WithTraceProvider(tp)),
		),
	)
	service.Init()

	greeter.RegisterGreeterHandler(service.Server(), &Greeter{
		ServiceName: sn,
		Client:      service.Client(),
	})

	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}
