package redis

import (
	"crypto/tls"
	"io"

	"github.com/realmicro/realmicro/common/util/sync"
	red "github.com/redis/go-redis/v9"
)

const (
	defaultDatabase = 0
	maxRetries      = 3
	idleConns       = 8
)

var clientManager = sync.NewResourceManager()

func getClient(r *Redis) (*red.Client, error) {
	if len(r.options.Addrs) == 0 {
		r.options.Addrs = []string{"127.0.0.1:6379"}
	}
	val, err := clientManager.GetResource(r.options.Addrs[0], func() (io.Closer, error) {
		var tlsConfig *tls.Config
		if r.options.tls {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		db := defaultDatabase
		if r.options.DB > 0 {
			db = r.options.DB
		}
		store := red.NewClient(&red.Options{
			Addr:         r.options.Addrs[0],
			Password:     r.options.Pass,
			DB:           db,
			MaxRetries:   maxRetries,
			MinIdleConns: idleConns,
			TLSConfig:    tlsConfig,
		})
		return store, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(*red.Client), nil
}
