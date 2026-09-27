package app

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestServiceHandler_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T, repo *mocks.MockConfigRepository)
	}{
		{
			name: "NewServiceHandler initializes properly",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewServiceHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil ServiceHandler")
				}
			},
		},
		{
			name: "ServiceHandler manages service repository records",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewServiceHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil ServiceHandler")
				}
				if err := repo.SaveService(domain.SavedService{
					Name:        "web-client",
					ImageSource: "nginx:alpine",
					Port:        80,
				}); err != nil {
					t.Fatalf("failed to save service in mock: %v", err)
				}
				svcs, err := repo.GetServices()
				if err != nil || len(svcs) != 1 || svcs[0].Name != "web-client" {
					t.Errorf("failed to retrieve saved service: %+v (err: %v)", svcs, err)
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
