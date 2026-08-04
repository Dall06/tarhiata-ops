package mocks

import (
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type MockConfigRepository struct {
	Config        *domain.ServerConfig
	Services      []domain.SavedService
	Databases     []domain.SavedDatabase
	Observability *domain.SavedObservability
	Links         []domain.ServiceLink
	Previews      []domain.SavedPreviewEnv
	Registries    []domain.SavedRegistryCredential
	Migrations    []domain.MigrationFile
	Backups       []domain.SavedBackup
	AuditLogs     []domain.AuditLog
}

func (m *MockConfigRepository) SaveServerConfig(config domain.ServerConfig) error {
	m.Config = &config
	return nil
}

func (m *MockConfigRepository) GetServerConfig() (*domain.ServerConfig, error) {
	return m.Config, nil
}

func (m *MockConfigRepository) SaveService(svc domain.SavedService) error {
	m.Services = append(m.Services, svc)
	return nil
}

func (m *MockConfigRepository) GetServices() ([]domain.SavedService, error) {
	return m.Services, nil
}

func (m *MockConfigRepository) GetService(name string) (*domain.SavedService, error) {
	for _, s := range m.Services {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteService(name string) error {
	var filtered []domain.SavedService
	for _, s := range m.Services {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	m.Services = filtered
	return nil
}

func (m *MockConfigRepository) SaveDatabase(db domain.SavedDatabase) error {
	m.Databases = append(m.Databases, db)
	return nil
}

func (m *MockConfigRepository) GetDatabases() ([]domain.SavedDatabase, error) {
	return m.Databases, nil
}

func (m *MockConfigRepository) GetDatabase(name string) (*domain.SavedDatabase, error) {
	for _, d := range m.Databases {
		if d.Name == name {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteDatabase(name string) error {
	var filtered []domain.SavedDatabase
	for _, d := range m.Databases {
		if d.Name != name {
			filtered = append(filtered, d)
		}
	}
	m.Databases = filtered
	return nil
}

func (m *MockConfigRepository) SaveObservability(obs domain.SavedObservability) error {
	m.Observability = &obs
	return nil
}

func (m *MockConfigRepository) GetObservability() (*domain.SavedObservability, error) {
	return m.Observability, nil
}

func (m *MockConfigRepository) DeleteObservability() error {
	m.Observability = nil
	return nil
}

func (m *MockConfigRepository) SaveServiceLink(link domain.ServiceLink) error {
	m.Links = append(m.Links, link)
	return nil
}

func (m *MockConfigRepository) GetServiceLinks() ([]domain.ServiceLink, error) {
	return m.Links, nil
}

func (m *MockConfigRepository) DeleteServiceLink(sourceSvc, targetSvc string) error {
	var filtered []domain.ServiceLink
	for _, l := range m.Links {
		if !(l.SourceSvc == sourceSvc && l.TargetSvc == targetSvc) {
			filtered = append(filtered, l)
		}
	}
	m.Links = filtered
	return nil
}

func (m *MockConfigRepository) SavePreviewEnv(env domain.SavedPreviewEnv) error {
	m.Previews = append(m.Previews, env)
	return nil
}

func (m *MockConfigRepository) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error) {
	return m.Previews, nil
}

func (m *MockConfigRepository) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) {
	for _, p := range m.Previews {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeletePreviewEnv(name string) error {
	var filtered []domain.SavedPreviewEnv
	for _, p := range m.Previews {
		if p.Name != name {
			filtered = append(filtered, p)
		}
	}
	m.Previews = filtered
	return nil
}

func (m *MockConfigRepository) SaveRegistryCredential(cred domain.SavedRegistryCredential) error {
	m.Registries = append(m.Registries, cred)
	return nil
}

func (m *MockConfigRepository) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) {
	return m.Registries, nil
}

func (m *MockConfigRepository) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) {
	for _, r := range m.Registries {
		if r.Server == server {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteRegistryCredential(server string) error {
	var filtered []domain.SavedRegistryCredential
	for _, r := range m.Registries {
		if r.Server != server {
			filtered = append(filtered, r)
		}
	}
	m.Registries = filtered
	return nil
}

func (m *MockConfigRepository) SaveMigrationFile(file domain.MigrationFile) error {
	m.Migrations = append(m.Migrations, file)
	return nil
}

func (m *MockConfigRepository) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error) {
	var res []domain.MigrationFile
	for _, f := range m.Migrations {
		if f.DBName == dbName {
			res = append(res, f)
		}
	}
	return res, nil
}

func (m *MockConfigRepository) DeleteMigrationFile(dbName, filename string) error {
	var filtered []domain.MigrationFile
	for _, f := range m.Migrations {
		if !(f.DBName == dbName && f.Filename == filename) {
			filtered = append(filtered, f)
		}
	}
	m.Migrations = filtered
	return nil
}

func (m *MockConfigRepository) RecordMigrationExecution(dbName, filename, status, logs string) error {
	return nil
}

func (m *MockConfigRepository) SaveBackup(backup domain.SavedBackup) error {
	m.Backups = append(m.Backups, backup)
	return nil
}

func (m *MockConfigRepository) GetBackups() ([]domain.SavedBackup, error) {
	return m.Backups, nil
}

func (m *MockConfigRepository) GetBackupByID(id int) (*domain.SavedBackup, error) {
	for _, b := range m.Backups {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteBackup(id int) error {
	var filtered []domain.SavedBackup
	for _, b := range m.Backups {
		if b.ID != id {
			filtered = append(filtered, b)
		}
	}
	m.Backups = filtered
	return nil
}

func (m *MockConfigRepository) SaveAuditLog(log domain.AuditLog) error {
	m.AuditLogs = append(m.AuditLogs, log)
	return nil
}

func (m *MockConfigRepository) GetAuditLogs(limit int) ([]domain.AuditLog, error) {
	return m.AuditLogs, nil
}

func (m *MockConfigRepository) Close() error {
	return nil
}
