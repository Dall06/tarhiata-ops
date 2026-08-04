package sys

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewShellHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewShellHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil ShellHandler")
	}
}

func TestNewToolHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewToolHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil ToolHandler")
	}
}

func TestNewConfigHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewConfigHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil ConfigHandler")
	}
}
