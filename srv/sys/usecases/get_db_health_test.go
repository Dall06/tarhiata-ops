package usecases

import (
	"errors"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestGetDBHealthUseCase_Execute(t *testing.T) {
	t.Run("conexión SSH fallida propaga error", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.ConnectError = errors.New("no se pudo conectar")
		uc := NewGetDBHealthUseCase(repo, mockSSH)

		if _, err := uc.Execute("my-db", domain.ServerConfig{Host: "1.2.3.4"}); err == nil {
			t.Fatal("se esperaba error de conexión SSH")
		}
	})

	t.Run("motor y conexiones activas de postgres se leen del query real", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		repo.Databases = []domain.SavedDatabase{{Name: "shop-db", Engine: "postgres"}}
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["pg_stat_activity"] = &domain.CommandResult{Output: "5", ExitCode: 0}
		uc := NewGetDBHealthUseCase(repo, mockSSH)

		health, err := uc.Execute("tarhiata-db-shop-db", domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if health.Engine != "postgres" || health.ActiveConnections != 5 {
			t.Errorf("unexpected health: %+v", health)
		}
	})

	t.Run("base de datos no encontrada usa valores por defecto sin fallar", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewGetDBHealthUseCase(repo, mockSSH)

		health, err := uc.Execute("no-existe", domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if health.Engine != "postgres" {
			t.Errorf("se esperaba engine por defecto 'postgres', got %q", health.Engine)
		}
	})

	// TestGetDBHealthUseCase_ShellInjectionPrevention: la contraseña de MySQL debe llegar
	// citada al comando "docker exec ... mysql -p...", cerrando la inyección que quedaba
	// al mover este código (usaba %q de Go, que no es shell-safe: no escapa $() ni backticks).
	t.Run("password de mysql con metacaracteres llega citada al comando real", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		repo.Databases = []domain.SavedDatabase{{
			Name:     "shop-db",
			Engine:   "mysql",
			Password: `p' ; rm -rf / #`,
		}}
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewGetDBHealthUseCase(repo, mockSSH)

		if _, err := uc.Execute("shop-db", domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var got string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.Contains(c, "mysql -u") {
				got = c
			}
		}
		want := `docker exec $(docker ps -q -f name=tarhiata-db-shop-db | head -n 1) mysql -u 'admin' -p'p'\'' ; rm -rf / #' -e "SHOW STATUS LIKE 'Threads_connected';" 2>/dev/null | tail -n 1 | awk '{print $2}'`
		if got != want {
			t.Errorf("comando inseguro:\n got:  %s\n want: %s", got, want)
		}
	})
}
