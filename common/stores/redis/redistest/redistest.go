package redistest

import (
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/realmicro/realmicro/common/stores/redis"
)

// CreateRedis returns an in process redis.Redis.
func CreateRedis(t *testing.T) *redis.Redis {
	r, _ := CreateRedisWithClean(t)
	return r
}

// CreateRedisWithClean returns an in process redis.Redis and a clean function.
func CreateRedisWithClean(t *testing.T) (r *redis.Redis, clean func()) {
	mr := miniredis.RunT(t)
	return redis.New(
		redis.Addrs(strings.Split(mr.Addr(), ",")),
	), mr.Close
}
