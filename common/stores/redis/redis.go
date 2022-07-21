package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	red "github.com/go-redis/redis/v8"
	"github.com/realmicro/realmicro/common/util/breaker"
	"github.com/realmicro/realmicro/common/util/mapping"
)

const (
	// ClusterType means redis cluster.
	ClusterType = "cluster"
	// NodeType means redis node.
	NodeType = "node"
	// Nil is an alias of redis.Nil.
	Nil = red.Nil

	blockingQueryTimeout = 5 * time.Second
	readWriteTimeout     = 2 * time.Second
	defaultSlowThreshold = time.Millisecond * 100
)

var (
	ErrNilNode = errors.New("nil redis node")
)

type (
	Redis struct {
		options Options
		brk     breaker.Breaker
	}

	// A Pair is a key/pair set used in redis zset.
	Pair struct {
		Key   string
		Score int64
	}

	// GeoLocation is used with GeoAdd to add geospatial location.
	GeoLocation = red.GeoLocation
	// GeoRadiusQuery is used with GeoRadius to query geospatial index.
	GeoRadiusQuery = red.GeoRadiusQuery
	// GeoPos is used to represent a geo position.
	GeoPos = red.GeoPos

	// Pipeliner is an alias of redis.Pipeliner.
	Pipeliner = red.Pipeliner

	// Z represents sorted set member.
	Z = red.Z
	// ZStore is an alias of redis.ZStore.
	ZStore = red.ZStore

	// IntCmd is an alias of redis.IntCmd.
	IntCmd = red.IntCmd
	// FloatCmd is an alias of redis.FloatCmd.
	FloatCmd = red.FloatCmd
)

func acceptable(err error) bool {
	return err == nil || err == red.Nil || err == context.Canceled
}

func New(opts ...Option) *Redis {
	r := &Redis{
		options: Options{Type: NodeType},
	}
	for _, opt := range opts {
		opt(&r.options)
	}
	r.brk = breaker.New(breaker.WithName(r.String()))
	return r
}

// String returns the string representation of s.
func (s *Redis) String() string {
	return "redis://" + strings.Join(s.options.Addrs, ",")
}

// Ping is the implementation of redis ping command.
func (s *Redis) Ping(ctx context.Context) (val bool) {
	_ = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return nil
		}
		v, err := conn.Ping(ctx).Result()
		if err != nil {
			val = false
			return nil
		}
		val = v == "PONG"
		return nil
	}, acceptable)
	return
}

// Publish is the implementation of redis publish command.
func (s *Redis) Publish(ctx context.Context, channel string, payload interface{}) (err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		if _, err = conn.Publish(ctx, channel, payload).Result(); err != nil {
			return err
		}
		return nil
	}, acceptable)
	return
}

// Subscribe is the implementation of redis subscribe command.
func (s *Redis) Subscribe(ctx context.Context, channel string) (subscriber *red.PubSub, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		if s.options.Type == NodeType {
			subscriber = conn.(*red.Client).Subscribe(ctx, channel)
		} else {
			subscriber = conn.(*red.ClusterClient).Subscribe(ctx, channel)
		}
		return nil
	}, acceptable)
	return
}

// BitCount is redis bitcount command implementation.
func (s *Redis) BitCount(ctx context.Context, key string, start, end int64) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitCount(ctx, key, &red.BitCount{
			Start: start,
			End:   end,
		}).Result()
		return err
	}, acceptable)

	return
}

// BitOpAnd is redis bit operation (and) command implementation.
func (s *Redis) BitOpAnd(ctx context.Context, destKey string, keys ...string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitOpAnd(ctx, destKey, keys...).Result()
		return err
	}, acceptable)

	return
}

// BitOpNot is redis bit operation (not) command implementation.
func (s *Redis) BitOpNot(ctx context.Context, destKey, key string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitOpNot(ctx, destKey, key).Result()
		return err
	}, acceptable)

	return
}

// BitOpOr is redis bit operation (or) command implementation.
func (s *Redis) BitOpOr(ctx context.Context, destKey string, keys ...string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitOpOr(ctx, destKey, keys...).Result()
		return err
	}, acceptable)

	return
}

// BitOpXor is redis bit operation (xor) command implementation.
func (s *Redis) BitOpXor(ctx context.Context, destKey string, keys ...string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitOpXor(ctx, destKey, keys...).Result()
		return err
	}, acceptable)

	return
}

// BitPos is redis bitpos command implementation.
func (s *Redis) BitPos(ctx context.Context, key string, bit, start, end int64) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.BitPos(ctx, key, bit, start, end).Result()
		return err
	}, acceptable)

	return
}

// Blpop uses passed in redis connection to execute blocking queries.
// Doesn't benefit from pooling redis connections of blocking queries
func (s *Redis) Blpop(ctx context.Context, node red.Cmdable, key string) (string, error) {
	if node == nil {
		return "", ErrNilNode
	}

	vals, err := node.BLPop(ctx, blockingQueryTimeout, key).Result()
	if err != nil {
		return "", err
	}

	if len(vals) < 2 {
		return "", fmt.Errorf("no value on key: %s", key)
	}

	return vals[1], nil
}

// BlpopEx uses passed in redis connection to execute blpop command.
// The difference against Blpop is that this method returns a bool to indicate success.
func (s *Redis) BlpopEx(ctx context.Context, node red.Cmdable, key string) (string, bool, error) {
	if node == nil {
		return "", false, ErrNilNode
	}

	vals, err := node.BLPop(ctx, blockingQueryTimeout, key).Result()
	if err != nil {
		return "", false, err
	}

	if len(vals) < 2 {
		return "", false, fmt.Errorf("no value on key: %s", key)
	}

	return vals[1], true, nil
}

// Decr is the implementation of redis decr command.
func (s *Redis) Decr(ctx context.Context, key string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.Decr(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Decrby is the implementation of redis decrby command.
func (s *Redis) Decrby(ctx context.Context, key string, decrement int64) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.DecrBy(ctx, key, decrement).Result()
		return err
	}, acceptable)

	return
}

// Del deletes keys.
func (s *Redis) Del(ctx context.Context, keys ...string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		v, err := conn.Del(ctx, keys...).Result()
		if err != nil {
			return err
		}
		val = int(v)
		return nil
	}, acceptable)

	return
}

// Eval is the implementation of redis eval command.
func (s *Redis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (val interface{}, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		val, err = conn.Eval(ctx, script, keys, args...).Result()
		return err
	}, acceptable)

	return
}

// EvalSha is the implementation of redis evalsha command.
func (s *Redis) EvalSha(ctx context.Context, sha string, keys []string,
	args ...interface{}) (val interface{}, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.EvalSha(ctx, sha, keys, args...).Result()
		return err
	}, acceptable)

	return
}

// Exists is the implementation of redis exists command.
func (s *Redis) Exists(ctx context.Context, key string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.Exists(ctx, key).Result()
		if err != nil {
			return err
		}

		val = v == 1
		return nil
	}, acceptable)

	return
}

// Expire is the implementation of redis expire command.
func (s *Redis) Expire(ctx context.Context, key string, seconds int) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.Expire(ctx, key, time.Duration(seconds)*time.Second).Err()
	}, acceptable)
}

// Expireat is the implementation of redis expireat command.
func (s *Redis) Expireat(ctx context.Context, key string, expireTime int64) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.ExpireAt(ctx, key, time.Unix(expireTime, 0)).Err()
	}, acceptable)
}

// GeoAdd is the implementation of redis geoadd command.
func (s *Redis) GeoAdd(ctx context.Context, key string, geoLocation ...*GeoLocation) (
	val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoAdd(ctx, key, geoLocation...).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// GeoDist is the implementation of redis geodist command.
func (s *Redis) GeoDist(ctx context.Context, key, member1, member2, unit string) (
	val float64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoDist(ctx, key, member1, member2, unit).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// GeoHash is the implementation of redis geohash command.
func (s *Redis) GeoHash(ctx context.Context, key string, members ...string) (
	val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoHash(ctx, key, members...).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// GeoRadius is the implementation of redis georadius command.
func (s *Redis) GeoRadius(ctx context.Context, key string, longitude, latitude float64,
	query *GeoRadiusQuery) (val []GeoLocation, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoRadius(ctx, key, longitude, latitude, query).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// GeoRadiusByMember is the implementation of redis georadiusbymember command.
func (s *Redis) GeoRadiusByMember(ctx context.Context, key, member string,
	query *GeoRadiusQuery) (val []GeoLocation, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoRadiusByMember(ctx, key, member, query).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// GeoPos is the implementation of redis geopos command.
func (s *Redis) GeoPos(ctx context.Context, key string, members ...string) (
	val []*GeoPos, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GeoPos(ctx, key, members...).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// Get is the implementation of redis get command.
func (s *Redis) Get(ctx context.Context, key string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		if val, err = conn.Get(ctx, key).Result(); err == red.Nil {
			return nil
		} else if err != nil {
			return err
		} else {
			return nil
		}
	}, acceptable)

	return
}

// GetBit is the implementation of redis getbit command.
func (s *Redis) GetBit(ctx context.Context, key string, offset int64) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.GetBit(ctx, key, offset).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// GetSet is the implementation of redis getset command.
func (s *Redis) GetSet(ctx context.Context, key, value string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		if val, err = conn.GetSet(ctx, key, value).Result(); err == red.Nil {
			return nil
		}

		return err
	}, acceptable)

	return
}

// Hdel is the implementation of redis hdel command.
func (s *Redis) Hdel(ctx context.Context, key string, fields ...string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.HDel(ctx, key, fields...).Result()
		if err != nil {
			return err
		}

		val = v >= 1
		return nil
	}, acceptable)

	return
}

// Hexists is the implementation of redis hexists command.
func (s *Redis) Hexists(ctx context.Context, key, field string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HExists(ctx, key, field).Result()
		return err
	}, acceptable)

	return
}

// Hget is the implementation of redis hget command.
func (s *Redis) Hget(ctx context.Context, key, field string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HGet(ctx, key, field).Result()
		return err
	}, acceptable)

	return
}

// Hgetall is the implementation of redis hgetall command.
func (s *Redis) Hgetall(ctx context.Context, key string) (val map[string]string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HGetAll(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Hincrby is the implementation of redis hincrby command.
func (s *Redis) Hincrby(ctx context.Context, key, field string, increment int) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.HIncrBy(ctx, key, field, int64(increment)).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Hkeys is the implementation of redis hkeys command.
func (s *Redis) Hkeys(ctx context.Context, key string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HKeys(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Hlen is the implementation of redis hlen command.
func (s *Redis) Hlen(ctx context.Context, key string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.HLen(ctx, key).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Hmget is the implementation of redis hmget command.
func (s *Redis) Hmget(ctx context.Context, key string, fields ...string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.HMGet(ctx, key, fields...).Result()
		if err != nil {
			return err
		}

		val = toStrings(v)
		return nil
	}, acceptable)

	return
}

// Hset is the implementation of redis hset command.
func (s *Redis) Hset(ctx context.Context, key, field, value string) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.HSet(ctx, key, field, value).Err()
	}, acceptable)
}

// Hsetnx is the implementation of redis hsetnx command.
func (s *Redis) Hsetnx(ctx context.Context, key, field, value string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HSetNX(ctx, key, field, value).Result()
		return err
	}, acceptable)

	return
}

// Hmset is the implementation of redis hmset command.
func (s *Redis) Hmset(ctx context.Context, key string, fieldsAndValues map[string]string) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		vals := make(map[string]interface{}, len(fieldsAndValues))
		for k, v := range fieldsAndValues {
			vals[k] = v
		}

		return conn.HMSet(ctx, key, vals).Err()
	}, acceptable)
}

// Hscan is the implementation of redis hscan command.
func (s *Redis) Hscan(ctx context.Context, key string, cursor uint64, match string, count int64) (
	keys []string, cur uint64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		keys, cur, err = conn.HScan(ctx, key, cursor, match, count).Result()
		return err
	}, acceptable)

	return
}

// Hvals is the implementation of redis hvals command.
func (s *Redis) Hvals(ctx context.Context, key string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.HVals(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Incr is the implementation of redis incr command.
func (s *Redis) Incr(ctx context.Context, key string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.Incr(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Incrby is the implementation of redis incrby command.
func (s *Redis) Incrby(ctx context.Context, key string, increment int64) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.IncrBy(ctx, key, increment).Result()
		return err
	}, acceptable)

	return
}

// Keys is the implementation of redis keys command.
func (s *Redis) Keys(ctx context.Context, pattern string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.Keys(ctx, pattern).Result()
		return err
	}, acceptable)

	return
}

// Llen is the implementation of redis llen command.
func (s *Redis) Llen(ctx context.Context, key string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.LLen(ctx, key).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Lindex is the implementation of redis lindex command.
func (s *Redis) Lindex(ctx context.Context, key string, index int64) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.LIndex(ctx, key, index).Result()
		return err
	}, acceptable)

	return
}

// Lpop is the implementation of redis lpop command.
func (s *Redis) Lpop(ctx context.Context, key string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.LPop(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Lpush is the implementation of redis lpush command.
func (s *Redis) Lpush(ctx context.Context, key string, values ...interface{}) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.LPush(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Lrange is the implementation of redis lrange command.
func (s *Redis) Lrange(ctx context.Context, key string, start, stop int) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.LRange(ctx, key, int64(start), int64(stop)).Result()
		return err
	}, acceptable)

	return
}

// Lrem is the implementation of redis lrem command.
func (s *Redis) Lrem(ctx context.Context, key string, count int, value string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.LRem(ctx, key, int64(count), value).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Ltrim is the implementation of redis ltrim command.
func (s *Redis) Ltrim(ctx context.Context, key string, start, stop int64) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.LTrim(ctx, key, start, stop).Err()
	}, acceptable)
}

// Mget is the implementation of redis mget command.
func (s *Redis) Mget(ctx context.Context, keys ...string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.MGet(ctx, keys...).Result()
		if err != nil {
			return err
		}

		val = toStrings(v)
		return nil
	}, acceptable)

	return
}

// Persist is the implementation of redis persist command.
func (s *Redis) Persist(ctx context.Context, key string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.Persist(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Pfadd is the implementation of redis pfadd command.
func (s *Redis) Pfadd(ctx context.Context, key string, values ...interface{}) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.PFAdd(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = v >= 1
		return nil
	}, acceptable)

	return
}

// Pfcount is the implementation of redis pfcount command.
func (s *Redis) Pfcount(ctx context.Context, key string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.PFCount(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Pfmerge is the implementation of redis pfmerge command.
func (s *Redis) Pfmerge(ctx context.Context, dest string, keys ...string) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		_, err = conn.PFMerge(ctx, dest, keys...).Result()
		return err
	}, acceptable)
}

// Pipelined lets fn execute pipelined commands.
// Results need to be retrieved by calling Pipeline.Exec()
func (s *Redis) Pipelined(ctx context.Context, fn func(Pipeliner) error) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		_, err = conn.Pipelined(ctx, fn)
		return err
	}, acceptable)
}

// Rpop is the implementation of redis rpop command.
func (s *Redis) Rpop(ctx context.Context, key string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.RPop(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Rpush is the implementation of redis rpush command.
func (s *Redis) Rpush(ctx context.Context, key string, values ...interface{}) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.RPush(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Sadd is the implementation of redis sadd command.
func (s *Redis) Sadd(ctx context.Context, key string, values ...interface{}) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SAdd(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Scan is the implementation of redis scan command.
func (s *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) (
	keys []string, cur uint64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		keys, cur, err = conn.Scan(ctx, cursor, match, count).Result()
		return err
	}, acceptable)

	return
}

// SetBit is the implementation of redis setbit command.
func (s *Redis) SetBit(ctx context.Context, key string, offset int64, value int) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SetBit(ctx, key, offset, value).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Sscan is the implementation of redis sscan command.
func (s *Redis) Sscan(ctx context.Context, key string, cursor uint64, match string, count int64) (
	keys []string, cur uint64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		keys, cur, err = conn.SScan(ctx, key, cursor, match, count).Result()
		return err
	}, acceptable)

	return
}

// Scard is the implementation of redis scard command.
func (s *Redis) Scard(ctx context.Context, key string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SCard(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// ScriptLoad is the implementation of redis script load command.
func (s *Redis) ScriptLoad(ctx context.Context, script string) (string, error) {
	conn, err := getRedis(s)
	if err != nil {
		return "", err
	}

	return conn.ScriptLoad(ctx, script).Result()
}

// Set is the implementation of redis set command.
func (s *Redis) Set(ctx context.Context, key, value string) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.Set(ctx, key, value, 0).Err()
	}, acceptable)
}

// Setex is the implementation of redis setex command.
func (s *Redis) Setex(ctx context.Context, key, value string, seconds int) error {
	return s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		return conn.Set(ctx, key, value, time.Duration(seconds)*time.Second).Err()
	}, acceptable)
}

// Setnx is the implementation of redis setnx command.
func (s *Redis) Setnx(ctx context.Context, key, value string) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SetNX(ctx, key, value, 0).Result()
		return err
	}, acceptable)

	return
}

// SetnxEx is the implementation of redis setnx command with expire.
func (s *Redis) SetnxEx(ctx context.Context, key, value string, seconds int) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SetNX(ctx, key, value, time.Duration(seconds)*time.Second).Result()
		return err
	}, acceptable)

	return
}

// Sismember is the implementation of redis sismember command.
func (s *Redis) Sismember(ctx context.Context, key string, value interface{}) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SIsMember(ctx, key, value).Result()
		return err
	}, acceptable)

	return
}

// Smembers is the implementation of redis smembers command.
func (s *Redis) Smembers(ctx context.Context, key string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SMembers(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Spop is the implementation of redis spop command.
func (s *Redis) Spop(ctx context.Context, key string) (val string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SPop(ctx, key).Result()
		return err
	}, acceptable)

	return
}

// Srandmember is the implementation of redis srandmember command.
func (s *Redis) Srandmember(ctx context.Context, key string, count int) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SRandMemberN(ctx, key, int64(count)).Result()
		return err
	}, acceptable)

	return
}

// Srem is the implementation of redis srem command.
func (s *Redis) Srem(ctx context.Context, key string, values ...interface{}) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SRem(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Sunion is the implementation of redis sunion command.
func (s *Redis) Sunion(ctx context.Context, keys ...string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SUnion(ctx, keys...).Result()
		return err
	}, acceptable)

	return
}

// Sunionstore is the implementation of redis sunionstore command.
func (s *Redis) Sunionstore(ctx context.Context, destination string, keys ...string) (
	val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SUnionStore(ctx, destination, keys...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Sdiff is the implementation of redis sdiff command.
func (s *Redis) Sdiff(ctx context.Context, keys ...string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SDiff(ctx, keys...).Result()
		return err
	}, acceptable)

	return
}

// Sdiffstore is the implementation of redis sdiffstore command.
func (s *Redis) Sdiffstore(ctx context.Context, destination string, keys ...string) (
	val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SDiffStore(ctx, destination, keys...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Sinter is the implementation of redis sinter command.
func (s *Redis) Sinter(ctx context.Context, keys ...string) (val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.SInter(ctx, keys...).Result()
		return err
	}, acceptable)

	return
}

// Sinterstore is the implementation of redis sinterstore command.
func (s *Redis) Sinterstore(ctx context.Context, destination string, keys ...string) (
	val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.SInterStore(ctx, destination, keys...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Ttl is the implementation of redis ttl command.
func (s *Redis) Ttl(ctx context.Context, key string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		duration, err := conn.TTL(ctx, key).Result()
		if err != nil {
			return err
		}

		val = int(duration / time.Second)
		return nil
	}, acceptable)

	return
}

// Zadd is the implementation of redis zadd command.
func (s *Redis) Zadd(ctx context.Context, key string, score int64, value string) (
	val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZAdd(ctx, key, &red.Z{
			Score:  float64(score),
			Member: value,
		}).Result()
		if err != nil {
			return err
		}

		val = v == 1
		return nil
	}, acceptable)

	return
}

// Zadds is the implementation of redis zadds command.
func (s *Redis) Zadds(ctx context.Context, key string, ps ...Pair) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		var zs []*red.Z
		for _, p := range ps {
			z := &red.Z{Score: float64(p.Score), Member: p.Key}
			zs = append(zs, z)
		}

		v, err := conn.ZAdd(ctx, key, zs...).Result()
		if err != nil {
			return err
		}

		val = v
		return nil
	}, acceptable)

	return
}

// Zcard is the implementation of redis zcard command.
func (s *Redis) Zcard(ctx context.Context, key string) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZCard(ctx, key).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Zcount is the implementation of redis zcount command.
func (s *Redis) Zcount(ctx context.Context, key string, start, stop int64) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZCount(ctx, key, strconv.FormatInt(start, 10),
			strconv.FormatInt(stop, 10)).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Zincrby is the implementation of redis zincrby command.
func (s *Redis) Zincrby(ctx context.Context, key string, increment int64, field string) (
	val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZIncrBy(ctx, key, float64(increment), field).Result()
		if err != nil {
			return err
		}

		val = int64(v)
		return nil
	}, acceptable)

	return
}

// Zscore is the implementation of redis zscore command.
func (s *Redis) Zscore(ctx context.Context, key, value string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZScore(ctx, key, value).Result()
		if err != nil {
			return err
		}

		val = int64(v)
		return nil
	}, acceptable)

	return
}

// Zrank is the implementation of redis zrank command.
func (s *Redis) Zrank(ctx context.Context, key, field string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.ZRank(ctx, key, field).Result()
		return err
	}, acceptable)

	return
}

// Zrem is the implementation of redis zrem command.
func (s *Redis) Zrem(ctx context.Context, key string, values ...interface{}) (val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRem(ctx, key, values...).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Zremrangebyscore is the implementation of redis zremrangebyscore command.
func (s *Redis) Zremrangebyscore(ctx context.Context, key string, start, stop int64) (
	val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRemRangeByScore(ctx, key, strconv.FormatInt(start, 10),
			strconv.FormatInt(stop, 10)).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Zremrangebyrank is the implementation of redis zremrangebyrank command.
func (s *Redis) Zremrangebyrank(ctx context.Context, key string, start, stop int64) (
	val int, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRemRangeByRank(ctx, key, start, stop).Result()
		if err != nil {
			return err
		}

		val = int(v)
		return nil
	}, acceptable)

	return
}

// Zrange is the implementation of redis zrange command.
func (s *Redis) Zrange(ctx context.Context, key string, start, stop int64) (
	val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.ZRange(ctx, key, start, stop).Result()
		return err
	}, acceptable)

	return
}

// ZrangeWithScores is the implementation of redis zrange command with scores.
func (s *Redis) ZrangeWithScores(ctx context.Context, key string, start, stop int64) (
	val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRangeWithScores(ctx, key, start, stop).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// ZRevRangeWithScores is the implementation of redis zrevrange command with scores.
func (s *Redis) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) (
	val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRevRangeWithScores(ctx, key, start, stop).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// ZrangebyscoreWithScores is the implementation of redis zrangebyscore command with scores.
func (s *Redis) ZrangebyscoreWithScores(ctx context.Context, key string, start, stop int64) (
	val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRangeByScoreWithScores(ctx, key, &red.ZRangeBy{
			Min: strconv.FormatInt(start, 10),
			Max: strconv.FormatInt(stop, 10),
		}).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// ZrangebyscoreWithScoresAndLimit is the implementation of redis zrangebyscore command
// with scores and limit.
func (s *Redis) ZrangebyscoreWithScoresAndLimit(ctx context.Context, key string, start,
	stop int64, page, size int) (val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		if size <= 0 {
			return nil
		}

		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRangeByScoreWithScores(ctx, key, &red.ZRangeBy{
			Min:    strconv.FormatInt(start, 10),
			Max:    strconv.FormatInt(stop, 10),
			Offset: int64(page * size),
			Count:  int64(size),
		}).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// Zrevrange is the implementation of redis zrevrange command.
func (s *Redis) Zrevrange(ctx context.Context, key string, start, stop int64) (
	val []string, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.ZRevRange(ctx, key, start, stop).Result()
		return err
	}, acceptable)

	return
}

// ZrevrangebyscoreWithScores is the implementation of redis zrevrangebyscore command with scores.
func (s *Redis) ZrevrangebyscoreWithScores(ctx context.Context, key string, start, stop int64) (
	val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRevRangeByScoreWithScores(ctx, key, &red.ZRangeBy{
			Min: strconv.FormatInt(start, 10),
			Max: strconv.FormatInt(stop, 10),
		}).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// ZrevrangebyscoreWithScoresAndLimit is the implementation of redis zrevrangebyscore command
// with scores and limit.
func (s *Redis) ZrevrangebyscoreWithScoresAndLimit(ctx context.Context, key string,
	start, stop int64, page, size int) (val []Pair, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		if size <= 0 {
			return nil
		}

		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		v, err := conn.ZRevRangeByScoreWithScores(ctx, key, &red.ZRangeBy{
			Min:    strconv.FormatInt(start, 10),
			Max:    strconv.FormatInt(stop, 10),
			Offset: int64(page * size),
			Count:  int64(size),
		}).Result()
		if err != nil {
			return err
		}

		val = toPairs(v)
		return nil
	}, acceptable)

	return
}

// Zrevrank is the implementation of redis zrevrank command.
func (s *Redis) Zrevrank(ctx context.Context, key, field string) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.ZRevRank(ctx, key, field).Result()
		return err
	}, acceptable)

	return
}

// Zunionstore is the implementation of redis zunionstore command.
func (s *Redis) Zunionstore(ctx context.Context, dest string, store *ZStore) (
	val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}

		val, err = conn.ZUnionStore(ctx, dest, store).Result()
		return err
	}, acceptable)

	return
}

// ExpireAt is the implementation of redis expireat command.
func (s *Redis) ExpireAt(ctx context.Context, key string, expireTime int64) (val bool, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		val, err = conn.ExpireAt(ctx, key, time.Unix(expireTime, 0)).Result()
		return err
	}, acceptable)

	return
}

// GeoRadiusStore is a writing GEORADIUS command.
func (s *Redis) GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query *GeoRadiusQuery) (val int64, err error) {
	err = s.brk.DoWithAcceptable(func() error {
		conn, err := getRedis(s)
		if err != nil {
			return err
		}
		val, err = conn.GeoRadiusStore(ctx, key, longitude, latitude, query).Result()
		return err
	}, acceptable)

	return
}

func getRedis(r *Redis) (red.Cmdable, error) {
	switch r.options.Type {
	case ClusterType:
		return getCluster(r)
	case NodeType:
		return getClient(r)
	default:
		return nil, fmt.Errorf("redis type '%s' is not supported", r.options.Type)
	}
}

func toPairs(vals []red.Z) []Pair {
	pairs := make([]Pair, len(vals))
	for i, val := range vals {
		switch member := val.Member.(type) {
		case string:
			pairs[i] = Pair{
				Key:   member,
				Score: int64(val.Score),
			}
		default:
			pairs[i] = Pair{
				Key:   mapping.Repr(val.Member),
				Score: int64(val.Score),
			}
		}
	}
	return pairs
}

func toStrings(vals []interface{}) []string {
	ret := make([]string, len(vals))
	for i, val := range vals {
		if val == nil {
			ret[i] = ""
		} else {
			switch val := val.(type) {
			case string:
				ret[i] = val
			default:
				ret[i] = mapping.Repr(val)
			}
		}
	}
	return ret
}
