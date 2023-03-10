# LRU
Golang LRU 实现

## 特性
1. 泛型
2. 缓存大小控制更精细，不是 kv 对的数目，而是 kv 实际占用内存大小（需要提供计算函数）

## 使用方法
```go
	cache := lru.New[int, []int](
		100, // 缓存大小
		nil, // 失效回调函数
		func(key int, value []int) int { return 8 + 8*len(value) }, // 缓存大小计算
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
```