package main

import (
	"context"

	"github.com/realmicro/realmicro"
	greeter "github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	service := realmicro.NewService(
		realmicro.Name("helloworld"),
		realmicro.Registry(etcd.NewRegistry(registry.Addrs([]string{"127.0.0.1:2379"}...))),
	)

	service.Init()

	greeter.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}
