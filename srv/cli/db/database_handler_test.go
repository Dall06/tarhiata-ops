package db

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewDatabaseHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewDatabaseHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil DatabaseHandler")
	}
}
