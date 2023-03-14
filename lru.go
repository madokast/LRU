package lru

import (
	"container/list"
	"sync"
)

type entry[K comparable, V interface{}] struct {
	key   K
	Value V
}

type Cache[K comparable, V interface{}] struct {
	li             *list.List
	m              map[K]*list.Element
	lock           sync.RWMutex
	expireCallback func(key K, value V)     // 失效回调
	sizeCal        func(key K, value V) int // key/value 大小计算函数
	maxSize        int
	curSize        int // size 并不是 len(m)，而是经过 sizeCal 计算累加值
}

// New 创建一个 LRU 缓存
// maxSize 最大缓存大小。缓存大小不是缓存项的数目，而是由 sizeCal 函数计算每项缓存的大小之和
// expireCallback 缓存失效回调，可以为空
// sizeCal 缓存项大小计算，可以为空，此时函数返回 1
func New[K comparable, V interface{}](maxSize int, expireCallback func(key K, value V), sizeCal func(key K, value V) int) *Cache[K, V] {
	if expireCallback == nil {
		expireCallback = func(key K, value V) {}
	}
	if sizeCal == nil {
		sizeCal = func(key K, value V) int { return 1 }
	}

	return &Cache[K, V]{
		li:             list.New(),
		m:              map[K]*list.Element{},
		expireCallback: expireCallback,
		sizeCal:        sizeCal,
		maxSize:        maxSize,
	}
}

func (c *Cache[K, V]) Put(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	ele, ok := c.m[key]
	if ok {
		c.curSize -= c.sizeCal(key, ele.Value.(*entry[K, V]).Value)
		ele.Value = &entry[K, V]{key: key, Value: value}
		c.curSize += c.sizeCal(key, value)
		c.li.MoveToFront(ele)
	} else {
		ele = c.li.PushFront(&entry[K, V]{key: key, Value: value})
		c.m[key] = ele
		c.curSize += c.sizeCal(key, value)
	}
	c.expireUnlock()
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ele, ok := c.m[key]
	if !ok {
		return value, false
	}
	return ele.Value.(*entry[K, V]).Value, true
}

func (c *Cache[K, V]) AllKeys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var ks []K

	for k := range c.m {
		ks = append(ks, k)
	}

	return ks
}

func (c *Cache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.removeUnlock(key)
}

func (c *Cache[K, V]) removeUnlock(key K) {
	ele, ok := c.m[key]
	if ok {
		delete(c.m, key)
		c.li.Remove(ele)
		c.curSize -= c.sizeCal(key, ele.Value.(*entry[K, V]).Value)
		c.expireCallback(key, ele.Value.(*entry[K, V]).Value)
	}
}

func (c *Cache[K, V]) Size() int {
	return c.curSize
}

func (c *Cache[K, V]) expireUnlock() {
	for c.curSize > c.maxSize && c.li.Len() > 0 {
		back := c.li.Back()
		c.removeUnlock(back.Value.(*entry[K, V]).key)
	}
}
