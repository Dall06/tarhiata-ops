package repositories

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/pkg/auditlog"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	_ "modernc.org/sqlite"
)

// SQLiteRepository implementa ConfigRepository usando SQLite local
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository crea e inicializa el archivo SQLite.
// dbPath suele ser algo como "~/.config/tarhiata/config.db"
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	// Asegurar que el directorio de configuración exista
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio db: %w", err)
	}

	dsn := dbPath
	if !strings.Contains(dbPath, "?") {
		dsn = dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falló al abrir sqlite: %w", err)
	}

	// Pool de conexiones apto para WAL mode (permite lecturas concurrentes sin bloquear el motor)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	repo := &SQLiteRepository{db: db}

	// Auto-Migración de las tablas
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("falló migración sqlite: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) migrate() error {
	// Usamos un CHECK (id=1) para garantizar que solo exista un servidor activo
	// (Si luego se quiere multi-server, se quita esa regla).
	query := `
	CREATE TABLE IF NOT EXISTS server_config (
		id INTEGER PRIMARY KEY CHECK (id = 1), 
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		user TEXT NOT NULL,
		private_key TEXT NOT NULL,
		vultr_api_token TEXT NOT NULL DEFAULT ''
	);`
	if _, err := r.db.Exec(query); err != nil {
		return err
	}

	r.addColumnIfMissing("server_config", "vultr_api_token", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("server_config", "do_api_token", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("server_config", "cloud_provider", "TEXT NOT NULL DEFAULT 'vultr'")

	// Tabla del catálogo de servicios
	queryServices := `
	CREATE TABLE IF NOT EXISTS services (
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
		env_vars TEXT NOT NULL DEFAULT ''
	);`
	if _, err := r.db.Exec(queryServices); err != nil {
		return err
	}

	r.addColumnIfMissing("services", "enable_ssl", "BOOLEAN NOT NULL DEFAULT 0")
	r.addColumnIfMissing("services", "healthcheck_cmd", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("services", "mounts_json", "TEXT NOT NULL DEFAULT '[]'")
	r.addColumnIfMissing("services", "env_vars", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("services", "target_node", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("services", "pre_deploy_hook", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("services", "custom_domains", "TEXT NOT NULL DEFAULT ''")

	// Tabla de Bases de Datos
	queryDBs := `
	CREATE TABLE IF NOT EXISTS databases (
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
	);`
	if _, err := r.db.Exec(queryDBs); err != nil {
		return err
	}

	r.addColumnIfMissing("databases", "password", "TEXT NOT NULL DEFAULT ''")
	r.addColumnIfMissing("databases", "target_node", "TEXT NOT NULL DEFAULT ''")

	// Tabla de Observabilidad
	queryObs := `
	CREATE TABLE IF NOT EXISTS observability_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		deploy_type TEXT NOT NULL,
		external_url TEXT NOT NULL,
		node_ip TEXT NOT NULL,
		grafana_password TEXT NOT NULL
	);`
	if _, err := r.db.Exec(queryObs); err != nil {
		return err
	}

	r.addColumnIfMissing("observability_config", "grafana_password", "TEXT DEFAULT ''")

	queryLinks := `
	CREATE TABLE IF NOT EXISTS service_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_svc TEXT NOT NULL,
		target_svc TEXT NOT NULL,
		env_var_name TEXT NOT NULL,
		target_url TEXT NOT NULL,
		UNIQUE(source_svc, env_var_name)
	);`
	if _, err := r.db.Exec(queryLinks); err != nil {
		return err
	}

	queryPreview := `
	CREATE TABLE IF NOT EXISTS preview_envs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		image_source TEXT NOT NULL,
		port INTEGER NOT NULL,
		domain TEXT NOT NULL,
		link_db_name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		target_node TEXT NOT NULL DEFAULT ''
	);`
	if _, err := r.db.Exec(queryPreview); err != nil {
		return err
	}

	r.addColumnIfMissing("preview_envs", "target_node", "TEXT NOT NULL DEFAULT ''")

	queryRegistry := `
	CREATE TABLE IF NOT EXISTS registry_credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server TEXT UNIQUE NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.Exec(queryRegistry); err != nil {
		return err
	}

	queryAuditLogs := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_name TEXT NOT NULL,
		details TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.Exec(queryAuditLogs); err != nil {
		return err
	}

	queryMigrations := `
	CREATE TABLE IF NOT EXISTS migration_files (
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
	);`
	if _, err := r.db.Exec(queryMigrations); err != nil {
		return err
	}

	queryBackups := `
	CREATE TABLE IF NOT EXISTS backups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_name TEXT NOT NULL,
		target_type TEXT NOT NULL,
		engine TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'completed',
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := r.db.Exec(queryBackups); err != nil {
		return err
	}

	r.addColumnIfMissing("migration_files", "down_content", "TEXT NOT NULL DEFAULT ''")

	return nil
}

// addColumnIfMissing verifica con pragma_table_info si una columna existe antes de intentar añadirla.
// Esto evita depender de strings de error frágiles que podrían cambiar entre versiones del driver SQLite.
func (r *SQLiteRepository) addColumnIfMissing(table, column, colDef string) {
	var count int
	_ = r.db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name='%s'", table, column),
	).Scan(&count)
	if count == 0 {
		r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, colDef))
	}
}

func (r *SQLiteRepository) SaveServerConfig(config domain.ServerConfig) error {
	// Usamos UPSERT: Inserta si no existe, si existe lo actualiza.
	query := `
	INSERT INTO server_config (id, host, port, user, private_key, vultr_api_token, do_api_token, cloud_provider) 
	VALUES (1, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		host=excluded.host, 
		port=excluded.port, 
		user=excluded.user, 
		private_key=excluded.private_key,
		vultr_api_token=excluded.vultr_api_token,
		do_api_token=excluded.do_api_token,
		cloud_provider=excluded.cloud_provider;`

	_, err := r.db.Exec(query, config.Host, config.Port, config.User, config.PrivateKey, config.VultrAPIToken, config.DOAPIToken, config.CloudProvider)
	return err
}

func (r *SQLiteRepository) GetServerConfig() (*domain.ServerConfig, error) {
	query := `SELECT host, port, user, private_key, vultr_api_token, do_api_token, cloud_provider FROM server_config WHERE id = 1;`
	row := r.db.QueryRow(query)

	var config domain.ServerConfig
	err := row.Scan(&config.Host, &config.Port, &config.User, &config.PrivateKey, &config.VultrAPIToken, &config.DOAPIToken, &config.CloudProvider)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay error, simplemente no hay configuración guardada aún
		}
		return nil, err
	}
	return &config, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// --- Operaciones del Catálogo de Servicios ---

func (r *SQLiteRepository) SaveService(svc domain.SavedService) error {
	query := `
	INSERT INTO services (name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json, env_vars, target_node, pre_deploy_hook, custom_domains)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		image_source=excluded.image_source,
		is_url=excluded.is_url,
		port=excluded.port,
		domain=excluded.domain,
		expose=excluded.expose,
		env_file_path=excluded.env_file_path,
		enable_ssl=excluded.enable_ssl,
		healthcheck_cmd=excluded.healthcheck_cmd,
		mounts_json=excluded.mounts_json,
		env_vars=excluded.env_vars,
		target_node=excluded.target_node,
		pre_deploy_hook=excluded.pre_deploy_hook,
		custom_domains=excluded.custom_domains;`

	_, err := r.db.Exec(query, svc.Name, svc.ImageSource, svc.IsURL, svc.Port, svc.Domain, svc.Expose, svc.EnvFilePath, svc.EnableSSL, svc.HealthcheckCmd, svc.MountsJSON, svc.EnvVars, svc.TargetNode, svc.PreDeployHook, svc.CustomDomains)
	return err
}

func (r *SQLiteRepository) GetServices() ([]domain.SavedService, error) {
	query := `SELECT id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json, env_vars, target_node, pre_deploy_hook, custom_domains FROM services ORDER BY name ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.SavedService
	for rows.Next() {
		var s domain.SavedService
		if err := rows.Scan(&s.ID, &s.Name, &s.ImageSource, &s.IsURL, &s.Port, &s.Domain, &s.Expose, &s.EnvFilePath, &s.EnableSSL, &s.HealthcheckCmd, &s.MountsJSON, &s.EnvVars, &s.TargetNode, &s.PreDeployHook, &s.CustomDomains); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *SQLiteRepository) GetService(name string) (*domain.SavedService, error) {
	query := `SELECT id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json, env_vars, target_node, pre_deploy_hook, custom_domains FROM services WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	var s domain.SavedService
	err := row.Scan(&s.ID, &s.Name, &s.ImageSource, &s.IsURL, &s.Port, &s.Domain, &s.Expose, &s.EnvFilePath, &s.EnableSSL, &s.HealthcheckCmd, &s.MountsJSON, &s.EnvVars, &s.TargetNode, &s.PreDeployHook, &s.CustomDomains)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No encontrado
		}
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteRepository) DeleteService(name string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM service_links WHERE source_svc = ? OR target_svc = ?", name, name); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM services WHERE name = ?", name); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Operaciones del Catálogo de Bases de Datos ---

func (r *SQLiteRepository) SaveDatabase(db domain.SavedDatabase) error {
	query := `
	INSERT INTO databases (name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password, target_node)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		engine=excluded.engine,
		deploy_type=excluded.deploy_type,
		external_url=excluded.external_url,
		internal_port=excluded.internal_port,
		volume_host_path=excluded.volume_host_path,
		node_ip=excluded.node_ip,
		password=excluded.password,
		target_node=excluded.target_node;`

	_, err := r.db.Exec(query, db.Name, db.Engine, db.DeployType, db.ExternalURL, db.InternalPort, db.VolumeHostPath, db.NodeIP, db.Password, db.TargetNode)
	return err
}

func (r *SQLiteRepository) GetDatabases() ([]domain.SavedDatabase, error) {
	query := `SELECT id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password, target_node FROM databases ORDER BY name ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []domain.SavedDatabase
	for rows.Next() {
		var d domain.SavedDatabase
		if err := rows.Scan(&d.ID, &d.Name, &d.Engine, &d.DeployType, &d.ExternalURL, &d.InternalPort, &d.VolumeHostPath, &d.NodeIP, &d.Password, &d.TargetNode); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

func (r *SQLiteRepository) GetDatabase(name string) (*domain.SavedDatabase, error) {
	query := `SELECT id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password, target_node FROM databases WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	var d domain.SavedDatabase
	err := row.Scan(&d.ID, &d.Name, &d.Engine, &d.DeployType, &d.ExternalURL, &d.InternalPort, &d.VolumeHostPath, &d.NodeIP, &d.Password, &d.TargetNode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No encontrado
		}
		return nil, err
	}
	return &d, nil
}

func (r *SQLiteRepository) DeleteDatabase(name string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cleanName := strings.TrimPrefix(name, "tarhiata-db-")
	cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

	if _, err := tx.Exec("DELETE FROM service_links WHERE source_svc = ? OR target_svc = ? OR source_svc = ? OR target_svc = ?", name, name, cleanName, cleanName); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM databases WHERE name = ? OR name = ? OR name = ?", name, cleanName, "tarhiata-db-"+cleanName); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Operaciones de Observabilidad ---

func (r *SQLiteRepository) SaveObservability(obs domain.SavedObservability) error {
	query := `
	INSERT INTO observability_config (id, deploy_type, external_url, node_ip, grafana_password)
	VALUES (1, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		deploy_type=excluded.deploy_type, 
		external_url=excluded.external_url, 
		node_ip=excluded.node_ip,
		grafana_password=excluded.grafana_password;`

	_, err := r.db.Exec(query, obs.DeployType, obs.ExternalURL, obs.NodeIP, obs.GrafanaPassword)
	return err
}

func (r *SQLiteRepository) GetObservability() (*domain.SavedObservability, error) {
	query := `SELECT id, deploy_type, external_url, node_ip, grafana_password FROM observability_config WHERE id = 1`
	var config domain.SavedObservability
	err := r.db.QueryRow(query).Scan(&config.ID, &config.DeployType, &config.ExternalURL, &config.NodeIP, &config.GrafanaPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay error, no hay configuración aún
		}
		return nil, err
	}
	return &config, nil
}

func (r *SQLiteRepository) DeleteObservability() error {
	_, err := r.db.Exec("DELETE FROM observability_config WHERE id = 1")
	return err
}

// --- Operaciones de Interconexión de Servicios ---

func (r *SQLiteRepository) SaveServiceLink(link domain.ServiceLink) error {
	query := `
	INSERT INTO service_links (source_svc, target_svc, env_var_name, target_url)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(source_svc, env_var_name) DO UPDATE SET 
		target_svc=excluded.target_svc, 
		target_url=excluded.target_url;`
	_, err := r.db.Exec(query, link.SourceSvc, link.TargetSvc, link.EnvVarName, link.TargetURL)
	return err
}

func (r *SQLiteRepository) GetServiceLinks() ([]domain.ServiceLink, error) {
	query := `SELECT id, source_svc, target_svc, env_var_name, target_url FROM service_links;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []domain.ServiceLink
	for rows.Next() {
		var l domain.ServiceLink
		if err := rows.Scan(&l.ID, &l.SourceSvc, &l.TargetSvc, &l.EnvVarName, &l.TargetURL); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func (r *SQLiteRepository) DeleteServiceLink(sourceSvc, targetSvc string) error {
	_, err := r.db.Exec("DELETE FROM service_links WHERE source_svc = ? AND target_svc = ?", sourceSvc, targetSvc)
	return err
}

// --- Operaciones de Entornos de Preview Temporales ---

func (r *SQLiteRepository) SavePreviewEnv(env domain.SavedPreviewEnv) error {
	if env.Status == "" { env.Status = "active" }
	query := `
	INSERT INTO preview_envs (name, image_source, port, domain, link_db_name, status, target_node)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		image_source=excluded.image_source,
		port=excluded.port,
		domain=excluded.domain,
		link_db_name=excluded.link_db_name,
		status=excluded.status,
		target_node=excluded.target_node;`

	_, err := r.db.Exec(query, env.Name, env.ImageSource, env.Port, env.Domain, env.LinkDBName, env.Status, env.TargetNode)
	return err
}

func (r *SQLiteRepository) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error) {
	query := `SELECT id, name, image_source, port, domain, link_db_name, status, created_at, target_node FROM preview_envs ORDER BY id DESC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envs []domain.SavedPreviewEnv
	for rows.Next() {
		var e domain.SavedPreviewEnv
		if err := rows.Scan(&e.ID, &e.Name, &e.ImageSource, &e.Port, &e.Domain, &e.LinkDBName, &e.Status, &e.CreatedAt, &e.TargetNode); err != nil {
			return nil, err
		}
		envs = append(envs, e)
	}
	return envs, nil
}

func (r *SQLiteRepository) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) {
	query := `SELECT id, name, image_source, port, domain, link_db_name, status, created_at, target_node FROM preview_envs WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	var e domain.SavedPreviewEnv
	err := row.Scan(&e.ID, &e.Name, &e.ImageSource, &e.Port, &e.Domain, &e.LinkDBName, &e.Status, &e.CreatedAt, &e.TargetNode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteRepository) DeletePreviewEnv(name string) error {
	query := `DELETE FROM preview_envs WHERE name = ?;`
	_, err := r.db.Exec(query, name)
	return err
}

// --- Operaciones de Docker Registry Credentials ---

func (r *SQLiteRepository) SaveRegistryCredential(cred domain.SavedRegistryCredential) error {
	query := `
	INSERT INTO registry_credentials (server, username, password)
	VALUES (?, ?, ?)
	ON CONFLICT(server) DO UPDATE SET
		username=excluded.username,
		password=excluded.password;`
	_, err := r.db.Exec(query, cred.Server, cred.Username, cred.Password)
	return err
}

func (r *SQLiteRepository) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) {
	query := `SELECT id, server, username, password, created_at FROM registry_credentials ORDER BY id DESC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []domain.SavedRegistryCredential
	for rows.Next() {
		var c domain.SavedRegistryCredential
		if err := rows.Scan(&c.ID, &c.Server, &c.Username, &c.Password, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, nil
}

func (r *SQLiteRepository) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) {
	query := `SELECT id, server, username, password, created_at FROM registry_credentials WHERE server = ?;`
	row := r.db.QueryRow(query, server)

	var c domain.SavedRegistryCredential
	if err := row.Scan(&c.ID, &c.Server, &c.Username, &c.Password, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) DeleteRegistryCredential(server string) error {
	query := `DELETE FROM registry_credentials WHERE server = ?;`
	_, err := r.db.Exec(query, server)
	return err
}

// --- Operaciones del Gestor de Migraciones de BD ---

func (r *SQLiteRepository) SaveMigrationFile(file domain.MigrationFile) error {
	query := `
	INSERT INTO migration_files (db_name, filename, content, down_content, status)
	VALUES (?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'pending'))
	ON CONFLICT(db_name, filename) DO UPDATE SET
		content=excluded.content,
		down_content=excluded.down_content;`
	_, err := r.db.Exec(query, file.DBName, file.Filename, file.Content, file.DownContent, file.Status)
	return err
}

func (r *SQLiteRepository) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error) {
	query := `SELECT id, db_name, filename, content, down_content, status, executed_at, log_output FROM migration_files WHERE db_name = ? ORDER BY filename ASC;`
	rows, err := r.db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []domain.MigrationFile
	for rows.Next() {
		var f domain.MigrationFile
		if err := rows.Scan(&f.ID, &f.DBName, &f.Filename, &f.Content, &f.DownContent, &f.Status, &f.ExecutedAt, &f.LogOutput); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *SQLiteRepository) DeleteMigrationFile(dbName, filename string) error {
	query := `DELETE FROM migration_files WHERE db_name = ? AND filename = ?;`
	_, err := r.db.Exec(query, dbName, filename)
	return err
}

func (r *SQLiteRepository) RecordMigrationExecution(dbName, filename, status, logs string) error {
	query := `
	UPDATE migration_files 
	SET status = ?, executed_at = CURRENT_TIMESTAMP, log_output = ?
	WHERE db_name = ? AND filename = ?;`
	_, err := r.db.Exec(query, status, logs, dbName, filename)
	return err
}

func (r *SQLiteRepository) SaveBackup(b domain.SavedBackup) error {
	createdAt := b.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().Format("2006-01-02 15:04:05")
	}
	query := `
	INSERT INTO backups (target_name, target_type, engine, filename, file_path, size_bytes, status, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, b.TargetName, b.TargetType, b.Engine, b.Filename, b.FilePath, b.SizeBytes, b.Status, createdAt)
	return err
}

func (r *SQLiteRepository) GetBackups() ([]domain.SavedBackup, error) {
	query := `SELECT id, target_name, target_type, engine, filename, file_path, size_bytes, status, created_at FROM backups ORDER BY id DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SavedBackup
	for rows.Next() {
		var b domain.SavedBackup
		if err := rows.Scan(&b.ID, &b.TargetName, &b.TargetType, &b.Engine, &b.Filename, &b.FilePath, &b.SizeBytes, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, nil
}

func (r *SQLiteRepository) GetBackupByID(id int) (*domain.SavedBackup, error) {
	query := `SELECT id, target_name, target_type, engine, filename, file_path, size_bytes, status, created_at FROM backups WHERE id = ?`
	row := r.db.QueryRow(query, id)
	var b domain.SavedBackup
	if err := row.Scan(&b.ID, &b.TargetName, &b.TargetType, &b.Engine, &b.Filename, &b.FilePath, &b.SizeBytes, &b.Status, &b.CreatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *SQLiteRepository) DeleteBackup(id int) error {
	_, err := r.db.Exec(`DELETE FROM backups WHERE id = ?`, id)
	return err
}

// --- Audit Log Repository Methods (Delegated to pkg/auditlog non-blocking worker) ---

func (r *SQLiteRepository) SaveAuditLog(log domain.AuditLog) error {
	auditlog.GetDefaultLogger().Log(auditlog.AuditEntry{
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceName: log.ResourceName,
		Details:      log.Details,
		Timestamp:    log.Timestamp,
	})
	return nil
}

func (r *SQLiteRepository) GetAuditLogs(limit int) ([]domain.AuditLog, error) {
	entries, err := auditlog.GetDefaultLogger().ReadRecent(limit)
	if err != nil {
		return nil, err
	}
	var list []domain.AuditLog
	for _, e := range entries {
		list = append(list, domain.AuditLog{
			ID:           e.ID,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceName: e.ResourceName,
			Details:      e.Details,
			Timestamp:    e.Timestamp,
		})
	}
	return list, nil
}
