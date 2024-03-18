package main

import (
	"context"

	"github.com/realmicro/realmicro/api"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := etcd.NewRegistry(
		registry.Addrs([]string{"127.0.0.1:2379"}...),
	)

	srv := api.NewApi(
		api.WithRegistry(reg),
		api.WithAddress(":9090"),
	)

	if err := srv.Run(ctx); err != nil {
		logger.Fatal(err)
	}
}
