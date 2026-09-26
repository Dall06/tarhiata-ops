package usecases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageDomainsUseCase_AddAndRemove(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_domains_test")
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
		Name:      "web-app",
		Domain:    "myapp.com",
		Expose:    true,
		EnableSSL: true,
	})

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageDomainsUseCase(repo, mockSSH)

	config := domain.ServerConfig{Host: "127.0.0.1"}

	// Add custom domain
	err = uc.AddCustomDomain("web-app", "www.myapp.com", "myapp.com", config)
	if err != nil {
		t.Fatalf("unexpected error adding custom domain: %v", err)
	}

	info, err := uc.GetServiceDomains("web-app")
	if err != nil {
		t.Fatalf("unexpected error getting domains: %v", err)
	}
	if info.PrimaryDomain != "myapp.com" {
		t.Errorf("expected primary domain myapp.com, got %s", info.PrimaryDomain)
	}
	if len(info.Rules) != 1 || info.Rules[0].Domain != "www.myapp.com" {
		t.Errorf("expected custom domain www.myapp.com, got %v", info.Rules)
	}

	// Remove custom domain
	err = uc.RemoveCustomDomain("web-app", "www.myapp.com", config)
	if err != nil {
		t.Fatalf("unexpected error removing custom domain: %v", err)
	}

	info2, errGet2 := uc.GetServiceDomains("web-app")
	if errGet2 != nil {
		t.Fatalf("unexpected error getting domains after removal: %v", errGet2)
	}
	if len(info2.Rules) != 0 {
		t.Errorf("expected 0 custom domains after removal, got %d", len(info2.Rules))
	}
}
