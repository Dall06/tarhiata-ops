package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestSyncClusterStateUseCase(t *testing.T) {
	repo := newMockRepoForBootstrap()
	sshExec := mocks.NewMockSSHExecutor()

	// Guardar datos iniciales
	_ = repo.SaveService(domain.SavedService{Name: "api-test", Port: 3000})
	_ = repo.SaveDatabase(domain.SavedDatabase{Name: "db-test", Engine: "postgres"})
	_ = repo.SaveServiceLink(domain.ServiceLink{SourceSvc: "api-test", TargetSvc: "db-test", EnvVarName: "DATABASE_URL"})

	uc := NewSyncClusterStateUseCase(repo, sshExec)

	// 1. Probar exportación
	if err := uc.ExportStateToRemote(); err != nil {
		t.Fatalf("Error exportando estado: %v", err)
	}

	// Mockear respuesta de lectura cat /opt/tarhiata/state.json
	mockStateJSON := `{
		"services": [{"name": "api-remote", "port": 8080}],
		"databases": [{"name": "db-remote", "engine": "mysql"}],
		"service_links": [{"source_svc": "api-remote", "target_svc": "db-remote", "env_var_name": "DB_URL"}]
	}`
	sshExec.MockResponses["cat /opt/tarhiata/state.json"] = &domain.CommandResult{
		Output:   mockStateJSON,
		ExitCode: 0,
	}

	// 2. Probar importación en nueva PC
	dump, err := uc.ImportStateFromRemote()
	if err != nil {
		t.Fatalf("Error importando estado remoto: %v", err)
	}

	if len(dump.Services) == 0 || dump.Services[0].Name != "api-remote" {
		t.Fatalf("Se esperaba servicio 'api-remote', obtenido: %v", dump.Services)
	}

	// Validar que se guardó en el repositorio local
	svc, err := repo.GetService("api-remote")
	if err != nil || svc == nil {
		t.Fatalf("Servicio 'api-remote' no fue importado en el repositorio local")
	}
}
