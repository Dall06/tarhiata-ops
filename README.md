> 🇪🇸 **¿Hablas Español?** [Haz click aquí para leer la documentación en Español](docs/es/README.md)

# 🚀 Tarhiata-ops (Your Private PaaS)

![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go)
![Docker Swarm](https://img.shields.io/badge/Docker_Swarm-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Traefik](https://img.shields.io/badge/Traefik_v3-24A1C1?style=for-the-badge&logo=traefik&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![Status](https://img.shields.io/badge/Status-Production_Ready-success?style=for-the-badge)

**Tarhiata-ops** is an interactive Terminal User Interface (TUI) tool designed to simplify the deployment of microservices, databases, and remote infrastructure management. It works as your own **private PaaS** (similar to Vercel, Heroku, or Railway) but runs 100% on your own VPS servers.

Forget about editing endless YAML files, manually logging via SSH to configure proxies, or memorizing long Docker commands. Tarhiata manages everything through interactive menus, injects environment variables, and maps your network architecture in real-time.

---

## ✨ Key Features

- **🎩 Interactive TUI Interface:** Built with `Charmbracelet (huh)`. Everything is *Point-and-Click* straight from your keyboard.
- **🛡️ Secure Bootstrapper:** Provision a fresh server in minutes. It installs Docker v27, initializes Swarm, secures your server with UFW and Fail2Ban, and deploys Traefik v3 automatically.
- **🔌 Dependency Injection (Global Link):** Connect microservices with a visual wizard. The CLI dynamically builds internal URLs (HTTP, gRPC, TCP, etc.) and injects them into environment variables (e.g. `DATABASE_URL`).
- **🗺️ ANSI Topology Map:** Visualize live connections between your services, ports, and public domains with a beautiful dynamically generated tree by reading your `.env` files.
- **🗄️ Database Catalog:** Install and secure databases (PostgreSQL, MongoDB, Redis, MySQL, MariaDB) with 1 click in the internal network (overlay) without exposing them to the internet.
- **📁 Mount System:** Inject multiple local configuration files (`config.json`, `certs`, etc.) directly into remote containers.

---

## 🛠️ Installation & Setup

Make sure you have **Go 1.21+** installed on your local machine.

You can test the tool directly with:

```bash
go run cmd/tarhiata/main.go
```

Or compile the binary to use it globally from anywhere in your system:

```bash
go build -o bin/tarhiata cmd/tarhiata/main.go
sudo mv bin/tarhiata /usr/local/bin/tarhiata
```

---

## 📖 Quick Start Guide

### 1. Configure Target Server
Open the CLI and select **"⚙️ Configure Server Credentials"**. Provide your VPS public IP, your username (e.g. `root`) and the path to your SSH key. Everything is saved locally on your machine. Use **"Test Connection"** to validate.

### 2. Initialize Server (Bootstrapper)
Select **"🚀 Initialize Virgin Server (Install Docker/Swarm)"**. 
Tarhiata will connect via SSH, install, and secure the environment. Upon finishing, your server will be running Traefik and will be ready to securely route web requests with auto-renewing SSL.

### 3. Create Databases
If your project requires storage, go to **"🗄️ Manage Databases"**. Deploy an engine and the CLI will guarantee that no one outside the cluster can access it.

### 4. Deploy Microservices
Go to **"📦 Deploy or Manage Services"** > **"➕ Add Service to Catalog"**.
Give it a name, indicate on which port your code runs, and choose the Docker image. If you decide to expose it, you can assign it a domain (e.g. `api.mydomain.com`) and the CLI will generate the SSL certificate automatically.

### 5. Link Services Magically 🔗
If your API needs to talk to the Database:
1. In the Services menu, choose **"🔗 Quickly Link Services"**.
2. Select who receives the connection and who is the destination.
3. Type the variable name (e.g. `DB_URI`).
4. Go to manage your service and hit **"🚀 Deploy / Update now"**. Tarhiata will take care of injecting the secret under the hood.

### 6. Network Topology Map 🗺️
Enter **"🗺️ View Interconnection Map (URLs)"** to see a piece of art in your terminal:

```text
🚀 SERVICE: backend-api
 ├─ 🔌 Internal DNS : http://backend-api:3000 (Visible in Swarm)
 ├─ 🌐 Public Net   : https://api.mydomain.com
 ├─ 📁 Mounts       : 1 injected files
 └─ 📝 Variables    : /Users/diego/.env
    │  └─ DATABASE_URL=postgres://admin:password@my-postgres:5432/db

🗄️  DATABASE: my-postgres (postgres)
 ├─ 🔌 Internal DNS : postgres://admin:password@my-postgres:5432/db
 └─ 🔒 Public Net   : [ACCESS DENIED - Secure by default]

========================================================
      🕸️   DEPENDENCY GRAPH (Interconnections)    
========================================================

 [backend-api] ────(DATABASE_URL)────▶ [my-postgres (DB)]
```

---

## 🔐 Architecture and Security

* **Offline State:** Your entire catalog of services, variables, and configurations is saved **exclusively locally** in an SQLite database at `~/.config/tarhiata/config.db`.
* **Zero Telemetry:** There are no intermediary servers. Your terminal communicates 1-to-1 via SSH encryption directly with your VPS.
* **Secure by Default:** Databases are never exposed to the public host. All containers use Swarm `overlay networks`, and public ports 80/443 are exclusively mediated by Traefik. Password login via SSH is blocked and Fail2Ban is built-in.
* **Anti-Crash Protection:** Linux installation `locks` (e.g. stuck `dpkg`) are self-resolved during Bootstrapping.

---
*Made with ❤️ to rule infrastructure from the terminal.*
