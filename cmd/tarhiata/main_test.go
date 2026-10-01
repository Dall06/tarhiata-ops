package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliusecases "github.com/Dall06/tarhiata-ops/srv/cli/usecases"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		return ""
	}
	os.Stdout = w

	fn()

	if errClose := w.Close(); errClose != nil {
		return ""
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, errCopy := io.Copy(&buf, r); errCopy != nil {
		return ""
	}
	return buf.String()
}

func setupTempRepo(t *testing.T) (*repositories.SQLiteRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")
	repo, err := repositories.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create temp sqlite repo: %v", err)
	}
	return repo, func() {
		repo.Close()
	}
}

func TestPrintHelp(t *testing.T) {
	output := captureOutput(func() {
		printHelp()
	})
	if !strings.Contains(output, "Tarhiata-Ops PaaS") {
		t.Errorf("expected printHelp output to contain 'Tarhiata-Ops PaaS', got: %s", output)
	}
}

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Error("expected Version constant to be non-empty")
	}
	if !strings.HasPrefix(Version, "v") {
		t.Errorf("expected Version to start with 'v', got %s", Version)
	}
}

func TestHandleConfigCommand_SaveAndDisplay(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	// 1. Display empty config
	out1 := captureOutput(func() {
		handleConfigCommand(repo, []string{})
	})
	if !strings.Contains(out1, "No hay servidor configurado") {
		t.Errorf("expected 'No hay servidor configurado', got: %s", out1)
	}

	// 2. Set config
	out2 := captureOutput(func() {
		handleConfigCommand(repo, []string{"--host", "10.0.0.1", "--user", "root", "--port", "2222"})
	})
	if !strings.Contains(out2, "guardada exitosamente") {
		t.Errorf("expected success message, got: %s", out2)
	}

	// 3. Display set config
	out3 := captureOutput(func() {
		handleConfigCommand(repo, []string{})
	})
	if !strings.Contains(out3, "10.0.0.1") {
		t.Errorf("expected host 10.0.0.1 in output, got: %s", out3)
	}
}

func TestHandleDeployServiceCommand_LocalSave(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	out := captureOutput(func() {
		handleDeployServiceCommand(repo, nil, []string{"--name", "my-app", "--image", "nginx:latest"})
	})
	if !strings.Contains(out, "registrado en catálogo local") {
		t.Errorf("expected local catalog saved message, got: %s", out)
	}

	svcs, errSvcs := repo.GetServices("")
	if errSvcs != nil {
		t.Fatalf("unexpected error getting services: %v", errSvcs)
	}
	if len(svcs) != 1 || svcs[0].Name != "my-app" {
		t.Errorf("expected service 'my-app' in DB, got: %v", svcs)
	}
}

func TestHandleDatabaseCommand_LocalSave(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	out := captureOutput(func() {
		handleDatabaseCommand(repo, nil, []string{"create", "--name", "my-db", "--engine", "postgres"})
	})
	if !strings.Contains(out, "registrada en catálogo local") {
		t.Errorf("expected local catalog saved message, got: %s", out)
	}

	dbs, errDbs := repo.GetDatabases("")
	if errDbs != nil {
		t.Fatalf("unexpected error getting databases: %v", errDbs)
	}
	if len(dbs) != 1 || dbs[0].Name != "my-db" {
		t.Errorf("expected database 'my-db' in DB, got: %v", dbs)
	}
}

func TestHandleListCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := repo.SaveService(domain.SavedService{Name: "svc1", ImageSource: "node:18"}); err != nil {
		t.Fatalf("unexpected error saving service: %v", err)
	}
	if err := repo.SaveDatabase(domain.SavedDatabase{Name: "db1", Engine: "postgres"}); err != nil {
		t.Fatalf("unexpected error saving database: %v", err)
	}

	out := captureOutput(func() {
		handleListCommand(repo, nil)
	})
	if !strings.Contains(out, "svc1") || !strings.Contains(out, "db1") {
		t.Errorf("expected svc1 and db1 in list output, got: %s", out)
	}
}

func TestHandleTopologyCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	out := captureOutput(func() {
		handleTopologyCommand(repo, nil)
	})
	if !strings.Contains(out, "TOPOLOGY") {
		t.Errorf("expected topology title, got: %s", out)
	}
}

func TestHandleStatusCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	out := captureOutput(func() {
		handleStatusCommand(repo, nil)
	})
	if !strings.Contains(out, "STATUS") {
		t.Errorf("expected status output, got: %s", out)
	}
}

func TestIsValidIdentifier_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "Valid alphanumeric", input: "my-service-1", expected: true},
		{name: "Valid with dots and underscores", input: "api.v1_node", expected: true},
		{name: "Valid single letter", input: "a", expected: true},
		{name: "Valid uppercase", input: "AppService2", expected: true},
		{name: "Invalid with space", input: "my service", expected: false},
		{name: "Invalid with semicolon", input: "service;rm -rf /", expected: false},
		{name: "Invalid with pipe", input: "service|cat", expected: false},
		{name: "Invalid with slash", input: "folder/service", expected: false},
		{name: "Invalid empty", input: "", expected: false},
		{name: "Invalid special char", input: "service$name", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidIdentifier(tc.input)
			if got != tc.expected {
				t.Errorf("isValidIdentifier(%q) = %v, expected %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsJSONOutput(t *testing.T) {
	os.Unsetenv("TARHIATA_JSON")
	if isJSONOutput() {
		t.Error("expected isJSONOutput() to be false when TARHIATA_JSON is not set")
	}

	os.Setenv("TARHIATA_JSON", "true")
	defer os.Unsetenv("TARHIATA_JSON")
	if !isJSONOutput() {
		t.Error("expected isJSONOutput() to be true when TARHIATA_JSON=true")
	}
}

func TestConfirmAction_AutoYes(t *testing.T) {
	os.Setenv("TARHIATA_AUTO_YES", "true")
	defer os.Unsetenv("TARHIATA_AUTO_YES")

	if !confirmAction("¿Eliminar nodo?") {
		t.Error("expected confirmAction to return true when TARHIATA_AUTO_YES is true")
	}
}

func TestHandleListCommand_JSON(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := repo.SaveService(domain.SavedService{Name: "json-svc", ImageSource: "node:18"}); err != nil {
		t.Fatalf("unexpected error saving service: %v", err)
	}

	os.Setenv("TARHIATA_JSON", "true")
	defer os.Unsetenv("TARHIATA_JSON")

	out := captureOutput(func() {
		handleListCommand(repo, nil)
	})

	if !strings.Contains(out, `"services"`) || !strings.Contains(out, `"json-svc"`) {
		t.Errorf("expected JSON output containing services array, got: %s", out)
	}
}

func TestHandleStatusCommand_JSON(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	os.Setenv("TARHIATA_JSON", "true")
	defer os.Unsetenv("TARHIATA_JSON")

	out := captureOutput(func() {
		handleStatusCommand(repo, &domain.ServerConfig{Host: "192.168.1.50", User: "ubuntu"})
	})

	if !strings.Contains(out, `"192.168.1.50"`) || !strings.Contains(out, `"configured": true`) {
		t.Errorf("expected JSON status output, got: %s", out)
	}
}

func TestHandleTopologyCommand_JSON(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := repo.SaveService(domain.SavedService{Name: "topo-svc", Port: 8080}); err != nil {
		t.Fatalf("unexpected error saving service: %v", err)
	}

	os.Setenv("TARHIATA_JSON", "true")
	defer os.Unsetenv("TARHIATA_JSON")

	out := captureOutput(func() {
		handleTopologyCommand(repo, nil)
	})

	if !strings.Contains(out, `"topo-svc"`) || !strings.Contains(out, `"service_links"`) {
		t.Errorf("expected JSON topology output, got: %s", out)
	}
}

func TestHandleLogsCLICommand_Validation(t *testing.T) {
	// Empty args
	outEmpty := captureOutput(func() {
		handleLogsCLICommand(nil, []string{})
	})
	if !strings.Contains(outEmpty, "Uso:") {
		t.Errorf("expected usage on empty args, got: %s", outEmpty)
	}

	// Invalid identifier
	outInvalid := captureOutput(func() {
		handleLogsCLICommand(nil, []string{"invalid;name"})
	})
	if !strings.Contains(outInvalid, "inválido") {
		t.Errorf("expected invalid identifier warning, got: %s", outInvalid)
	}

	// Unconfigured VPS
	outNoVPS := captureOutput(func() {
		handleLogsCLICommand(nil, []string{"my-service"})
	})
	if !strings.Contains(outNoVPS, "no configurado") {
		t.Errorf("expected unconfigured VPS warning, got: %s", outNoVPS)
	}
}

func TestHandleServiceStopCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	if err := repo.SaveService(domain.SavedService{Name: "app-to-stop", ImageSource: "nginx"}); err != nil {
		t.Fatalf("unexpected error saving service: %v", err)
	}

	os.Setenv("TARHIATA_AUTO_YES", "true")
	defer os.Unsetenv("TARHIATA_AUTO_YES")

	out := captureOutput(func() {
		handleServiceStopCommand(repo, nil, []string{"app-to-stop"})
	})

	if !strings.Contains(out, "detenido") {
		t.Errorf("expected stopped confirmation, got: %s", out)
	}

	svc, errGet := repo.GetService("app-to-stop", "")
	if errGet != nil {
		t.Fatalf("unexpected error fetching service: %v", errGet)
	}
	if svc != nil {
		t.Error("expected service to be removed from local repo")
	}
}

func TestHandleTraefikCLICommand_Validation(t *testing.T) {
	outNoVPS := captureOutput(func() {
		handleTraefikCLICommand(nil, []string{"status"})
	})
	if !strings.Contains(outNoVPS, "no configurado") {
		t.Errorf("expected unconfigured VPS warning, got: %s", outNoVPS)
	}

	outInvalid := captureOutput(func() {
		handleTraefikCLICommand(&domain.ServerConfig{Host: "1.2.3.4"}, []string{"invalid-subcmd"})
	})
	if !strings.Contains(outInvalid, "Uso:") {
		t.Errorf("expected usage warning on invalid subcmd, got: %s", outInvalid)
	}
}

func TestHandleSyncCLICommand_Validation(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	outNoVPS := captureOutput(func() {
		handleSyncCLICommand(repo, nil, []string{"import"})
	})
	if !strings.Contains(outNoVPS, "no configurado") {
		t.Errorf("expected unconfigured VPS warning, got: %s", outNoVPS)
	}
}

func TestTUICommand_Execution(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	cfg := &domain.ServerConfig{
		Name: "test-vps",
		Host: "192.168.1.100",
		User: "root",
	}

	out := captureOutput(func() {
		cliusecases.NewDashboardHandler(repo).RenderDashboard(cfg)
	})

	if !strings.Contains(out, "TARHIATA") {
		t.Errorf("expected TUI render output to contain 'TARHIATA', got: %s", out)
	}
}

func TestHandleMigrateCommand_StatusThenUp(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	// Justo después de NewSQLiteRepository, migrate() ya corrió el auto-apply al
	// arrancar, así que no debería quedar nada pendiente.
	outStatus := captureOutput(func() {
		handleMigrateCommand(repo, []string{"status"})
	})
	if !strings.Contains(outStatus, "ESTADO DE MIGRACIONES") {
		t.Errorf("expected status header, got: %s", outStatus)
	}
	if !strings.Contains(outStatus, "migration-001-server-name-scoping.sql") {
		t.Errorf("expected migration-001 listed as applied, got: %s", outStatus)
	}
	if !strings.Contains(outStatus, "Pendientes (0)") {
		t.Errorf("expected 0 pending migrations right after auto-apply, got: %s", outStatus)
	}

	outUp := captureOutput(func() {
		handleMigrateCommand(repo, []string{"up"})
	})
	if !strings.Contains(outUp, "No hay migraciones pendientes") {
		t.Errorf("expected no-op 'up' when nothing is pending, got: %s", outUp)
	}
}

