package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type mockRepoForRegistry struct {
	creds map[string]domain.SavedRegistryCredential
}

func newMockRepoForRegistry() *mockRepoForRegistry {
	return &mockRepoForRegistry{
		creds: make(map[string]domain.SavedRegistryCredential),
	}
}

func (m *mockRepoForRegistry) SaveServerConfig(config domain.ServerConfig) error { return nil }
func (m *mockRepoForRegistry) GetServerConfig() (*domain.ServerConfig, error)    { return nil, nil }
func (m *mockRepoForRegistry) SaveService(svc domain.SavedService) error          { return nil }
func (m *mockRepoForRegistry) GetServices() ([]domain.SavedService, error)         { return nil, nil }
func (m *mockRepoForRegistry) GetService(name string) (*domain.SavedService, error) { return nil, nil }
func (m *mockRepoForRegistry) DeleteService(name string) error                     { return nil }
func (m *mockRepoForRegistry) SaveDatabase(db domain.SavedDatabase) error         { return nil }
func (m *mockRepoForRegistry) GetDatabases() ([]domain.SavedDatabase, error)        { return nil, nil }
func (m *mockRepoForRegistry) GetDatabase(name string) (*domain.SavedDatabase, error) { return nil, nil }
func (m *mockRepoForRegistry) DeleteDatabase(name string) error                    { return nil }
func (m *mockRepoForRegistry) SaveObservability(obs domain.SavedObservability) error { return nil }
func (m *mockRepoForRegistry) GetObservability() (*domain.SavedObservability, error) { return nil, nil }
func (m *mockRepoForRegistry) DeleteObservability() error                          { return nil }
func (m *mockRepoForRegistry) SaveServiceLink(link domain.ServiceLink) error       { return nil }
func (m *mockRepoForRegistry) GetServiceLinks() ([]domain.ServiceLink, error)      { return nil, nil }
func (m *mockRepoForRegistry) DeleteServiceLink(sourceSvc, targetSvc string) error { return nil }
func (m *mockRepoForRegistry) SavePreviewEnv(env domain.SavedPreviewEnv) error      { return nil }
func (m *mockRepoForRegistry) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)   { return nil, nil }
func (m *mockRepoForRegistry) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) { return nil, nil }
func (m *mockRepoForRegistry) DeletePreviewEnv(name string) error                 { return nil }

func (m *mockRepoForRegistry) SaveRegistryCredential(cred domain.SavedRegistryCredential) error {
	m.creds[cred.Server] = cred
	return nil
}
func (m *mockRepoForRegistry) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) {
	var res []domain.SavedRegistryCredential
	for _, c := range m.creds {
		res = append(res, c)
	}
	return res, nil
}
func (m *mockRepoForRegistry) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) {
	c, ok := m.creds[server]
	if !ok {
		return nil, nil
	}
	return &c, nil
}
func (m *mockRepoForRegistry) DeleteRegistryCredential(server string) error {
	delete(m.creds, server)
	return nil
}
func (m *mockRepoForRegistry) SaveMigrationFile(file domain.MigrationFile) error              { return nil }
func (m *mockRepoForRegistry) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error){ return nil, nil }
func (m *mockRepoForRegistry) DeleteMigrationFile(dbName, filename string) error              { return nil }
func (m *mockRepoForRegistry) RecordMigrationExecution(dbName, filename, status, logs string) error { return nil }
func (m *mockRepoForRegistry) SaveBackup(b domain.SavedBackup) error               { return nil }
func (m *mockRepoForRegistry) GetBackups() ([]domain.SavedBackup, error)         { return nil, nil }
func (m *mockRepoForRegistry) GetBackupByID(id int) (*domain.SavedBackup, error) { return nil, nil }
func (m *mockRepoForRegistry) DeleteBackup(id int) error                          { return nil }
func (m *mockRepoForRegistry) SaveAuditLog(log domain.AuditLog) error             { return nil }
func (m *mockRepoForRegistry) GetAuditLogs(limit int) ([]domain.AuditLog, error) { return nil, nil }
func (m *mockRepoForRegistry) Close() error { return nil }

func TestManageRegistryAuthUseCase_SaveAndList(t *testing.T) {
	repo := newMockRepoForRegistry()
	uc := NewManageRegistryAuthUseCase(repo, nil)

	cred := domain.SavedRegistryCredential{
		Server:   "ghcr.io",
		Username: "myuser",
		Password: "ghp_secret_token",
	}

	err := uc.Save(cred, domain.ServerConfig{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	list, err := uc.List()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}
	if list[0].Server != "ghcr.io" || list[0].Username != "myuser" {
		t.Errorf("unexpected credential: %+v", list[0])
	}
}

func TestManageRegistryAuthUseCase_Delete(t *testing.T) {
	repo := newMockRepoForRegistry()
	uc := NewManageRegistryAuthUseCase(repo, nil)

	cred := domain.SavedRegistryCredential{
		Server:   "docker.io",
		Username: "dockeruser",
		Password: "password123",
	}
	_ = uc.Save(cred, domain.ServerConfig{})

	err := uc.Delete("docker.io", domain.ServerConfig{})
	if err != nil {
		t.Fatalf("expected no error deleting, got %v", err)
	}

	list, _ := uc.List()
	if len(list) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(list))
	}
}
