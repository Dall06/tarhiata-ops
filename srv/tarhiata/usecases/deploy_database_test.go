package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/tests/mocks"
)

func TestDeployDatabase_Execute(t *testing.T) {
	tests := []struct {
		name              string
		db                domain.SavedDatabase
		expectCmdContains []string
		expectError       bool
	}{
		{
			name: "Deploy Postgres single-node",
			db: domain.SavedDatabase{
				Name:           "mi-postgres",
				Engine:         "postgres",
				DeployType:     "single-node",
				Password:       "secure-pass",
				VolumeHostPath: "/opt/data/pg",
			},
			expectCmdContains: []string{
				"mkdir -p /host/opt/data/pg",
				"chown -R 70:70",
				"docker service rm tarhiata-db-mi-postgres",
				"docker service create",
				"--name tarhiata-db-mi-postgres",
				"POSTGRES_PASSWORD='secure-pass'",
				`--constraint "node.role == manager"`,
			},
		},
		{
			name: "Deploy Mongo multi-node",
			db: domain.SavedDatabase{
				Name:           "mi-mongo",
				Engine:         "mongo",
				DeployType:     "multi-node",
				Password:       "secure-pass-mongo",
				VolumeHostPath: "/opt/data/mongo",
			},
			expectCmdContains: []string{
				"mkdir -p /host/opt/data/mongo",
				"chown -R 999:999",
				"docker service rm tarhiata-db-mi-mongo",
				"docker service create",
				"--name tarhiata-db-mi-mongo",
				"MONGO_INITDB_ROOT_PASSWORD='secure-pass-mongo'",
				`--constraint "node.labels.type == db_mi-mongo"`,
			},
		},
		{
			name: "Fail on external DB",
			db: domain.SavedDatabase{
				DeployType: "external",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := mocks.NewMockSSHExecutor()
			uc := NewDeployDatabaseUseCase(mockSSH)
			config := domain.ServerConfig{}

			err := uc.Execute(tt.db, config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			allCmds := strings.Join(mockSSH.CommandsExecuted, " ||| ")
			for _, expected := range tt.expectCmdContains {
				if !strings.Contains(allCmds, expected) {
					t.Errorf("expected command to contain '%s', but it wasn't found in: %s", expected, allCmds)
				}
			}
		})
	}
}
