package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManagePreviewEnv_CreateAndListAndDestroy(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	sshExec := mocks.NewMockSSHExecutor()

	// Guardar una BD para probar link
	if err := repo.SaveDatabase(domain.SavedDatabase{
		Name:         "shop-db",
		Engine:       "postgres",
		InternalPort: 5432,
	}); err != nil {
		t.Fatalf("unexpected error saving database: %v", err)
	}

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

	listAfter, errListAfter := uc.List()
	if errListAfter != nil {
		t.Fatalf("Error listando entornos preview tras destrucción: %v", errListAfter)
	}
	if len(listAfter) != 0 {
		t.Errorf("Se esperaba lista vacía tras destrucción, obtenida: %d", len(listAfter))
	}
}
