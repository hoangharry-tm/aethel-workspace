package config

import (
	"sync"
	"time"
)

const cacheTTL = 5 * time.Minute

// ConfigCache holds the single runtime config for this installation.
// Safe for concurrent use.
type ConfigCache struct {
	mu        sync.RWMutex
	value     *OrgConfig
	expiresAt time.Time
}

func NewConfigCache() *ConfigCache {
	return &ConfigCache{}
}

// Get returns the cached config and true if the cache is valid; nil and false on miss.
func (c *ConfigCache) Get() (*OrgConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.value == nil || time.Now().After(c.expiresAt) {
		return nil, false
	}
	return c.value, true
}

// Set stores a new config value, resetting the TTL.
func (c *ConfigCache) Set(cfg *OrgConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = cfg
	c.expiresAt = time.Now().Add(cacheTTL)
}

// Invalidate clears the cache. The next Get will miss and trigger a DB reload.
func (c *ConfigCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = nil
}
