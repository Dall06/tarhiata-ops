package repositories

import (
	"path/filepath"
	"strings"
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

		saved, errGet := repo.GetService("api")
		if errGet != nil {
			t.Fatalf("Error consultando servicio eliminado: %v", errGet)
		}
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

		saved, errGet := repo.GetDatabase("mi-postgres-ext")
		if errGet != nil {
			t.Fatalf("Error consultando BD eliminada: %v", errGet)
		}
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

func TestSQLiteEncryptionAtRest(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "crypto_test.db")

	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("error initializing db: %v", err)
	}
	defer repo.Close()

	// 1. Test ServerConfig encryption at rest
	rawKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret_key_data\n-----END OPENSSH PRIVATE KEY-----"
	rawVultrToken := "vultr_sec_tok_123456789"
	rawDOToken := "do_sec_tok_987654321"

	srv := domain.ServerConfig{
		Name:          "prod-vps",
		Host:          "192.168.1.50",
		Port:          22,
		User:          "root",
		PrivateKey:    rawKey,
		VultrAPIToken: rawVultrToken,
		DOAPIToken:    rawDOToken,
	}

	if err := repo.SaveServerConfig(srv); err != nil {
		t.Fatalf("error saving server config: %v", err)
	}

	// Verify raw SQLite storage is encrypted
	var dbKey, dbVultr, dbDO string
	query := "SELECT private_key, vultr_api_token, do_api_token FROM server_configs WHERE name = 'prod-vps'"
	if err := repo.db.QueryRow(query).Scan(&dbKey, &dbVultr, &dbDO); err != nil {
		t.Fatalf("error querying raw database: %v", err)
	}

	if dbKey == rawKey || !strings.HasPrefix(dbKey, "enc:v1:") {
		t.Errorf("expected private_key to be encrypted in SQLite, got: %s", dbKey)
	}
	if dbVultr == rawVultrToken || !strings.HasPrefix(dbVultr, "enc:v1:") {
		t.Errorf("expected vultr_api_token to be encrypted in SQLite, got: %s", dbVultr)
	}
	if dbDO == rawDOToken || !strings.HasPrefix(dbDO, "enc:v1:") {
		t.Errorf("expected do_api_token to be encrypted in SQLite, got: %s", dbDO)
	}

	// Verify transparent decryption on read
	retrievedSrv, err := repo.GetServerConfigByName("prod-vps")
	if err != nil || retrievedSrv == nil {
		t.Fatalf("error retrieving server: %v", err)
	}
	if retrievedSrv.PrivateKey != rawKey {
		t.Errorf("expected decrypted private key %q, got %q", rawKey, retrievedSrv.PrivateKey)
	}
	if retrievedSrv.VultrAPIToken != rawVultrToken {
		t.Errorf("expected decrypted vultr token %q, got %q", rawVultrToken, retrievedSrv.VultrAPIToken)
	}
	if retrievedSrv.DOAPIToken != rawDOToken {
		t.Errorf("expected decrypted do token %q, got %q", rawDOToken, retrievedSrv.DOAPIToken)
	}

	// 2. Test Database password encryption at rest
	rawDBPassword := "super_secret_db_pass_999!"
	dbModel := domain.SavedDatabase{
		Name:     "postgres-app",
		Engine:   "postgres",
		Password: rawDBPassword,
	}
	if err := repo.SaveDatabase(dbModel); err != nil {
		t.Fatalf("error saving database: %v", err)
	}

	var rawStoredDBPass string
	if err := repo.db.QueryRow("SELECT password FROM databases WHERE name = 'postgres-app'").Scan(&rawStoredDBPass); err != nil {
		t.Fatalf("error querying raw db pass: %v", err)
	}
	if rawStoredDBPass == rawDBPassword || !strings.HasPrefix(rawStoredDBPass, "enc:v1:") {
		t.Errorf("expected database password to be encrypted in SQLite, got: %s", rawStoredDBPass)
	}

	retrievedDB, err := repo.GetDatabase("postgres-app")
	if err != nil || retrievedDB == nil {
		t.Fatalf("error retrieving database: %v", err)
	}
	if retrievedDB.Password != rawDBPassword {
		t.Errorf("expected decrypted password %q, got %q", rawDBPassword, retrievedDB.Password)
	}

	// 3. Test Registry credential encryption at rest
	rawRegPassword := "docker_token_pat_112233"
	reg := domain.SavedRegistryCredential{
		Server:   "ghcr.io",
		Username: "tarhiata",
		Password: rawRegPassword,
	}
	if err := repo.SaveRegistryCredential(reg); err != nil {
		t.Fatalf("error saving registry credential: %v", err)
	}

	var rawStoredRegPass string
	if err := repo.db.QueryRow("SELECT password FROM registry_credentials WHERE server = 'ghcr.io'").Scan(&rawStoredRegPass); err != nil {
		t.Fatalf("error querying raw registry password: %v", err)
	}
	if rawStoredRegPass == rawRegPassword || !strings.HasPrefix(rawStoredRegPass, "enc:v1:") {
		t.Errorf("expected registry password to be encrypted in SQLite, got: %s", rawStoredRegPass)
	}

	retrievedReg, err := repo.GetRegistryCredential("ghcr.io")
	if err != nil || retrievedReg == nil {
		t.Fatalf("error retrieving registry credential: %v", err)
	}
	if retrievedReg.Password != rawRegPassword {
		t.Errorf("expected decrypted registry password %q, got %q", rawRegPassword, retrievedReg.Password)
	}

	// 4. Test backward compatibility: Reading legacy unencrypted records
	legacyKey := "~/.ssh/legacy_id_rsa"
	_, err = repo.db.Exec(`INSERT INTO server_configs (name, host, port, user, private_key, cloud_provider, is_active)
		VALUES ('legacy-server', '10.0.0.1', 22, 'ubuntu', ?, 'vps-direct', 0)`, legacyKey)
	if err != nil {
		t.Fatalf("error inserting legacy unencrypted server: %v", err)
	}

	legacySrv, err := repo.GetServerConfigByName("legacy-server")
	if err != nil || legacySrv == nil {
		t.Fatalf("error retrieving legacy server: %v", err)
	}
	if legacySrv.PrivateKey != legacyKey {
		t.Errorf("expected legacy plaintext key %q to be preserved, got %q", legacyKey, legacySrv.PrivateKey)
	}
}
