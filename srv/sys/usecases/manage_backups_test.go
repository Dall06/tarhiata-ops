package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestCreateSnapshotContainerResolution(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Databases = []domain.SavedDatabase{
		{Name: "postgres-main", Engine: "postgres", DeployType: "single-node"},
	}
	mockSSH := mocks.NewMockSSHExecutor()
	mockSSH.MockResponses["docker exec"] = &domain.CommandResult{Output: "success", ExitCode: 0}

	uc := NewManageBackupsUseCase(repo, mockSSH)
	req := domain.BackupRequest{
		TargetName: "postgres-main",
		TargetType: "database",
	}

	cfg := domain.ServerConfig{Host: "127.0.0.1", User: "root"}
	backup, err := uc.CreateSnapshot(req, cfg)
	if err != nil {
		t.Fatalf("expected snapshot success, got error: %v", err)
	}

	if backup == nil || backup.TargetName != "postgres-main" {
		t.Errorf("snapshot returned unexpected backup: %+v", backup)
	}
}
