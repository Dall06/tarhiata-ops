package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

type mockRepoForBackups struct {
	dbs     []domain.SavedDatabase
	backups []domain.SavedBackup
}

func (m *mockRepoForBackups) SaveServerConfig(config domain.ServerConfig) error { return nil }
func (m *mockRepoForBackups) GetServerConfig() (*domain.ServerConfig, error)    { return nil, nil }
func (m *mockRepoForBackups) SaveService(service domain.SavedService) error     { return nil }
func (m *mockRepoForBackups) GetServices() ([]domain.SavedService, error)        { return nil, nil }
func (m *mockRepoForBackups) GetService(name string) (*domain.SavedService, error) {
	return nil, nil
}
func (m *mockRepoForBackups) DeleteService(name string) error { return nil }
func (m *mockRepoForBackups) SaveDatabase(db domain.SavedDatabase) error {
	m.dbs = append(m.dbs, db)
	return nil
}
func (m *mockRepoForBackups) GetDatabases() ([]domain.SavedDatabase, error) { return m.dbs, nil }
func (m *mockRepoForBackups) GetDatabase(name string) (*domain.SavedDatabase, error) {
	for _, d := range m.dbs {
		if d.Name == name {
			return &d, nil
		}
	}
	return nil, nil
}
func (m *mockRepoForBackups) DeleteDatabase(name string) error                     { return nil }
func (m *mockRepoForBackups) SaveObservability(obs domain.SavedObservability) error { return nil }
func (m *mockRepoForBackups) GetObservability() (*domain.SavedObservability, error) {
	return nil, nil
}
func (m *mockRepoForBackups) SaveServiceLink(link domain.ServiceLink) error  { return nil }
func (m *mockRepoForBackups) GetServiceLinks() ([]domain.ServiceLink, error)  { return nil, nil }
func (m *mockRepoForBackups) DeleteServiceLink(sourceSvc, targetSvc string) error {
	return nil
}
func (m *mockRepoForBackups) SaveBackup(backup domain.SavedBackup) error {
	m.backups = append([]domain.SavedBackup{backup}, m.backups...)
	return nil
}
func (m *mockRepoForBackups) GetBackups() ([]domain.SavedBackup, error) { return m.backups, nil }
func (m *mockRepoForBackups) GetBackupByID(id int) (*domain.SavedBackup, error) {
	if len(m.backups) > 0 {
		return &m.backups[0], nil
	}
	return nil, nil
}
func (m *mockRepoForBackups) DeleteBackup(id int) error                             { return nil }
func (m *mockRepoForBackups) SavePreviewEnv(prev domain.SavedPreviewEnv) error       { return nil }
func (m *mockRepoForBackups) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)     { return nil, nil }
func (m *mockRepoForBackups) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) {
	return nil, nil
}
func (m *mockRepoForBackups) DeletePreviewEnv(name string) error { return nil }
func (m *mockRepoForBackups) SaveRegistryCredential(cred domain.SavedRegistryCredential) error {
	return nil
}
func (m *mockRepoForBackups) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) {
	return nil, nil
}
func (m *mockRepoForBackups) SaveMigrationFile(file domain.MigrationFile) error { return nil }
func (m *mockRepoForBackups) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error) {
	return nil, nil
}
func (m *mockRepoForBackups) DeleteMigrationFile(dbName, filename string) error { return nil }
func (m *mockRepoForBackups) RecordMigrationExecution(dbName, filename, status, logs string) error {
	return nil
}
func (m *mockRepoForBackups) DeleteObservability() error { return nil }
func (m *mockRepoForBackups) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) {
	return nil, nil
}
func (m *mockRepoForBackups) DeleteRegistryCredential(serverURL string) error { return nil }
func (m *mockRepoForBackups) SaveAuditLog(log domain.AuditLog) error             { return nil }
func (m *mockRepoForBackups) GetAuditLogs(limit int) ([]domain.AuditLog, error) { return nil, nil }
func (m *mockRepoForBackups) Close() error                                      { return nil }

func TestCreateSnapshotContainerResolution(t *testing.T) {
	repo := &mockRepoForBackups{
		dbs: []domain.SavedDatabase{
			{Name: "postgres-main", Engine: "postgres", DeployType: "single-node"},
		},
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
