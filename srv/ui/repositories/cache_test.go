package repositories

import (
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// 1. Set and Get
	cache.Set("user_1", "diego", 100*time.Millisecond)
	val, ok := cache.Get("user_1")
	if !ok || val != "diego" {
		t.Fatalf("expected to find 'diego', got %v (found=%v)", val, ok)
	}

	// 2. Expiration
	time.Sleep(120 * time.Millisecond)
	_, ok = cache.Get("user_1")
	if ok {
		t.Fatalf("expected key to be expired, but it was found")
	}

	// 3. Delete
	cache.Set("user_2", "antigravity", 1*time.Minute)
	cache.Delete("user_2")
	_, ok = cache.Get("user_2")
	if ok {
		t.Fatalf("expected deleted key not to be found")
	}

	// 4. Flush
	cache.Set("k1", "v1", 1*time.Minute)
	cache.Set("k2", "v2", 1*time.Minute)
	cache.Flush()
	_, ok = cache.Get("k1")
	if ok {
		t.Fatalf("expected flushed cache to be empty")
	}
}
