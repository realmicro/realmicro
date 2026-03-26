package main

import (
	"flag"
	"fmt"
	"os"

	"bootstrap/internal/config"
	"bootstrap/internal/config/mc"
	"bootstrap/internal/logic"
	wrapper "bootstrap/internal/middleware"
	"bootstrap/internal/svc"
	"bootstrap/pkg/utils"
	"bootstrap/proto"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/broker"
	mconfig "github.com/realmicro/realmicro/config"
	"github.com/realmicro/realmicro/config/encoder/yaml"
	"github.com/realmicro/realmicro/config/reader"
	"github.com/realmicro/realmicro/config/reader/json"
	"github.com/realmicro/realmicro/config/source/env"
	"github.com/realmicro/realmicro/config/source/file"
	"github.com/realmicro/realmicro/debug/health/http"
	"github.com/realmicro/realmicro/logger"
	mlogrus "github.com/realmicro/realmicro/logger/logrus"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

var configFile = flag.String("f", "etc/config.yaml", "the config file")

func main() {
	flag.Parse()

	c, _ := mconfig.NewConfig(
		mconfig.WithReader(
			json.NewReader(
				reader.WithEncoder(yaml.NewEncoder()),
			),
		),
	)
	var err error

	if utils.EnvLookUpPrefixKey("REALMICRO_CFG") {
		// load config from env
		if err = c.Load(env.NewSource(
			env.WithStrippedPrefix("REALMICRO_CFG"),
		)); err != nil {
			fmt.Println(err)
			return
		}
	} else {
		// load the config from a file source
		if err = c.Load(file.NewSource(
			file.WithPath(*configFile),
		)); err != nil {
			fmt.Println(err)
			return
		}
	}

	var cfg config.Config
	if err = c.Scan(&cfg); err != nil {
		fmt.Println(err)
		return
	}

	logger.DefaultLogger = mlogrus.NewLogger(mlogrus.WithJSONFormatter(&logrus.JSONFormatter{}))

	service := realmicro.NewService(
		realmicro.Name(cfg.ServiceName),
		realmicro.Version(cfg.Version),
		realmicro.Metadata(map[string]string{
			"env":     cfg.Env,
			"project": cfg.Project,
		}),
		realmicro.Registry(etcd.NewRegistry(registry.Addrs(cfg.Hosts.Etcd.Address...))),
		realmicro.Broker(broker.DefaultBroker),
		realmicro.WrapHandler(wrapper.LogHandler()),
		realmicro.WrapClient(wrapper.LogCall),
	)
	if cfg.HealthCheck {
		service.Init(realmicro.Health(http.NewHealth()))
	} else {
		service.Init()
	}

	if err = mc.DefaultConfig.Init(
		mc.WithAddress(cfg.Hosts.Etcd.Address),
		mc.WithServiceName(cfg.ServiceName),
	); err != nil {
		logger.Fatal(err)
	}

	ctx := svc.NewServiceContext(&cfg, service)
	if err = bootstrap.RegisterBootstrapServiceHandler(ctx.Server, &logic.BootstrapService{
		SvcContext: ctx,
	}); err != nil {
		logger.Fatal(err)
	}

	if err = service.Run(); err != nil {
		logger.Fatal(err)
	}
}
