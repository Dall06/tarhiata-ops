package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/cli/tests/mocks"
)

func TestNewBootstrapHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "NewBootstrapHandler initializes properly with mock repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockConfigRepository()
			handler := NewBootstrapHandler(repo)
			if handler == nil {
				t.Fatal("expected handler to not be nil")
			}
		})
	}
}
