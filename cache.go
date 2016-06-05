package ldbl

import (
	"log"
	"sync"
)

// ItemsCache is used inside Storage implementations for caching already loaded items
type ItemsCache struct {
	sync.RWMutex
	OptionalLogger
	data       map[string]map[uint64]Loadable
	itemsCount int
	maxSize    int
	logger     *log.Logger
}

func NewItemsCache(maxSize int) *ItemsCache {
	c := &ItemsCache{maxSize: maxSize}
	c.LogPrefix = "Cache"
	c.data = make(map[string]map[uint64]Loadable)
	return c
}

func (c *ItemsCache) Add(item Loadable) {
	if c.maxSize == 0 {
		return
	}
	cname := item.CollectionName()
	c.Lock()
	defer c.Unlock()
	if c.itemsCount == c.maxSize {
		c.init()
	}
	if _, set := c.data[cname]; !set {
		c.data[cname] = make(map[uint64]Loadable)
	}
	c.data[cname][item.Id()] = item
	c.itemsCount++
	c.Log("%s#%d cached", item.CollectionName(), item.Id())
}

func (c *ItemsCache) Lookup(forItem Loadable, id uint64) bool {
	if c.maxSize == 0 {
		return false
	}
	cname := forItem.CollectionName()
	c.RLock()
	defer c.RUnlock()
	if _, set := c.data[cname]; !set {
		return false
	}
	if result, found := c.data[cname][id]; found {
		if result == nil {
			return false
		}
		if result, canLoadFields := result.(Storable); canLoadFields {
			forItem.Fill(id, result.Fields())
			c.Log("Hit: %s#%d", forItem.CollectionName(), forItem.Id())
			return true
		}
	}
	return false
}

func (c *ItemsCache) Remove(item Loadable) {
	if c.maxSize == 0 {
		return
	}
	cname := item.CollectionName()
	c.Lock()
	defer c.Unlock()
	if _, set := c.data[cname]; !set {
		return
	}
	if _, found := c.data[cname][item.Id()]; found {
		c.data[cname][item.Id()] = nil
		c.Log("%s#%d removed", item.CollectionName(), item.Id())
	}
}

func (c *ItemsCache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.init()
}

func (c *ItemsCache) init() {
	c.data = make(map[string]map[uint64]Loadable)
	c.itemsCount = 0
	c.Log("All items cleared")
}
