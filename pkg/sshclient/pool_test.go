package sshclient

import (
	"sync"
	"testing"
	"time"
)

func TestPool_ReuseConnection(t *testing.T) {
	pool := NewPool(10*time.Second, 1*time.Second)
	defer pool.Stop()

	dialCount := 0
	pool.SetDialer(func(host, user, privateKeyPath string, port int) (*Client, error) {
		dialCount++
		c := New()
		c.SetMockConnected(true)
		return c, nil
	})

	tests := []struct {
		name          string
		host          string
		user          string
		port          int
		key           string
		expectNewDial bool
	}{
		{
			name:          "first get dials connection",
			host:          "192.168.1.10",
			user:          "root",
			port:          22,
			key:           "key1",
			expectNewDial: true,
		},
		{
			name:          "second get with same target reuses connection",
			host:          "192.168.1.10",
			user:          "root",
			port:          22,
			key:           "key1",
			expectNewDial: false,
		},
		{
			name:          "different host triggers new dial",
			host:          "192.168.1.20",
			user:          "root",
			port:          22,
			key:           "key1",
			expectNewDial: true,
		},
	}

	var previousClient *Client
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialDials := dialCount
			client, err := pool.Get(tt.host, tt.user, tt.key, tt.port)
			if err != nil {
				t.Fatalf("unexpected error getting client: %v", err)
			}
			if client == nil {
				t.Fatal("expected non-nil client")
			}

			if tt.expectNewDial && dialCount == initialDials {
				t.Errorf("expected new dial to occur, but dialCount stayed at %d", dialCount)
			}
			if !tt.expectNewDial && dialCount != initialDials {
				t.Errorf("expected connection to be reused, but dialCount incremented to %d", dialCount)
			}
			if !tt.expectNewDial && client != previousClient {
				t.Error("expected exact same client instance on reuse")
			}
			previousClient = client
		})
	}
}

func TestPool_ReleaseAndIdleCleanup(t *testing.T) {
	// 50ms idle timeout, 20ms cleanup interval
	pool := NewPool(50*time.Millisecond, 20*time.Millisecond)
	defer pool.Stop()

	pool.SetDialer(func(host, user, privateKeyPath string, port int) (*Client, error) {
		c := New()
		c.SetMockConnected(true)
		return c, nil
	})

	client, err := pool.Get("10.0.0.1", "ubuntu", "k", 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if count := pool.Count(); count != 1 {
		t.Fatalf("expected 1 connection in pool, got %d", count)
	}

	key := PoolKey("10.0.0.1", "ubuntu", "k", 22)
	pool.Release(key)

	// Wait 120ms to allow idle timeout (50ms) and cleaner (20ms) to trigger
	time.Sleep(120 * time.Millisecond)

	if count := pool.Count(); count != 0 {
		t.Fatalf("expected pool to be cleaned up (0 connections), got %d", count)
	}
}

func TestPool_InUsePreventsCleanup(t *testing.T) {
	pool := NewPool(40*time.Millisecond, 15*time.Millisecond)
	defer pool.Stop()

	pool.SetDialer(func(host, user, privateKeyPath string, port int) (*Client, error) {
		c := New()
		c.SetMockConnected(true)
		return c, nil
	})

	_, err := pool.Get("10.0.0.2", "admin", "k", 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait without calling Release (inUse == 1)
	time.Sleep(100 * time.Millisecond)

	if count := pool.Count(); count != 1 {
		t.Fatalf("expected connection to remain because it is in use, got %d", count)
	}

	key := PoolKey("10.0.0.2", "admin", "k", 22)
	pool.Release(key)

	time.Sleep(100 * time.Millisecond)
	if count := pool.Count(); count != 0 {
		t.Fatalf("expected connection to be pruned after release, got %d", count)
	}
}

func TestPool_Invalidate(t *testing.T) {
	pool := NewPool(10*time.Second, 1*time.Second)
	defer pool.Stop()

	pool.SetDialer(func(host, user, privateKeyPath string, port int) (*Client, error) {
		c := New()
		c.SetMockConnected(true)
		return c, nil
	})

	_, err := pool.Get("10.0.0.3", "deploy", "k", 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	key := PoolKey("10.0.0.3", "deploy", "k", 22)
	if err := pool.Invalidate(key); err != nil {
		t.Errorf("unexpected error on invalidate: %v", err)
	}

	if count := pool.Count(); count != 0 {
		t.Errorf("expected 0 connections after invalidate, got %d", count)
	}
}

func TestPool_Concurrency(t *testing.T) {
	pool := NewPool(10*time.Second, 1*time.Second)
	defer pool.Stop()

	pool.SetDialer(func(host, user, privateKeyPath string, port int) (*Client, error) {
		c := New()
		c.SetMockConnected(true)
		return c, nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client, err := pool.Get("10.0.0.4", "root", "k", 22)
			if err != nil {
				t.Errorf("goroutine %d failed to get client: %v", id, err)
				return
			}
			key := PoolKey("10.0.0.4", "root", "k", 22)
			pool.Release(key)
			if client == nil {
				t.Errorf("goroutine %d got nil client", id)
			}
		}(i)
	}
	wg.Wait()

	if count := pool.Count(); count != 1 {
		t.Errorf("expected exactly 1 pooled connection for same host, got %d", count)
	}
}
