package search

import (
	"sort"
	"sync"
	"time"

	"github.com/sharknado/backend/internal/models"
)

type cacheEntry struct {
	results   []models.SearchResult
	expiresAt time.Time
}

type searchCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	maxSize int
}

func newSearchCache(ttl time.Duration) *searchCache {
	c := &searchCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: 200,
	}
	go c.cleanup()
	return c
}

func (c *searchCache) key(query string, services []string) string {
	sorted := make([]string, len(services))
	copy(sorted, services)
	sort.Strings(sorted)
	k := query
	for _, s := range sorted {
		k += "|" + s
	}
	return k
}

func (c *searchCache) get(query string, services []string) []models.SearchResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[c.key(query, services)]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}

	// Return a copy
	out := make([]models.SearchResult, len(entry.results))
	copy(out, entry.results)
	return out
}

func (c *searchCache) set(query string, services []string, results []models.SearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[c.key(query, services)] = cacheEntry{
		results:   results,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *searchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}

func (c *searchCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range c.entries {
		if oldestKey == "" || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *searchCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
