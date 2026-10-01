# Aislamiento por servidor de services/databases/service_links

## Contexto y motivación

Las tablas SQLite `services`, `databases` y `service_links` (`srv/sys/repositories/sqlite.go:140-219`) son globales: no tienen columna que identifique a qué servidor del fleet pertenece cada fila. En `srv/ui/handlers/web_server.go::handleSwarmStatus` (líneas ~1274-1346), y en al menos otros ~10 puntos de la UI web, cualquier servidor consultado recibe siempre la lista global completa de `w.repo.GetServices()/GetDatabases()/GetServiceLinks()`. Esto produce dos síntomas reportados: el servidor "local" muestra entidades que en realidad corren en otros servidores del fleet, y distintos servidores ven información repetida entre sí.

Es deuda de antes de que existiera el concepto multi-servidor ("fleet"): `services`/`databases` ya tienen una columna `target_node` (string libre, afinidad Swarm dentro de un único servidor — "manager"/"worker"/IP de nodo), pero eso es un concepto distinto de "a qué servidor del fleet pertenece la fila". `service_links` no tiene ni siquiera eso.

El problema es transversal, no exclusivo de la UI web: los usecases `sync_cluster_state.go`, `bootstrap_master.go`, `unlink_services.go`, `manage_ssl.go`, `manage_migrations.go`, y todo el CLI (`srv/cli/usecases/*`, `cmd/tarhiata/main.go`) hacen la misma asunción global.

El servidor "activo" de una request web ya se resuelve hoy por nombre único: `getTargetServerConfig`/`resolveTargetServer` (`web_server.go:100-112, 896-917`) leen `?server=`/`?name=` o caen al `server_config` con `is_active=1`. Ese nombre nunca se propaga hacia los repos de services/databases/links.

## Alcance

**Dentro de alcance:**
- Columna `server_name TEXT` en `services`, `databases` y `service_links`.
- Cambio de contrato `ConfigRepository` (`srv/sys/ports/repo.go`) para que `GetServices`/`GetDatabases`/`GetServiceLinks` reciban `serverName` y filtren por él.
- Propagación del `serverName` a todos los callers: handlers web, usecases, y CLI (`srv/cli/usecases/*`, `cmd/tarhiata/main.go`).
- Sistema de migraciones nuevo basado en archivos `.sql` versionados (ver sección correspondiente), con tracking idempotente, aplicado automáticamente al arrancar **y** disponible vía comando CLI manual.
- Actualización de mocks y tests existentes a la nueva firma.

**Fuera de alcance (explícitamente diferido):**
- Migrar retroactivamente las columnas ya agregadas con `addColumnIfMissing` (`target_node`, etc.) al nuevo sistema de archivos `.sql`. Siguen como están; solo los cambios de esquema de aquí en adelante usan el sistema nuevo.
- Wizard interactivo de selección de entorno/tenant (no aplica: tarhiata-ops es un único daemon con una sola SQLite por instancia, no multi-tenant Postgres remoto como bro-ops).
- Deploy keys SSH, TLS, y cualquier tema no relacionado al scoping por servidor.

## Modelo de datos

`services`:
```sql
server_name TEXT NOT NULL DEFAULT ''
-- UNIQUE(name) → UNIQUE(name, server_name)
```

`databases`: mismo tratamiento — `server_name TEXT NOT NULL DEFAULT ''`, `UNIQUE(name)` → `UNIQUE(name, server_name)`.

`service_links`: columna propia `server_name TEXT NOT NULL DEFAULT ''` (no se deriva de `source_svc` vía JOIN — decisión explícita para evitar una query más compleja). `UNIQUE(source_svc, env_var_name)` → `UNIQUE(source_svc, env_var_name, server_name)`.

**Unicidad:** el mismo `name` de servicio puede existir en servidores distintos del fleet (ej. "api" desplegado en "local" y en "vps-prod"); la combinación `(name, server_name)` es lo que debe ser único, no `name` solo.

**Backfill de filas preexistentes:**
```sql
UPDATE services SET server_name = (SELECT name FROM server_configs WHERE is_active = 1) WHERE server_name = '';
-- análogo para databases y service_links
```
Asume que los datos de antes del concepto fleet pertenecen al servidor actualmente marcado `is_active=1`. Si no hay ningún `server_config` activo al migrar, no hay filas que backfillear (tampoco habría `services` creados sin servidor activo).

SQLite no soporta `ALTER TABLE ... DROP CONSTRAINT`; el cambio de `UNIQUE` requiere el patrón estándar: crear tabla nueva con el constraint correcto, copiar datos, drop de la tabla vieja, rename — explícito en el `.sql` de la migración.

`domain.SavedService`, `domain.SavedDatabase`, y la entidad de link ganan el campo `ServerName string`.

## Sistema de migraciones

Carpeta nueva `srv/sys/repositories/migrations/` con archivos `migration-NNN-descripcion.sql` (convención de naming tomada del repo hermano `bro-ops`, adaptada: ahí las migraciones se corren manualmente por un operador vía SSH contra Postgres multi-tenant remoto sin tracking; acá hay una sola SQLite embebida por instancia que se auto-migra sola).

- Primer archivo: `migration-001-server-name-scoping.sql` con el DDL de la sección anterior.
- Tabla de control `schema_migrations(filename TEXT PRIMARY KEY, applied_at DATETIME)`.
- Los `.sql` se embeben en el binario (`//go:embed migrations/*.sql`, stdlib) para no depender del filesystem del servidor desplegado.
- Runner único `ApplyPendingMigrations(db)`: lista los `.sql` embebidos ordenados por el número del nombre, salta los ya registrados en `schema_migrations`, ejecuta cada uno en una transacción, inserta el registro si tiene éxito.
- `migrate()` en `sqlite.go` llama a este runner al abrir la conexión (auto-apply al arrancar, sin cambio de comportamiento para el operador).
- Nuevo subcomando CLI `tarhiata migrate` (`srv/cli/`) con `status` (aplicadas/pendientes) y `up` (corre las pendientes) — llama al mismo runner, no reimplementa nada. Ambos modos conviven porque el tracking es idempotente: correr `up` manualmente después de que el daemon ya auto-aplicó no reintenta nada.
- El `addColumnIfMissing` actual queda intacto para lo ya agregado con ese mecanismo.

## Cambio de contrato y propagación

`srv/sys/ports/repo.go` (`ConfigRepository`):
```go
GetServices(serverName string) ([]domain.SavedService, error)
GetDatabases(serverName string) ([]domain.SavedDatabase, error)
GetServiceLinks(serverName string) ([]domain.ServiceLink, error)
// Save*/Create* reciben server_name como parte de la entidad, no como parámetro aparte
```

- **Handlers web** (`web_server.go`, ~10 puntos incluido `handleSwarmStatus`): ya resuelven el servidor de la request vía `getTargetServerConfig`/`resolveTargetServer` — se les agrega pasar `cfg.Name` a las llamadas del repo.
- **Usecases** (`sync_cluster_state.go`, `bootstrap_master.go`, `unlink_services.go`, `manage_ssl.go`, `manage_migrations.go`): ya operan en el contexto de un servidor conocido (viene del handler que los invoca) — ganan `serverName` como parámetro de su función pública.
- **CLI** (`srv/cli/usecases/dashboard.go`, `service.go`, `database.go`, `cmd/tarhiata/main.go`): hoy no resuelven "servidor activo" al listar — se les agrega resolver igual que la UI web (default: `server_config` con `is_active=1`), con flag `--server <name>` para override donde el comando ya acepta flags.
- `srv/sys/tests/mocks/mock.go` y `srv/ui/handlers/web_server_test.go`: firmas actualizadas en el mismo PR, no como deuda aparte.

## Testing

- Runner de migraciones: aplicar `migration-001` sobre SQLite en memoria con datos legacy preinsertados → verificar backfill al `server_name` del `is_active=1`; correr `ApplyPendingMigrations` dos veces seguidas → la segunda no reintenta nada (idempotencia vía `schema_migrations`).
- `UNIQUE(name, server_name)`: mismo `name` en dos `server_name` distintos → éxito; mismo `name` + mismo `server_name` dos veces → falla, igual que antes.
- `GetServices(serverName)`/`GetDatabases(serverName)`/`GetServiceLinks(serverName)`: con datos de dos servidores sembrados, cada llamada devuelve solo las filas de su `server_name` — este es el test que cierra el bug original.
- Mocks y tests existentes (`web_server_test.go`, `main_test.go`) actualizados a la nueva firma — el repo no compila sin esto.
- Sin tests end-to-end nuevos de Swarm real (el repo no los tiene para estas rutas hoy; no es parte de este cambio).
