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
	AlertSettings *domain.AlertSettings
	Deployments   []domain.DeploymentRecord
}

func NewMockConfigRepository() *MockConfigRepository {
	return &MockConfigRepository{
		Services:      []domain.SavedService{},
		Databases:     []domain.SavedDatabase{},
		Links:         []domain.ServiceLink{},
		Previews:      []domain.SavedPreviewEnv{},
		Registries:    []domain.SavedRegistryCredential{},
		Migrations:    []domain.MigrationFile{},
		Backups:       []domain.SavedBackup{},
		AuditLogs:     []domain.AuditLog{},
		AlertSettings: &domain.AlertSettings{Enabled: false},
		Deployments:   []domain.DeploymentRecord{},
	}
}

func (m *MockConfigRepository) SaveServerConfig(config domain.ServerConfig) error {
	m.Config = &config
	return nil
}

func (m *MockConfigRepository) GetServerConfig() (*domain.ServerConfig, error) {
	return m.Config, nil
}

func (m *MockConfigRepository) GetAllServerConfigs() ([]domain.ServerConfig, error) {
	if m.Config != nil {
		return []domain.ServerConfig{*m.Config}, nil
	}
	return []domain.ServerConfig{}, nil
}

func (m *MockConfigRepository) GetServerConfigByName(name string) (*domain.ServerConfig, error) {
	if m.Config != nil && m.Config.Name == name {
		return m.Config, nil
	}
	return nil, nil
}

func (m *MockConfigRepository) SetActiveServerConfig(name string) error {
	if m.Config != nil && m.Config.Name == name {
		m.Config.IsActive = true
	}
	return nil
}

func (m *MockConfigRepository) DeleteServerConfig(name string) error {
	if m.Config != nil && m.Config.Name == name {
		m.Config = nil
	}
	return nil
}

// SaveService simula el upsert-por-nombre del repositorio real (ON CONFLICT DO UPDATE):
// reemplaza el servicio existente con ese nombre en vez de duplicarlo con append.
func (m *MockConfigRepository) SaveService(svc domain.SavedService) error {
	for i, s := range m.Services {
		if s.Name == svc.Name && s.ServerName == svc.ServerName {
			m.Services[i] = svc
			return nil
		}
	}
	m.Services = append(m.Services, svc)
	return nil
}

func (m *MockConfigRepository) GetServices(serverName string) ([]domain.SavedService, error) {
	var out []domain.SavedService
	for _, s := range m.Services {
		if s.ServerName == serverName {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *MockConfigRepository) GetService(name, serverName string) (*domain.SavedService, error) {
	for _, s := range m.Services {
		if s.Name == name && s.ServerName == serverName {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteService(name, serverName string) error {
	var filtered []domain.SavedService
	for _, s := range m.Services {
		if !(s.Name == name && s.ServerName == serverName) {
			filtered = append(filtered, s)
		}
	}
	m.Services = filtered
	return nil
}

func (m *MockConfigRepository) SaveDatabase(db domain.SavedDatabase) error {
	for i, d := range m.Databases {
		if d.Name == db.Name && d.ServerName == db.ServerName {
			m.Databases[i] = db
			return nil
		}
	}
	m.Databases = append(m.Databases, db)
	return nil
}

func (m *MockConfigRepository) GetDatabases(serverName string) ([]domain.SavedDatabase, error) {
	var out []domain.SavedDatabase
	for _, d := range m.Databases {
		if d.ServerName == serverName {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *MockConfigRepository) GetDatabase(name, serverName string) (*domain.SavedDatabase, error) {
	for _, d := range m.Databases {
		if d.Name == name && d.ServerName == serverName {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) DeleteDatabase(name, serverName string) error {
	var filtered []domain.SavedDatabase
	for _, d := range m.Databases {
		if !(d.Name == name && d.ServerName == serverName) {
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

func (m *MockConfigRepository) GetServiceLinks(serverName string) ([]domain.ServiceLink, error) {
	var out []domain.ServiceLink
	for _, l := range m.Links {
		if l.ServerName == serverName {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *MockConfigRepository) DeleteServiceLink(sourceSvc, targetSvc, serverName string) error {
	var filtered []domain.ServiceLink
	for _, l := range m.Links {
		if !(l.SourceSvc == sourceSvc && l.TargetSvc == targetSvc && l.ServerName == serverName) {
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
	for i := range m.Migrations {
		if m.Migrations[i].DBName == dbName && m.Migrations[i].Filename == filename {
			m.Migrations[i].Status = status
			m.Migrations[i].LogOutput = logs
		}
	}
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

func (m *MockConfigRepository) SaveAlertSettings(settings domain.AlertSettings) error {
	m.AlertSettings = &settings
	return nil
}

func (m *MockConfigRepository) GetAlertSettings() (*domain.AlertSettings, error) {
	if m.AlertSettings == nil {
		return &domain.AlertSettings{Enabled: false}, nil
	}
	return m.AlertSettings, nil
}

func (m *MockConfigRepository) SaveDeploymentRecord(record domain.DeploymentRecord) error {
	record.ID = len(m.Deployments) + 1
	m.Deployments = append(m.Deployments, record)
	return nil
}

func (m *MockConfigRepository) GetDeploymentHistory(serviceName string, limit int) ([]domain.DeploymentRecord, error) {
	var filtered []domain.DeploymentRecord
	for i := len(m.Deployments) - 1; i >= 0; i-- {
		if m.Deployments[i].ServiceName == serviceName {
			filtered = append(filtered, m.Deployments[i])
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}
	return filtered, nil
}

func (m *MockConfigRepository) GetDeploymentRecordByID(id int) (*domain.DeploymentRecord, error) {
	for _, d := range m.Deployments {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *MockConfigRepository) Close() error {
	return nil
}
