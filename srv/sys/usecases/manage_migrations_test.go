package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageDBMigrationsUseCase_SaveAndExecute(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Databases = []domain.SavedDatabase{
		{Name: "postgres-main", Engine: "postgres", Password: "secretpassword"},
	}
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

// TestBuildMigrationCommand_ShellInjectionPrevention valida que un password o
// serviceName con metacaracteres de shell llegue citado al comando docker exec real.
func TestBuildMigrationCommand_ShellInjectionPrevention(t *testing.T) {
	const evilPassword = `pass' ; rm -rf / #`
	const evilService = `svc' ; rm -rf / #`

	tests := []struct {
		name   string
		engine string
		want   string
	}{
		{
			name:   "postgres no usa password pero cita el serviceName",
			engine: "postgres",
			want:   `echo 'Y29udGVudA==' | base64 -d | docker exec -i $(docker ps -q -f name='svc'\'' ; rm -rf / #' | head -n 1) psql -U admin -d db`,
		},
		{
			name:   "mongo cita password y serviceName",
			engine: "mongo",
			want:   `echo 'Y29udGVudA==' | base64 -d | docker exec -i $(docker ps -q -f name='svc'\'' ; rm -rf / #' | head -n 1) mongosh -u admin -p 'pass'\'' ; rm -rf / #' db`,
		},
		{
			name:   "mysql (default) cita password y serviceName",
			engine: "mysql",
			want:   `echo 'Y29udGVudA==' | base64 -d | docker exec -i $(docker ps -q -f name='svc'\'' ; rm -rf / #' | head -n 1) mysql -u admin -p'pass'\'' ; rm -rf / #' db`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildMigrationCommand(tc.engine, evilPassword, evilService, "Y29udGVudA==")
			if got != tc.want {
				t.Errorf("comando inseguro:\n got:  %s\n want: %s", got, tc.want)
			}
		})
	}
}
