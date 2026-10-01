-- Aisla services/databases/service_links por servidor del fleet.
-- SQLite no soporta ALTER TABLE para cambiar un UNIQUE, así que se recrea cada tabla.

ALTER TABLE services RENAME TO services_old_001;

CREATE TABLE services (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
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
	dockerfile_path TEXT NOT NULL DEFAULT '',
	server_name TEXT NOT NULL DEFAULT '',
	UNIQUE(name, server_name)
);

INSERT INTO services (id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json, env_vars, target_node, pre_deploy_hook, custom_domains, webhook_secret, source_type, git_repo_url, git_branch, git_access_token, dockerfile_path, server_name)
SELECT id, name, image_source, is_url, port, domain, expose, env_file_path, enable_ssl, healthcheck_cmd, mounts_json, env_vars, target_node, pre_deploy_hook, custom_domains, webhook_secret, source_type, git_repo_url, git_branch, git_access_token, dockerfile_path, ''
FROM services_old_001;

UPDATE services SET server_name = (SELECT name FROM server_configs WHERE is_active = 1 LIMIT 1)
WHERE server_name = '' AND (SELECT COUNT(*) FROM server_configs WHERE is_active = 1) > 0;

DROP TABLE services_old_001;

ALTER TABLE databases RENAME TO databases_old_001;

CREATE TABLE databases (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	engine TEXT NOT NULL,
	deploy_type TEXT NOT NULL,
	external_url TEXT NOT NULL,
	internal_port INTEGER NOT NULL,
	volume_host_path TEXT NOT NULL,
	node_ip TEXT NOT NULL,
	password TEXT NOT NULL DEFAULT '',
	target_node TEXT NOT NULL DEFAULT '',
	server_name TEXT NOT NULL DEFAULT '',
	UNIQUE(name, server_name)
);

INSERT INTO databases (id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password, target_node, server_name)
SELECT id, name, engine, deploy_type, external_url, internal_port, volume_host_path, node_ip, password, target_node, ''
FROM databases_old_001;

UPDATE databases SET server_name = (SELECT name FROM server_configs WHERE is_active = 1 LIMIT 1)
WHERE server_name = '' AND (SELECT COUNT(*) FROM server_configs WHERE is_active = 1) > 0;

DROP TABLE databases_old_001;

ALTER TABLE service_links RENAME TO service_links_old_001;

CREATE TABLE service_links (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_svc TEXT NOT NULL,
	target_svc TEXT NOT NULL,
	env_var_name TEXT NOT NULL,
	target_url TEXT NOT NULL,
	server_name TEXT NOT NULL DEFAULT '',
	UNIQUE(source_svc, env_var_name, server_name)
);

INSERT INTO service_links (id, source_svc, target_svc, env_var_name, target_url, server_name)
SELECT id, source_svc, target_svc, env_var_name, target_url, ''
FROM service_links_old_001;

UPDATE service_links SET server_name = (SELECT name FROM server_configs WHERE is_active = 1 LIMIT 1)
WHERE server_name = '' AND (SELECT COUNT(*) FROM server_configs WHERE is_active = 1) > 0;

DROP TABLE service_links_old_001;
