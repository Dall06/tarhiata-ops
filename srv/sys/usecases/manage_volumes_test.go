package usecases

import (
	"os"
	"path/filepath"
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
	_ = vols

	// List files
	files, err := uc.ListVolumeFiles("/opt/data", config)
	if err != nil {
		t.Fatalf("unexpected list volume files error: %v", err)
	}
	_ = files

	// Delete file safely
	err = uc.DeleteFile("/opt/data/old_temp.log", config)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
}
