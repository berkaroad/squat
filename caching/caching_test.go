package caching

import (
	"testing"
	"time"

	"github.com/berkaroad/squat/serialization"
)

func TestMemoryCacheSet(t *testing.T) {
	t.Run("get cache should success", func(t *testing.T) {
		cache := (&MemoryCache{}).Initialize(&serialization.GobSerializer{})
		serialization.MapTo[*Class1]("Class1")
		data1 := &Class1{ID: "agg1"}
		cache.Set("key1", data1, 0)
		dataInterface, loaded := cache.Get("key1", "Class1")
		if !loaded {
			t.Error("cache key 'key1' not found in cache")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("cache key 'key1' in cache, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("cache key 'key1' in cache, not equal with last one")
		}
	})

	t.Run("get cache should fail if expired", func(t *testing.T) {
		cache := (&MemoryCache{}).Initialize(&serialization.GobSerializer{})
		serialization.MapTo[*Class1]("Class1")
		data1 := &Class1{ID: "agg1"}
		cache.Set("key1", data1, time.Millisecond*100)
		dataInterface, loaded := cache.Get("key1", "Class1")
		if !loaded {
			t.Error("cache key 'key1' not found in cache")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("cache key 'key1' in cache, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("cache key 'key1' in cache, not equal with last one")
		}

		time.Sleep(time.Millisecond * 100)
		_, loaded = cache.Get("key1", "Class1")
		if loaded {
			t.Error("cache key 'key1' should expired in cache")
			return
		}
	})
}

func TestMemoryCacheRmove(t *testing.T) {
	t.Run("get cache should fail if removed", func(t *testing.T) {
		cache := (&MemoryCache{}).Initialize(&serialization.GobSerializer{})
		serialization.MapTo[*Class1]("Class1")
		data1 := &Class1{ID: "agg1"}
		cache.Set("key1", data1, 0)
		dataInterface, loaded := cache.Get("key1", "Class1")
		if !loaded {
			t.Error("cache key 'key1' not found in cache")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("cache key 'key1' in cache, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("cache key 'key1' in cache, not equal with last one")
		}

		cache.Remove("key1")
		_, loaded = cache.Get("key1", "Class1")
		if loaded {
			t.Error("cache key 'key1' should removed in cache")
			return
		}
	})
}

type Class1 struct {
	ID string
}
