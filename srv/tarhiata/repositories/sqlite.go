package repositories

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
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

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("falló al abrir sqlite: %w", err)
	}

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
		do_api_token TEXT NOT NULL DEFAULT ''
	);`
	if _, err := r.db.Exec(query); err != nil {
		return err
	}

	_, errAlter0 := r.db.Exec("ALTER TABLE server_config ADD COLUMN do_api_token TEXT NOT NULL DEFAULT '';")
	if errAlter0 != nil && !strings.Contains(errAlter0.Error(), "duplicate column name") {
		return errAlter0
	}

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
		mounts_json TEXT NOT NULL DEFAULT '[]'
	);`
	if _, err := r.db.Exec(queryServices); err != nil {
		return err
	}

	// Mini migración para bases de datos existentes
	_, errAlter1 := r.db.Exec("ALTER TABLE services ADD COLUMN enable_ssl BOOLEAN NOT NULL DEFAULT 0;")
	if errAlter1 != nil && !strings.Contains(errAlter1.Error(), "duplicate column name") {
		return errAlter1
	}
	_, errAlter2 := r.db.Exec("ALTER TABLE services ADD COLUMN healthcheck_cmd TEXT NOT NULL DEFAULT '';")
	if errAlter2 != nil && !strings.Contains(errAlter2.Error(), "duplicate column name") {
		return errAlter2
	}
	_, errAlter3 := r.db.Exec("ALTER TABLE services ADD COLUMN mounts_json TEXT NOT NULL DEFAULT '[]';")
	if errAlter3 != nil && !strings.Contains(errAlter3.Error(), "duplicate column name") {
		return errAlter3
	}

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
		password TEXT NOT NULL DEFAULT ''
	);`
	if _, err := r.db.Exec(queryDBs); err != nil {
		return err
	}

	_, errAlter4 := r.db.Exec("ALTER TABLE databases ADD COLUMN password TEXT NOT NULL DEFAULT '';")
	if errAlter4 != nil && !strings.Contains(errAlter4.Error(), "duplicate column name") {
		return errAlter4
	}

	// Tabla de Observabilidad
	queryObs := `
	CREATE TABLE IF NOT EXISTS observability_config (
		id INTEGER PRIMARY KEY CHECK (id = 1), 
		deploy_type TEXT NOT NULL,
		external_url TEXT NOT NULL,
		node_ip TEXT NOT NULL
	);`
	if _, err := r.db.Exec(queryObs); err != nil {
		return err
	}

	return nil
}

func (r *SQLiteRepository) SaveServerConfig(config domain.ServerConfig) error {
	// Usamos UPSERT: Inserta si no existe, si existe lo actualiza.
	query := `
	INSERT INTO server_config (id, host, port, user, private_key, do_api_token) 
	VALUES (1, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		host=excluded.host, 
		port=excluded.port, 
		user=excluded.user, 
		private_key=excluded.private_key,
		do_api_token=excluded.do_api_token;`

	_, err := r.db.Exec(query, config.Host, config.Port, config.User, config.PrivateKey, config.DOAPIToken)
	return err
}

func (r *SQLiteRepository) GetServerConfig() (*domain.ServerConfig, error) {
	query := `SELECT host, port, user, private_key, do_api_token FROM server_config WHERE id = 1;`
	row := r.db.QueryRow(query)

	var config domain.ServerConfig
	err := row.Scan(&config.Host, &config.Port, &config.User, &config.PrivateKey, &config.DOAPIToken)
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
	INSERT INTO services (name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		image_source=excluded.image_source,
		is_url=excluded.is_url,
		port=excluded.port,
		domain=excluded.domain,
		expose=excluded.expose,
		env_file_path=excluded.env_file_path,
		enable_ssl=excluded.enable_ssl,
		healthcheck_cmd=excluded.healthcheck_cmd,
		mounts_json=excluded.mounts_json;`

	_, err := r.db.Exec(query, svc.Name, svc.ImageSource, svc.IsURL, svc.Port, svc.Domain, svc.Expose, svc.EnvFilePath, svc.EnableSSL, svc.HealthcheckCmd, svc.MountsJSON)
	return err
}

func (r *SQLiteRepository) GetServices() ([]domain.SavedService, error) {
	query := `SELECT id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json FROM services ORDER BY name ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.SavedService
	for rows.Next() {
		var s domain.SavedService
		if err := rows.Scan(&s.ID, &s.Name, &s.ImageSource, &s.IsURL, &s.Port, &s.Domain, &s.Expose, &s.EnvFilePath, &s.EnableSSL, &s.HealthcheckCmd, &s.MountsJSON); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *SQLiteRepository) GetService(name string) (*domain.SavedService, error) {
	query := `SELECT id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json FROM services WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	var s domain.SavedService
	err := row.Scan(&s.ID, &s.Name, &s.ImageSource, &s.IsURL, &s.Port, &s.Domain, &s.Expose, &s.EnvFilePath, &s.EnableSSL, &s.HealthcheckCmd, &s.MountsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No encontrado
		}
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteRepository) DeleteService(name string) error {
	_, err := r.db.Exec("DELETE FROM services WHERE name = ?", name)
	return err
}

// --- Operaciones del Catálogo de Bases de Datos ---

func (r *SQLiteRepository) SaveDatabase(db domain.SavedDatabase) error {
	query := `
	INSERT INTO databases (name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		engine=excluded.engine,
		deploy_type=excluded.deploy_type,
		external_url=excluded.external_url,
		internal_port=excluded.internal_port,
		volume_host_path=excluded.volume_host_path,
		node_ip=excluded.node_ip,
		password=excluded.password;`

	_, err := r.db.Exec(query, db.Name, db.Engine, db.DeployType, db.ExternalURL, db.InternalPort, db.VolumeHostPath, db.NodeIP, db.Password)
	return err
}

func (r *SQLiteRepository) GetDatabases() ([]domain.SavedDatabase, error) {
	query := `SELECT id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password FROM databases ORDER BY name ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []domain.SavedDatabase
	for rows.Next() {
		var d domain.SavedDatabase
		if err := rows.Scan(&d.ID, &d.Name, &d.Engine, &d.DeployType, &d.ExternalURL, &d.InternalPort, &d.VolumeHostPath, &d.NodeIP, &d.Password); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

func (r *SQLiteRepository) GetDatabase(name string) (*domain.SavedDatabase, error) {
	query := `SELECT id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password FROM databases WHERE name = ?;`
	row := r.db.QueryRow(query, name)

	var d domain.SavedDatabase
	err := row.Scan(&d.ID, &d.Name, &d.Engine, &d.DeployType, &d.ExternalURL, &d.InternalPort, &d.VolumeHostPath, &d.NodeIP, &d.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No encontrado
		}
		return nil, err
	}
	return &d, nil
}

func (r *SQLiteRepository) DeleteDatabase(name string) error {
	_, err := r.db.Exec("DELETE FROM databases WHERE name = ?", name)
	return err
}

// --- Operaciones de Observabilidad ---

func (r *SQLiteRepository) SaveObservability(obs domain.SavedObservability) error {
	query := `
	INSERT INTO observability_config (id, deploy_type, external_url, node_ip) 
	VALUES (1, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		deploy_type=excluded.deploy_type, 
		external_url=excluded.external_url, 
		node_ip=excluded.node_ip;`

	_, err := r.db.Exec(query, obs.DeployType, obs.ExternalURL, obs.NodeIP)
	return err
}

func (r *SQLiteRepository) GetObservability() (*domain.SavedObservability, error) {
	query := `SELECT id, deploy_type, external_url, node_ip FROM observability_config WHERE id = 1;`
	row := r.db.QueryRow(query)

	var obs domain.SavedObservability
	err := row.Scan(&obs.ID, &obs.DeployType, &obs.ExternalURL, &obs.NodeIP)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No hay error, no hay configuración aún
		}
		return nil, err
	}
	return &obs, nil
}

func (r *SQLiteRepository) DeleteObservability() error {
	_, err := r.db.Exec("DELETE FROM observability_config WHERE id = 1")
	return err
}
