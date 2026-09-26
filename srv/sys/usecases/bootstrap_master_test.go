package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestBootstrapMasterService_Execute(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Config = &domain.ServerConfig{Host: "127.0.0.1"}
	sshExec := mocks.NewMockSSHExecutor()

	// 1. Configurar un enlace previo para api-shop -> old-db
	if err := repo.SaveServiceLink(domain.ServiceLink{
		SourceSvc:  "api-shop",
		TargetSvc:  "old-db",
		EnvVarName: "DATABASE_URL",
	}); err != nil {
		t.Fatalf("unexpected error saving service link: %v", err)
	}

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
