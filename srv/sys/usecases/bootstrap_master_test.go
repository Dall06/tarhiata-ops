package usecases

import (
	"fmt"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

type mockRepoForBootstrap struct {
	services map[string]domain.SavedService
	dbs      map[string]domain.SavedDatabase
	links    []domain.ServiceLink
}

func newMockRepoForBootstrap() *mockRepoForBootstrap {
	return &mockRepoForBootstrap{
		services: make(map[string]domain.SavedService),
		dbs:      make(map[string]domain.SavedDatabase),
		links:    []domain.ServiceLink{},
	}
}

func (m *mockRepoForBootstrap) SaveServerConfig(config domain.ServerConfig) error { return nil }
func (m *mockRepoForBootstrap) GetServerConfig() (*domain.ServerConfig, error) {
	return &domain.ServerConfig{Host: "127.0.0.1"}, nil
}
func (m *mockRepoForBootstrap) Close() error { return nil }

func (m *mockRepoForBootstrap) SaveService(service domain.SavedService) error {
	m.services[service.Name] = service
	return nil
}
func (m *mockRepoForBootstrap) GetServices() ([]domain.SavedService, error) {
	var list []domain.SavedService
	for _, s := range m.services {
		list = append(list, s)
	}
	return list, nil
}
func (m *mockRepoForBootstrap) GetService(name string) (*domain.SavedService, error) {
	if s, ok := m.services[name]; ok {
		return &s, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (m *mockRepoForBootstrap) DeleteService(name string) error {
	delete(m.services, name)
	return nil
}
func (m *mockRepoForBootstrap) SaveDatabase(db domain.SavedDatabase) error {
	m.dbs[db.Name] = db
	return nil
}
func (m *mockRepoForBootstrap) GetDatabases() ([]domain.SavedDatabase, error) {
	var list []domain.SavedDatabase
	for _, d := range m.dbs {
		list = append(list, d)
	}
	return list, nil
}
func (m *mockRepoForBootstrap) GetDatabase(name string) (*domain.SavedDatabase, error) {
	if d, ok := m.dbs[name]; ok {
		return &d, nil
	}
	return nil, fmt.Errorf("database not found")
}
func (m *mockRepoForBootstrap) DeleteDatabase(name string) error {
	delete(m.dbs, name)
	return nil
}
func (m *mockRepoForBootstrap) SaveObservability(obs domain.SavedObservability) error { return nil }
func (m *mockRepoForBootstrap) GetObservability() (*domain.SavedObservability, error) { return nil, nil }
func (m *mockRepoForBootstrap) DeleteObservability() error                            { return nil }

func (m *mockRepoForBootstrap) SaveServiceLink(link domain.ServiceLink) error {
	m.links = append(m.links, link)
	return nil
}
func (m *mockRepoForBootstrap) GetServiceLinks() ([]domain.ServiceLink, error) {
	return m.links, nil
}
func (m *mockRepoForBootstrap) DeleteServiceLink(sourceSvc string, targetSvc string) error {
	var newLinks []domain.ServiceLink
	for _, l := range m.links {
		if l.SourceSvc == sourceSvc && l.TargetSvc == targetSvc {
			continue
		}
		newLinks = append(newLinks, l)
	}
	m.links = newLinks
	return nil
}

func (m *mockRepoForBootstrap) SavePreviewEnv(env domain.SavedPreviewEnv) error               { return nil }
func (m *mockRepoForBootstrap) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)             { return nil, nil }
func (m *mockRepoForBootstrap) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error)   { return nil, nil }
func (m *mockRepoForBootstrap) DeletePreviewEnv(name string) error                            { return nil }
func (m *mockRepoForBootstrap) SaveRegistryCredential(cred domain.SavedRegistryCredential) error { return nil }
func (m *mockRepoForBootstrap) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForBootstrap) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForBootstrap) DeleteRegistryCredential(server string) error                  { return nil }
func (m *mockRepoForBootstrap) SaveMigrationFile(file domain.MigrationFile) error              { return nil }
func (m *mockRepoForBootstrap) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error){ return nil, nil }
func (m *mockRepoForBootstrap) DeleteMigrationFile(dbName, filename string) error              { return nil }
func (m *mockRepoForBootstrap) RecordMigrationExecution(dbName, filename, status, logs string) error { return nil }
func (m *mockRepoForBootstrap) SaveBackup(b domain.SavedBackup) error                           { return nil }
func (m *mockRepoForBootstrap) GetBackups() ([]domain.SavedBackup, error)                     { return nil, nil }
func (m *mockRepoForBootstrap) GetBackupByID(id int) (*domain.SavedBackup, error)             { return nil, nil }
func (m *mockRepoForBootstrap) DeleteBackup(id int) error                                      { return nil }
func (m *mockRepoForBootstrap) SaveAuditLog(log domain.AuditLog) error                          { return nil }
func (m *mockRepoForBootstrap) GetAuditLogs(limit int) ([]domain.AuditLog, error)              { return nil, nil }

func TestBootstrapMasterService_Execute(t *testing.T) {
	repo := newMockRepoForBootstrap()
	sshExec := mocks.NewMockSSHExecutor()

	// 1. Configurar un enlace previo para api-shop -> old-db
	_ = repo.SaveServiceLink(domain.ServiceLink{
		SourceSvc:  "api-shop",
		TargetSvc:  "old-db",
		EnvVarName: "DATABASE_URL",
	})

	linkUC := NewLinkServicesUseCase(repo, sshExec)
	unlinkUC := NewUnlinkServicesUseCase(repo, sshExec)
	dbUC := NewDeployDatabaseUseCase(sshExec)
	svcUC := NewDeployServiceUseCase(sshExec)

	bootstrapUC := NewBootstrapMasterServiceUseCase(repo, sshExec, linkUC, unlinkUC, dbUC, svcUC)

	input := ports.BootstrapMasterInput{
		AppName:      "api-shop",
		Image:        "node:18-alpine",
		Port:         8080,
		Domain:       "shop.tarhiata.local",
		ExposePublic: true,
		DBEngine:     "postgres",
		EnvVarName:   "DATABASE_URL",
	}

	config := domain.ServerConfig{Host: "127.0.0.1"}

	res, err := bootstrapUC.Execute(input, config)
	if err != nil {
		t.Fatalf("Se esperaba éxito en BootstrapMasterService, error: %v", err)
	}

	// Verificar desacople automático de la BD vieja
	if len(res.UnlinkedOld) != 1 || res.UnlinkedOld[0] != "old-db" {
		t.Errorf("Se esperaba desvincular 'old-db', se desvinculó: %v", res.UnlinkedOld)
	}

	// Verificar creación de nueva App
	if res.App.Name != "api-shop" {
		t.Errorf("Nombre de app esperado 'api-shop', obtenido '%s'", res.App.Name)
	}

	// Verificar creación de nueva DB
	if res.Database == nil || res.Database.Engine != "postgres" {
		t.Fatalf("Se esperaba creación de base de datos postgres")
	}

	// Verificar auto-link inyectado
	if res.Link == nil || res.Link.TargetSvc != "postgres-api-shop" {
		t.Errorf("Link objetivo esperado 'postgres-api-shop', obtenido '%v'", res.Link)
	}
}
