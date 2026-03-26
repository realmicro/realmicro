package svc

import (
	"context"
	"fmt"
	"os"

	"bootstrap/internal/config"
	"bootstrap/models"

	"github.com/realmicro/realmicro"
	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/server"
	"xorm.io/xorm"
)

type ServiceContext struct {
	Config *config.Config
	Server server.Server
	Client client.Client
}

func NewServiceContext(config *config.Config, service realmicro.Service) *ServiceContext {
	models.Init(
		models.WithBdPath(config.Hosts.Sqlite.DBPath),
		models.WithPassword(config.Hosts.Sqlite.Password),
		models.IfShowSql(config.Hosts.Sqlite.IfShowSql),
		models.IfSyncDB(config.Hosts.Sqlite.IfSyncDB),
		models.AfterInit(func(x *xorm.Engine) {
			logger.Info("Sqlite Init Success")
			if config.Hosts.Sqlite.IfSyncDB {
				// sync tables
				if err := x.Sync2(
					new(models.Account),
				); err != nil {
					fmt.Printf("db sync error: %v\n", err)
					os.Exit(1)
				}
			}
		}),
	)

	sc := &ServiceContext{
		Config: config,
		Server: service.Server(),
		Client: service.Client(),
	}

	return sc
}

func (sc *ServiceContext) Context() context.Context {
	return context.Background()
}
