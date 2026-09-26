package repositories

import (
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestSQLiteServiceCatalog(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Fallo al inicializar base de datos: %v", err)
	}
	defer repo.Close()

	tests := []struct {
		name    string
		service domain.SavedService
	}{
		{
			name: "Guardar servicio sin SSL",
			service: domain.SavedService{
				Name:        "api",
				ImageSource: "node:18-alpine",
				IsURL:       false,
				Port:        80,
				Domain:      "api.test",
				Expose:      true,
				EnvFilePath: "/tmp/.env",
				EnableSSL:   false,
			},
		},
		{
			name: "Guardar servicio con SSL",
			service: domain.SavedService{
				Name:        "web",
				ImageSource: "react",
				IsURL:       false,
				Port:        3000,
				Domain:      "web.test",
				Expose:      true,
				EnvFilePath: "",
				EnableSSL:   true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.SaveService(tc.service)
			if err != nil {
				t.Fatalf("Error guardando servicio: %v", err)
			}

			saved, err := repo.GetService(tc.service.Name)
			if err != nil || saved == nil {
				t.Fatalf("Error leyendo servicio: %v", err)
			}

			if saved.Name != tc.service.Name || saved.EnableSSL != tc.service.EnableSSL {
				t.Errorf("Los datos recuperados no coinciden. Esperado: %+v, Obtenido: %+v", tc.service, saved)
			}
		})
	}

	// Test Delete (fuera del struct para probar el estado secuencial)
	t.Run("Eliminar servicio", func(t *testing.T) {
		err := repo.DeleteService("api")
		if err != nil {
			t.Fatalf("Error eliminando servicio: %v", err)
		}

		saved, _ := repo.GetService("api")
		if saved != nil {
			t.Errorf("El servicio api no se eliminó correctamente")
		}
	})
}

func TestSQLiteDatabaseCatalog(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Fallo al inicializar base de datos: %v", err)
	}
	defer repo.Close()

	tests := []struct {
		name string
		db   domain.SavedDatabase
	}{
		{
			name: "Guardar DB Externa",
			db: domain.SavedDatabase{
				Name:        "mi-postgres-ext",
				Engine:      "postgres",
				DeployType:  "external",
				ExternalURL: "postgres://user:pass@host:5432/db",
			},
		},
		{
			name: "Guardar DB Single Node",
			db: domain.SavedDatabase{
				Name:           "mi-mongo-local",
				Engine:         "mongo",
				DeployType:     "single-node",
				InternalPort:   27017,
				VolumeHostPath: "/opt/tarhiata/data/mongo",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.SaveDatabase(tc.db)
			if err != nil {
				t.Fatalf("Error guardando BD: %v", err)
			}

			saved, err := repo.GetDatabase(tc.db.Name)
			if err != nil || saved == nil {
				t.Fatalf("Error leyendo BD: %v", err)
			}

			if saved.Name != tc.db.Name || saved.DeployType != tc.db.DeployType {
				t.Errorf("Los datos recuperados no coinciden. Esperado: %+v, Obtenido: %+v", tc.db, saved)
			}
		})
	}

	t.Run("Eliminar BD", func(t *testing.T) {
		err := repo.DeleteDatabase("mi-postgres-ext")
		if err != nil {
			t.Fatalf("Error eliminando BD: %v", err)
		}

		saved, _ := repo.GetDatabase("mi-postgres-ext")
		if saved != nil {
			t.Errorf("La BD no se eliminó correctamente")
		}
	})
}

func TestSQLiteMultiServerCatalog(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "multi_server.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("Fallo al inicializar base de datos: %v", err)
	}
	defer repo.Close()

	// Initial get when empty
	initial, err := repo.GetServerConfig()
	if err != nil {
		t.Fatalf("unexpected error on empty repo: %v", err)
	}
	if initial != nil {
		t.Errorf("expected nil initial config, got %+v", initial)
	}

	servers := []domain.ServerConfig{
		{
			Name:          "local",
			Host:          "localhost",
			Port:          0,
			User:          "local",
			CloudProvider: "local",
			IsActive:      true,
		},
		{
			Name:          "vps-prod",
			Host:          "108.61.33.61",
			Port:          22,
			User:          "root",
			PrivateKey:    "/root/.ssh/id_rsa",
			CloudProvider: "vps-direct",
			IsActive:      false,
		},
		{
			Name:          "vps-staging",
			Host:          "157.230.12.8",
			Port:          22,
			User:          "root",
			PrivateKey:    "/root/.ssh/id_rsa",
			CloudProvider: "vps-direct",
			IsActive:      false,
		},
	}

	for _, s := range servers {
		if err := repo.SaveServerConfig(s); err != nil {
			t.Fatalf("error saving server %s: %v", s.Name, err)
		}
	}

	// Verify all saved
	all, err := repo.GetAllServerConfigs()
	if err != nil {
		t.Fatalf("error getting all servers: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(all))
	}

	// Verify active server is "local"
	active, err := repo.GetServerConfig()
	if err != nil {
		t.Fatalf("error getting active server: %v", err)
	}
	if active == nil || active.Name != "local" {
		t.Fatalf("expected active server 'local', got %+v", active)
	}

	// Switch active server to "vps-prod"
	if err := repo.SetActiveServerConfig("vps-prod"); err != nil {
		t.Fatalf("error setting active server: %v", err)
	}

	active, err = repo.GetServerConfig()
	if err != nil || active == nil || active.Name != "vps-prod" {
		t.Fatalf("expected active server 'vps-prod', got %+v", active)
	}

	// Delete server
	if err := repo.DeleteServerConfig("vps-staging"); err != nil {
		t.Fatalf("error deleting server: %v", err)
	}

	all, err = repo.GetAllServerConfigs()
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 servers after deletion, got %d", len(all))
	}
}
