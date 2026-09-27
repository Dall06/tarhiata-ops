package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/cli/tests/mocks"
)

func TestNewShellHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewShellHandler initializes properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockConfigRepository()
			handler := NewShellHandler(repo)
			if handler == nil {
				t.Fatal("expected handler to not be nil")
			}
		})
	}
}
