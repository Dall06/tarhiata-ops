package repositories

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// TestSQLiteServiceCatalog_GitSourceFields valida el flujo completo de persistencia
	// de los campos nuevos de build-from-source, incluyendo que el PAT quede cifrado en
	// reposo pero se devuelva en claro al leer (igual que WebhookSecret/Password).
	t.Run("Campos de origen git se persisten y el PAT queda cifrado en reposo", func(t *testing.T) {
		svc := domain.SavedService{
			Name:           "git-app",
			SourceType:     "git",
			GitRepoURL:     "https://github.com/org/repo.git",
			GitBranch:      "main",
			GitAccessToken: "ghp_super_secret_token",
			DockerfilePath: "docker/Dockerfile",
		}
		if err := repo.SaveService(svc); err != nil {
			t.Fatalf("error guardando servicio git: %v", err)
		}

		saved, err := repo.GetService("git-app")
		if err != nil || saved == nil {
			t.Fatalf("error leyendo servicio git: %v", err)
		}
		if saved.SourceType != "git" || saved.GitRepoURL != svc.GitRepoURL || saved.GitBranch != "main" || saved.DockerfilePath != "docker/Dockerfile" {
			t.Errorf("campos de origen git no coinciden: %+v", saved)
		}
		if saved.GitAccessToken != "ghp_super_secret_token" {
			t.Errorf("el PAT debió descifrarse correctamente al leer, got %q", saved.GitAccessToken)
		}

		// El valor crudo en la base de datos NO debe ser el PAT en texto plano.
		var rawToken string
		if err := repo.db.QueryRow("SELECT git_access_token FROM services WHERE name = ?", "git-app").Scan(&rawToken); err != nil {
			t.Fatalf("error leyendo valor crudo: %v", err)
		}
		if rawToken == "ghp_super_secret_token" {
			t.Errorf("el PAT se guardó en texto plano en vez de cifrado")
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

func TestSQLiteMigrations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migrations_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	// Initial empty list
	files, err := repo.GetMigrationFiles("db-test")
	if err != nil {
		t.Fatalf("unexpected error getting migration files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 migration files initially, got %d", len(files))
	}

	// Save migration file
	mig := domain.MigrationFile{
		DBName:      "db-test",
		Filename:    "001_init.sql",
		Content:     "CREATE TABLE users (id int);",
		DownContent: "DROP TABLE users;",
		Status:      "pending",
	}

	if err := repo.SaveMigrationFile(mig); err != nil {
		t.Fatalf("failed to save migration file: %v", err)
	}

	// Record execution
	if err := repo.RecordMigrationExecution("db-test", "001_init.sql", "applied", "success"); err != nil {
		t.Fatalf("failed to record execution: %v", err)
	}

	filesAfter, err := repo.GetMigrationFiles("db-test")
	if err != nil {
		t.Fatalf("failed to get migration files: %v", err)
	}
	if len(filesAfter) != 1 || filesAfter[0].Status != "applied" {
		t.Errorf("expected 1 applied migration, got: %+v", filesAfter)
	}

	// Delete
	if err := repo.DeleteMigrationFile("db-test", "001_init.sql"); err != nil {
		t.Fatalf("failed to delete migration file: %v", err)
	}
}

func TestSQLitePreviewEnvs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "preview_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	env := domain.SavedPreviewEnv{
		Name:        "pr-42",
		ImageSource: "shop-api:pr-42",
		Port:        3000,
		Domain:      "pr-42.example.com",
		LinkDBName:  "shop-db",
		CreatedAt:   "2026-09-26 14:00:00",
		Status:      "running",
	}

	if err := repo.SavePreviewEnv(env); err != nil {
		t.Fatalf("failed to save preview env: %v", err)
	}

	envs, err := repo.GetPreviewEnvs()
	if err != nil {
		t.Fatalf("failed to get preview envs: %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "pr-42" {
		t.Errorf("expected 1 preview env 'pr-42', got: %+v", envs)
	}

	// Delete
	if err := repo.DeletePreviewEnv("pr-42"); err != nil {
		t.Fatalf("failed to delete preview env: %v", err)
	}

	envsAfter, err := repo.GetPreviewEnvs()
	if err != nil {
		t.Fatalf("failed to get preview envs after delete: %v", err)
	}
	if len(envsAfter) != 0 {
		t.Errorf("expected 0 preview envs after delete, got %d", len(envsAfter))
	}
}

func TestSQLiteObservability(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "obs_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	obs := domain.SavedObservability{
		DeployType:      "swarm",
		ExternalURL:     "https://logs.example.com",
		GrafanaPassword: "admin_password_123",
	}

	if err := repo.SaveObservability(obs); err != nil {
		t.Fatalf("failed to save observability: %v", err)
	}

	saved, err := repo.GetObservability()
	if err != nil || saved == nil {
		t.Fatalf("failed to get observability: %v", err)
	}
	if saved.DeployType != "swarm" || saved.ExternalURL != "https://logs.example.com" {
		t.Errorf("observability data mismatch: %+v", saved)
	}

	if err := repo.DeleteObservability(); err != nil {
		t.Fatalf("failed to delete observability: %v", err)
	}

	deleted, err := repo.GetObservability()
	if err != nil {
		t.Fatalf("unexpected error after delete: %v", err)
	}
	if deleted != nil {
		t.Errorf("expected nil after delete observability, got: %+v", deleted)
	}
}

func TestSQLiteBackups(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "backups_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	backup := domain.SavedBackup{
		TargetName: "db-production",
		TargetType: "database",
		Engine:     "postgres",
		Filename:   "backup_2026.sql",
		FilePath:   "/opt/tarhiata/backups/backup_2026.sql",
		SizeBytes:  1048576,
		Status:     "completed",
		CreatedAt:  "2026-09-26 15:00:00",
	}

	if err := repo.SaveBackup(backup); err != nil {
		t.Fatalf("failed to save backup: %v", err)
	}

	backups, err := repo.GetBackups()
	if err != nil {
		t.Fatalf("failed to get backups: %v", err)
	}
	if len(backups) != 1 || backups[0].TargetName != "db-production" {
		t.Errorf("expected backup 'db-production', got: %+v", backups)
	}

	if err := repo.DeleteBackup(backups[0].ID); err != nil {
		t.Fatalf("failed to delete backup: %v", err)
	}

	backupsAfter, err := repo.GetBackups()
	if err != nil {
		t.Fatalf("failed to get backups after delete: %v", err)
	}
	if len(backupsAfter) != 0 {
		t.Errorf("expected 0 backups after delete, got %d", len(backupsAfter))
	}
}

func TestSQLiteNotFoundCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "notfound_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	svc, err := repo.GetService("non-existent-svc")
	if err != nil || svc != nil {
		t.Errorf("expected nil service, got: %v (err: %v)", svc, err)
	}

	db, err := repo.GetDatabase("non-existent-db")
	if err != nil || db != nil {
		t.Errorf("expected nil database, got: %v (err: %v)", db, err)
	}

	srv, err := repo.GetServerConfigByName("non-existent-server")
	if err != nil || srv != nil {
		t.Errorf("expected nil server, got: %v (err: %v)", srv, err)
	}

	reg, err := repo.GetRegistryCredential("non-existent-reg")
	if err != nil || reg != nil {
		t.Errorf("expected nil registry, got: %v (err: %v)", reg, err)
	}
}

func TestSQLiteAlertSettings_TableDriven(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "alerts_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	tests := []struct {
		name     string
		settings domain.AlertSettings
	}{
		{
			name: "Save and retrieve enabled Discord and Slack",
			settings: domain.AlertSettings{
				DiscordURL: "https://discord.com/api/webhooks/123/abc",
				SlackURL:   "https://hooks.slack.com/services/T00/B00/X00",
				Enabled:    true,
			},
		},
		{
			name: "Save and retrieve Telegram and Generic Webhook",
			settings: domain.AlertSettings{
				TelegramToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				TelegramChat:  "-1001234567890",
				GenericURL:    "https://api.mycompany.com/tarhiata-alerts",
				Enabled:       false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := repo.SaveAlertSettings(tc.settings); err != nil {
				t.Fatalf("SaveAlertSettings() error = %v", err)
			}
			got, err := repo.GetAlertSettings()
			if err != nil {
				t.Fatalf("GetAlertSettings() error = %v", err)
			}
			if got.DiscordURL != tc.settings.DiscordURL {
				t.Errorf("expected DiscordURL %q, got %q", tc.settings.DiscordURL, got.DiscordURL)
			}
			if got.TelegramToken != tc.settings.TelegramToken {
				t.Errorf("expected TelegramToken %q, got %q", tc.settings.TelegramToken, got.TelegramToken)
			}
			if got.TelegramChat != tc.settings.TelegramChat {
				t.Errorf("expected TelegramChat %q, got %q", tc.settings.TelegramChat, got.TelegramChat)
			}
			if got.SlackURL != tc.settings.SlackURL {
				t.Errorf("expected SlackURL %q, got %q", tc.settings.SlackURL, got.SlackURL)
			}
			if got.GenericURL != tc.settings.GenericURL {
				t.Errorf("expected GenericURL %q, got %q", tc.settings.GenericURL, got.GenericURL)
			}
			if got.Enabled != tc.settings.Enabled {
				t.Errorf("expected Enabled %v, got %v", tc.settings.Enabled, got.Enabled)
			}
		})
	}
}

func TestSQLiteDeploymentHistory_TableDriven(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "deploys_test.db")
	repo, err := NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	records := []domain.DeploymentRecord{
		{
			ServiceName: "app-api",
			ImageTag:    "node:18-alpine",
			EnvVars:     "PORT=3000\nNODE_ENV=production",
			Port:        3000,
			Domain:      "api.example.com",
			Expose:      true,
			DeployedAt:  time.Now().Add(-2 * time.Hour),
			Status:      "success",
		},
		{
			ServiceName: "app-api",
			ImageTag:    "node:20-alpine",
			EnvVars:     "PORT=3000\nNODE_ENV=production\nFEATURE_X=true",
			Port:        3000,
			Domain:      "api.example.com",
			Expose:      true,
			DeployedAt:  time.Now().Add(-1 * time.Hour),
			Status:      "success",
		},
		{
			ServiceName: "app-frontend",
			ImageTag:    "nginx:alpine",
			Port:        80,
			Domain:      "app.example.com",
			Expose:      true,
			DeployedAt:  time.Now(),
			Status:      "success",
		},
	}

	for _, r := range records {
		if err := repo.SaveDeploymentRecord(r); err != nil {
			t.Fatalf("SaveDeploymentRecord() error = %v", err)
		}
	}

	history, err := repo.GetDeploymentHistory("app-api", 10)
	if err != nil {
		t.Fatalf("GetDeploymentHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history records for app-api, got %d", len(history))
	}
	if history[0].ImageTag != "node:20-alpine" {
		t.Errorf("expected latest imageTag node:20-alpine, got %s", history[0].ImageTag)
	}

	single, err := repo.GetDeploymentRecordByID(history[0].ID)
	if err != nil || single == nil {
		t.Fatalf("GetDeploymentRecordByID() error = %v, got %v", err, single)
	}
	if single.ServiceName != "app-api" {
		t.Errorf("expected serviceName app-api, got %s", single.ServiceName)
	}
}

