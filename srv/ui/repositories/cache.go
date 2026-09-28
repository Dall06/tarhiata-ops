package repositories

import (
	"sync"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/ui/domain"
)

// MemoryCache implementa un almacén de caché en memoria con expiración por TTL.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]domain.CacheItem
}

// NewMemoryCache crea una nueva instancia de MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]domain.CacheItem),
	}
}

// Get obtiene un valor de la caché si no ha expirado.
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}
	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Data, true
}

// Set almacena un valor en la caché con un TTL determinado.
func (c *MemoryCache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = domain.CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete elimina una clave de la caché.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Flush limpia todos los elementos de la caché.
func (c *MemoryCache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]domain.CacheItem)
}
