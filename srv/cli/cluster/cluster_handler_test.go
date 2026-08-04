package cluster

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewBootstrapHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewBootstrapHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil BootstrapHandler")
	}
}

func TestNewObservabilityHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewObservabilityHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil ObservabilityHandler")
	}
}
