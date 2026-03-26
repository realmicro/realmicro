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

	bootstrapServerName = "Bootstrap"
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
		"X-User-Id":   "john",
		"X-From-Id":   "script",
		"traceparent": fmt.Sprintf("traceparent%d", time.Now().Unix()),
	})

	rsp := &greeter.Response{}

	// Call service
	if err := c.Call(ctx, req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp)
}

type Status struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Account struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAccountRequest struct {
	Account *Account `json:"account"`
}

type CreateAccountResponse struct {
	Status Status `json:"status"`
	Id     int64  `json:"id"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Status Status `json:"status"`
	Token  string `json:"token"`
}

func callBootstrap(c client.Client) {
	ctx := metadata.NewContext(context.Background(), map[string]string{
		"X-User-Id":   "john",
		"X-From-Id":   "script",
		"traceparent": fmt.Sprintf("traceparent%d", time.Now().Unix()),
	})

	createAccountReq := c.NewRequest(bootstrapServerName, "BootstrapService.CreateAccount", &CreateAccountRequest{
		Account: &Account{
			Username: "john",
			Password: "123456",
		},
	})
	createAccountRsp := &CreateAccountResponse{}
	// Call service
	if err := c.Call(ctx, createAccountReq, createAccountRsp); err != nil {
		fmt.Println("call err: ", err, createAccountRsp)
		return
	}
	fmt.Println("CreateAccountRsp:", createAccountRsp)

	loginReq := c.NewRequest(bootstrapServerName, "BootstrapService.Login", &LoginRequest{
		Username: "john",
		Password: "123456",
	})
	loginRsp := &LoginResponse{}
	// Call service
	if err := c.Call(ctx, loginReq, loginRsp); err != nil {
		fmt.Println("call err: ", err, loginRsp)
		return
	}
	fmt.Println("LoginRsp:", loginRsp)
}

func main() {
	flag.Parse()

	fmt.Println(*endpoint, *token)
	tp, err := common.NewTraceProvider(context.Background(), *endpoint, *token, "realmicro-client")
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

	//go func() {
	//	for i := 0; i < 10; i++ {
	//		call("g1", i, service.Client())
	//	}
	//}()
	//
	//for i := 0; i < 10; i++ {
	//	call("main", i, service.Client())
	//}

	callBootstrap(service.Client())

	time.Sleep(time.Second * 10)
}
