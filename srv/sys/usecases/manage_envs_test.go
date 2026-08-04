package usecases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageEnvVarsUseCase_ParseAndFormat(t *testing.T) {
	raw := `
# Comentario de prueba
PORT=8080
DATABASE_URL="postgres://user:pass@localhost:5432/db"
STRIPE_KEY='sk_test_12345'
`
	parsed := ParseEnvContent(raw)

	if parsed["PORT"] != "8080" {
		t.Errorf("expected PORT=8080, got %s", parsed["PORT"])
	}
	if parsed["DATABASE_URL"] != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("expected DATABASE_URL decoded cleanly, got %s", parsed["DATABASE_URL"])
	}
	if parsed["STRIPE_KEY"] != "sk_test_12345" {
		t.Errorf("expected STRIPE_KEY=sk_test_12345, got %s", parsed["STRIPE_KEY"])
	}

	formatted := FormatEnvMap(parsed)
	if formatted == "" {
		t.Errorf("expected formatted env string, got empty")
	}
}

func TestManageEnvVarsUseCase_UpdateAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_env_test")
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

	// Seed service
	repo.SaveService(domain.SavedService{
		Name:    "web-api",
		EnvVars: "INITIAL_KEY=val1\n",
	})

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageEnvVarsUseCase(repo, mockSSH)

	// Get initial env
	raw, envMap, err := uc.GetEnvVars("web-api")
	if err != nil {
		t.Fatalf("unexpected get env error: %v", err)
	}
	if envMap["INITIAL_KEY"] != "val1" {
		t.Errorf("expected INITIAL_KEY=val1, got %s (raw: %s)", envMap["INITIAL_KEY"], raw)
	}

	// Update bulk env
	newEnv := "PORT=3000\nNODE_ENV=production\nAPI_KEY=secret99\n"
	err = uc.UpdateEnvVars("web-api", newEnv, domain.ServerConfig{Host: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected update env error: %v", err)
	}

	// Verify persistence
	_, updatedMap, err := uc.GetEnvVars("web-api")
	if err != nil {
		t.Fatalf("unexpected error re-fetching env: %v", err)
	}
	if updatedMap["PORT"] != "3000" || updatedMap["NODE_ENV"] != "production" {
		t.Errorf("env vars not updated correctly: %v", updatedMap)
	}
}
