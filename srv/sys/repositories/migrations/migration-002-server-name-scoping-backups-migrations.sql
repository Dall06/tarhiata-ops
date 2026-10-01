-- Aisla migration_files y backups por servidor del fleet. Mismo patrón que
-- migration-001: SQLite no soporta ALTER TABLE para agregar un UNIQUE compuesto
-- nuevo, así que se recrea cada tabla.

ALTER TABLE migration_files RENAME TO migration_files_old_002;

CREATE TABLE migration_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	db_name TEXT NOT NULL,
	filename TEXT NOT NULL,
	content TEXT NOT NULL,
	down_content TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	executed_at TEXT NOT NULL DEFAULT '',
	log_output TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	server_name TEXT NOT NULL DEFAULT '',
	UNIQUE(db_name, filename, server_name)
);

INSERT INTO migration_files (id, db_name, filename, content, down_content, status, executed_at, log_output, created_at, server_name)
SELECT id, db_name, filename, content, down_content, status, executed_at, log_output, created_at, ''
FROM migration_files_old_002;

-- Backfill: el server_name de la BD referenciada (ya scoped desde migration-001) es el
-- dato correcto de a qué servidor pertenece cada archivo de migración.
UPDATE migration_files SET server_name = (
	SELECT server_name FROM databases WHERE databases.name = migration_files.db_name LIMIT 1
) WHERE server_name = '' AND EXISTS (SELECT 1 FROM databases WHERE databases.name = migration_files.db_name);

-- Si no hay ninguna BD con ese nombre en el catálogo (huérfano o ya borrada), cae al
-- mismo criterio de migration-001: el servidor activo, o si no hay ninguno, el primero.
UPDATE migration_files SET server_name = (
	SELECT name FROM server_configs ORDER BY is_active DESC, id ASC LIMIT 1
) WHERE server_name = '' AND EXISTS (SELECT 1 FROM server_configs);

DROP TABLE migration_files_old_002;

ALTER TABLE backups RENAME TO backups_old_002;

CREATE TABLE backups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	target_name TEXT NOT NULL,
	target_type TEXT NOT NULL,
	engine TEXT NOT NULL,
	filename TEXT NOT NULL,
	file_path TEXT NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'completed',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	server_name TEXT NOT NULL DEFAULT ''
);

INSERT INTO backups (id, target_name, target_type, engine, filename, file_path, size_bytes, status, created_at, server_name)
SELECT id, target_name, target_type, engine, filename, file_path, size_bytes, status, created_at, ''
FROM backups_old_002;

UPDATE backups SET server_name = (
	SELECT name FROM server_configs ORDER BY is_active DESC, id ASC LIMIT 1
) WHERE server_name = '' AND EXISTS (SELECT 1 FROM server_configs);

DROP TABLE backups_old_002;
