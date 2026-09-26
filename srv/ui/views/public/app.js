/**
 * Tarhiata Cloud Studio — Controller v4.0
 * Responsive, Card-based PaaS Controller (Railway / Vercel style)
 */

document.addEventListener('DOMContentLoaded', () => {
    // --- API Key & Session Security Interceptor ---
    try {
        const urlParams = new URLSearchParams(window.location.search);
        const initialKey = urlParams.get('key') || urlParams.get('api_key');
        if (initialKey) {
            sessionStorage.setItem('tarhiata_api_key', initialKey);
        }
        const originalFetch = window.fetch;
        window.fetch = function(input, init) {
            const key = sessionStorage.getItem('tarhiata_api_key') || localStorage.getItem('tarhiata_api_key');
            if (key) {
                init = init || {};
                if (!init.headers) {
                    init.headers = {};
                }
                if (init.headers instanceof Headers) {
                    if (!init.headers.has('X-API-Key')) init.headers.set('X-API-Key', key);
                } else if (Array.isArray(init.headers)) {
                    init.headers.push(['X-API-Key', key]);
                } else {
                    if (!init.headers['X-API-Key']) init.headers['X-API-Key'] = key;
                }
            }
            return originalFetch.call(this, input, init);
        };
    } catch (secErr) {
        console.debug('Error configurando interceptor de seguridad:', secErr);
    }

    // --- Application State ---
    let servers = [];
    let activeServer = null;
    let selectedServerName = null;
    let selectedInspection = null;
    let currentHostServices = [];
    let swarmServicesCache = [];
    let swarmDatabasesCache = [];
    let swarmNodesCache = [];
    let currentServiceLinks = [];
    let modalMode = 'local';
    let dbDeployMode = 'single-node';

    // --- Helper Utilities ---
    function getDefaultPort(engine) {
        const eng = (engine || '').toLowerCase();
        if (eng === 'redis') return 6379;
        if (eng === 'mysql' || eng === 'mariadb') return 3306;
        if (eng === 'mongo' || eng === 'mongodb') return 27017;
        if (eng === 'minio') return 9000;
        return 5432;
    }

    function debounce(fn, waitMs = 150) {
        let timeout;
        return function(...args) {
            clearTimeout(timeout);
            timeout = setTimeout(() => fn.apply(this, args), waitMs);
        };
    }

    function getGaugeColor(pct) {
        if (pct >= 85) return 'var(--accent-danger, #ef4444)';
        if (pct >= 60) return 'var(--accent-warning, #f59e0b)';
        return 'var(--accent-success, #10b981)';
    }

    async function copyToClipboard(text) {
        if (!text) return false;
        if (navigator.clipboard && window.isSecureContext) {
            try {
                await navigator.clipboard.writeText(text);
                return true;
            } catch (clipErr) {
                console.debug('clipboard.writeText no soportado o denegado:', clipErr);
            }
        }
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.opacity = '0';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        let ok = false;
        try {
            ok = document.execCommand('copy');
        } catch (execErr) {
            console.debug('execCommand copy falló:', execErr);
        }
        document.body.removeChild(textArea);
        return ok;
    }

    // --- DOM Elements: Top Bar & Server Switcher ---
    const serverSwitcherBtn = document.getElementById('serverSwitcherBtn');
    const serverPopover = document.getElementById('serverPopover');
    const topActiveName = document.getElementById('topActiveName');
    const topActiveHost = document.getElementById('topActiveHost');
    const topActiveDot = document.getElementById('topActiveDot');
    const topActiveLatency = document.getElementById('topActiveLatency');
    const btnTopActiveTerminal = document.getElementById('btnTopActiveTerminal');
    const btnTestAll = document.getElementById('btnTestAll');
    const btnOpenAddModal = document.getElementById('btnOpenAddModal');

    // --- DOM Elements: Popover & Fleet ---
    const serverSwitcherWrap = document.getElementById('serverSwitcherWrap');
    const fleetList = document.getElementById('fleetList');
    const fleetSearchInput = document.getElementById('fleetSearchInput');
    const btnSidebarOpenWorker = document.getElementById('btnSidebarOpenWorker');
    const btnGlobalDeploy = document.getElementById('btnGlobalDeploy');

    // --- Skeleton Loader Helper ---
    function deactivateInitialSkeletons() {
        document.body.classList.remove('is-initial-loading');
        document.body.classList.remove('is-server-loading');
    }

    // --- DOM Elements: Bento Server Hero ---
    const serverBentoHero = document.getElementById('serverBentoHero');
    const deskStatusDot = document.getElementById('deskStatusDot');
    const deskServerTitle = document.getElementById('deskServerTitle');
    const deskModeBadge = document.getElementById('deskModeBadge');
    const deskActiveBadge = document.getElementById('deskActiveBadge');
    const deskHost = document.getElementById('deskHost');
    const deskOS = document.getElementById('deskOS');
    const deskDocker = document.getElementById('deskDocker');
    const deskSwarm = document.getElementById('deskSwarm');
    const swarmStateBadge = document.getElementById('swarmStateBadge');
    const btnBootstrapSwarm = document.getElementById('btnBootstrapSwarm');

    const btnDeskActivate = document.getElementById('btnDeskActivate');
    const btnDeskRefresh = document.getElementById('btnDeskRefresh');
    const btnDeskTerminal = document.getElementById('btnDeskTerminal');

    // Bento Mini-Gauges
    const tileCpuPct = document.getElementById('tileCpuPct');
    const tileCpuCores = document.getElementById('tileCpuCores');
    const tileCpuBar = document.getElementById('tileCpuBar');
    const tileCpuLoad = document.getElementById('tileCpuLoad');

    const tileRamUsed = document.getElementById('tileRamUsed');
    const tileRamTotal = document.getElementById('tileRamTotal');
    const tileRamPct = document.getElementById('tileRamPct');
    const tileRamBar = document.getElementById('tileRamBar');

    const tileDiskUsed = document.getElementById('tileDiskUsed');
    const tileDiskTotal = document.getElementById('tileDiskTotal');
    const tileDiskPct = document.getElementById('tileDiskPct');
    const tileDiskBar = document.getElementById('tileDiskBar');

    const tileUptime = document.getElementById('tileUptime');
    const tileDistroTag = document.getElementById('tileDistroTag');
    const tileHostTag = document.getElementById('tileHostTag');
    const tileLatencyDisplay = document.getElementById('tileLatencyDisplay');

    // Dashboards Web Links
    const linkPortainer = document.getElementById('linkPortainer');
    const linkDozzle = document.getElementById('linkDozzle');
    const linkTraefik = document.getElementById('linkTraefik');

    // --- Navigation Tabs Elements ---
    const tabSwarmServices = document.getElementById('tabSwarmServices');
    const tabSwarmDatabases = document.getElementById('tabSwarmDatabases');
    const tabSwarmTopology = document.getElementById('tabSwarmTopology');
    const tabHostServices = document.getElementById('tabHostServices');
    const tabHostDevices = document.getElementById('tabHostDevices');

    const tabServicesCount = document.getElementById('tabServicesCount');
    const tabDatabasesCount = document.getElementById('tabDatabasesCount');
    const tabTopologyCount = document.getElementById('tabTopologyCount');
    const tabHostCount = document.getElementById('tabHostCount');
    const tabDevicesCount = document.getElementById('tabDevicesCount');

    // Content Views
    const viewSwarmServices = document.getElementById('viewSwarmServices');
    const viewSwarmDatabases = document.getElementById('viewSwarmDatabases');
    const viewSwarmTopology = document.getElementById('viewSwarmTopology');
    const viewHostServices = document.getElementById('viewHostServices');
    const viewHostDevices = document.getElementById('viewHostDevices');

    // Hardware view elements
    const btnRefreshDevices = document.getElementById('btnRefreshDevices');
    const devicesSummaryText = document.getElementById('devicesSummaryText');
    const hwStorageCount = document.getElementById('hwStorageCount');
    const hwGpuCount = document.getElementById('hwGpuCount');
    const hwUsbCount = document.getElementById('hwUsbCount');
    const hwDisplaysCount = document.getElementById('hwDisplaysCount');
    const hwStorageBadge = document.getElementById('hwStorageBadge');
    const hwGpuBadge = document.getElementById('hwGpuBadge');
    const hwUsbBadge = document.getElementById('hwUsbBadge');
    const hwDisplaysBadge = document.getElementById('hwDisplaysBadge');
    const hwPciBadge = document.getElementById('hwPciBadge');
    const hwStorageTableBody = document.getElementById('hwStorageTableBody');
    const hwGpuCardsContainer = document.getElementById('hwGpuCardsContainer');
    const hwUsbTableBody = document.getElementById('hwUsbTableBody');
    const hwDisplaysContainer = document.getElementById('hwDisplaysContainer');
    const hwPciTableBody = document.getElementById('hwPciTableBody');

    // Card Grids & Empty States
    const swarmServicesCardsGrid = document.getElementById('swarmServicesCardsGrid');
    const swarmServicesEmpty = document.getElementById('swarmServicesEmpty');
    const swarmDatabasesCardsGrid = document.getElementById('swarmDatabasesCardsGrid');
    const swarmDatabasesEmpty = document.getElementById('swarmDatabasesEmpty');

    // Fallback table bodies
    const swarmServicesTableBody = document.getElementById('swarmServicesTableBody');
    const swarmDatabasesTableBody = document.getElementById('swarmDatabasesTableBody');
    const swarmNodesTableBody = document.getElementById('swarmNodesTableBody');
    const linksTableBody = document.getElementById('linksTableBody');
    const topologyServicesTableBody = document.getElementById('topologyServicesTableBody');
    const topologyServicesCountBadge = document.getElementById('topologyServicesCountBadge');
    const serviceListFrom = document.getElementById('serviceListFrom');
    const serviceListTo = document.getElementById('serviceListTo');

    // System Services elements
    const serviceSearchInput = document.getElementById('serviceSearchInput');
    const servicesSummaryText = document.getElementById('servicesSummaryText');
    const servicesTableBody = document.getElementById('servicesTableBody');

    // --- Skeleton Render Engine ---
    function renderSkeletonCards() {
        if (swarmServicesCardsGrid) {
            if (swarmServicesEmpty) swarmServicesEmpty.style.display = 'none';
            swarmServicesCardsGrid.innerHTML = `
                <div class="skeleton-card">
                    <div class="skeleton-card-header">
                        <div class="skeleton-card-title-group">
                            <div class="skeleton-box skeleton-card-icon"></div>
                            <div class="skeleton-card-text">
                                <div class="skeleton-box skeleton-card-line-lg"></div>
                                <div class="skeleton-box skeleton-card-line-sm"></div>
                            </div>
                        </div>
                        <div class="skeleton-box skeleton-card-pill"></div>
                    </div>
                    <div class="skeleton-box skeleton-card-url" style="margin-top:14px;"></div>
                    <div class="skeleton-card-footer">
                        <div class="skeleton-box skeleton-card-pill" style="width:60px;"></div>
                        <div class="skeleton-card-actions">
                            <div class="skeleton-box skeleton-card-btn"></div>
                            <div class="skeleton-box skeleton-card-btn"></div>
                        </div>
                    </div>
                </div>
                <div class="skeleton-card">
                    <div class="skeleton-card-header">
                        <div class="skeleton-card-title-group">
                            <div class="skeleton-box skeleton-card-icon"></div>
                            <div class="skeleton-card-text">
                                <div class="skeleton-box skeleton-card-line-lg"></div>
                                <div class="skeleton-box skeleton-card-line-sm"></div>
                            </div>
                        </div>
                        <div class="skeleton-box skeleton-card-pill"></div>
                    </div>
                    <div class="skeleton-box skeleton-card-url" style="margin-top:14px;"></div>
                    <div class="skeleton-card-footer">
                        <div class="skeleton-box skeleton-card-pill" style="width:60px;"></div>
                        <div class="skeleton-card-actions">
                            <div class="skeleton-box skeleton-card-btn"></div>
                            <div class="skeleton-box skeleton-card-btn"></div>
                        </div>
                    </div>
                </div>
                <div class="skeleton-card">
                    <div class="skeleton-card-header">
                        <div class="skeleton-card-title-group">
                            <div class="skeleton-box skeleton-card-icon"></div>
                            <div class="skeleton-card-text">
                                <div class="skeleton-box skeleton-card-line-lg"></div>
                                <div class="skeleton-box skeleton-card-line-sm"></div>
                            </div>
                        </div>
                        <div class="skeleton-box skeleton-card-pill"></div>
                    </div>
                    <div class="skeleton-box skeleton-card-url" style="margin-top:14px;"></div>
                    <div class="skeleton-card-footer">
                        <div class="skeleton-box skeleton-card-pill" style="width:60px;"></div>
                        <div class="skeleton-card-actions">
                            <div class="skeleton-box skeleton-card-btn"></div>
                            <div class="skeleton-box skeleton-card-btn"></div>
                        </div>
                    </div>
                </div>
            `;
        }
        if (swarmDatabasesCardsGrid) {
            if (swarmDatabasesEmpty) swarmDatabasesEmpty.style.display = 'none';
            swarmDatabasesCardsGrid.innerHTML = `
                <div class="skeleton-card">
                    <div class="skeleton-card-header">
                        <div class="skeleton-card-title-group">
                            <div class="skeleton-box skeleton-card-icon"></div>
                            <div class="skeleton-card-text">
                                <div class="skeleton-box skeleton-card-line-lg"></div>
                                <div class="skeleton-box skeleton-card-line-sm"></div>
                            </div>
                        </div>
                        <div class="skeleton-box skeleton-card-pill"></div>
                    </div>
                    <div class="skeleton-box skeleton-card-url" style="margin-top:14px;"></div>
                </div>
                <div class="skeleton-card">
                    <div class="skeleton-card-header">
                        <div class="skeleton-card-title-group">
                            <div class="skeleton-box skeleton-card-icon"></div>
                            <div class="skeleton-card-text">
                                <div class="skeleton-box skeleton-card-line-lg"></div>
                                <div class="skeleton-box skeleton-card-line-sm"></div>
                            </div>
                        </div>
                        <div class="skeleton-box skeleton-card-pill"></div>
                    </div>
                    <div class="skeleton-box skeleton-card-url" style="margin-top:14px;"></div>
                </div>
            `;
        }

        if (swarmNodesTableBody) {
            swarmNodesTableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty"><div class="skeleton-box" style="height:20px; width:60%; margin:auto;"></div></td></tr>`;
        }
        if (linkPortainer) linkPortainer.href = '#';
        if (linkDozzle) linkDozzle.href = '#';
        if (linkTraefik) linkTraefik.href = '#';
        if (swarmStateBadge) {
            swarmStateBadge.className = 'swarm-status-tag skeleton-target';
            swarmStateBadge.style.color = '';
            swarmStateBadge.style.background = '';
            swarmStateBadge.textContent = 'Verificando...';
        }
        if (deskSwarm) deskSwarm.textContent = '—';
    }

    function activateServerLoadingSkeletons(server) {
        document.body.classList.add('is-server-loading');

        if (deskServerTitle) deskServerTitle.textContent = server.name;
        if (deskHost) deskHost.textContent = `${server.user || 'root'}@${server.host || 'localhost'}${server.port > 0 ? `:${server.port}` : ''}`;
        if (deskModeBadge) deskModeBadge.textContent = (server.cloudProvider || 'SSH').toUpperCase();
        if (deskActiveBadge) deskActiveBadge.style.display = server.isActive ? 'inline-block' : 'none';
        if (btnDeskActivate) btnDeskActivate.style.display = server.isActive ? 'none' : 'inline-flex';

        if (topActiveName) topActiveName.textContent = server.name;
        if (topActiveHost) topActiveHost.textContent = server.host || 'localhost';
        if (topActiveLatency) topActiveLatency.textContent = 'Midiendo...';
        if (topActiveDot) topActiveDot.className = 'status-dot status-pending';

        if (deskStatusDot) deskStatusDot.className = 'ops-status-dot status-pending';
        if (tileLatencyDisplay) tileLatencyDisplay.textContent = 'Midiendo...';
        if (tileUptime) tileUptime.textContent = 'Conectando...';

        renderSkeletonCards();
    }

    // Modales
    const serverModal = document.getElementById('serverModal');
    const btnCloseModal = document.getElementById('btnCloseModal');
    const btnCancelServer = document.getElementById('btnCancelServer');
    const tabLocal = document.getElementById('tabLocal');
    const tabRemote = document.getElementById('tabRemote');
    const tabCloud = document.getElementById('tabCloud');
    const formServer = document.getElementById('formServer');
    const localFastNotice = document.getElementById('localFastNotice');
    const cloudFields = document.getElementById('cloudFields');
    const cfgProvider = document.getElementById('cfgProvider');
    const cfgRegion = document.getElementById('cfgRegion');
    const cfgToken = document.getElementById('cfgToken');
    const cfgPlan = document.getElementById('cfgPlan');
    const standardFields = document.getElementById('standardFields');
    const nameField = document.getElementById('nameField');
    const cfgName = document.getElementById('cfgName');
    const hostField = document.getElementById('hostField');
    const cfgHost = document.getElementById('cfgHost');
    const userField = document.getElementById('userField');
    const cfgUser = document.getElementById('cfgUser');
    const portField = document.getElementById('portField');
    const cfgPort = document.getElementById('cfgPort');
    const keyField = document.getElementById('keyField');
    const cfgKey = document.getElementById('cfgKey');
    const cfgIsActive = document.getElementById('cfgIsActive');
    const modalTestResult = document.getElementById('modalTestResult');
    const btnModalTest = document.getElementById('btnModalTest');
    const btnModalTestText = document.getElementById('btnModalTestText');
    const btnModalSave = document.getElementById('btnModalSave');
    const btnModalSaveText = document.getElementById('btnModalSaveText');

    // Deploy Modal Elements
    const deployModal = document.getElementById('deployModal');
    const btnOpenDeployModal = document.getElementById('btnOpenDeployModal');
    const btnCloseDeployModal = document.getElementById('btnCloseDeployModal');
    const btnCancelDeploy = document.getElementById('btnCancelDeploy');
    const formDeploy = document.getElementById('formDeploy');
    const depName = document.getElementById('depName');
    const depPort = document.getElementById('depPort');
    const depImage = document.getElementById('depImage');
    const depDomain = document.getElementById('depDomain');
    const depDomainDnsFeedback = document.getElementById('depDomainDnsFeedback');
    const depDB = document.getElementById('depDB');
    const depEnv = document.getElementById('depEnv');
    const btnSubmitDeploy = document.getElementById('btnSubmitDeploy');

    // Database Modal Elements
    const dbModal = document.getElementById('dbModal');
    const btnOpenDeployDBModal = document.getElementById('btnOpenDeployDBModal');
    const btnCloseDBModal = document.getElementById('btnCloseDBModal');
    const btnCancelDB = document.getElementById('btnCancelDB');
    const formDeployDB = document.getElementById('formDeployDB');
    const tabDBLocal = document.getElementById('tabDBLocal');
    const tabDBNode = document.getElementById('tabDBNode');
    const tabDBExternal = document.getElementById('tabDBExternal');
    const dbName = document.getElementById('dbName');
    const dbEngine = document.getElementById('dbEngine');
    const dbUrlField = document.getElementById('dbUrlField');
    const dbExternalURL = document.getElementById('dbExternalURL');
    const dbPathField = document.getElementById('dbPathField');
    const dbVolumePath = document.getElementById('dbVolumePath');
    const dbPortField = document.getElementById('dbPortField');
    const dbPort = document.getElementById('dbPort');
    const dbTargetNodeField = document.getElementById('dbTargetNodeField');
    const dbTargetNode = document.getElementById('dbTargetNode');
    const btnSubmitDB = document.getElementById('btnSubmitDB');
    const btnSubmitDBText = document.getElementById('btnSubmitDBText');

    // Link Modal Elements
    const linkModal = document.getElementById('linkModal');
    const btnOpenLinkModal = document.getElementById('btnOpenLinkModal');
    const btnCloseLinkModal = document.getElementById('btnCloseLinkModal');
    const btnCancelLink = document.getElementById('btnCancelLink');
    const formLink = document.getElementById('formLink');
    const linkFrom = document.getElementById('linkFrom');
    const linkTo = document.getElementById('linkTo');
    const linkVar = document.getElementById('linkVar');
    const btnSubmitLink = document.getElementById('btnSubmitLink');

    // Backups Modal Elements
    const backupsModal = document.getElementById('backupsModal');
    const btnOpenBackupsModal = document.getElementById('btnOpenBackupsModal');
    const btnCloseBackupsModal = document.getElementById('btnCloseBackupsModal');
    const btnDismissBackupsModal = document.getElementById('btnDismissBackupsModal');
    const btnRefreshBackupsModal = document.getElementById('btnRefreshBackupsModal');
    const backupsTableBody = document.getElementById('backupsTableBody');

    // Edit Service Modal Elements
    const editServiceModal = document.getElementById('editServiceModal');
    const btnCloseEditServiceModal = document.getElementById('btnCloseEditServiceModal');
    const btnCancelEditService = document.getElementById('btnCancelEditService');
    const formEditService = document.getElementById('formEditService');
    const editServiceTitle = document.getElementById('editServiceTitle');
    const editServiceName = document.getElementById('editServiceName');
    const editServiceExpose = document.getElementById('editServiceExpose');
    const editDomainField = document.getElementById('editDomainField');
    const editServiceDomain = document.getElementById('editServiceDomain');
    const editDomainDnsFeedback = document.getElementById('editDomainDnsFeedback');
    const editServicePort = document.getElementById('editServicePort');
    const btnSubmitEditService = document.getElementById('btnSubmitEditService');

    // Worker Modal Elements
    const workerModal = document.getElementById('workerModal');
    const btnOpenWorkerModal = document.getElementById('btnOpenWorkerModal');
    const btnCopyJoinToken = document.getElementById('btnCopyJoinToken');
    const btnCloseWorkerModal = document.getElementById('btnCloseWorkerModal');
    const btnCancelWorker = document.getElementById('btnCancelWorker');
    const formWorker = document.getElementById('formWorker');
    const workerName = document.getElementById('workerName');
    const workerProvider = document.getElementById('workerProvider');
    const workerApiKey = document.getElementById('workerApiKey');
    const workerRegion = document.getElementById('workerRegion');
    const workerPlan = document.getElementById('workerPlan');
    const workerLabel = document.getElementById('workerLabel');
    const workerLogsBox = document.getElementById('workerLogsBox');
    const workerLogsContent = document.getElementById('workerLogsContent');
    const btnSubmitWorker = document.getElementById('btnSubmitWorker');
    const btnSubmitWorkerText = document.getElementById('btnSubmitWorkerText');

    // Logs Modal Elements
    const logsModal = document.getElementById('logsModal');
    const btnCloseLogsModal = document.getElementById('btnCloseLogsModal');
    const btnDismissLogs = document.getElementById('btnDismissLogs');
    const logsStatusDot = document.getElementById('logsStatusDot');
    const logsServiceNameTitle = document.getElementById('logsServiceNameTitle');
    const logsSearchInput = document.getElementById('logsSearchInput');
    const logsTailSelect = document.getElementById('logsTailSelect');
    const logsLiveToggle = document.getElementById('logsLiveToggle');
    const logsAutoscrollToggle = document.getElementById('logsAutoscrollToggle');
    const btnRestartFromLogs = document.getElementById('btnRestartFromLogs');
    const btnRefreshLogs = document.getElementById('btnRefreshLogs');
    const btnCopyLogs = document.getElementById('btnCopyLogs');
    const btnDownloadLogs = document.getElementById('btnDownloadLogs');
    const logsTerminalViewport = document.getElementById('logsTerminalViewport');
    const logsTerminalContent = document.getElementById('logsTerminalContent');
    const logsLineCount = document.getElementById('logsLineCount');
    const logsLastUpdate = document.getElementById('logsLastUpdate');

    let currentLogsServiceName = '';
    let logsPollTimer = null;
    let rawLogsText = '';

    // Env Modal Elements
    const envModal = document.getElementById('envModal');
    const btnCloseEnvModal = document.getElementById('btnCloseEnvModal');
    const btnCancelEnv = document.getElementById('btnCancelEnv');
    const btnSaveEnv = document.getElementById('btnSaveEnv');
    const envModalServiceName = document.getElementById('envModalServiceName');
    const envTabTable = document.getElementById('envTabTable');
    const envTabRaw = document.getElementById('envTabRaw');
    const btnAddEnvRow = document.getElementById('btnAddEnvRow');
    const btnCopyEnv = document.getElementById('btnCopyEnv');
    const envTableView = document.getElementById('envTableView');
    const envRawView = document.getElementById('envRawView');
    const envTableBody = document.getElementById('envTableBody');
    const envRawTextarea = document.getElementById('envRawTextarea');
    const envModalStatus = document.getElementById('envModalStatus');

    let currentEnvServiceName = '';
    let currentEnvMode = 'table';

    // --- Server Switcher Popover Interactions ---
    serverSwitcherBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpen = serverPopover.style.display === 'flex';
        serverPopover.style.display = isOpen ? 'none' : 'flex';
        serverSwitcherBtn.classList.toggle('open', !isOpen);
        if (!isOpen) {
            fleetSearchInput.value = '';
            renderFleetDirectory();
            setTimeout(() => fleetSearchInput.focus(), 50);
        }
    });

    document.addEventListener('click', (e) => {
        if (!serverSwitcherWrap.contains(e.target)) {
            serverPopover.style.display = 'none';
            serverSwitcherBtn.classList.remove('open');
        }
    });

    fleetSearchInput.addEventListener('input', debounce(() => {
        renderFleetDirectory();
    }, 150));

    // Soporte global tecla Escape para cerrar modales y popover
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const modals = [serverModal, deployModal, linkModal, dbModal, editServiceModal, workerModal, logsModal, envModal, volumeModal, terminalModal, sslModal];
            modals.forEach(m => {
                if (m && m.style.display !== 'none' && m.style.display !== '') {
                    if (m === logsModal) stopLogsPolling();
                    m.style.display = 'none';
                }
            });
            if (serverPopover) serverPopover.style.display = 'none';
            if (serverSwitcherBtn) serverSwitcherBtn.classList.remove('open');
        }
    });

    // --- Tab Switching Navigation ---
    function activateTab(tabName) {
        const tabs = [tabSwarmServices, tabSwarmDatabases, tabHostServices, tabHostDevices, tabSwarmTopology].filter(Boolean);
        const views = [viewSwarmServices, viewSwarmDatabases, viewHostServices, viewHostDevices, viewSwarmTopology].filter(Boolean);
        tabs.forEach(t => { if (t) t.classList.remove('active'); });
        views.forEach(v => { if (v) v.style.display = 'none'; });

        const effectiveTab = (tabName === 'topology') ? 'services' : tabName;
        if (location.hash !== `#${effectiveTab}`) {
            history.replaceState(null, '', `#${effectiveTab}`);
        }

        if (effectiveTab === 'services' && tabSwarmServices && viewSwarmServices) {
            tabSwarmServices.classList.add('active');
            viewSwarmServices.style.display = 'block';
            renderAppCards(swarmServicesCache);
            renderNodesTable(swarmNodesCache);
            renderTopologyServicesTable(swarmServicesCache, swarmDatabasesCache, currentServiceLinks);
            if (selectedServerName) loadServiceLinks();
        }
        if (effectiveTab === 'databases' && tabSwarmDatabases && viewSwarmDatabases) {
            tabSwarmDatabases.classList.add('active');
            viewSwarmDatabases.style.display = 'block';
        }
        if (effectiveTab === 'host' && tabHostServices && viewHostServices) {
            tabHostServices.classList.add('active');
            viewHostServices.style.display = 'block';
            renderServicesTable(currentHostServices);
        }
        if (effectiveTab === 'devices' && tabHostDevices && viewHostDevices) {
            tabHostDevices.classList.add('active');
            viewHostDevices.style.display = 'block';
            loadHostDevices();
        }
    }

    if (tabSwarmServices) tabSwarmServices.addEventListener('click', () => activateTab('services'));
    if (tabSwarmDatabases) tabSwarmDatabases.addEventListener('click', () => activateTab('databases'));
    if (tabHostServices) tabHostServices.addEventListener('click', () => activateTab('host'));
    if (tabHostDevices) tabHostDevices.addEventListener('click', () => activateTab('devices'));
    if (tabSwarmTopology) tabSwarmTopology.addEventListener('click', () => activateTab('services'));
    if (btnRefreshDevices) btnRefreshDevices.addEventListener('click', () => loadHostDevices(true));

    // --- Render Popover Server Directory & Quick Fleet Chips ---
    function renderFleetDirectory() {
        if (fleetList) {
            fleetList.innerHTML = '';
            if (servers.length === 0) {
                fleetList.innerHTML = `<div class="popover-loading">Sin servidores registrados</div>`;
            } else {
                const query = (fleetSearchInput ? fleetSearchInput.value : '').toLowerCase().trim();
                const filtered = servers.filter(s =>
                    !query ||
                    (s.name && s.name.toLowerCase().includes(query)) ||
                    (s.host && s.host.toLowerCase().includes(query))
                );

                if (filtered.length === 0) {
                    fleetList.innerHTML = `<div class="popover-loading">No se encontraron resultados</div>`;
                } else {
                    filtered.forEach(s => {
                        const isSelected = (s.name === selectedServerName);
                        const row = document.createElement('div');
                        row.className = `popover-server-row ${isSelected ? 'selected' : ''}`;
                        row.innerHTML = `
                            <div style="display:flex; align-items:center; gap:8px;">
                                <span class="status-dot ${s.isActive ? 'status-online' : 'status-pending'}" id="fleet-dot-${escapeHtml(s.name)}"></span>
                                <div>
                                    <div style="font-weight:700; color:#fff; font-size:0.84rem;">${escapeHtml(s.name)}</div>
                                    <div style="font-size:0.72rem; color:var(--text-muted); font-family:var(--font-mono);">${escapeHtml(s.host || 'localhost')}</div>
                                </div>
                            </div>
                            <div style="display:flex; align-items:center; gap:6px;">
                                ${s.isActive ? '<span class="t-badge" style="color:#a5b4fc; background:rgba(99,102,241,0.2);">Activo</span>' : ''}
                                <span class="t-badge">${escapeHtml((s.cloudProvider || 'SSH').toUpperCase())}</span>
                            </div>
                        `;

                        row.addEventListener('click', () => {
                            selectServer(s.name);
                            if (serverPopover) serverPopover.style.display = 'none';
                            if (serverSwitcherBtn) serverSwitcherBtn.classList.remove('open');
                        });

                        fleetList.appendChild(row);
                    });
                }
            }
        }
    }

    // --- Select Server & Load Hero Telemetry ---
    async function selectServer(serverName) {
        selectedServerName = serverName;
        try {
            localStorage.setItem('tarhiata_last_server', serverName);
        } catch (storageErr) {
            console.debug('No se pudo guardar último servidor en localStorage:', storageErr);
        }

        let s = servers.find(x => x.name === serverName);
        if (!s) {
            if (activeServer && activeServer.name === serverName) {
                s = activeServer;
            } else if (servers.length > 0) {
                s = servers[0];
                serverName = s.name;
                selectedServerName = s.name;
            }
        }

        renderFleetDirectory();

        if (!s) return;

        // Activar estados de esqueleto de carga de inmediato para el nuevo VPS
        activateServerLoadingSkeletons(s);

        await refreshServerTelemetry(serverName);
        if (location.hash === '#devices' || (viewHostDevices && viewHostDevices.style.display !== 'none')) {
            loadHostDevices();
        }
    }

    // --- Refresh Telemetry (Parallel Host & Swarm + Silent Refresh) ---
    let telemetryRequestId = 0;
    async function refreshServerTelemetry(serverName, isSilent = false) {
        if (!serverName) return;
        const currentReq = ++telemetryRequestId;

        if (!isSilent) {
            btnDeskRefresh.disabled = true;
            deskStatusDot.className = 'ops-status-dot status-pending';
        }
        const tStart = performance.now();

        try {
            // Disparar inspección de host y estado de Swarm en paralelo para máxima velocidad y cero retardo
            const freshParam = !isSilent ? '&fresh=true' : '';
            const [hostRes, swarmRes] = await Promise.all([
                fetch(`/api/host/inspect?server=${encodeURIComponent(serverName)}${freshParam}`),
                fetch(`/api/swarm/status?server=${encodeURIComponent(serverName)}${freshParam}`)
            ]);
            const latency = Math.round(performance.now() - tStart);

            if (currentReq !== telemetryRequestId || serverName !== selectedServerName) return;

            if (!hostRes.ok) {
                const errText = await hostRes.text();
                deskStatusDot.className = 'ops-status-dot status-offline';
                tileUptime.textContent = 'Sin conexión';
                tileLatencyDisplay.textContent = 'Latencia: —';
                if (topActiveName.textContent === serverName) {
                    topActiveDot.className = 'status-dot status-offline';
                    topActiveLatency.textContent = 'OFFLINE';
                }
                const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
                if (dotFleet) dotFleet.className = 'status-dot status-offline';
                if (!isSilent) showToast(`Fallo al sondear host '${serverName}': ${errText}`, 'error');
                if (currentReq === telemetryRequestId && serverName === selectedServerName) {
                    processSwarmStatus(serverName, { active: false }, isSilent);
                }
                btnBootstrapSwarm.style.display = 'inline-flex';
                btnBootstrapSwarm.disabled = false;
                btnBootstrapSwarm.classList.remove('swarm-configured');
                btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
                btnBootstrapSwarm.title = `Instalar Framework de orquestación en '${serverName}'`;
                return;
            }

            const data = await hostRes.json();
            if (currentReq !== telemetryRequestId || serverName !== selectedServerName) return;

            selectedInspection = data;
            deskStatusDot.className = 'ops-status-dot status-online';
            tileLatencyDisplay.textContent = `Latencia: ${latency} ms`;

            if (topActiveName.textContent === serverName) {
                topActiveDot.className = 'status-dot status-online';
                topActiveLatency.textContent = `${latency} ms`;
            }
            const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
            if (dotFleet) dotFleet.className = 'status-dot status-online';

            const m = data.metrics || {};
            deskOS.textContent = m.os || 'Linux';
            if (tileDistroTag) tileDistroTag.textContent = m.os || 'Linux';
            tileHostTag.textContent = m.hostname || data.host;
            tileUptime.textContent = m.uptime || 'En línea';

            const dockerSvc = (data.services || []).find(sv => sv.name && sv.name.toLowerCase().includes('docker'));
            if (dockerSvc) {
                deskDocker.textContent = dockerSvc.subState === 'running' ? 'Activo' : (dockerSvc.activeState || 'Inactivo');
            }

            // CPU Gauge
            tileCpuCores.textContent = `${m.cpuCores || 1} Cores`;
            const cpuVal = (m.cpuPercent || 0).toFixed(1);
            tileCpuPct.textContent = `${cpuVal}%`;
            tileCpuBar.style.width = `${Math.min(100, Math.max(0, m.cpuPercent || 0))}%`;
            tileCpuBar.style.backgroundColor = getGaugeColor(m.cpuPercent || 0);
            tileCpuLoad.textContent = `Carga: ${m.loadAvg || '—'}`;

            // RAM Gauge
            tileRamUsed.textContent = `${m.memoryUsedMb || 0} MB`;
            tileRamTotal.textContent = `Total: ${m.memoryTotalMb || 0} MB`;
            const ramVal = (m.memoryPercent || 0).toFixed(1);
            tileRamPct.textContent = `${ramVal}%`;
            tileRamBar.style.width = `${Math.min(100, Math.max(0, m.memoryPercent || 0))}%`;
            tileRamBar.style.backgroundColor = getGaugeColor(m.memoryPercent || 0);

            // Disk Gauge
            tileDiskUsed.textContent = `${(m.diskUsedGb || 0).toFixed(1)} GB`;
            tileDiskTotal.textContent = `Total: ${(m.diskTotalGb || 0).toFixed(1)} GB`;
            const diskVal = (m.diskPercent || 0).toFixed(1);
            tileDiskPct.textContent = `${diskVal}%`;
            tileDiskBar.style.width = `${Math.min(100, Math.max(0, m.diskPercent || 0))}%`;
            tileDiskBar.style.backgroundColor = getGaugeColor(m.diskPercent || 0);

            // Host Services list (actualizar solo si cambió la longitud o no es silencioso)
            const newHostServices = data.services || [];
            if (!isSilent || currentHostServices.length !== newHostServices.length) {
                currentHostServices = newHostServices;
                if (tabHostCount) tabHostCount.textContent = currentHostServices.length;
                renderServicesTable(currentHostServices);
            }

            // Procesar estado de Swarm
            if (swarmRes.ok) {
                const swarmData = await swarmRes.json();
                if (currentReq === telemetryRequestId && serverName === selectedServerName) {
                    processSwarmStatus(serverName, swarmData, isSilent);
                }
            } else {
                if (currentReq === telemetryRequestId && serverName === selectedServerName) {
                    processSwarmStatus(serverName, { active: false }, isSilent);
                }
            }

        } catch (err) {
            if (currentReq === telemetryRequestId && serverName === selectedServerName) {
                processSwarmStatus(serverName, { active: false }, isSilent);
            }
            deskStatusDot.className = 'ops-status-dot status-offline';
            tileLatencyDisplay.textContent = 'Latencia: —';
            if (topActiveName.textContent === serverName) {
                topActiveDot.className = 'status-dot status-offline';
                topActiveLatency.textContent = 'OFFLINE';
            }
            const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
            if (dotFleet) dotFleet.className = 'status-dot status-offline';
            btnBootstrapSwarm.style.display = 'inline-flex';
            btnBootstrapSwarm.disabled = false;
            btnBootstrapSwarm.classList.remove('swarm-configured');
            btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
            btnBootstrapSwarm.title = `Instalar Framework de orquestación en '${serverName}'`;
        } finally {
            if (!isSilent) btnDeskRefresh.disabled = false;
            deactivateInitialSkeletons();
        }
    }

    function formatDockerVersion(raw) {
        if (!raw) return '—';
        const match = raw.match(/Docker version\s+([0-9.]+)/i);
        if (match) {
            return `v${match[1]}`;
        }
        if (raw.length <= 16) return raw;
        return raw.split(',')[0].replace(/Docker version\s*/i, 'v').trim();
    }

    // --- Live Swarm Status Processor ---
    function processSwarmStatus(serverName, data, isSilent = false) {
        if (!data.active) {
            swarmStateBadge.className = 'swarm-status-tag';
            swarmStateBadge.style.color = '';
            swarmStateBadge.style.background = '';
            swarmStateBadge.textContent = 'Sin Framework';
            deskSwarm.textContent = 'Inactivo';
            if (data.dockerVersion) {
                deskDocker.textContent = formatDockerVersion(data.dockerVersion);
                deskDocker.title = data.dockerVersion;
            }
            btnBootstrapSwarm.style.display = 'inline-flex';
            btnBootstrapSwarm.disabled = false;
            btnBootstrapSwarm.classList.remove('swarm-configured');
            btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
            btnBootstrapSwarm.title = `Instalar Framework de orquestación (Docker Swarm, Traefik, Portainer) en '${serverName}'`;
            if (tabServicesCount) tabServicesCount.textContent = '0';
            if (tabDatabasesCount) tabDatabasesCount.textContent = '0';
            if (tabTopologyCount) tabTopologyCount.textContent = '0';

            swarmServicesCache = [];
            swarmDatabasesCache = [];
            swarmNodesCache = [];

            swarmServicesCardsGrid.innerHTML = '';
            swarmServicesEmpty.style.display = 'flex';
            swarmDatabasesCardsGrid.innerHTML = '';
            swarmDatabasesEmpty.style.display = 'flex';
            if (swarmNodesTableBody) {
                swarmNodesTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">El Framework no está instalado o activo en este servidor. Haz clic en "Instalar Framework".</td></tr>`;
            }
            if (topologyServicesTableBody) {
                topologyServicesTableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty">El Framework no está instalado o activo en este servidor.</td></tr>`;
            }

            if (linkPortainer) linkPortainer.href = '#';
            if (linkDozzle) linkDozzle.href = '#';
            if (linkTraefik) linkTraefik.href = '#';
            return;
        }

        swarmStateBadge.className = 'swarm-status-tag';
        swarmStateBadge.style.color = 'var(--status-online)';
        swarmStateBadge.style.background = 'rgba(16, 185, 129, 0.12)';
        swarmStateBadge.textContent = '★ Clúster Operacional';
        deskSwarm.textContent = 'Operacional';
        if (data.dockerVersion) {
            deskDocker.textContent = formatDockerVersion(data.dockerVersion);
            deskDocker.title = data.dockerVersion;
        } else if (!deskDocker.textContent || deskDocker.textContent === '—') {
            deskDocker.textContent = 'Activo';
        }

        btnBootstrapSwarm.style.display = 'inline-flex';
        btnBootstrapSwarm.disabled = true;
        btnBootstrapSwarm.classList.add('swarm-configured');
        btnBootstrapSwarm.innerHTML = `✓ <span>Framework Instalado</span>`;
        btnBootstrapSwarm.title = `El Framework ya está instalado y activo en '${serverName}'`;

        if (linkPortainer) linkPortainer.href = (data.dashboards && data.dashboards.portainer) || '#';
        if (linkDozzle) linkDozzle.href = (data.dashboards && data.dashboards.dozzle) || '#';
        if (linkTraefik) linkTraefik.href = (data.dashboards && data.dashboards.traefik) || '#';

        const newServices = data.services || [];
        const newDbs = data.databases || [];
        const newNodes = data.nodes || [];

        // Comprobación de cambios para evitar destruir y recrear el DOM cada 15 segundos
        const servicesSig = JSON.stringify(newServices);
        const oldServicesSig = JSON.stringify(swarmServicesCache);
        const dbsSig = JSON.stringify(newDbs);
        const oldDbsSig = JSON.stringify(swarmDatabasesCache);
        const nodesSig = JSON.stringify(newNodes);
        const oldNodesSig = JSON.stringify(swarmNodesCache);

        swarmServicesCache = newServices;
        swarmDatabasesCache = newDbs;
        swarmNodesCache = newNodes;

        if (tabServicesCount) tabServicesCount.textContent = swarmServicesCache.length;
        if (tabDatabasesCount) tabDatabasesCount.textContent = swarmDatabasesCache.length;
        if (tabTopologyCount) tabTopologyCount.textContent = (swarmNodesCache.length + swarmServicesCache.length + swarmDatabasesCache.length);

        if (!isSilent || servicesSig !== oldServicesSig) {
            renderAppCards(swarmServicesCache);
        }
        if (!isSilent || dbsSig !== oldDbsSig) {
            renderDatabaseCards(swarmDatabasesCache);
        }
        if (!isSilent || nodesSig !== oldNodesSig || servicesSig !== oldServicesSig || dbsSig !== oldDbsSig) {
            renderNodesTable(swarmNodesCache);
            renderTopologyServicesTable(swarmServicesCache, swarmDatabasesCache, currentServiceLinks);
        }
    }

    async function loadSwarmStatus(serverName) {
        return refreshServerTelemetry(serverName, false);
    }

    // --- Render Modern App Cards (Railway/Coolify Style) ---
    function renderAppCards(services) {
        swarmServicesCardsGrid.innerHTML = '';

        if (!services || services.length === 0) {
            swarmServicesEmpty.style.display = 'flex';
            return;
        }
        swarmServicesEmpty.style.display = 'none';

        services.forEach(svc => {
            const isPublic = svc.expose || (svc.domain && svc.domain !== '');
            const card = document.createElement('div');
            card.className = 'app-card';

            const domainHtml = svc.domain ? `
                <div class="app-card-url-box">
                    <a href="http://${escapeHtml(svc.domain)}" target="_blank" class="app-card-url-link">
                        🌐 https://${escapeHtml(svc.domain)} ↗
                    </a>
                </div>
            ` : `
                <div class="app-card-url-box" style="color:var(--text-dim); font-size:0.75rem;">
                    🔒 Red Interna Swarm
                </div>
            `;

            card.innerHTML = `
                <div>
                    <div class="app-card-header">
                        <div class="app-card-title-group">
                            <div class="app-card-icon">🚀</div>
                            <div style="min-width:0; flex:1; overflow:hidden;">
                                <h3 class="app-card-name" title="${escapeHtml(svc.name)}">${escapeHtml(svc.name)}</h3>
                                <div class="app-card-image" title="${escapeHtml(svc.image)}">${escapeHtml(svc.image)}</div>
                            </div>
                        </div>
                        <div class="app-card-badges">
                            ${isPublic && svc.domain ? '<span class="card-ssl-pill" style="background:rgba(16,185,129,0.12); color:#10b981; border:1px solid rgba(16,185,129,0.25);" title="Certificado SSL Activo vía Traefik">🔒 SSL</span>' : ''}
                            <span class="svc-pill svc-pill-active" style="flex-shrink:0;">
                                <span class="status-dot status-online" style="width:6px; height:6px;"></span>
                                ${escapeHtml(svc.replicas)} Réplicas
                            </span>
                        </div>
                    </div>
                    <div style="margin-top:14px;">
                        ${domainHtml}
                    </div>
                </div>

                <div class="app-card-actions">
                    <div class="app-card-actions-meta">
                        <span class="t-badge" style="font-size:0.72rem;">${isPublic ? '🌐 Público SSL' : '🔒 Privado'}</span>
                    </div>
                    <div class="app-card-actions-buttons">
                        <button type="button" class="mini-btn btn-logs-svc" data-name="${escapeHtml(svc.name)}" title="Ver logs en tiempo real">
                            📜 Logs
                        </button>
                        <button type="button" class="mini-btn btn-restart-svc" data-name="${escapeHtml(svc.name)}" title="Reiniciar servicio en Docker">
                            🔄 Reiniciar
                        </button>
                        <button type="button" class="mini-btn btn-env-svc" data-name="${escapeHtml(svc.name)}" title="Gestionar variables de entorno .env">
                            🔑 Env
                        </button>
                        <button type="button" class="mini-btn btn-vol-svc" data-name="${escapeHtml(svc.name)}" title="Explorar archivos del volumen de almacenamiento">
                            📁 Archivos
                        </button>
                        ${isPublic ? `
                        <button type="button" class="mini-btn btn-maint-svc ${activeMaintenanceServices.has(svc.name) ? 'maint-active' : ''}" data-name="${escapeHtml(svc.name)}" title="Alternar modo mantenimiento (503 HTTP Drain)">
                            ${activeMaintenanceServices.has(svc.name) ? '🚧 Mantenimiento ON' : '🚧 Mantenimiento'}
                        </button>
                        ` : ''}
                        <button type="button" class="mini-btn btn-edit-svc" data-name="${escapeHtml(svc.name)}" data-expose="${isPublic}" data-domain="${escapeHtml(svc.domain || '')}">
                            ⚙️ Configurar
                        </button>
                        <button type="button" class="mini-btn btn-del-svc" data-name="${escapeHtml(svc.name)}" style="color:var(--status-offline);" title="Eliminar servicio">
                            ✕
                        </button>
                    </div>
                </div>
            `;

            swarmServicesCardsGrid.appendChild(card);
        });

        // Eventos de logs de servicios
        document.querySelectorAll('.btn-logs-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openLogsModal(name);
            });
        });

        // Eventos de reinicio de servicios
        document.querySelectorAll('.btn-restart-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) restartServiceOrContainer(name, btn);
            });
        });

        // Eventos de variables de entorno de servicios
        document.querySelectorAll('.btn-env-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openEnvModal(name);
            });
        });

        // Eventos de explorador de archivos del servicio
        document.querySelectorAll('.btn-vol-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openVolumeModal(`/opt/data/${name}`);
            });
        });

        // Eventos de modo mantenimiento (503)
        document.querySelectorAll('.btn-maint-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) {
                    const currentlyActive = activeMaintenanceServices.has(name);
                    toggleMaintenanceMode(name, !currentlyActive, btn);
                }
            });
        });

        // Eventos de edición de servicios
        document.querySelectorAll('.btn-edit-svc').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                const expose = btn.getAttribute('data-expose') === 'true';
                const domain = btn.getAttribute('data-domain');
                openEditServiceModal(name, expose, domain);
            });
        });

        // Eventos de eliminación de servicios
        document.querySelectorAll('.btn-del-svc').forEach(btn => {
            btn.addEventListener('click', async () => {
                const name = btn.getAttribute('data-name');
                if (!confirm(`¿Estás seguro de eliminar el servicio '${name}' de Docker Swarm?`)) return;
                try {
                    const res = await fetch(`/api/services/${encodeURIComponent(name)}?server=${encodeURIComponent(selectedServerName || '')}`, { method: 'DELETE' });
                    if (res.ok) {
                        showToast(`Servicio '${name}' eliminado.`, 'info');
                        if (selectedServerName) {
                            await loadSwarmStatus(selectedServerName);
                            await loadServiceLinks();
                        }
                        return;
                    }
                    const errText = await res.text();
                    showToast(`Error al eliminar: ${errText}`, 'error');
                } catch (err) {
                    showToast(`Fallo: ${err.message}`, 'error');
                }
            });
        });
    }

    // --- Render Modern Database Cards ---
    function renderDatabaseCards(databases) {
        swarmDatabasesCardsGrid.innerHTML = '';

        if (!databases || databases.length === 0) {
            swarmDatabasesEmpty.style.display = 'flex';
            return;
        }
        swarmDatabasesEmpty.style.display = 'none';

        databases.forEach(db => {
            const card = document.createElement('div');
            const engineClass = `db-card-engine-${(db.engine || 'postgres').toLowerCase()}`;
            card.className = `db-card ${engineClass}`;

            const isOnline = db.status === 'running';
            const port = db.internalPort || getDefaultPort(db.engine);
            const connStr = db.internalDns ? `${db.engine}://${db.name}:${port}/${db.name}` : (db.externalUrl || 'dns-interno');

            card.innerHTML = `
                <div>
                    <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:12px; gap:8px;">
                        <div style="display:flex; align-items:center; gap:10px; min-width:0; flex:1; overflow:hidden;">
                            <div style="font-size:1.6rem; line-height:1; flex-shrink:0;">🗄️</div>
                            <div style="min-width:0; flex:1; overflow:hidden;">
                                <h3 style="font-size:1.02rem; font-weight:700; color:#fff; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;" title="${escapeHtml(db.name)}">${escapeHtml(db.name)}</h3>
                                <div style="font-size:0.72rem; color:var(--text-muted); text-transform:uppercase; font-weight:700;">
                                    ${escapeHtml(db.engine)} · ${escapeHtml(db.deployType)}
                                </div>
                            </div>
                        </div>
                        <span class="svc-pill ${isOnline ? 'svc-pill-active' : ''}" style="flex-shrink:0;">
                            ${escapeHtml(db.status ? db.status.toUpperCase() : 'ONLINE')}
                        </span>
                    </div>

                    <div class="db-conn-box">
                        <span style="overflow:hidden; text-overflow:ellipsis; white-space:nowrap; min-width:0; flex:1;" title="${escapeHtml(connStr)}">${escapeHtml(connStr)}</span>
                        <button type="button" class="mini-btn btn-copy-conn" data-conn="${escapeHtml(connStr)}" title="Copiar cadena de conexión">
                            Copiar
                        </button>
                    </div>
                </div>

                <div class="db-card-actions">
                    <div class="db-card-actions-meta">
                        <span>Puerto: <code style="font-family:var(--font-mono);">${escapeHtml(port)}</code></span>
                    </div>
                    <div class="db-card-actions-buttons">
                        <button type="button" class="mini-btn btn-logs-db" data-name="${escapeHtml(db.name)}" title="Ver logs en tiempo real">
                            📜 Logs
                        </button>
                        <button type="button" class="mini-btn btn-restart-db" data-name="${escapeHtml(db.name)}" title="Reiniciar contenedor de base de datos">
                            🔄 Reiniciar
                        </button>
                        <button type="button" class="mini-btn btn-backup-db" data-name="${escapeHtml(db.name)}" data-engine="${escapeHtml(db.engine || 'postgres')}" title="Generar snapshot y descargar backup">
                            💾 Backup
                        </button>
                        <button type="button" class="mini-btn btn-vol-db" data-name="${escapeHtml(db.name)}" title="Explorar archivos del volumen persistente">
                            📁 Archivos
                        </button>
                        <button type="button" class="mini-btn btn-delete-db" data-name="${escapeHtml(db.name)}" style="color:var(--status-offline);" title="Eliminar base de datos">
                            ✕ Eliminar
                        </button>
                    </div>
                </div>
            `;

            swarmDatabasesCardsGrid.appendChild(card);
        });

        // Eventos de logs de bases de datos
        document.querySelectorAll('.btn-logs-db').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openLogsModal(name);
            });
        });

        // Eventos de reinicio de bases de datos
        document.querySelectorAll('.btn-restart-db').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) restartServiceOrContainer(name, btn);
            });
        });

        // Eventos de respaldo de bases de datos
        document.querySelectorAll('.btn-backup-db').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                const engine = btn.getAttribute('data-engine');
                if (name) triggerDatabaseBackup(name, engine, btn);
            });
        });

        // Eventos de explorador de archivos de la base de datos
        document.querySelectorAll('.btn-vol-db').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openVolumeModal('/opt/data/db-storage');
            });
        });

        // Eventos de copia y borrado
        document.querySelectorAll('.btn-copy-conn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const conn = btn.getAttribute('data-conn');
                const ok = await copyToClipboard(conn);
                if (ok) {
                    showToast('Cadena de conexión copiada al portapapeles', 'success');
                } else {
                    prompt('Cadena de conexión:', conn);
                }
            });
        });

        document.querySelectorAll('.btn-delete-db').forEach(btn => {
            btn.addEventListener('click', async () => {
                const name = btn.getAttribute('data-name');
                if (!confirm(`¿Estás seguro de eliminar la base de datos '${name}'?`)) return;
                try {
                    const delRes = await fetch(`/api/databases?name=${encodeURIComponent(name)}&server=${encodeURIComponent(selectedServerName || '')}`, { method: 'DELETE' });
                    if (delRes.ok) {
                        showToast(`Base de datos '${name}' eliminada`, 'info');
                        if (selectedServerName) {
                            await loadSwarmStatus(selectedServerName);
                            await loadServiceLinks();
                        }
                        return;
                    }
                    const errText = await delRes.text();
                    showToast(`Error al eliminar la base de datos: ${errText}`, 'error');
                } catch (e) {
                    showToast(`Error: ${e.message}`, 'error');
                }
            });
        });
    }

    // --- Render Nodes Table ---
    function renderNodesTable(nodes) {
        if (!swarmNodesTableBody) return;
        if (!nodes || nodes.length === 0) {
            swarmNodesTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">No se detectaron nodos adicionales.</td></tr>`;
            return;
        }

        swarmNodesTableBody.innerHTML = '';
        nodes.forEach(node => {
            const tr = document.createElement('tr');
            const isLeader = node.managerStatus && node.managerStatus.toLowerCase().includes('leader');
            const isManager = isLeader || (node.managerStatus && node.managerStatus.toLowerCase().includes('reachable'));
            const avail = (node.availability || 'active').toLowerCase();
            const isDrain = avail === 'drain';
            const nodeHost = (node.hostname || '').toLowerCase();

            // Buscar qué servicios y BDs están alojados en este nodo
            const assignedList = [];
            (swarmServicesCache || []).forEach(s => {
                const tgt = (s.targetNode || '').toLowerCase();
                if (!tgt || tgt === 'all' || tgt === nodeHost || (isManager && (tgt === 'manager' || tgt === 'lider' || tgt === 'leader'))) {
                    assignedList.push({ name: s.name, type: 'app' });
                }
            });
            (swarmDatabasesCache || []).forEach(db => {
                const tgt = (db.targetNode || '').toLowerCase();
                if (!tgt || tgt === 'all' || tgt === nodeHost || (isManager && (tgt === 'manager' || tgt === 'lider' || tgt === 'leader'))) {
                    assignedList.push({ name: db.name, type: 'db' });
                }
            });

            let assignedHtml = '<span style="color:var(--text-muted); font-size:0.75rem;">Sin cargas asignadas</span>';
            if (assignedList.length > 0) {
                assignedHtml = `<div style="display:flex; gap:4px; flex-wrap:wrap;">` +
                    assignedList.map(item => {
                        const icon = item.type === 'db' ? '🗄️' : '🚀';
                        return `<span class="t-badge t-badge-active" style="font-size:0.72rem; padding:1px 6px;">${icon} ${escapeHtml(item.name)}</span>`;
                    }).join('') +
                    `</div>`;
            }

            let actionsHtml = '';
            if (isLeader) {
                actionsHtml = `<span style="color:var(--text-muted); font-size:0.75rem;">Líder de Clúster</span>`;
            } else {
                const availBtnHtml = isDrain
                    ? `<button type="button" class="mini-btn btn-node-avail" data-id="${escapeHtml(node.id)}" data-avail="active" title="Activar nodo para recibir contenedores" style="color:var(--status-online);">▶️ Activar</button>`
                    : `<button type="button" class="mini-btn btn-node-avail" data-id="${escapeHtml(node.id)}" data-avail="drain" title="Drenar nodo (desalojar tareas)" style="color:var(--accent-warning, #f59e0b);">⏸️ Drenar</button>`;

                const deleteBtnHtml = `<button type="button" class="mini-btn btn-node-delete" data-id="${escapeHtml(node.id)}" data-hostname="${escapeHtml(node.hostname)}" title="Expulsar nodo del clúster" style="color:var(--status-offline);">🗑️ Expulsar</button>`;

                actionsHtml = `<div style="display:flex; gap:6px; align-items:center;">${availBtnHtml}${deleteBtnHtml}</div>`;
            }

            tr.innerHTML = `
                <td style="font-weight:700; color:#fff;">${escapeHtml(node.hostname)}</td>
                <td><span class="svc-pill ${node.status && node.status.toLowerCase() === 'ready' ? 'svc-pill-active' : ''}">${escapeHtml(node.status)}</span></td>
                <td style="color:${isDrain ? 'var(--accent-warning, #f59e0b)' : 'var(--text-muted)'};">${escapeHtml(node.availability)}</td>
                <td><span class="t-badge ${isLeader ? 't-badge-active' : ''}">${escapeHtml(node.managerStatus || 'Worker')}</span></td>
                <td>${assignedHtml}</td>
                <td style="color:var(--text-muted); font-family:var(--font-mono); font-size:0.75rem;">${escapeHtml(node.engineVersion || '—')}</td>
                <td>${actionsHtml}</td>
            `;
            swarmNodesTableBody.appendChild(tr);
        });

        // Eventos para Drenar / Activar Nodos
        document.querySelectorAll('.btn-node-avail').forEach(btn => {
            btn.addEventListener('click', async () => {
                const id = btn.getAttribute('data-id');
                const targetAvail = btn.getAttribute('data-avail');
                if (!id || !targetAvail) return;
                btn.disabled = true;
                try {
                    const res = await fetch(`/api/nodes/update?server=${encodeURIComponent(selectedServerName || '')}`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ id, availability: targetAvail })
                    });
                    if (res.ok) {
                        showToast(`Disponibilidad de nodo actualizada a '${targetAvail}'`, 'success');
                        if (selectedServerName) await loadSwarmStatus(selectedServerName);
                    } else {
                        const errText = await res.text();
                        showToast(`Error actualizando nodo: ${errText}`, 'error');
                    }
                } catch (err) {
                    showToast(`Error de red: ${err.message}`, 'error');
                } finally {
                    btn.disabled = false;
                }
            });
        });

        // Eventos para Expulsar Nodos del Clúster
        document.querySelectorAll('.btn-node-delete').forEach(btn => {
            btn.addEventListener('click', async () => {
                const id = btn.getAttribute('data-id');
                const hostname = btn.getAttribute('data-hostname') || id;
                if (!id) return;
                if (!confirm(`¿Estás seguro de expulsar el nodo '${hostname}' del clúster Swarm? Sus contenedores y tareas serán reubicados o desalojados.`)) return;
                btn.disabled = true;
                try {
                    const res = await fetch(`/api/nodes?id=${encodeURIComponent(id)}&server=${encodeURIComponent(selectedServerName || '')}`, {
                        method: 'DELETE'
                    });
                    if (res.ok) {
                        showToast(`Nodo '${hostname}' expulsado exitosamente del clúster`, 'success');
                        if (selectedServerName) await loadSwarmStatus(selectedServerName);
                    } else {
                        const errText = await res.text();
                        showToast(`Error expulsando nodo: ${errText}`, 'error');
                    }
                } catch (err) {
                    showToast(`Error de red: ${err.message}`, 'error');
                } finally {
                    btn.disabled = false;
                }
            });
        });
    }

    // --- Render Host Services Table (Systemd) ---
    function renderServicesTable(services) {
        if (!servicesTableBody) return;
        if (tabHostCount) tabHostCount.textContent = (services || []).length;

        if (!services || services.length === 0) {
            servicesTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">No se detectaron unidades de servicio del host.</td></tr>`;
            servicesSummaryText.textContent = '0 unidades';
            return;
        }

        const query = (serviceSearchInput.value || '').toLowerCase().trim();
        const filtered = !query ? services : services.filter(s =>
            (s.name && s.name.toLowerCase().includes(query)) ||
            (s.description && s.description.toLowerCase().includes(query))
        );

        servicesSummaryText.textContent = `${filtered.length} de ${services.length} unidades`;

        if (filtered.length === 0) {
            servicesTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Ningún servicio coincide con '${escapeHtml(query)}'</td></tr>`;
            return;
        }

        servicesTableBody.innerHTML = '';
        filtered.forEach(s => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td style="font-weight:600; color:#fff;">${escapeHtml(s.name)}</td>
                <td><span class="svc-pill svc-pill-active">${escapeHtml(s.activeState || 'active')}</span></td>
                <td style="color:var(--text-muted); font-size:0.75rem;">${escapeHtml(s.subState || 'running')}</td>
                <td style="color:var(--text-secondary); font-size:0.8rem;">
                    <div style="display:flex; justify-content:space-between; align-items:center; gap:8px;">
                        <span>${escapeHtml(s.description || '—')}</span>
                        <div style="display:flex; gap:6px;">
                            <button type="button" class="mini-btn btn-logs-host" data-name="${escapeHtml(s.name)}" title="Ver logs en tiempo real">
                                📜 Logs
                            </button>
                            <button type="button" class="mini-btn btn-restart-host" data-name="${escapeHtml(s.name)}" title="Reiniciar servicio del host">
                                🔄 Reiniciar
                            </button>
                        </div>
                    </div>
                </td>
            `;
            servicesTableBody.appendChild(tr);
        });

        // Eventos de logs de servicios del host
        document.querySelectorAll('.btn-logs-host').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) openLogsModal(name);
            });
        });

        // Eventos de reinicio de servicios del host
        document.querySelectorAll('.btn-restart-host').forEach(btn => {
            btn.addEventListener('click', () => {
                const name = btn.getAttribute('data-name');
                if (name) restartServiceOrContainer(name, btn);
            });
        });
    }

    serviceSearchInput.addEventListener('input', debounce(() => {
        renderServicesTable(currentHostServices);
    }, 150));

    // --- Hardware & Devices (Storage, GPU, USB, Displays, PCI) ---
    let currentHostDevices = null;
    let isLoadingDevices = false;

    async function loadHostDevices(forceFresh = false) {
        if (!selectedServerName || isLoadingDevices) return;
        isLoadingDevices = true;
        if (btnRefreshDevices) btnRefreshDevices.disabled = true;

        if (hwStorageTableBody) hwStorageTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">Consultando unidades de disco...</td></tr>`;
        if (hwGpuCardsContainer) hwGpuCardsContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">Consultando tarjetas gráficas...</div>`;
        if (hwUsbTableBody) hwUsbTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Consultando periféricos USB...</td></tr>`;
        if (hwDisplaysContainer) hwDisplaysContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">Consultando salidas de video...</div>`;
        if (hwPciTableBody) hwPciTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Consultando controladores PCI...</td></tr>`;

        try {
            const freshParam = forceFresh ? '&fresh=true' : '';
            const res = await fetch(`/api/host/devices?server=${encodeURIComponent(selectedServerName)}${freshParam}`);
            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error al obtener dispositivos: ${errText}`, 'error');
                return;
            }
            const data = await res.json();
            currentHostDevices = data;
            renderHostDevices(data);
        } catch (err) {
            showToast(`Fallo de conexión al inspeccionar dispositivos: ${err.message}`, 'error');
        } finally {
            isLoadingDevices = false;
            if (btnRefreshDevices) btnRefreshDevices.disabled = false;
        }
    }

    function renderHostDevices(data) {
        if (!data) return;

        const storage = data.storage || [];
        const gpus = data.gpus || [];
        const usb = data.usb || [];
        const displays = data.displays || [];
        const pci = data.pci || [];

        const totalDevices = storage.length + gpus.length + usb.length + displays.length + pci.length;

        if (tabDevicesCount) tabDevicesCount.textContent = totalDevices;
        if (devicesSummaryText) devicesSummaryText.textContent = `${totalDevices} componentes detectados`;

        if (hwStorageCount) hwStorageCount.textContent = `${storage.length} unidades`;
        if (hwGpuCount) hwGpuCount.textContent = `${gpus.length} detectadas`;
        if (hwUsbCount) hwUsbCount.textContent = `${usb.length} periféricos`;
        if (hwDisplaysCount) hwDisplaysCount.textContent = `${displays.length} salidas`;

        if (hwStorageBadge) hwStorageBadge.textContent = `${storage.length} unidades`;
        if (hwGpuBadge) hwGpuBadge.textContent = `${gpus.length} GPUs`;
        if (hwUsbBadge) hwUsbBadge.textContent = `${usb.length} dispositivos`;
        if (hwDisplaysBadge) hwDisplaysBadge.textContent = `${displays.length} salidas`;
        if (hwPciBadge) hwPciBadge.textContent = `${pci.length} adaptadores`;

        // 1. Almacenamiento
        if (hwStorageTableBody) {
            if (storage.length === 0) {
                hwStorageTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">No se detectaron unidades de almacenamiento accesibles.</td></tr>`;
            } else {
                hwStorageTableBody.innerHTML = storage.map(s => {
                    const techTag = s.rotational
                        ? `<span class="hw-tag-hdd">HDD</span>`
                        : `<span class="hw-tag-ssd">SSD / NVMe</span>`;
                    return `
                        <tr>
                            <td><strong style="color:var(--text-pure);">${escapeHtml(s.name || '—')}</strong></td>
                            <td>${escapeHtml(s.size || '—')}</td>
                            <td><span class="t-badge">${escapeHtml(s.type || 'disk')}</span></td>
                            <td>${techTag}</td>
                            <td><code>${escapeHtml(s.fsType || '—')}</code></td>
                            <td><strong style="color:var(--brand-primary);">${escapeHtml(s.mountPoint || '—')}</strong></td>
                            <td style="color:var(--text-muted);">${escapeHtml(s.model || '—')}</td>
                        </tr>
                    `;
                }).join('');
            }
        }

        // 2. Unidades GPU
        if (hwGpuCardsContainer) {
            if (gpus.length === 0) {
                hwGpuCardsContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">No se detectaron GPUs dedicadas o aceleradores gráficos.</div>`;
            } else {
                hwGpuCardsContainer.innerHTML = gpus.map(g => `
                    <div class="hw-device-card">
                        <div class="hw-card-top">
                            <span class="hw-device-name">🎮 ${escapeHtml(g.model || 'GPU Acelerador')}</span>
                            <span class="t-badge">${escapeHtml(g.vendor || 'Hardware')}</span>
                        </div>
                        <div class="hw-device-detail">
                            <span>Memoria VRAM:</span>
                            <strong style="color:var(--text-pure);">${escapeHtml(g.memoryTotal || 'Compartida')}</strong>
                        </div>
                        <div class="hw-device-detail">
                            <span>Bus PCI:</span>
                            <code>${escapeHtml(g.pciAddress || 'Interno')}</code>
                        </div>
                        ${g.driver ? `
                        <div class="hw-device-detail">
                            <span>Driver:</span>
                            <span class="t-badge">${escapeHtml(g.driver)}</span>
                        </div>` : ''}
                    </div>
                `).join('');
            }
        }

        // 3. Dispositivos USB
        if (hwUsbTableBody) {
            if (usb.length === 0) {
                hwUsbTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">No se detectaron periféricos conectados a puertos USB.</td></tr>`;
            } else {
                hwUsbTableBody.innerHTML = usb.map(u => `
                    <tr>
                        <td><code>Bus ${escapeHtml(u.bus || '001')}</code></td>
                        <td><code>Dev ${escapeHtml(u.device || '001')}</code></td>
                        <td><strong style="color:var(--brand-primary);">${escapeHtml(u.id || '—')}</strong></td>
                        <td>${escapeHtml(u.description || 'Dispositivo USB Genérico')}</td>
                    </tr>
                `).join('');
            }
        }

        // 4. Salidas de Pantalla / Video
        if (hwDisplaysContainer) {
            if (displays.length === 0) {
                hwDisplaysContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">No se detectaron salidas de video o pantallas conectadas.</div>`;
            } else {
                hwDisplaysContainer.innerHTML = displays.map(d => {
                    const isConn = d.status === 'connected';
                    const tag = isConn
                        ? `<span class="hw-tag-connected">● Conectado</span>`
                        : `<span class="hw-tag-disconnected">○ Desconectado</span>`;
                    return `
                        <div class="hw-device-card">
                            <div class="hw-card-top">
                                <span class="hw-device-name">🖥️ ${escapeHtml(d.connector || 'Monitor')}</span>
                                ${tag}
                            </div>
                            <div class="hw-device-detail">
                                <span>Resolución:</span>
                                <strong style="color:var(--text-pure);">${escapeHtml(d.resolution || 'No activa / Standby')}</strong>
                            </div>
                        </div>
                    `;
                }).join('');
            }
        }

        // 5. Buses PCI
        if (hwPciTableBody) {
            if (pci.length === 0) {
                hwPciTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">No se detectaron controladores PCI o la utilidad lspci no está instalada.</td></tr>`;
            } else {
                hwPciTableBody.innerHTML = pci.map(p => `
                    <tr>
                        <td><code>${escapeHtml(p.address || '—')}</code></td>
                        <td><span class="t-badge">${escapeHtml(p.class || 'Dispositivo PCI')}</span></td>
                        <td>${escapeHtml(p.vendor || '—')}</td>
                        <td><strong style="color:var(--text-pure);">${escapeHtml(p.device || '—')}</strong></td>
                    </tr>
                `).join('');
            }
        }
    }

    // --- Render Topology Services Table (Services & Databases network mapping) ---
    function renderTopologyServicesTable(services, databases, links) {
        if (!topologyServicesTableBody) return;
        const svcs = services || [];
        const dbs = databases || [];
        const lnks = links || [];
        const total = svcs.length + dbs.length;

        if (topologyServicesCountBadge) {
            topologyServicesCountBadge.textContent = `${total} ${total === 1 ? 'servicio' : 'servicios'}`;
        }

        if (total === 0) {
            topologyServicesTableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty">Sin servicios desplegados en la topología de red. Despliega una app o base de datos.</td></tr>`;
            return;
        }

        let rowsHtml = '';

        // 1. Apps Docker
        svcs.forEach(s => {
            const isPublic = s.expose || (s.domain && s.domain !== '');
            let port = '80';
            if (s.port) {
                port = s.port;
            } else if (s.ports) {
                const match = String(s.ports).match(/(\d+)/);
                if (match) port = match[1];
            }

            // Enlaces asociados a esta app
            const relatedLinks = lnks.filter(l => (l.source_svc || l.SourceSvc) === s.name);
            let linkBadgesHtml = '';
            if (relatedLinks.length > 0) {
                linkBadgesHtml = `<div style="display:flex; gap:4px; flex-wrap:wrap; margin-top:4px;">` +
                    relatedLinks.map(l => {
                        const target = l.target_svc || l.TargetSvc;
                        const vName = l.env_var_name || l.EnvVarName;
                        return `<span class="t-badge" style="font-size:0.7rem; border-color:rgba(99,102,241,0.3); background:rgba(99,102,241,0.1); color:#a5b4fc;" title="Variable inyectada: ${escapeHtml(vName)}">🔗 ${escapeHtml(target)}</span>`;
                    }).join('') +
                    `</div>`;
            }

            const publicRouteHtml = isPublic && s.domain
                ? `<a href="https://${escapeHtml(s.domain)}" target="_blank" style="color:var(--brand-primary); text-decoration:none; display:inline-flex; align-items:center; gap:4px; font-weight:600;">
                     🌐 https://${escapeHtml(s.domain)} ↗
                   </a>`
                : `<span style="color:var(--text-muted); font-size:0.75rem;">🔒 Red Interna Swarm</span>`;

            const targetNodeHtml = s.targetNode
                ? `<span class="t-badge" style="font-size:0.75rem;">${escapeHtml(s.targetNode)}</span>`
                : `<span style="color:var(--text-muted); font-size:0.75rem;">Cualquiera (Global/Replica)</span>`;

            rowsHtml += `
                <tr>
                    <td>
                        <div style="display:flex; align-items:center; gap:8px;">
                            <span style="font-size:1.1rem; line-height:1;">🚀</span>
                            <div style="min-width:0; overflow:hidden;">
                                <strong style="color:#fff; font-size:0.88rem; display:block; text-overflow:ellipsis; overflow:hidden; white-space:nowrap;">${escapeHtml(s.name)}</strong>
                                <span style="font-size:0.72rem; color:var(--text-muted); font-family:var(--font-mono); display:block; text-overflow:ellipsis; overflow:hidden; white-space:nowrap;">${escapeHtml(s.image || 'imagen docker')}</span>
                            </div>
                        </div>
                    </td>
                    <td><span class="t-badge t-badge-active" style="font-size:0.72rem;">App Docker</span></td>
                    <td>
                        <code style="font-family:var(--font-mono); font-size:0.78rem; color:var(--brand-primary);">${escapeHtml(s.name)}:${escapeHtml(port)}</code>
                        ${linkBadgesHtml}
                    </td>
                    <td>${publicRouteHtml}</td>
                    <td>${targetNodeHtml}</td>
                    <td><span class="svc-pill svc-pill-active" style="font-size:0.72rem;">${escapeHtml(s.replicas || '1/1')}</span></td>
                </tr>
            `;
        });

        // 2. Bases de Datos
        dbs.forEach(db => {
            const dbPort = db.port || getDefaultPort(db.engine);
            const targetNodeHtml = db.targetNode
                ? `<span class="t-badge" style="font-size:0.75rem;">${escapeHtml(db.targetNode)}</span>`
                : `<span style="color:var(--text-muted); font-size:0.75rem;">Manager / Primario</span>`;

            // Buscar si alguna app está consumiendo esta BD
            const consumerLinks = lnks.filter(l => (l.target_svc || l.TargetSvc) === db.name);
            let consumerBadgesHtml = '';
            if (consumerLinks.length > 0) {
                consumerBadgesHtml = `<div style="display:flex; gap:4px; flex-wrap:wrap; margin-top:4px;">` +
                    consumerLinks.map(l => {
                        const src = l.source_svc || l.SourceSvc;
                        return `<span class="t-badge" style="font-size:0.7rem; border-color:rgba(16,185,129,0.3); background:rgba(16,185,129,0.1); color:#34d399;" title="Consumido por ${escapeHtml(src)}">⬅️ ${escapeHtml(src)}</span>`;
                    }).join('') +
                    `</div>`;
            }

            rowsHtml += `
                <tr>
                    <td>
                        <div style="display:flex; align-items:center; gap:8px;">
                            <span style="font-size:1.1rem; line-height:1;">🗄️</span>
                            <div style="min-width:0; overflow:hidden;">
                                <strong style="color:#fff; font-size:0.88rem; display:block; text-overflow:ellipsis; overflow:hidden; white-space:nowrap;">${escapeHtml(db.name)}</strong>
                                <span style="font-size:0.72rem; color:var(--text-muted); text-transform:uppercase; font-weight:600;">${escapeHtml(db.engine || 'BD')} · ${escapeHtml(db.deployType || 'single-node')}</span>
                            </div>
                        </div>
                    </td>
                    <td><span class="t-badge" style="font-size:0.72rem; background:rgba(16,185,129,0.12); color:#10b981; border:1px solid rgba(16,185,129,0.25);">Base de Datos</span></td>
                    <td>
                        <code style="font-family:var(--font-mono); font-size:0.78rem; color:#34d399;">tarhiata-db-${escapeHtml(db.name)}:${escapeHtml(dbPort)}</code>
                        ${consumerBadgesHtml}
                    </td>
                    <td><span style="color:var(--text-muted); font-size:0.75rem;">🔒 Aislada (Red Swarm)</span></td>
                    <td>${targetNodeHtml}</td>
                    <td><span class="svc-pill ${db.status === 'online' ? 'svc-pill-active' : ''}" style="font-size:0.72rem;">${escapeHtml(db.status ? db.status.toUpperCase() : '1/1')}</span></td>
                </tr>
            `;
        });

        topologyServicesTableBody.innerHTML = rowsHtml;
    }

    // --- Populate Link Modal Dropdowns ---
    function populateLinkModalDropdowns() {
        if (serviceListFrom) {
            serviceListFrom.innerHTML = '';
            (swarmServicesCache || []).forEach(svc => {
                const opt = document.createElement('option');
                opt.value = svc.name;
                opt.label = `${svc.name} (App)`;
                serviceListFrom.appendChild(opt);
            });
        }
        if (serviceListTo) {
            serviceListTo.innerHTML = '';
            (swarmDatabasesCache || []).forEach(db => {
                const opt = document.createElement('option');
                opt.value = db.name;
                opt.label = `${db.name} (${db.engine || 'BD'})`;
                serviceListTo.appendChild(opt);
            });
            (swarmServicesCache || []).forEach(svc => {
                const opt = document.createElement('option');
                opt.value = svc.name;
                opt.label = `${svc.name} (App)`;
                serviceListTo.appendChild(opt);
            });
        }
    }

    // --- Load Service Links ---
    async function loadServiceLinks() {
        if (!linksTableBody) return;
        try {
            const res = await fetch('/api/links');
            if (!res.ok) return;

            const links = await res.json();
            currentServiceLinks = Array.isArray(links) ? links : [];
            renderTopologyServicesTable(swarmServicesCache, swarmDatabasesCache, currentServiceLinks);

            if (!currentServiceLinks || currentServiceLinks.length === 0) {
                linksTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Sin enlaces activos. Haz clic en '+ Enlazar Servicios'.</td></tr>`;
                return;
            }

            linksTableBody.innerHTML = '';
            currentServiceLinks.forEach(l => {
                const tr = document.createElement('tr');
                const src = l.source_svc || l.SourceSvc;
                const tgt = l.target_svc || l.TargetSvc;
                const v = l.env_var_name || l.EnvVarName;
                const u = l.target_url || l.TargetURL || 'interno';

                tr.innerHTML = `
                    <td style="font-weight:700; color:#fff;">${escapeHtml(src)}</td>
                    <td><span class="svc-pill svc-pill-active">${escapeHtml(v)}</span></td>
                    <td style="color:var(--text-secondary); font-size:0.8rem;">${escapeHtml(tgt)} (${escapeHtml(u)})</td>
                    <td>
                        <button type="button" class="mini-btn btn-unlink" data-from="${escapeHtml(src)}" data-to="${escapeHtml(tgt)}" style="color:var(--status-offline);">
                            Desconectar
                        </button>
                    </td>
                `;
                linksTableBody.appendChild(tr);
            });

            document.querySelectorAll('.btn-unlink').forEach(btn => {
                btn.addEventListener('click', async () => {
                    const fromSvc = btn.getAttribute('data-from');
                    const toSvc = btn.getAttribute('data-to');
                    if (!confirm(`¿Desenlazar '${fromSvc}' de '${toSvc}'?`)) return;

                    try {
                        const delRes = await fetch(`/api/links?source_svc=${encodeURIComponent(fromSvc)}&target_svc=${encodeURIComponent(toSvc)}`, { method: 'DELETE' });
                        if (delRes.ok) {
                            showToast(`Enlace eliminado: ${fromSvc} ⤬ ${toSvc}`, 'info');
                            loadServiceLinks();
                            return;
                        }
                        showToast('Error al desenlazar', 'error');
                    } catch (err) {
                        showToast(`Error: ${err.message}`, 'error');
                    }
                });
            });
        } catch (err) {
            console.error('Error cargando enlaces:', err);
        }
    }

    // --- Helper: Consumo y decodificación de Streams NDJSON con reporte de progreso ---
    async function consumeNDJSONStream(res, onStep, onError) {
        if (!res || !res.body) {
            return { hasError: false, lastError: '' };
        }
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let hasError = false;
        let lastError = '';

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop();

            for (const rawLine of lines) {
                const trimmed = rawLine.trim();
                if (!trimmed) continue;
                try {
                    const event = JSON.parse(trimmed.startsWith('data: ') ? trimmed.substring(6) : trimmed);
                    const eventType = event.t || event.type;
                    const msg = event.m || event.message || (event.d && event.d.message) || '';
                    if (eventType === 'step' && typeof onStep === 'function') {
                        onStep(msg);
                    } else if (eventType === 'error') {
                        hasError = true;
                        lastError = msg;
                        if (typeof onError === 'function') {
                            onError(msg);
                        }
                    }
                } catch (parseErr) {
                    console.debug('Error parseando chunk NDJSON:', parseErr);
                }
            }
        }
        return { hasError, lastError };
    }

    // --- Bootstrap Swarm / Install Framework Action ---
    btnBootstrapSwarm.addEventListener('click', async () => {
        if (!selectedServerName) {
            showToast('Primero añade o conecta un servidor VPS para instalar el Framework.', 'info');
            openAddModal();
            return;
        }
        if (!confirm(`¿Instalar el Framework de orquestación (Docker Swarm, Traefik, Portainer y Dozzle) en '${selectedServerName}'?`)) return;

        btnBootstrapSwarm.disabled = true;
        btnBootstrapSwarm.classList.remove('swarm-configured');
        btnBootstrapSwarm.innerHTML = '⏳ <span>Instalando Framework...</span>';
        showToast(`Iniciando instalación del Framework en '${selectedServerName}'...`, 'info');

        const s = servers.find(x => x.name === selectedServerName);
        const payload = {
            acmeEmail: 'admin@tarhiata.local',
            installObservability: true
        };
        if (s && s.host && s.host !== 'localhost' && s.host !== '127.0.0.1') {
            payload.host = s.host;
            payload.port = s.port > 0 ? s.port : 22;
            payload.user = s.user || 'root';
            payload.keyPath = s.privateKey || '';
        }

        try {
            const res = await fetch(`/api/bootstrap?server=${encodeURIComponent(selectedServerName || '')}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error al instalar Framework: ${errText}`, 'error');
                btnBootstrapSwarm.disabled = false;
                btnBootstrapSwarm.classList.remove('swarm-configured');
                btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
                return;
            }

            const streamResult = await consumeNDJSONStream(
                res,
                (msg) => showToast(msg, 'info'),
                (msg) => showToast(`Error: ${msg}`, 'error')
            );

            if (!streamResult.hasError) {
                showToast(`¡Framework instalado con éxito en '${selectedServerName}'!`, 'success');
            }

            await loadSwarmStatus(selectedServerName);
            await refreshServerTelemetry(selectedServerName);
        } catch (err) {
            showToast(`Fallo: ${err.message}`, 'error');
            btnBootstrapSwarm.disabled = false;
            btnBootstrapSwarm.classList.remove('swarm-configured');
            btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
            await loadSwarmStatus(selectedServerName);
        }
    });

    // --- Native Terminal Direct Launcher ---
    async function launchNativeTerminal(serverName, buttonEl) {
        const btn = buttonEl || btnDeskTerminal;
        const origContent = btn.innerHTML;
        btn.disabled = true;
        btn.innerHTML = '<span>...</span>';

        try {
            const res = await fetch('/api/servers/open-terminal', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: serverName })
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error abriendo terminal de ${serverName}: ${errText}`, 'error');
                return;
            }

            const data = await res.json();
            if (data.launched) {
                showToast(`Terminal abierta en tu sistema (${serverName})`, 'success');
                return;
            }
            if (data.command) {
                const ok = await copyToClipboard(data.command);
                if (ok) {
                    showToast(`Comando copiado al portapapeles: ${data.command}`, 'info');
                } else {
                    prompt('Copia el comando de conexión SSH:', data.command);
                }
            }
        } catch (err) {
            showToast(`Fallo: ${err.message}`, 'error');
        } finally {
            btn.disabled = false;
            btn.innerHTML = origContent;
        }
    }

    btnDeskTerminal.addEventListener('click', () => {
        if (selectedServerName) openTerminalModal(selectedServerName);
    });

    const btnDeskNativeTerminal = document.getElementById('btnDeskNativeTerminal');
    if (btnDeskNativeTerminal) {
        btnDeskNativeTerminal.addEventListener('click', () => {
            if (selectedServerName) launchNativeTerminal(selectedServerName, btnDeskNativeTerminal);
        });
    }

    btnTopActiveTerminal.addEventListener('click', () => {
        const targetServer = (activeServer && activeServer.name) ? activeServer.name : selectedServerName;
        if (!targetServer) {
            showToast('No hay ningún servidor configurado.', 'error');
            return;
        }
        openTerminalModal(targetServer);
    });

    btnDeskRefresh.addEventListener('click', () => {
        if (selectedServerName) refreshServerTelemetry(selectedServerName);
    });

    btnDeskActivate.addEventListener('click', () => {
        if (selectedServerName) switchActiveServer(selectedServerName);
    });

    // --- Switch Active Server Context ---
    async function switchActiveServer(name) {
        showToast(`Cambiando a servidor activo: '${name}'...`, 'info');
        try {
            const res = await fetch('/api/servers/active', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });
            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error: ${errText}`, 'error');
                return;
            }
            showToast(`Servidor principal: '${name}'.`, 'success');
            selectedServerName = name;
            await loadHubState();
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        }
    }

    // --- Delete Server ---
    async function deleteServer(name) {
        if (!confirm(`¿Eliminar definitivamente el servidor '${name}'?`)) return;

        try {
            const res = await fetch(`/api/servers?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error: ${errText}`, 'error');
                return;
            }
            showToast(`Servidor '${name}' eliminado.`, 'success');
            if (selectedServerName === name) {
                selectedServerName = null;
            }
            await loadHubState();
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        }
    }

    // --- Test Connectivity ---
    async function testSingleServer(name) {
        const dotEl = document.getElementById(`fleet-dot-${name}`);
        const chipDot = document.getElementById(`chip-dot-${name}`);
        if (dotEl) dotEl.className = 'status-dot status-pending';
        if (chipDot) chipDot.className = 'status-dot status-pending';

        try {
            const s = servers.find(x => x.name === name);
            if (!s) return;

            const res = await fetch('/api/connect', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(s)
            });

            if (!res.ok) {
                if (dotEl) dotEl.className = 'status-dot status-offline';
                if (chipDot) chipDot.className = 'status-dot status-offline';
                return;
            }

            const diag = await res.json();
            if (!diag.connected) {
                if (dotEl) dotEl.className = 'status-dot status-offline';
                if (chipDot) chipDot.className = 'status-dot status-offline';
                return;
            }

            if (dotEl) dotEl.className = 'status-dot status-online';
            if (chipDot) chipDot.className = 'status-dot status-online';

            if (name === selectedServerName) {
                deskDocker.textContent = diag.dockerActive ? formatDockerVersion(diag.dockerVersion) : 'Inactivo';
                deskDocker.title = diag.dockerActive ? (diag.dockerVersion || '') : 'Inactivo';
                deskSwarm.textContent = diag.swarmActive ? 'Operacional' : 'Inactivo';
                tileLatencyDisplay.textContent = `Latencia: ${diag.latencyMs} ms`;
                topActiveDot.className = 'status-dot status-online';
                topActiveLatency.textContent = `${diag.latencyMs} ms`;
            }
        } catch (err) {
            if (dotEl) dotEl.className = 'status-dot status-offline';
            if (chipDot) chipDot.className = 'status-dot status-offline';
            if (name === selectedServerName) {
                topActiveDot.className = 'status-dot status-offline';
                topActiveLatency.textContent = 'OFFLINE';
            }
        }
    }

    async function testAllServers(isManual = false) {
        btnTestAll.disabled = true;
        btnTestAll.innerHTML = '<span>⟳ Sondeando...</span>';

        try {
            const res = await fetch('/api/connect/all', { method: 'POST' });
            if (res.ok) {
                const results = await res.json();
                results.forEach(diag => {
                    const dotEl = document.getElementById(`fleet-dot-${diag.name}`);
                    if (dotEl) {
                        dotEl.className = diag.connected ? 'status-dot status-online' : 'status-dot status-offline';
                    }
                    const chipDot = document.getElementById(`chip-dot-${diag.name}`);
                    if (chipDot) {
                        chipDot.className = diag.connected ? 'status-dot status-online' : 'status-dot status-offline';
                    }
                    if (diag.name === selectedServerName && diag.connected) {
                        deskDocker.textContent = diag.dockerActive ? formatDockerVersion(diag.dockerVersion) : 'Inactivo';
                        deskDocker.title = diag.dockerActive ? (diag.dockerVersion || '') : 'Inactivo';
                        deskSwarm.textContent = diag.swarmActive ? 'Operacional' : 'Inactivo';
                        tileLatencyDisplay.textContent = `Latencia: ${diag.latencyMs} ms`;
                        topActiveDot.className = 'status-dot status-online';
                        topActiveLatency.textContent = `${diag.latencyMs} ms`;
                    }
                });
            } else {
                await Promise.all(servers.map(s => testSingleServer(s.name)));
            }
        } catch (err) {
            console.debug('Fallo /api/connect/all, usando sondeo individual:', err);
            await Promise.all(servers.map(s => testSingleServer(s.name)));
        } finally {
            btnTestAll.disabled = false;
            btnTestAll.innerHTML = `
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6"></path><path d="M1 20v-6h6"></path><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
                <span>Sondear Red</span>
            `;
            if (isManual) {
                showToast('Sondeo de conectividad completado.', 'info');
            }
        }
    }

    btnTestAll.addEventListener('click', () => testAllServers(true));

    // --- Modal Controllers: Add Server ---
    function setModalMode(mode) {
        modalMode = mode;
        tabLocal.classList.toggle('active', mode === 'local');
        tabRemote.classList.toggle('active', mode === 'remote');
        tabCloud.classList.toggle('active', mode === 'cloud');
        modalTestResult.style.display = 'none';

        if (mode === 'local') {
            localFastNotice.style.display = 'flex';
            cloudFields.style.display = 'none';
            standardFields.style.display = 'grid';
            nameField.style.display = 'flex';
            hostField.style.display = 'none';
            userField.style.display = 'none';
            portField.style.display = 'none';
            keyField.style.display = 'none';

            cfgName.value = 'local';
            cfgHost.value = 'localhost';
            cfgUser.value = 'local';
            cfgPort.value = '0';
            btnModalTest.style.display = 'inline-flex';
            btnModalSaveText.textContent = 'Conectar Local';
            return;
        }
        if (mode === 'remote') {
            localFastNotice.style.display = 'none';
            cloudFields.style.display = 'none';
            standardFields.style.display = 'grid';
            nameField.style.display = 'flex';
            hostField.style.display = 'flex';
            userField.style.display = 'flex';
            portField.style.display = 'flex';
            keyField.style.display = 'flex';

            if (cfgName.value === 'local' || cfgName.value.startsWith('cloud-')) {
                cfgName.value = 'vps-servidor';
            }
            if (cfgHost.value === 'localhost' || cfgHost.value === '127.0.0.1') {
                cfgHost.value = '';
            }
            cfgUser.value = 'root';
            cfgPort.value = '22';
            cfgKey.value = '~/.ssh/id_rsa';
            btnModalTest.style.display = 'inline-flex';
            btnModalSaveText.textContent = 'Guardar VPS';
            return;
        }
        if (mode === 'cloud') {
            localFastNotice.style.display = 'none';
            cloudFields.style.display = 'grid';
            standardFields.style.display = 'grid';
            nameField.style.display = 'flex';
            hostField.style.display = 'none';
            userField.style.display = 'none';
            portField.style.display = 'none';
            keyField.style.display = 'none';

            if (!cfgName.value || cfgName.value === 'local') {
                cfgName.value = 'cloud-node-1';
            }
            btnModalTest.style.display = 'none';
            btnModalSaveText.textContent = 'Crear con OpenTofu';
            return;
        }
    }

    tabLocal.addEventListener('click', () => setModalMode('local'));
    tabRemote.addEventListener('click', () => setModalMode('remote'));
    tabCloud.addEventListener('click', () => setModalMode('cloud'));

    function openAddModal() {
        formServer.reset();
        setModalMode('remote');
        serverModal.style.display = 'flex';
        serverPopover.style.display = 'none';
    }

    btnOpenAddModal.addEventListener('click', openAddModal);
    const btnFleetAddServer = document.getElementById('btnFleetAddServer');
    if (btnFleetAddServer) {
        btnFleetAddServer.addEventListener('click', openAddModal);
    }
    function closeServerModal() {
        formServer.reset();
        modalTestResult.innerHTML = '';
        serverModal.style.display = 'none';
    }
    btnCloseModal.addEventListener('click', closeServerModal);
    if (btnCancelServer) btnCancelServer.addEventListener('click', closeServerModal);
    serverModal.addEventListener('click', (e) => {
        if (e.target === serverModal) closeServerModal();
    });

    formServer.addEventListener('submit', async (e) => {
        e.preventDefault();
        const isCloud = (modalMode === 'cloud');
        const isLoc = (modalMode === 'local');

        let payload = {
            name: cfgName.value.trim(),
            host: isLoc ? 'localhost' : cfgHost.value.trim(),
            port: isLoc ? 0 : parseInt(cfgPort.value || '22', 10),
            user: isLoc ? 'local' : cfgUser.value.trim(),
            privateKey: isLoc ? '' : cfgKey.value.trim(),
            cloudProvider: isLoc ? 'local' : (isCloud ? cfgProvider.value : 'custom'),
            isActive: cfgIsActive.checked
        };

        if (isCloud) {
            payload.provider = cfgProvider.value;
            payload.apiToken = cfgToken.value.trim();
            payload.region = cfgRegion.value.trim() || 'mex';
            payload.plan = cfgPlan.value.trim() || 'vc2-1c-1gb';
            payload.isActive = cfgIsActive.checked;
        }

        btnModalSave.disabled = true;
        btnModalSaveText.textContent = 'Guardando...';

        try {
            const endpoint = isCloud ? '/api/servers/provision' : '/api/servers';
            const res = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error: ${errText}`, 'error');
                modalTestResult.style.display = 'block';
                modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(errText)}</span>`;
                return;
            }

            showToast(`Servidor '${payload.name}' conectado con éxito!`, 'success');
            serverModal.style.display = 'none';
            formServer.reset();
            selectedServerName = payload.name;
            await loadHubState();
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        } finally {
            btnModalSave.disabled = false;
            btnModalSaveText.textContent = isCloud ? 'Crear con OpenTofu' : 'Guardar Servidor';
        }
    });

    btnModalTest.addEventListener('click', async () => {
        btnModalTest.disabled = true;
        btnModalTestText.textContent = 'Probando...';
        modalTestResult.style.display = 'block';
        modalTestResult.innerHTML = `<span style="color:var(--status-warning);">⏳ Verificando conexión...</span>`;

        const isLoc = (modalMode === 'local');
        const testPayload = {
            name: cfgName.value.trim() || 'test',
            host: isLoc ? 'localhost' : cfgHost.value.trim(),
            port: isLoc ? 0 : parseInt(cfgPort.value || '22', 10),
            user: isLoc ? 'local' : cfgUser.value.trim(),
            privateKey: isLoc ? '' : cfgKey.value.trim(),
            cloudProvider: isLoc ? 'local' : 'custom'
        };

        try {
            const res = await fetch('/api/connect', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(testPayload)
            });
            if (!res.ok) {
                const errText = await res.text();
                modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ Error (${res.status}): ${escapeHtml(errText)}</span>`;
                return;
            }
            const data = await res.json();

            if (!data.connected) {
                modalTestResult.innerHTML = `<div style="color:var(--status-offline); font-weight:700;">✕ Conexión fallida: ${escapeHtml(data.message)}</div>`;
                return;
            }

            modalTestResult.innerHTML = `<div style="color:var(--status-online); font-weight:700;">✓ Conectado exitosamente (${data.latencyMs} ms)</div>`;
        } catch (err) {
            modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ Error: ${escapeHtml(err.message)}</span>`;
        } finally {
            btnModalTest.disabled = false;
            btnModalTestText.textContent = 'Probar Conexión';
        }
    });

    // --- Verificador Reactivo de DNS en Tiempo Real ---
    const checkDomainDnsDebounced = debounce(async (domain, feedbackEl) => {
        if (!feedbackEl) return;
        const clean = (domain || '')
            .replace(/^https?:\/\//i, '')
            .split('/')[0]
            .split(':')[0]
            .trim()
            .toLowerCase();

        if (!clean || clean.length < 3 || !clean.includes('.')) {
            feedbackEl.style.display = 'none';
            feedbackEl.innerHTML = '';
            return;
        }

        feedbackEl.style.display = 'inline-flex';
        feedbackEl.className = 'dns-feedback-hint dns-checking';
        feedbackEl.innerHTML = '<span>⏳ Consultando resolución DNS...</span>';

        try {
            const res = await fetch(`/api/dns/check?domain=${encodeURIComponent(clean)}`);
            if (!res.ok) {
                feedbackEl.style.display = 'none';
                return;
            }
            const data = await res.json();
            const serverIp = data.server_ip || (activeServer ? activeServer.host : '');

            if (data.status === 'match') {
                feedbackEl.className = 'dns-feedback-hint dns-match';
                feedbackEl.innerHTML = `<span>🟢 ✓ Apunta a este servidor (${escapeHtml(serverIp || 'IP del VPS')})</span>`;
            } else if (data.status === 'mismatch') {
                const ips = (data.resolved_ips || []).join(', ') || 'otra IP';
                feedbackEl.className = 'dns-feedback-hint dns-mismatch';
                feedbackEl.innerHTML = `<span>🟡 ⚠️ Apunta a ${escapeHtml(ips)} (Tu VPS: ${escapeHtml(serverIp || 'este host')})</span>`;
            } else {
                feedbackEl.className = 'dns-feedback-hint dns-not-found';
                feedbackEl.innerHTML = `<span>⚪ ❓ Sin registro A (agrega DNS tipo A hacia ${escapeHtml(serverIp || 'tu VPS')})</span>`;
            }
        } catch (dnsErr) {
            console.debug('Error comprobando resolución DNS:', dnsErr);
            feedbackEl.style.display = 'none';
        }
    }, 350);

    // --- Modal: Deploy App ---
    function openDeployModal() {
        formDeploy.reset();
        if (depDomainDnsFeedback) {
            depDomainDnsFeedback.style.display = 'none';
            depDomainDnsFeedback.innerHTML = '';
        }
        deployModal.style.display = 'flex';
    }

    if (btnGlobalDeploy) btnGlobalDeploy.addEventListener('click', openDeployModal);
    btnOpenDeployModal.addEventListener('click', openDeployModal);
    btnCloseDeployModal.addEventListener('click', () => {
        formDeploy.reset();
        if (depDomainDnsFeedback) depDomainDnsFeedback.style.display = 'none';
        deployModal.style.display = 'none';
    });
    btnCancelDeploy.addEventListener('click', () => {
        formDeploy.reset();
        if (depDomainDnsFeedback) depDomainDnsFeedback.style.display = 'none';
        deployModal.style.display = 'none';
    });
    deployModal.addEventListener('click', (e) => {
        if (e.target === deployModal) {
            formDeploy.reset();
            if (depDomainDnsFeedback) depDomainDnsFeedback.style.display = 'none';
            deployModal.style.display = 'none';
        }
    });

    if (depDomain && depDomainDnsFeedback) {
        depDomain.addEventListener('input', (e) => {
            checkDomainDnsDebounced(e.target.value, depDomainDnsFeedback);
        });
    }

    formDeploy.addEventListener('submit', async (e) => {
        e.preventDefault();
        btnSubmitDeploy.disabled = true;
        btnSubmitDeploy.innerHTML = '<span>Desplegando...</span>';
        showToast(`Iniciando despliegue de ${depName.value}...`, 'info');

        try {
            let res = null;
            const domain = depDomain.value.trim();
            const targetServer = selectedServerName || '';

            if (depDB.value) {
                res = await fetch(`/api/bootstrap-master?server=${encodeURIComponent(targetServer)}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        app_name: depName.value.trim(),
                        image: depImage.value.trim(),
                        port: parseInt(depPort.value || '80', 10),
                        domain: domain,
                        expose_public: true,
                        db_engine: depDB.value,
                        env_var_name: depEnv.value.trim() || 'DATABASE_URL',
                        target_node: 'manager'
                    })
                });
            } else {
                res = await fetch(`/api/deploy-service?server=${encodeURIComponent(targetServer)}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        name: depName.value.trim(),
                        imageSource: depImage.value.trim(),
                        port: parseInt(depPort.value || '80', 10),
                        domain: domain,
                        expose: true,
                        targetNode: 'manager'
                    })
                });
            }

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Fallo en despliegue: ${errText}`, 'error');
                return;
            }

            const streamResult = await consumeNDJSONStream(
                res,
                (stepMsg) => showToast(stepMsg, 'info'),
                (errMsg) => showToast(`Error en despliegue: ${errMsg}`, 'error')
            );

            if (streamResult.hasError) {
                return;
            }

            showToast(`¡Aplicación '${depName.value}' desplegada con éxito en '${targetServer || 'servidor'}'!`, 'success');
            deployModal.style.display = 'none';
            formDeploy.reset();
            if (selectedServerName) {
                await loadSwarmStatus(selectedServerName);
                await loadServiceLinks();
            }
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        } finally {
            btnSubmitDeploy.disabled = false;
            btnSubmitDeploy.innerHTML = '<span>Desplegar en Swarm</span>';
        }
    });

    // --- Modal: Deploy Database ---
    function setDBMode(mode) {
        dbDeployMode = mode;
        tabDBLocal.classList.toggle('active', mode === 'single-node');
        tabDBNode.classList.toggle('active', mode === 'multi-node');
        tabDBExternal.classList.toggle('active', mode === 'external');

        if (mode === 'single-node') {
            dbUrlField.style.display = 'none';
            dbPathField.style.display = 'flex';
            dbPortField.style.display = 'flex';
            dbTargetNodeField.style.display = 'none';
            btnSubmitDBText.textContent = 'Crear BD Local';
            return;
        }
        if (mode === 'multi-node') {
            dbUrlField.style.display = 'none';
            dbPathField.style.display = 'flex';
            dbPortField.style.display = 'flex';
            dbTargetNodeField.style.display = 'flex';
            btnSubmitDBText.textContent = 'Crear en Nodo Dedicado';
            return;
        }
        if (mode === 'external') {
            dbUrlField.style.display = 'flex';
            dbPathField.style.display = 'none';
            dbPortField.style.display = 'none';
            dbTargetNodeField.style.display = 'none';
            btnSubmitDBText.textContent = 'Registrar BD Externa';
            return;
        }
    }

    tabDBLocal.addEventListener('click', () => setDBMode('single-node'));
    tabDBNode.addEventListener('click', () => setDBMode('multi-node'));
    tabDBExternal.addEventListener('click', () => setDBMode('external'));

    dbEngine.addEventListener('change', () => {
        dbPort.value = getDefaultPort(dbEngine.value);
    });

    btnOpenDeployDBModal.addEventListener('click', () => {
        formDeployDB.reset();
        setDBMode('single-node');
        dbPort.value = getDefaultPort(dbEngine.value);
        dbModal.style.display = 'flex';
    });
    btnCloseDBModal.addEventListener('click', () => { formDeployDB.reset(); dbModal.style.display = 'none'; });
    btnCancelDB.addEventListener('click', () => { formDeployDB.reset(); dbModal.style.display = 'none'; });
    dbModal.addEventListener('click', (e) => {
        if (e.target === dbModal) { formDeployDB.reset(); dbModal.style.display = 'none'; }
    });

    formDeployDB.addEventListener('submit', async (e) => {
        e.preventDefault();
        btnSubmitDB.disabled = true;
        btnSubmitDBText.textContent = 'Creando...';

        try {
            const targetServer = selectedServerName || '';
            const payload = {
                name: dbName.value.trim(),
                engine: dbEngine.value,
                deployType: dbDeployMode,
                externalUrl: dbExternalURL.value.trim(),
                volumeHostPath: dbVolumePath.value.trim(),
                internalPort: parseInt(dbPort.value || getDefaultPort(dbEngine.value), 10),
                targetNode: dbTargetNode.value.trim() || 'manager'
            };

            const res = await fetch(`/api/deploy-db?server=${encodeURIComponent(targetServer)}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error: ${errText}`, 'error');
                return;
            }

            const streamResult = await consumeNDJSONStream(
                res,
                (stepMsg) => showToast(stepMsg, 'info'),
                (errMsg) => showToast(`Error al crear BD: ${errMsg}`, 'error')
            );

            if (streamResult.hasError) {
                return;
            }

            showToast(`¡Base de datos '${payload.name}' creada exitosamente en '${targetServer || 'servidor'}'!`, 'success');
            dbModal.style.display = 'none';
            formDeployDB.reset();
            if (selectedServerName) {
                await loadSwarmStatus(selectedServerName);
            }
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        } finally {
            btnSubmitDB.disabled = false;
            btnSubmitDBText.textContent = 'Crear Base de Datos';
        }
    });

    // --- Modal: Service Linking ---
    if (btnOpenLinkModal) {
        btnOpenLinkModal.addEventListener('click', () => {
            populateLinkModalDropdowns();
            formLink.reset();
            linkModal.style.display = 'flex';
        });
    }
    btnCloseLinkModal.addEventListener('click', () => { formLink.reset(); linkModal.style.display = 'none'; });
    btnCancelLink.addEventListener('click', () => { formLink.reset(); linkModal.style.display = 'none'; });
    linkModal.addEventListener('click', (e) => {
        if (e.target === linkModal) { formLink.reset(); linkModal.style.display = 'none'; }
    });

    formLink.addEventListener('submit', async (e) => {
        e.preventDefault();
        btnSubmitLink.disabled = true;
        btnSubmitLink.innerHTML = '<span>Conectando...</span>';

        try {
            const res = await fetch('/api/links', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    sourceSvc: linkFrom.value.trim(),
                    targetSvc: linkTo.value.trim(),
                    envVarName: linkVar.value.trim() || 'DATABASE_URL'
                })
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error: ${errText}`, 'error');
                return;
            }

            showToast(`¡Enlace creado! '${linkFrom.value}' conectado a '${linkTo.value}'`, 'success');
            linkModal.style.display = 'none';
            formLink.reset();
            await loadServiceLinks();
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        } finally {
            btnSubmitLink.disabled = false;
            btnSubmitLink.innerHTML = '<span>🔗 Conectar</span>';
        }
    });

    // --- Modal: Edit Service ---
    function openEditServiceModal(name, expose, domain) {
        formEditService.reset();
        editServiceName.value = name;
        editServiceTitle.textContent = `Ajustes: ${name}`;
        editServiceExpose.value = expose ? 'true' : 'false';
        editServiceDomain.value = domain || '';
        editDomainField.style.display = expose ? 'flex' : 'none';
        if (editDomainDnsFeedback) {
            editDomainDnsFeedback.style.display = 'none';
            editDomainDnsFeedback.innerHTML = '';
            if (expose && domain) {
                checkDomainDnsDebounced(domain, editDomainDnsFeedback);
            }
        }
        editServiceModal.style.display = 'flex';
    }

    editServiceExpose.addEventListener('change', () => {
        const isPublic = editServiceExpose.value === 'true';
        editDomainField.style.display = isPublic ? 'flex' : 'none';
        if (!isPublic && editDomainDnsFeedback) {
            editDomainDnsFeedback.style.display = 'none';
        } else if (isPublic && editDomainDnsFeedback && editServiceDomain.value) {
            checkDomainDnsDebounced(editServiceDomain.value, editDomainDnsFeedback);
        }
    });

    if (editServiceDomain && editDomainDnsFeedback) {
        editServiceDomain.addEventListener('input', (e) => {
            checkDomainDnsDebounced(e.target.value, editDomainDnsFeedback);
        });
    }

    btnCloseEditServiceModal.addEventListener('click', () => {
        formEditService.reset();
        if (editDomainDnsFeedback) editDomainDnsFeedback.style.display = 'none';
        editServiceModal.style.display = 'none';
    });
    btnCancelEditService.addEventListener('click', () => {
        formEditService.reset();
        if (editDomainDnsFeedback) editDomainDnsFeedback.style.display = 'none';
        editServiceModal.style.display = 'none';
    });
    editServiceModal.addEventListener('click', (e) => {
        if (e.target === editServiceModal) {
            formEditService.reset();
            if (editDomainDnsFeedback) editDomainDnsFeedback.style.display = 'none';
            editServiceModal.style.display = 'none';
        }
    });

    formEditService.addEventListener('submit', async (e) => {
        e.preventDefault();
        btnSubmitEditService.disabled = true;

        const name = editServiceName.value;
        const expose = editServiceExpose.value === 'true';
        const domain = editServiceDomain.value.trim();
        const port = parseInt(editServicePort.value || '80', 10);

        try {
            let svc = { name, expose, domain, port };
            const getRes = await fetch(`/api/services/${encodeURIComponent(name)}`);
            if (getRes.ok) {
                const existing = await getRes.json();
                svc = { ...existing, expose, domain, port };
            }

            const updateRes = await fetch(`/api/services/${encodeURIComponent(name)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(svc)
            });

            if (!updateRes.ok) {
                const errText = await updateRes.text();
                showToast(`Error: ${errText}`, 'error');
                return;
            }

            showToast(`¡Servicio '${name}' actualizado!`, 'success');
            editServiceModal.style.display = 'none';
            if (selectedServerName) {
                await loadSwarmStatus(selectedServerName);
            }
        } catch (err) {
            showToast(`Error: ${err.message}`, 'error');
        } finally {
            btnSubmitEditService.disabled = false;
        }
    });

    // --- Swarm Join Token Action ---
    if (btnCopyJoinToken) {
        btnCopyJoinToken.addEventListener('click', async () => {
            if (!selectedServerName) {
                showToast('Selecciona primero un servidor activo.', 'info');
                return;
            }
            btnCopyJoinToken.disabled = true;
            const originalHtml = btnCopyJoinToken.innerHTML;
            btnCopyJoinToken.innerHTML = '⏳ Obteniendo token...';
            try {
                const res = await fetch(`/api/nodes/join-token?role=worker&server=${encodeURIComponent(selectedServerName || '')}`);
                if (!res.ok) {
                    const errText = await res.text();
                    showToast(`Error obteniendo token: ${errText}`, 'error');
                    return;
                }
                const data = await res.json();
                const cmd = data.worker_cmd || (data.worker_token ? `docker swarm join --token ${data.worker_token} ${data.host || '127.0.0.1'}:2377` : '');
                if (!cmd) {
                    showToast('No se encontró el comando de unión en la respuesta', 'error');
                    return;
                }
                const copied = await copyToClipboard(cmd);
                if (copied) {
                    showToast('¡Comando "docker swarm join" copiado al portapapeles! Ejecútalo en el nuevo nodo worker.', 'success');
                } else {
                    prompt('Copia manualmente este comando en tu nuevo nodo worker:', cmd);
                }
            } catch (err) {
                showToast(`Error de red: ${err.message}`, 'error');
            } finally {
                btnCopyJoinToken.disabled = false;
                btnCopyJoinToken.innerHTML = originalHtml;
            }
        });
    }

    // --- Modal: Cloud Worker ---
    if (btnOpenWorkerModal) {
        btnOpenWorkerModal.addEventListener('click', () => {
            if (formWorker) formWorker.reset();
            const current = servers.find(s => s.name === selectedServerName) || activeServer;
            if (current) {
                if (current.cloudProvider === 'digitalocean') {
                    workerProvider.value = 'digitalocean';
                    workerApiKey.value = current.doApiToken || '';
                    workerRegion.value = 'nyc1';
                    workerPlan.value = 's-1vcpu-1gb';
                } else {
                    workerProvider.value = 'vultr';
                    workerApiKey.value = current.vultrApiToken || '';
                    workerRegion.value = 'mex';
                    workerPlan.value = 'vc2-1c-1gb';
                }
            }
            workerLogsBox.style.display = 'none';
            workerLogsContent.innerHTML = '';
            workerModal.style.display = 'flex';
        });
    }

    if (btnSidebarOpenWorker) {
        btnSidebarOpenWorker.addEventListener('click', () => {
            if (btnOpenWorkerModal) btnOpenWorkerModal.click();
            serverPopover.style.display = 'none';
        });
    }

    function closeWorkerModal() {
        if (formWorker) formWorker.reset();
        if (workerLogsBox) workerLogsBox.style.display = 'none';
        if (workerLogsContent) workerLogsContent.innerHTML = '';
        if (workerModal) workerModal.style.display = 'none';
    }
    if (btnCloseWorkerModal) btnCloseWorkerModal.addEventListener('click', closeWorkerModal);
    if (btnCancelWorker) btnCancelWorker.addEventListener('click', closeWorkerModal);
    if (workerModal) {
        workerModal.addEventListener('click', (e) => {
            if (e.target === workerModal) closeWorkerModal();
        });
    }

    if (formWorker) {
        formWorker.addEventListener('submit', async (e) => {
            e.preventDefault();
            btnSubmitWorker.disabled = true;
            btnSubmitWorkerText.textContent = 'Aprovisionando...';
            workerLogsBox.style.display = 'block';
            workerLogsContent.innerHTML = '<span style="color:var(--brand-primary); font-weight:700;">⚡ Conectando con orquestador cloud...</span><br>';

            const payload = {
                nodeName: workerName.value.trim(),
                provider: workerProvider.value,
                apiKey: workerApiKey.value.trim(),
                region: workerRegion.value.trim(),
                plan: workerPlan.value.trim(),
                labelType: workerLabel.value
            };

            try {
                const response = await fetch('/api/workers', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    workerLogsContent.innerHTML += `<span style="color:var(--status-offline);">✕ Error: ${escapeHtml(errorText)}</span><br>`;
                    return;
                }

                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { value, done } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop();

                    for (const rawLine of lines) {
                        let line = rawLine.trim();
                        if (!line) continue;
                        if (line.startsWith('data: ')) line = line.substring(6).trim();

                        try {
                            const event = JSON.parse(line);
                            const eventType = event.t || event.type;
                            const msg = event.m || event.message || (event.d && event.d.message) || '';

                            if (eventType === 'step') {
                                workerLogsContent.innerHTML += `<span style="color:var(--brand-primary); font-weight:700;">${escapeHtml(msg)}</span><br>`;
                            } else if (eventType === 'log') {
                                workerLogsContent.innerHTML += `<span style="color:var(--text-secondary);">${escapeHtml(msg)}</span><br>`;
                            } else if (eventType === 'error') {
                                workerLogsContent.innerHTML += `<span style="color:var(--status-offline); font-weight:700;">✕ ${escapeHtml(msg)}</span><br>`;
                            } else if (eventType === 'done') {
                                workerLogsContent.innerHTML += `<span style="color:var(--status-online); font-weight:800;">🎉 ${escapeHtml(msg || 'Worker provisionado con éxito!')}</span><br>`;
                                showToast(`¡Worker ${payload.nodeName} listo y unido al clúster!`, 'success');
                                setTimeout(async () => {
                                    workerModal.style.display = 'none';
                                    await loadHubState();
                                    if (selectedServerName) await loadSwarmStatus(selectedServerName);
                                }, 1500);
                            }
                            workerLogsBox.scrollTop = workerLogsBox.scrollHeight;
                        } catch (errJson) {
                            console.error('Error parseando stream:', errJson);
                        }
                    }
                }
            } catch (err) {
                workerLogsContent.innerHTML += `<span style="color:var(--status-offline);">✕ Error de red: ${escapeHtml(err.message)}</span><br>`;
            } finally {
                btnSubmitWorker.disabled = false;
                btnSubmitWorkerText.textContent = 'Crear y Unir al Clúster';
            }
        });
    }

    // --- Modal: Visor de Logs en Tiempo Real ---
    function openLogsModal(serviceName) {
        if (!logsModal) return;
        currentLogsServiceName = serviceName;
        if (logsServiceNameTitle) logsServiceNameTitle.textContent = serviceName;
        if (logsSearchInput) logsSearchInput.value = '';
        if (logsTerminalContent) logsTerminalContent.textContent = 'Consultando logs del contenedor...';
        logsModal.style.display = 'flex';
        fetchAndRenderLogs(false);
        startLogsPolling();
    }

    function closeLogsModal() {
        stopLogsPolling();
        if (logsModal) logsModal.style.display = 'none';
        currentLogsServiceName = '';
        rawLogsText = '';
    }

    function startLogsPolling() {
        stopLogsPolling();
        if (logsLiveToggle && logsLiveToggle.checked) {
            logsPollTimer = setInterval(() => {
                if (logsModal && logsModal.style.display !== 'none' && currentLogsServiceName) {
                    fetchAndRenderLogs(true);
                } else {
                    stopLogsPolling();
                }
            }, 3000);
        }
    }

    function stopLogsPolling() {
        if (logsPollTimer) {
            clearInterval(logsPollTimer);
            logsPollTimer = null;
        }
    }

    function renderFilteredLogs() {
        if (!logsTerminalContent) return;
        const query = (logsSearchInput ? logsSearchInput.value : '').trim().toLowerCase();
        if (!rawLogsText) {
            logsTerminalContent.textContent = '(Sin registros recibidos)';
            if (logsLineCount) logsLineCount.textContent = '0 líneas';
            return;
        }

        const lines = rawLogsText.split('\n');
        if (logsLineCount) {
            logsLineCount.textContent = `${lines.length} líneas`;
        }

        if (!query) {
            logsTerminalContent.textContent = rawLogsText;
        } else {
            const matched = lines.filter(line => line.toLowerCase().includes(query));
            if (matched.length === 0) {
                logsTerminalContent.innerHTML = `<span style="color:var(--text-muted); font-style:italic;">No se encontraron registros para "${escapeHtml(query)}"</span>`;
                return;
            }
            const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
            const regex = new RegExp(`(${escapedQuery})`, 'gi');
            const highlighted = matched.map(line => {
                return escapeHtml(line).replace(regex, '<mark>$1</mark>');
            }).join('\n');
            logsTerminalContent.innerHTML = highlighted;
        }

        if (logsAutoscrollToggle && logsAutoscrollToggle.checked && logsTerminalViewport) {
            logsTerminalViewport.scrollTop = logsTerminalViewport.scrollHeight;
        }
    }

    async function fetchAndRenderLogs(isPoll) {
        if (!currentLogsServiceName) return;
        const lines = logsTailSelect ? logsTailSelect.value : '100';

        try {
            const res = await fetch(`/api/logs?name=${encodeURIComponent(currentLogsServiceName)}&lines=${encodeURIComponent(lines)}`);
            if (res.ok) {
                const data = await res.json();
                rawLogsText = (data && typeof data.logs === 'string') ? data.logs : '';
                renderFilteredLogs();
                if (logsLastUpdate) logsLastUpdate.textContent = `Actualizado: ${new Date().toLocaleTimeString()}`;
                if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-online logs-live-pulse';
            } else {
                const errText = await res.text();
                if (!isPoll && logsTerminalContent) {
                    logsTerminalContent.textContent = `Error al consultar logs: ${errText}`;
                }
                if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-offline';
            }
        } catch (err) {
            if (!isPoll && logsTerminalContent) {
                logsTerminalContent.textContent = `Fallo de conexión: ${err.message}`;
            }
            if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-offline';
        }
    }

    if (btnCloseLogsModal) btnCloseLogsModal.addEventListener('click', closeLogsModal);
    if (btnDismissLogs) btnDismissLogs.addEventListener('click', closeLogsModal);
    if (logsModal) {
        logsModal.addEventListener('click', (e) => {
            if (e.target === logsModal) closeLogsModal();
        });
    }

    if (btnRefreshLogs) {
        btnRefreshLogs.addEventListener('click', () => fetchAndRenderLogs(false));
    }

    if (logsTailSelect) {
        logsTailSelect.addEventListener('change', () => fetchAndRenderLogs(false));
    }

    if (logsLiveToggle) {
        logsLiveToggle.addEventListener('change', () => {
            if (logsLiveToggle.checked) {
                startLogsPolling();
            } else {
                stopLogsPolling();
            }
        });
    }

    if (logsSearchInput) {
        logsSearchInput.addEventListener('input', debounce(() => renderFilteredLogs(), 120));
    }

    if (btnCopyLogs) {
        btnCopyLogs.addEventListener('click', async () => {
            if (!rawLogsText) {
                showToast('No hay logs para copiar', 'info');
                return;
            }
            const ok = await copyToClipboard(rawLogsText);
            if (ok) {
                showToast('Logs copiados al portapapeles', 'success');
            } else {
                showToast('No se pudo copiar automáticamente', 'error');
            }
        });
    }

    if (btnDownloadLogs) {
        btnDownloadLogs.addEventListener('click', () => {
            if (!rawLogsText) {
                showToast('No hay logs para descargar', 'info');
                return;
            }
            const blob = new Blob([rawLogsText], { type: 'text/plain;charset=utf-8' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${currentLogsServiceName || 'service'}_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            showToast('Descarga iniciada', 'success');
        });
    }

    // --- Reinicio Rápido de Servicios / Contenedores / Bases de Datos ---
    async function restartServiceOrContainer(name, btnElement) {
        if (!name) return;
        if (!confirm(`¿Estás seguro de reiniciar el servicio/contenedor '${name}'?`)) return;

        let originalContent = '';
        if (btnElement) {
            btnElement.disabled = true;
            originalContent = btnElement.innerHTML;
            btnElement.innerHTML = '<span>⏳...</span>';
        }

        showToast(`Reiniciando '${name}' en Docker...`, 'info');

        try {
            const res = await fetch('/api/services/restart', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error al reiniciar: ${errText}`, 'error');
                return;
            }

            const data = await res.json();
            const typeLabel = data.type === 'swarm_service' ? 'Servicio Swarm' :
                             data.type === 'systemd_service' ? 'Servicio Host' : 'Contenedor';
            showToast(`¡${typeLabel} '${name}' reiniciado correctamente!`, 'success');

            if (selectedServerName) {
                await loadSwarmStatus(selectedServerName);
            }
        } catch (err) {
            showToast(`Fallo de conexión: ${err.message}`, 'error');
        } finally {
            if (btnElement) {
                btnElement.disabled = false;
                btnElement.innerHTML = originalContent;
            }
        }
    }

    if (btnRestartFromLogs) {
        btnRestartFromLogs.addEventListener('click', async () => {
            if (currentLogsServiceName) {
                await restartServiceOrContainer(currentLogsServiceName, btnRestartFromLogs);
                fetchAndRenderLogs(false);
            }
        });
    }

    // --- Backup Manual de Bases de Datos ---
    async function triggerDatabaseBackup(name, engine, btnElement) {
        if (!name) return;
        if (!confirm(`¿Generar snapshot de respaldo para la base de datos '${name}' (${engine || 'postgres'}) ahora?`)) return;

        let originalContent = '';
        if (btnElement) {
            btnElement.disabled = true;
            originalContent = btnElement.innerHTML;
            btnElement.innerHTML = '<span>⏳ Respaldando...</span>';
        }

        showToast(`Generando respaldo de '${name}'...`, 'info');

        try {
            const res = await fetch(`/api/databases/backup?server=${encodeURIComponent(selectedServerName || '')}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    targetName: name,
                    targetType: 'database',
                    engine: engine || 'postgres',
                    server: selectedServerName || ''
                })
            });

            if (!res.ok) {
                const errText = await res.text();
                showToast(`Error al crear respaldo: ${errText}`, 'error');
                return;
            }

            const backup = await res.json();
            const filename = backup.filename || backup.Filename || `${name}.sql.gz`;
            const backupId = backup.id || backup.ID;

            showToast(`¡Respaldo '${filename}' generado con éxito!`, 'success');

            if (backupId) {
                const downloadLink = document.createElement('a');
                downloadLink.href = `/api/backups/download?id=${backupId}&server=${encodeURIComponent(selectedServerName || '')}`;
                downloadLink.download = filename;
                document.body.appendChild(downloadLink);
                downloadLink.click();
                document.body.removeChild(downloadLink);
                showToast(`Descarga de '${filename}' iniciada`, 'info');
            }

            if (backupsModal && backupsModal.style.display !== 'none') {
                loadBackups();
            }
        } catch (err) {
            showToast(`Fallo de conexión: ${err.message}`, 'error');
        } finally {
            if (btnElement) {
                btnElement.disabled = false;
                btnElement.innerHTML = originalContent;
            }
        }
    }

    // --- Modal: Historial y Restauración de Backups ---
    async function loadBackups() {
        if (!backupsTableBody) return;
        backupsTableBody.innerHTML = '<tr><td colspan="6" class="t-td-empty">Consultando copias de seguridad...</td></tr>';
        try {
            const res = await fetch(`/api/backups?server=${encodeURIComponent(selectedServerName || '')}`);
            if (!res.ok) {
                backupsTableBody.innerHTML = '<tr><td colspan="6" class="t-td-empty">Error al consultar respaldos del servidor.</td></tr>';
                return;
            }
            const backups = await res.json();
            if (!backups || backups.length === 0) {
                backupsTableBody.innerHTML = '<tr><td colspan="6" class="t-td-empty">No se encontraron copias de seguridad registradas. Genera una desde tus bases de datos.</td></tr>';
                return;
            }
            backupsTableBody.innerHTML = '';
            backups.forEach(b => {
                const tr = document.createElement('tr');
                const sizeStr = formatFileSize(b.sizeBytes || b.SizeBytes || 0);
                const bId = b.id || b.ID;
                const targetName = b.targetName || b.TargetName || '—';
                const engine = b.engine || b.Engine || 'database';
                const filename = b.filename || b.Filename || '—';
                const createdAt = b.createdAt || b.CreatedAt || '—';

                tr.innerHTML = `
                    <td style="font-weight:700; color:#fff;">${escapeHtml(targetName)}</td>
                    <td><span class="svc-pill svc-pill-active">${escapeHtml(engine)}</span></td>
                    <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-muted); max-width:200px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="${escapeHtml(filename)}">${escapeHtml(filename)}</td>
                    <td style="color:var(--text-secondary); font-size:0.8rem;">${escapeHtml(sizeStr)}</td>
                    <td style="color:var(--text-muted); font-size:0.75rem;">${escapeHtml(createdAt)}</td>
                    <td>
                        <div style="display:flex; gap:6px; align-items:center;">
                            <a class="mini-btn btn-dl-backup" href="/api/backups/download?id=${bId}&server=${encodeURIComponent(selectedServerName || '')}" download="${escapeHtml(filename)}" title="Descargar snapshot SQL/archivo" style="text-decoration:none;">
                                📥 Descargar
                            </a>
                            <button type="button" class="mini-btn btn-restore-backup" data-id="${bId}" data-target="${escapeHtml(targetName)}" title="Restaurar base de datos desde este snapshot" style="color:var(--accent-warning, #f59e0b);">
                                ♻️ Restaurar
                            </button>
                            <button type="button" class="mini-btn btn-del-backup" data-id="${bId}" data-file="${escapeHtml(filename)}" title="Eliminar respaldo" style="color:var(--status-offline);">
                                🗑️
                            </button>
                        </div>
                    </td>
                `;
                backupsTableBody.appendChild(tr);
            });

            // Bind restore events
            document.querySelectorAll('.btn-restore-backup').forEach(btn => {
                btn.addEventListener('click', async () => {
                    const id = btn.getAttribute('data-id');
                    const target = btn.getAttribute('data-target');
                    if (!id) return;
                    if (!confirm(`¿Restaurar la base de datos '${target}' desde este respaldo? ADVERTENCIA: Esta operación sobrescribirá los datos actuales con el contenido del snapshot.`)) return;
                    btn.disabled = true;
                    showToast(`Restaurando '${target}' desde snapshot...`, 'info');
                    try {
                        const res = await fetch(`/api/backups/restore?server=${encodeURIComponent(selectedServerName || '')}`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ backupId: parseInt(id, 10), server: selectedServerName || '' })
                        });
                        if (res.ok) {
                            showToast(`¡Base de datos '${target}' restaurada con éxito!`, 'success');
                            if (selectedServerName) await loadSwarmStatus(selectedServerName);
                        } else {
                            const errText = await res.text();
                            showToast(`Error al restaurar: ${errText}`, 'error');
                        }
                    } catch (err) {
                        showToast(`Error de red: ${err.message}`, 'error');
                    } finally {
                        btn.disabled = false;
                    }
                });
            });

            // Bind delete events
            document.querySelectorAll('.btn-del-backup').forEach(btn => {
                btn.addEventListener('click', async () => {
                    const id = btn.getAttribute('data-id');
                    const file = btn.getAttribute('data-file');
                    if (!id) return;
                    if (!confirm(`¿Eliminar definitivamente el respaldo '${file}'?`)) return;
                    btn.disabled = true;
                    try {
                        const res = await fetch(`/api/backups?id=${encodeURIComponent(id)}&server=${encodeURIComponent(selectedServerName || '')}`, {
                            method: 'DELETE'
                        });
                        if (res.ok) {
                            showToast(`Respaldo eliminado`, 'info');
                            await loadBackups();
                        } else {
                            const errText = await res.text();
                            showToast(`Error al eliminar: ${errText}`, 'error');
                        }
                    } catch (err) {
                        showToast(`Error de red: ${err.message}`, 'error');
                    } finally {
                        btn.disabled = false;
                    }
                });
            });

        } catch (err) {
            backupsTableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty">Error cargando copias de seguridad: ${escapeHtml(err.message)}</td></tr>`;
        }
    }

    if (btnOpenBackupsModal) {
        btnOpenBackupsModal.addEventListener('click', () => {
            if (backupsModal) {
                backupsModal.style.display = 'flex';
                loadBackups();
            }
        });
    }

    function closeBackupsModal() {
        if (backupsModal) backupsModal.style.display = 'none';
    }
    if (btnCloseBackupsModal) btnCloseBackupsModal.addEventListener('click', closeBackupsModal);
    if (btnDismissBackupsModal) btnDismissBackupsModal.addEventListener('click', closeBackupsModal);
    if (btnRefreshBackupsModal) btnRefreshBackupsModal.addEventListener('click', () => loadBackups());
    if (backupsModal) {
        backupsModal.addEventListener('click', (e) => {
            if (e.target === backupsModal) closeBackupsModal();
        });
    }

    // --- Modal: Gestor de Variables de Entorno (.env) ---
    function openEnvModal(serviceName) {
        if (!envModal) return;
        currentEnvServiceName = serviceName;
        currentEnvMode = 'table';
        if (envModalServiceName) envModalServiceName.textContent = serviceName;
        if (envModalStatus) envModalStatus.textContent = '';
        if (envTabTable) envTabTable.classList.add('active');
        if (envTabRaw) envTabRaw.classList.remove('active');
        if (envTableView) envTableView.style.display = 'block';
        if (envRawView) envRawView.style.display = 'none';

        if (envTableBody) {
            envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Consultando variables de '${escapeHtml(serviceName)}'...</td></tr>`;
        }
        if (envRawTextarea) envRawTextarea.value = '';

        envModal.style.display = 'flex';
        fetchAndRenderEnvVars(serviceName);
    }

    function closeEnvModal() {
        if (envModal) envModal.style.display = 'none';
        currentEnvServiceName = '';
    }

    async function fetchAndRenderEnvVars(serviceName) {
        try {
            const res = await fetch(`/api/env?service=${encodeURIComponent(serviceName)}`);
            if (!res.ok) {
                if (envTableBody) {
                    envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas aún. Pulsa '+ Agregar Variable'.</td></tr>`;
                }
                return;
            }
            const data = await res.json();
            const raw = data.rawContent || '';
            if (envRawTextarea) envRawTextarea.value = raw;
            renderEnvTableFromRaw(raw);
        } catch (err) {
            if (envTableBody) {
                envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty" style="color:var(--status-offline);">Error de conexión: ${escapeHtml(err.message)}</td></tr>`;
            }
        }
    }

    function isSensitiveKey(key) {
        const k = (key || '').toUpperCase();
        return k.includes('PASS') || k.includes('SECRET') || k.includes('KEY') || k.includes('TOKEN') || k.includes('AUTH') || k.includes('PRIVATE');
    }

    function renderEnvTableFromRaw(rawContent) {
        if (!envTableBody) return;
        envTableBody.innerHTML = '';

        const lines = (rawContent || '').split('\n');
        let count = 0;

        lines.forEach(line => {
            const trimmed = line.trim();
            if (!trimmed || trimmed.startsWith('#')) return;
            const idx = trimmed.indexOf('=');
            if (idx === -1) return;

            const key = trimmed.slice(0, idx).trim();
            let val = trimmed.slice(idx + 1).trim();
            if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
                val = val.slice(1, -1);
            }

            createEnvTableRow(key, val);
            count++;
        });

        if (count === 0) {
            envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas. Pulsa '+ Agregar Variable'.</td></tr>`;
        }
    }

    function createEnvTableRow(key, val) {
        if (!envTableBody) return;

        const emptyTr = envTableBody.querySelector('.t-td-empty');
        if (emptyTr) emptyTr.parentElement.remove();

        const tr = document.createElement('tr');
        const sensitive = isSensitiveKey(key);

        tr.innerHTML = `
            <td>
                <input type="text" class="env-key-input" placeholder="VARIABLE_NAME" value="${escapeHtml(key || '')}" spellcheck="false">
            </td>
            <td>
                <div class="env-val-wrap">
                    <input type="${sensitive ? 'password' : 'text'}" class="env-val-input" placeholder="valor" value="${escapeHtml(val || '')}" spellcheck="false">
                    <button type="button" class="env-toggle-mask" title="${sensitive ? 'Mostrar valor' : 'Ocultar valor'}">
                        ${sensitive ? '👁️' : '🔒'}
                    </button>
                </div>
            </td>
            <td style="text-align: center;">
                <button type="button" class="env-del-btn" title="Eliminar variable">✕</button>
            </td>
        `;

        const maskBtn = tr.querySelector('.env-toggle-mask');
        const valInput = tr.querySelector('.env-val-input');
        maskBtn.addEventListener('click', () => {
            const isPass = valInput.type === 'password';
            valInput.type = isPass ? 'text' : 'password';
            maskBtn.textContent = isPass ? '🔒' : '👁️';
            maskBtn.title = isPass ? 'Ocultar valor' : 'Mostrar valor';
        });

        const delBtn = tr.querySelector('.env-del-btn');
        delBtn.addEventListener('click', () => {
            tr.remove();
            if (envTableBody.children.length === 0) {
                envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas. Pulsa '+ Agregar Variable'.</td></tr>`;
            }
        });

        envTableBody.appendChild(tr);
    }

    function collectEnvFromTable() {
        if (!envTableBody) return '';
        const rows = envTableBody.querySelectorAll('tr');
        const lines = [];

        rows.forEach(tr => {
            const keyInput = tr.querySelector('.env-key-input');
            const valInput = tr.querySelector('.env-val-input');
            if (keyInput && valInput) {
                const k = keyInput.value.trim();
                const v = valInput.value.trim();
                if (k) {
                    lines.push(`${k}=${v}`);
                }
            }
        });

        return lines.join('\n');
    }

    if (envTabTable) {
        envTabTable.addEventListener('click', () => {
            if (currentEnvMode === 'raw') {
                renderEnvTableFromRaw(envRawTextarea.value);
            }
            currentEnvMode = 'table';
            envTabTable.classList.add('active');
            envTabRaw.classList.remove('active');
            envTableView.style.display = 'block';
            envRawView.style.display = 'none';
        });
    }

    if (envTabRaw) {
        envTabRaw.addEventListener('click', () => {
            if (currentEnvMode === 'table') {
                envRawTextarea.value = collectEnvFromTable();
            }
            currentEnvMode = 'raw';
            envTabRaw.classList.add('active');
            envTabTable.classList.remove('active');
            envRawView.style.display = 'block';
            envTableView.style.display = 'none';
        });
    }

    if (btnAddEnvRow) {
        btnAddEnvRow.addEventListener('click', () => {
            if (currentEnvMode === 'raw') {
                envTabTable.click();
            }
            createEnvTableRow('', '');
            const inputs = envTableBody.querySelectorAll('.env-key-input');
            if (inputs.length > 0) {
                inputs[inputs.length - 1].focus();
            }
        });
    }

    if (btnCopyEnv) {
        btnCopyEnv.addEventListener('click', async () => {
            const textToCopy = currentEnvMode === 'raw' ? envRawTextarea.value : collectEnvFromTable();
            if (!textToCopy) {
                showToast('No hay variables para copiar', 'info');
                return;
            }
            const ok = await copyToClipboard(textToCopy);
            if (ok) {
                showToast('Variables .env copiadas al portapapeles', 'success');
            } else {
                showToast('No se pudo copiar automáticamente', 'error');
            }
        });
    }

    if (btnSaveEnv) {
        btnSaveEnv.addEventListener('click', async () => {
            if (!currentEnvServiceName) return;
            const content = currentEnvMode === 'raw' ? envRawTextarea.value : collectEnvFromTable();

            btnSaveEnv.disabled = true;
            const originalText = btnSaveEnv.innerHTML;
            btnSaveEnv.innerHTML = '<span>⏳ Guardando...</span>';
            if (envModalStatus) envModalStatus.textContent = 'Actualizando servicio en Swarm...';

            try {
                const res = await fetch('/api/env', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        serviceName: currentEnvServiceName,
                        rawContent: content
                    })
                });

                if (!res.ok) {
                    const errText = await res.text();
                    showToast(`Error al guardar variables: ${errText}`, 'error');
                    if (envModalStatus) envModalStatus.textContent = `Error: ${errText}`;
                    return;
                }

                showToast(`¡Variables de entorno de '${currentEnvServiceName}' actualizadas!`, 'success');
                closeEnvModal();
                if (selectedServerName) {
                    await loadSwarmStatus(selectedServerName);
                }
            } catch (err) {
                showToast(`Fallo de conexión: ${err.message}`, 'error');
                if (envModalStatus) envModalStatus.textContent = `Fallo: ${err.message}`;
            } finally {
                btnSaveEnv.disabled = false;
                btnSaveEnv.innerHTML = originalText;
            }
        });
    }

    if (btnCloseEnvModal) btnCloseEnvModal.addEventListener('click', closeEnvModal);
    if (btnCancelEnv) btnCancelEnv.addEventListener('click', closeEnvModal);
    if (envModal) {
        envModal.addEventListener('click', (e) => {
            if (e.target === envModal) closeEnvModal();
        });
    }

    const btnEditServiceOpenEnv = document.getElementById('btnEditServiceOpenEnv');
    if (btnEditServiceOpenEnv) {
        btnEditServiceOpenEnv.addEventListener('click', () => {
            const name = editServiceName.value;
            editServiceModal.style.display = 'none';
            if (name) openEnvModal(name);
        });
    }

    // ==========================================================================
    // --- Modal: Explorador de Volúmenes y Archivos Persistentes (/opt/data) ---
    // ==========================================================================
    const volumeModal = document.getElementById('volumeModal');
    const btnCloseVolumeModal = document.getElementById('btnCloseVolumeModal');
    const btnCloseVolumeModalBottom = document.getElementById('btnCloseVolumeModalBottom');
    const btnDeskVolumes = document.getElementById('btnDeskVolumes');
    const volumeBreadcrumbs = document.getElementById('volumeBreadcrumbs');
    const btnVolumeNewFolder = document.getElementById('btnVolumeNewFolder');
    const volumeFileInput = document.getElementById('volumeFileInput');
    const btnVolumeRefresh = document.getElementById('btnVolumeRefresh');
    const volumeSearchInput = document.getElementById('volumeSearchInput');
    const volumePathDisplay = document.getElementById('volumePathDisplay');
    const volumeFilesView = document.getElementById('volumeFilesView');
    const volumeTableBody = document.getElementById('volumeTableBody');
    const volumeEditorView = document.getElementById('volumeEditorView');
    const btnVolumeBackToList = document.getElementById('btnVolumeBackToList');
    const volumeEditorFilePath = document.getElementById('volumeEditorFilePath');
    const btnVolumeEditorDownload = document.getElementById('btnVolumeEditorDownload');
    const btnVolumeEditorSave = document.getElementById('btnVolumeEditorSave');
    const volumeEditorTextarea = document.getElementById('volumeEditorTextarea');
    const volumeModalStatus = document.getElementById('volumeModalStatus');

    let currentVolumePath = '/opt/data';
    let currentVolumeFiles = [];
    let currentEditingFilePath = '';

    function formatFileSize(bytes) {
        if (!bytes || bytes <= 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    function getFileIcon(name, isDir) {
        if (isDir) return '📁';
        const ext = name.split('.').pop().toLowerCase();
        switch (ext) {
            case 'sql': return '🗄️';
            case 'gz':
            case 'tar':
            case 'zip': return '🗜️';
            case 'json':
            case 'yaml':
            case 'yml':
            case 'conf':
            case 'toml': return '⚙️';
            case 'log': return '📜';
            case 'env': return '🔑';
            case 'sh': return '💻';
            default: return '📄';
        }
    }

    function openVolumeModal(initialPath) {
        if (!volumeModal) return;
        currentVolumePath = initialPath || '/opt/data';
        if (volumeSearchInput) volumeSearchInput.value = '';
        if (volumeFilesView) volumeFilesView.style.display = 'block';
        if (volumeEditorView) volumeEditorView.style.display = 'none';
        volumeModal.style.display = 'flex';
        fetchAndRenderVolumeFiles(currentVolumePath);
    }

    function closeVolumeModal() {
        if (volumeModal) volumeModal.style.display = 'none';
        currentEditingFilePath = '';
    }

    function renderVolumeBreadcrumbs(targetPath) {
        if (!volumeBreadcrumbs) return;
        volumeBreadcrumbs.innerHTML = '';

        const base = '/opt/data';
        let rel = targetPath.startsWith(base) ? targetPath.slice(base.length) : targetPath;
        if (rel.startsWith('/')) rel = rel.slice(1);
        const parts = rel ? rel.split('/').filter(Boolean) : [];

        const rootCrumb = document.createElement('span');
        rootCrumb.className = `breadcrumb-crumb ${parts.length === 0 ? 'active' : ''}`;
        rootCrumb.textContent = '🏠 /opt/data';
        rootCrumb.addEventListener('click', () => {
            if (currentVolumePath !== base) {
                currentVolumePath = base;
                fetchAndRenderVolumeFiles(base);
            }
        });
        volumeBreadcrumbs.appendChild(rootCrumb);

        let accumulated = base;
        parts.forEach((p, idx) => {
            const sep = document.createElement('span');
            sep.className = 'breadcrumb-sep';
            sep.textContent = '/';
            volumeBreadcrumbs.appendChild(sep);

            accumulated += '/' + p;
            const crumb = document.createElement('span');
            const isLast = idx === parts.length - 1;
            crumb.className = `breadcrumb-crumb ${isLast ? 'active' : ''}`;
            crumb.textContent = p;
            const target = accumulated;
            if (!isLast) {
                crumb.addEventListener('click', () => {
                    currentVolumePath = target;
                    fetchAndRenderVolumeFiles(target);
                });
            }
            volumeBreadcrumbs.appendChild(crumb);
        });
    }

    async function fetchAndRenderVolumeFiles(targetPath) {
        if (!volumeTableBody) return;
        renderVolumeBreadcrumbs(targetPath);
        if (volumePathDisplay) volumePathDisplay.textContent = targetPath;
        if (volumeModalStatus) volumeModalStatus.textContent = 'Explorando directorio remoto...';

        volumeTableBody.innerHTML = '<tr><td colspan="4" class="t-td-empty">Explorando directorio remoto...</td></tr>';

        try {
            const srvParam = selectedServerName ? `&server=${encodeURIComponent(selectedServerName)}` : '';
            const res = await fetch(`/api/volumes/files?path=${encodeURIComponent(targetPath)}${srvParam}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || 'Error al obtener archivos');
            }
            const items = await res.json();
            currentVolumeFiles = Array.isArray(items) ? items : [];

            // Carpetas primero, luego archivos alfabéticos
            currentVolumeFiles.sort((a, b) => {
                if (a.isDir && !b.isDir) return -1;
                if (!a.isDir && b.isDir) return 1;
                return a.name.localeCompare(b.name);
            });

            renderVolumeTableRows(currentVolumeFiles);
            if (volumeModalStatus) {
                const dirs = currentVolumeFiles.filter(f => f.isDir).length;
                const files = currentVolumeFiles.filter(f => !f.isDir).length;
                volumeModalStatus.textContent = `${dirs} carpetas, ${files} archivos en ${targetPath}`;
            }
        } catch (err) {
            console.error('Error cargando archivos de volumen:', err);
            volumeTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty" style="color:var(--status-offline);">Error: ${escapeHtml(err.message)}</td></tr>`;
            if (volumeModalStatus) volumeModalStatus.textContent = `Error: ${err.message}`;
        }
    }

    function renderVolumeTableRows(items) {
        if (!volumeTableBody) return;
        volumeTableBody.innerHTML = '';

        if (!items || items.length === 0) {
            volumeTableBody.innerHTML = '<tr><td colspan="4" class="t-td-empty">Directorio vacío. Puedes crear carpetas o subir archivos.</td></tr>';
            return;
        }

        items.forEach(item => {
            const tr = document.createElement('tr');
            tr.className = 'volume-item-row';

            const icon = getFileIcon(item.name, item.isDir);
            const sizeStr = item.isDir ? '—' : formatFileSize(item.size);
            const nameClass = item.isDir ? 'volume-item-folder' : 'volume-item-file';

            tr.innerHTML = `
                <td>
                    <div class="volume-item-name">
                        <span>${icon}</span>
                        <span class="${nameClass}" data-path="${escapeHtml(item.path)}" data-dir="${item.isDir}">${escapeHtml(item.name)}</span>
                    </div>
                </td>
                <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-secondary);">${escapeHtml(sizeStr)}</td>
                <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-muted);">${escapeHtml(item.modTime || '—')}</td>
                <td style="text-align:right;">
                    <div style="display:flex; justify-content:flex-end; gap:4px;">
                        ${!item.isDir ? `
                            <button type="button" class="mini-btn btn-vol-read" data-path="${escapeHtml(item.path)}" title="Ver o editar texto">
                                👁️
                            </button>
                            <button type="button" class="mini-btn btn-vol-dl" data-path="${escapeHtml(item.path)}" title="Descargar archivo">
                                ⬇️
                            </button>
                        ` : ''}
                        <button type="button" class="mini-btn btn-vol-del" data-path="${escapeHtml(item.path)}" data-name="${escapeHtml(item.name)}" style="color:var(--status-offline);" title="Eliminar">
                            🗑️
                        </button>
                    </div>
                </td>
            `;

            if (item.isDir) {
                const folderLink = tr.querySelector('.volume-item-folder');
                if (folderLink) {
                    folderLink.addEventListener('click', () => {
                        currentVolumePath = item.path;
                        fetchAndRenderVolumeFiles(item.path);
                    });
                }
            }

            volumeTableBody.appendChild(tr);
        });

        volumeTableBody.querySelectorAll('.btn-vol-read').forEach(btn => {
            btn.addEventListener('click', () => {
                const path = btn.getAttribute('data-path');
                if (path) openVolumeFileEditor(path);
            });
        });

        volumeTableBody.querySelectorAll('.btn-vol-dl').forEach(btn => {
            btn.addEventListener('click', () => {
                const path = btn.getAttribute('data-path');
                if (path) {
                    const srvParam = selectedServerName ? `&server=${encodeURIComponent(selectedServerName)}` : '';
                    window.location.href = `/api/volumes/download?path=${encodeURIComponent(path)}${srvParam}`;
                }
            });
        });

        volumeTableBody.querySelectorAll('.btn-vol-del').forEach(btn => {
            btn.addEventListener('click', async () => {
                const path = btn.getAttribute('data-path');
                const name = btn.getAttribute('data-name');
                if (!path) return;
                if (!confirm(`¿Estás seguro de eliminar permanentemente "${name}" del VPS?`)) return;

                try {
                    btn.disabled = true;
                    const srvParam = selectedServerName ? `&server=${encodeURIComponent(selectedServerName)}` : '';
                    const res = await fetch(`/api/volumes/delete?path=${encodeURIComponent(path)}${srvParam}`, {
                        method: 'DELETE'
                    });
                    if (!res.ok) {
                        const errText = await res.text();
                        throw new Error(errText || 'Error al eliminar');
                    }
                    showToast(`"${name}" eliminado correctamente`, 'success');
                    fetchAndRenderVolumeFiles(currentVolumePath);
                } catch (err) {
                    showToast(`Error al eliminar: ${err.message}`, 'error');
                    btn.disabled = false;
                }
            });
        });
    }

    async function openVolumeFileEditor(filePath) {
        currentEditingFilePath = filePath;
        if (volumeFilesView) volumeFilesView.style.display = 'none';
        if (volumeEditorView) volumeEditorView.style.display = 'block';
        if (volumeEditorFilePath) volumeEditorFilePath.textContent = filePath;
        if (volumeEditorTextarea) volumeEditorTextarea.value = 'Cargando contenido del archivo...';

        try {
            const srvParam = selectedServerName ? `&server=${encodeURIComponent(selectedServerName)}` : '';
            const res = await fetch(`/api/volumes/read?path=${encodeURIComponent(filePath)}${srvParam}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || 'Error al leer archivo');
            }
            const data = await res.json();
            if (volumeEditorTextarea) volumeEditorTextarea.value = data.content || '';
        } catch (err) {
            if (volumeEditorTextarea) volumeEditorTextarea.value = `Error al leer archivo: ${err.message}`;
            showToast(`Error: ${err.message}`, 'error');
        }
    }

    if (btnVolumeBackToList) {
        btnVolumeBackToList.addEventListener('click', () => {
            if (volumeEditorView) volumeEditorView.style.display = 'none';
            if (volumeFilesView) volumeFilesView.style.display = 'block';
            currentEditingFilePath = '';
        });
    }

    if (btnVolumeEditorDownload) {
        btnVolumeEditorDownload.addEventListener('click', () => {
            if (currentEditingFilePath) {
                const srvParam = selectedServerName ? `&server=${encodeURIComponent(selectedServerName)}` : '';
                window.location.href = `/api/volumes/download?path=${encodeURIComponent(currentEditingFilePath)}${srvParam}`;
            }
        });
    }

    if (btnVolumeEditorSave) {
        btnVolumeEditorSave.addEventListener('click', async () => {
            if (!currentEditingFilePath || !volumeEditorTextarea) return;
            const originalText = btnVolumeEditorSave.innerHTML;
            btnVolumeEditorSave.disabled = true;
            btnVolumeEditorSave.innerHTML = '<span>Guardando...</span>';

            try {
                const srvParam = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
                const res = await fetch(`/api/volumes/write${srvParam}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        path: currentEditingFilePath,
                        content: volumeEditorTextarea.value
                    })
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al guardar archivo');
                }
                showToast('Archivo guardado correctamente en el VPS', 'success');
            } catch (err) {
                showToast(`Error al guardar: ${err.message}`, 'error');
            } finally {
                btnVolumeEditorSave.disabled = false;
                btnVolumeEditorSave.innerHTML = originalText;
            }
        });
    }

    if (btnVolumeNewFolder) {
        btnVolumeNewFolder.addEventListener('click', async () => {
            const folderName = prompt('Nombre de la nueva carpeta:');
            if (!folderName || !folderName.trim()) return;

            const cleanName = folderName.trim().replace(/[/\\]/g, '');
            const newFolderPath = `${currentVolumePath}/${cleanName}`;

            try {
                const srvParam = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
                const res = await fetch(`/api/volumes/mkdir${srvParam}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: newFolderPath })
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al crear carpeta');
                }
                showToast(`Carpeta "${cleanName}" creada`, 'success');
                fetchAndRenderVolumeFiles(currentVolumePath);
            } catch (err) {
                showToast(`Error: ${err.message}`, 'error');
            }
        });
    }

    if (volumeFileInput) {
        volumeFileInput.addEventListener('change', async (e) => {
            const file = e.target.files && e.target.files[0];
            if (!file) return;

            const formData = new FormData();
            formData.append('file', file);
            formData.append('dir', currentVolumePath);

            showToast(`Subiendo "${file.name}"...`, 'info');
            try {
                const srvParam = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
                const res = await fetch(`/api/volumes/upload${srvParam}`, {
                    method: 'POST',
                    body: formData
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al subir archivo');
                }
                showToast(`"${file.name}" subido con éxito`, 'success');
                fetchAndRenderVolumeFiles(currentVolumePath);
            } catch (err) {
                showToast(`Error al subir: ${err.message}`, 'error');
            } finally {
                volumeFileInput.value = '';
            }
        });
    }

    if (btnVolumeRefresh) {
        btnVolumeRefresh.addEventListener('click', () => {
            fetchAndRenderVolumeFiles(currentVolumePath);
        });
    }

    if (volumeSearchInput) {
        volumeSearchInput.addEventListener('input', () => {
            const query = volumeSearchInput.value.toLowerCase().trim();
            if (!query) {
                renderVolumeTableRows(currentVolumeFiles);
                return;
            }
            const filtered = currentVolumeFiles.filter(item => item.name.toLowerCase().includes(query));
            renderVolumeTableRows(filtered);
        });
    }

    if (btnDeskVolumes) {
        btnDeskVolumes.addEventListener('click', () => {
            openVolumeModal('/opt/data');
        });
    }

    if (btnCloseVolumeModal) btnCloseVolumeModal.addEventListener('click', closeVolumeModal);
    if (btnCloseVolumeModalBottom) btnCloseVolumeModalBottom.addEventListener('click', closeVolumeModal);
    if (volumeModal) {
        volumeModal.addEventListener('click', (e) => {
            if (e.target === volumeModal) closeVolumeModal();
        });
    }

    // ==========================================================================
    // --- Modal: Consola Terminal Web Interactiva & Dual Terminal ---
    // ==========================================================================
    const terminalModal = document.getElementById('terminalModal');
    const btnCloseTerminalModal = document.getElementById('btnCloseTerminalModal');
    const btnCloseTerminalModalBottom = document.getElementById('btnCloseTerminalModalBottom');
    const btnTerminalOpenPC = document.getElementById('btnTerminalOpenPC');
    const btnTerminalDirectPC = document.getElementById('btnTerminalDirectPC');
    const btnFallbackOpenPC = document.getElementById('btnFallbackOpenPC');
    const terminalHostLabel = document.getElementById('terminalHostLabel');
    const terminalTargetSelect = document.getElementById('terminalTargetSelect');
    const terminalFallbackBanner = document.getElementById('terminalFallbackBanner');
    const terminalFallbackMsg = document.getElementById('terminalFallbackMsg');
    const terminalViewport = document.getElementById('terminalViewport');
    const terminalOutput = document.getElementById('terminalOutput');
    const terminalPromptPrefix = document.getElementById('terminalPromptPrefix');
    const terminalCommandInput = document.getElementById('terminalCommandInput');
    const btnTerminalSend = document.getElementById('btnTerminalSend');
    const btnTerminalClear = document.getElementById('btnTerminalClear');
    const btnTerminalCopy = document.getElementById('btnTerminalCopy');

    let terminalActiveServer = '';
    let terminalCommandHistory = [];
    let terminalHistoryIndex = -1;
    let isTerminalExecuting = false;

    function openTerminalModal(serverName, targetContainer = '') {
        if (!terminalModal) return;
        terminalActiveServer = serverName || selectedServerName || '';
        if (terminalHostLabel) terminalHostLabel.textContent = terminalActiveServer;

        if (terminalFallbackBanner) terminalFallbackBanner.style.display = 'none';

        if (terminalTargetSelect) {
            terminalTargetSelect.innerHTML = `<option value="">🖥️ Host VPS (${escapeHtml(terminalActiveServer)})</option>`;
            
            if (swarmServicesCache && swarmServicesCache.length > 0) {
                const groupSvc = document.createElement('optgroup');
                groupSvc.label = 'Servicios Swarm / Contenedores';
                swarmServicesCache.forEach(s => {
                    const opt = document.createElement('option');
                    opt.value = s.name;
                    opt.textContent = `🐳 ${s.name}`;
                    if (targetContainer && targetContainer === s.name) opt.selected = true;
                    groupSvc.appendChild(opt);
                });
                terminalTargetSelect.appendChild(groupSvc);
            }

            if (swarmDatabasesCache && swarmDatabasesCache.length > 0) {
                const groupDb = document.createElement('optgroup');
                groupDb.label = 'Bases de Datos';
                swarmDatabasesCache.forEach(db => {
                    const opt = document.createElement('option');
                    opt.value = `tarhiata-db-${db.name}`;
                    opt.textContent = `🗄️ ${db.name} (${db.engine || 'db'})`;
                    if (targetContainer && (targetContainer === db.name || targetContainer === `tarhiata-db-${db.name}`)) opt.selected = true;
                    groupDb.appendChild(opt);
                });
                terminalTargetSelect.appendChild(groupDb);
            }
        }

        updateTerminalPrompt();

        if (terminalOutput && terminalOutput.children.length === 0) {
            appendTerminalSystemNotice(`✨ Tarhiata-Ops Web Terminal v4.2 iniciada.\nConectado a: ${terminalActiveServer}\nEscribe comandos bash directamente o usa los atajos rápidos.\n👉 Puedes usar la Terminal de tu computadora en cualquier momento con "Terminal del PC".`);
        }

        terminalModal.style.display = 'flex';
        setTimeout(() => {
            if (terminalCommandInput) terminalCommandInput.focus();
        }, 80);
    }

    function closeTerminalModal() {
        if (terminalModal) terminalModal.style.display = 'none';
    }

    function updateTerminalPrompt() {
        if (!terminalPromptPrefix) return;
        const target = terminalTargetSelect ? terminalTargetSelect.value : '';
        if (target) {
            terminalPromptPrefix.textContent = `root@${target}:#`;
            terminalPromptPrefix.style.color = '#38bdf8';
        } else {
            terminalPromptPrefix.textContent = `root@${terminalActiveServer || 'vps'}:~$`;
            terminalPromptPrefix.style.color = '#10b981';
        }
    }

    function appendTerminalSystemNotice(text) {
        if (!terminalOutput) return;
        const div = document.createElement('div');
        div.className = 'terminal-line';
        div.style.color = '#94a3b8';
        div.style.fontStyle = 'italic';
        div.textContent = text;
        terminalOutput.appendChild(div);
        scrollTerminalToBottom();
    }

    function scrollTerminalToBottom() {
        if (terminalViewport) {
            terminalViewport.scrollTop = terminalViewport.scrollHeight;
        }
    }

    async function executeTerminalCommand(cmdToRun) {
        const rawCmd = (cmdToRun !== undefined ? cmdToRun : (terminalCommandInput ? terminalCommandInput.value : '')).trim();
        if (!rawCmd || isTerminalExecuting) return;

        terminalCommandHistory.push(rawCmd);
        terminalHistoryIndex = terminalCommandHistory.length;

        if (terminalCommandInput) {
            terminalCommandInput.value = '';
        }

        if (rawCmd.toLowerCase() === 'clear') {
            if (terminalOutput) terminalOutput.innerHTML = '';
            return;
        }

        const targetContainer = terminalTargetSelect ? terminalTargetSelect.value : '';
        const promptLabel = terminalPromptPrefix ? terminalPromptPrefix.textContent : '$';

        const cmdLine = document.createElement('div');
        cmdLine.className = 'terminal-line terminal-line-cmd';
        cmdLine.innerHTML = `<span>${escapeHtml(promptLabel)}</span> <span>${escapeHtml(rawCmd)}</span>`;
        terminalOutput.appendChild(cmdLine);

        const runningIndicator = document.createElement('div');
        runningIndicator.className = 'terminal-line';
        runningIndicator.style.color = '#64748b';
        runningIndicator.textContent = '⏳ Ejecutando en servidor remoto...';
        terminalOutput.appendChild(runningIndicator);
        scrollTerminalToBottom();

        isTerminalExecuting = true;
        if (btnTerminalSend) btnTerminalSend.disabled = true;

        try {
            const res = await fetch('/api/servers/terminal', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: terminalActiveServer,
                    command: rawCmd,
                    container: targetContainer
                })
            });

            runningIndicator.remove();

            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || `Error HTTP ${res.status}`);
            }

            const data = await res.json();

            // Si falló la conexión por navegador, activar fallback a PC sin bloquearlo
            if (data.connected === false) {
                const errLine = document.createElement('div');
                errLine.className = 'terminal-line terminal-line-err';
                errLine.textContent = `❌ Fallo de conexión web: ${data.output || 'No se pudo conectar por SSH'}\n💡 Puedes conectar directamente abriendo la Terminal del PC (disponible abajo o en cabecera).`;
                terminalOutput.appendChild(errLine);

                if (terminalFallbackBanner) {
                    terminalFallbackBanner.style.display = 'flex';
                    if (terminalFallbackMsg) {
                        terminalFallbackMsg.textContent = `${data.output || 'Fallo de conexión SSH'}.`;
                    }
                }
            } else {
                if (terminalFallbackBanner) terminalFallbackBanner.style.display = 'none';

                const outLine = document.createElement('div');
                outLine.className = data.exitCode !== 0 ? 'terminal-line terminal-line-err' : 'terminal-line terminal-line-out';
                outLine.textContent = data.output || '(sin salida)';
                terminalOutput.appendChild(outLine);
            }
        } catch (err) {
            runningIndicator.remove();
            const errLine = document.createElement('div');
            errLine.className = 'terminal-line terminal-line-err';
            errLine.textContent = `❌ Error al ejecutar en navegador: ${err.message}\n💡 La opción de usar la Terminal del PC permanece totalmente disponible.`;
            terminalOutput.appendChild(errLine);

            if (terminalFallbackBanner) {
                terminalFallbackBanner.style.display = 'flex';
                if (terminalFallbackMsg) {
                    terminalFallbackMsg.textContent = `${err.message}.`;
                }
            }
        } finally {
            isTerminalExecuting = false;
            if (btnTerminalSend) btnTerminalSend.disabled = false;
            scrollTerminalToBottom();
            if (terminalCommandInput) terminalCommandInput.focus();
        }
    }

    // Disparador de terminal de PC (NUNCA BLOQUEADO)
    function triggerPCTerminal() {
        const targetServer = terminalActiveServer || selectedServerName || (activeServer && activeServer.name);
        if (!targetServer) {
            showToast('Selecciona un servidor primero.', 'error');
            return;
        }
        launchNativeTerminal(targetServer, btnTerminalOpenPC);
    }

    if (btnTerminalOpenPC) btnTerminalOpenPC.addEventListener('click', triggerPCTerminal);
    if (btnTerminalDirectPC) btnTerminalDirectPC.addEventListener('click', triggerPCTerminal);
    if (btnFallbackOpenPC) btnFallbackOpenPC.addEventListener('click', triggerPCTerminal);

    // Atajos de comandos rápidos
    document.querySelectorAll('.btn-term-quick').forEach(btn => {
        btn.addEventListener('click', () => {
            const cmd = btn.getAttribute('data-cmd');
            if (cmd) executeTerminalCommand(cmd);
        });
    });

    if (terminalCommandInput) {
        terminalCommandInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                executeTerminalCommand();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                if (terminalCommandHistory.length > 0 && terminalHistoryIndex > 0) {
                    terminalHistoryIndex--;
                    terminalCommandInput.value = terminalCommandHistory[terminalHistoryIndex];
                }
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                if (terminalHistoryIndex < terminalCommandHistory.length - 1) {
                    terminalHistoryIndex++;
                    terminalCommandInput.value = terminalCommandHistory[terminalHistoryIndex];
                } else {
                    terminalHistoryIndex = terminalCommandHistory.length;
                    terminalCommandInput.value = '';
                }
            } else if (e.ctrlKey && e.key === 'l') {
                e.preventDefault();
                if (terminalOutput) terminalOutput.innerHTML = '';
            }
        });
    }

    if (btnTerminalSend) {
        btnTerminalSend.addEventListener('click', () => executeTerminalCommand());
    }

    if (btnTerminalClear) {
        btnTerminalClear.addEventListener('click', () => {
            if (terminalOutput) terminalOutput.innerHTML = '';
            if (terminalCommandInput) terminalCommandInput.focus();
        });
    }

    if (btnTerminalCopy) {
        btnTerminalCopy.addEventListener('click', async () => {
            if (!terminalOutput) return;
            const text = terminalOutput.innerText || terminalOutput.textContent || '';
            const ok = await copyToClipboard(text);
            if (ok) {
                showToast('Salida de la terminal copiada al portapapeles', 'success');
            } else {
                showToast('No se pudo copiar el texto', 'error');
            }
        });
    }

    if (terminalTargetSelect) {
        terminalTargetSelect.addEventListener('change', () => {
            updateTerminalPrompt();
            if (terminalCommandInput) terminalCommandInput.focus();
        });
    }

    if (btnCloseTerminalModal) btnCloseTerminalModal.addEventListener('click', closeTerminalModal);
    if (btnCloseTerminalModalBottom) btnCloseTerminalModalBottom.addEventListener('click', closeTerminalModal);
    if (terminalModal) {
        terminalModal.addEventListener('click', (e) => {
            if (e.target === terminalModal) closeTerminalModal();
        });
    }

    // --- Load Initial State ---
    async function loadHubState() {
        try {
            const res = await fetch('/api/status');
            if (!res.ok) {
                topActiveName.textContent = 'Error de API';
                showToast(`Error al consultar estado (${res.status})`, 'error');
                deactivateInitialSkeletons();
                return;
            }

            const data = await res.json();
            servers = data.servers || [];
            activeServer = data.config || null;

            if (servers.length === 0 && activeServer && activeServer.host) {
                servers = [activeServer];
            }

            topActiveName.textContent = 'Ninguno';
            topActiveHost.textContent = '—';
            topActiveDot.className = 'status-dot status-offline';
            topActiveLatency.textContent = '—';

            if (activeServer && activeServer.name) {
                topActiveName.textContent = activeServer.name;
                topActiveHost.textContent = activeServer.host || 'localhost';
                topActiveDot.className = data.isOnline ? 'status-dot status-online' : 'status-dot status-offline';
                topActiveLatency.textContent = data.isOnline ? 'ONLINE' : 'OFFLINE';
            }

            const savedServer = localStorage.getItem('tarhiata_last_server');
            if (savedServer && servers.some(s => s.name === savedServer)) {
                selectedServerName = savedServer;
            } else if (!selectedServerName || !servers.some(s => s.name === selectedServerName)) {
                if (activeServer && activeServer.name) {
                    selectedServerName = activeServer.name;
                } else if (servers.length > 0) {
                    selectedServerName = servers[0].name;
                }
            }

            renderFleetDirectory();

            // Restaurar pestaña activa desde URL Hash (#apps, #dbs, #services, #devices, #topology)
            const initialTab = (location.hash || '').replace('#', '');
            if (['services', 'databases', 'host', 'devices', 'topology'].includes(initialTab)) {
                activateTab(initialTab);
            }

            if (selectedServerName) {
                await selectServer(selectedServerName);
            } else if (servers.length === 0) {
                deskServerTitle.textContent = 'Sin Servidores';
                deskHost.textContent = '—';
                deskModeBadge.textContent = 'NINGUNO';
                deskActiveBadge.style.display = 'none';
                btnDeskActivate.style.display = 'none';
                tileCpuPct.textContent = '—';
                tileRamUsed.textContent = '—';
                tileDiskUsed.textContent = '—';
                tileDiskBar.style.width = '0%';
                tileUptime.textContent = '—';
                swarmStateBadge.className = 'swarm-status-tag';
                swarmStateBadge.textContent = 'Sin Framework';
                deskSwarm.textContent = '—';
                deskDocker.textContent = '—';
                deskOS.textContent = '—';
                btnBootstrapSwarm.style.display = 'inline-flex';
                btnBootstrapSwarm.disabled = false;
                btnBootstrapSwarm.classList.remove('swarm-configured');
                btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
                btnBootstrapSwarm.title = 'Añade un servidor VPS para instalar el Framework de orquestación';
                swarmServicesCardsGrid.innerHTML = '';
                swarmServicesEmpty.style.display = 'flex';
                swarmDatabasesCardsGrid.innerHTML = '';
                swarmDatabasesEmpty.style.display = 'flex';
                if (swarmNodesTableBody) swarmNodesTableBody.innerHTML = '<tr><td colspan="5" class="t-td-empty">Sin servidores registrados. Haz clic en "Instalar Framework" para comenzar.</td></tr>';
                if (servicesTableBody) servicesTableBody.innerHTML = '<tr><td colspan="4" class="t-td-empty">Sin servidores registrados</td></tr>';
            }

            // Deactivate all component skeletons at once when the very last data has arrived & rendered
            deactivateInitialSkeletons();

            if (servers.length > 0) {
                testAllServers(false);
            }
        } catch (err) {
            console.error('Error inicializando Hub:', err);
            showToast(`Fallo de conexión: ${err.message}`, 'error');
            deactivateInitialSkeletons();
        }
    }

    // --- Toast Notifications ---
    function showToast(message, type = 'info') {
        const container = document.getElementById('toastContainer');
        if (!container) return;
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;

        const icon = type === 'success' ? '✓' : type === 'error' ? '✕' : 'ℹ';
        toast.innerHTML = `<span style="font-weight:700;">${icon}</span> <span>${escapeHtml(message)}</span>`;

        container.appendChild(toast);
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateY(10px)';
            toast.style.transition = 'all 0.25s ease';
            setTimeout(() => toast.remove(), 250);
        }, 3500);
    }

    function escapeHtml(str) {
        if (!str) return '';
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    // ==========================================================================
    // --- Modal: Certificados SSL & Mantenimiento Traefik ---
    // ==========================================================================
    const sslModal = document.getElementById('sslModal');
    const btnCloseSSLModal = document.getElementById('btnCloseSSLModal');
    const btnCloseSSLModalBottom = document.getElementById('btnCloseSSLModalBottom');
    const btnTopSSL = document.getElementById('btnTopSSL');
    const btnRefreshSSL = document.getElementById('btnRefreshSSL');
    const btnReloadTraefik = document.getElementById('btnReloadTraefik');
    const sslSearchInput = document.getElementById('sslSearchInput');
    const sslTableBody = document.getElementById('sslTableBody');
    const sslEmptyNotice = document.getElementById('sslEmptyNotice');

    const activeMaintenanceServices = new Set();
    let sslItemsCache = [];

    async function openSSLModal() {
        if (!sslModal) return;
        sslModal.style.display = 'flex';
        if (sslSearchInput) sslSearchInput.value = '';
        await fetchAndRenderSSL();
    }

    async function fetchAndRenderSSL() {
        if (!sslTableBody) return;
        sslTableBody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding:24px; color:var(--text-muted);">Inspeccionando certificados TLS en puerto 443...</td></tr>`;
        if (sslEmptyNotice) sslEmptyNotice.style.display = 'none';

        try {
            const serverQuery = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
            const res = await fetch(`/api/ssl/inspect${serverQuery}`);
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || 'Error al inspeccionar SSL');
            }
            const data = await res.json();
            sslItemsCache = Array.isArray(data) ? data : [];
            renderSSLTable(sslItemsCache);
        } catch (err) {
            sslTableBody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding:24px; color:var(--status-offline);">Error: ${escapeHtml(err.message)}</td></tr>`;
            showToast(err.message, 'error');
        }
    }

    function renderSSLTable(items) {
        if (!sslTableBody) return;
        sslTableBody.innerHTML = '';

        const query = sslSearchInput ? sslSearchInput.value.toLowerCase().trim() : '';
        const filtered = items.filter(item => {
            if (!query) return true;
            return (item.domain && item.domain.toLowerCase().includes(query)) ||
                   (item.serviceName && item.serviceName.toLowerCase().includes(query));
        });

        if (filtered.length === 0) {
            if (sslEmptyNotice) {
                sslEmptyNotice.style.display = 'block';
                sslEmptyNotice.textContent = items.length === 0 ? 'No hay servicios públicos con dominio configurado en este servidor.' : 'Ningún dominio coincide con la búsqueda.';
            }
            return;
        }

        if (sslEmptyNotice) sslEmptyNotice.style.display = 'none';

        filtered.forEach(item => {
            const tr = document.createElement('tr');

            let badgeHtml = '';
            if (item.status === 'active') {
                badgeHtml = `<span class="ssl-badge ssl-badge-active">🔒 Válido (${item.daysRemaining} días)</span>`;
            } else if (item.status === 'expiring_soon') {
                badgeHtml = `<span class="ssl-badge ssl-badge-expiring">⚠️ Expira pronto (${item.daysRemaining} días)</span>`;
            } else if (item.status === 'expired') {
                badgeHtml = `<span class="ssl-badge ssl-badge-expired">❌ Expirado</span>`;
            } else {
                badgeHtml = `<span class="ssl-badge ssl-badge-http">🌐 Solo HTTP</span>`;
            }

            const isMaint = activeMaintenanceServices.has(item.serviceName);

            tr.innerHTML = `
                <td>
                    <a href="https://${escapeHtml(item.domain)}" target="_blank" style="color:var(--brand-primary); font-weight:600; text-decoration:none;">
                        https://${escapeHtml(item.domain)} ↗
                    </a>
                </td>
                <td><code style="font-family:var(--font-mono); font-size:0.75rem;">${escapeHtml(item.serviceName)}</code></td>
                <td>${badgeHtml}</td>
                <td style="color:var(--text-secondary); font-size:0.75rem;">${escapeHtml(item.issuer || 'Let\'s Encrypt / ACME')}</td>
                <td style="color:var(--text-muted); font-size:0.75rem;">${escapeHtml(item.expiryDate || 'N/A')}</td>
                <td>
                    <button type="button" class="mini-btn btn-modal-maint ${isMaint ? 'maint-active' : ''}" data-svc="${escapeHtml(item.serviceName)}" title="Alternar modo 503 HTTP Drain">
                        ${isMaint ? '🚧 Mantenimiento ON' : '🚧 Modo Mantenimiento'}
                    </button>
                </td>
            `;
            sslTableBody.appendChild(tr);
        });

        // Eventos de botones de mantenimiento en el modal
        sslTableBody.querySelectorAll('.btn-modal-maint').forEach(btn => {
            btn.addEventListener('click', () => {
                const svcName = btn.getAttribute('data-svc');
                if (svcName) {
                    const currentlyActive = activeMaintenanceServices.has(svcName);
                    toggleMaintenanceMode(svcName, !currentlyActive, btn);
                }
            });
        });
    }

    async function toggleMaintenanceMode(serviceName, enable, triggerBtn) {
        if (!serviceName) return;
        const origText = triggerBtn ? triggerBtn.textContent : '';
        if (triggerBtn) {
            triggerBtn.disabled = true;
            triggerBtn.textContent = enable ? 'Activando...' : 'Desactivando...';
        }

        try {
            const serverQuery = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
            const res = await fetch(`/api/maintenance/toggle${serverQuery}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ serviceName, enable })
            });

            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || 'Error al cambiar modo mantenimiento');
            }

            if (enable) {
                activeMaintenanceServices.add(serviceName);
                showToast(`Servicio '${serviceName}' ahora devuelve 503 Maintenance`, 'info');
            } else {
                activeMaintenanceServices.delete(serviceName);
                showToast(`Modo mantenimiento desactivado para '${serviceName}'`, 'success');
            }

            // Actualizar botones en modal y en tarjetas
            document.querySelectorAll(`.btn-maint-svc[data-name="${CSS.escape(serviceName)}"], .btn-modal-maint[data-svc="${CSS.escape(serviceName)}"]`).forEach(b => {
                if (enable) {
                    b.classList.add('maint-active');
                    b.textContent = '🚧 Mantenimiento ON';
                } else {
                    b.classList.remove('maint-active');
                    b.textContent = b.classList.contains('btn-maint-svc') ? '🚧 Mantenimiento' : '🚧 Modo Mantenimiento';
                }
            });
        } catch (err) {
            showToast(err.message, 'error');
            if (triggerBtn) triggerBtn.textContent = origText;
        } finally {
            if (triggerBtn) triggerBtn.disabled = false;
        }
    }

    async function reloadTraefikProxy(btn) {
        if (btn) {
            btn.disabled = true;
            btn.textContent = '⏳ Recargando...';
        }
        try {
            const serverQuery = selectedServerName ? `?server=${encodeURIComponent(selectedServerName)}` : '';
            const res = await fetch(`/api/ssl/reload${serverQuery}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            if (!res.ok) {
                const errText = await res.text();
                throw new Error(errText || 'Error al recargar Traefik');
            }
            showToast('Traefik recargado y certificados revalidados', 'success');
            await fetchAndRenderSSL();
        } catch (err) {
            showToast(err.message, 'error');
        } finally {
            if (btn) {
                btn.disabled = false;
                btn.textContent = '🚀 Recargar Traefik';
            }
        }
    }

    if (btnTopSSL) btnTopSSL.addEventListener('click', openSSLModal);
    if (btnRefreshSSL) btnRefreshSSL.addEventListener('click', fetchAndRenderSSL);
    if (btnReloadTraefik) btnReloadTraefik.addEventListener('click', () => reloadTraefikProxy(btnReloadTraefik));
    if (btnCloseSSLModal) btnCloseSSLModal.addEventListener('click', () => { sslModal.style.display = 'none'; });
    if (btnCloseSSLModalBottom) btnCloseSSLModalBottom.addEventListener('click', () => { sslModal.style.display = 'none'; });
    if (sslModal) sslModal.addEventListener('click', (e) => {
        if (e.target === sslModal) sslModal.style.display = 'none';
    });
    if (sslSearchInput) sslSearchInput.addEventListener('input', debounce(() => renderSSLTable(sslItemsCache), 150));

    // Inicializar Studio
    loadHubState();

    // Auto-refresh periódico de telemetría (cada 15s) cuando la pestaña está activa
    let isAutoRefreshing = false;
    setInterval(async () => {
        if (document.hidden || isAutoRefreshing) return;
        if (selectedServerName && !btnDeskRefresh.disabled) {
            isAutoRefreshing = true;
            try {
                await refreshServerTelemetry(selectedServerName, true);
            } catch (pollErr) {
                console.debug('Fallo durante polling automático de telemetría:', pollErr);
            } finally {
                isAutoRefreshing = false;
            }
        }
    }, 15000);

    // Reanudación inmediata al regresar a la pestaña activa
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden && selectedServerName && !isAutoRefreshing) {
            refreshServerTelemetry(selectedServerName, true);
        }
    });
});
