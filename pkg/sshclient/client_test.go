package sshclient

import (
	"sync"
	"testing"
)

func TestClient_ConcurrencyAndLocking(t *testing.T) {
	client := New()

	if client.CheckConnection() {
		t.Error("expected CheckConnection to be false when conn is nil")
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.CheckConnection()
			_, _, _ = client.RunCommand("echo test")
		}()
	}
	wg.Wait()

	if err := client.Close(); err != nil {
		t.Errorf("unexpected error on Close: %v", err)
	}
}
