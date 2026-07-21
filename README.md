# 🚀 Tarhiata-ops (Tu PaaS Privado)

![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go)
![Docker Swarm](https://img.shields.io/badge/Docker_Swarm-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Traefik](https://img.shields.io/badge/Traefik_v3-24A1C1?style=for-the-badge&logo=traefik&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![Status](https://img.shields.io/badge/Status-Production_Ready-success?style=for-the-badge)

**Tarhiata-ops** es una herramienta de terminal interactiva (TUI) diseñada para simplificar el despliegue de microservicios, bases de datos y la gestión de infraestructura remota. Funciona como tu propio **PaaS privado** (estilo Vercel, Heroku o Railway) pero ejecutándose 100% en tus propios servidores VPS.

Olvídate de editar interminables archivos YAML, entrar por SSH manualmente a configurar proxies o recordar largos comandos de Docker. Tarhiata gestiona todo mediante menús interactivos, inyecta variables de entorno y traza tu arquitectura de red en tiempo real.

---

## ✨ Características Principales

- **🎩 Interfaz TUI Interactiva:** Desarrollado con `Charmbracelet (huh)`. Todo es *Point-and-Click* desde tu teclado.
- **🛡️ Bootstrapper Seguro:** Aprovisiona un servidor virgen en minutos. Instala Docker v27, inicializa Swarm, configura UFW (Firewall) y despliega Traefik v3 automáticamente.
- **🔌 Inyección de Dependencias (Global Link):** Conecta microservicios entre sí con un asistente visual. El CLI construye dinámicamente las URLs internas (HTTP, gRPC, TCP, etc.) y las inyecta en variables de entorno (ej. `DATABASE_URL`).
- **🗺️ Mapa de Topología ANSI:** Visualiza las conexiones en vivo entre tus servicios, puertos y dominios públicos con un hermoso árbol generado dinámicamente leyendo los archivos `.env`.
- **🗄️ Catálogo de Bases de Datos:** Instala y asegura bases de datos (PostgreSQL, MongoDB, Redis, MySQL, MariaDB) con 1 clic en la red interna (overlay) sin exponerlas a internet.
- **📁 Sistema de Mounts:** Inyecta múltiples archivos de configuración locales (`config.json`, `certs`, etc.) directamente en los contenedores remotos.

---

## 🛠️ Instalación y Arranque

Asegúrate de tener **Go 1.21+** instalado en tu máquina local.

Puedes probar la herramienta directamente con:

```bash
go run cmd/tarhiata/main.go
```

O compilar el binario para usarlo globalmente desde cualquier parte de tu sistema:

```bash
go build -o bin/tarhiata cmd/tarhiata/main.go
sudo mv bin/tarhiata /usr/local/bin/tarhiata
```

---

## 📖 Guía Rápida de Uso

### 1. Configurar Servidor Destino
Abre el CLI y selecciona **"⚙️  Configurar Credenciales del Servidor"**. Provee la IP pública de tu VPS, tu usuario (ej. `root`) y la ruta a tu llave SSH. Todo se guarda localmente en tu máquina. Usa **"Probar Conexión"** para validar.

### 2. Inicializar el Servidor (Bootstrapper)
Selecciona **"🚀 Inicializar Servidor Virgen (Instalar Docker/Swarm)"**. 
Tarhiata se conectará por SSH e instalará y blindará el entorno. Al terminar, tu servidor estará corriendo Traefik y listo para recibir peticiones web de forma segura con auto-renovación de SSL.

### 3. Crear Bases de Datos
Si tu proyecto requiere almacenamiento, ve a **"🗄️  Gestionar Bases de Datos"**. Despliega un motor y el CLI garantizará que nadie fuera del clúster pueda acceder a ella.

### 4. Desplegar Microservicios
Ve a **"📦 Desplegar o Administrar Servicios"** > **"➕ Agregar Servicio al Catálogo"**.
Ponle un nombre, indica en qué puerto corre tu código y elige la imagen de Docker. Si decides exponerlo, puedes asignarle un dominio (ej. `api.midominio.com`) y el CLI generará el certificado SSL automáticamente.

### 5. Vincular Servicios Mágicamente 🔗
Si tu API necesita hablar con la Base de Datos:
1. En el menú de Servicios, elige **"🔗 Vincular Servicios Rápidamente"**.
2. Selecciona quién recibe la conexión y quién es el destino.
3. Escribe el nombre de la variable (ej. `DB_URI`).
4. Entra a administrar tu servicio y dale a **"🚀 Desplegar / Actualizar ahora"**. Tarhiata se encarga de inyectar el secreto por debajo.

### 6. Mapa Topológico de Red 🗺️
Entra a **"🗺️  Ver Mapa de Interconexión (URLs)"** para ver una obra de arte en tu terminal:

```text
🚀 SERVICIO: backend-api
 ├─ 🔌 DNS Interno : http://backend-api:3000 (Visible en Swarm)
 ├─ 🌐 Red Pública : https://api.midominio.com
 ├─ 📁 Mounts      : 1 archivos inyectados
 └─ 📝 Variables   : /Users/diego/.env
    │  └─ DATABASE_URL=postgres://admin:password@mi-postgres:5432/db

🗄️  BASE DE DATOS: mi-postgres (postgres)
 ├─ 🔌 DNS Interno : postgres://admin:password@mi-postgres:5432/db
 └─ 🔒 Red Pública : [ACCESO DENEGADO - Seguro por defecto]

========================================================
      🕸️   GRAFO DE DEPENDENCIAS (Interconexiones)    
========================================================

 [backend-api] ────(DATABASE_URL)────▶ [mi-postgres (BD)]
```

---

## 🔐 Arquitectura y Seguridad

* **Estado Offline:** Todo tu catálogo de servicios, variables y configuraciones se guarda **exclusivamente local** en una base de datos SQLite en `~/.config/tarhiata/config.db`.
* **Cero Telemetría:** No hay servidores intermediarios. Tu terminal se comunica 1 a 1 vía cifrado SSH directamente con tu VPS.
* **Seguridad por Defecto:** Las bases de datos nunca se exponen al host público. Todos los contenedores usan `overlay networks` de Swarm, y el puerto público 80/443 está exclusivamente mediado por Traefik.
* **Protección Anti-Crash:** Los `locks` de instalación en Linux (ej. `dpkg` trancado) se auto-resuelven durante el Bootstrapping.

---
*Hecho con ❤️ para dominar la infraestructura desde la terminal.*
