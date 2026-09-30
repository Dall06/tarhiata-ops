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

func TestManageVolumesUseCase_SanitizePath(t *testing.T) {
	validPath := "/opt/data/my-app/config.json"
	cleaned, err := sanitizePath(validPath)
	if err != nil {
		t.Fatalf("expected valid path, got error: %v", err)
	}
	if cleaned != validPath {
		t.Errorf("expected %s, got %s", validPath, cleaned)
	}

	invalidPath := "/opt/data/../../etc/passwd"
	_, err = sanitizePath(invalidPath)
	if err == nil {
		t.Errorf("expected error for path traversal attempt, got nil")
	}
}

func TestManageVolumesUseCase_Operations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tarhiata_vol_test")
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
	uc := NewManageVolumesUseCase(repo, mockSSH)

	config := domain.ServerConfig{Host: "127.0.0.1"}

	// List volumes
	vols, err := uc.ListVolumes(config)
	if err != nil {
		t.Fatalf("unexpected list volumes error: %v", err)
	}
	if vols == nil {
		t.Fatal("expected non-nil volumes list")
	}

	// List files
	files, err := uc.ListVolumeFiles("/opt/data", config)
	if err != nil {
		t.Fatalf("unexpected list volume files error: %v", err)
	}
	if files == nil {
		t.Fatal("expected non-nil volume files list")
	}

	// Delete file safely
	err = uc.DeleteFile("/opt/data/old_temp.log", config)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

// TestManageVolumesUseCase_ShellInjectionPrevention valida el flujo completo de cada
// operación (no solo sanitizePath): un nombre de archivo con metacaracteres de shell
// debe llegar SIEMPRE citado (comillas simples) al comando real enviado por SSH, para
// que $(...) / backticks / ; no se ejecuten en el host remoto.
func TestManageVolumesUseCase_ShellInjectionPrevention(t *testing.T) {
	const maliciousPath = "/opt/data/$(whoami)"
	const quoted = "'/opt/data/$(whoami)'"
	config := domain.ServerConfig{Host: "127.0.0.1"}

	lastCmd := func(m *mocks.MockSSHExecutor) string {
		if len(m.CommandsExecuted) == 0 {
			t.Fatal("no se ejecutó ningún comando SSH")
		}
		return m.CommandsExecuted[len(m.CommandsExecuted)-1]
	}

	t.Run("ReadFileContent cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if _, err := uc.ReadFileContent(maliciousPath, config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := lastCmd(mockSSH); got != "head -c 200000 "+quoted {
			t.Errorf("comando inseguro: %s", got)
		}
	})

	t.Run("DeleteFile cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if err := uc.DeleteFile(maliciousPath, config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := lastCmd(mockSSH); got != "rm -rf "+quoted {
			t.Errorf("comando inseguro: %s", got)
		}
	})

	t.Run("CreateDirectory cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if err := uc.CreateDirectory(maliciousPath, config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := lastCmd(mockSSH); got != "mkdir -p "+quoted {
			t.Errorf("comando inseguro: %s", got)
		}
	})

	t.Run("DownloadFile cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if _, err := uc.DownloadFile(maliciousPath, config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "base64 " + quoted + " 2>/dev/null || cat " + quoted
		if got := lastCmd(mockSSH); got != want {
			t.Errorf("comando inseguro: %s", got)
		}
	})

	t.Run("WriteFileContent cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if err := uc.WriteFileContent(maliciousPath, "hola", config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := lastCmd(mockSSH)
		if !strings.HasSuffix(got, "| xxd -r -p > "+quoted) {
			t.Errorf("comando inseguro: %s", got)
		}
	})

	t.Run("ListVolumeFiles cita el path", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageVolumesUseCase(nil, mockSSH)
		if _, err := uc.ListVolumeFiles(maliciousPath, config); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := lastCmd(mockSSH)
		if !strings.Contains(got, "mkdir -p "+quoted) || !strings.Contains(got, "for f in "+quoted+"/*") {
			t.Errorf("comando inseguro: %s", got)
		}
	})
}
