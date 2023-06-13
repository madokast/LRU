package lru

import (
	"reflect"
	"sort"
	"testing"
)

func TestCache_Get(t *testing.T) {
	cache := New[string, int](5, nil, nil)
	cache.Put("abc", 5)
	value, ok := cache.Get("abc")
	if !ok || value != 5 {
		panic(value)
	}
}

func TestCache_Get2(t *testing.T) {
	cache := New[string, int](5, nil, nil)
	cache.Put("abc", 5)
	cache.Put("abc", 6)
	value, ok := cache.Get("abc")
	if !ok || value != 6 {
		panic(value)
	}
}

func TestCache_Put(t *testing.T) {
	cache := New[int, int](5, nil, nil)
	for i := 0; i < 10; i++ {
		cache.Put(i, i*10)
	}
	if cache.Size() != 5 {
		panic(cache.Size())
	}
}

func TestCache_Put2(t *testing.T) {
	cache := New[int, []int](100, nil, func(key int, value []int) int { return 8 + 8*len(value) })
	for i := 0; i < 100; i++ {
		cache.Put(i, make([]int, i%10))
	}
	if cache.Size() > 100 {
		panic(cache.Size())
	}
}

func TestCache_Put3(t *testing.T) {
	cache := New[int, []int](
		100, // 缓存大小
		nil, //
		func(key int, value []int) int { return 8 + 8*len(value) },
	)
	for i := 0; i < 100; i++ {
		cache.Put(i, make([]int, i%10))
		value, ok := cache.Get(i)
		if !ok || len(value) != i%10 {
			panic(len(value))
		}
	}
	for i := 0; i < 100; i++ {
		cache.Remove(i)
	}
	if cache.Size() != 0 {
		panic(cache.Size())
	}
}

func TestCache_Callback(t *testing.T) {
	cache := New[int, []int](0, func(key int, value []int) { value[0] = 123 }, nil)
	value := []int{0}
	cache.Put(1, value)
	if value[0] != 123 {
		panic(value)
	}
}

func TestCache_AllKeys(t *testing.T) {
	cache := New[int, int](5, nil, nil)
	for i := 0; i < 10; i++ {
		cache.Put(i, i*10)
	}
	keys := cache.AllKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	for i := range keys {
		if keys[i] != i+5 {
			panic(keys)
		}
	}
}

func TestCache_RemoveAll(t *testing.T) {
	cache := New[int, int](5, nil, nil)
	for i := 0; i < 10; i++ {
		cache.Put(i, i*10)
	}
	cache.RemoveAll()
	if cache.Size() != 0 {
		panic(cache.Size())
	}
	if cache.Number() != 0 {
		panic(cache.Number())
	}
}

func TestCache_Get1(t *testing.T) {
	cache := New[int, int](2, func(key int, value int) {
		t.Log("remove", key, value)
	}, nil)
	cache.Put(100, 255)
	for i := 0; i < 10; i++ {
		cache.Put(i, i)
		_, ok := cache.Get(100)
		if !ok {
			t.Error("100 removed")
		}
	}
	val, ok := cache.Get(100)
	if !ok {
		t.Error("100 removed")
	}
	t.Log(val)
}

func TestCache_Scan(t *testing.T) {
	cache := New[int, int](5, nil, nil)
	for i := 0; i < 10; i++ {
		cache.Put(i, i*10)
		_, _ = cache.Get(2)
	}
	var ks []int
	cache.Scan(func(k int, v int) bool {
		t.Log(k, v)
		ks = append(ks, k)
		return true
	})
	if len(ks) != 5 {
		panic(ks)
	}
	if ks[0] != 2 {
		panic(ks)
	}
	if ks[1] != 9 {
		panic(ks)
	}
	if ks[4] != 6 {
		panic(ks)
	}
}

func TestCache_RemoveIf(t *testing.T) {
	cache := New[int, int](10, nil, nil)
	for i := 0; i < 10; i++ {
		cache.Put(i, i*10)
	}

	keys := cache.AllKeys()
	t.Log(keys)
	size := cache.Size()
	t.Log(size)
	if !reflect.DeepEqual(keys, []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}) {
		panic(keys)
	}
	if size != 10 {
		panic(size)
	}

	cache.RemoveIf(func(k int) bool {
		return k%2 != 0
	})

	keys = cache.AllKeys()
	t.Log(keys)
	size = cache.Size()
	t.Log(size)
	if !reflect.DeepEqual(keys, []int{8, 6, 4, 2, 0}) {
		panic(keys)
	}
	if size != 5 {
		panic(size)
	}
}

func TestCache_LeastRecentlyUsed(t *testing.T) {
	cache := New[int, int](10, nil, nil)
	cache.Put(1, 2)
	cache.Put(3, 4)

	kv, _ := cache.LeastRecentlyUsed()
	t.Log(kv)
	if kv.key != 1 {
		panic(kv)
	}

	_, _ = cache.Get(1)

	kv, _ = cache.LeastRecentlyUsed()
	t.Log(kv)
	if kv.key != 3 {
		panic(kv)
	}
}
