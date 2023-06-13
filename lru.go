package lru

import (
	"container/list"
	"sync"
)

type Entry[K comparable, V interface{}] struct {
	key   K
	value V
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
		li:             list.New(), // list<*Entry>
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
		c.curSize -= c.sizeCal(key, ele.Value.(*Entry[K, V]).value)
		ele.Value = &Entry[K, V]{key: key, value: value}
		c.curSize += c.sizeCal(key, value)
		c.li.MoveToFront(ele)
	} else {
		ele = c.li.PushFront(&Entry[K, V]{key: key, value: value})
		c.m[key] = ele
		c.curSize += c.sizeCal(key, value)
	}
	c.expireUnlock()
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	ele, ok := c.m[key]
	if !ok {
		return value, false
	}
	c.li.MoveToFront(ele)
	return ele.Value.(*Entry[K, V]).value, true
}

// LeastRecentlyUsed 返回最近最少使用的 KV，即队列中最后一个 KV
// 如果容器为空，返回 nil, false
func (c *Cache[K, V]) LeastRecentlyUsed() (*Entry[K, V], bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	back := c.li.Back()
	if back != nil {
		return back.Value.(*Entry[K, V]), true
	}

	return nil, false
}

// AllKeys 按照访问先后获取全部 key
func (c *Cache[K, V]) AllKeys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ks := make([]K, 0, c.li.Len())

	cur := c.li.Front()
	for cur != nil {
		ks = append(ks, cur.Value.(*Entry[K, V]).key)
		cur = cur.Next()
	}

	return ks
}

// Scan 按照访问先后遍历所有 KV 对，consumer 返回 bool 指示扫描是否继续
// 扫描不会修改访问先后顺序
func (c *Cache[K, V]) Scan(consumer func(K, V) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	element := c.li.Front()
	for element != nil {
		e := element.Value.(*Entry[K, V])
		if !consumer(e.key, e.value) {
			break
		}
		element = element.Next()
	}
}

func (c *Cache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.removeUnlock(key)
}

// RemoveIf 按照条件移除 KV，不会修改扫描先后顺序
func (c *Cache[K, V]) RemoveIf(remove func(K) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cur := c.li.Front()
	var next *list.Element
	for cur != nil {
		next = cur.Next() // 提前记录 next，因为 cur 可能被移除
		key := cur.Value.(*Entry[K, V]).key
		if remove(key) {
			c.removeUnlock(key) // 正常移除即可
		}
		cur = next // 注意不能用 cur = cur.next()
	}
}

func (c *Cache[K, V]) removeUnlock(key K) {
	ele, ok := c.m[key]
	if ok {
		delete(c.m, key)
		c.li.Remove(ele)
		c.curSize -= c.sizeCal(key, ele.Value.(*Entry[K, V]).value)
		c.expireCallback(key, ele.Value.(*Entry[K, V]).value)
	}
}

func (c *Cache[K, V]) RemoveAll() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for k, ele := range c.m {
		c.expireCallback(k, ele.Value.(*Entry[K, V]).value)
	}
	c.li = list.New()
	c.m = map[K]*list.Element{}
	c.curSize = 0
}

// Size 返回内存占用
func (c *Cache[K, V]) Size() int {
	return c.curSize
}

// Number 返回元素个数
func (c *Cache[K, V]) Number() int {
	return c.li.Len()
}

func (c *Cache[K, V]) expireUnlock() {
	for c.curSize > c.maxSize && c.li.Len() > 0 {
		back := c.li.Back()
		c.removeUnlock(back.Value.(*Entry[K, V]).key)
	}
}
