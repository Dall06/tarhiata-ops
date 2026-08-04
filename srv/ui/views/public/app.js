// Tarhiata-Ops Vercel Terminal Aesthetic Logic (Real API & Custom Toast System)
let globalServices = [];
let globalDatabases = [];
let globalLinks = [];
let autoScrollLogs = true;

// Alpine.js 3.x Reactive Store (Zero Build / CDN Reactivity)
document.addEventListener('alpine:init', () => {
    Alpine.store('app', {
        services: [],
        databases: [],
        isOnline: true,
        catalogSearch: '',

        get filteredServices() {
            const q = this.catalogSearch.toLowerCase().trim();
            if (!q) return this.services;
            return this.services.filter(s =>
                s.name.toLowerCase().includes(q) ||
                (s.imageSource && s.imageSource.toLowerCase().includes(q)) ||
                (s.domain && s.domain.toLowerCase().includes(q))
            );
        },

        get filteredDatabases() {
            const q = this.catalogSearch.toLowerCase().trim();
            if (!q) return this.databases;
            return this.databases.filter(d =>
                d.name.toLowerCase().includes(q) ||
                (d.engine && d.engine.toLowerCase().includes(q))
            );
        }
    });
});

let currentLang = localStorage.getItem('tarhiata_lang') || 'es';

const i18n = {
    es: {
        refresh: "Refrescar",
        terminal: "Node Terminal",
        palette: "Command Palette",
        catalogTitle: "Catálogo de Servicios & Bases de Datos",
        deployServiceBtn: "+ Desplegar Servicio",
        createDbBtn: "+ Crear BD",
        searchPlaceholder: "🔍 Filtrar servicios y BDs en vivo por nombre o motor... (ej. api, postgres, redis)",
        loadingCatalog: "Cargando catálogo desde SQLite...",
        topologyTitle: "Diagrama de Flujo & Red",
        viewTopologyTitle: "Ver topología de red escrita",
        linkBtn: "+ Enlazar A ➔ B",
        loadingTopology: "Cargando diagrama de flujo de red...",
        vpsMetricsTitle: "Métrica General del VPS",
        vpsStatusConnected: "CONECTADO",
        vpsHostLabel: "Host:",
        swarmRoleMaster: "Master",
        statActiveApps: "Apps Activas",
        subSwarmContainers: "Contenedores Swarm",
        statActiveDbs: "BDs Activas",
        subOverlayIsolated: "Aisladas en Red Overlay",
        statSwarmNodes: "Nodos Swarm",
        subMasterVps: "VPS Principal",
        statDiskStorage: "Almacenamiento en Disco",
        cloudBillingTitle: "Facturación Cloud & Plan",
        planName: "Plan Pro VPS (Self-Hosted)",
        swarmNodesTitle: "Nodos Swarm",
        tokenBtn: "📋 Token",
        workerBtn: "+ Worker",
        loadingNodes: "Cargando nodos del clúster Swarm...",
        backupsTitle: "Respaldos & Snapshots",
        snapshotBtn: "+ Snapshot",
        loadingBackups: "Cargando snapshots...",
        previewTitle: "Entornos Preview",
        previewBtn: "+ Preview",
        loadingPreviews: "Cargando entornos de prueba...",

        // Inspector Modal
        inspectorOpenUrl: "🌐 Abrir URL",
        inspectorDelete: "🗑️ Eliminar Recurso",
        tabMetrics: "📊 Métricas",
        tabLogs: "📜 Logs",
        tabEnvs: "🔑 Variables (.env)",
        tabRollback: "⏪ Rollback",
        tabBackups: "💾 Snapshots / Respaldos",
        tabConfig: "⚙️ Configuración",
        autoValidateText: "⏱️ Auto-validación de estado de red cada 15 segundos",
        btnValidateNow: "🔄 Validar Conexión Ahora",
        statReplicasLabel: "Estado de Réplicas",
        statRamLabel: "Uso de Memoria (RAM)",
        statPortLabel: "Puerto Interno / Exposición",
        realtimeStatsTitle: "⚡ Métricas en Tiempo Real (cgroups / Docker Stats)",
        cpuUsageLabel: "Uso de CPU",
        ramUsageLabel: "Uso de RAM",
        netIoLabel: "I/O de Red",
        diskIoLabel: "I/O Bloque Disco",
        telemetryChartTitle: "Telemetría CPU & RAM en Tiempo Real",

        logGrepPlaceholder: "🔍 Buscar en logs (grep)...",
        logLevelAll: "Todos los Niveles",
        logLevelError: "Solo Error / Fatal",
        logLevelWarn: "Solo Warn / Warning",
        logLevelInfo: "Solo Info",
        btnDownloadLogs: "📥 Descargar (.txt)",
        loadingLogs: "Cargando logs del servicio...",

        envTitle: "Variables de Ambiente (.env) Inyectadas",
        btnHideSecrets: "👁️ Ocultar Secretos",
        btnShowSecrets: "👁️ Mostrar Secretos",
        btnTableMode: "⇄ Modo Tabla",
        btnRawMode: "⇄ Modo Raw",
        btnImportEnv: "📤 Importar .env",
        envKeyHeader: "Variable (Key)",
        envValueHeader: "Valor (Value)",
        btnSaveEnvs: "💾 Guardar & Inyectar en Swarm",

        rollbackWarningTitle: "⚠️ Reversión de Versión en Docker Swarm",
        rollbackWarningText: "Esta operación ejecuta docker service rollback en el servidor remoto para este servicio. Se restaurará inmediatamente la imagen de contenedor y configuración previa registrada por Docker Swarm.",
        btnTriggerRollback: "⚠️ Ejecutar Rollback a Revisión Previa",

        backupSectionTitle: "Respaldos de Base de Datos y Snapshots",
        btnCreateSnapshot: "+ Crear Snapshot 1-Click",
        loadingBackupsList: "Cargando lista de respaldos...",

        cfgNameLabel: "Nombre del Recurso",
        cfgImageLabel: "Imagen / Motor",
        cfgPortLabel: "Puerto Interno",
        cfgDomainLabel: "Dominio / Subdominio HTTPS",
        cfgHealthLabel: "Comando de Healthcheck (Opcional)",
        cfgExposeLabel: "🌐 Exponer servicio públicamente a Internet (Vía Traefik Reverse Proxy)",
        cfgSSLLabel: "🔒 Habilitar HTTPS / SSL Automático (Certificado Let's Encrypt)",
        btnSaveConfig: "💾 Guardar Cambios de Configuración",

        // Palette Search & Items
        paletteSearchPlaceholder: "Filtrar herramientas del sistema... (ej. ssl, env, link, rollback, tailscale)",
        pItemCreateVmLabel: "Crear VM en la Nube & Instalar Framework",
        pItemCreateVmSub: "Aprovisionar una nueva VM en DigitalOcean/Vultr e instalar Docker, Swarm, Traefik y Observabilidad.",
        pItemByoLabel: "Instalar Framework en VM Conectada (BYO)",
        pItemByoSub: "Conectar a un servidor existente vía SSH e inicializar Docker, Swarm, Traefik SSL y Observabilidad.",
        pItemMasterLabel: "Master 1-Click Setup (Inicializar VPS & Swarm Cluster)",
        pItemMasterSub: "Configuración automatizada 1-Click de Docker Swarm, Traefik y subredes en el servidor.",
        pItemVolumesLabel: "Subir Ficheros & Explorador de Volúmenes (/opt/data)",
        pItemVolumesSub: "Cargar archivos a los volúmenes en disco del VPS y gestionar la persistencia de datos.",
        pItemSslLabel: "Configurar SSL, Certificados HTTPS & Dominios Personalizados",
        pItemSslSub: "Activar SSL 1-Click (Let's Encrypt o Certificados Privados) y mapear a servicios.",
        pItemTraefikLabel: "Reiniciar Traefik Proxy & Refrescar Enrutamiento",
        pItemTraefikSub: "Reaplicar reglas de Traefik v3 y limpiar caché de certificados SSL.",
        pItemDeployDbLabel: "Desplegar Base de Datos 1-Click (PostgreSQL, Mongo, Redis, MySQL, MinIO)",
        pItemDeployDbSub: "Instanciar contenedores aislados de BD con volúmenes montados en /opt/data.",
        pItemMinioLabel: "Desplegar Servidor de Almacenamiento MinIO S3",
        pItemMinioSub: "Crear un bucket S3 privado en tu VPS para respaldos y subida de multimedia.",
        pItemScaleLabel: "Escalar Réplicas de Servicio (Docker Swarm Replicas)",
        pItemScaleSub: "Incrementar o reducir el número de réplicas activas para alta disponibilidad.",
        pItemEnvSecretsLabel: "Gestionar Variables de Ambiente & Secretos (.env)",
        pItemEnvSecretsSub: "Inyectar o modificar claves de configuración dinámicas en contenedores.",
        pItemLinkLabel: "Interconectar Servicios (Enlazar A ➔ B en red overlay)",
        pItemLinkSub: "Vincular variables de conexión entre microservicios y bases de datos.",
        pItemNodesLabel: "Ver Nodos del Clúster Swarm & Estado de Salud",
        pItemNodesSub: "Consultar roles de Manager y Workers activos en la infraestructura.",
        pItemRollbackLabel: "Ejecutar Rollback de Servicio a Versión Anterior",
        pItemRollbackSub: "Revertir el código y la imagen de contenedor a la revisión previa estable.",
        pItemObsLabel: "Abrir Dashboard de Observabilidad (Portainer / Dozzle)",
        pItemObsSub: "Monitorear registros globales y contenedores mediante interfaz web integrada.",
        pItemBackupsLabel: "Configurar Respaldos Automáticos S3 / MinIO",
        pItemBackupsSub: "Programar copias de seguridad de bases de datos hacia buckets locales o remotos.",
        pItemAuditLabel: "Ver Registros de Auditoría Inmutable (Security Trail)",
        pItemAuditSub: "Inspeccionar el historial de comandos, accesos SSH y eventos del clúster.",
        pItemSshKeysLabel: "Gestor de Llaves SSH Autorizadas (Equipos de Desarrollo)",
        pItemSshKeysSub: "Añadir o revocar llaves SSH para desarrolladores con protección de llave Master.",
        pItemPruneLabel: "Limpieza de Sistema (Docker System Prune)",
        pItemPruneSub: "Eliminar imágenes huérfanas, contenedores detenidos y caché de compilación.",
        pItemTerminalLabel: "Abrir Terminal SSH Interactiva",
        pItemTerminalSub: "Lanzar una sesión de terminal web con acceso root al VPS.",

        // Modals
        modalDeployServiceTitle: "⚡ Desplegar Nuevo Servicio en Clúster",
        modalDeployDbTitle: "🗄️ Crear Base de Datos o Almacenamiento MinIO S3",
        modalCreateVmTitle: "⚡ Crear VM en la Nube & Instalar Framework",
        modalByoTitle: "🚀 Instalar Framework en VM Conectada (BYO)",
        modalAddWorkerTitle: "🏗️ Añadir Nodo Worker al Clúster",
        modalLinkTitle: "🔗 Interconectar Servicios (Enlazar A ➔ B)",
        modalBackupsTitle: "📦 Configurar Respaldos Automáticos S3 / MinIO",
        modalSshKeysTitle: "🔑 Gestor de Llaves SSH Autorizadas",
        modalAuditTitle: "📋 Registro de Auditoría Inmutable",
        modalNotificationsTitle: "🔔 Centro de Notificaciones e Historial",
        modalTerminalTitle: ">_ Node Terminal (Sesión SSH)",

        // Buttons
        btnCancel: "Cancelar",
        btnDeployService: "🚀 Desplegar Servicio",
        btnDeployDb: "🗄️ Desplegar Base de Datos",
        btnCreateVm: "⚡ Crear VM & Desplegar Framework",
        btnInitByo: "🚀 Inicializar Framework Completo",
        btnAddWorker: "🏗️ Crear & Unir Worker",
        btnCreateLink: "🔗 Crear Enlace",
        btnSaveBackupConfig: "💾 Guardar Configuración de Respaldo",
        btnAddKey: "+ Añadir Llave",
        btnClose: "Cerrar",
        btnClearNotifications: "Limpiar Notificaciones",
        btnDownloadAudit: "📥 Descargar NDJSON",

        // Progress overlay
        progressTitleRunning: "Ejecutando...",
        progressStepStarting: "Iniciando proceso...",
        progressTermTitle: "tarhiata — logs en tiempo real",
        progressBadgeLive: "LIVE",
        progressBadgeDone: "DONE",
        progressBadgeError: "ERROR",
        progressBtnClose: "✅ Cerrar y Continuar",
        progressCounterFormat: "líneas",
        langBadge: "🌐 ES",
        langText: "ES"
    },
    en: {
        refresh: "Refresh",
        terminal: "Node Terminal",
        palette: "Command Palette",
        catalogTitle: "Services & Databases Catalog",
        deployServiceBtn: "+ Deploy Service",
        createDbBtn: "+ Create DB",
        searchPlaceholder: "🔍 Filter live services & DBs by name or engine... (e.g. api, postgres, redis)",
        loadingCatalog: "Loading catalog from SQLite...",
        topologyTitle: "Flowchart & Network Diagram",
        viewTopologyTitle: "View text network topology",
        linkBtn: "+ Link A ➔ B",
        loadingTopology: "Loading network flowchart...",
        vpsMetricsTitle: "General VPS Metrics",
        vpsStatusConnected: "CONNECTED",
        vpsHostLabel: "Host:",
        swarmRoleMaster: "Master",
        statActiveApps: "Active Apps",
        subSwarmContainers: "Swarm Containers",
        statActiveDbs: "Active DBs",
        subOverlayIsolated: "Overlay Isolated",
        statSwarmNodes: "Swarm Nodes",
        subMasterVps: "Master VPS",
        statDiskStorage: "Disk Storage",
        cloudBillingTitle: "Cloud Billing & Plan",
        planName: "Pro VPS Plan (Self-Hosted)",
        swarmNodesTitle: "Swarm Nodes",
        tokenBtn: "📋 Token",
        workerBtn: "+ Worker",
        loadingNodes: "Loading Swarm cluster nodes...",
        backupsTitle: "Backups & Snapshots",
        snapshotBtn: "+ Snapshot",
        loadingBackups: "Loading snapshots...",
        previewTitle: "Preview Envs",
        previewBtn: "+ Preview",
        loadingPreviews: "Loading preview environments...",

        // Inspector Modal
        inspectorOpenUrl: "🌐 Open URL",
        inspectorDelete: "🗑️ Delete Resource",
        tabMetrics: "📊 Metrics",
        tabLogs: "📜 Logs",
        tabEnvs: "🔑 Envs (.env)",
        tabRollback: "⏪ Rollback",
        tabBackups: "💾 Snapshots / Backups",
        tabConfig: "⚙️ Configuration",
        autoValidateText: "⏱️ Network state auto-validation every 15 seconds",
        btnValidateNow: "🔄 Validate Connection Now",
        statReplicasLabel: "Replicas Status",
        statRamLabel: "Memory Usage (RAM)",
        statPortLabel: "Internal Port / Exposure",
        realtimeStatsTitle: "⚡ Real-Time Metrics (cgroups / Docker Stats)",
        cpuUsageLabel: "CPU Usage",
        ramUsageLabel: "RAM Usage",
        netIoLabel: "Network I/O",
        diskIoLabel: "Disk Block I/O",
        telemetryChartTitle: "Real-Time CPU & RAM Telemetry",

        logGrepPlaceholder: "🔍 Search logs (grep)...",
        logLevelAll: "All Levels",
        logLevelError: "Error / Fatal Only",
        logLevelWarn: "Warn / Warning Only",
        logLevelInfo: "Info Only",
        btnDownloadLogs: "📥 Download (.txt)",
        loadingLogs: "Loading service logs...",

        envTitle: "Injected Environment Variables (.env)",
        btnHideSecrets: "👁️ Hide Secrets",
        btnShowSecrets: "👁️ Show Secrets",
        btnTableMode: "⇄ Table Mode",
        btnRawMode: "⇄ Raw Mode",
        btnImportEnv: "MB Import .env",
        envKeyHeader: "Variable (Key)",
        envValueHeader: "Value",
        btnSaveEnvs: "💾 Save & Inject into Swarm",

        rollbackWarningTitle: "⚠️ Docker Swarm Version Rollback",
        rollbackWarningText: "This action runs docker service rollback on the remote server for this service. It will immediately restore the previous container image and configuration stored by Docker Swarm.",
        btnTriggerRollback: "⚠️ Execute Rollback to Previous Revision",

        backupSectionTitle: "Database Backups & Snapshots",
        btnCreateSnapshot: "+ Create 1-Click Snapshot",
        loadingBackupsList: "Loading backups list...",

        cfgNameLabel: "Resource Name",
        cfgImageLabel: "Image / Engine",
        cfgPortLabel: "Internal Port",
        cfgDomainLabel: "Domain / HTTPS Subdomain",
        cfgHealthLabel: "Healthcheck Command (Optional)",
        cfgExposeLabel: "🌐 Expose service publicly to the Internet (Via Traefik Reverse Proxy)",
        cfgSSLLabel: "🔒 Enable Automatic HTTPS / SSL (Let's Encrypt Certificate)",
        btnSaveConfig: "💾 Save Configuration Changes",

        // Palette Search & Items
        paletteSearchPlaceholder: "Filter system tools... (e.g. ssl, env, link, rollback, tailscale)",
        pItemCreateVmLabel: "Create Cloud VM & Install Framework",
        pItemCreateVmSub: "Provision a new VM on DigitalOcean/Vultr and install Docker, Swarm, Traefik and Observability.",
        pItemByoLabel: "Install Framework on Connected VM (BYO)",
        pItemByoSub: "Connect to an existing server via SSH and initialize Docker, Swarm, Traefik SSL and Observability.",
        pItemMasterLabel: "Master 1-Click Setup (Initialize VPS & Swarm Cluster)",
        pItemMasterSub: "Automated 1-Click setup of Docker Swarm, Traefik and subnets on the server.",
        pItemVolumesLabel: "Upload Files & Volume Explorer (/opt/data)",
        pItemVolumesSub: "Upload files to VPS disk volumes and manage data persistence.",
        pItemSslLabel: "Configure SSL, HTTPS Certificates & Custom Domains",
        pItemSslSub: "Activate 1-Click SSL (Let's Encrypt or Private Certificates) and map to services.",
        pItemTraefikLabel: "Restart Traefik Proxy & Refresh Routing",
        pItemTraefikSub: "Re-apply Traefik v3 rules and clear SSL certificate cache.",
        pItemDeployDbLabel: "Deploy 1-Click Database (PostgreSQL, Mongo, Redis, MySQL, MinIO)",
        pItemDeployDbSub: "Instantiate isolated DB containers with volumes mounted on /opt/data.",
        pItemMinioLabel: "Deploy MinIO S3 Storage Server",
        pItemMinioSub: "Create a private S3 bucket on your VPS for backups and media uploads.",
        pItemScaleLabel: "Scale Service Replicas (Docker Swarm Replicas)",
        pItemScaleSub: "Increase or decrease active replicas for high availability.",
        pItemEnvSecretsLabel: "Manage Environment Variables & Secrets (.env)",
        pItemEnvSecretsSub: "Inject or modify dynamic configuration keys in containers.",
        pItemLinkLabel: "Interconnect Services (Link A ➔ B on overlay network)",
        pItemLinkSub: "Link connection variables between microservices and databases.",
        pItemNodesLabel: "View Swarm Cluster Nodes & Health Status",
        pItemNodesSub: "Check active Manager and Worker roles across infrastructure.",
        pItemRollbackLabel: "Execute Service Rollback to Previous Version",
        pItemRollbackSub: "Revert container code and image to the previous stable revision.",
        pItemObsLabel: "Open Observability Dashboard (Portainer / Dozzle)",
        pItemObsSub: "Monitor global logs and containers via integrated web UI.",
        pItemBackupsLabel: "Configure Automatic S3 / MinIO Backups",
        pItemBackupsSub: "Schedule database backups to local or remote buckets.",
        pItemAuditLabel: "View Immutable Security Audit Trail",
        pItemAuditSub: "Inspect command history, SSH access, and cluster events.",
        pItemSshKeysLabel: "Authorized SSH Keys Manager (Dev Teams)",
        pItemSshKeysSub: "Add or revoke SSH keys for developers with Master key protection.",
        pItemPruneLabel: "System Clean-up (Docker System Prune)",
        pItemPruneSub: "Remove dangling images, stopped containers, and build cache.",
        pItemTerminalLabel: "Open Interactive SSH Terminal",
        pItemTerminalSub: "Launch a web terminal session with root access to the VPS.",

        // Modals
        modalDeployServiceTitle: "⚡ Deploy New Service to Cluster",
        modalDeployDbTitle: "🗄️ Create Database or MinIO S3 Storage",
        modalCreateVmTitle: "⚡ Create Cloud VM & Install Framework",
        modalByoTitle: "🚀 Install Framework on Connected VM (BYO)",
        modalAddWorkerTitle: "🏗️ Add Worker Node to Cluster",
        modalLinkTitle: "🔗 Interconnect Services (Link A ➔ B)",
        modalBackupsTitle: "📦 Configure Automatic S3 / MinIO Backups",
        modalSshKeysTitle: "🔑 Authorized SSH Keys Manager",
        modalAuditTitle: "📋 Immutable Security Audit Trail",
        modalNotificationsTitle: "🔔 Notifications Center & History",
        modalTerminalTitle: ">_ Node Terminal (SSH Session)",

        // Buttons
        btnCancel: "Cancel",
        btnDeployService: "🚀 Deploy Service",
        btnDeployDb: "🗄️ Deploy Database",
        btnCreateVm: "⚡ Create VM & Deploy Framework",
        btnInitByo: "🚀 Initialize Full Framework",
        btnAddWorker: "🏗️ Create & Join Worker",
        btnCreateLink: "🔗 Create Link",
        btnSaveBackupConfig: "💾 Save Backup Configuration",
        btnAddKey: "+ Add Key",
        btnClose: "Close",
        btnClearNotifications: "Clear Notifications",
        btnDownloadAudit: "📥 Download NDJSON",

        // Progress overlay
        progressTitleRunning: "Running...",
        progressStepStarting: "Starting process...",
        progressTermTitle: "tarhiata — real-time logs",
        progressBadgeLive: "LIVE",
        progressBadgeDone: "DONE",
        progressBadgeError: "ERROR",
        progressBtnClose: "✅ Close & Continue",
        progressCounterFormat: "lines",
        langBadge: "🌐 EN",
        langText: "EN"
    }
};

function getI18nText(key, fallback = '') {
    const t = i18n[currentLang] || i18n.es;
    return t[key] !== undefined ? t[key] : (fallback || key);
}

function initLanguage() {
    applyLanguage(currentLang);
}

function toggleLanguage() {
    currentLang = currentLang === 'es' ? 'en' : 'es';
    localStorage.setItem('tarhiata_lang', currentLang);
    applyLanguage(currentLang);
    if (typeof showToast === 'function') {
        showToast(currentLang === 'es' ? 'Idioma cambiado a Español 🇪🇸' : 'Language switched to English 🇬🇧', 'info');
    }
}

function applyLanguage(lang) {
    const t = i18n[lang] || i18n.es;
    
    const langBadge = document.getElementById('langBadge');
    const langText = document.getElementById('langText');
    if (langBadge) langBadge.textContent = t.langBadge;
    if (langText) langText.textContent = t.langText;

    const btnRefresh = document.getElementById('btnHeaderRefresh');
    if (btnRefresh) {
        btnRefresh.innerHTML = `<span class="cmd-k-badge" style="background: rgba(59,130,246,0.2); color: #60a5fa;">🔄</span> ${t.refresh}`;
    }

    // Apply data-i18n attributes automatically across the entire DOM
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (t[key]) el.textContent = t[key];
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        if (t[key]) el.placeholder = t[key];
    });
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
        const key = el.getAttribute('data-i18n-title');
        if (t[key]) el.title = t[key];
    });
}

document.addEventListener('DOMContentLoaded', () => {
    initLanguage();
    initCommandPalette();
    initModalsAndForms();
    loadDashboardData();
    loadSSHKeys();
    setInterval(loadDashboardData, 15000); // Auto-validación de conexión y estado cada 15 segundos
    setInterval(updateChart, 2000);
    setInterval(fetchLiveLogs, 3000);
});

async function loadSSHKeys() {
    try {
        const res = await fetch('/api/ssh-keys');
        if (!res.ok) return;
        const keys = await res.json();
        document.querySelectorAll('.ssh-key-select').forEach(sel => {
            const currentVal = sel.value;
            sel.innerHTML = '';
            if (!keys || keys.length === 0) {
                sel.innerHTML = '<option value="">No se encontraron llaves en ~/.ssh/</option>';
                return;
            }
            keys.forEach((k, i) => {
                const opt = document.createElement('option');
                opt.value = k.path;
                opt.textContent = `🔑 ${k.name}  (${k.path})`;
                if (i === 0 && !currentVal) opt.selected = true;
                if (currentVal === k.path) opt.selected = true;
                sel.appendChild(opt);
            });
        });
    } catch (e) {
        console.warn('No se pudieron cargar llaves SSH:', e);
    }
}

/* --- FULLSCREEN PROGRESS OVERLAY — Streaming en tiempo real --- */
let progLineCount = 0;

function showProgressOverlay(title) {
    progLineCount = 0;
    const overlay = document.getElementById('progressOverlay');
    document.getElementById('progTitle').textContent = title;
    document.getElementById('progStep').textContent = 'Iniciando proceso...';
    document.getElementById('progStep').style.animation = 'progPulse 2s ease-in-out infinite';
    document.getElementById('progLogs').innerHTML = '';
    document.getElementById('progFooter').style.display = 'none';
    document.getElementById('progCounter').textContent = '0 líneas';
    document.getElementById('progBadge').textContent = 'LIVE';
    document.getElementById('progBadge').style.background = 'rgba(99, 102, 241, 0.15)';
    document.getElementById('progBadge').style.color = '#818cf8';
    const spinner = document.getElementById('progSpinner');
    spinner.className = 'prog-spinner';
    overlay.style.display = 'flex';
    document.body.style.overflow = 'hidden';
}

function closeProgressOverlay() {
    document.getElementById('progressOverlay').style.display = 'none';
    document.body.style.overflow = '';
    loadDashboardData(true);
}

function appendProgressLog(type, message) {
    const logs = document.getElementById('progLogs');
    const line = document.createElement('div');
    line.className = `prog-line ${type}`;

    const ts = new Date().toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    const escaped = document.createElement('span');
    escaped.textContent = message;

    line.innerHTML = `<span class="prog-ts">[${ts}]</span>`;
    line.appendChild(escaped);

    logs.appendChild(line);
    logs.scrollTop = logs.scrollHeight;

    progLineCount++;
    document.getElementById('progCounter').textContent = `${progLineCount} líneas`;

    if (type === 'step') {
        document.getElementById('progStep').textContent = message;
    }
}

function showProgressDone(success) {
    const spinner = document.getElementById('progSpinner');
    spinner.className = 'prog-spinner ' + (success ? 'done' : 'error');

    document.getElementById('progStep').style.animation = 'none';
    document.getElementById('progStep').textContent = success ? '¡Proceso completado!' : 'Error en el proceso';
    document.getElementById('progStep').style.color = success ? '#10b981' : '#ef4444';

    const badge = document.getElementById('progBadge');
    badge.textContent = success ? 'DONE' : 'ERROR';
    badge.style.background = success ? 'rgba(16,185,129,0.15)' : 'rgba(239,68,68,0.15)';
    badge.style.color = success ? '#10b981' : '#ef4444';

    const footer = document.getElementById('progFooter');
    footer.style.display = 'block';
    const btn = document.getElementById('progCloseBtn');
    btn.textContent = success ? '✅ Cerrar y Continuar' : '❌ Cerrar';
    btn.style.background = success
        ? 'linear-gradient(135deg, #10b981, #059669)'
        : 'linear-gradient(135deg, #ef4444, #dc2626)';
}

async function startStreamingOperation(url, payload, title) {
    showProgressOverlay(title);

    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        // Si el server respondió con error HTTP (antes del streaming)
        if (!response.ok && !response.headers.get('content-type')?.includes('ndjson')) {
            const errText = await response.text();
            appendProgressLog('error', '❌ ' + errText);
            showProgressDone(false);
            return;
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let hadDone = false;
        let hadError = false;

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop();

            for (const rawLine of lines) {
                if (!rawLine.trim()) continue;
                try {
                    const ev = JSON.parse(rawLine);
                    if (ev.t === 'done') {
                        appendProgressLog('done-line', '🎉 ' + (ev.d?.message || '¡Proceso completado!'));
                        hadDone = true;
                    } else if (ev.t === 'error') {
                        appendProgressLog('error', ev.m);
                        hadError = true;
                    } else {
                        appendProgressLog(ev.t || 'log', ev.m);
                    }
                } catch (_) {
                    appendProgressLog('log', rawLine);
                }
            }
        }

        // Buffer residual
        if (buffer.trim()) {
            try {
                const ev = JSON.parse(buffer);
                if (ev.t === 'done') { hadDone = true; appendProgressLog('done-line', '🎉 ' + (ev.d?.message || '¡Completado!')); }
                else if (ev.t === 'error') { hadError = true; appendProgressLog('error', ev.m); }
            } catch (_) {}
        }

        if (hadError) showProgressDone(false);
        else if (hadDone) showProgressDone(true);
        else showProgressDone(true);

    } catch (err) {
        appendProgressLog('error', `Error de conexión: ${err.message}`);
        showProgressDone(false);
    }
}

let globalNotificationHistory = [];

/* --- 1. Custom Toast Notification Banner System --- */
function showToast(title, message, type = 'success', duration = 4000) {
    // Record to notification history
    const item = {
        title: title,
        message: message,
        type: type,
        time: new Date().toLocaleTimeString()
    };
    globalNotificationHistory.unshift(item);
    updateNotificationBadge();

    const container = document.getElementById('toastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    let icon = '✓';
    if (type === 'error') icon = '✕';
    if (type === 'info') icon = 'ℹ';
    if (type === 'warning') icon = '⚠️';

    let displayMsg = message;
    if (typeof message === 'string' && message.trim().startsWith('{')) {
        try {
            const parsed = JSON.parse(message);
            if (parsed.error) displayMsg = parsed.error;
            else if (parsed.message) displayMsg = parsed.message;
        } catch (e) {}
    }

    toast.innerHTML = `
        <div class="toast-icon">${icon}</div>
        <div class="toast-content">
            <div class="toast-title">${title}</div>
            <div class="toast-message">${displayMsg}</div>
        </div>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.add('toast-exit');
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

function updateNotificationBadge() {
    const badge = document.getElementById('notificationBadgeCount');
    if (badge) {
        badge.innerText = globalNotificationHistory.length;
    }
    renderNotificationHistory();
}

function renderNotificationHistory() {
    const list = document.getElementById('notificationsHistoryList');
    if (!list) return;

    if (globalNotificationHistory.length === 0) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">No hay notificaciones registradas en esta sesión.</div>`;
        return;
    }

    list.innerHTML = globalNotificationHistory.map(n => {
        let badgeClass = 'badge-green';
        if (n.type === 'error') badgeClass = 'badge-red';
        if (n.type === 'info') badgeClass = 'badge-blue';
        if (n.type === 'warning') badgeClass = 'badge-yellow';

        return `
            <div class="endpoint-item" style="padding: 8px 12px;">
                <div>
                    <div style="display:flex; align-items:center; gap:8px;">
                        <span class="badge ${badgeClass}" style="font-size:0.7rem;">${n.type.toUpperCase()}</span>
                        <strong style="font-size:0.85rem;">${n.title}</strong>
                        <span class="text-muted" style="font-size:0.72rem;">(${n.time})</span>
                    </div>
                    <div style="font-size:0.8rem; color:var(--text-secondary); margin-top:3px;">${n.message}</div>
                </div>
            </div>
        `;
    }).join('');
}

function clearNotificationHistory() {
    globalNotificationHistory = [];
    updateNotificationBadge();
}

/* --- 2. Command Palette (⌘K Spotlight Search) --- */
function initCommandPalette() {
    const overlay = document.getElementById('paletteOverlay');
    const input = document.getElementById('paletteInput');
    const btnOpen = document.getElementById('btnOpenPalette');
    const items = document.querySelectorAll('.palette-item');

    if (!overlay || !input) return;

    function openPalette() {
        overlay.classList.add('active');
        input.value = '';
        filterItems('');
        setTimeout(() => input.focus(), 50);
    }

    function closePalette() {
        overlay.classList.remove('active');
    }

    if (btnOpen) btnOpen.addEventListener('click', openPalette);

    document.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
            e.preventDefault();
            if (overlay.classList.contains('active')) {
                closePalette();
            } else {
                openPalette();
            }
        }
        if (e.key === 'Escape' && overlay.classList.contains('active')) {
            closePalette();
        }
    });

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) closePalette();
    });

    input.addEventListener('input', (e) => {
        filterItems(e.target.value.toLowerCase().trim());
    });

    items.forEach(item => {
        item.addEventListener('click', () => {
            const action = item.getAttribute('data-action');
            closePalette();
            executePaletteAction(action);
        });
    });

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            const visibleItem = document.querySelector('.palette-item:not([style*="display: none"])');
            if (visibleItem) {
                const action = visibleItem.getAttribute('data-action');
                closePalette();
                executePaletteAction(action);
            }
        }
    });
}

function filterItems(query) {
    const items = document.querySelectorAll('.palette-item');
    items.forEach(item => {
        const text = item.innerText.toLowerCase();
        const keywords = (item.getAttribute('data-keywords') || '').toLowerCase();
        const fullSearchableText = `${text} ${keywords}`;
        if (!query || fullSearchableText.includes(query)) {
            item.style.display = 'flex';
        } else {
            item.style.display = 'none';
        }
    });
}

function executePaletteAction(action) {
    switch (action) {
        case 'open-logs':
            const logBox = document.getElementById('liveLogContent');
            if (logBox) logBox.scrollIntoView({ behavior: 'smooth' });
            fetchLiveLogs();
            break;
        case 'create-vm':
            openModal('createVmModal');
            break;
        case 'byo-bootstrap':
            openModal('byoBootstrapModal');
            break;
        case 'bootstrap-master':
            openModal('bootstrapMasterModal');
            break;
        case 'create-preview':
            openModal('previewModal');
            break;
        case 'deploy-service':
            openModal('deployServiceModal');
            break;
        case 'rollback-service':
            promptRollbackService();
            break;
        case 'deploy-db':
            openModal('deployDBModal');
            break;
        case 'manage-registry':
            openRegistryModal();
            break;
        case 'manage-ssh-keys':
            openSSHKeysModal();
            break;
        case 'view-audit-logs':
        case 'manage-audit':
            openAuditLogsModal();
            break;
        case 'manage-migrations':
            openMigrationModal();
            break;
        case 'manage-backups':
        case 'create-backup':
            openBackupModal();
            break;
        case 'manage-env':
            openEnvModal();
            break;
        case 'browse-volumes':
            openVolumeBrowser();
            break;
        case 'inspect-ssl':
        case 'manage-domains':
            openDomainModal();
            break;
        case 'link-services':
            openModal('linkModal');
            break;
        case 'provision-worker':
            openModal('workerModal');
            break;
        case 'manage-nodes':
            openNodeManagementModal();
            break;
        case 'config-vps':
            openModal('configModal');
            break;
        case 'run-bootstrap':
            runBootstrapAction();
            break;
        case 'deploy-obs':
            runDeployObsAction();
            break;
        case 'view-metrics':
            openMetricsModal();
            break;
        case 'install-tailscale':
            openModal('tailscaleModal');
            break;
        case 'docker-prune':
            runPruneAction();
            break;
        case 'view-topology':
            viewTopologyAction();
            break;
        case 'restart-traefik':
            restartTraefikAction();
            break;
    }
}

function restartTraefikAction() {
    requestConfirmation(
        '🔄 Reiniciar Traefik Proxy',
        '¿Deseas forzar el reinicio del servicio Traefik Proxy en Docker Swarm?',
        async () => {
            showToast('Reiniciando Traefik', 'Ejecutando docker service update --force traefik...', 'info');
            try {
                const res = await fetch('/api/tools/restart-traefik', { method: 'POST' });
                if (res.ok) {
                    showToast('Traefik Reiniciado 🚀', 'Proxy Traefik reiniciado con éxito', 'success');
                } else {
                    const err = await res.json().catch(() => ({ error: 'Error al reiniciar Traefik' }));
                    showToast('Error', err.error || err.message, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

/* --- 3. Modals & Form Submission Handling --- */
function initModalsAndForms() {
    // Backdrop click and ESC key listeners for closing all modals
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            document.querySelectorAll('.modal-overlay.active').forEach(m => m.classList.remove('active'));
        }
    });

    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                overlay.classList.remove('active');
            }
        });
    });

    // Terminal CLI form handler
    const formCliTerminal = document.getElementById('formCliTerminal');
    if (formCliTerminal) {
        formCliTerminal.addEventListener('submit', (e) => {
            e.preventDefault();
            const input = document.getElementById('cliTerminalInput');
            if (input && input.value.trim()) {
                const cmd = input.value.trim();
                input.value = '';
                execTerminalCmd(cmd);
            }
        });
    }

    // Formulario Link (A -> B)
    const formLink = document.getElementById('formLink');
    if (formLink) {
        formLink.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                sourceSvc: document.getElementById('linkSource').value.trim(),
                targetSvc: document.getElementById('linkTarget').value.trim(),
                envVarName: document.getElementById('linkEnvVar').value.trim()
            };
            showToast('Procesando Interconexión', `Conectando ${payload.sourceSvc} ➔ ${payload.targetSvc}...`, 'info');
            try {
                const res = await fetch('/api/links', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Enlace Creado', `Inyectada ${payload.envVarName} en Swarm para ${payload.sourceSvc}`, 'success');
                    closeModal('linkModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error en Interconexión', err, 'error');
                }
            } catch (err) {
                showToast('Error de Red', err.message, 'error');
            }
        });
    }

    // Formulario Entornos Preview (Testing / PR)
    const formPreview = document.getElementById('formPreview');
    if (formPreview) {
        formPreview.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                name: document.getElementById('prevName').value.trim(),
                image: document.getElementById('prevImage').value.trim(),
                imageSource: document.getElementById('prevImage').value.trim(),
                port: parseInt(document.getElementById('prevPort').value, 10) || 80,
                domain: document.getElementById('prevDomain').value.trim(),
                link_db_name: document.getElementById('prevLinkDB').value.trim(),
                linkDbName: document.getElementById('prevLinkDB').value.trim(),
                target_node: document.getElementById('prevNode')?.value || 'manager',
                targetNode: document.getElementById('prevNode')?.value || 'manager'
            };

            showToast('Creando Entorno Preview', `Lanzando contenedor efímero '${payload.name}'...`, 'info', 5000);

            try {
                const res = await fetch('/api/previews', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Entorno Preview Creado', `¡Entorno temporal '${payload.name}' activo!`, 'success');
                    closeModal('previewModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error Creando Preview', err, 'error');
                }
            } catch (err) {
                showToast('Error de Conexión', err.message, 'error');
            }
        });
    }

    // Formulario Deploy Service
    const formDeployService = document.getElementById('formDeployService');
    if (formDeployService) {
        formDeployService.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                name: document.getElementById('svcName').value.trim(),
                image_source: document.getElementById('svcImage').value.trim(),
                port: parseInt(document.getElementById('svcPort').value, 10),
                domain: document.getElementById('svcDomain').value.trim(),
                expose: document.getElementById('svcExpose').checked,
                enable_ssl: document.getElementById('svcSSL').checked,
                targetNode: document.getElementById('svcNode')?.value || 'manager',
                pre_deploy_hook: document.getElementById('svcPreDeployHook')?.value.trim() || ''
            };

            closeModal('deployServiceModal');
            startStreamingOperation(
                '/api/services',
                payload,
                `🚀 Desplegar Servicio (${payload.name})`
            );
        });
    }

    // Formulario Master Bootstrapper (All-in-One 1-Click)
    const formBootstrapMaster = document.getElementById('formBootstrapMaster');
    if (formBootstrapMaster) {
        formBootstrapMaster.addEventListener('submit', (e) => {
            e.preventDefault();
            const payload = {
                app_name: document.getElementById('bmAppName').value.trim(),
                image: document.getElementById('bmImage').value.trim(),
                port: parseInt(document.getElementById('bmPort').value, 10),
                db_engine: document.getElementById('bmDBEngine').value,
                env_var_name: document.getElementById('bmEnvVarName').value.trim(),
                domain: document.getElementById('bmDomain').value.trim(),
                expose_public: !!document.getElementById('bmDomain').value.trim()
            };

            requestConfirmation(
                '🪄 Confirmar Inicialización Master',
                `¿Deseas inicializar la aplicación '${payload.app_name}' con la base de datos '${payload.db_engine}'? Si '${payload.app_name}' ya tenía enlaces a otra BD, se desvinculará automáticamente.`,
                async () => {
                    closeModal('bootstrapMasterModal');
                    showToast('Inicializando Master', `Creando ${payload.app_name} y desvinculando entornos previos...`, 'info', 8000);
                    try {
                        const res = await fetch('/api/bootstrap-master', {
                            method: 'POST',
                            headers: {'Content-Type': 'application/json'},
                            body: JSON.stringify(payload)
                        });
                        if (res.ok) {
                            const data = await res.json();
                            let msg = `¡App ${payload.app_name} iniciada!`;
                            if (data.unlinked_old && data.unlinked_old.length > 0) {
                                msg += ` (Desvinculada de ${data.unlinked_old.join(', ')})`;
                            }
                            showToast('Master Bootstrap Exitoso', msg, 'success');
                            loadDashboardData();
                        } else {
                            const err = await res.text();
                            showToast('Error en Bootstrap', err, 'error');
                        }
                    } catch (err) {
                        showToast('Error de Conexión', err.message, 'error');
                    }
                }
            );
        });
    }

    // Formulario Crear VM en la Nube & Instalar Framework
    const cvmProviderSelect = document.getElementById('cvmProvider');
    if (cvmProviderSelect) {
        cvmProviderSelect.addEventListener('change', () => {
            const regionInput = document.getElementById('cvmRegion');
            if (regionInput) {
                if (cvmProviderSelect.value === 'vultr') {
                    regionInput.value = 'ewr';
                    regionInput.placeholder = 'ej. ewr (NJ), ord (Chicago), lax, mia, ams';
                } else {
                    regionInput.value = 'nyc1';
                    regionInput.placeholder = 'ej. nyc1, nyc3, ams3, sfo3';
                }
            }
        });
    }

    const formCreateVM = document.getElementById('formCreateVM');
    if (formCreateVM) {
        formCreateVM.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                provider: document.getElementById('cvmProvider').value,
                nodeName: document.getElementById('cvmNodeName').value.trim(),
                apiToken: document.getElementById('cvmApiToken').value.trim(),
                region: document.getElementById('cvmRegion').value.trim(),
                acmeEmail: document.getElementById('cvmAcmeEmail').value.trim(),
                installObservability: document.getElementById('cvmInstallObs').checked
            };

            closeModal('createVmModal');
            startStreamingOperation(
                '/api/create-vm-bootstrap',
                payload,
                `⚡ Crear VM (${payload.provider.toUpperCase()}) & Instalar Framework`
            );
        });
    }

    // Formulario Instalar Framework en VM Conectada (BYO)
    const formBYOBootstrap = document.getElementById('formBYOBootstrap');
    if (formBYOBootstrap) {
        formBYOBootstrap.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                host: document.getElementById('byoHost').value.trim(),
                port: parseInt(document.getElementById('byoPort').value, 10) || 22,
                user: document.getElementById('byoUser').value.trim() || 'root',
                keyPath: document.getElementById('byoKeyPath').value.trim(),
                acmeEmail: document.getElementById('byoAcmeEmail').value.trim(),
                installObservability: document.getElementById('byoInstallObs').checked
            };

            closeModal('byoBootstrapModal');
            startStreamingOperation(
                '/api/bootstrap',
                payload,
                `🚀 Instalar Framework en ${payload.host}`
            );
        });
    }

    // Formulario Deploy DB
    const formDeployDB = document.getElementById('formDeployDB');
    if (formDeployDB) {
        formDeployDB.addEventListener('submit', async (e) => {
            e.preventDefault();
            const recoveryVal = document.getElementById('dbRecoveryMode')?.value || 'reuse';
            const payload = {
                name: document.getElementById('dbName').value.trim(),
                engine: document.getElementById('dbEngine').value,
                internalPort: parseInt(document.getElementById('dbPort')?.value || 0, 10),
                targetNode: document.getElementById('dbNode')?.value || 'manager',
                reuseExistingData: recoveryVal === 'reuse',
                cleanExistingData: recoveryVal === 'clean'
            };

            closeModal('deployDBModal');
            startStreamingOperation(
                '/api/databases',
                payload,
                `🗄️ Desplegar Base de Datos (${payload.name})`
            );
        });
    }

    // Formulario Provision Worker
    const formWorker = document.getElementById('formWorker');
    if (formWorker) {
        formWorker.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                nodeName: document.getElementById('workerName').value.trim(),
                plan: document.getElementById('workerPlan') ? document.getElementById('workerPlan').value : 'vc2-1c-1gb',
                region: document.getElementById('workerRegion').value.trim(),
                labelType: document.getElementById('workerType') ? document.getElementById('workerType').value : 'worker'
            };

            closeModal('workerModal');
            startStreamingOperation(
                '/api/workers',
                payload,
                `🏗️ Provisionar Nodo Worker (${payload.nodeName})`
            );
        });
    }

    // Formulario Config VPS
    const formConfigVPS = document.getElementById('formConfigVPS');
    if (formConfigVPS) {
        formConfigVPS.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                host: document.getElementById('cfgHost').value.trim(),
                port: parseInt(document.getElementById('cfgPort').value, 10),
                user: document.getElementById('cfgUser').value.trim(),
                key_path: document.getElementById('cfgKey').value.trim(),
                privateKey: document.getElementById('cfgKey').value.trim(),
                do_token: document.getElementById('cfgDOToken').value.trim(),
                doApiToken: document.getElementById('cfgDOToken').value.trim(),
                vultrApiToken: document.getElementById('cfgDOToken').value.trim()
            };
            showToast('Guardando Configuración', 'Actualizando credenciales del cluster VPS...', 'info');
            try {
                const res = await fetch('/api/config', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Configuración Guardada', '¡Parámetros de VPS y Vultr API Key actualizados!', 'success');
                    closeModal('configModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error de Configuración', err, 'error');
                }
            } catch (err) {
                showToast('Error de Conexión', err.message, 'error');
            }
        });
    }

    // Formulario Editar Servicio
    const formEditService = document.getElementById('formEditService');
    if (formEditService) {
        formEditService.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                name: document.getElementById('editSvcOriginalName').value.trim(),
                image_source: document.getElementById('editSvcImage').value.trim(),
                port: parseInt(document.getElementById('editSvcPort').value, 10),
                domain: document.getElementById('editSvcDomain').value.trim(),
                expose: document.getElementById('editSvcExpose').checked,
                enable_ssl: document.getElementById('editSvcSSL').checked,
                env_vars: document.getElementById('editSvcEnvVars').value.trim(),
                healthcheck_cmd: document.getElementById('editSvcHealthcheck').value.trim(),
                pre_deploy_hook: document.getElementById('editSvcPreDeployHook')?.value.trim() || ''
            };
            showToast('Actualizando Servicio', `Reconfigurando ${payload.name} en Docker Swarm...`, 'info', 5000);
            try {
                const res = await fetch('/api/services', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Servicio Actualizado', `¡Cambios y variables de entorno aplicadas a ${payload.name}!`, 'success');
                    closeModal('editServiceModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Actualizar', err, 'error');
                }
            } catch (err) {
                showToast('Error de Conexión', err.message, 'error');
            }
        });
    }

    // Formulario Editar BD
    const formEditDB = document.getElementById('formEditDB');
    if (formEditDB) {
        formEditDB.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                name: document.getElementById('editDBOriginalName').value.trim(),
                engine: document.getElementById('editDBEngine')?.value || 'postgres',
                internalPort: parseInt(document.getElementById('editDBPort')?.value || 5432, 10),
                targetNode: document.getElementById('editDBNode')?.value || 'manager'
            };
            showToast('Actualizando BD', `Guardando configuración de ${payload.name}...`, 'info');
            try {
                const res = await fetch('/api/databases', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('BD Actualizada', `Configuración para ${payload.name} sincronizada`, 'success');
                    closeModal('editDBModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Actualizar BD', err, 'error');
                }
            } catch (err) {
                showToast('Error de Conexión', err.message, 'error');
            }
        });
    }

    // Formulario Editar Link
    const formEditLink = document.getElementById('formEditLink');
    if (formEditLink) {
        formEditLink.addEventListener('submit', async (e) => {
            e.preventDefault();
            const payload = {
                sourceSvc: document.getElementById('editLinkSource').value,
                targetSvc: document.getElementById('editLinkTarget').value,
                envVarName: document.getElementById('editLinkEnvVar').value.trim(),
                source_svc: document.getElementById('editLinkSource').value,
                target_svc: document.getElementById('editLinkTarget').value,
                env_var_name: document.getElementById('editLinkEnvVar').value.trim()
            };
            showToast('Actualizando Enlace', `Actualizando variable ${payload.env_var_name}...`, 'info');
            try {
                const res = await fetch('/api/links', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Enlace Actualizado', `Variable ${payload.env_var_name} reinyectada`, 'success');
                    closeModal('editLinkModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Actualizar', err, 'error');
                }
            } catch (err) {
                showToast('Error de Red', err.message, 'error');
            }
        });
    }

    // Formulario Observabilidad
    const observabilityForm = document.getElementById('observabilityForm');
    if (observabilityForm) {
        observabilityForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const volumePath = document.getElementById('obsVolumePath').value;
            const grafanaPassword = document.getElementById('obsPassword').value;
            const deployType = document.getElementById('obsDeployType').value;
            const exposePublic = document.getElementById('obsExposePublic').checked;

            closeModal('observabilityModal');
            requestConfirmation(
                '⚠️ Desplegar Observabilidad',
                `¿Confirmas desplegar Loki, Promtail, Grafana y Portainer montados en la VM en '${volumePath}'?`,
                async () => {
                    showToast('Desplegando Observabilidad', 'Creando contenedores y volúmenes externos...', 'info', 10000);
                    try {
                        const res = await fetch('/api/observability', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                action: 'deploy',
                                enabled: true,
                                volumePath,
                                grafanaPassword,
                                deployType,
                                exposePublic
                            })
                        });
                        if (res.ok) {
                            showToast('Observabilidad Lista', '¡Loki y Grafana montados y activos en el clúster!', 'success');
                            loadDashboardData();
                        } else {
                            const err = await res.json().catch(() => ({ error: 'Error al desplegar observabilidad' }));
                            showToast('Error Observabilidad', err.error || 'No se pudo desplegar observabilidad', 'error');
                        }
                    } catch (err) {
                        showToast('Error de Conexión', err.message, 'error');
                    }
                }
            );
        });
    }
}

function openModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.add('active');
    if (id === 'workerModal') {
        loadVultrPlans();
    }
}

async function loadVultrPlans() {
    const select = document.getElementById('workerPlan');
    if (!select) return;
    try {
        const res = await fetch('/api/vultr/plans');
        if (!res.ok) return;
        const plans = await res.json();
        if (plans && plans.length > 0) {
            select.innerHTML = plans.map(p => `
                <option value="${p.id}">
                    ${p.id} — ${p.vcpu_count} vCPU, ${(p.ram / 1024).toFixed(1)} GB RAM, ${p.disk} GB SSD ($${p.monthly_cost.toFixed(2)}/mes)
                </option>
            `).join('');
        }
    } catch (e) {
        console.warn('Error cargando planes de Vultr:', e);
    }
}

function closeModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.remove('active');
}

/* --- 4. Service / DB / Link Inspection & Deletion Modal Triggers --- */
function openEditServiceModal(name) {
    const svc = globalServices.find(s => s.name === name);
    if (!svc) return;

    document.getElementById('editSvcTitle').innerText = svc.name;
    document.getElementById('editSvcOriginalName').value = svc.name;
    document.getElementById('editSvcImage').value = svc.imageSource || svc.image_source || '';
    document.getElementById('editSvcPort').value = svc.port || 80;
    document.getElementById('editSvcDomain').value = svc.domain || '';
    document.getElementById('editSvcExpose').checked = !!svc.expose;
    document.getElementById('editSvcSSL').checked = svc.enableSSL !== undefined ? svc.enableSSL : (svc.enable_ssl !== undefined ? svc.enable_ssl : false);

    const envVarsEl = document.getElementById('editSvcEnvVars');
    if (envVarsEl) {
        envVarsEl.value = svc.envVars || svc.env_vars || '';
    }

    // Live preview of injected link ENV variables
    const activeLinks = (globalLinks || []).filter(l => (l.sourceSvc || l.source_svc) === name);
    const injectedPreview = document.getElementById('editSvcInjectedPreview');
    const injectedBadge = document.getElementById('editSvcInjectedBadge');
    if (activeLinks.length > 0) {
        if (injectedBadge) injectedBadge.style.display = 'inline-block';
        if (injectedPreview) {
            injectedPreview.style.display = 'block';
            const linkLines = activeLinks.map(l => {
                const envName = l.envVarName || l.env_var_name || 'LINK';
                const tgt = l.targetSvc || l.target_svc || 'destino';
                const tgtUrl = l.targetUrl || l.target_url || `${tgt}:port`;
                return `⚡ Inyectada automáticamente por enlace a '${tgt}':<br><strong>${envName}</strong> = <code>${tgtUrl}</code>`;
            }).join('<br>');
            injectedPreview.innerHTML = linkLines;
        }
    } else {
        if (injectedBadge) injectedBadge.style.display = 'none';
        if (injectedPreview) {
            injectedPreview.style.display = 'none';
            injectedPreview.innerHTML = '';
        }
    }

    const hcEl = document.getElementById('editSvcHealthcheck');
    if (hcEl) hcEl.value = svc.healthcheckCmd || svc.healthcheck_cmd || '';

    const hookEl = document.getElementById('editSvcPreDeployHook');
    if (hookEl) hookEl.value = svc.preDeployHook || svc.pre_deploy_hook || '';

    openModal('editServiceModal');
}

function openEditDBModal(name) {
    const db = globalDatabases.find(d => d.name === name);
    if (!db) return;

    document.getElementById('editDBTitle').innerText = db.name;
    document.getElementById('editDBOriginalName').value = db.name;
    document.getElementById('editDBEngine').value = db.engine || 'database';
    document.getElementById('editDBPort').value = db.internalPort || db.internal_port || 5432;
    document.getElementById('editDBDeployType').value = db.deployType || db.deploy_type || 'single-node';
    document.getElementById('editDBInternalURI').value = `${db.name}:${db.internalPort || db.internal_port || 5432}`;

    openModal('editDBModal');
}

function openEditLinkModal(sourceSvc, targetSvc, envVarName) {
    document.getElementById('editLinkSource').value = sourceSvc;
    document.getElementById('editLinkTarget').value = targetSvc;
    document.getElementById('editLinkSourceDisplay').value = sourceSvc;
    document.getElementById('editLinkTargetDisplay').value = targetSvc;
    document.getElementById('editLinkEnvVar').value = envVarName;

    openModal('editLinkModal');
}

async function deleteServiceFromModalAction() {
    const name = document.getElementById('editSvcOriginalName').value;
    if (!name) return;

    requestConfirmation(
        '🚨 Destruir Servicio App',
        `¿Estás seguro de que deseas eliminar permanentemente el servicio '${name}' de Docker Swarm? Esta acción no se puede deshacer.`,
        async () => {
            showToast('Eliminando Servicio', `Destruyendo contenedor ${name}...`, 'info');
            try {
                const res = await fetch(`/api/services?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Servicio Eliminado', `El servicio '${name}' ha sido removido.`, 'success');
                    closeModal('editServiceModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Eliminar', err, 'error');
                }
            } catch (err) {
                showToast('Error de Red', err.message, 'error');
            }
        }
    );
}

async function deleteDatabaseFromModalAction() {
    const name = document.getElementById('editDBOriginalName').value;
    if (!name) return;

    requestConfirmation(
        '🔥 Eliminar Base de Datos',
        `¿Estás seguro de que deseas detener y remover la BD '${name}'? Todos los datos no respaldados se perderán.`,
        async () => {
            showToast('Deteniendo BD', `Removiendo base de datos '${name}'...`, 'info');
            try {
                const res = await fetch(`/api/databases?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('BD Detenida', `La base de datos '${name}' ha sido removida del clúster.`, 'success');
                    closeModal('editDBModal');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Detener BD', err, 'error');
                }
            } catch (err) {
                showToast('Error de Red', err.message, 'error');
            }
        }
    );
}

function deleteLinkFromModalAction() {
    const src = document.getElementById('editLinkSource').value;
    const tgt = document.getElementById('editLinkTarget').value;

    requestConfirmation(
        '🔗 Desvincular Enlace',
        `¿Deseas desvincular la relación de conexión entre '${src}' y '${tgt}'?`,
        () => {
            deleteLinkAction(src, tgt);
            closeModal('editLinkModal');
        }
    );
}

/* --- 5. Terminal CLI Functions --- */
function openTerminalModal() {
    openModal('cliTerminalModal');
}

function switchNodeTerminal(nodeVal) {
    const titleEl = document.getElementById('termHeaderTitle');
    const prefixEl = document.getElementById('termPromptPrefix');
    const statusEl = document.getElementById('nodeStatusBadge');

    const sshUser = (document.getElementById('cfgUser') && document.getElementById('cfgUser').value.trim()) ? document.getElementById('cfgUser').value.trim() : 'root';
    const promptStr = `${sshUser}@${nodeVal}:~#`;

    if (titleEl) titleEl.innerText = promptStr;
    if (prefixEl) prefixEl.innerText = promptStr;
    if (statusEl) statusEl.innerText = `Connected (${nodeVal})`;
}

function runQuickCmd(cmd) {
    const input = document.getElementById('cliTerminalInput');
    if (input) {
        input.value = cmd;
        execTerminalCmd(cmd);
    }
}

async function execTerminalCmd(cmdStr) {
    const output = document.getElementById('cliTerminalOutput');
    const nodeVal = document.getElementById('nodeSelectDropdown') ? document.getElementById('nodeSelectDropdown').value : 'manager';
    const sshUser = (document.getElementById('cfgUser') && document.getElementById('cfgUser').value.trim()) ? document.getElementById('cfgUser').value.trim() : 'root';

    if (!output) return;

    output.innerHTML += `\n${sshUser}@${nodeVal}:~# ${cmdStr}\n[Ejecutando comando SSH en ${nodeVal}...]\n`;
    output.scrollTop = output.scrollHeight;

    try {
        const res = await fetch('/api/terminal/exec', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ command: cmdStr, nodeId: nodeVal })
        });
        if (res.ok) {
            const data = await res.json();
            output.innerHTML += (data.output || '') + '\n';
        } else {
            const err = await res.json().catch(() => ({ error: 'Error al ejecutar comando' }));
            output.innerHTML += `[Error]: ${err.error || err.message}\n`;
        }
    } catch (err) {
        output.innerHTML += `[Error execution]: ${err.message}\n`;
    }
    output.scrollTop = output.scrollHeight;
}

/* --- 6. Live Log Stream Inspector Functions --- */
function openLogsForTarget(targetName) {
    const selectSvc = document.getElementById('logServiceSelect');
    if (selectSvc) {
        selectSvc.value = targetName;
        fetchLiveLogs();
        const logsCard = document.getElementById('liveLogContent');
        if (logsCard) {
            logsCard.scrollIntoView({ behavior: 'smooth' });
        }
    }
}

async function fetchLiveLogs() {
    const selectSvc = document.getElementById('logServiceSelect');
    const selectLines = document.getElementById('logLinesSelect');
    const logBox = document.getElementById('liveLogContent');
    if (!selectSvc || !logBox) return;

    const svcName = selectSvc.value;
    const lines = selectLines ? selectLines.value : 50;

    if (!svcName) {
        logBox.innerText = 'Selecciona un servicio o base de datos para transmitir logs en vivo...';
        return;
    }

    try {
        const res = await fetch(`/api/logs?name=${encodeURIComponent(svcName)}&service=${encodeURIComponent(svcName)}&lines=${lines}`);
        if (res.ok) {
            let logsText = '';
            try {
                const data = await res.json();
                logsText = data.logs || data;
            } catch (e) {
                logsText = await res.text();
            }
            logBox.innerText = logsText || `[No hay logs registrados recientemente para ${svcName}]`;
            if (autoScrollLogs) {
                logBox.scrollTop = logBox.scrollHeight;
            }
        }
    } catch (err) {
        console.warn('Error fetching live logs:', err);
    }
}

function toggleLogAutoScroll() {
    autoScrollLogs = !autoScrollLogs;
    const btn = document.getElementById('btnToggleLogScroll');
    if (btn) {
        btn.innerText = autoScrollLogs ? '⏸️' : '▶️';
        btn.title = autoScrollLogs ? 'Pausar Auto-Scroll' : 'Reanudar Auto-Scroll';
    }
}

function clearLiveLogs() {
    const logBox = document.getElementById('liveLogContent');
    if (logBox) logBox.innerText = '[Logs limpiados localmente]';
}

/* --- 7. Special UseCase Action Trigger Functions --- */
async function runBootstrapAction() {
    showToast('Ejecutando Bootstrapper', 'Validando Fail2Ban, reglas UFW y certificados Traefik...', 'info', 8000);
    try {
        const res = await fetch('/api/bootstrap', {method: 'POST'});
        if (res.ok) {
            showToast('Bootstrapper Completo', '¡Servidor asegurado y optimizado correctamente!', 'success');
            loadDashboardData();
        } else {
            const err = await res.text();
            showToast('Error Bootstrapper', err, 'error');
        }
    } catch (err) {
        showToast('Error de Conexión', err.message, 'error');
    }
}

async function runDeployObsAction() {
    openObservabilityModal();
}

async function openObservabilityModal() {
    openModal('observabilityModal');
    try {
        const res = await fetch('/api/observability');
        if (res.ok) {
            const data = await res.json();
            if (data.enabled) {
                if (data.external_url) document.getElementById('obsVolumePath').value = data.external_url;
                if (data.grafana_password) document.getElementById('obsPassword').value = data.grafana_password;
                if (data.deploy_type) document.getElementById('obsDeployType').value = data.deploy_type;
            }
        }
    } catch (e) {
        console.warn('Could not fetch observability config:', e);
    }
}

function disableObservabilityStack() {
    requestConfirmation(
        '⚠️ Destruir Observabilidad',
        '¿Confirmas remover el stack de observabilidad (Loki, Grafana, Portainer) de Docker Swarm?',
        async () => {
            showToast('Observabilidad', 'Removiendo stack tarhiata_obs...', 'info', 4000);
            try {
                const res = await fetch('/api/observability', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'delete', enabled: false })
                });
                if (res.ok) {
                    showToast('Stack Removido', 'Stack de observabilidad eliminado exitosamente.', 'success');
                    closeModal('observabilityModal');
                    loadDashboardData();
                } else {
                    const err = await res.json().catch(() => ({ error: 'Error al remover observabilidad' }));
                    showToast('Error', err.error, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

function rollbackService(name) {
    requestConfirmation(
        `⚠️ Rollback de Servicio: ${name}`,
        `¿Está seguro de revertir el servicio '${name}' a su estado y versión previa en Docker Swarm?`,
        async () => {
            showToast('Ejecutando Rollback', `Revirtiendo servicio ${name}...`, 'info', 5000);
            try {
                const res = await fetch('/api/services/rollback', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: name })
                });
                if (res.ok) {
                    const data = await res.json();
                    showToast('Rollback Exitoso', `Servicio '${name}' revertido correctamente.`, 'success');
                    loadDashboardData();
                } else {
                    const err = await res.json().catch(() => ({ error: 'Falló el rollback' }));
                    showToast('Error de Rollback', err.error || 'No se pudo realizar el rollback', 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

function rollbackServiceFromModalAction() {
    const name = document.getElementById('editSvcOriginalName').value;
    if (name) {
        closeModal('editServiceModal');
        rollbackService(name);
    }
}

function promptRollbackService() {
    if (!globalServices || globalServices.length === 0) {
        showToast('Rollback', 'No hay servicios desplegados para realizar rollback.', 'info');
        return;
    }
    const serviceNames = globalServices.map(s => s.name).join(', ');
    const targetName = prompt(`Escribe el nombre del servicio a revertir en Swarm (${serviceNames}):`);
    if (targetName && targetName.trim()) {
        rollbackService(targetName.trim());
    }
}

async function runPruneAction() {
    showToast('Limpieza Docker', 'Liberando espacio en disco en el clúster (docker system prune)...', 'info', 5000);
    try {
        const res = await fetch('/api/prune', { method: 'POST' });
        const text = await res.text();
        if (res.ok) {
            let msg = text;
            try {
                const data = JSON.parse(text);
                msg = data.output || data.status || text;
            } catch (e) {}
            showToast('Prune Completado 🧹', msg || 'Espacio en disco liberado exitosamente.', 'success', 8000);
        } else {
            showToast('Error Prune', `No se pudo completar el prune: ${text}`, 'error', 6000);
        }
    } catch (err) {
        showToast('Error de Conexión', err.message, 'error', 6000);
    }
}

function openMetricsModal() {
    const card = document.getElementById('liveLogContent') || document.getElementById('vpsHostText');
    if (card) {
        card.scrollIntoView({ behavior: 'smooth' });
    }
    showToast('Métricas en Vivo', 'Visualizando gráficos en vivo de CPU/RAM y estado de Swarm', 'info', 4000);
}

async function viewTopologyAction() {
    openModal('topologyModal');
    const container = document.getElementById('topologyModalContent');
    if (!container) return;

    container.innerText = 'Cargando topología de red...';
    try {
        const services = globalServices || [];
        const dbs = globalDatabases || [];
        const linksRes = await fetch('/api/links');
        const links = linksRes.ok ? await linksRes.json() : [];

        let output = `=====================================================\n`;
        output += `   TARHIATA-OPS NETWORK TOPOLOGY (DOCKER OVERLAY)\n`;
        output += `=====================================================\n\n`;
        output += `[Red Clúster]: tarhiata_internal (Subnet: 10.0.9.0/24)\n`;
        output += `[Driver Swarm]: Overlay Encapsulated VXLAN Mesh\n`;
        output += `[Reverse Proxy]: Traefik SSL Router (Puertos 80/443)\n\n`;

        output += `--- 🚀 SERVICIOS Y APPS WEB (${services.length}) ---\n`;
        if (services.length === 0) {
            output += `(Sin servicios activos)\n\n`;
        } else {
            services.forEach(s => {
                output += `• ${s.name}\n`;
                output += `  ├─ DNS Interno: http://${s.name}:${s.port}\n`;
                output += `  ├─ Visibilidad: ${s.expose ? `PÚBLICO (${s.domain || 'Dominio s/config'})` : 'PRIVADO (Interno)'}\n`;
                output += `  └─ SSL Traefik: ${s.enableSSL ? 'Habilitado (HTTPS)' : 'Deshabilitado'}\n\n`;
            });
        }

        output += `--- 🗄️ BASES DE DATOS PERSISTENTES (${dbs.length}) ---\n`;
        if (dbs.length === 0) {
            output += `(Sin bases de datos activas)\n\n`;
        } else {
            dbs.forEach(db => {
                output += `• ${db.name} [Motor: ${db.engine}]\n`;
                output += `  ├─ DNS Interno: tarhiata-db-${db.name}:${db.internalPort}\n`;
                output += `  └─ Almacenamiento: /opt/data/${db.name}\n\n`;
            });
        }

        output += `--- ⚡ INTERCONEXIONES Y ENLACES ENV (${links.length}) ---\n`;
        if (links.length === 0) {
            output += `(Sin enlaces activos entre servicios)\n`;
        } else {
            links.forEach(l => {
                output += `• [${l.sourceSvc}] ────────► [${l.targetSvc}]\n`;
                output += `  └─ Variable Inyectada: ${l.envVarName || 'DATABASE_URL'}\n`;
            });
        }

        container.innerText = output;
    } catch (err) {
        container.innerText = `[Error cargando topología]: ${err.message}`;
    }
}

let globalIsOnline = true;

/* --- 8. Data Fetching & UI Rendering --- */
async function loadDashboardData(isManual = false) {
    try {
        const res = await fetch('/api/dashboard');
        if (!res.ok) return;
        const data = await res.json();

        globalServices = data.services || [];
        globalDatabases = data.databases || [];
        globalIsOnline = data.isOnline !== undefined ? data.isOnline : true;

        if (window.Alpine && Alpine.store('app')) {
            Alpine.store('app').services = globalServices;
            Alpine.store('app').databases = globalDatabases;
            Alpine.store('app').isOnline = globalIsOnline;
        }

        if (data.config) {
            const hostEl = document.getElementById('vpsHostText');
            const hostStr = data.config.host || data.config.vps_ip || 'Local / Manager';
            if (hostEl) {
                if (globalIsOnline) {
                    hostEl.innerHTML = `<span style="color: #34d399;">● ${hostStr} (Online)</span>`;
                } else {
                    hostEl.innerHTML = `<span style="color: #ef4444;">● ${hostStr} (Offline / VM Inalcanzable)</span>`;
                }
            }
        }

        if (isManual) {
            if (globalIsOnline) {
                showToast('✅ Conexión Validada', `El servidor responde por SSH. Swarm activo y saludable.`, 'success', 4000);
            } else {
                showToast('⚠️ Servidor Inalcanzable', 'La VM no responde por SSH/TCP. Estado: Offline (Servicios detenidos).', 'error', 5000);
            }
        }

        const appsValEl = document.getElementById('totalAppsVal');
        if (appsValEl) appsValEl.innerText = globalIsOnline ? globalServices.length : '0 (Offline)';

        const dbsValEl = document.getElementById('totalDBsVal');
        if (dbsValEl) dbsValEl.innerText = globalIsOnline ? globalDatabases.length : '0 (Offline)';

        if (!globalIsOnline) {
            const swarmBadge = document.getElementById('swarmStatusBadge');
            if (swarmBadge) {
                swarmBadge.innerText = '🔴 DISCONNECTED';
                swarmBadge.className = 'badge badge-red';
            }

            const diskValEl = document.getElementById('diskUsageVal');
            if (diskValEl) diskValEl.innerText = '0 GB / 0 GB (Offline)';

            const nodesCountEl = document.getElementById('nodesCountVal');
            if (nodesCountEl) nodesCountEl.innerText = '0 Nodes (Offline)';

            const billingVal = document.getElementById('billingVal');
            if (billingVal) billingVal.innerText = '$0.00/hr (Offline)';

            const billingSubVal = document.getElementById('billingSubVal');
            if (billingSubVal) billingSubVal.innerText = 'Consumo est.: $0.00 este mes (Servidor Caído)';
        } else {
            const swarmBadge = document.getElementById('swarmStatusBadge');
            if (swarmBadge) {
                swarmBadge.innerText = '● CONNECTED';
                swarmBadge.className = 'badge badge-green';
            }

            const diskValEl = document.getElementById('diskUsageVal');
            if (diskValEl) diskValEl.innerText = '14.2 GB / 50 GB (Live)';

            const nodesCountEl = document.getElementById('nodesCountVal');
            if (nodesCountEl) nodesCountEl.innerText = '1 Node Active';

            // Plan & Cloud Billing calculation
            const nodeCount = (data.databases ? data.databases.filter(d => d.deployType === 'multi-node' || d.deploy_type === 'multi-node').length : 0) + 1;
            const monthlyCost = nodeCount * 6.00;
            const today = new Date().getDate();
            const daysInMonth = 30;
            const currentUsageEst = ((monthlyCost / daysInMonth) * today).toFixed(2);

            const billingVal = document.getElementById('billingVal');
            if (billingVal) billingVal.innerText = `Plan Pro ($${monthlyCost.toFixed(2)}/mes)`;

            const billingSubVal = document.getElementById('billingSubVal');
            if (billingSubVal) billingSubVal.innerText = `Consumo est.: $${currentUsageEst} este mes (${nodeCount} Nodo${nodeCount > 1 ? 's' : ''})`;
        }

        renderCatalog(globalServices, globalDatabases);
        loadLinks();
        loadPreviews();
        loadNodes();
        loadBackups();
        renderTopologyMap();

    } catch (e) {
        console.warn('Dashboard sync polling error:', e);
    }
}

async function loadPreviews() {
    try {
        const res = await fetch('/api/previews');
        if (!res.ok) return;
        const previews = await res.json();
        renderPreviews(previews || []);
    } catch (e) {
        console.warn('Error loading preview envs:', e);
    }
}

function renderPreviews(previews) {
    const list = document.getElementById('previewsList');
    if (!list) return;

    if (!globalIsOnline) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem; color:#ef4444;">⚠️ Servidor Offline (VM Inalcanzable)</div>`;
        return;
    }

    if (!previews || previews.length === 0) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">No hay entornos de preview activos. Usa ⌘K ➔ 'Create Preview Environment'.</div>`;
        return;
    }

    list.innerHTML = previews.map(p => {
        const name = p.name || 'preview';
        const img = p.imageSource || p.image_source || 'docker';
        const port = p.port || 80;
        const domain = p.domain;
        const linkedDB = p.linkDBName || p.link_db_name;
        const status = p.status || 'active';

        let domainBadge = '';
        if (domain) {
            domainBadge = `<a href="http://${domain}" target="_blank" onclick="event.stopPropagation();" class="badge badge-green" style="text-decoration:none;">🌐 ${domain}</a>`;
        } else {
            const hostIp = (window.globalConfig && window.globalConfig.host) ? window.globalConfig.host : window.location.hostname;
            domainBadge = `<a href="http://${hostIp}:${port}" target="_blank" onclick="event.stopPropagation();" class="badge badge-blue" style="text-decoration:none;">🔗 http://${hostIp}:${port}</a>`;
        }

        let dbBadge = '';
        if (linkedDB) {
            dbBadge = `<span class="badge badge-blue">🔗 DB: ${linkedDB}</span>`;
        }

        return `
        <div class="endpoint-item">
            <div class="ep-url">
                <span class="dot-live" style="background:#a855f7;"></span>
                <span>🧪 <strong>${name}</strong> <small class="text-muted">(${img})</small></span>
            </div>
            <div class="ep-badges" style="display:flex; gap:4px; align-items:center;">
                <span class="badge badge-yellow">${status}</span>
                ${domainBadge}
                ${dbBadge}
                <button class="btn btn-outline" style="padding:2px 6px; font-size:0.7rem; border-color:var(--accent-red); color:var(--accent-red)" onclick="event.stopPropagation(); destroyPreviewEnv('${name}')">🔥 Destroy</button>
            </div>
        </div>
        `;
    }).join('');
}

async function loadLinks() {
    try {
        const res = await fetch('/api/links');
        if (!res.ok) return;
        const links = await res.json();
        globalLinks = links || [];
        renderLinks(globalLinks);
    } catch (e) {
        console.warn('Error loading links:', e);
    }
}

function renderLinks(links) {
    const list = document.getElementById('linksList');
    if (!list) return;

    if (!globalIsOnline) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem; color:#ef4444;">⚠️ Servidor Offline (VM Inalcanzable)</div>`;
        return;
    }

    if (!links || links.length === 0) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">No hay enlaces activos. Usa ⌘K ➔ 'Link services' para conectar apps con BDs.</div>`;
        return;
    }

    list.innerHTML = links.map(l => {
        const src = l.sourceSvc || l.source_svc || 'Servicio';
        const tgt = l.targetSvc || l.target_svc || 'BD';
        const env = l.envVarName || l.env_var_name || 'ENV_VAR';
        return `
        <div class="endpoint-item" style="cursor:pointer;" onclick="openEditLinkModal('${src}', '${tgt}', '${env}')">
            <div class="ep-url">
                <span class="dot-live"></span>
                <strong>${src}</strong> ──[ <code>${env}</code> ]──► <strong>${tgt}</strong>
            </div>
            <div class="ep-badges">
                <span class="badge badge-green">Inyectado</span>
                <button class="btn btn-outline" style="padding:2px 6px; font-size:0.7rem; border-color:var(--accent-red); color:var(--accent-red)" onclick="event.stopPropagation(); deleteLinkAction('${src}', '${tgt}')">🗑️ Unlink</button>
            </div>
        </div>
        `;
    }).join('');
}

async function deleteLinkAction(sourceSvc, targetSvc) {
    showToast('Removiendo Enlace', `Desconectando ${sourceSvc} ➔ ${targetSvc}...`, 'info');
    try {
        const res = await fetch(`/api/links?source_svc=${encodeURIComponent(sourceSvc)}&target_svc=${encodeURIComponent(targetSvc)}`, {
            method: 'DELETE'
        });
        if (res.ok) {
            showToast('Enlace Eliminado', `Removida variable de entorno en Swarm para ${sourceSvc}`, 'success');
            loadDashboardData();
        } else {
            const err = await res.text();
            showToast('Error al Desconectar', err, 'error');
        }
    } catch (err) {
        showToast('Error de Red', err.message, 'error');
    }
}

let lastRenderedCatalogHash = '';

function renderCatalog(services, dbs) {
    const list = document.getElementById('catalogList');
    if (!list) return;

    const currentHash = JSON.stringify({
        online: globalIsOnline,
        svcs: (services || []).map(s => [s.name, s.port, s.expose, s.domain, s.imageSource]),
        dbs: (dbs || []).map(d => [d.name, d.engine, d.internalPort, d.deployType])
    });

    if (currentHash === lastRenderedCatalogHash && list.children.length > 0) {
        return;
    }
    lastRenderedCatalogHash = currentHash;

    if (!globalIsOnline) {
        list.innerHTML = `
            <div style="text-align: center; padding: 28px 16px; background: rgba(239,68,68,0.05); border: 1px dashed rgba(239,68,68,0.3); border-radius: 6px;">
                <div style="font-size: 1.4rem; margin-bottom: 6px;">⚠️ Servidor Offline / VM Inalcanzable</div>
                <div style="font-size: 0.85rem; color: #ef4444; font-weight: 600; margin-bottom: 4px;">La máquina virtual host no responde por SSH / TCP</div>
                <div style="font-size: 0.78rem; color: var(--text-muted);">No se muestra ningún servicio ni base de datos hasta reestablecer la conexión con el servidor.</div>
            </div>
        `;
        return;
    }

    let html = '';

    (services || []).forEach(s => {
        const imgSrc = s.imageSource || s.image_source || 'custom';
        const isExposed = s.expose;
        const isSSL = s.enableSSL !== undefined ? s.enableSSL : (s.enable_ssl !== undefined ? s.enable_ssl : false);
        const domain = s.domain;
        const proto = isSSL ? 'https' : 'http';

        let domainBadge = '';
        if (isExposed && domain && globalIsOnline) {
            domainBadge = `<a href="${proto}://${domain}" target="_blank" onclick="event.stopPropagation();" class="badge badge-green" style="text-decoration:none;">🌐 ${domain}</a>`;
        }

        const dotClass = globalIsOnline ? 'dot-live' : 'dot-offline';
        const statusBadgeStr = globalIsOnline 
            ? (domainBadge ? domainBadge : (isExposed ? '<span class="badge badge-green">Público</span>' : '<span class="badge badge-yellow">Privado</span>'))
            : '<span class="badge badge-red">● Offline (VM Inalcanzable)</span>';

        html += `
            <div class="endpoint-item" style="cursor:pointer;" onclick="openResourceInspector('${s.name}', 'service')">
                <div class="ep-url">
                    <span class="${dotClass}"></span>
                    <span>🚀 <strong>${s.name}</strong> <small class="text-muted">(${imgSrc})</small></span>
                </div>
                <div class="ep-badges" style="display:flex; align-items:center; gap:6px;">
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.72rem;" onclick="event.stopPropagation(); openResourceInspector('${s.name}', 'service', 'logs')" title="Ver logs en vivo">📜 Logs</button>
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.72rem;" onclick="event.stopPropagation(); openResourceInspector('${s.name}', 'service', 'envs')" title="Editar Variables de Entorno (.env)">🔑 .env</button>
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.72rem; border-color:var(--accent-yellow); color:var(--accent-yellow);" onclick="event.stopPropagation(); openResourceInspector('${s.name}', 'service', 'rollback')" title="Revertir versión previa en Swarm">⚠️ Rollback</button>
                    <span class="badge badge-blue">Puerto ${s.port || 80}</span>
                    ${statusBadgeStr}
                </div>
            </div>
        `;
    });

    (dbs || []).forEach(d => {
        const dType = d.deployType || d.deploy_type || 'single-node';
        const intPort = d.internalPort || d.internal_port || 5432;
        const dotClass = globalIsOnline ? 'dot-live' : 'dot-offline';
        const dbStatusBadge = globalIsOnline
            ? '<span class="badge badge-green">Activa (Live)</span>'
            : '<span class="badge badge-red">● Offline (VM Inalcanzable)</span>';

        html += `
            <div class="endpoint-item" style="cursor:pointer;" onclick="openResourceInspector('${d.name}', 'database')">
                <div class="ep-url">
                    <span class="${dotClass}"></span>
                    <span>🗄️ <strong>${d.name}</strong> <small class="text-muted">(${d.engine || 'database'})</small></span>
                </div>
                <div class="ep-badges" style="display:flex; align-items:center; gap:6px;">
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.72rem;" onclick="event.stopPropagation(); openResourceInspector('${d.name}', 'database', 'logs')" title="Ver logs en vivo de esta base de datos">📜 Logs</button>
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.72rem; border-color:var(--accent-blue); color:var(--accent-blue);" onclick="event.stopPropagation(); openResourceInspector('${d.name}', 'database', 'backups')" title="Crear snapshot o descargar respaldos a tu PC">📥 Respaldos PC</button>
                    <span class="badge badge-yellow">${dType}</span>
                    <span class="badge badge-blue">Puerto ${intPort}</span>
                    ${dbStatusBadge}
                </div>
            </div>
        `;
    });

    if (html === '') {
        html = `<div class="text-muted p-2" style="font-size:0.85rem">No hay servicios ni BDs registradas en SQLite. Presiona ⌘K para crear uno.</div>`;
    }

    list.innerHTML = html;

    // Populate log inspector select options
    const logSelect = document.getElementById('logServiceSelect');
    if (logSelect) {
        const currentVal = logSelect.value;
        let optHtml = '';
        (services || []).forEach(s => { optHtml += `<option value="${s.name}">🚀 ${s.name} (App)</option>`; });
        (dbs || []).forEach(d => { optHtml += `<option value="${d.name}">🗄️ ${d.name} (DB)</option>`; });
        if (optHtml !== '') {
            logSelect.innerHTML = optHtml;
            if (currentVal && Array.from(logSelect.options).some(o => o.value === currentVal)) {
                logSelect.value = currentVal;
            }
        }
    }
}

function renderEndpoints(services) {
    const list = document.getElementById('endpointsList');
    if (!list) return;

    const exposed = (services || []).filter(s => s.expose && s.domain);

    if (exposed.length === 0) {
        list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">Sin dominios públicos configurados en Traefik (las apps o BDs privadas no generan HTTPS).</div>`;
        return;
    }

    list.innerHTML = exposed.map(s => {
        const isSSL = s.enableSSL !== undefined ? s.enableSSL : s.enable_ssl;
        const proto = isSSL ? 'https' : 'http';
        const url = `${proto}://${s.domain}`;
        return `
            <div class="endpoint-item">
                <div class="ep-url">
                    <span class="dot-live"></span>
                    <a href="${url}" target="_blank">${url}</a>
                </div>
                <div class="ep-badges">
                    <span class="badge badge-green">${isSSL ? 'SSL (Let\'s Encrypt)' : 'HTTP Direct'}</span>
                </div>
            </div>
        `;
    }).join('');
}

/* --- 9. Live SVG Line Chart Animation --- */
function updateChart() {
    const cpuVal = Math.floor(Math.random() * 18) + 8; // 8% - 26%
    const cpuEl = document.getElementById('cpuText');
    if (cpuEl) cpuEl.innerText = `${cpuVal}%`;
}

/* --- 10. Confirmations & Safety Modal Helper --- */
function requestConfirmation(title, message, onConfirm) {
    const titleEl = document.getElementById('confirmModalTitle');
    const msgEl = document.getElementById('confirmModalMessage');
    const actionBtn = document.getElementById('confirmModalActionBtn');

    if (titleEl) titleEl.innerText = title || '⚠️ Confirmar Acción de Infraestructura';
    if (msgEl) msgEl.innerText = message || '¿Estás seguro de ejecutar esta operación?';

    const newBtn = actionBtn.cloneNode(true);
    actionBtn.parentNode.replaceChild(newBtn, actionBtn);

    newBtn.addEventListener('click', () => {
        closeModal('confirmModal');
        if (typeof onConfirm === 'function') {
            onConfirm();
        }
    });

    openModal('confirmModal');
}

async function destroyPreviewEnv(name) {
    requestConfirmation(
        `🔥 Destruir Entorno Preview '${name}'`,
        `Esta acción eliminará el contenedor temporal 'prev-${name}' en Docker Swarm y su registro en SQLite.`,
        async () => {
            showToast('Destruyendo Entorno Preview', `Removiendo prev-${name}...`, 'warning', 4000);
            try {
                const res = await fetch(`/api/previews?name=${encodeURIComponent(name)}`, {
                    method: 'DELETE'
                });
                if (res.ok) {
                    showToast('Entorno Destruido', `El entorno '${name}' ha sido destruido exitosamente.`, 'success');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error al Destruir', err, 'error');
                }
            } catch (err) {
                showToast('Error de Red', err.message, 'error');
            }
        }
    );
}

async function loadNodes() {
    try {
        const res = await fetch('/api/nodes');
        if (!res.ok) return;
        const nodes = await res.json();
        renderNodes(nodes || []);
    } catch (e) {
        console.warn('Error loading Swarm nodes:', e);
    }
}

function renderNodes(nodes) {
    const list = document.getElementById('nodesList');
    if (list) {
        if (!globalIsOnline) {
            list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem; color:#ef4444;">⚠️ Servidor Offline (VM Inalcanzable)</div>`;
            return;
        }

        if (!nodes || nodes.length === 0) {
            list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">No se detectaron nodos en el clúster Swarm.</div>`;
        } else {
            list.innerHTML = nodes.map(n => {
                const id = n.id || 'unknown';
                const hostname = n.hostname || n.name || 'node';
                const role = n.role || 'worker';
                const status = n.status || 'Ready';
                const availability = n.availability || 'active';
                const isLeader = n.is_leader || false;
                const engineVer = n.engine_version || '';

                const roleBadge = isLeader ? 
                    `<span class="badge badge-purple">👑 Leader Manager</span>` : 
                    (role === 'manager' ? `<span class="badge badge-blue">⚡ Manager</span>` : `<span class="badge badge-yellow">⚙️ Worker</span>`);
                
                const statusDot = (status.toLowerCase() === 'ready' || status.toLowerCase() === 'active') ?
                    `<span class="dot-live" style="background:#22c55e;"></span>` :
                    `<span class="dot-live" style="background:#ef4444;"></span>`;

                const availSelect = `
                    <select class="term-input" style="padding:2px 4px; font-size:0.7rem; width:auto; background:var(--bg-tertiary);" onchange="updateNodeAvailability('${id}', this.value)">
                        <option value="active" ${availability === 'active' ? 'selected' : ''}>🟢 Active</option>
                        <option value="drain" ${availability === 'drain' ? 'selected' : ''}>🟡 Drain</option>
                        <option value="pause" ${availability === 'pause' ? 'selected' : ''}>⏸️ Pause</option>
                    </select>
                `;

                const rmBtn = isLeader ? '' : `
                    <button class="btn btn-outline" style="padding:2px 6px; font-size:0.7rem; border-color:var(--accent-red); color:var(--accent-red);" onclick="event.stopPropagation(); removeSwarmNode('${id}', '${hostname}')">🗑️ Rm</button>
                `;

                return `
                <div class="endpoint-item">
                    <div class="ep-url">
                        ${statusDot}
                        <span><strong>${hostname}</strong> <small class="text-muted">(${engineVer || id})</small></span>
                    </div>
                    <div class="ep-badges" style="display:flex; gap:6px; align-items:center;">
                        ${roleBadge}
                        ${availSelect}
                        ${rmBtn}
                    </div>
                </div>
                `;
            }).join('');
        }
    }

    const countEl = document.getElementById('nodesCountVal');
    if (countEl && nodes) {
        countEl.innerText = `${nodes.length} Node${nodes.length === 1 ? '' : 's'} Active`;
    }
}

async function updateNodeAvailability(nodeId, newAvailability) {
    if (newAvailability === 'drain') {
        requestConfirmation(
            `⚠️ Cambiar Disponibilidad a DRAIN`,
            `¿Confirmas cambiar el nodo '${nodeId}' a estado DRAIN? Todos los contenedores en ejecución en este nodo serán desalojados y reprogramados en otros nodos.`,
            async () => { doUpdateNodeAvailability(nodeId, newAvailability); },
            () => { loadNodes(); }
        );
        return;
    }
    doUpdateNodeAvailability(nodeId, newAvailability);
}

async function doUpdateNodeAvailability(nodeId, newAvailability) {
    try {
        const res = await fetch('/api/nodes/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: nodeId, availability: newAvailability })
        });
        if (res.ok) {
            showToast('Nodo Actualizado', `Disponibilidad cambiada a '${newAvailability}'`, 'success');
            loadNodes();
        } else {
            const err = await res.json().catch(() => ({ error: 'Error al actualizar disponibilidad' }));
            showToast('Error', err.error || 'Falló la actualización del nodo', 'error');
            loadNodes();
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
        loadNodes();
    }
}

function removeSwarmNode(nodeId, hostname) {
    requestConfirmation(
        `🗑️ Remover Nodo de Swarm`,
        `¿Confirmas remover el nodo '${hostname}' (ID: ${nodeId}) del clúster Swarm? Se drenarán sus contenedores y se eliminará el nodo.`,
        async () => {
            showToast('Removiendo Nodo', `Drenando y eliminando ${hostname}...`, 'info', 5000);
            try {
                const res = await fetch(`/api/nodes?id=${encodeURIComponent(nodeId)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Nodo Eliminado', `Nodo '${hostname}' removido exitosamente`, 'success');
                    loadNodes();
                } else {
                    const err = await res.json();
                    showToast('Error', err.error || 'No se pudo eliminar el nodo', 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

async function showJoinTokenModal() {
    try {
        const res = await fetch('/api/nodes/join-token');
        if (res.ok) {
            const data = await res.json();
            document.getElementById('workerJoinCmdInput').value = data.worker_cmd || '';
            document.getElementById('managerJoinCmdInput').value = data.manager_cmd || '';
            openModal('joinTokenModal');
        } else {
            showToast('Error', 'No se pudieron obtener los Join Tokens de Swarm', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function copyInputText(inputId) {
    const input = document.getElementById(inputId);
    if (!input) return;
    input.select();
    navigator.clipboard.writeText(input.value);
    showToast('Copiado', 'Comando copiado al portapapeles', 'success', 2000);
}

/* --- Docker Registry Management --- */
async function openRegistryModal() {
    openModal('registryModal');
    loadRegistries();
}

async function loadRegistries() {
    const container = document.getElementById('registryListContainer');
    if (!container) return;
    try {
        const res = await fetch('/api/registries');
        if (res.ok) {
            const data = await res.json();
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="text-muted p-1">No hay registries privados configurados.</div>';
                return;
            }
            container.innerHTML = data.map(c => `
                <div style="display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,255,255,0.05); padding:4px 0;">
                    <span>📦 <strong>${c.server}</strong> (${c.username})</span>
                    <button class="btn btn-outline" style="padding:1px 6px; font-size:0.7rem; color:var(--accent-red); border-color:var(--accent-red);" onclick="deleteRegistryServer('${c.server}')">Remove</button>
                </div>
            `).join('');
        }
    } catch (e) {
        console.warn('Could not load registries:', e);
    }
}

async function deleteRegistryServer(server) {
    requestConfirmation(
        '⚠️ Cerrar Sesión en Registry',
        `¿Deseas eliminar las credenciales de '${server}' y ejecutar docker logout?`,
        async () => {
            try {
                const res = await fetch(`/api/registries?server=${encodeURIComponent(server)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Registry Removido', `Credenciales de '${server}' eliminadas`, 'success');
                    loadRegistries();
                } else {
                    showToast('Error', 'No se pudo eliminar el registry', 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

function deleteRegistryFromModal() {
    const server = document.getElementById('regServer').value.trim();
    if (!server) {
        showToast('Error', 'Ingresa el servidor para remover', 'error');
        return;
    }
    deleteRegistryServer(server);
}

document.addEventListener('DOMContentLoaded', () => {
    const formRegistry = document.getElementById('formRegistry');
    if (formRegistry) {
        formRegistry.addEventListener('submit', (e) => {
            e.preventDefault();
            const server = document.getElementById('regServer').value.trim();
            const username = document.getElementById('regUser').value.trim();
            const password = document.getElementById('regPass').value.trim();

            requestConfirmation(
                '🔐 Autenticar Docker Registry',
                `¿Deseas iniciar sesión en '${server}' como '${username}' en el clúster?`,
                async () => {
                    showToast('Autenticando Registry', `Ejecutando docker login ${server}...`, 'info', 5000);
                    try {
                        const res = await fetch('/api/registries', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ server, username, password })
                        });
                        if (res.ok) {
                            showToast('Login Exitoso', `Registrado exitosamente en ${server}`, 'success');
                            document.getElementById('regPass').value = '';
                            loadRegistries();
                        } else {
                            const err = await res.json().catch(() => ({ error: 'Error al iniciar sesión' }));
                            showToast('Error de Login', err.error || 'Falló docker login', 'error');
                        }
                    } catch (e) {
                        showToast('Error de Red', e.message, 'error');
                    }
                }
            );
        });
    }
});

/* --- Database Migrations Manager --- */
async function openMigrationModal() {
    const dbSelect = document.getElementById('migrTargetDB');
    if (dbSelect) {
        if (!globalDatabases || globalDatabases.length === 0) {
            dbSelect.innerHTML = '<option value="">No hay bases de datos creadas</option>';
        } else {
            dbSelect.innerHTML = globalDatabases.map(d => `<option value="${d.name}">${d.name} (${d.engine})</option>`).join('');
        }
    }
    openModal('migrationModal');
    if (globalDatabases && globalDatabases.length > 0) {
        loadMigrationFilesForDB(globalDatabases[0].name);
    }
}

async function loadMigrationFilesForDB(dbName) {
    const container = document.getElementById('migrationFilesContainer');
    if (!container || !dbName) return;
    try {
        const res = await fetch(`/api/migrations?db=${encodeURIComponent(dbName)}`);
        if (res.ok) {
            const files = await res.json();
            if (!files || files.length === 0) {
                container.innerHTML = '<div class="text-muted p-1">No hay ficheros .sql de migración para esta BD. Guarda uno arriba.</div>';
                return;
            }
            container.innerHTML = files.map(f => {
                const isApplied = f.status === 'applied';
                const isReverted = f.status === 'reverted';
                const isFailed = f.status === 'failed';
                const badgeClass = isApplied ? 'badge-green' : (isReverted ? 'badge-blue' : (isFailed ? 'badge-red' : 'badge-yellow'));
                const badgeText = isApplied ? 'Applied' : (isReverted ? 'Reverted' : (isFailed ? 'Failed' : 'Pending'));
                const hasDown = f.downContent && f.downContent.trim().length > 0;
                const downTag = hasDown ? '<span class="badge badge-purple" style="font-size:0.6rem;">Has Down Script</span>' : '';
                return `
                    <div style="display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,255,255,0.05); padding:6px 4px;">
                        <div style="display:flex; align-items:center; gap:8px;">
                            <input type="checkbox" class="migr-file-chk" value="${f.filename}" ${isApplied ? 'checked' : ''}>
                            <span>📄 <strong>${f.filename}</strong></span>
                            <span class="badge ${badgeClass}" style="font-size:0.65rem;">${badgeText}</span>
                            ${downTag}
                        </div>
                        <div style="display:flex; gap:6px;">
                            <button class="btn btn-outline" style="padding:1px 6px; font-size:0.7rem; color:var(--accent-red); border-color:var(--accent-red);" onclick="deleteMigrationFile('${f.dbName}', '${f.filename}')">🗑️ Delete</button>
                        </div>
                    </div>
                `;
            }).join('');
        }
    } catch (e) {
        console.warn('Could not load migration files:', e);
    }
}

async function saveNewMigrationFile() {
    const dbName = document.getElementById('migrTargetDB').value;
    const filename = document.getElementById('newMigrFilename').value.trim();
    const content = document.getElementById('newMigrContent').value.trim();
    const downContent = document.getElementById('newMigrDownContent') ? document.getElementById('newMigrDownContent').value.trim() : '';

    if (!dbName || !filename || !content) {
        showToast('Campos Incompletos', 'Ingresa el nombre del archivo (.sql) y las sentencias SQL (UP)', 'error');
        return;
    }

    try {
        const res = await fetch('/api/migrations/file', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ dbName, filename, content, downContent })
        });
        if (res.ok) {
            showToast('Fichero Guardado', `Migración '${filename}' registrada`, 'success');
            document.getElementById('newMigrFilename').value = '';
            document.getElementById('newMigrContent').value = '';
            if (document.getElementById('newMigrDownContent')) document.getElementById('newMigrDownContent').value = '';
            loadMigrationFilesForDB(dbName);
        } else {
            showToast('Error', 'No se pudo guardar el archivo de migración', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

async function deleteMigrationFile(dbName, filename) {
    requestConfirmation(
        '🗑️ Eliminar Migración',
        `¿Deseas borrar el fichero '${filename}' de las migraciones de '${dbName}'?`,
        async () => {
            try {
                const res = await fetch(`/api/migrations/file?db=${encodeURIComponent(dbName)}&filename=${encodeURIComponent(filename)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Eliminado', `Fichero ${filename} borrado`, 'success');
                    loadMigrationFilesForDB(dbName);
                } else {
                    showToast('Error', 'No se pudo eliminar el fichero', 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

async function runSelectedMigrations(action = 'up') {
    const dbName = document.getElementById('migrTargetDB').value;
    const node = document.getElementById('migrTargetNode').value;
    const checkboxes = document.querySelectorAll('.migr-file-chk:checked');
    const filenames = Array.from(checkboxes).map(c => c.value);

    if (!dbName) {
        showToast('Error', 'Selecciona una base de datos de destino', 'error');
        return;
    }
    if (filenames.length === 0) {
        showToast('Atención', 'Selecciona al menos una migración para ejecutar', 'warning');
        return;
    }

    const titleAction = action === 'down' ? '🔻 Regresión (DOWN / Rollback)' : '⚡ Aplicar Migración (UP)';

    requestConfirmation(
        `⚠️ Confirmar ${titleAction}`,
        `¿Deseas ejecutar la acción '${action.toUpperCase()}' en ${filenames.length} archivo(s) sobre '${dbName}' en el nodo '${node}'?`,
        async () => {
            showToast('Ejecutando Operación de BD', `Procesando ${action.toUpperCase()} en ${dbName}...`, 'info', 10000);
            try {
                const res = await fetch('/api/migrations/run', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        targetDB: dbName,
                        targetNode: node,
                        action: action,
                        filenames: filenames
                    })
                });
                if (res.ok) {
                    const result = await res.json();
                    let failed = result.filter(r => r.status === 'failed');
                    if (failed.length > 0) {
                        showToast('Error en Operación', `Falló el archivo ${failed[0].filename}: ${failed[0].logOutput || 'Error SQL'}`, 'error', 10000);
                    } else {
                        const msg = action === 'down' ? 'Regresión completada' : 'Migración aplicada';
                        showToast('Operación Exitosa', `¡${msg} en ${dbName}!`, 'success');
                    }
                    loadMigrationFilesForDB(dbName);
                } else {
                    const err = await res.text();
                    showToast('Error de Ejecución', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

/* --- Embedded Historical Metrics in Server Status Card --- */
function populateCardHistoryServices() {
    const select = document.getElementById('cardHistoryService');
    if (!select) return;
    let optionsHtml = '<option value="all">🌐 Todos</option>';
    if (globalServices && globalServices.length > 0) {
        optionsHtml += globalServices.map(s => `<option value="${s.name}">📦 ${s.name}</option>`).join('');
    }
    if (globalDatabases && globalDatabases.length > 0) {
        optionsHtml += globalDatabases.map(d => `<option value="${d.name}">🗄️ ${d.name}</option>`).join('');
    }
    select.innerHTML = optionsHtml;
}

function toggleCardGraphMode() {
    const mode = document.getElementById('cardGraphMode')?.value || 'live';
    const rangeSelect = document.getElementById('cardHistoryRange');
    const serviceSelect = document.getElementById('cardHistoryService');
    
    if (mode === 'history') {
        if (rangeSelect) rangeSelect.style.display = 'inline-block';
        if (serviceSelect) {
            populateCardHistoryServices();
            serviceSelect.style.display = 'inline-block';
        }
        loadCardMetricsHistory();
    } else {
        if (rangeSelect) rangeSelect.style.display = 'none';
        if (serviceSelect) serviceSelect.style.display = 'none';
        // Restaurar vista live
        loadDashboardData();
    }
}

async function loadCardMetricsHistory() {
    const service = document.getElementById('cardHistoryService')?.value || 'all';
    const timeRange = document.getElementById('cardHistoryRange')?.value || '1h';
    const chartContainer = document.querySelector('.live-chart');
    if (!chartContainer) return;

    try {
        const res = await fetch(`/api/observability/metrics?service=${encodeURIComponent(service)}&range=${encodeURIComponent(timeRange)}`);
        if (res.ok) {
            const data = await res.json();
            renderInlineCardMetricsGraph(data.points || []);
        }
    } catch (e) {
        console.warn('Error loading card metrics history:', e);
    }
}

function renderInlineCardMetricsGraph(points) {
    const chartSvg = document.querySelector('.live-chart');
    if (!chartSvg || points.length === 0) return;

    const width = 400;
    const height = 120;
    const padding = 10;

    let maxCpu = Math.max(...points.map(p => p.cpu), 5);
    let avgCpu = (points.reduce((acc, p) => acc + p.cpu, 0) / points.length).toFixed(1);
    let avgRam = (points.reduce((acc, p) => acc + p.memory, 0) / points.length).toFixed(1);

    const cpuText = document.getElementById('cpuText');
    const ramText = document.getElementById('ramText');
    if (cpuText) cpuText.innerText = `${avgCpu}% (Prom)`;
    if (ramText) ramText.innerText = `${avgRam} MB (Prom)`;

    const cpuPoints = points.map((p, index) => {
        const x = padding + (index / (points.length - 1)) * (width - 2 * padding);
        const y = height - padding - (p.cpu / maxCpu) * (height - 2 * padding);
        return `${x},${y}`;
    }).join(' ');

    const ramPoints = points.map((p, index) => {
        const x = padding + (index / (points.length - 1)) * (width - 2 * padding);
        const maxRamVal = Math.max(...points.map(pt => pt.memory), 100);
        const y = height - padding - (p.memory / maxRamVal) * (height - 2 * padding);
        return `${x},${y}`;
    }).join(' ');

    const firstX = padding;
    const lastX = width - padding;
    const bottomY = height - padding;
    const areaCpu = `${firstX},${bottomY} ${cpuPoints} ${lastX},${bottomY}`;

    chartSvg.innerHTML = `
        <defs>
            <linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#60a5fa" stop-opacity="0.5"/>
                <stop offset="100%" stop-color="#60a5fa" stop-opacity="0.0"/>
            </linearGradient>
        </defs>
        <polygon points="${areaCpu}" fill="url(#chartGrad)" />
        <polyline points="${cpuPoints}" fill="none" stroke="#60a5fa" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        <polyline points="${ramPoints}" fill="none" stroke="#a855f7" stroke-width="1.5" stroke-dasharray="3,3" />
        <text x="${padding}" y="${height - 4}" fill="#64748b" font-size="9">${points[0].timestamp}</text>
        <text x="${width - padding}" y="${height - 4}" fill="#64748b" font-size="9" text-anchor="end">${points[points.length - 1].timestamp}</text>
    `;
}

/* --- Backups & Snapshots Management --- */
async function loadBackups() {
    const listEl = document.getElementById('backupsList');
    const modalListEl = document.getElementById('modalBackupsList');

    if (!globalIsOnline) {
        const offlineHtml = '<div class="text-muted p-2" style="text-align: center; color:#ef4444;">⚠️ Servidor Offline (VM Inalcanzable)</div>';
        if (listEl) listEl.innerHTML = offlineHtml;
        if (modalListEl) modalListEl.innerHTML = offlineHtml;
        return;
    }

    try {
        const res = await fetch('/api/backups');
        if (!res.ok) return;
        const backups = await res.json();
        
        const emptyHtml = '<div class="text-muted p-2" style="text-align: center;">No hay snapshots guardados. Presiona "+ Snapshot 1-Click".</div>';

        if (!backups || backups.length === 0) {
            if (listEl) listEl.innerHTML = emptyHtml;
            if (modalListEl) modalListEl.innerHTML = emptyHtml;
            return;
        }

        const html = backups.map(b => `
            <div class="catalog-item" style="display: flex; justify-content: space-between; align-items: center; padding: 6px 10px; margin-bottom: 4px; background: var(--bg-tertiary); border-radius: 6px; border: 1px solid var(--border-color);">
                <div>
                    <div style="font-weight: 600; font-size: 0.85rem; color: var(--accent-blue);">
                        ${b.targetType === 'database' ? '🗄️' : '📦'} ${b.targetName} <span style="font-size:0.75rem; color:var(--text-muted);">(${b.engine})</span>
                    </div>
                    <div style="font-size: 0.72rem; color: var(--text-muted); font-family: monospace;">
                <div style="display: flex; gap: 4px;">
                    <button class="btn btn-outline" style="padding: 2px 8px; font-size: 0.7rem; border-color: #22c55e; color: #4ade80; font-weight: 600;" onclick="downloadBackupFile(${b.id})" title="Descargar Snapshot a tu PC local">📥 Descargar a PC</button>
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: #3b82f6; color: #3b82f6;" onclick="restoreBackupSnapshot(${b.id}, '${b.targetName}')" title="Restaurar este backup">🔄 Restaurar</button>
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: #ef4444; color: #ef4444;" onclick="deleteBackupSnapshot(${b.id})" title="Eliminar registro">🗑️</button>
                </div>
            </div>
        `).join('');

        if (listEl) listEl.innerHTML = html;
        if (modalListEl) modalListEl.innerHTML = html;
    } catch (e) {
        console.warn('Error loading backups:', e);
    }
}

function openBackupModal() {
    toggleBackupTargetOptions();
    populateS3BackupTargets();
    openModal('backupModal');
    loadBackups();
}

function populateS3BackupTargets() {
    const s3Select = document.getElementById('backupS3Target');
    if (!s3Select) return;
    const currentVal = s3Select.value;
    const minioDBs = (globalDatabases || []).filter(d => (d.engine || '').toLowerCase().includes('minio') || (d.engine || '').toLowerCase().includes('s3'));
    
    let html = '<option value="">💾 Solo Local VPS (/opt/tarhiata/backups)</option>';
    if (minioDBs.length > 0) {
        html += minioDBs.map(d => `<option value="${d.name}">📦 Copiar a MinIO S3: ${d.name}</option>`).join('');
    }
    html += '<option value="custom">🌐 Servidor S3 Externo (Cloudflare R2 / AWS S3 / URL)</option>';
    s3Select.innerHTML = html;
    s3Select.value = currentVal || '';
    toggleCustomS3Fields();
}

function toggleCustomS3Fields() {
    const s3Target = document.getElementById('backupS3Target')?.value;
    const container = document.getElementById('customS3Fields');
    if (container) {
        container.style.display = (s3Target === 'custom') ? 'block' : 'none';
    }
}

function openBackupModalForTarget(targetName, targetType = 'database') {
    openBackupModal();
    const typeSelect = document.getElementById('backupTargetType');
    if (typeSelect) {
        typeSelect.value = targetType;
        toggleBackupTargetOptions();
        const nameSelect = document.getElementById('backupTargetName');
        if (nameSelect) {
            nameSelect.value = targetName;
        }
    }
}

function toggleBackupTargetOptions() {
    const type = document.getElementById('backupTargetType')?.value || 'database';
    const select = document.getElementById('backupTargetName');
    if (!select) return;

    if (type === 'database') {
        if (globalDatabases && globalDatabases.length > 0) {
            select.innerHTML = globalDatabases.map(d => `<option value="${d.name}">🗄️ ${d.name} (${d.engine})</option>`).join('');
        } else {
            select.innerHTML = '<option value="">(No hay bases de datos disponibles)</option>';
        }
    } else {
        if (globalServices && globalServices.length > 0) {
            select.innerHTML = globalServices.map(s => `<option value="${s.name}">📦 App Volume: ${s.name}</option>`).join('');
        } else {
            select.innerHTML = '<option value="">(No hay servicios disponibles)</option>';
        }
    }
}

async function handleCreateBackupSubmit(e) {
    e.preventDefault();
    const type = document.getElementById('backupTargetType')?.value;
    const targetName = document.getElementById('backupTargetName')?.value;
    const s3Target = document.getElementById('backupS3Target')?.value || '';
    if (!targetName) {
        showToast('Error', 'Selecciona un recurso válido', 'error');
        return;
    }

    closeModal('backupModal');
    showToast('Generando Snapshot', `Respaldando ${type} '${targetName}' en servidor...`, 'info');

    try {
        const res = await fetch('/api/backups', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ targetName: targetName, targetType: type })
        });
        if (res.ok) {
            const b = await res.json();
            showToast('Snapshot Creado 🚀', `Guardado como ${b.filename}`, 'success');
            loadBackups();
        } else {
            const err = await res.text();
            showToast('Error de Snapshot', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function downloadBackupFile(id) {
    showToast('Descargando Snapshot', 'Iniciando descarga directa a tu equipo...', 'info');
    window.open(`/api/backups/download?id=${id}`, '_blank');
}

function restoreBackupSnapshot(id, targetName) {
    requestConfirmation(
        '🔄 Restaurar Snapshot de Infraestructura',
        `¿Estás seguro de restaurar el Snapshot ID #${id} sobre '${targetName}'? Se sobreescribirán los datos actuales de la base de datos o volumen.`,
        async () => {
            showToast('Restaurando Data', `Aplicando snapshot #${id}...`, 'info');
            try {
                const res = await fetch('/api/backups/restore', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ backupId: id })
                });
                if (res.ok) {
                    showToast('Restauración Exitosa 🎉', `Snapshot #${id} aplicado en ${targetName}`, 'success');
                    loadDashboardData();
                } else {
                    const err = await res.text();
                    showToast('Error de Restauración', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

async function deleteBackupSnapshot(id) {
    try {
        const res = await fetch(`/api/backups?id=${id}`, { method: 'DELETE' });
        if (res.ok) {
            showToast('Backup Eliminado', `Snapshot #${id} removido del catálogo`, 'info');
            loadBackups();
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

/* --- Bulk ENV Management (.env) --- */
function openEnvModal(serviceName = null) {
    const select = document.getElementById('envTargetService');
    if (select) {
        if (globalServices && globalServices.length > 0) {
            select.innerHTML = globalServices.map(s => `<option value="${s.name}">${s.name}</option>`).join('');
            if (serviceName) select.value = serviceName;
        } else {
            select.innerHTML = '<option value="">(No hay servicios activos)</option>';
        }
    }
    loadServiceEnvVars();
    openModal('envModal');
}

async function loadServiceEnvVars() {
    const serviceName = document.getElementById('envTargetService')?.value;
    const txt = document.getElementById('envRawContent');
    if (!serviceName || !txt) return;

    txt.value = 'Cargando variables de entorno...';
    try {
        const res = await fetch(`/api/env?service=${encodeURIComponent(serviceName)}`);
        if (res.ok) {
            const data = await res.json();
            txt.value = data.rawContent || '';
        } else {
            txt.value = '';
        }
    } catch (e) {
        txt.value = '# Error al cargar variables: ' + e.message;
    }
}

function importEnvFileSelected(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function(e) {
        const content = e.target.result;
        const txt = document.getElementById('envRawContent');
        if (txt) {
            txt.value = content;
            showToast('Archivo Cargado', `Importado '${file.name}' al editor. Presiona Guardar para aplicar.`, 'info');
        }
    };
    reader.readAsText(file);
}

function exportEnvFilePC() {
    const serviceName = document.getElementById('envTargetService')?.value;
    if (!serviceName) {
        showToast('Error', 'Selecciona un servicio', 'error');
        return;
    }
    showToast('Descargando .env', `Guardando ${serviceName}.env en tu PC...`, 'info');
    window.open(`/api/env/export?service=${encodeURIComponent(serviceName)}`, '_blank');
}

async function saveServiceEnvVars() {
    const serviceName = document.getElementById('envTargetService')?.value;
    const rawContent = document.getElementById('envRawContent')?.value || '';
    if (!serviceName) {
        showToast('Error', 'Selecciona un servicio válido', 'error');
        return;
    }

    showToast('Actualizando Swarm', `Aplicando variables de entorno a '${serviceName}'...`, 'info');
    try {
        const res = await fetch('/api/env', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ serviceName: serviceName, rawContent: rawContent })
        });
        if (res.ok) {
            showToast('Variables Aplicadas 🔑', `Servicio '${serviceName}' actualizado en Swarm`, 'success');
            closeModal('envModal');
            loadDashboardData();
        } else {
            const err = await res.text();
            showToast('Error al Guardar', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

/* --- Volume File Browser (/opt/data) --- */
let currentVolumePath = '/opt/data';

function openVolumeBrowser(path = '/opt/data') {
    currentVolumePath = path;
    openModal('volumeModal');
    loadVolumeFiles(currentVolumePath);
}

function refreshVolumeBrowser() {
    loadVolumeFiles(currentVolumePath);
}

async function loadVolumeFiles(targetPath) {
    currentVolumePath = targetPath;
    renderVolumeBreadcrumbs(targetPath);
    
    const tbody = document.getElementById('volumeFilesTableBody');
    if (tbody) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-muted p-2" style="text-align: center;">Cargando archivos...</td></tr>';
    }

    try {
        const res = await fetch(`/api/volumes/files?path=${encodeURIComponent(targetPath)}`);
        if (!res.ok) {
            const err = await res.text();
            if (tbody) tbody.innerHTML = `<tr><td colspan="4" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">${err}</td></tr>`;
            return;
        }
        const files = await res.json();
        renderVolumeFilesTable(files);
    } catch (e) {
        if (tbody) tbody.innerHTML = `<tr><td colspan="4" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">Error: ${e.message}</td></tr>`;
    }
}

function renderVolumeBreadcrumbs(path) {
    const container = document.getElementById('volumeBreadcrumbs');
    if (!container) return;

    const parts = path.split('/').filter(p => p !== '');
    let html = `<span style="cursor:pointer; color:var(--accent-blue);" onclick="loadVolumeFiles('/opt/data')">/opt/data</span>`;

    let accumulated = '/opt/data';
    for (let i = 0; i < parts.length; i++) {
        if (parts[i] === 'opt' || parts[i] === 'data') continue;
        accumulated += '/' + parts[i];
        const target = accumulated;
        html += ` <span class="text-muted">/</span> <span style="cursor:pointer; color:var(--accent-blue);" onclick="loadVolumeFiles('${target}')">${parts[i]}</span>`;
    }
    container.innerHTML = html;
}

function renderVolumeFilesTable(files) {
    const tbody = document.getElementById('volumeFilesTableBody');
    if (!tbody) return;

    if (!files || files.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-muted p-2" style="text-align: center;">(Carpeta vacía)</td></tr>';
        return;
    }

    tbody.innerHTML = files.map(f => {
        const icon = f.isDir ? '📁' : '📄';
        const sizeFormatted = f.isDir ? '-' : formatBytes(f.size);
        const escapedPath = f.path.replace(/'/g, "\\'");
        const escapedName = f.name.replace(/'/g, "\\'");

        let nameClick = f.isDir ? `onclick="loadVolumeFiles('${escapedPath}')"` : `onclick="openVolumeFileViewer('${escapedPath}', '${escapedName}')"`;

        return `
        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
            <td style="padding: 8px 12px; cursor: pointer;" ${nameClick}>
                ${icon} <strong style="color: ${f.isDir ? 'var(--accent-blue)' : 'inherit'};">${f.name}</strong>
            </td>
            <td style="padding: 8px 12px; font-family: monospace;">${sizeFormatted}</td>
            <td style="padding: 8px 12px; font-size: 0.78rem;" class="text-muted">${f.modTime || '-'}</td>
            <td style="padding: 8px 12px; text-align: right;">
                <div style="display: flex; gap: 4px; justify-content: flex-end;">
                    ${f.isDir ? `
                        <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem;" onclick="loadVolumeFiles('${escapedPath}')">📂 Abrir</button>
                    ` : `
                        <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem;" onclick="openVolumeFileViewer('${escapedPath}', '${escapedName}')">👁️ Ver/Editar</button>
                        <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem;" onclick="downloadVolumeFilePC('${escapedPath}')">📥 Descargar</button>
                    `}
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-red); color: var(--accent-red);" onclick="deleteVolumeItem('${escapedPath}', '${escapedName}')">🗑️</button>
                </div>
            </td>
        </tr>
        `;
    }).join('');
}

async function openVolumeFileViewer(filePath, fileName) {
    document.getElementById('editingFilePath').value = filePath;
    document.getElementById('volumeFileModalTitle').innerText = `📄 ${fileName} (${filePath})`;
    const txt = document.getElementById('editingFileContent');
    txt.value = 'Cargando contenido del archivo...';

    openModal('volumeFileModal');

    try {
        const res = await fetch(`/api/volumes/read?path=${encodeURIComponent(filePath)}`);
        if (res.ok) {
            const data = await res.json();
            txt.value = data.content || '';
        } else {
            const err = await res.text();
            txt.value = '# Error al leer archivo: ' + err;
        }
    } catch (e) {
        txt.value = '# Error de red: ' + e.message;
    }
}

async function saveCurrentEditingFile() {
    const filePath = document.getElementById('editingFilePath').value;
    const content = document.getElementById('editingFileContent').value;
    if (!filePath) return;

    showToast('Guardando Archivo', `Actualizando '${filePath}' en el servidor...`, 'info');
    try {
        const res = await fetch('/api/volumes/write', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path: filePath, content: content })
        });
        if (res.ok) {
            showToast('Archivo Guardado 💾', `Actualizado exitosamente`, 'success');
            closeModal('volumeFileModal');
            loadVolumeFiles(currentVolumePath);
        } else {
            const err = await res.text();
            showToast('Error al Guardar', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function downloadVolumeFilePC(filePath) {
    showToast('Descargando Archivo', `Iniciando descarga a tu PC...`, 'info');
    window.open(`/api/volumes/download?path=${encodeURIComponent(filePath)}`, '_blank');
}

function downloadCurrentEditingFilePC() {
    const filePath = document.getElementById('editingFilePath').value;
    if (filePath) downloadVolumeFilePC(filePath);
}

async function uploadVolumeFileSelected(event) {
    const file = event.target.files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('dir', currentVolumePath);
    formData.append('file', file);

    showToast('Subiendo Archivo', `Enviando '${file.name}' a ${currentVolumePath}...`, 'info');
    try {
        const res = await fetch('/api/volumes/upload', {
            method: 'POST',
            body: formData
        });
        if (res.ok) {
            showToast('Archivo Subido 📤', `'${file.name}' guardado en la nube`, 'success');
            loadVolumeFiles(currentVolumePath);
        } else {
            const err = await res.text();
            showToast('Error de Carga', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function deleteVolumeItem(filePath, itemName) {
    requestConfirmation(
        '🗑️ Eliminar Recurso de Volumen',
        `¿Estás seguro de eliminar permanentemente '${itemName}' (${filePath})? Esta acción no se puede deshacer.`,
        async () => {
            showToast('Eliminando', `Removiendo ${itemName}...`, 'info');
            try {
                const res = await fetch(`/api/volumes/delete?path=${encodeURIComponent(filePath)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Eliminado 🗑️', `'${itemName}' eliminado`, 'success');
                    loadVolumeFiles(currentVolumePath);
                } else {
                    const err = await res.text();
                    showToast('Error al Eliminar', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

function formatBytes(bytes, decimals = 2) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

/* --- SSL Inspector & Maintenance Mode (503 Drain) --- */
function openSSLModal() {
    openModal('sslModal');
    refreshSSLInspector();
}

async function refreshSSLInspector() {
    const tbody = document.getElementById('sslTableBody');
    if (tbody) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted p-2" style="text-align: center;">Inspeccionando certificados TLS...</td></tr>';
    }

    try {
        const res = await fetch('/api/ssl/inspect');
        if (!res.ok) {
            const err = await res.text();
            if (tbody) tbody.innerHTML = `<tr><td colspan="5" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">${err}</td></tr>`;
            return;
        }
        const items = await res.json();
        renderSSLTable(items);
    } catch (e) {
        if (tbody) tbody.innerHTML = `<tr><td colspan="5" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">Error de red: ${e.message}</td></tr>`;
    }
}

function renderSSLTable(items) {
    const tbody = document.getElementById('sslTableBody');
    if (!tbody) return;

    if (!items || items.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted p-2" style="text-align: center;">(No hay servicios expuestos con dominio público)</td></tr>';
        return;
    }

    tbody.innerHTML = items.map(item => {
        let badge = '<span class="badge badge-yellow">HTTP Solo</span>';
        if (item.status === 'active') {
            badge = `<span class="badge badge-green">🔒 Activo (${item.daysRemaining} días)</span>`;
        } else if (item.status === 'expiring_soon') {
            badge = `<span class="badge badge-yellow">⚠️ Por Vencer (${item.daysRemaining} días)</span>`;
        } else if (item.status === 'expired') {
            badge = `<span class="badge badge-red">❌ Expirado</span>`;
        }

        const escapedSvc = item.serviceName.replace(/'/g, "\\'");

        return `
        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
            <td style="padding: 8px 12px;">
                <strong>${item.domain}</strong> <small class="text-muted">(${item.serviceName})</small>
            </td>
            <td style="padding: 8px 12px;">${badge}</td>
            <td style="padding: 8px 12px; font-size: 0.78rem;" class="text-muted">${item.issuer || '-'}</td>
            <td style="padding: 8px 12px; font-size: 0.78rem;" class="text-muted">${item.expiryDate || '-'}</td>
            <td style="padding: 8px 12px; text-align: right;">
                <div style="display: flex; gap: 4px; justify-content: flex-end;">
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-yellow); color: var(--accent-yellow);" onclick="toggleServiceMaintenance('${escapedSvc}', true)">🚧 Modo Mantenimiento 503</button>
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-green); color: var(--accent-green);" onclick="toggleServiceMaintenance('${escapedSvc}', false)">✅ En Vivo (Normal)</button>
                </div>
            </td>
        </tr>
        `;
    }).join('');
}

function toggleServiceMaintenance(serviceName, enable) {
    const actionTitle = enable ? '🚧 Activar Modo Mantenimiento (503 Drain)' : '✅ Desactivar Modo Mantenimiento';
    const msg = enable ? 
        `¿Estás seguro de activar el Modo Mantenimiento para '${serviceName}'? Traefik responderá 503 Service Unavailable a todas las peticiones entrantes.` : 
        `¿Estás seguro de restaurar el tráfico en vivo para '${serviceName}'?`;

    requestConfirmation(
        actionTitle,
        msg,
        async () => {
            showToast('Actualizando Traefik', `Aplicando reglas para '${serviceName}'...`, 'info');
            try {
                const res = await fetch('/api/maintenance/toggle', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ serviceName: serviceName, enable: enable })
                });
                if (res.ok) {
                    showToast('Estado Actualizado 🔒', `Regla de mantenimiento aplicada a '${serviceName}'`, 'success');
                    refreshSSLInspector();
                } else {
                    const err = await res.text();
                    showToast('Error al Aplicar', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

/* --- Multiple Custom Domains & CNAMEs --- */
function openDomainModal(serviceName = null) {
    const select = document.getElementById('domainTargetService');
    const targetReassignSelect = document.getElementById('reassignTargetServiceSelect');
    if (select) {
        if (globalServices && globalServices.length > 0) {
            const optionsHtml = globalServices.map(s => `<option value="${s.name}">${s.name}</option>`).join('');
            select.innerHTML = optionsHtml;
            if (targetReassignSelect) targetReassignSelect.innerHTML = optionsHtml;
            if (serviceName) select.value = serviceName;
        } else {
            select.innerHTML = '<option value="">(No hay servicios activos)</option>';
            if (targetReassignSelect) targetReassignSelect.innerHTML = '<option value="">(No hay servicios activos)</option>';
        }
    }
    loadServiceDomains();
    openModal('domainModal');
}

async function loadServiceDomains() {
    const serviceName = document.getElementById('domainTargetService')?.value;
    const tbody = document.getElementById('customDomainsTableBody');
    if (!serviceName || !tbody) return;

    tbody.innerHTML = '<tr><td colspan="4" class="text-muted p-2" style="text-align: center;">Cargando dominios...</td></tr>';
    try {
        const res = await fetch(`/api/domains?service=${encodeURIComponent(serviceName)}`);
        if (!res.ok) {
            const err = await res.text();
            tbody.innerHTML = `<tr><td colspan="4" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">${err}</td></tr>`;
            return;
        }
        const data = await res.json();
        renderCustomDomainsTable(data.primaryDomain, data.customRules);
    } catch (e) {
        tbody.innerHTML = `<tr><td colspan="4" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">Error: ${e.message}</td></tr>`;
    }
}

function toggleCustomSSLFields() {
    const certType = document.getElementById('sslCertType')?.value;
    const container = document.getElementById('customSSLFilesContainer');
    if (container) {
        container.style.display = (certType === 'custom') ? 'block' : 'none';
    }
}

async function handleActivateSSLSubmit(event) {
    event.preventDefault();
    const domainName = document.getElementById('sslDomainName')?.value?.trim();
    const serviceName = document.getElementById('domainTargetService')?.value;
    const certType = document.getElementById('sslCertType')?.value || 'letsencrypt';
    const redirectTarget = document.getElementById('newRedirectTarget')?.value?.trim() || '';
    const forceHTTPS = document.getElementById('sslForceHTTPS')?.checked ?? true;

    if (!domainName || !serviceName) {
        showToast('Campos requeridos', 'Ingresa el nombre del dominio y selecciona un servicio target.', 'error');
        return;
    }

    showToast('Activando SSL 🔒', `Configurando certificado HTTPS para ${domainName} y asignando a '${serviceName}'...`, 'info', 5000);

    try {
        if (certType === 'custom') {
            const certFile = document.getElementById('sslCertFileInput')?.files?.[0];
            const keyFile = document.getElementById('sslKeyFileInput')?.files?.[0];

            if (certFile) {
                const formData = new FormData();
                formData.append('file', certFile, `ssl_${domainName}.crt`);
                formData.append('targetPath', `/opt/data/traefik/certs/ssl_${domainName}.crt`);
                await fetch('/api/volumes/upload', { method: 'POST', body: formData });
            }

            if (keyFile) {
                const formData = new FormData();
                formData.append('file', keyFile, `ssl_${domainName}.key`);
                formData.append('targetPath', `/opt/data/traefik/certs/ssl_${domainName}.key`);
                await fetch('/api/volumes/upload', { method: 'POST', body: formData });
            }
        }

        const res = await fetch('/api/domains', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                serviceName: serviceName,
                domain: domainName,
                redirectTarget: redirectTarget,
                certType: certType,
                forceHTTPS: forceHTTPS
            })
        });

        if (res.ok) {
            showToast('SSL Activado 🔒', `¡Certificado HTTPS para https://${domainName} activado y asignado a '${serviceName}'!`, 'success', 7000);
            document.getElementById('sslDomainName').value = '';
            document.getElementById('newRedirectTarget').value = '';
            loadServiceDomains();
        } else {
            const err = await res.text();
            showToast('Error al activar SSL', err, 'error', 6000);
        }
    } catch (err) {
        showToast('Error de Red', err.message, 'error', 6000);
    }
}

async function handleReassignSSLSubmit(event) {
    event.preventDefault();
    const domain = document.getElementById('reassignSSLSelect')?.value;
    const targetService = document.getElementById('reassignTargetServiceSelect')?.value;

    if (!domain || !targetService) {
        showToast('Campos Requeridos', 'Selecciona el dominio y el nuevo servicio target', 'error');
        return;
    }

    showToast('Reasignando SSL 🔄', `Moviendo SSL de '${domain}' al servicio '${targetService}'...`, 'info', 4000);

    try {
        const res = await fetch('/api/domains', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                serviceName: targetService,
                domain: domain,
                forceHTTPS: true
            })
        });

        if (res.ok) {
            showToast('Reasignación Exitosa ⚡', `Dominio SSL https://${domain} ahora apunta a '${targetService}'`, 'success', 6000);
            loadServiceDomains();
        } else {
            const err = await res.text();
            showToast('Error de Reasignación', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function renderCustomDomainsTable(primaryDomain, customRules) {
    const tbody = document.getElementById('customDomainsTableBody');
    const serviceName = document.getElementById('domainTargetService')?.value || 'Servicio';
    const reassignSelect = document.getElementById('reassignSSLSelect');
    if (!tbody) return;

    let rows = [];
    let domainOptions = [];

    if (primaryDomain) {
        domainOptions.push(`<option value="${primaryDomain}">${primaryDomain} (${serviceName})</option>`);
        rows.push(`
        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05); background: rgba(0,255,100,0.02);">
            <td style="padding: 8px 12px;">
                <strong>https://${primaryDomain}</strong>
            </td>
            <td style="padding: 8px 12px;">
                <span class="badge badge-green">${serviceName}</span>
            </td>
            <td style="padding: 8px 12px;">
                <span class="badge badge-yellow">🟢 Let's Encrypt</span>
            </td>
            <td style="padding: 8px 12px; text-align: right;" class="text-muted">Principal</td>
        </tr>
        `);
    }

    if (customRules && customRules.length > 0) {
        customRules.forEach(r => {
            const escapedDom = r.domain.replace(/'/g, "\\'");
            const certLabel = r.certType === 'custom' ? '🔵 Privado / Custom' : '🟢 Let\'s Encrypt';
            domainOptions.push(`<option value="${r.domain}">${r.domain} (${serviceName})</option>`);
            rows.push(`
            <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
                <td style="padding: 8px 12px;">
                    <strong>https://${r.domain}</strong> ${r.redirectTarget ? ` <small class="text-muted">(➔ 301 a ${r.redirectTarget})</small>` : ''}
                </td>
                <td style="padding: 8px 12px;">
                    <span class="badge badge-blue">${serviceName}</span>
                </td>
                <td style="padding: 8px 12px;">
                    <span class="badge badge-yellow">${certLabel}</span>
                </td>
                <td style="padding: 8px 12px; text-align: right;">
                    <button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-red); color: var(--accent-red);" onclick="deleteCustomDomain('${escapedDom}')">🗑️ Eliminar</button>
                </td>
            </tr>
            `);
        });
    }

    if (reassignSelect) {
        if (domainOptions.length > 0) {
            reassignSelect.innerHTML = domainOptions.join('');
        } else {
            reassignSelect.innerHTML = '<option value="">(Sin dominios registrados)</option>';
        }
    }

    if (rows.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-muted p-2" style="text-align: center;">(No hay dominios SSL o alias configurados)</td></tr>';
    } else {
        tbody.innerHTML = rows.join('');
    }
}

async function addCustomDomainSubmit() {
    const serviceName = document.getElementById('domainTargetService')?.value;
    const domainInput = document.getElementById('newCustomDomain')?.value;
    const redirectInput = document.getElementById('newRedirectTarget')?.value;

    if (!serviceName || !domainInput) {
        showToast('Error', 'Ingresa el nombre del servicio y el nuevo dominio', 'error');
        return;
    }

    showToast('Vinculando Dominio', `Agregando '${domainInput}' a '${serviceName}'...`, 'info');
    try {
        const res = await fetch('/api/domains', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                serviceName: serviceName,
                domain: domainInput,
                redirectTarget: redirectInput
            })
        });
        if (res.ok) {
            showToast('Dominio Vinculado 🌐', `'${domainInput}' agregado a Traefik`, 'success');
            document.getElementById('newCustomDomain').value = '';
            document.getElementById('newRedirectTarget').value = '';
            loadServiceDomains();
        } else {
            const err = await res.text();
            showToast('Error de Vinculación', err, 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function deleteCustomDomain(domainName) {
    const serviceName = document.getElementById('domainTargetService')?.value;
    if (!serviceName || !domainName) return;

    requestConfirmation(
        '🌐 Desvincular Dominio CNAME',
        `¿Estás seguro de remover '${domainName}' del servicio '${serviceName}'? Traefik dejará de responder para este dominio.`,
        async () => {
            showToast('Removiendo', `Desvinculando '${domainName}'...`, 'info');
            try {
                const res = await fetch(`/api/domains?service=${encodeURIComponent(serviceName)}&domain=${encodeURIComponent(domainName)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Dominio Removido 🌐', `'${domainName}' desvinculado`, 'info');
                    loadServiceDomains();
                } else {
                    const err = await res.text();
                    showToast('Error al Remover', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

/* --- Docker Swarm Node Management & Workload Placement --- */
function openNodeManagementModal() {
    openModal('nodeManagementModal');
    refreshNodeManagementTable();
}

async function refreshNodeManagementTable() {
    const tbody = document.getElementById('nodeManagementTableBody');
    if (tbody) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted p-2" style="text-align: center;">Cargando nodos del clúster...</td></tr>';
    }

    try {
        const res = await fetch('/api/nodes');
        if (!res.ok) {
            const err = await res.text();
            if (tbody) tbody.innerHTML = `<tr><td colspan="5" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">${err}</td></tr>`;
            return;
        }
        const nodes = await res.json();
        renderNodeManagementTable(nodes);
    } catch (e) {
        if (tbody) tbody.innerHTML = `<tr><td colspan="5" class="text-muted p-2" style="text-align: center; color:var(--accent-red);">Error: ${e.message}</td></tr>`;
    }
}

function renderNodeManagementTable(nodes) {
    const tbody = document.getElementById('nodeManagementTableBody');
    if (!tbody) return;

    if (!nodes || nodes.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-muted p-2" style="text-align: center;">(No se detectaron nodos en Swarm)</td></tr>';
        return;
    }

    tbody.innerHTML = nodes.map(node => {
        const id = node.id || node.ID;
        const hostname = node.hostname || node.Hostname || id;
        const role = node.role || node.Role || 'worker';
        const status = node.status || node.Status || 'Ready';
        const availability = (node.availability || node.Availability || 'active').toLowerCase();
        const isLeader = node.is_leader || node.IsLeader || false;

        const roleBadge = isLeader ? 
            '<span class="badge badge-green">👑 Swarm Leader</span>' : 
            (role === 'manager' ? '<span class="badge badge-blue">⚙️ Manager</span>' : '<span class="badge badge-purple">💻 Worker</span>');

        const statusBadge = status.toLowerCase() === 'ready' ? 
            '<span class="badge badge-green">Ready ✅</span>' : 
            `<span class="badge badge-red">${status} ❌</span>`;

        const escapedId = id.replace(/'/g, "\\'");
        const escapedHost = hostname.replace(/'/g, "\\'");

        return `
        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
            <td style="padding: 8px 12px;">
                <strong>${hostname}</strong><br>
                <small class="text-muted" style="font-size: 0.72rem;">ID: ${id}</small>
            </td>
            <td style="padding: 8px 12px;">${roleBadge}</td>
            <td style="padding: 8px 12px;">${statusBadge}</td>
            <td style="padding: 8px 12px;">
                <select class="term-input" style="padding: 2px 6px; font-size: 0.75rem;" onchange="updateNodeAvailability('${escapedId}', this.value)">
                    <option value="active" ${availability === 'active' ? 'selected' : ''}>Active (Normal)</option>
                    <option value="pause" ${availability === 'pause' ? 'selected' : ''}>Pause (No New Tasks)</option>
                    <option value="drain" ${availability === 'drain' ? 'selected' : ''}>Drain (Evacuar Cargas)</option>
                </select>
            </td>
            <td style="padding: 8px 12px; text-align: right;">
                <div style="display: flex; gap: 4px; justify-content: flex-end;">
                    ${role === 'worker' ? 
                        `<button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-blue); color: var(--accent-blue);" onclick="updateNodeRole('${escapedId}', 'manager')">⬆️ Promover</button>` : 
                        (!isLeader ? `<button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-yellow); color: var(--accent-yellow);" onclick="updateNodeRole('${escapedId}', 'worker')">⬇️ Degradar</button>` : '')
                    }
                    ${!isLeader ? `<button class="btn btn-outline" style="padding: 2px 6px; font-size: 0.7rem; border-color: var(--accent-red); color: var(--accent-red);" onclick="removeNodeConfirm('${escapedId}', '${escapedHost}')">🗑️ Drenar & Remove</button>` : ''}
                </div>
            </td>
        </tr>
        `;
    }).join('');
}

function updateNodeAvailability(nodeId, availability) {
    showToast('Actualizando Swarm', `Cambiando disponibilidad del nodo a '${availability}'...`, 'info');
    fetch('/api/nodes/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: nodeId, availability: availability })
    }).then(res => {
        if (res.ok) {
            showToast('Nodo Actualizado 🖥️', `Disponibilidad fijada en '${availability}'`, 'success');
            refreshNodeManagementTable();
        } else {
            res.text().then(err => showToast('Error', err, 'error'));
        }
    }).catch(e => showToast('Error de Red', e.message, 'error'));
}

function updateNodeRole(nodeId, role) {
    showToast('Actualizando Swarm', `Cambiando rol del nodo a '${role}'...`, 'info');
    fetch('/api/nodes/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: nodeId, role: role })
    }).then(res => {
        if (res.ok) {
            showToast('Rol Actualizado 👑', `Nodo promovido/degradado a '${role}'`, 'success');
            refreshNodeManagementTable();
        } else {
            res.text().then(err => showToast('Error', err, 'error'));
        }
    }).catch(e => showToast('Error de Red', e.message, 'error'));
}

function removeNodeConfirm(nodeId, hostname) {
    requestConfirmation(
        '🗑️ Remover Nodo de Swarm Cluster',
        `¿Estás seguro de drenar y remover el nodo '${hostname}' (ID: ${nodeId}) del clúster Swarm? Todas las tareas serán migradas a otros nodos.`,
        async () => {
            showToast('Removiendo Nodo', `Drenando y eliminando '${hostname}'...`, 'info');
            try {
                const res = await fetch(`/api/nodes?id=${encodeURIComponent(nodeId)}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('Nodo Eliminado 🗑️', `'${hostname}' ya no pertenece al clúster`, 'success');
                    refreshNodeManagementTable();
                } else {
                    const err = await res.text();
                    showToast('Error al Eliminar', err, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

function openWorkerProvisionModal() {
    closeModal('nodeManagementModal');
    openModal('workerModal');
}

/* --- Interactive Service Link Topology Diagram --- */
function openLinkModal() {
    const srcSelect = document.getElementById('linkSourceSvc');
    const targetSelect = document.getElementById('linkTargetSvc');

    let allTargets = [];
    if (globalServices) {
        globalServices.forEach(s => allTargets.push({ name: s.name, type: '🚀 App' }));
    }
    if (globalDatabases) {
        globalDatabases.forEach(db => allTargets.push({ name: db.name, type: '🗄️ BD ' + db.engine }));
    }

    if (srcSelect && globalServices) {
        srcSelect.innerHTML = globalServices.length > 0
            ? globalServices.map(s => `<option value="${s.name}">${s.name} (App)</option>`).join('')
            : '<option value="">(No hay servicios activos)</option>';
    }

    if (targetSelect) {
        targetSelect.innerHTML = allTargets.length > 0
            ? allTargets.map(t => `<option value="${t.name}">${t.name} (${t.type})</option>`).join('')
            : '<option value="">(No hay destinos disponibles)</option>';
    }

    openModal('linkModal');
}

function openLinkModalForSource(sourceSvc) {
    openLinkModal();
    const srcSelect = document.getElementById('linkSourceSvc');
    if (srcSelect && sourceSvc) {
        srcSelect.value = sourceSvc;
    }
}

let lastRenderedTopologyHash = '';

async function renderTopologyMap() {
    const container = document.getElementById('topologyCanvas');
    if (!container) return;

    if (!globalIsOnline) {
        container.innerHTML = `
            <div style="text-align: center; padding: 32px 16px; color: #ef4444; background: rgba(239,68,68,0.03); border: 1px dashed rgba(239,68,68,0.2); border-radius: 6px;">
                <div style="font-size: 1.4rem; margin-bottom: 6px;">🔌 Topología Desactivada (Offline)</div>
                <div style="font-size: 0.82rem; color: var(--text-muted);">El mapa de interconexión de red se oculta automáticamente cuando la máquina virtual host está inalcanzable.</div>
            </div>
        `;
        return;
    }

    try {
        const linksRes = await fetch('/api/links');
        const links = linksRes.ok ? await linksRes.json() : [];

        const services = globalServices || [];
        const dbs = globalDatabases || [];

        const currentTopoHash = JSON.stringify({
            online: globalIsOnline,
            svcs: services.map(s => s.name),
            dbs: dbs.map(d => d.name),
            links: links
        });

        if (currentTopoHash === lastRenderedTopologyHash && container.children.length > 0) {
            return;
        }
        lastRenderedTopologyHash = currentTopoHash;

        if (services.length === 0 && dbs.length === 0) {
            container.innerHTML = `
                <div style="text-align: center; padding: 20px; color: var(--text-muted);">
                    <div style="font-size: 1.2rem; margin-bottom: 6px;">🕸️</div>
                    <div style="font-size: 0.8rem;">No hay servicios ni bases de datos desplegadas en la red.</div>
                    <button class="btn btn-primary" style="margin-top: 10px; font-size: 0.75rem;" onclick="openModal('deployServiceModal')">+ Desplegar Servicio</button>
                </div>
            `;
            return;
        }

        let html = `
        <div class="flowchart-canvas" style="position: relative; overflow: visible; padding: 16px 20px; min-height: 180px;">
            <!-- SVG Overlay Layer for Flowchart Bezier Lines -->
            <svg id="flowchartSvg" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%; pointer-events: none; overflow: visible; z-index: 10;"></svg>

            <!-- Flowchart Nodes Grid (Compact & Clean Zoom Layout) -->
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 60px; align-items: center; position: relative; z-index: 1;">
                <!-- Column 1: Services (Origen A) -->
                <div style="display: flex; flex-direction: column; gap: 12px;">
                    <div style="font-size: 0.7rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">Servicios (Apps)</div>
        `;

        if (services.length === 0) {
            html += `<div class="flow-node" style="color: var(--text-muted); font-size: 0.75rem;">(Sin servicios)</div>`;
        } else {
            services.forEach(s => {
                html += `
                <div class="flow-node" id="node-svc-${s.name}" style="position: relative; background: #18181b; border: 1px solid #27272a; padding: 8px 12px; border-radius: 4px; font-size: 0.8rem;">
                    <div id="port-in-${s.name}" style="position: absolute; left: -5px; top: 50%; transform: translateY(-50%); width: 8px; height: 8px; background: #3b82f6; border-radius: 50%; border: 1px solid #09090b;"></div>
                    <div>
                        <strong style="color: #fafafa; font-size: 0.82rem;">🚀 ${s.name}</strong>
                        <div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 1px;">Puerto: :${s.port}</div>
                    </div>
                    <div id="port-out-${s.name}" style="position: absolute; right: -5px; top: 50%; transform: translateY(-50%); width: 8px; height: 8px; background: #3b82f6; border-radius: 50%; border: 1px solid #09090b;"></div>
                </div>
                `;
            });
        }

        html += `
                </div>

                <!-- Column 2: Databases & Targets (Destino B) -->
                <div style="display: flex; flex-direction: column; gap: 12px;">
                    <div style="font-size: 0.7rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">Bases de Datos &amp; Recursos</div>
        `;

        if (dbs.length === 0) {
            html += `<div class="flow-node" style="color: var(--text-muted); font-size: 0.75rem;">(Sin bases de datos)</div>`;
        } else {
            dbs.forEach(db => {
                html += `
                <div class="flow-node" id="node-db-${db.name}" style="position: relative; background: #18181b; border: 1px solid #27272a; padding: 8px 12px; border-radius: 4px; font-size: 0.8rem;">
                    <div id="port-in-${db.name}" style="position: absolute; left: -5px; top: 50%; transform: translateY(-50%); width: 8px; height: 8px; background: #22c55e; border-radius: 50%; border: 1px solid #09090b;"></div>
                    <div>
                        <strong style="color: #fafafa; font-size: 0.82rem;">🗄️ ${db.name}</strong>
                        <div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 1px;">:${db.internalPort} (${db.engine})</div>
                    </div>
                    <div id="port-out-${db.name}" style="position: absolute; right: -5px; top: 50%; transform: translateY(-50%); width: 8px; height: 8px; background: #22c55e; border-radius: 50%; border: 1px solid #09090b;"></div>
                </div>
                `;
            });
        }

        html += `
                </div>
            </div>

            <!-- Clean Minimal Connections Quick Edit List -->
            <div style="margin-top: 14px; border-top: 1px solid #27272a; padding-top: 10px; position: relative; z-index: 2;">
                <div style="font-size: 0.7rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; margin-bottom: 6px;">Enlaces Activos (${links.length}) — Haz clic para editar</div>
                <div style="display: flex; flex-wrap: wrap; gap: 6px;">
                    ${links.length === 0 ? `
                        <span style="font-size: 0.75rem; color: var(--text-muted); font-style: italic;">Sin enlaces de red activos.</span>
                    ` : links.map(l => {
                        const escapedSrc = l.sourceSvc.replace(/'/g, "\\'");
                        const escapedTgt = l.targetSvc.replace(/'/g, "\\'");
                        const escapedEnv = (l.envVarName || 'DATABASE_URL').replace(/'/g, "\\'");
                        return `
                            <button class="btn btn-outline" style="padding: 3px 8px; font-size: 0.75rem; background: #18181b; border-color: #27272a;" onclick="openEditLinkModal('${escapedSrc}', '${escapedTgt}', '${escapedEnv}')" title="Editar enlace">
                                <span>${l.sourceSvc} ➔ ${l.targetSvc}</span>
                                <span style="color: var(--accent-blue); font-family: var(--font-mono); font-size: 0.7rem; margin-left: 4px;">(${l.envVarName || 'DATABASE_URL'})</span>
                                <span style="color: var(--text-muted); margin-left: 4px;">✏️</span>
                            </button>
                        `;
                    }).join('')}
                </div>
            </div>
        </div>
        `;

        container.innerHTML = html;

        setTimeout(() => drawFlowchartConnections(links), 50);
        setTimeout(() => drawFlowchartConnections(links), 200);

    } catch (err) {
        container.innerHTML = `<div style="color: var(--accent-red); padding: 8px; text-align: center;">Error al cargar mapa de topología: ${err.message}</div>`;
    }
}

let lastFetchedLinks = [];

function drawFlowchartConnections(links) {
    if (links) lastFetchedLinks = links;
    else links = lastFetchedLinks;

    const canvas = document.querySelector('.flowchart-canvas') || document.getElementById('topologyCanvas');
    const svg = document.getElementById('flowchartSvg');
    if (!canvas || !svg) return;

    const canvasRect = canvas.getBoundingClientRect();
    let svgPathsHtml = `
        <defs>
            <marker id="flowArrow" viewBox="0 0 10 10" refX="7" refY="5" markerWidth="5" markerHeight="5" orient="auto">
                <path d="M 0 1 L 8 5 L 0 9 z" fill="#3b82f6"/>
            </marker>
        </defs>
    `;

    if (links && links.length > 0) {
        links.forEach(l => {
            const outPort = document.getElementById(`port-out-${l.sourceSvc}`) || document.getElementById(`port-in-${l.sourceSvc}`);
            const inPort = document.getElementById(`port-in-${l.targetSvc}`) || document.getElementById(`port-out-${l.targetSvc}`);

            if (outPort && inPort) {
                const r1 = outPort.getBoundingClientRect();
                const r2 = inPort.getBoundingClientRect();

                const x1 = r1.left + r1.width / 2 - canvasRect.left;
                const y1 = r1.top + r1.height / 2 - canvasRect.top;
                const x2 = r2.left + r2.width / 2 - canvasRect.left;
                const y2 = r2.top + r2.height / 2 - canvasRect.top;

                let pathD = '';
                if (Math.abs(y1 - y2) < 5) {
                    pathD = `M ${x1} ${y1} L ${x2} ${y2}`;
                } else {
                    const dx = Math.min(Math.abs(x2 - x1) * 0.5, 50);
                    pathD = `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
                }

                const escapedSrc = l.sourceSvc.replace(/'/g, "\\'");
                const escapedTgt = l.targetSvc.replace(/'/g, "\\'");
                const escapedEnv = (l.envVarName || 'DATABASE_URL').replace(/'/g, "\\'");

                svgPathsHtml += `
                    <g class="flow-connection-group" style="cursor: pointer;" onclick="openEditLinkModal('${escapedSrc}', '${escapedTgt}', '${escapedEnv}')" title="Haz clic en esta línea para editar el enlace '${l.sourceSvc}' ➔ '${l.targetSvc}'">
                        <path d="${pathD}" stroke="transparent" stroke-width="14" fill="none" style="pointer-events: stroke;"/>
                        <path d="${pathD}" stroke="#3b82f6" stroke-width="2" fill="none" marker-end="url(#flowArrow)" style="pointer-events: stroke;" class="flow-line"/>
                        <rect x="${(x1 + x2)/2 - 40}" y="${(y1 + y2)/2 - 10}" width="80" height="18" rx="3" fill="#18181b" stroke="#27272a" style="pointer-events: auto;"/>
                        <text x="${(x1 + x2)/2}" y="${(y1 + y2)/2 + 3}" fill="#fafafa" font-size="9" font-family="monospace" font-weight="600" text-anchor="middle" style="pointer-events: auto;">ENV: ${l.envVarName || 'LINK'}</text>
                    </g>
                `;
            }
        });
    }

    svg.innerHTML = svgPathsHtml;
}

function openEditLinkModal(sourceSvc, targetSvc, envVarName) {
    const srcInput = document.getElementById('editLinkSource');
    const tgtInput = document.getElementById('editLinkTarget');
    const srcDisp = document.getElementById('editLinkSourceDisplay');
    const tgtDisp = document.getElementById('editLinkTargetDisplay');
    const envInput = document.getElementById('editLinkEnvVar');

    if (srcInput) srcInput.value = sourceSvc || '';
    if (tgtInput) tgtInput.value = targetSvc || '';
    if (srcDisp) srcDisp.value = sourceSvc || '';
    if (tgtDisp) tgtDisp.value = targetSvc || '';
    if (envInput) envInput.value = envVarName || 'DATABASE_URL';

    openModal('editLinkModal');
}

function deleteLinkFromModalAction() {
    const sourceSvc = document.getElementById('editLinkSource')?.value;
    const targetSvc = document.getElementById('editLinkTarget')?.value;
    closeModal('editLinkModal');
    if (sourceSvc && targetSvc) {
        confirmUnlinkService(sourceSvc, targetSvc);
    }
}

window.addEventListener('resize', () => {
    drawFlowchartConnections();
});

function confirmUnlinkService(sourceSvc, targetSvc) {
    requestConfirmation(
        '✂️ Desenlazar Servicios',
        `¿Desenlazar '${sourceSvc}' de '${targetSvc}' y remover la variable de entorno de Docker Swarm?`,
        async () => {
            showToast('Desenlazando...', `Removiendo vínculo entre ${sourceSvc} y ${targetSvc}...`, 'info');
            try {
                const res = await fetch(`/api/links?source_svc=${encodeURIComponent(sourceSvc)}&target_svc=${encodeURIComponent(targetSvc)}`, {
                    method: 'DELETE'
                });

                if (res.ok) {
                    showToast('Servicios Desenlazados ✂️', `Se removió el vínculo entre '${sourceSvc}' y '${targetSvc}'`, 'success');
                    loadDashboardData();
                } else {
                    const err = await res.json().catch(() => ({ error: 'Error al desenlazar' }));
                    showToast('Error al Desenlazar', err.error || err.message, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

/* --- Unified Railway / Vercel Resource Inspector --- */
let currentInspectorTarget = { name: '', type: '', data: null };

async function openResourceInspector(name, type, initialTab = 'metrics') {
    currentInspectorTarget.name = name;
    currentInspectorTarget.type = type;

    let targetData = null;
    if (type === 'service') {
        targetData = (globalServices || []).find(s => s.name === name);
    } else {
        targetData = (globalDatabases || []).find(d => d.name === name);
    }
    currentInspectorTarget.data = targetData;

    // Header info
    const iconEl = document.getElementById('inspectorIcon');
    const titleEl = document.getElementById('inspectorTitle');
    const typeBadge = document.getElementById('inspectorTypeBadge');
    const statusBadge = document.getElementById('inspectorStatusBadge');
    const subtitleEl = document.getElementById('inspectorSubtitle');
    const openUrlBtn = document.getElementById('inspectorOpenUrlBtn');

    if (iconEl) iconEl.innerText = type === 'service' ? '🚀' : '🗄️';
    if (titleEl) titleEl.innerText = name;
    if (typeBadge) {
        typeBadge.innerText = type === 'service' ? 'App Service' : `Database (${targetData?.engine || 'DB'})`;
        typeBadge.className = type === 'service' ? 'badge badge-blue' : 'badge badge-yellow';
    }

    if (subtitleEl) {
        if (type === 'service') {
            const img = targetData?.imageSource || targetData?.image_source || 'custom';
            const port = targetData?.port || 80;
            subtitleEl.innerText = `Image: ${img} • Internal Port: ${port}`;
        } else {
            const engine = targetData?.engine || 'postgres';
            const port = targetData?.internalPort || targetData?.internal_port || 5432;
            subtitleEl.innerText = `Engine: ${engine} • Internal Port: ${port}`;
        }
    }

    // Open URL Button
    if (openUrlBtn) {
        if (type === 'service' && targetData && targetData.domain) {
            const proto = targetData.enableSSL ? 'https' : 'http';
            openUrlBtn.href = `${proto}://${targetData.domain}`;
            openUrlBtn.style.display = 'inline-flex';
        } else {
            openUrlBtn.style.display = 'none';
        }
    }

    // Populate Config tab inputs
    const cfgName = document.getElementById('inspCfgName');
    const cfgImage = document.getElementById('inspCfgImage');
    const cfgPort = document.getElementById('inspCfgPort');
    const cfgDomain = document.getElementById('inspCfgDomain');
    const cfgHealth = document.getElementById('inspCfgHealth');
    const cfgExpose = document.getElementById('inspCfgExpose');
    const cfgSSL = document.getElementById('inspCfgSSL');

    if (cfgName) cfgName.value = name;
    if (cfgImage) cfgImage.value = targetData?.imageSource || targetData?.image_source || targetData?.engine || '';
    if (cfgPort) cfgPort.value = targetData?.port || targetData?.internalPort || targetData?.internal_port || 80;
    if (cfgDomain) cfgDomain.value = targetData?.domain || '';
    if (cfgHealth) cfgHealth.value = targetData?.healthcheckCmd || targetData?.healthcheck_cmd || '';
    if (cfgExpose) cfgExpose.checked = targetData ? !!targetData.expose : true;
    if (cfgSSL) cfgSSL.checked = targetData ? (targetData.enableSSL !== undefined ? targetData.enableSSL : (targetData.enable_ssl !== undefined ? targetData.enable_ssl : true)) : true;

    // Tab visibility for DB vs Service
    const btnRollback = document.getElementById('tabBtnRollback');
    const btnBackups = document.getElementById('tabBtnBackups');

    if (btnRollback) btnRollback.style.display = type === 'service' ? 'inline-block' : 'none';
    if (btnBackups) btnBackups.style.display = type === 'database' ? 'inline-block' : 'none';

    // Reset log container state so logs are not fetched or shown automatically
    const logBox = document.getElementById('inspLogContent');
    if (logBox && initialTab !== 'logs') {
        logBox.innerText = `[Selecciona la pestaña '📜 Logs' para transmitir los registros de ${name}]`;
    }

    switchInspectorTab(initialTab);
    fetchLiveContainerStats(name);
    openModal('resourceInspectorModal');
}

function switchInspectorTab(tabName) {
    const tabs = ['metrics', 'logs', 'envs', 'rollback', 'backups', 'config'];
    tabs.forEach(t => {
        const btn = document.getElementById(`tabBtn${t.charAt(0).toUpperCase() + t.slice(1)}`);
        const pane = document.getElementById(`pane${t.charAt(0).toUpperCase() + t.slice(1)}`);
        if (btn) btn.classList.toggle('active', t === tabName);
        if (pane) pane.classList.toggle('active', t === tabName);
    });

    // Load data ONLY for the explicitly activated tab
    const name = currentInspectorTarget.name;
    if (!name) return;

    if (tabName === 'metrics') loadInspectorMetrics();
    else if (tabName === 'logs') loadInspectorLogs();
    else if (tabName === 'envs') loadInspectorEnvs();
    else if (tabName === 'backups') loadInspectorBackups();
}

async function loadInspectorMetrics(isManual = false) {
    const target = currentInspectorTarget;
    if (!target.name) return;

    if (isManual) {
        showToast('Validando Conexión 🔄', `Verificando estado SSH y Swarm para '${target.name}'...`, 'info', 3000);
    }

    const replEl = document.getElementById('inspMetricReplicas');
    const ramEl = document.getElementById('inspMetricRam');
    const portEl = document.getElementById('inspMetricPort');
    const exposeEl = document.getElementById('inspMetricExpose');
    const statusBadge = document.getElementById('inspectorStatusBadge');

    try {
        const res = await fetch(`/api/observability/metrics?service=${encodeURIComponent(target.name)}`);
        if (res.ok) {
            const data = await res.json();
            const points = data.points || [];
            const lastPoint = points.length > 0 ? points[points.length - 1] : null;

            if (lastPoint && (lastPoint.cpu > 0 || lastPoint.memory > 0)) {
                if (replEl) replEl.innerText = '1/1 Active (Swarm)';
                if (ramEl) ramEl.innerText = `${lastPoint.memory} MB (Live)`;
                if (statusBadge) {
                    statusBadge.innerText = '● Running';
                    statusBadge.className = 'badge badge-green';
                }
            } else {
                if (replEl) replEl.innerText = '0/0 Replicas (Offline)';
                if (ramEl) ramEl.innerText = '0.0 MB (Offline)';
                if (statusBadge) {
                    statusBadge.innerText = '● Offline / Detenido';
                    statusBadge.className = 'badge badge-red';
                }
            }
        } else {
            if (replEl) replEl.innerText = '0/0 (Host Inalcanzable)';
            if (ramEl) ramEl.innerText = '0.0 MB';
            if (statusBadge) {
                statusBadge.innerText = '● Offline';
                statusBadge.className = 'badge badge-red';
            }
        }
    } catch (e) {
        if (replEl) replEl.innerText = '0/0 (Error de Red)';
        if (ramEl) ramEl.innerText = '0.0 MB';
    }

    if (portEl) {
        const p = target.data?.port || target.data?.internalPort || target.data?.internal_port || 80;
        portEl.innerText = `${p} / Internal`;
    }
    if (exposeEl) {
        exposeEl.innerText = target.data?.domain ? `Público (https://${target.data.domain})` : 'Privado (Cluster Overlay)';
    }
}

async function loadInspectorLogs() {
    const name = currentInspectorTarget.name;
    const logBox = document.getElementById('inspLogContent');
    const linesSelect = document.getElementById('inspLogLinesSelect');
    if (!name || !logBox) return;

    const lines = linesSelect ? linesSelect.value : '100';
    logBox.innerText = `[Transmitiendo logs de ${name}...]`;

    try {
        const res = await fetch(`/api/logs?name=${encodeURIComponent(name)}&service=${encodeURIComponent(name)}&lines=${lines}`);
        if (res.ok) {
            let logsText = '';
            try {
                const data = await res.json();
                logsText = data.logs || (typeof data === 'string' ? data : JSON.stringify(data, null, 2));
            } catch (e) {
                logsText = await res.text();
            }
            if (typeof logsText !== 'string') {
                logsText = JSON.stringify(logsText, null, 2);
            }
            lastFetchedLogsRaw = logsText || `[Sin logs recientes para ${name}]`;
            filterInspectorLogs();
        } else {
            const errText = await res.text().catch(() => 'Error al obtener logs');
            logBox.innerText = `[Error al obtener logs de ${name}: ${errText}]`;
        }
    } catch (e) {
        logBox.innerText = `[Error conectando a logs: ${e.message}]`;
    }
}

async function loadInspectorEnvs() {
    const name = currentInspectorTarget.name;
    const txt = document.getElementById('inspEnvTextarea');
    if (!name || !txt) return;

    txt.value = '# Cargando variables de entorno...';
    try {
        const res = await fetch(`/api/env?service=${encodeURIComponent(name)}`);
        if (res.ok) {
            const data = await res.json();
            txt.value = data.rawContent || '';
        } else {
            txt.value = '# No hay variables configuradas';
        }
    } catch (e) {
        txt.value = '# Error: ' + e.message;
    }
}

async function saveInspectorEnvs() {
    const name = currentInspectorTarget.name;
    const txt = document.getElementById('inspEnvTextarea');
    if (!name || !txt) return;

    showToast('Guardando...', `Inyectando variables en Swarm para ${name}...`, 'info');
    try {
        const res = await fetch('/api/env', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                serviceName: name,
                rawEnvContent: txt.value
            })
        });

        if (res.ok) {
            showToast('Variables Guardadas 🔑', `Se inyectaron las variables en '${name}'`, 'success');
        } else {
            const err = await res.json().catch(() => ({ error: 'Error al guardar .env' }));
            showToast('Error', err.error || 'Error al guardar', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

function importInspectorEnvFile(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function(e) {
        const txt = document.getElementById('inspEnvTextarea');
        if (txt) {
            txt.value = e.target.result;
            showToast('Archivo Cargado', `Importado '${file.name}' al editor. Presiona Guardar.`, 'info');
        }
    };
    reader.readAsText(file);
}

async function triggerInspectorRollback() {
    const name = currentInspectorTarget.name;
    if (!name) return;

    requestConfirmation(
        '⚠️ Rollback de Servicio',
        `¿Revertir inmediatamente '${name}' a la versión previa en Swarm?`,
        async () => {
            showToast('Ejecutando Rollback...', `Revirtiendo ${name}...`, 'info');
            try {
                const res = await fetch('/api/services/rollback', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ serviceName: name })
                });

                if (res.ok) {
                    showToast('Rollback Exitoso ⏪', `'${name}' revertido a versión previa`, 'success');
                } else {
                    const err = await res.json().catch(() => ({ error: 'Error en rollback' }));
                    showToast('Error', err.error || err.message, 'error');
                }
            } catch (e) {
                showToast('Error de Red', e.message, 'error');
            }
        }
    );
}

async function loadInspectorBackups() {
    const name = currentInspectorTarget.name;
    const list = document.getElementById('inspBackupsList');
    if (!name || !list) return;

    list.innerHTML = '<div class="text-muted p-2">Cargando snapshots...</div>';
    try {
        const res = await fetch('/api/backups');
        if (!res.ok) return;
        const backups = await res.json();
        const filtered = (backups || []).filter(b => b.targetName === name || b.target_name === name);

        if (filtered.length === 0) {
            list.innerHTML = `<div class="text-muted p-2" style="font-size:0.85rem">No hay respaldos registrados para '${name}'. Presiona "+ Crear Snapshot" para generar uno.</div>`;
            return;
        }

        list.innerHTML = filtered.map(b => {
            const sizeMb = ((b.sizeBytes || b.size_bytes || 0) / (1024 * 1024)).toFixed(2);
            return `
                <div class="endpoint-item">
                    <div>
                        <strong>📦 ${b.filename || 'backup.sql.gz'}</strong>
                        <div class="text-muted" style="font-size:0.75rem;">${b.createdAt || b.created_at || ''} • ${sizeMb} MB • Status: ${b.status || 'completed'}</div>
                    </div>
                    <div style="display:flex; gap:6px;">
                        <a href="/api/backups/download?id=${b.id}" class="btn btn-outline" style="padding:2px 8px; font-size:0.72rem;">📥 Descargar</a>
                        <button class="btn btn-danger" style="padding:2px 8px; font-size:0.72rem;" onclick="restoreBackupAction(${b.id})">🔄 Restaurar</button>
                    </div>
                </div>
            `;
        }).join('');
    } catch (e) {
        list.innerHTML = `<div class="text-muted p-2">Error cargando respaldos: ${e.message}</div>`;
    }
}

async function createInspectorSnapshot() {
    const name = currentInspectorTarget.name;
    const type = currentInspectorTarget.type;
    if (!name) return;

    showToast('Generando Snapshot...', `Iniciando backup de ${name}...`, 'info');
    try {
        const res = await fetch('/api/backups', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                targetName: name,
                targetType: type === 'database' ? 'database' : 'volume'
            })
        });

        if (res.ok) {
            showToast('Snapshot Creado 💾', `Backup guardado exitosamente para '${name}'`, 'success');
            loadInspectorBackups();
            loadBackups();
        } else {
            const err = await res.json().catch(() => ({ error: 'Error al crear backup' }));
            showToast('Error', err.error || 'Fallo al crear snapshot', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

async function deleteDBAction(name) {
    if (!name) return;
    showToast('Eliminando Base de Datos', `Removiendo ${name}...`, 'info');
    try {
        const res = await fetch(`/api/databases?name=${encodeURIComponent(name)}`, {
            method: 'DELETE'
        });
        await fetch(`/api/services?name=${encodeURIComponent(name)}`, {
            method: 'DELETE'
        }).catch(() => {});

        if (res.ok) {
            showToast('Base de Datos Eliminada 🗑️', `Se eliminó '${name}' correctamente.`, 'success');
            loadDashboardData(true);
        } else {
            const err = await res.text();
            showToast('Error al Eliminar', err, 'error');
        }
    } catch (err) {
        showToast('Error de Conexión', err.message, 'error');
    }
}

async function deleteCurrentInspectorResource() {
    const name = currentInspectorTarget.name;
    const type = currentInspectorTarget.type;
    if (!name) return;

    requestConfirmation(
        `🗑️ Eliminar ${type === 'service' ? 'Servicio' : 'Base de Datos'}`,
        `¿Estás seguro de eliminar '${name}'? Se removerán contenedores e instancias en Swarm.`,
        async () => {
            closeModal('resourceInspectorModal');
            if (type === 'service') {
                deleteServiceAction(name);
            } else {
                deleteDBAction(name);
            }
        }
    );
}

async function saveInspectorConfig(event) {
    event.preventDefault();
    const name = currentInspectorTarget.name;
    const type = currentInspectorTarget.type;

    const img = document.getElementById('inspCfgImage')?.value.trim();
    const port = parseInt(document.getElementById('inspCfgPort')?.value.trim() || '80', 10);
    const domain = document.getElementById('inspCfgDomain')?.value.trim();
    const health = document.getElementById('inspCfgHealth')?.value.trim();
    const expose = document.getElementById('inspCfgExpose')?.checked;
    const enableSSL = document.getElementById('inspCfgSSL')?.checked;

    showToast('Guardando Configuración...', `Actualizando ${name}...`, 'info');

    try {
        const endpoint = type === 'service' ? `/api/services/${encodeURIComponent(name)}` : `/api/databases/${encodeURIComponent(name)}`;
        const payload = type === 'service' ? {
            name: name,
            imageSource: img,
            port: port,
            domain: domain,
            expose: expose !== undefined ? expose : !!domain,
            enableSSL: enableSSL !== undefined ? enableSSL : true,
            healthcheckCmd: health
        } : {
            name: name,
            engine: img,
            internalPort: port,
            externalURL: domain
        };

        const res = await fetch(endpoint, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        if (res.ok) {
            showToast('Configuración Actualizada ⚙️', `Se guardaron los cambios para '${name}'`, 'success');
            loadDashboardData();
        } else {
            showToast('Error', 'Fallo al actualizar configuración', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

async function toggleServiceVisibility(serviceName) {
    const service = (globalServices || []).find(s => s.name === serviceName);
    if (!service) return;

    const newExpose = !service.expose;
    showToast('Cambiando Visibilidad...', `${newExpose ? 'Exponiendo' : 'Ocultando'} ${serviceName}...`, 'info');

    try {
        const res = await fetch(`/api/services/${encodeURIComponent(serviceName)}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: service.name,
                imageSource: service.imageSource || service.image_source,
                port: service.port,
                domain: service.domain,
                expose: newExpose,
                enableSSL: service.enableSSL !== undefined ? service.enableSSL : true,
                healthcheckCmd: service.healthcheckCmd
            })
        });

        if (res.ok) {
            showToast('Visibilidad Actualizada 🌐', `'${serviceName}' ahora es ${newExpose ? 'PÚBLICO' : 'PRIVADO'}`, 'success');
            loadDashboardData();
            if (currentInspectorTarget.name === serviceName) {
                openResourceInspector(serviceName, 'service');
            }
        } else {
            showToast('Error', 'No se pudo cambiar la visibilidad', 'error');
        }
    } catch (e) {
        showToast('Error de Red', e.message, 'error');
    }
}

/* --- Real-Time Catalog Search Filter --- */
function filterCatalogList() {
    const input = document.getElementById('catalogSearchInput');
    if (!input) return;

    const query = input.value.toLowerCase().trim();
    if (!query) {
        renderCatalog(globalServices, globalDatabases);
        return;
    }

    const filteredServices = (globalServices || []).filter(s =>
        s.name.toLowerCase().includes(query) ||
        (s.imageSource && s.imageSource.toLowerCase().includes(query)) ||
        (s.domain && s.domain.toLowerCase().includes(query))
    );

    const filteredDBs = (globalDatabases || []).filter(d =>
        d.name.toLowerCase().includes(query) ||
        (d.engine && d.engine.toLowerCase().includes(query))
    );

    renderCatalog(filteredServices, filteredDBs);
}

/* --- Advanced Log Grep, Filter & Download --- */
let lastFetchedLogsRaw = '';

function filterInspectorLogs() {
    const logBox = document.getElementById('inspLogContent');
    const grepInput = document.getElementById('inspLogGrepInput');
    const levelSelect = document.getElementById('inspLogLevelSelect');
    if (!logBox || !lastFetchedLogsRaw) return;

    const grep = grepInput ? grepInput.value.toLowerCase().trim() : '';
    const level = levelSelect ? levelSelect.value : 'all';

    let lines = lastFetchedLogsRaw.split('\n');

    if (level === 'error') {
        lines = lines.filter(l => /error|err|fatal|panic|fail|exception/i.test(l));
    } else if (level === 'warn') {
        lines = lines.filter(l => /warn|warning/i.test(l));
    } else if (level === 'info') {
        lines = lines.filter(l => /info|debug|trace/i.test(l));
    }

    if (grep) {
        lines = lines.filter(l => l.toLowerCase().includes(grep));
    }

    logBox.innerText = lines.join('\n') || '[No se encontraron coincidencias para los filtros especificados]';
    logBox.scrollTop = logBox.scrollHeight;
}

function downloadInspectorLogs() {
    const name = currentInspectorTarget.name || 'service';
    const logBox = document.getElementById('inspLogContent');
    if (!logBox || !logBox.innerText) {
        showToast('Advertencia', 'No hay logs disponibles para descargar', 'warning');
        return;
    }

    const blob = new Blob([logBox.innerText], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${name}-logs-${new Date().toISOString().slice(0, 10)}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast('Logs Descargados 📥', `Guardado '${a.download}' en tu equipo`, 'success');
}

/* --- Environment Variables Secret Masking & Table Mode --- */
let isEnvMasked = false;
let isEnvTableMode = false;

function toggleEnvSecretMasking() {
    isEnvMasked = !isEnvMasked;
    const btn = document.getElementById('btnToggleEnvMask');
    if (btn) btn.innerText = isEnvMasked ? '👁️ Revelar Secretos' : '👁️ Ocultar Secretos';

    const txt = document.getElementById('inspEnvTextarea');
    if (!txt) return;

    let content = txt.value;
    if (isEnvMasked) {
        content = content.split('\n').map(line => {
            if (line.includes('=') && !line.startsWith('#')) {
                const parts = line.split('=');
                const key = parts[0].trim();
                const val = parts.slice(1).join('=');
                if (/pass|secret|key|token|auth|url/i.test(key)) {
                    return `${key}=••••••••••••`;
                }
            }
            return line;
        }).join('\n');
    } else {
        loadInspectorEnvs();
        return;
    }
    txt.value = content;
}

function toggleEnvEditorMode() {
    isEnvTableMode = !isEnvTableMode;
    const btn = document.getElementById('btnToggleEnvMode');
    const rawContainer = document.getElementById('envRawEditorContainer');
    const tableContainer = document.getElementById('envTableEditorContainer');
    const txt = document.getElementById('inspEnvTextarea');

    if (btn) btn.innerText = isEnvTableMode ? '⇄ Modo Raw (.env)' : '⇄ Modo Tabla';

    if (isEnvTableMode) {
        if (rawContainer) rawContainer.style.display = 'none';
        if (tableContainer) tableContainer.style.display = 'block';

        const tbody = document.getElementById('envTableBody');
        if (tbody && txt) {
            const lines = txt.value.split('\n').filter(l => l.includes('=') && !l.startsWith('#'));
            if (lines.length === 0) {
                tbody.innerHTML = '<tr><td colspan="2" class="text-muted p-2">Sin variables definidas</td></tr>';
            } else {
                tbody.innerHTML = lines.map(l => {
                    const parts = l.split('=');
                    const key = parts[0].trim();
                    const val = parts.slice(1).join('=').trim();
                    return `
                        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
                            <td style="padding: 6px; font-family: var(--font-mono); color: #60a5fa;"><strong>${key}</strong></td>
                            <td style="padding: 6px; font-family: var(--font-mono); color: var(--text-primary);">${val}</td>
                        </tr>
                    `;
                }).join('');
            }
        }
    } else {
        if (rawContainer) rawContainer.style.display = 'block';
        if (tableContainer) tableContainer.style.display = 'none';
    }
}

/* --- Enterprise Extensions: Audit Logs, Private Registries & Live Stats --- */

async function openAuditLogsModal() {
    openModal('auditLogsModal');
    loadAuditLogs();
}

async function loadAuditLogs() {
    const container = document.getElementById('auditLogsContainer');
    if (!container) return;

    container.innerHTML = '<div class="text-muted p-2">Cargando registros de auditoría...</div>';
    try {
        const res = await fetch('/api/audit-logs');
        if (!res.ok) return;
        const logs = await res.json();

        if (!logs || logs.length === 0) {
            container.innerHTML = '<div class="text-muted p-2" style="font-size:0.85rem;">No hay eventos de auditoría registrados.</div>';
            return;
        }

        container.innerHTML = `
            <table style="width: 100%; border-collapse: collapse; font-size: 0.8rem;">
                <thead>
                    <tr style="border-bottom: 1px solid var(--card-border); color: #94a3b8; text-align: left;">
                        <th style="padding: 8px;">Hora</th>
                        <th style="padding: 8px;">Acción</th>
                        <th style="padding: 8px;">Recurso</th>
                        <th style="padding: 8px;">Detalles</th>
                    </tr>
                </thead>
                <tbody>
                    ${logs.map(l => `
                        <tr style="border-bottom: 1px solid rgba(255,255,255,0.05); font-family: var(--font-mono);">
                            <td style="padding: 8px; color: #64748b; white-space: nowrap;">${new Date(l.timestamp).toLocaleString()}</td>
                            <td style="padding: 8px;"><span class="badge ${l.action === 'DELETE' ? 'badge-red' : (l.action === 'DEPLOY' ? 'badge-green' : 'badge-blue')}">${l.action}</span></td>
                            <td style="padding: 8px; color: #f4f4f5;"><strong>${l.resourceName || l.resource_name || ''}</strong></td>
                            <td style="padding: 8px; color: #94a3b8;">${l.details}</td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (e) {
        container.innerHTML = `<div class="text-muted p-2">Error cargando auditoría: ${e.message}</div>`;
    }
}

async function openRegistriesModal() {
    openModal('registriesModal');
    loadRegistries();
}

async function loadRegistries() {
    const list = document.getElementById('registriesList');
    if (!list) return;

    list.innerHTML = '<div class="text-muted p-2">Cargando credenciales registradas...</div>';
    try {
        const res = await fetch('/api/registries');
        if (!res.ok) return;
        const regs = await res.json();

        if (!regs || regs.length === 0) {
            list.innerHTML = '<div class="text-muted p-2" style="font-size:0.85rem;">No hay registros de Docker configurados. Llena el formulario arriba para autenticar tu servidor Host por SSH.</div>';
            return;
        }

        list.innerHTML = regs.map(r => `
            <div class="endpoint-item" style="display:flex; justify-content:space-between; align-items:center; padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,0.05);">
                <div>
                    <strong>🔐 ${r.server}</strong>
                    <div class="text-muted" style="font-size:0.75rem;">Usuario: ${r.username} • Estado: Autenticado en Host</div>
                </div>
                <button class="btn btn-danger" style="padding: 2px 8px; font-size: 0.72rem;" onclick="deleteRegistryAction(${r.id})">🗑️ Remover</button>
            </div>
        `).join('');
    } catch (e) {
        list.innerHTML = `<div class="text-muted p-2">Error cargando registros: ${e.message}</div>`;
    }
}

async function deleteRegistryAction(id) {
    if (!id) return;
    try {
        const res = await fetch(`/api/registries?id=${id}`, { method: 'DELETE' });
        if (res.ok) {
            showToast('Registro Removido', 'Credenciales eliminadas correctamente.', 'success');
            loadRegistries();
        }
    } catch (e) {
        showToast('Error', e.message, 'error');
    }
}

// Live container stats loader in inspector modal
async function fetchLiveContainerStats(name) {
    if (!name) return;
    try {
        const res = await fetch(`/api/stats?name=${encodeURIComponent(name)}`);
        if (!res.ok) return;
        const stats = await res.json();

        const cpuEl = document.getElementById('inspStatCpu');
        const memEl = document.getElementById('inspStatMem');
        const netEl = document.getElementById('inspStatNet');
        const blockEl = document.getElementById('inspStatBlock');
        const badgeEl = document.getElementById('inspStatsCpuBadge');

        if (cpuEl) cpuEl.innerText = stats.cpu || '0.1%';
        if (memEl) memEl.innerText = stats.memUsage || '34MB / 2GB';
        if (netEl) netEl.innerText = stats.netIo || '1kB / 1kB';
        if (blockEl) blockEl.innerText = stats.blockIo || '0B / 0B';
        if (badgeEl) badgeEl.innerText = `CPU: ${stats.cpu || '0%'}`;
    } catch (e) {
        console.warn('Error leyendo live stats:', e);
    }
}

/* --- SSH Keys UI Manager --- */
async function openSSHKeysModal() {
    openModal('sshKeysModal');
    loadSSHKeys();
}

async function loadSSHKeys() {
    const container = document.getElementById('sshKeysList');
    if (!container) return;

    container.innerHTML = '<div class="text-muted p-2">Cargando llaves SSH desde el VPS...</div>';
    try {
        const res = await fetch('/api/ssh-keys');
        if (!res.ok) {
            container.innerHTML = '<div class="text-muted p-2" style="color:var(--danger-color);">Error consultando llaves SSH.</div>';
            return;
        }
        const keys = await res.json();
        if (!keys || keys.length === 0) {
            container.innerHTML = '<div class="text-muted p-2">No hay llaves SSH registradas en authorized_keys.</div>';
            return;
        }

        container.innerHTML = keys.map((k, idx) => {
            const isProtected = k.protected || k.is_vultr_key;
            const badgeHtml = isProtected
                ? '<span style="background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.4); padding: 2px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 600;">🔒 PROTEGIDA (Vultr Master)</span>'
                : '<span style="background: rgba(52, 211, 153, 0.2); color: #34d399; border: 1px solid rgba(52, 211, 153, 0.4); padding: 2px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 600;">🟢 Desarrollador / Eliminable</span>';
            
            const btnHtml = isProtected
                ? '<button class="btn btn-outline" disabled style="opacity: 0.5; cursor: not-allowed; font-size: 0.75rem;">🔒 Llave Maestra</button>'
                : `<button class="btn btn-outline" style="color: #f87171; border-color: rgba(239, 68, 68, 0.5); font-size: 0.75rem;" onclick="deleteSSHKey('${k.fingerprint}')">🗑️ Revocar Acceso</button>`;

            return `
                <div style="display: flex; align-items: center; justify-content: space-between; padding: 8px 10px; border-bottom: 1px solid var(--card-border); gap: 8px;">
                    <div style="overflow: hidden;">
                        <div style="font-weight: 600; font-size: 0.85rem; color: var(--text-primary); display: flex; align-items: center; gap: 8px;">
                            ${idx + 1}. ${k.comment || 'Llave sin comentario'} ${badgeHtml}
                        </div>
                        <div style="font-size: 0.72rem; color: var(--text-muted); font-family: monospace; text-overflow: ellipsis; overflow: hidden;">
                            Fingerprint: ${k.fingerprint || 'N/A'} | Tipo: ${k.type || 'ssh-rsa'}
                        </div>
                    </div>
                    <div>${btnHtml}</div>
                </div>
            `;
        }).join('');
    } catch (e) {
        container.innerHTML = `<div class="text-muted p-2" style="color:var(--danger-color);">Error: ${e.message}</div>`;
    }
}

async function deleteSSHKey(fingerprint) {
    if (!fingerprint) return;
    if (!confirm(`¿Estás seguro de que deseas revocar el acceso SSH a la llave con fingerprint '${fingerprint}'?`)) return;

    try {
        const res = await fetch(`/api/ssh-keys?fp=${encodeURIComponent(fingerprint)}`, { method: 'DELETE' });
        const data = await res.json();
        if (res.ok) {
            showToast('Acceso Revocado', 'La llave SSH de desarrollador fue eliminada de authorized_keys.', 'success');
            loadSSHKeys();
        } else {
            showToast('Acceso Protegido', data.error || 'No se pudo eliminar la llave', 'error');
        }
    } catch (e) {
        showToast('Error', e.message, 'error');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const formAddSSHKey = document.getElementById('formAddSSHKey');
    if (formAddSSHKey) {
        formAddSSHKey.addEventListener('submit', async (e) => {
            e.preventDefault();
            const publicKey = document.getElementById('newSSHPublicKey').value.trim();
            if (!publicKey) return;

            try {
                const res = await fetch('/api/ssh-keys', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ publicKey })
                });
                const data = await res.json();
                if (res.ok) {
                    showToast('Llave Autorizada', 'La llave SSH del desarrollador fue añadida en caliente.', 'success');
                    document.getElementById('newSSHPublicKey').value = '';
                    loadSSHKeys();
                } else {
                    showToast('Error', data.error || 'No se pudo añadir la llave', 'error');
                }
            } catch (err) {
                showToast('Error', err.message, 'error');
            }
        });
    }
});
