# Build-from-source (git push to deploy)

## Contexto y motivación

Hoy `tarhiata-ops` despliega servicios de dos formas, ambas en `srv/sys/usecases/deploy_service.go::provisionImage`:

1. **`IsURL=false`**: `docker pull <ImageSource>` contra un registry (Docker Hub, GHCR, etc.) — el usuario ya construyó y publicó la imagen por su cuenta.
2. **`IsURL=true`**: `wget` + `unzip` + `docker load` de un **zip con una imagen ya empaquetada**. `IsURL` nunca significó "repo git".

El webhook de auto-deploy (`srv/sys/usecases/trigger_webhook_deploy.go`, handler en `web_server.go:handleWebhookDeploy`) solo re-pullea un tag existente (`docker service update --image`). No hay git awareness en ningún punto del flujo: el payload del webhook hoy solo trae `service` + `image`, nunca repo/branch/commit. No existe build de imágenes (`docker build`) en ningún lugar del repo, ni un registry Docker privado propio, ni manejo de credenciales git.

**Objetivo de negocio:** reposicionar `tarhiata-ops` como un PaaS vendible al estilo Railway — "conecta tu repo, haz push, se despliega" — sin perder la propuesta actual (self-hosted, 0MB overhead, dueño de tu infra).

**Ya resuelto en esta misma sesión, previo a este diseño:** el webhook de auto-deploy tenía su verificación de firma HMAC rota (comparaba contra un secreto que venía en la misma request entrante — bypass trivial) y fallaba abierto sin firma. Se corrigió con un secreto por servicio (`SavedService.WebhookSecret`, cifrado en reposo) validado server-side, fail-closed. Esta feature se construye sobre esa base ya segura.

## Alcance

**Dentro de alcance (v1):**
- Build de imagen Docker **solo a partir de un `Dockerfile` en el repo** (sin autodetección de stack, sin Nixpacks/buildpacks).
- Un registry Docker privado propio (`registry:2`), interno al Swarm, sin TLS (no se expone fuera de la red overlay).
- Fetch del repo vía HTTPS + Personal Access Token (PAT) opcional, cifrado en reposo — sirve para GitHub, GitLab, Gitea, Bitbucket por igual.
- `SavedService` gana un tercer "origen" (`SourceType: "git"`), junto a los dos que ya existen (`"image"`, `"archive"`) — **sin alterar el comportamiento de los dos existentes**.
- El webhook existente (`/api/webhooks/deploy`, firma ya corregida) dispara el build en background cuando el servicio es `SourceType=git`, y responde rápido (202) sin hacer esperar al proveedor de git.
- Un botón manual "Rebuild & Deploy" en la UI dispara el mismo pipeline de forma síncrona con streaming directo.
- Logs de build en vivo, vía un registro de builds en memoria + endpoint de streaming NDJSON.
- Soporte de payload de webhook para GitHub, GitLab y Gitea (los 3 que el código ya menciona soportar).

**Fuera de alcance (explícitamente diferido):**
- Autodetección de stack / Nixpacks / buildpacks (Dockerfile es obligatorio en v1).
- Deploy keys SSH por servicio (solo PAT vía HTTPS).
- Rollback automático si el redeploy final falla (hoy tampoco existe; no se agrega aquí).
- Cache de capas de Docker entre builds, build concurrency limits por nodo, cola de builds — un build a la vez por servicio es suficiente para v1.
- TLS/exposición pública del registry propio.
- Multi-arquitectura (build solo para la arquitectura del nodo manager).

## Modelo de datos

`domain.SavedService` (ya existe en `srv/sys/domain/entities.go`) gana:

```go
SourceType     string `json:"sourceType"`     // "image" (default, retrocompatible) | "archive" | "git"
GitRepoURL     string `json:"gitRepoUrl"`      // https://github.com/org/repo.git
GitBranch      string `json:"gitBranch"`       // default "main"
GitAccessToken string `json:"gitAccessToken"`  // PAT opcional, cifrado en reposo (igual que WebhookSecret)
DockerfilePath string `json:"dockerfilePath"`  // default "Dockerfile", relativo a la raíz del repo
```

Migración SQLite: 4 columnas nuevas vía `addColumnIfMissing` en la tabla `services`, con default `''` — ningún servicio existente cambia de comportamiento (`SourceType` vacío se trata como `"image"`, igual que hoy).

Nueva tabla/concepto en memoria (no persistido en SQLite, vive y muere con el proceso, igual que `sshclient.GlobalPool`):

```go
type BuildJob struct {
    ID        string
    Service   string
    Status    string // "running" | "success" | "failed"
    Lines     []string
    StartedAt time.Time
    mu        sync.Mutex
    done      chan struct{}
}
```

## Infraestructura: registry propio

- Se agrega `registry:2` al stack de infraestructura que ya despliega `init_server.go` (junto a Traefik), en la red overlay interna, puerto `5000` no publicado al host.
- Cada nodo del Swarm necesita `"insecure-registries": ["tarhiata-registry:5000"]` en `/etc/docker/daemon.json` + `systemctl restart docker` — paso nuevo en `init_server.go` (bootstrap) y en el flujo de "unirse al cluster" para workers nuevos (`provision_worker.go` / `handleNodeJoinToken`), para que puedan hacer `docker pull` de ahí sin TLS. El restart del daemon tira momentáneamente los contenedores de ESE nodo (Swarm los reprograma en otro nodo si hay replicas); por eso este paso va en el bootstrap/join inicial, no como algo que se repite en nodos ya en producción.
- Nombre de imagen generado: `tarhiata-registry:5000/<service-name>:<commit-corto>`.

## Arquitectura

**Nuevo usecase** `srv/sys/usecases/build_from_source.go`:

```go
type BuildFromSourceUseCase struct {
    ssh ports.SSHExecutor
}

func (uc *BuildFromSourceUseCase) Execute(svc domain.SavedService, commitRef string, config domain.ServerConfig, onLine func(string)) (imageTag string, err error)
```

- Conecta SSH al manager (mismo patrón que todo el repo: `Connect` + `defer Close()`).
- `git clone --branch <GitBranch> <url-con-PAT-si-hay> /opt/tarhiata/builds/<service>-<timestamp>` (sin `--depth 1`: si llegan dos pushes casi juntos, un shallow clone podría no alcanzar a traer el commit exacto que reportó el webhook) y luego `git checkout <commitRef>` si vino uno; si no vino (botón manual sin build previo), se queda en el HEAD del branch. Cada línea de salida se pasa a `onLine` para alimentar el log en vivo.
- `docker build -f <DockerfilePath> -t tarhiata-registry:5000/<service>:<commitRef> .`
- `docker push tarhiata-registry:5000/<service>:<commitRef>`
- `rm -rf` del workdir de clone (no deja basura).
- Devuelve el tag final; el **redeploy en sí reusa `TriggerWebhookDeployUseCase`/`DeployServiceUseCase` tal cual existen hoy** — este usecase nuevo solo construye y publica la imagen, no duplica lógica de despliegue.
- El PAT se interpola en la URL de clone con `validator.ShellQuote` (mismo patrón de todo el repo) — nunca se loggea en las líneas que van a `onLine`.
- Límite conocido v1: el PAT viaja en el comando `git clone` ejecutado vía SSH, por lo que aparece en claro si alguien lista procesos (`ps aux`) en esa ventana de tiempo en el manager. Mismo nivel de exposición que cualquier credencial pasada por línea de comando hoy en el repo (ej. passwords de BD en `docker exec ... -p<pass>`); no se resuelve aquí.

**Registro de builds en memoria** (nuevo paquete pequeño, p.ej. `srv/sys/usecases/build_registry.go` o junto al usecase): mapa `map[string]*BuildJob` con mutex, igual de simple que `sshclient.Pool`. `onLine` del usecase escribe ahí; un canal `done` se cierra al terminar.

**Endpoints nuevos** en `web_server.go`:
- `GET /api/builds/{id}/stream` (NDJSON, reusa `setupStreaming`/`streamJSON` ya existentes) — transmite las líneas ya acumuladas y las nuevas hasta que el job termina.

**Cambio en `handleWebhookDeploy`:**
- Después de validar la firma (ya arreglado) y resolver `svc`, si `svc.SourceType == "git"`: parsear el payload según el proveedor (header `X-GitHub-Event`/`X-Gitea-Event` o la forma del JSON) para sacar el commit SHA, crear un `BuildJob`, lanzar `BuildFromSourceUseCase.Execute` en una goroutine que al terminar dispara el redeploy existente, y responder `202 {"buildId": "..."}` de inmediato.
- Si `svc.SourceType != "git"`, comportamiento idéntico al de hoy (sin cambios).

**Botón manual "Rebuild & Deploy":** nuevo endpoint `POST /api/services/{name}/rebuild` que llama el mismo usecase de forma síncrona, con `onLine` escribiendo directo a la respuesta vía streaming (igual que `handleBootstrap` ya hace) — no pasa por el registro de builds en memoria porque no lo necesita (la conexión ya está abierta).

## Manejo de errores

Ninguna falla en clone/build/push toca el servicio que ya está corriendo — el redeploy solo se dispara si el build completo salió bien. Cada fallo (repo inválido, PAT sin permiso, branch inexistente, Dockerfile roto, push al registry caído) termina el `BuildJob` en `status=failed` con el error real en las líneas de log; no hay rollback automático del redeploy final porque hoy tampoco existe para ningún otro flujo de deploy.

## Testing

- `BuildFromSourceUseCase`: tabla de casos con `MockSSHExecutor` (mismo patrón de todo el repo) — clone falla, build falla, push falla, happy path con el tag correcto.
- Parser de payload por proveedor: tabla con payloads reales de GitHub/GitLab/Gitea fijados como fixtures.
- Registro de builds: test de concurrencia con `-race` (dos builds del mismo servicio no se pisan; IDs distintos no interfieren) — mismo rigor que se usó esta sesión para el pool de SSH.
- `validator.ShellQuote` en la URL de clone con PAT: test de inyección con el mismo patrón ya usado en todo el repo.
