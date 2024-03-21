package redis

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	red "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func runOnRedis(t *testing.T, fn func(client *Redis)) {
	s := miniredis.RunT(t)
	fn(New(
		Addrs(strings.Split(s.Addr(), ",")),
	))
}

func TestRedis_GetSet(t *testing.T) {
	ctx := context.Background()
	t.Run("set_get", func(t *testing.T) {
		runOnRedis(t, func(client *Redis) {
			val, err := client.GetSet(ctx, "hello", "world")
			assert.Nil(t, err)
			assert.Equal(t, "", val)
			val, err = client.Get(ctx, "hello")
			assert.Nil(t, err)
			assert.Equal(t, "world", val)
			val, err = client.GetSet(ctx, "hello", "newworld")
			assert.Nil(t, err)
			assert.Equal(t, "world", val)
			val, err = client.Get(ctx, "hello")
			assert.Nil(t, err)
			assert.Equal(t, "newworld", val)
			ret, err := client.Del(ctx, "hello")
			assert.Nil(t, err)
			assert.Equal(t, 1, ret)
		})
	})
}

func TestRedisGeo(t *testing.T) {
	r1, err := miniredis.Run()
	assert.NoError(t, err)
	defer r1.Close()

	ctx := context.Background()
	t.Run("geo", func(t *testing.T) {
		runOnRedis(t, func(client *Redis) {
			client.Ping(ctx)
			geoLocation := []*GeoLocation{{Longitude: 13.361389, Latitude: 38.115556, Name: "Palermo"},
				{Longitude: 15.087269, Latitude: 37.502669, Name: "Catania"}}
			v, err := client.GeoAdd(ctx, "sicily", geoLocation...)
			assert.Nil(t, err)
			assert.Equal(t, int64(2), v)

			v2, err := client.GeoDist(ctx, "sicily", "Palermo", "Catania", "m")
			assert.Nil(t, err)
			assert.Equal(t, 166274, int(v2))

			v3, err := client.GeoPos(ctx, "sicily", "Palermo", "Catania")
			assert.Nil(t, err)
			assert.Equal(t, int64(v3[0].Longitude), int64(13))
			assert.Equal(t, int64(v3[0].Latitude), int64(38))
			assert.Equal(t, int64(v3[1].Longitude), int64(15))
			assert.Equal(t, int64(v3[1].Latitude), int64(37))

			v4, err := client.GeoRadius(ctx, "sicily", 15, 37, &red.GeoRadiusQuery{
				WithDist: true,
				Unit:     "km", Radius: 200,
			})
			assert.Nil(t, err)
			assert.Equal(t, int64(v4[0].Dist), int64(190))
			assert.Equal(t, int64(v4[1].Dist), int64(56))

			geoLocation2 := []*GeoLocation{{Longitude: 13.583333, Latitude: 37.316667, Name: "Agrigento"}}
			v5, err := client.GeoAdd(ctx, "sicily", geoLocation2...)
			assert.Nil(t, err)
			assert.Equal(t, int64(1), v5)

			v6, err := client.GeoRadiusByMember(ctx, "sicily", "Agrigento",
				&red.GeoRadiusQuery{Unit: "km", Radius: 100})
			assert.Nil(t, err)
			assert.Equal(t, v6[0].Name, "Agrigento")
			assert.Equal(t, v6[1].Name, "Palermo")
		})
	})
}
