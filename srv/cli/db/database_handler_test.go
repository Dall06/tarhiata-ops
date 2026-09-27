package db

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestDatabaseHandler_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T, repo *mocks.MockConfigRepository)
	}{
		{
			name: "NewDatabaseHandler initializes properly",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewDatabaseHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil DatabaseHandler")
				}
			},
		},
		{
			name: "DatabaseHandler interacts with database repository",
			testFunc: func(t *testing.T, repo *mocks.MockConfigRepository) {
				handler := NewDatabaseHandler(repo)
				if handler == nil {
					t.Fatal("expected non-nil DatabaseHandler")
				}
				if err := repo.SaveDatabase(domain.SavedDatabase{
					Name:         "analytics-db",
					Engine:       "postgres",
					InternalPort: 5432,
				}); err != nil {
					t.Fatalf("failed to save database in mock: %v", err)
				}
				dbs, err := repo.GetDatabases()
				if err != nil || len(dbs) != 1 || dbs[0].Name != "analytics-db" {
					t.Errorf("failed to retrieve saved database: %+v (err: %v)", dbs, err)
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
