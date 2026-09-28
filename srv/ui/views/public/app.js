// Tarhiata-Ops Web Studio Single-Page Application Orchestrator
import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { openModal, closeModal, setupModalDismissals } from '/pkg/modal/modal.js';
import { setupSecurityInterceptor, apiFetch } from '/pkg/apiclient/api.js';

import { loadHubState, selectServer, setupFleetListeners } from '/components/fleet/fleet.js';
import { refreshServerTelemetry } from '/components/telemetry/telemetry.js';
import { loadSwarmStatus, renderAppCards, renderServicesTable, setupServicesListeners } from '/components/services/services.js';
import { setupDatabasesListeners } from '/components/databases/databases.js';
import { renderNodesTable, setupNodesListeners } from '/components/nodes/nodes.js';
import { loadServiceLinks, renderTopologyServicesTable, setupTopologyListeners } from '/components/topology/topology.js';
import { loadHostDevices, renderHostDevices } from '/components/hardware/hardware.js';
import { setupSpotlightListeners, openSpotlight, downloadSystemReport } from '/components/spotlight/spotlight.js';
import { openTerminalModal, closeTerminalModal, executeTerminalCommand, launchNativeTerminal, setupTerminalListeners } from '/components/terminal/terminal.js';
import { openEnvModal, closeEnvModal, setupEnvListeners } from '/components/env/env.js';
import { openVolumeModal, closeVolumeModal, setupVolumeListeners } from '/components/volumes/volumes.js';
import { openLogsModal, closeLogsModal, setupLogsListeners } from '/components/logs/logs.js';
import { openSSLModal, closeSSLModal, setupSSLListeners } from '/components/ssl/ssl.js';

// Expose critical handlers to window for inline events and cross-module convenience
window.state = state;
window.showToast = showToast;
window.openModal = openModal;
window.closeModal = closeModal;
window.selectServer = selectServer;
window.loadSwarmStatus = loadSwarmStatus;
window.loadHostDevices = loadHostDevices;
window.openTerminalModal = openTerminalModal;
window.closeTerminalModal = closeTerminalModal;
window.executeTerminalCommand = executeTerminalCommand;
window.launchNativeTerminal = launchNativeTerminal;
window.openEnvModal = openEnvModal;
window.closeEnvModal = closeEnvModal;
window.openVolumeModal = openVolumeModal;
window.closeVolumeModal = closeVolumeModal;
window.openLogsModal = openLogsModal;
window.closeLogsModal = closeLogsModal;
window.openSSLModal = openSSLModal;
window.closeSSLModal = closeSSLModal;
window.openSpotlight = openSpotlight;
window.downloadSystemReport = downloadSystemReport;

// Tab Switching Navigation
export function activateTab(tabName) {
    const tabSwarmServices = document.getElementById('tabSwarmServices');
    const tabSwarmDatabases = document.getElementById('tabSwarmDatabases');
    const tabHostServices = document.getElementById('tabHostServices');
    const tabHostDevices = document.getElementById('tabHostDevices');
    const tabSwarmTopology = document.getElementById('tabSwarmTopology');

    const viewSwarmServices = document.getElementById('viewSwarmServices');
    const viewSwarmDatabases = document.getElementById('viewSwarmDatabases');
    const viewHostServices = document.getElementById('viewHostServices');
    const viewHostDevices = document.getElementById('viewHostDevices');
    const viewSwarmTopology = document.getElementById('viewSwarmTopology');

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
        renderAppCards(state.swarmServicesCache);
        renderNodesTable(state.swarmNodesCache);
        renderTopologyServicesTable(state.swarmServicesCache, state.swarmDatabasesCache, state.currentServiceLinks);
        if (state.selectedServerName) loadServiceLinks();
    }
    if (effectiveTab === 'databases' && tabSwarmDatabases && viewSwarmDatabases) {
        tabSwarmDatabases.classList.add('active');
        viewSwarmDatabases.style.display = 'block';
    }
    if (effectiveTab === 'host' && tabHostServices && viewHostServices) {
        tabHostServices.classList.add('active');
        viewHostServices.style.display = 'block';
        renderServicesTable(state.currentHostServices);
    }
    if (effectiveTab === 'devices' && tabHostDevices && viewHostDevices) {
        tabHostDevices.classList.add('active');
        viewHostDevices.style.display = 'block';
        if (state.currentHostDevices) {
            renderHostDevices(state.currentHostDevices);
        } else {
            loadHostDevices();
        }
    }
}
window.activateTab = activateTab;

function initApp() {
    // 1. Interceptors & Dismissals
    setupSecurityInterceptor();
    setupModalDismissals();

    // 2. Setup Modules Listeners
    setupFleetListeners();
    setupServicesListeners();
    setupDatabasesListeners();
    setupNodesListeners();
    setupTopologyListeners();
    setupTerminalListeners();
    setupSpotlightListeners();
    setupEnvListeners(async () => {
        if (state.selectedServerName) await loadSwarmStatus(state.selectedServerName);
    });
    setupVolumeListeners();
    setupLogsListeners();
    setupSSLListeners();

    // 3. Tab listeners
    const tabSwarmServices = document.getElementById('tabSwarmServices');
    const tabSwarmDatabases = document.getElementById('tabSwarmDatabases');
    const tabHostServices = document.getElementById('tabHostServices');
    const tabHostDevices = document.getElementById('tabHostDevices');
    const tabSwarmTopology = document.getElementById('tabSwarmTopology');
    const btnRefreshDevices = document.getElementById('btnRefreshDevices');
    const btnNavSpotlight = document.getElementById('btnNavSpotlight');
    const btnDownloadReport = document.getElementById('btnDownloadReport');

    if (tabSwarmServices) tabSwarmServices.addEventListener('click', () => activateTab('services'));
    if (tabSwarmDatabases) tabSwarmDatabases.addEventListener('click', () => activateTab('databases'));
    if (tabHostServices) tabHostServices.addEventListener('click', () => activateTab('host'));
    if (tabHostDevices) tabHostDevices.addEventListener('click', () => activateTab('devices'));
    if (tabSwarmTopology) tabSwarmTopology.addEventListener('click', () => activateTab('services'));
    if (btnRefreshDevices) btnRefreshDevices.addEventListener('click', () => loadHostDevices(true));
    if (btnNavSpotlight) btnNavSpotlight.addEventListener('click', () => openSpotlight());
    if (btnDownloadReport) btnDownloadReport.addEventListener('click', () => downloadSystemReport());

    // 4. Initial Load
    const initialHash = (location.hash || '#services').replace('#', '');
    activateTab(initialHash || 'services');
    loadHubState();

    // 5. Periodic auto-refresh
    let isAutoRefreshing = false;
    setInterval(async () => {
        if (document.hidden || isAutoRefreshing) return;
        const btnDeskRefresh = document.getElementById('btnDeskRefresh');
        const viewHostDevices = document.getElementById('viewHostDevices');
        if (state.selectedServerName && (!btnDeskRefresh || !btnDeskRefresh.disabled)) {
            isAutoRefreshing = true;
            try {
                await refreshServerTelemetry(state.selectedServerName, true);
                if (location.hash === '#devices' || (viewHostDevices && viewHostDevices.style.display !== 'none')) {
                    await loadHostDevices();
                }
            } catch (pollErr) {
                console.debug('Polling error:', pollErr);
            } finally {
                isAutoRefreshing = false;
            }
        }
    }, 15000);

    // 6. Immediate refresh when tab becomes visible
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden && state.selectedServerName && !isAutoRefreshing) {
            refreshServerTelemetry(state.selectedServerName, true);
        }
    });
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
} else {
    initApp();
}
