package lru

import "testing"

type kv struct {
	key, value interface{}
}

type sizedItem int

func (i sizedItem) Size() int {
	return int(i)
}

func TestLRU(t *testing.T) {
	size := 2
	evicted := make(chan *kv, 1)
	cache := NewWithEvict(size, func(key, value interface{}) {
		evicted <- &kv{key, value}
	})

	cache.Add(1, "a")
	if val, ok := cache.Get(1); !ok || val.(string) != "a" {
		t.Fatalf("Value mismatch: %v != %s", val, "a")
	}

	cache.Add(1, "A")
	if val, ok := cache.Get(1); !ok || val.(string) != "A" {
		t.Fatalf("Value mismatch: %v != %s", val, "A")
	}

	cache.Add(2, "b")
	cache.Add(3, "c")
	if val, ok := cache.Get(1); ok {
		t.Fatalf("Found in cache: %v", val)
	}
	entry := <-evicted
	if entry.key.(int) != 1 || entry.value.(string) != "A" {
		t.Fatalf("Unexpected eviction: %v", entry)
	}

	if n := cache.Len(); n != size {
		t.Fatalf("Invalid cache length: %d, expected: %d", n, size)
	}

	cache.Get(2)
	cache.Add(4, "d")
	if val, ok := cache.Get(3); ok {
		t.Fatalf("Found in cache: %v", val)
	}
	entry = <-evicted
	if entry.key.(int) != 3 || entry.value.(string) != "c" {
		t.Fatalf("Unexpected eviction: %v", entry)
	}
}

func TestLRUBySize(t *testing.T) {
	var evictedSize int
	cache := NewWithEvict(10, func(key, value interface{}) {
		evictedSize += value.(Sizer).Size()
	})
	i1, i2, i3, i4 := sizedItem(2), sizedItem(4), sizedItem(4), sizedItem(4)
	cache.Add("i1", i1)
	if n := cache.Size(); n != 2 {
		t.Fatalf("Invalid cache size: %d != 2", n)
	}
	cache.Add("i2", i2)
	cache.Add("i3", i3)
	if n, l := cache.Size(), cache.Len(); evictedSize != 0 || n != 10 || l != 3 {
		t.Fatalf("Invalid cache: evicted %d, size %d, len %d", evictedSize, n, l)
	}
	cache.Add("i4", i4)
	if n, l := cache.Size(), cache.Len(); evictedSize != 6 || n != 8 || l != 2 {
		t.Fatalf("Invalid cache: evicted %d, size %d, len %d", evictedSize, n, l)
	}

	evictedSize = 0
	i3 = 8
	cache.Add("i3", i3)
	if n, l := cache.Size(), cache.Len(); evictedSize != 4 || n != 8 || l != 1 {
		t.Fatalf("Invalid cache: evicted %d, size %d, len %d", evictedSize, n, l)
	}
}
