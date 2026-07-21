# Tarhiata-ops: Planeación y Arquitectura

## 1. Visión General del Producto
Tarhiata-ops es una herramienta de terminal interactiva (TUI) diseñada para facilitar despliegues y operaciones de infraestructura. Asume que la seguridad de red primaria ya está resuelta (el usuario accede a través de una VPN o SSH).

El objetivo es centralizar y simplificar las tareas de operaciones mediante menús interactivos, evitando que el usuario tenga que memorizar comandos o editar archivos de configuración crudos manualmente.

## 2. Flujo de Usuario (Experiencia TUI)
1. **Arranque:** Ejecuta en la terminal el comando `tarhiata`.
2. **Interfaz Principal:** Se despliega el menú interactivo inicial con las opciones:
   - **⚙️ Configuración / Credenciales:** El usuario ingresa aquí primero para configurar la IP del servidor destino, su usuario y la ruta de su llave SSH (`~/.ssh/id_rsa`). Incluye una opción para **Validar la Conexión SSH** en tiempo real.
   - **🚀 Bootstrapper (Swarm Init y Seguridad):** Se conecta al servidor virgen, instala Docker y ejecuta `docker swarm init`. Además, configura el Firewall (`ufw`) bloqueando todos los puertos excepto SSH (Regla de oro de seguridad).
   - **📦 Desplegar y Configurar Servicios:**
     - **Provisión de Imagen:** El usuario indica el origen. Si es de **Docker Hub**, el CLI inyecta las credenciales (guardadas en SQLite) y hace `docker login` remotamente. Si es una **URL (zip/rar/tar)**, el CLI manda comandos al servidor para descargarla (`wget`), descomprimirla y cargarla (`docker load`).
     - **Inyección de Configuración:** Te permite ajustar Variables de Entorno y adjuntar archivos extra (`.env`, `secrets.json`). Si no los tienes localmente, abre `nano/vim` para crearlos y los transfiere al servidor.
     - **Exposición y Dominios (SSL):** Al desplegar, el usuario decide si el servicio debe ser público. Si es así, el CLI abre el puerto necesario en el Firewall. Opcionalmente, permite vincular un dominio y provisionar certificados SSL (ej. vía Traefik/Certbot) para conexiones seguras.
     - **Inyección Automática de Dependencias:** A través del menú principal, un asistente global permite vincular servicios entre sí. El CLI construye dinámicamente la URL interna (HTTP, gRPC, TCP, etc.) y la inyecta como variable de entorno.
     - **Mapa Topológico y Grafo Visual:** Un visor de mapa de red integrado permite visualizar interconexiones entre contenedores en tiempo real leyendo sus variables de entorno mediante un diseño visual TUI.
     - **Ejecución:** Corre `docker stack deploy` con la configuración final.
   - **📊 Estado del Servidor:** Ver recursos o contenedores corriendo a través del túnel SSH.

## 3. Componentes Clave
- **Motor Interactivo:** Implementado con un framework que soporte TUI, de modo que la herramienta sea "point and click" (con teclado) en la terminal.
- **Gestor de Credenciales:** Un módulo (`srv/tarhiata/usecases/credentials` o similar) encargado de cifrar y almacenar de forma segura las credenciales de los distintos servicios.
- **Ejecutor de Tareas (Ops):** El encargado de leer las intenciones del usuario desde el menú y traducirlas a operaciones reales.

## 4. Decisiones Técnicas (MVP)
- **Sistema Operativo Destino:** El bootstrapper se enfocará exclusivamente en **Ubuntu/Debian** para la versión 1.0 (usando `apt`).
- **Almacenamiento Local:** Se utilizará **SQLite** localmente para guardar la configuración (ej. ruta de la llave SSH). Cualquier contraseña o token sensible en la base de datos deberá ser cifrado antes de guardarse.
- **Monitoreo de Conexión (Async):** Se utilizarán `goroutines` y `channels` para mantener un chequeo en segundo plano de la conexión SSH, permitiendo que la TUI muestre un indicador de estado en tiempo real (conectado/desconectado) sin bloquear la interfaz.
- **Framework TUI:** Se usará el ecosistema de Charmbracelet (`huh` / `bubbletea`) para manejar los estados asíncronos y los formularios complejos.
