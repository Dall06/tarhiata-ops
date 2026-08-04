package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

type mockRepoForMigrations struct {
	files []domain.MigrationFile
	dbs   []domain.SavedDatabase
}

func newMockRepoForMigrations() *mockRepoForMigrations {
	return &mockRepoForMigrations{
		files: make([]domain.MigrationFile, 0),
		dbs: []domain.SavedDatabase{
			{Name: "postgres-main", Engine: "postgres", Password: "secretpassword"},
		},
	}
}

func (m *mockRepoForMigrations) SaveServerConfig(config domain.ServerConfig) error { return nil }
func (m *mockRepoForMigrations) GetServerConfig() (*domain.ServerConfig, error)    { return nil, nil }
func (m *mockRepoForMigrations) SaveService(svc domain.SavedService) error          { return nil }
func (m *mockRepoForMigrations) GetServices() ([]domain.SavedService, error)         { return nil, nil }
func (m *mockRepoForMigrations) GetService(name string) (*domain.SavedService, error) { return nil, nil }
func (m *mockRepoForMigrations) DeleteService(name string) error                     { return nil }
func (m *mockRepoForMigrations) SaveDatabase(db domain.SavedDatabase) error         { return nil }
func (m *mockRepoForMigrations) GetDatabases() ([]domain.SavedDatabase, error)        { return m.dbs, nil }
func (m *mockRepoForMigrations) GetDatabase(name string) (*domain.SavedDatabase, error) {
	for _, db := range m.dbs {
		if db.Name == name {
			return &db, nil
		}
	}
	return nil, nil
}
func (m *mockRepoForMigrations) DeleteDatabase(name string) error                    { return nil }
func (m *mockRepoForMigrations) SaveObservability(obs domain.SavedObservability) error { return nil }
func (m *mockRepoForMigrations) GetObservability() (*domain.SavedObservability, error) { return nil, nil }
func (m *mockRepoForMigrations) DeleteObservability() error                          { return nil }
func (m *mockRepoForMigrations) SaveServiceLink(link domain.ServiceLink) error       { return nil }
func (m *mockRepoForMigrations) GetServiceLinks() ([]domain.ServiceLink, error)      { return nil, nil }
func (m *mockRepoForMigrations) DeleteServiceLink(sourceSvc, targetSvc string) error { return nil }
func (m *mockRepoForMigrations) SavePreviewEnv(env domain.SavedPreviewEnv) error      { return nil }
func (m *mockRepoForMigrations) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)   { return nil, nil }
func (m *mockRepoForMigrations) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) { return nil, nil }
func (m *mockRepoForMigrations) DeletePreviewEnv(name string) error                 { return nil }
func (m *mockRepoForMigrations) SaveRegistryCredential(cred domain.SavedRegistryCredential) error { return nil }
func (m *mockRepoForMigrations) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForMigrations) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) { return nil, nil }
func (m *mockRepoForMigrations) DeleteRegistryCredential(server string) error        { return nil }

func (m *mockRepoForMigrations) SaveMigrationFile(file domain.MigrationFile) error {
	m.files = append(m.files, file)
	return nil
}

func (m *mockRepoForMigrations) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error) {
	var res []domain.MigrationFile
	for _, f := range m.files {
		if f.DBName == dbName {
			res = append(res, f)
		}
	}
	return res, nil
}

func (m *mockRepoForMigrations) DeleteMigrationFile(dbName, filename string) error {
	var newFiles []domain.MigrationFile
	for _, f := range m.files {
		if !(f.DBName == dbName && f.Filename == filename) {
			newFiles = append(newFiles, f)
		}
	}
	m.files = newFiles
	return nil
}

func (m *mockRepoForMigrations) RecordMigrationExecution(dbName, filename, status, logs string) error {
	for i := range m.files {
		if m.files[i].DBName == dbName && m.files[i].Filename == filename {
			m.files[i].Status = status
			m.files[i].LogOutput = logs
		}
	}
	return nil
}
func (m *mockRepoForMigrations) SaveBackup(b domain.SavedBackup) error               { return nil }
func (m *mockRepoForMigrations) GetBackups() ([]domain.SavedBackup, error)         { return nil, nil }
func (m *mockRepoForMigrations) GetBackupByID(id int) (*domain.SavedBackup, error) { return nil, nil }
func (m *mockRepoForMigrations) DeleteBackup(id int) error                          { return nil }
func (m *mockRepoForMigrations) SaveAuditLog(log domain.AuditLog) error             { return nil }
func (m *mockRepoForMigrations) GetAuditLogs(limit int) ([]domain.AuditLog, error) { return nil, nil }

func (m *mockRepoForMigrations) Close() error { return nil }

func TestManageDBMigrationsUseCase_SaveAndExecute(t *testing.T) {
	repo := newMockRepoForMigrations()
	sshExec := mocks.NewMockSSHExecutor()

	uc := NewManageDBMigrationsUseCase(repo, sshExec)

	// 1. Guardar dos archivos de migración (con sentencias de regresión)
	err := uc.SaveFile("postgres-main", "01_init.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);", "DROP TABLE users;")
	if err != nil {
		t.Fatalf("error inesperado en SaveFile: %v", err)
	}

	err = uc.SaveFile("postgres-main", "02_add_roles.sql", "ALTER TABLE users ADD COLUMN role TEXT;", "ALTER TABLE users DROP COLUMN role;")
	if err != nil {
		t.Fatalf("error inesperado en SaveFile 2: %v", err)
	}

	// 2. Verificar GetFiles
	files, err := uc.GetFiles("postgres-main")
	if err != nil || len(files) != 2 {
		t.Fatalf("se esperaban 2 archivos de migración, se obtuvieron: %d (err=%v)", len(files), err)
	}

	// 3. Ejecutar la migración seleccionada (UP)
	req := domain.DatabaseMigrationRequest{
		TargetDB:   "postgres-main",
		TargetNode: "manager",
		Action:     "up",
		Filenames:  []string{"01_init.sql"},
	}

	executed, err := uc.Execute(req, domain.ServerConfig{Host: "1.2.3.4"})
	if err != nil {
		t.Fatalf("error ejecutando migración UP: %v", err)
	}

	if len(executed) != 1 || executed[0].Status != "applied" {
		t.Fatalf("se esperaba 1 migración aplicada, resultado: %+v", executed)
	}

	// 4. Ejecutar regresión (DOWN / Rollback)
	reqDown := domain.DatabaseMigrationRequest{
		TargetDB:   "postgres-main",
		TargetNode: "manager",
		Action:     "down",
		Filenames:  []string{"01_init.sql"},
	}

	executedDown, err := uc.Execute(reqDown, domain.ServerConfig{Host: "1.2.3.4"})
	if err != nil {
		t.Fatalf("error ejecutando regresión DOWN: %v", err)
	}

	if len(executedDown) != 1 || executedDown[0].Status != "reverted" {
		t.Fatalf("se esperaba 1 migración revertida (status=reverted), resultado: %+v", executedDown)
	}
}
