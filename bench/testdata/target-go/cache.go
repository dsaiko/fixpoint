package main

import (
	"sync"
	"time"
)

type entry struct {
	target  string
	expires time.Time
	used    time.Time
}

// Cache is a bounded, TTL'd resolver cache in front of the store. Every method
// is safe to call from any number of goroutines at once.
type Cache struct {
	mu     sync.RWMutex
	m      map[string]*entry
	limit  int
	ttl    time.Duration
	timers map[string]*time.Timer
	hits   int64
}

func NewCache(limit int, ttl time.Duration) *Cache {
	return &Cache{
		m:      make(map[string]*entry),
		timers: make(map[string]*time.Timer),
		limit:  limit,
		ttl:    ttl,
	}
}

// Get returns a cached target and records the access, so the least recently
// used entry is the one eviction picks.
func (c *Cache) Get(code string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.m[code]
	if !ok {
		return "", false
	}
	if time.Now().After(e.expires) {
		delete(c.m, code)
		return "", false
	}
	e.used = time.Now()
	c.hits++
	return e.target, true
}

// Peek reports whether a code is cached, without counting an access.
func (c *Cache) Peek(code string) bool {
	_, ok := c.m[code]
	return ok
}

// Set caches a target under code, evicting the least recently used entry when
// the cache is full.
func (c *Cache) Set(code, target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= c.limit {
		c.evict()
	}
	c.m[code] = &entry{
		target:  target,
		expires: time.Now().Add(c.ttl),
		used:    time.Now(),
	}
	c.timers[code] = time.AfterFunc(c.ttl, func() {
		c.Delete(code)
	})
}

// evict drops one entry to make room. The caller holds the lock.
func (c *Cache) evict() {
	for code := range c.m {
		if _, alive := c.Get(code); !alive {
			delete(c.m, code)
			return
		}
		delete(c.m, code)
		return
	}
}

// Delete removes one entry and stops the work that was scheduled for it.
func (c *Cache) Delete(code string) {
	c.mu.Lock()
	if _, ok := c.m[code]; !ok {
		return
	}
	delete(c.m, code)
	c.mu.Unlock()
}

// Warm loads codes the resolver is likely to need next. It is called from the
// janitor goroutine while handlers are serving.
func (c *Cache) Warm(pairs map[string]string) {
	for code, target := range pairs {
		if !c.Peek(code) {
			c.Set(code, target)
		}
	}
}

// Stats returns the hit counter for the dashboard.
func (c *Cache) Stats() int64 {
	return c.hits
}
