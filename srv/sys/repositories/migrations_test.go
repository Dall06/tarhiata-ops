package repositories

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// openLegacyDB crea una SQLite con el esquema de ANTES de migration-001 (sin
// server_name, UNIQUE(name) simple), para poder probar el runner de migraciones
// sobre datos preexistentes tal como estarían en una instancia real.
func openLegacyDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("abrir sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ddl := []string{
		`CREATE TABLE server_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			user TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE services (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			image_source TEXT NOT NULL,
			is_url BOOLEAN NOT NULL,
			port INTEGER NOT NULL,
			domain TEXT NOT NULL,
			expose BOOLEAN NOT NULL,
			env_file_path TEXT NOT NULL,
			enable_ssl BOOLEAN NOT NULL DEFAULT 0,
			healthcheck_cmd TEXT NOT NULL DEFAULT '',
			mounts_json TEXT NOT NULL DEFAULT '[]',
			env_vars TEXT NOT NULL DEFAULT '',
			target_node TEXT NOT NULL DEFAULT '',
			pre_deploy_hook TEXT NOT NULL DEFAULT '',
			custom_domains TEXT NOT NULL DEFAULT '',
			webhook_secret TEXT NOT NULL DEFAULT '',
			source_type TEXT NOT NULL DEFAULT '',
			git_repo_url TEXT NOT NULL DEFAULT '',
			git_branch TEXT NOT NULL DEFAULT '',
			git_access_token TEXT NOT NULL DEFAULT '',
			dockerfile_path TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE databases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			engine TEXT NOT NULL,
			deploy_type TEXT NOT NULL,
			external_url TEXT NOT NULL,
			internal_port INTEGER NOT NULL,
			volume_host_path TEXT NOT NULL,
			node_ip TEXT NOT NULL,
			password TEXT NOT NULL DEFAULT '',
			target_node TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE service_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_svc TEXT NOT NULL,
			target_svc TEXT NOT NULL,
			env_var_name TEXT NOT NULL,
			target_url TEXT NOT NULL,
			UNIQUE(source_svc, env_var_name)
		);`,
		`CREATE TABLE migration_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			db_name TEXT NOT NULL,
			filename TEXT NOT NULL,
			content TEXT NOT NULL,
			down_content TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			executed_at TEXT NOT NULL DEFAULT '',
			log_output TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(db_name, filename)
		);`,
		`CREATE TABLE backups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_name TEXT NOT NULL,
			target_type TEXT NOT NULL,
			engine TEXT NOT NULL,
			filename TEXT NOT NULL,
			file_path TEXT NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'completed',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("crear esquema legacy: %v", err)
		}
	}
	return db
}

// TestSplitSQLStatements_IgnoresSemicolonInsideComments regresión: un ';' dentro de un
// comentario de línea ('-- ...') no debe cortar el statement a la mitad. Se reprodujo
// en la práctica al agregar un comentario con ';' en migration-001.
func TestSplitSQLStatements_IgnoresSemicolonInsideComments(t *testing.T) {
	content := `-- Un comentario con un punto y coma; que no debería cortar nada
CREATE TABLE t (id INTEGER);

-- Otro comentario; con otro punto y coma
INSERT INTO t (id) VALUES (1);`

	stmts := splitSQLStatements(content)
	if len(stmts) != 2 {
		t.Fatalf("esperaba 2 statements, obtuve %d: %#v", len(stmts), stmts)
	}
	if stmts[0] != "CREATE TABLE t (id INTEGER)" {
		t.Errorf("statement 0 inesperado: %q", stmts[0])
	}
	if stmts[1] != "INSERT INTO t (id) VALUES (1)" {
		t.Errorf("statement 1 inesperado: %q", stmts[1])
	}
}

func TestApplyPendingMigrationsBackfillsActiveServer(t *testing.T) {
	db := openLegacyDB(t)

	if _, err := db.Exec(`INSERT INTO server_configs (name, host, port, user, is_active) VALUES ('local', '127.0.0.1', 22, 'root', 1)`); err != nil {
		t.Fatalf("seed server_configs: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO services (name, image_source, is_url, port, domain, expose, env_file_path) VALUES ('api', 'node:18', 0, 80, 'api.test', 1, '')`); err != nil {
		t.Fatalf("seed services: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO databases (name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip) VALUES ('pg', 'postgres', 'single-node', '', 5432, '/data', '')`); err != nil {
		t.Fatalf("seed databases: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO service_links (source_svc, target_svc, env_var_name, target_url) VALUES ('api', 'pg', 'DATABASE_URL', 'postgres://pg:5432')`); err != nil {
		t.Fatalf("seed service_links: %v", err)
	}

	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("ApplyPendingMigrations: %v", err)
	}

	var svcServer, dbServer, linkServer string
	if err := db.QueryRow(`SELECT server_name FROM services WHERE name = 'api'`).Scan(&svcServer); err != nil {
		t.Fatalf("leer server_name de services: %v", err)
	}
	if err := db.QueryRow(`SELECT server_name FROM databases WHERE name = 'pg'`).Scan(&dbServer); err != nil {
		t.Fatalf("leer server_name de databases: %v", err)
	}
	if err := db.QueryRow(`SELECT server_name FROM service_links WHERE source_svc = 'api'`).Scan(&linkServer); err != nil {
		t.Fatalf("leer server_name de service_links: %v", err)
	}

	if svcServer != "local" || dbServer != "local" || linkServer != "local" {
		t.Fatalf("backfill esperado a 'local', obtuve services=%q databases=%q service_links=%q", svcServer, dbServer, linkServer)
	}
}

// TestApplyPendingMigrationsBackfillsFirstServerWhenNoneActive regresión: si al migrar
// no hay ningún server_configs con is_active=1 (ej. el admin borró el activo antes de
// actualizar el binario), el backfill no debe dejar las filas con server_name=''
// huérfanas para siempre -- debe caer al primer servidor existente, igual que
// GetAllServerConfigs ordena (is_active DESC, id ASC).
func TestApplyPendingMigrationsBackfillsFirstServerWhenNoneActive(t *testing.T) {
	db := openLegacyDB(t)

	if _, err := db.Exec(`INSERT INTO server_configs (name, host, port, user, is_active) VALUES ('vps-prod', '1.2.3.4', 22, 'root', 0)`); err != nil {
		t.Fatalf("seed server_configs sin activo: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO services (name, image_source, is_url, port, domain, expose, env_file_path) VALUES ('api', 'node:18', 0, 80, 'api.test', 1, '')`); err != nil {
		t.Fatalf("seed services: %v", err)
	}

	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("ApplyPendingMigrations: %v", err)
	}

	var svcServer string
	if err := db.QueryRow(`SELECT server_name FROM services WHERE name = 'api'`).Scan(&svcServer); err != nil {
		t.Fatalf("leer server_name de services: %v", err)
	}
	if svcServer != "vps-prod" {
		t.Fatalf("esperaba backfill a 'vps-prod' (único servidor existente) sin servidor activo, obtuve %q", svcServer)
	}
}

// TestApplyPendingMigrationsBackfillsMigrationFilesAndBackups regresión: migration_files
// y backups también ganan server_name (migration-002). migration_files hereda el
// server_name de la BD que referencia por db_name; un archivo de migración huérfano
// (sin BD coincidente en el catálogo) cae al mismo fallback que migration-001 (servidor
// activo, o el primero si no hay ninguno activo).
func TestApplyPendingMigrationsBackfillsMigrationFilesAndBackups(t *testing.T) {
	db := openLegacyDB(t)

	if _, err := db.Exec(`INSERT INTO server_configs (name, host, port, user, is_active) VALUES ('local', '127.0.0.1', 22, 'root', 1)`); err != nil {
		t.Fatalf("seed server_configs: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO databases (name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip) VALUES ('pg', 'postgres', 'single-node', '', 5432, '/data', '')`); err != nil {
		t.Fatalf("seed databases: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO migration_files (db_name, filename, content) VALUES ('pg', '001_init.sql', 'CREATE TABLE t (id int);')`); err != nil {
		t.Fatalf("seed migration_files (ligado a una BD real): %v", err)
	}
	if _, err := db.Exec(`INSERT INTO migration_files (db_name, filename, content) VALUES ('ya-borrada', '001_init.sql', 'CREATE TABLE t (id int);')`); err != nil {
		t.Fatalf("seed migration_files huérfano (sin BD coincidente): %v", err)
	}
	if _, err := db.Exec(`INSERT INTO backups (target_name, target_type, engine, filename, file_path) VALUES ('pg', 'database', 'postgres', 'b.sql.gz', '/opt/tarhiata/backups/b.sql.gz')`); err != nil {
		t.Fatalf("seed backups: %v", err)
	}

	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("ApplyPendingMigrations: %v", err)
	}

	var linkedServer, orphanServer, backupServer string
	if err := db.QueryRow(`SELECT server_name FROM migration_files WHERE db_name = 'pg'`).Scan(&linkedServer); err != nil {
		t.Fatalf("leer server_name de migration_files ligado: %v", err)
	}
	if linkedServer != "local" {
		t.Errorf("esperaba que el archivo de migración ligado a 'pg' herede server_name='local' de la BD, obtuve %q", linkedServer)
	}

	if err := db.QueryRow(`SELECT server_name FROM migration_files WHERE db_name = 'ya-borrada'`).Scan(&orphanServer); err != nil {
		t.Fatalf("leer server_name de migration_files huérfano: %v", err)
	}
	if orphanServer != "local" {
		t.Errorf("esperaba que el archivo huérfano caiga al fallback 'local' (servidor activo), obtuve %q", orphanServer)
	}

	if err := db.QueryRow(`SELECT server_name FROM backups WHERE target_name = 'pg'`).Scan(&backupServer); err != nil {
		t.Fatalf("leer server_name de backups: %v", err)
	}
	if backupServer != "local" {
		t.Errorf("esperaba backup con server_name='local', obtuve %q", backupServer)
	}
}

func TestApplyPendingMigrationsIsIdempotent(t *testing.T) {
	db := openLegacyDB(t)

	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("primera corrida: %v", err)
	}
	appliedFirst, err := AppliedMigrations(db)
	if err != nil {
		t.Fatalf("AppliedMigrations: %v", err)
	}
	if len(appliedFirst) == 0 {
		t.Fatalf("se esperaba al menos una migración aplicada")
	}

	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("segunda corrida: %v", err)
	}
	appliedSecond, err := AppliedMigrations(db)
	if err != nil {
		t.Fatalf("AppliedMigrations tras segunda corrida: %v", err)
	}

	if len(appliedSecond) != len(appliedFirst) {
		t.Fatalf("la segunda corrida reaplicó migraciones: antes=%v después=%v", appliedFirst, appliedSecond)
	}

	pending, err := PendingMigrations(db)
	if err != nil {
		t.Fatalf("PendingMigrations: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("no deberían quedar migraciones pendientes, quedaron: %v", pending)
	}
}

func TestApplyPendingMigrationsEnforcesCompositeUnique(t *testing.T) {
	db := openLegacyDB(t)
	if err := ApplyPendingMigrations(db); err != nil {
		t.Fatalf("ApplyPendingMigrations: %v", err)
	}

	insert := `INSERT INTO services (name, image_source, is_url, port, domain, expose, env_file_path, server_name) VALUES (?, 'img', 0, 80, 'd', 1, '', ?)`

	if _, err := db.Exec(insert, "api", "local"); err != nil {
		t.Fatalf("insertar api en local: %v", err)
	}
	if _, err := db.Exec(insert, "api", "vps-prod"); err != nil {
		t.Fatalf("mismo name en servidor distinto debería permitirse: %v", err)
	}
	if _, err := db.Exec(insert, "api", "local"); err == nil {
		t.Fatalf("mismo name + mismo server_name debería fallar por UNIQUE(name, server_name)")
	}
}
