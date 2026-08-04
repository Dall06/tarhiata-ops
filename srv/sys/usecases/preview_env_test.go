package usecases

import (
	"fmt"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

type mockRepoForPreview struct {
	previews map[string]domain.SavedPreviewEnv
	dbs      map[string]domain.SavedDatabase
}

func newMockRepoForPreview() *mockRepoForPreview {
	return &mockRepoForPreview{
		previews: make(map[string]domain.SavedPreviewEnv),
		dbs:      make(map[string]domain.SavedDatabase),
	}
}

func (m *mockRepoForPreview) SaveServerConfig(config domain.ServerConfig) error { return nil }
func (m *mockRepoForPreview) GetServerConfig() (*domain.ServerConfig, error) {
	return &domain.ServerConfig{Host: "127.0.0.1"}, nil
}
func (m *mockRepoForPreview) SaveService(svc domain.SavedService) error            { return nil }
func (m *mockRepoForPreview) GetServices() ([]domain.SavedService, error)          { return nil, nil }
func (m *mockRepoForPreview) GetService(name string) (*domain.SavedService, error) { return nil, nil }
func (m *mockRepoForPreview) DeleteService(name string) error                       { return nil }

func (m *mockRepoForPreview) SaveDatabase(db domain.SavedDatabase) error {
	m.dbs[db.Name] = db
	return nil
}
func (m *mockRepoForPreview) GetDatabases() ([]domain.SavedDatabase, error) { return nil, nil }
func (m *mockRepoForPreview) GetDatabase(name string) (*domain.SavedDatabase, error) {
	if db, ok := m.dbs[name]; ok {
		return &db, nil
	}
	return nil, fmt.Errorf("database not found")
}
func (m *mockRepoForPreview) DeleteDatabase(name string) error                            { return nil }
func (m *mockRepoForPreview) SaveObservability(obs domain.SavedObservability) error       { return nil }
func (m *mockRepoForPreview) GetObservability() (*domain.SavedObservability, error)        { return nil, nil }
func (m *mockRepoForPreview) DeleteObservability() error                                  { return nil }
func (m *mockRepoForPreview) SaveServiceLink(link domain.ServiceLink) error               { return nil }
func (m *mockRepoForPreview) GetServiceLinks() ([]domain.ServiceLink, error)              { return nil, nil }
func (m *mockRepoForPreview) DeleteServiceLink(sourceSvc string, targetSvc string) error { return nil }

func (m *mockRepoForPreview) SavePreviewEnv(env domain.SavedPreviewEnv) error {
	m.previews[env.Name] = env
	return nil
}
func (m *mockRepoForPreview) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error) {
	var list []domain.SavedPreviewEnv
	for _, p := range m.previews {
		list = append(list, p)
	}
	return list, nil
}
func (m *mockRepoForPreview) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) {
	if p, ok := m.previews[name]; ok {
		return &p, nil
	}
	return nil, fmt.Errorf("preview env not found")
}
func (m *mockRepoForPreview) DeletePreviewEnv(name string) error {
	delete(m.previews, name)
	return nil
}
func (m *mockRepoForPreview) SaveRegistryCredential(cred domain.SavedRegistryCredential) error { return nil }
func (m *mockRepoForPreview) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForPreview) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForPreview) DeleteRegistryCredential(server string) error                  { return nil }
func (m *mockRepoForPreview) SaveMigrationFile(file domain.MigrationFile) error              { return nil }
func (m *mockRepoForPreview) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error){ return nil, nil }
func (m *mockRepoForPreview) DeleteMigrationFile(dbName, filename string) error              { return nil }
func (m *mockRepoForPreview) RecordMigrationExecution(dbName, filename, status, logs string) error { return nil }
func (m *mockRepoForPreview) SaveBackup(b domain.SavedBackup) error               { return nil }
func (m *mockRepoForPreview) GetBackups() ([]domain.SavedBackup, error)         { return nil, nil }
func (m *mockRepoForPreview) GetBackupByID(id int) (*domain.SavedBackup, error) { return nil, nil }
func (m *mockRepoForPreview) DeleteBackup(id int) error                          { return nil }
func (m *mockRepoForPreview) SaveAuditLog(log domain.AuditLog) error             { return nil }
func (m *mockRepoForPreview) GetAuditLogs(limit int) ([]domain.AuditLog, error) { return nil, nil }
func (m *mockRepoForPreview) Close() error { return nil }

func TestManagePreviewEnv_CreateAndListAndDestroy(t *testing.T) {
	repo := newMockRepoForPreview()
	sshExec := mocks.NewMockSSHExecutor()

	// Guardar una BD para probar link
	_ = repo.SaveDatabase(domain.SavedDatabase{
		Name:         "shop-db",
		Engine:       "postgres",
		InternalPort: 5432,
	})

	uc := NewManagePreviewEnvUseCase(repo, sshExec)
	config := domain.ServerConfig{Host: "127.0.0.1"}

	input := ports.CreatePreviewEnvInput{
		Name:       "feat-checkout",
		Image:      "myrepo/shop-api:pr-88",
		Port:       3000,
		Domain:     "pr-88.shop.local",
		LinkDBName: "shop-db",
	}

	// 1. Test Create
	created, err := uc.Create(input, config)
	if err != nil {
		t.Fatalf("Error inesperado al crear entorno preview: %v", err)
	}

	if created.Name != "feat-checkout" || created.Status != "active" {
		t.Errorf("Entorno preview creado incorrecto: %+v", created)
	}

	// 2. Test List
	list, err := uc.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("Se esperaba 1 entorno preview en la lista, obtenido: %d, err: %v", len(list), err)
	}

	// 3. Test Destroy
	err = uc.Destroy("feat-checkout", config)
	if err != nil {
		t.Fatalf("Error destruyendo entorno preview: %v", err)
	}

	listAfter, _ := uc.List()
	if len(listAfter) != 0 {
		t.Errorf("Se esperaba lista vacía tras destrucción, obtenida: %d", len(listAfter))
	}
}
