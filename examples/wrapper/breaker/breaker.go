package main

import (
	"context"
	"fmt"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/errors"
	"github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/realmicro/realmicro/selector"
	"github.com/realmicro/realmicro/wrapper/breaker/realbreaker"
)

func main() {
	reg := etcd.NewRegistry(registry.Addrs([]string{"127.0.0.1:2379"}...))
	s := selector.NewSelector(selector.Registry(reg))
	c := client.NewClient(
		client.Selector(s),
		// add the breaker wrapper
		client.Wrap(realbreaker.NewClientWrapper()),
	)

	// Create new request to service go.micro.srv.example, method Example.Call
	req := c.NewRequest("real.micro.helloworld", "Greeter.Hello", &greeter.Request{
		Name: "BreakerError",
	})

	// create context with metadata
	ctx := metadata.NewContext(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &greeter.Response{}

	// Force to point of trip
	for i := 0; i < 8; i++ {
		c.Call(ctx, req, rsp)
	}

	req = c.NewRequest("real.micro.helloworld", "Greeter.Hello", &greeter.Request{
		Name: "World",
	})
	err := c.Call(ctx, req, rsp)
	if err == nil {
		fmt.Println("Expecting tripped breaker, got nil error")
		return
	}

	merr := err.(*errors.Error)
	if merr.Code != 502 {
		fmt.Println("Expecting tripped breaker, got ", err)
	}
}
