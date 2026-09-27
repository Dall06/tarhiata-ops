package cluster

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestClusterHandlers_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T, repo *mocks.MockConfigRepository)
	}{
		{
			name: "NewBootstrapHandler initializes properly with mock repo",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewBootstrapHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil BootstrapHandler")
				}
			},
		},
		{
			name: "NewObservabilityHandler initializes and reads state",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewObservabilityHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil ObservabilityHandler")
				}
				if err := repo.SaveObservability(domain.SavedObservability{
					DeployType:      "swarm",
					ExternalURL:     "https://obs.local",
					GrafanaPassword: "secretpassword",
				}); err != nil {
					t.Fatalf("failed to save observability in mock: %v", err)
				}
				obs, err := repo.GetObservability()
				if err != nil || obs == nil || obs.DeployType != "swarm" {
					t.Errorf("failed to retrieve observability from mock: %+v (err: %v)", obs, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockConfigRepository()
			tt.testFunc(t, repo)
		})
	}
}
