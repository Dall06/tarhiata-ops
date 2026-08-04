package app

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewServiceHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewServiceHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil ServiceHandler")
	}
}
