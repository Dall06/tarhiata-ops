package domain

import (
	"testing"
)

func TestDomainEntities(t *testing.T) {
	tests := []struct {
		name      string
		testCheck func(t *testing.T)
	}{
		{
			name: "SavedService initialization and fields",
			testCheck: func(t *testing.T) {
				svc := SavedService{
					Name:        "test-app",
					ImageSource: "nginx:latest",
					Port:        80,
				}
				if svc.Name != "test-app" || svc.Port != 80 {
					t.Errorf("unexpected service fields: %+v", svc)
				}
			},
		},
		{
			name: "SavedDatabase initialization and fields",
			testCheck: func(t *testing.T) {
				db := SavedDatabase{
					Name:   "test-db",
					Engine: "postgres",
				}
				if db.Engine != "postgres" || db.Name != "test-db" {
					t.Errorf("unexpected database fields: %+v", db)
				}
			},
		},
		{
			name: "ServiceLink initialization and fields",
			testCheck: func(t *testing.T) {
				link := ServiceLink{
					SourceSvc:  "app-web",
					TargetSvc:  "app-db",
					EnvVarName: "DATABASE_URL",
				}
				if link.SourceSvc != "app-web" || link.TargetSvc != "app-db" {
					t.Errorf("unexpected link fields: %+v", link)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testCheck(t)
		})
	}
}

func TestServerConfig_IsLocal(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ServerConfig
		expected bool
	}{
		{
			name:     "localhost host",
			cfg:      ServerConfig{Host: "localhost"},
			expected: true,
		},
		{
			name:     "127.0.0.1 host",
			cfg:      ServerConfig{Host: "127.0.0.1"},
			expected: true,
		},
		{
			name:     "::1 host",
			cfg:      ServerConfig{Host: "::1"},
			expected: true,
		},
		{
			name:     "local string host",
			cfg:      ServerConfig{Host: "local"},
			expected: true,
		},
		{
			name:     "cloudProvider local",
			cfg:      ServerConfig{Host: "my-vps", CloudProvider: "local"},
			expected: true,
		},
		{
			name:     "remote host ip",
			cfg:      ServerConfig{Host: "192.168.1.50", CloudProvider: "custom"},
			expected: false,
		},
		{
			name:     "remote vps hostname",
			cfg:      ServerConfig{Host: "vps.example.com", CloudProvider: "vultr"},
			expected: false,
		},
		{
			name:     "empty host and custom provider",
			cfg:      ServerConfig{Host: "", CloudProvider: "custom"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsLocal()
			if got != tt.expected {
				t.Errorf("IsLocal() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
