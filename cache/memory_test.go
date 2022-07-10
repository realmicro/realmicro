package cache

import (
	"context"
	"testing"
	"time"
)

var (
	ctx             = context.TODO()
	key string      = "test"
	val interface{} = "hello realmicro"
)

// TestMemCache tests the in-memory cache implementation.
func TestCache(t *testing.T) {
	t.Run("CacheGetMiss", func(t *testing.T) {
		if _, _, err := NewCache().Get(ctx, key); err == nil {
			t.Error("expected to get no value from cache")
		}
	})

	t.Run("CacheGetHit", func(t *testing.T) {
		c := NewCache()

		if err := c.Put(ctx, key, val, 0); err != nil {
			t.Error(err)
		}

		if a, _, err := c.Get(ctx, key); err != nil {
			t.Errorf("Expected a value, got err: %s", err)
		} else if a != val {
			t.Errorf("Expected '%v', got '%v'", val, a)
		}
	})

	t.Run("CacheGetExpired", func(t *testing.T) {
		c := NewCache()
		e := 20 * time.Millisecond

		if err := c.Put(ctx, key, val, e); err != nil {
			t.Error(err)
		}

		<-time.After(25 * time.Millisecond)
		if _, _, err := c.Get(ctx, key); err == nil {
			t.Error("expected to get no value from cache")
		}
	})

	t.Run("CacheGetValid", func(t *testing.T) {
		c := NewCache()
		e := 25 * time.Millisecond

		if err := c.Put(ctx, key, val, e); err != nil {
			t.Error(err)
		}

		<-time.After(20 * time.Millisecond)
		if _, _, err := c.Get(ctx, key); err != nil {
			t.Errorf("expected a value, got err: %s", err)
		}
	})

	t.Run("CacheDeleteMiss", func(t *testing.T) {
		if err := NewCache().Delete(ctx, key); err == nil {
			t.Error("expected to delete no value from cache")
		}
	})

	t.Run("CacheDeleteHit", func(t *testing.T) {
		c := NewCache()

		if err := c.Put(ctx, key, val, 0); err != nil {
			t.Error(err)
		}

		if err := c.Delete(ctx, key); err != nil {
			t.Errorf("Expected to delete an item, got err: %s", err)
		}

		if _, _, err := c.Get(ctx, key); err == nil {
			t.Errorf("Expected error")
		}
	})
}

func TestCacheWithOptions(t *testing.T) {
	t.Run("CacheWithExpiration", func(t *testing.T) {
		c := NewCache(Expiration(20 * time.Millisecond))

		if err := c.Put(ctx, key, val, 0); err != nil {
			t.Error(err)
		}

		<-time.After(25 * time.Millisecond)
		if _, _, err := c.Get(ctx, key); err == nil {
			t.Error("expected to get no value from cache")
		}
	})

	t.Run("CacheWithItems", func(t *testing.T) {
		c := NewCache(Items(map[string]Item{key: {val, 0}}))

		if a, _, err := c.Get(ctx, key); err != nil {
			t.Errorf("Expected a value, got err: %s", err)
		} else if a != val {
			t.Errorf("Expected '%v', got '%v'", val, a)
		}
	})
}
