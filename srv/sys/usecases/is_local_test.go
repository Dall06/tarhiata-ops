package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestIsLocal(t *testing.T) {
	tests := []struct {
		name     string
		cfg      domain.ServerConfig
		expected bool
	}{
		{
			name: "local cloud provider",
			cfg: domain.ServerConfig{
				CloudProvider: "local",
			},
			expected: true,
		},
		{
			name: "localhost ip",
			cfg: domain.ServerConfig{
				Host: "127.0.0.1",
			},
			expected: true,
		},
		{
			name: "localhost string",
			cfg: domain.ServerConfig{
				Host: "localhost",
			},
			expected: true,
		},
		{
			name: "remote host",
			cfg: domain.ServerConfig{
				Host:          "192.168.1.100",
				CloudProvider: "custom",
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsLocal(tc.cfg)
			if got != tc.expected {
				t.Errorf("IsLocal(%+v) = %v; want %v", tc.cfg, got, tc.expected)
			}
		})
	}
}
