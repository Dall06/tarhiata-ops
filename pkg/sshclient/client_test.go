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
			if client.CheckConnection() {
				t.Error("expected false connection")
			}
			out, code, err := client.RunCommand("echo test")
			if err == nil {
				t.Errorf("expected error running command on uninitialized client, got code %d, out %s", code, out)
			}
		}()
	}
	wg.Wait()

	if err := client.Close(); err != nil {
		t.Errorf("unexpected error on Close: %v", err)
	}
}
