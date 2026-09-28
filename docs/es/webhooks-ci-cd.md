# 🚀 Auto-Despliegue Continuo con Git Webhooks (CI/CD)

Tarhiata-Ops incluye un endpoint receptor de webhooks nativo y seguro, diseñado para habilitar despliegues automáticos continuos cuando se realiza un `git push` o cuando un pipeline de CI/CD compila una nueva imagen Docker.

---

## 📋 Visión General y Arquitectura

```
                                      ┌────────────────────────────────────────┐
                                      │              Tarhiata-Ops              │
                                      │           http://localhost:8080        │
                                      │                                        │
  [ GitHub / GitLab / Gitea ]         │  POST /api/webhooks/deploy             │
  [ Docker Hub / GitHub Actions ] ───►│  ├── 1. Valida Secreto HMAC SHA-256    │
                                      │  ├── 2. Resuelve Servicio e Imagen     │
                                      │  ├── 3. Ejecuta Swarm Rolling Update   │
                                      │  ├── 4. Guarda Snapshot en Historial   │
                                      │  └── 5. Registra en Audit Log          │
                                      └────────────────────────────────────────┘
```

---

## 🔒 Seguridad: Verificación HMAC SHA-256

Para evitar peticiones no autorizadas, Tarhiata-Ops soporta verificación criptográfica mediante la cabecera estándar `HMAC-SHA256`:

* **Cabecera:** `X-Hub-Signature-256: sha256=<hash_hexadecimal>`
* **Mecanismo:** El cuerpo en crudo de la petición HTTP se procesa con tu clave secreta usando HMAC-SHA256. Tarhiata evalúa el hash en tiempo constante (`hmac.Equal`) para evitar ataques de temporización (*timing attacks*).
* Si se define un token en la query (`?token=secreto`) y la cabecera de firma está presente, cualquier firma inválida es rechazada de inmediato con código `401 Unauthorized`.

---

## 📡 Especificación del Endpoint

### `POST /api/webhooks/deploy`

#### Parámetros en la URL (Query String)

| Parámetro | Tipo | Requerido | Descripción |
| :--- | :---: | :---: | :--- |
| `service` | `string` | **Sí** | Nombre del servicio en Tarhiata / Docker Swarm (ej. `mi-backend`). |
| `token` | `string` | Opcional | Secreto compartido para validación HMAC o token de acceso. |
| `image` | `string` | Opcional | Tag de la imagen Docker a desplegar (ej. `ghcr.io/org/repo:v1.2.0`). Si se omite, usa la imagen guardada en catálogo o el tag `latest`. |

#### Cuerpo JSON Opcional (Body)
También puedes enviar los datos en el cuerpo JSON de la petición:

```json
{
  "service": "mi-backend",
  "image": "ghcr.io/mi-org/mi-backend:v1.2.0"
}
```

#### Respuesta Exitosa (`200 OK`)
```json
{
  "status": "deployed",
  "record": {
    "id": 12,
    "serviceName": "mi-backend",
    "imageTag": "ghcr.io/mi-org/mi-backend:v1.2.0",
    "port": 8080,
    "domain": "api.tudominio.com",
    "expose": true,
    "deployedAt": "2026-09-28T15:20:00Z",
    "status": "success"
  },
  "message": "Servicio 'mi-backend' actualizado con éxito a la imagen ghcr.io/mi-org/mi-backend:v1.2.0"
}
```

---

## 🛠️ Guías de Configuración

### 1. Webhook Directo en Repositorio GitHub

1. Entra a tu repositorio en GitHub → **Settings** → **Webhooks** → **Add webhook**.
2. Configura la **Payload URL**:
   ```
   https://tarhiata.tudominio.com/api/webhooks/deploy?service=mi-backend&token=MI_SECRETO_SEGURO
   ```
3. **Content type**: `application/json`.
4. **Secret**: `MI_SECRETO_SEGURO`.
5. Eventos: Selecciona **Pushes** o **Releases**.
6. Haz clic en **Add webhook**.

---

### 2. Flujo Completo con GitHub Actions (CI/CD)

A continuación un pipeline completo de GitHub Actions que compila la imagen Docker, la publica en GitHub Container Registry (GHCR) y notifica a Tarhiata para desplegar:

```yaml
name: Build & Auto-Deploy

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout del código
        uses: actions/checkout@v4

      - name: Iniciar sesión en GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Compilar y publicar imagen Docker
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}

      - name: Notificar a Tarhiata para Despliegue
        run: |
          curl -f -X POST "https://tarhiata.tudominio.com/api/webhooks/deploy?service=mi-backend&token=${{ secrets.TARHIATA_WEBHOOK_SECRET }}&image=ghcr.io/${{ github.repository }}:${{ github.sha }}"
```

---

### 3. Pipeline en GitLab CI/CD

En tu `.gitlab-ci.yml`:

```yaml
stages:
  - build
  - deploy

deploy_to_swarm:
  stage: deploy
  only:
    - main
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - 'curl -f -X POST "https://tarhiata.tudominio.com/api/webhooks/deploy?service=mi-backend&token=$TARHIATA_WEBHOOK_SECRET&image=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA"'
```

---

### 4. Disparo Manual con cURL

```bash
curl -X POST "https://tarhiata.tudominio.com/api/webhooks/deploy?service=mi-backend&token=clave_secreta&image=nginx:alpine"
```

---

## ⚡ Características Clave y Resiliencia

* **Rolling Update Zero-Downtime:** Ejecuta `docker service update --image ... --force` en el clúster Swarm, reemplazando tareas de manera progresiva sin cortes de servicio.
* **Rollback Instantáneo Disponible:** Cada auto-despliegue guarda automáticamente un registro en la base de datos `deployment_history`, permitiendo restaurar versiones anteriores en 1 clic desde el Web Studio.
* **Trazabilidad y Auditoría:** Registra un evento de tipo `DEPLOY` en `/api/audit-logs` con el tag de la imagen, el servicio y la marca de tiempo exacta.
