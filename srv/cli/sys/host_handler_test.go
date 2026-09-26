package sys

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewHostHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewHostHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil HostHandler")
	}
}

func TestResolveTargetConfig(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{
		Config: &domain.ServerConfig{Name: "active-node", Host: "5.6.7.8", IsActive: true},
	}
	handler := NewHostHandler(mockRepo)


	tests := []struct {
		name         string
		serverName   string
		expectedHost string
		expectErr    bool
	}{
		{
			name:         "resolve active server when name is empty",
			serverName:   "",
			expectedHost: "5.6.7.8",
			expectErr:    false,
		},
		{
			name:         "resolve specific server by name",
			serverName:   "active-node",
			expectedHost: "5.6.7.8",
			expectErr:    false,
		},
		{
			name:         "non-existent server returns error",
			serverName:   "ghost-node",
			expectedHost: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := handler.resolveTargetConfig(tt.serverName)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Host != tt.expectedHost {
				t.Errorf("expected host %s, got %s", tt.expectedHost, cfg.Host)
			}
		})
	}
}

func TestRenderBar(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{0.0, "[░░░░░░░░░░]"},
		{50.0, "[█████░░░░░]"},
		{100.0, "[██████████]"},
		{-10.0, "[░░░░░░░░░░]"},
		{120.0, "[██████████]"},
	}

	for _, tt := range tests {
		res := renderBar(tt.percent)
		if res != tt.expected {
			t.Errorf("renderBar(%.1f) = %s, expected %s", tt.percent, res, tt.expected)
		}
	}
}
