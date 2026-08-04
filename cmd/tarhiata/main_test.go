package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
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

	svcs, _ := repo.GetServices()
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

	dbs, _ := repo.GetDatabases()
	if len(dbs) != 1 || dbs[0].Name != "my-db" {
		t.Errorf("expected database 'my-db' in DB, got: %v", dbs)
	}
}

func TestHandleListCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	_ = repo.SaveService(domain.SavedService{Name: "svc1", ImageSource: "node:18"})
	_ = repo.SaveDatabase(domain.SavedDatabase{Name: "db1", Engine: "postgres"})

	out := captureOutput(func() {
		handleListCommand(repo)
	})
	if !strings.Contains(out, "svc1") || !strings.Contains(out, "db1") {
		t.Errorf("expected svc1 and db1 in list output, got: %s", out)
	}
}

func TestHandleTopologyCommand(t *testing.T) {
	repo, cleanup := setupTempRepo(t)
	defer cleanup()

	out := captureOutput(func() {
		handleTopologyCommand(repo)
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
