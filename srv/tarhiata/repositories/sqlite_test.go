package repositories

import (
	"path/filepath"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
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
				ImageSource: "nginx",
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
