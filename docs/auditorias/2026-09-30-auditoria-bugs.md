# Auditoría de bugs — 2026-09-30

Auditoría de solo lectura (sin cambios de código) realizada con 3 agentes en paralelo:
1. Backend Go — `srv/sys/repositories/*`, `srv/sys/usecases/*`, migraciones.
2. Capa web/CLI — `srv/ui/handlers/web_server.go`, `cmd/tarhiata/main.go`, `srv/cli/usecases/*`, seguridad.
3. Frontend — `srv/ui/views/public/*`, `srv/ui/components/*/*.js`, `pkg/*`.

Contexto: disparada justo después del commit `2be38c3` (aislamiento por servidor de `services`/`databases`/`service_links`) y del agrupamiento de los 5 botones de la navbar en el dropdown "⚙️ Herramientas ▾".

Las líneas (`archivo:línea`) son las reportadas por los agentes al momento de la auditoría — verificar antes de corregir, el código puede haberse movido.

---

## CRÍTICOS

### Seguridad — endpoints sin autenticación en `srv/ui/handlers/web_server.go`

El hallazgo más grave. CORS está en `AllowOrigins: []string{"*"}` (líneas 248-252), así que cualquier página web que el admin visite puede disparar estos fetch contra `localhost:8080`.

- **`handleSSHKeys`** (4639-4695, ruta 437): sin `echoAuth`/`isAuthorized`. POST agrega una llave pública a `authorized_keys` del VPS, DELETE la quita → toma de control SSH root completa.
- **`handleServers` GET** (698-707, ruta 364): sin auth. Filtra `PrivateKey`/`VultrAPIToken`/`DOAPIToken` **desencriptados** de todo el fleet.
- **`handleServices` GET** (1399-1417, ruta 350): sin auth. Filtra `WebhookSecret`/`GitAccessToken`/`EnvVars` en claro.
- **`handleServices` DELETE** (1532-1581): sin ningún chequeo de auth (a diferencia de su propio POST) → cualquiera borra el servicio y dispara `docker service rm` remoto.
- **`handleNodeJoinToken`** (3241-3294, ruta 416): sin auth. Devuelve el **manager join-token** de Swarm → control total del clúster.
- **`handleTopology`** (2877-2938, ruta 413): sin auth. En texto plano expone la contraseña real de cada BD.
- **`handleObservability`** (2480-2569, ruta 381): sin auth. GET expone `grafana_password`; POST borra el stack.
- **`handleCustomDomains`** (4344-4407, ruta 407): sin auth, muta ruteo de Traefik vía SSH.
- **`handleRegistries`** (3693-3753, ruta 385): sin auth, guarda/borra credenciales de registry con `docker login` remoto.
- **`handlePreviewEnvs`** (3633-3691, ruta 384): sin auth, crea/destruye infraestructura real.
- **`handleDownloadBackup`** (3961-3992, ruta 393): sin auth, descarga dump completo de BD.
- **`handleEnvVars` POST** (4021-4053, ruta 394): sin auth, sobrescribe `.env` y redeploya vía SSH.
- **`handleExportEnvVars`** (4056-4080, ruta 395): sin auth, exporta `.env` completo en claro.
- **`handleSyncState`** (4589-4622, ruta 436): sin auth, acepta incluso GET mutante que puede sobrescribir el `state.json` remoto.
- **`handleMigrationFile` POST** (3765-3789, ruta 387): sin auth, permite plantar SQL malicioso en el catálogo de migraciones.
- **`handleHostInspect/Metrics/Services/Devices/Security`, `handleSystemReport`, `handleSwarmStatus`** (924-1355, rutas 369-375): sin auth, exponen reglas de firewall, jaulas de fail2ban, hostname, versión de Docker, y fuerzan conexiones SSH arbitrarias vía `?server=`.

### Inyección de comandos preexistente

- **`srv/sys/usecases/deploy_database.go:195,203,214-278`** — `db.VolumeHostPath` se interpola crudo en comandos SSH (`rm -rf %s/*`, `mkdir/chown`, `docker service create --mount`) sin pasar nunca por `validator.ShellQuote` ni validación de formato en todo el flujo desde el POST. RCE como root en el host Docker.

### Bug de scoping — el más grave de esta categoría (el bug original que motivó la iniciativa)

- **`handleWebhookDeploy`** (4998-5138): resuelve correctamente `webhookCfg` para validar la firma HMAC, pero luego usa `cfg := w.getConfig()` (servidor globalmente activo) para el deploy real → un webhook de CI para un servicio del Servidor B puede desplegar/sobrescribir el servicio homónimo del Servidor A.

### Pérdida silenciosa de datos en migración

- **`migrations/migration-001-server-name-scoping.sql:36-37,62-63,83-84`** — si no hay ningún `server_configs.is_active=1` al momento de migrar, el backfill deja `server_name=''` en TODAS las filas existentes. Como ningún `GetServices`/`GetService` filtra jamás por `""`, ese catálogo queda invisible para siempre, sin log ni error.

### Sync roto entre servidores del fleet

- **`srv/sys/usecases/deploy_service.go:133`** y **`deploy_database.go:70`** — `ExportStateToRemote("")` hardcodeado en vez de propagar `config.Name` (el dato está disponible a 3 líneas de distancia en `deploy_database.go`). Cada deploy sincroniza al VPS un catálogo vacío/incorrecto en cualquier fleet con más de un servidor.

### XSS real

- **`srv/ui/pkg/toast/toast.js:18`** — `showToast()` inyecta el mensaje vía `innerHTML` sin `escapeHtml`, y decenas de call-sites le pasan strings de backend (`res.error`) o del propio input del usuario sin sanear.

---

## ALTOS

- `srv/sys/repositories/sqlite.go:536-549` — `DeleteServerConfig` no promueve otro servidor a activo tras borrar el activo; `GetServerConfig()` cae a un fallback legacy con `Name="default"` que no calza con ningún servidor real → "se desaparecieron los servicios".
- `sqlite.go:352-419, 509-515` — `SaveServerConfig`/`SetActiveServerConfig` no son atómicos (dos `Exec` sueltos) → puede terminar con 0 o 2 servidores `is_active=1` bajo concurrencia/crash.
- `srv/sys/usecases/manage_envs.go:80-92` — `UpdateEnvVars` crea un `SavedService` duplicado con `server_name=""` cuando no encuentra el servicio (en vez de error, como sí hace `manage_domains.go`).
- `isAuthorized` (`web_server.go:205`) compara la API key con `==` en vez de `subtle.ConstantTimeCompare`/`hmac.Equal` → timing attack.
- `srv/sys/usecases/manage_nodes.go:201,230` + `handleNodeLabels` (3380-3417) — `input.Key`/`input.Value` sin validar, interpolados crudos en `docker node update --label-add %s=%s`.
- `handleDatabaseItem` DELETE (`web_server.go:1972`) usa `w.getConfig()` en vez de `resolveTargetServer(?server=)` como sus hermanos GET/PUT → `?server=` ignorado, SSH ejecuta contra el servidor activo, y el borrado en catálogo es un no-op silencioso si no coincide `server_name`.
- `handleLinks` POST/DELETE (2980-3032) ignoran `?server=` (a diferencia de su propio GET) → link/unlink siempre opera sobre el servidor activo global.
- Handlers que ignoran `?server=` por completo: `handlePrune`, `handleRestartTraefik`, `handleRepairTraefik`, `handleServerUpdate`, `handleLogs`, `handleRunMigrations`, `handleObservabilityMetrics`, `handleContainerStats`, `handleDBHealth`, `handleServiceRebuild`, GET de `handleNodes`.
- `srv/cli/usecases/config.go` — el wizard `tarhiata config` nunca asigna un `Name` estable al `ServerConfig` (cae a `Host`/IP); si la IP cambia, huérfana todos los `services`/`databases`/`links` ya escopeados — justo el escenario que el refactor de `server_name` buscaba evitar.
- `srv/ui/views/public/app.js:150-167` + `components/fleet/fleet.js:442-458` — el dropdown `#navToolsDropdown` y `#serverPopover` usan `stopPropagation()` sobre el mismo listener global de "cerrar al click afuera" → pueden quedar ambos abiertos simultáneamente.
- `app.js:148` + `components/ssl/ssl.js:269` — `btnTopSSL` quedó con doble listener tras el refactor del dropdown → doble fetch/doble render al click.

---

## MEDIOS

- Backfill de migración-001 con `LIMIT 1` no determinista si llegara a haber más de un `server_configs.is_active=1`.
- `addColumnIfMissing` (sqlite.go) interpola tabla/columna con `fmt.Sprintf` sin parametrizar — footgun para SQLi futura.
- `manage_migrations.go` (`GetMigrationFiles`) y `manage_backups.go` (`GetBackups`) — las tablas `migration_files` y `backups` nunca ganaron columna `server_name`; colisionan entre servidores con recursos homónimos.
- `bootstrap_master.go` no propaga `input.TargetNode` al deploy real; además despliega en remoto ANTES de `SaveDatabase`/`SaveService` → recursos huérfanos si el save falla.
- `web_server.go:5072` — build-from-source se lanza en goroutine sin drenado/cancelación en shutdown → build colgado a medias si el proceso reinicia.
- `handleLogs` (3511-3575) — si SSH falla, devuelve logs **sintéticos/fabricados** ("Healthcheck PASSED") indistinguibles de reales, en vez de error.
- CLI no soporta `--server <nombre>` en ningún subcomando (inconsistente con el panel web).
- 7 modales ad-hoc (alerts, audit, terminal, env, logs, ssl, volumes) nunca reciben `.active` → el Escape global nunca los cierra; alerts/audit/terminal tampoco cierran con click afuera.
- `telemetry.js:335-357` — el bloque `catch` de `refreshServerTelemetry` escribe en DOM sin el guard de `requestId`/`selectedServerName` que sí tienen las líneas de éxito — misma clase de bug ya corregida en `processSwarmStatus`, reapareció en la rama de error tardío.
- Ningún de los 3 patrones de dropdown (navbar, server popover, row-actions) maneja teclado: sin Escape, sin `aria-expanded`/`aria-haspopup`/`role="menu"`.
- `handleDatabaseItem` GET no redacta `db.Password` (a diferencia de `handleDatabases` GET) — inconsistencia de redacción en el mismo recurso.

---

## BAJOS / deuda técnica

- `splitSQLStatements` (migrations.go) hace split naive por `;` — bomba de tiempo si una migración futura mete `;` dentro de un string o comentario.
- `migrationNumber` — dos archivos con el mismo NNN no tienen orden garantizado; nombre que no matchea el patrón devuelve `0` silenciosamente.
- `GetServerConfig()` fallback legacy marca `IsActive=true` incondicionalmente.
- `db.DeployType = "manager"` en `bootstrap_master.go` no es ninguno de los 3 valores documentados en `domain.SavedDatabase.DeployType`.
- Error de `linkUC.Execute` en `bootstrap_master.go` se descarta en silencio.
- `handleListSSHKeys` (2037-2072) es código muerto, no está registrado en ninguna ruta.
- `handleTerminalExec` mezcla quoting `%q` con un fallback sin comillas.
- `escapeHtml(str)` (`pkg/jsutil/utils.js:6-8`) trata `0`/`false` como vacío → un puerto `0` se renderiza en blanco.
- `isTestingAll`/`isPollingTelemetry` en `state.js` nunca se usan; `setupModalDismissals()` es un bucle muerto al iniciar.
- Botón `▾` de `row-action-chevron` solo tiene `title`, sin `aria-label`.
- `dashboard.go:111-115` (CLI) imprime "✔ Swarm Node Active"/"✔ SSL Active" fijo, sin verificar nada contra el servidor real.
