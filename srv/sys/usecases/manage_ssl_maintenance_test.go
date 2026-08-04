package usecases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageSSLMaintenanceUseCase_InspectAndToggle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_ssl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repositories.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed sqlite init: %v", err)
	}
	defer repo.Close()

	// Seed test service
	repo.SaveService(domain.SavedService{
		Name:      "shop-web",
		Domain:    "shop.tarhiata.internal",
		Expose:    true,
		EnableSSL: true,
	})

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageSSLMaintenanceUseCase(repo, mockSSH)

	// Inspect SSL
	items, err := uc.InspectSSL()
	if err != nil {
		t.Fatalf("unexpected error inspecting SSL: %v", err)
	}
	if len(items) == 0 {
		t.Errorf("expected at least 1 domain ssl item, got 0")
	}

	// Toggle maintenance mode
	config := domain.ServerConfig{Host: "127.0.0.1"}
	err = uc.ToggleMaintenanceMode("shop-web", true, config)
	if err != nil {
		t.Fatalf("unexpected error enabling maintenance: %v", err)
	}

	err = uc.ToggleMaintenanceMode("shop-web", false, config)
	if err != nil {
		t.Fatalf("unexpected error disabling maintenance: %v", err)
	}
}
