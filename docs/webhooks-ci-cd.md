# 🚀 Continuous Git Webhook Auto-Deploy (CI/CD)

Tarhiata-Ops includes a built-in, secure webhook listener endpoint designed to enable automatic deployments whenever code is pushed to your Git repository or when your CI/CD pipeline builds a new Docker image.

---

## 📋 Overview & Architecture

```
                                      ┌────────────────────────────────────────┐
                                      │              Tarhiata-Ops              │
                                      │           http://localhost:8080        │
                                      │                                        │
  [ GitHub / GitLab / Gitea ]         │  POST /api/webhooks/deploy             │
  [ Docker Hub / GitHub Actions ] ───►│  ├── 1. Validate HMAC SHA-256 Secret   │
                                      │  ├── 2. Resolve Service & Image Target │
                                      │  ├── 3. Execute Swarm Rolling Update   │
                                      │  ├── 4. Record Snapshot in History     │
                                      │  └── 5. Write to Immutable Audit Log   │
                                      └────────────────────────────────────────┘
```

---

## 🔒 Security: HMAC SHA-256 Verification

To prevent unauthorized triggers, Tarhiata-Ops supports cryptographic signature verification using the industry-standard `HMAC-SHA256` digest:

* **Header:** `X-Hub-Signature-256: sha256=<hex_digest>`
* **Mechanism:** The raw request body is hashed with your secret key using HMAC-SHA256. Tarhiata validates the hash using constant-time comparison (`hmac.Equal`) to mitigate timing attacks.
* If a token is provided in the query string (`?token=secret`) and the signature header is present, invalid signatures are rejected immediately with `401 Unauthorized`.

---

## 📡 Endpoint Specification

### `POST /api/webhooks/deploy`

#### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :---: | :---: | :--- |
| `service` | `string` | **Yes** | The name of the service in Tarhiata / Docker Swarm (e.g., `my-backend`). |
| `token` | `string` | Optional | Shared secret key for HMAC SHA-256 or token validation. |
| `image` | `string` | Optional | Docker image tag to deploy (e.g., `ghcr.io/org/repo:v1.2.0`). If omitted, uses the currently configured image in the catalog or latest tag. |

#### Request Body (Optional JSON)
You can also pass parameters inside the JSON payload body:

```json
{
  "service": "my-backend",
  "image": "ghcr.io/my-org/my-backend:v1.2.0"
}
```

#### Successful Response (`200 OK`)
```json
{
  "status": "deployed",
  "record": {
    "id": 12,
    "serviceName": "my-backend",
    "imageTag": "ghcr.io/my-org/my-backend:v1.2.0",
    "port": 8080,
    "domain": "api.example.com",
    "expose": true,
    "deployedAt": "2026-09-28T15:20:00Z",
    "status": "success"
  },
  "message": "Servicio 'my-backend' actualizado con éxito a la imagen ghcr.io/my-org/my-backend:v1.2.0"
}
```

---

## 🛠️ Configuration Guides

### 1. GitHub Repository Webhook

1. Navigate to your GitHub repository → **Settings** → **Webhooks** → **Add webhook**.
2. Set **Payload URL**:
   ```
   https://tarhiata.yourdomain.com/api/webhooks/deploy?service=my-backend&token=YOUR_WEBHOOK_SECRET
   ```
3. Set **Content type**: `application/json`.
4. Set **Secret**: `YOUR_WEBHOOK_SECRET`.
5. Select event triggers: **Pushes** or **Releases**.
6. Click **Add webhook**.

---

### 2. GitHub Actions Workflow

Below is a complete GitHub Actions CI/CD pipeline that builds a Docker container, pushes it to GitHub Container Registry (GHCR), and notifies Tarhiata to deploy:

```yaml
name: Build & Auto-Deploy

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}

      - name: Trigger Tarhiata Auto-Deploy
        run: |
          curl -f -X POST "https://tarhiata.yourdomain.com/api/webhooks/deploy?service=my-backend&token=${{ secrets.TARHIATA_WEBHOOK_SECRET }}&image=ghcr.io/${{ github.repository }}:${{ github.sha }}"
```

---

### 3. GitLab CI/CD Pipeline

In your `.gitlab-ci.yml`:

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
    - 'curl -f -X POST "https://tarhiata.yourdomain.com/api/webhooks/deploy?service=my-backend&token=$TARHIATA_WEBHOOK_SECRET&image=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA"'
```

---

### 4. Manual / cURL Deployment

```bash
curl -X POST "https://tarhiata.yourdomain.com/api/webhooks/deploy?service=my-backend&token=my_secret_key&image=nginx:alpine"
```

---

## ⚡ Key Features & Resiliency

* **Zero-Downtime Rolling Update:** Executes `docker service update --image ... --force` in Swarm, replacing old tasks gradually without downtime.
* **Instant Rollback Availability:** Every webhook deployment automatically saves a snapshot in SQLite (`deployment_history`), allowing you to revert to any previous version with 1-click in the UI Studio.
* **Audit Trail:** Logs a `DEPLOY` event in `/api/audit-logs` recording the deployed image tag, target service, and timestamp.
