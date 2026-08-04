> 🇪🇸 **¿Hablas Español?** [Haz click aquí para leer la documentación en Español](docs/es/README.md)

# ⚡ Tarhiata-Ops (100% Stateless & Local-First Private PaaS)

![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go)
![Docker Swarm](https://img.shields.io/badge/Docker_Swarm-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Traefik](https://img.shields.io/badge/Traefik_v3-24A1C1?style=for-the-badge&logo=traefik&logoColor=white)
![Vultr](https://img.shields.io/badge/Vultr_API_v2-007BFF?style=for-the-badge&logo=vultr&logoColor=white)
![Version](https://img.shields.io/badge/Version-v1.0.0--beta_(Beta)-orange?style=for-the-badge)
![Made in Mexico](https://img.shields.io/badge/Made_in-M%C3%A9xico_%F0%9F%87%B2%F0%9F%87%BD-006847?style=for-the-badge)

**Tarhiata-Ops** is a zero-overhead, local-first private PaaS (Platform as a Service) control plane designed to deploy, orchestrate, and observe containerized microservices, databases, and multi-node clusters on your own VPS host with **0 MB RAM overhead**.

Experience the speed, aesthetics, and developer UX of Vercel and Railway, but running 100% on your infrastructure with complete data sovereignty and zero SaaS monthly fees.

<p align="center">
  <br><br>
  <img width="1097" height="709" alt="Screenshot 2026-08-03 at 23 06 38" src="https://github.com/user-attachments/assets/4ef5732d-b679-446a-9559-769e08de9d14" />
  <br><br>
  <img width="1450" height="851" alt="Screenshot 2026-08-03 at 22 46 19" src="https://github.com/user-attachments/assets/fae05291-157c-4f12-adac-1d6921457b7a" />

</p>

---

## 📥 Pre-Compiled Downloads (Beta Release)

Download the pre-compiled executable binary directly for your operating system without compiling:

- 🍏 **macOS (Apple Silicon M1/M2/M3/M4):** [`tarhiata-darwin-arm64`](https://github.com/Dall06/tarhiata-ops/releases/latest/download/tarhiata-darwin-arm64)
- 🍎 **macOS (Intel):** [`tarhiata-darwin-amd64`](https://github.com/Dall06/tarhiata-ops/releases/latest/download/tarhiata-darwin-amd64)
- 🐧 **Linux (x86_64 / amd64):** [`tarhiata-linux-amd64`](https://github.com/Dall06/tarhiata-ops/releases/latest/download/tarhiata-linux-amd64)
- 🐧 **Linux (ARM64 / Raspberry Pi):** [`tarhiata-linux-arm64`](https://github.com/Dall06/tarhiata-ops/releases/latest/download/tarhiata-linux-arm64)
- 🪟 **Windows (x64):** [`tarhiata-windows-amd64.exe`](https://github.com/Dall06/tarhiata-ops/releases/latest/download/tarhiata-windows-amd64.exe)

---

## 🏛️ Architectural Philosophy

1. **100% Stateless VPS Host (Zero Overhead):**
   * Unlike agent-heavy PaaS control planes that consume 500 MB – 2 GB of RAM on your server, Tarhiata-Ops runs **100% locally on your workstation**.
   * 100% of your VPS RAM and CPU are strictly reserved for your applications and databases.
2. **Local-First & Direct SSH Control:**
   * No intermediary SaaS servers, no telemetry, no exposed inbound webhooks listening to the public internet.
   * Communication is executed directly via encrypted SSH sessions.
3. **Multi-PC Resilient State Synchronization:**
   * Infrastructure topology state is automatically exported to non-sensitive state references (`/opt/tarhiata/state.json`) on your VPS.
   * Connecting from a new PC automatically imports the entire cluster state instantly.
4. **Zero-Config Buildpacks & Docker Auto-Detection:**
   * Auto-detects `Dockerfile`, `package.json`, `go.mod`, or `requirements.txt` to build and deploy applications without configuration friction.

---

## ✨ Key Features

- **🌐 Vercel/Railway Aesthetic Web Control Plane (`http://localhost:8080`):**
  - Spotlight Command Palette (`⌘K` / `Ctrl+K`) for fast keyboard-driven navigation.
  - Real-time SVG topology graph, cgroups telemetry metrics, live container logs, and interactive SSH terminal.
- **🗄️ 1-Click Database & Object Storage Engines:**
  - Instant provisioning for **PostgreSQL**, **MongoDB**, **MySQL**, **Redis**, and **MinIO (S3 Object Storage)**.
  - Recovery Mode (`/opt/data/db-*` host volume retention) prevents accidental data loss during redeployments.
- **🚀 Zero-Downtime Rolling Updates:**
  - Automated Traefik v3 HTTPS SSL certificate management (Let's Encrypt / ZeroSSL).
  - Swarm `start-first` rolling updates with automatic rollback on deployment failures.
- **🔑 Developer SSH Key Access Control:**
  - Real-time team key management (`tarhiata ssh-key`). Add and revoke team developer SSH access instantly with **Vultr Master Account Protection** (blocking accidental deletion of primary server keys).
- **🏗️ Multi-Node Cluster Scaling via Vultr API v2:**
  - Automated Terraform provisioning of dedicated worker nodes across regions, defaulted to **Mexico City (`mex`)**.
- **📦 Custom S3 Backup Exporter:**
  - Backup databases and volumes to local VPS storage (`/opt/tarhiata/backups`) or stream them directly to **MinIO S3** or external S3 providers (**Cloudflare R2**, **AWS S3**, **DigitalOcean Spaces**, **Wasabi**).
- **📜 NDJSON File Audit Logger:**
  - Immutable local security audit trail tracking all deployment, linking, node, and configuration events.

---

## ☁️ Supported Cloud Infrastructure Providers

- [x] 🟢 **Vultr** *(100% Fully Covered & Supported - API v2, Mexico `mex` region default & automated plan provisioning)*
- [ ] ⏳ **DigitalOcean** *(Planned / Roadmap)*
- [ ] ⏳ **Hetzner Cloud** *(Planned / Roadmap)*
- [ ] ⏳ **AWS EC2** *(Planned / Roadmap)*
- [ ] ⏳ **Linode / Akamai** *(Planned / Roadmap)*

> ℹ️ **Note:** Currently, **Vultr** is the primary 100% covered provider for automated cloud worker node provisioning and live plan selection. Support for additional cloud providers will be rolled out in upcoming releases!

---

## 🖥️ Self-Hosted Existing VPS Support vs. Cloud Automation

Tarhiata-Ops works **100% with any existing VPS host or bare-metal server** (Ubuntu/Debian) that you have already provisioned on any provider!

| Feature / Capability | Existing Custom VPS (Any Provider / Self-Hosted) | Automated Cloud Integration (Vultr API) |
| :--- | :---: | :---: |
| **0-Downtime App Deployments (`tarhiata deploy`)** | ✅ 100% Supported | ✅ 100% Supported |
| **1-Click Databases & MinIO S3 (`db create`)** | ✅ 100% Supported | ✅ 100% Supported |
| **SSL HTTPS & Traefik v3 Routing** | ✅ 100% Supported | ✅ 100% Supported |
| **Env Vars, Link/Unlink, Volume Explorer** | ✅ 100% Supported | ✅ 100% Supported |
| **SSH Key Management & Audit Logs** | ✅ 100% Supported | ✅ 100% Supported |
| **Automated Worker Node Provisioning (`worker add`)** | ⚠️ Fallback to Single-Node / Manager | ✅ Automated via Terraform |
| **Live Cloud VM Plan Selection & Scaling** | ⚠️ Managed manually at your Provider | ✅ Automated via API v2 |

> 💡 **Standalone VPS Mode:** If you supply an existing VPS without a Cloud API Key, Tarhiata-Ops automatically operates in **Manager-Anchored Mode**—giving you full control over app deployments, databases, SSL routing, and backups without requiring cloud API permissions!

---

## 🛠️ Quickstart & Setup

### Prerequisites
- **Go 1.25+** installed on your workstation.
- A fresh VPS (Ubuntu 22.04 / 24.04 recommended) with SSH root access.

### Running Locally
```bash
# Clone the repository
git clone https://github.com/Dall06/tarhiata-ops.git
cd tarhiata-ops

# Launch the Web Dashboard (Spotlight ⌘K enabled)
go run cmd/tarhiata/main.go
# Automatically opens http://localhost:8080
```

### Installing Globally
```bash
go build -o bin/tarhiata cmd/tarhiata/main.go
sudo mv bin/tarhiata /usr/local/bin/tarhiata
```

---

## 📖 CLI Commands Reference

```text
Tarhiata-Ops PaaS - CLI & Control Plane

Usage:
  tarhiata [command] [options]

Commands:
  (no command)      Starts the Web Dashboard at http://localhost:8080 (⌘K Spotlight)
  dashboard | ui     Starts the Web Dashboard at http://localhost:8080
  config set         Configure SSH credentials and Vultr API Token Key
  init | bootstrap   Execute InitServerUseCase (Docker Swarm + Traefik HTTPS + Fail2Ban)
  deploy             Deploy an app service in Swarm with SSL and Traefik
  preview            Manage ephemeral preview environments (create/list/destroy)
  db create          Deploy a database (PostgreSQL, Mongo, MySQL, Redis, MinIO S3)
  backup             Create, list, or restore snapshots of DBs & Volumes
  env                Bulk manage environment variables (.env) & secrets
  volume             Persistent volume file explorer for /opt/data (ls/cat/rm)
  link               Connect 2 services and inject target DNS URI into an Env Var
  unlink             Remove service link and clear target Env Var
  node               Manage Docker Swarm nodes (ls/token/add/rm/update)
  worker add         Provision a new worker node in Vultr via Terraform
  master             Initialize 1-Click All-in-One service stack
  rollback           Rollback a Docker Swarm service to its previous version
  obs deploy         Deploy observability stack (Portainer, Loki, Grafana)
  ssh-key            Manage team developer SSH keys (ls/add/rm with Vultr protection)
  update             Update host OS packages and Docker daemon
  list               List all registered services and databases
  status             Show host and cluster health status
  topology           Display dependency graph and internal DNS routes
  prune              Clean unused images and volumes on the VPS host
```

---

## 🔐 Security & Production Best Practices

- **Zero Exposed Database Ports:** Databases run strictly inside Swarm `overlay networks` (`tarhiata_internal`) and are unreachable from the public internet.
- **Fail2Ban & UFW Integration:** Automatically configured on `tarhiata init` to block brute-force SSH attacks and restrict host ports.
- **Local SQLite Storage:** Local state is saved in `~/.config/tarhiata/config.db`.
- **Master Key Security:** Master Vultr API keys and primary SSH credentials cannot be removed via API or CLI.

---

*Built with ❤️ to manage infrastructure with zero overhead directly from your terminal and web dashboard.*
