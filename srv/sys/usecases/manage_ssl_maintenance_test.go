package usecases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageSSLMaintenanceUseCase_TableDriven(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_ssl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("cleanup error: %v", rmErr)
		}
	}()

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repositories.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed sqlite init: %v", err)
	}
	defer func() {
		if clErr := repo.Close(); clErr != nil {
			t.Logf("repo close error: %v", clErr)
		}
	}()

	// Seed test services
	if err := repo.SaveService(domain.SavedService{
		Name:      "shop-web",
		Domain:    "shop.tarhiata.internal",
		Expose:    true,
		EnableSSL: true,
	}); err != nil {
		t.Fatalf("failed saving service: %v", err)
	}

	if err := repo.SaveService(domain.SavedService{
		Name:      "api-internal",
		Domain:    "api.tarhiata.internal",
		Expose:    true,
		EnableSSL: false,
	}); err != nil {
		t.Fatalf("failed saving service: %v", err)
	}

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageSSLMaintenanceUseCase(repo, mockSSH)

	tests := []struct {
		name        string
		action      func() error
		expectError bool
	}{
		{
			name: "Inspect SSL across registered exposed services",
			action: func() error {
				items, err := uc.InspectSSL()
				if err != nil {
					return err
				}
				if len(items) != 2 {
					t.Fatalf("expected 2 items, got %d", len(items))
				}
				return nil
			},
			expectError: false,
		},
		{
			name: "Enable maintenance mode for shop-web",
			action: func() error {
				cfg := domain.ServerConfig{Host: "127.0.0.1"}
				return uc.ToggleMaintenanceMode("shop-web", true, cfg)
			},
			expectError: false,
		},
		{
			name: "Disable maintenance mode for shop-web",
			action: func() error {
				cfg := domain.ServerConfig{Host: "127.0.0.1"}
				return uc.ToggleMaintenanceMode("shop-web", false, cfg)
			},
			expectError: false,
		},
		{
			name: "Toggle maintenance for non-existent service should return error",
			action: func() error {
				cfg := domain.ServerConfig{Host: "127.0.0.1"}
				return uc.ToggleMaintenanceMode("non-existent-svc", true, cfg)
			},
			expectError: true,
		},
		{
			name: "Reload Traefik configuration",
			action: func() error {
				cfg := domain.ServerConfig{Host: "127.0.0.1"}
				return uc.ReloadTraefik(cfg)
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.action()
			if tc.expectError && err == nil {
				t.Errorf("[%s] expected error, got nil", tc.name)
			}
			if !tc.expectError && err != nil {
				t.Errorf("[%s] unexpected error: %v", tc.name, err)
			}
		})
	}
}
