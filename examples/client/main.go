package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/examples/common"
	greeter "github.com/realmicro/realmicro/examples/helloworld/proto"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/realmicro/realmicro/wrapper/trace/opentelemetry"
)

var (
	serverName = "realmicro.helloworld"
)

var endpoint = flag.String("e", "", "trace endpoint")
var token = flag.String("t", "", "trace token")

func call(n string, i int, c client.Client) {
	// Create new request to service real.micro.example, method Greeter.Hello
	req := c.NewRequest(serverName, "Greeter.Hello", &greeter.Request{
		Id:   uint64(i),
		Name: fmt.Sprintf("%s%d", n, i),
	})

	// create context with metadata
	ctx := metadata.NewContext(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &greeter.Response{}

	// Call service
	if err := c.Call(ctx, req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp)
}

func main() {
	flag.Parse()

	tp, err := common.NewTraceProvider(context.Background(), *endpoint, *token, serverName)
	if err != nil {
		logger.Fatal(err)
	}
	service := realmicro.NewService(
		realmicro.Registry(etcd.NewRegistry(
			registry.Addrs([]string{"127.0.0.1:2379"}...),
		)),
		realmicro.WrapClient(opentelemetry.NewClientWrapper(opentelemetry.WithTraceProvider(tp))),
	)
	service.Init()

	fmt.Println("\n--- Call example ---")

	go func() {
		for i := 0; i < 10; i++ {
			call("g1", i, service.Client())
		}
	}()

	for i := 0; i < 10; i++ {
		call("main", i, service.Client())
	}

	time.Sleep(time.Second * 10)
}
