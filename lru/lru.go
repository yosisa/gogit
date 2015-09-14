package lru

import "container/list"

type Sizer interface {
	Size() int
}

type Cache struct {
	capacity  int
	size      int
	ll        *list.List
	items     map[interface{}]*list.Element
	onEvicted func(key, value interface{})
}

type entry struct {
	key   interface{}
	value interface{}
	size  int
}

func New(capacity int) *Cache {
	return NewWithEvict(capacity, nil)
}

func NewWithEvict(capacity int, onEvicted func(key, value interface{})) *Cache {
	return &Cache{
		capacity:  capacity,
		ll:        list.New(),
		items:     make(map[interface{}]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *Cache) Add(key interface{}, value interface{}) {
	size := 1
	if sizer, ok := value.(Sizer); ok {
		size = sizer.Size()
	}

	if e, ok := c.items[key]; ok {
		c.ll.MoveToFront(e)
		ent := e.Value.(*entry)
		ent.value = value
		if ent.size != size {
			c.size = c.size - ent.size + size
			ent.size = size
			c.prune()
		}
		return
	}

	e := &entry{key, value, size}
	c.items[key] = c.ll.PushFront(e)
	c.size += e.size
	c.prune()
}

func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	if e, ok := c.items[key]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(*entry).value, true
	}
	return
}

func (c *Cache) Size() int {
	return c.size
}

func (c *Cache) Len() int {
	return c.ll.Len()
}

func (c *Cache) prune() {
	for c.size > c.capacity {
		e := c.ll.Back()
		if e == nil {
			return
		}
		c.ll.Remove(e)
		ent := e.Value.(*entry)
		delete(c.items, ent.key)
		if c.onEvicted != nil {
			c.onEvicted(ent.key, ent.value)
		}
		c.size -= ent.size
	}
}
