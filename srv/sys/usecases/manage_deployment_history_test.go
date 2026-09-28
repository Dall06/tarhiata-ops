package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageDeploymentHistoryUseCase_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		historyID   int
		setupRepo   func(r *mocks.MockConfigRepository)
		setupExec   func(e *mockSecuritySSHExecutor)
		wantErr     bool
	}{
		{
			name:        "Successful rollback to historical version",
			serviceName: "app-prod",
			historyID:   1,
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Deployments = []domain.DeploymentRecord{
					{
						ID:          1,
						ServiceName: "app-prod",
						ImageTag:    "nginx:1.24-alpine",
						DeployedAt:  time.Now().Add(-24 * time.Hour),
						Status:      "success",
					},
				}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker service update": "app-prod updated",
				}
			},
			wantErr: false,
		},
		{
			name:        "Version not found error",
			serviceName: "app-prod",
			historyID:   99,
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Deployments = []domain.DeploymentRecord{}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {},
			wantErr:   true,
		},
		{
			name:        "Mismatched service name",
			serviceName: "app-prod",
			historyID:   1,
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Deployments = []domain.DeploymentRecord{
					{
						ID:          1,
						ServiceName: "other-service",
						ImageTag:    "redis:alpine",
					},
				}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {},
			wantErr:   true,
		},
		{
			name:        "SSH execution error",
			serviceName: "app-prod",
			historyID:   1,
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Deployments = []domain.DeploymentRecord{
					{
						ID:          1,
						ServiceName: "app-prod",
						ImageTag:    "node:20",
					},
				}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.err = errors.New("SSH connection broken")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockConfigRepository()
			tc.setupRepo(repo)

			exec := &mockSecuritySSHExecutor{responses: make(map[string]string)}
			tc.setupExec(exec)

			uc := NewManageDeploymentHistoryUseCase(repo, exec)
			rec, err := uc.RollbackToVersion(tc.serviceName, tc.historyID, domain.ServerConfig{Host: "192.168.1.50"})

			if (err != nil) != tc.wantErr {
				t.Fatalf("RollbackToVersion() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && rec == nil {
				t.Errorf("expected non-nil deployment record on success")
			}
		})
	}
}

func TestManageDeploymentHistoryUseCase_RecordAndGet(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	exec := &mockSecuritySSHExecutor{responses: make(map[string]string)}
	uc := NewManageDeploymentHistoryUseCase(repo, exec)

	rec := domain.DeploymentRecord{
		ServiceName: "worker-api",
		ImageTag:    "golang:1.24",
		Port:        8080,
		Status:      "success",
	}

	if err := uc.RecordDeployment(rec); err != nil {
		t.Fatalf("RecordDeployment() error = %v", err)
	}

	history, err := uc.GetHistory("worker-api", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].ImageTag != "golang:1.24" {
		t.Errorf("expected imageTag golang:1.24, got %s", history[0].ImageTag)
	}
}
