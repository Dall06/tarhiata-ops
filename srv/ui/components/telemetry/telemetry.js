/**
 * Tarhiata Cloud Studio — Telemetry & Swarm Component (opt/telemetry)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { getGaugeColor, formatDockerVersion } from '/pkg/jsutil/utils.js';
import { apiFetch } from '/pkg/apiclient/api.js';

let telemetryRequestId = 0;

export function deactivateInitialSkeletons() {
    document.body.classList.remove('is-initial-loading');
    document.querySelectorAll('.is-loading-skeleton').forEach(el => {
        el.classList.remove('is-loading-skeleton');
    });
}

export function activateServerLoadingSkeletons(server) {
    const serverNameVal = document.getElementById('serverNameVal');
    const heroServerIP = document.getElementById('heroServerIP');
    const heroServerStatus = document.getElementById('heroServerStatus');
    const deskOS = document.getElementById('deskOS');
    const deskDocker = document.getElementById('deskDocker');
    const deskSwarm = document.getElementById('deskSwarm');
    const tileUptime = document.getElementById('tileUptime');

    if (serverNameVal && server) serverNameVal.textContent = server.name;
    if (heroServerIP && server) heroServerIP.textContent = server.host;
    if (heroServerStatus) {
        heroServerStatus.className = 'status-dot status-pending';
        heroServerStatus.title = 'Conectando...';
    }
    if (deskOS) deskOS.textContent = 'Consultando...';
    if (deskDocker) deskDocker.textContent = 'Verificando...';
    if (deskSwarm) deskSwarm.textContent = 'Sincronizando...';
    if (tileUptime) tileUptime.textContent = 'Calculando...';
}

export async function refreshServerTelemetry(serverName, isSilent = false, onProcessSwarm = null, onRenderHostServices = null) {
    if (!serverName) return;
    const currentReq = ++telemetryRequestId;

    const btnDeskRefresh = document.getElementById('btnDeskRefresh');
    const deskStatusDot = document.getElementById('deskStatusDot');
    const tileUptime = document.getElementById('tileUptime');
    const tileLatencyDisplay = document.getElementById('tileLatencyDisplay');
    const topActiveName = document.getElementById('topActiveName');
    const topActiveDot = document.getElementById('topActiveDot');
    const topActiveLatency = document.getElementById('topActiveLatency');
    const deskOS = document.getElementById('deskOS');
    const tileDistroTag = document.getElementById('tileDistroTag');
    const tileHostTag = document.getElementById('tileHostTag');
    const deskDocker = document.getElementById('deskDocker');
    const tileCpuCores = document.getElementById('tileCpuCores');
    const tileCpuPct = document.getElementById('tileCpuPct');
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
    const btnBootstrapSwarm = document.getElementById('btnBootstrapSwarm');
    const tabHostCount = document.getElementById('tabHostCount');

    if (!isSilent && btnDeskRefresh) {
        btnDeskRefresh.disabled = true;
    }
    if (!isSilent && deskStatusDot) {
        deskStatusDot.className = 'ops-status-dot status-pending';
    }
    const tStart = performance.now();

    try {
        const freshParam = !isSilent ? '&fresh=true' : '';
        const [hostRes, swarmRes] = await Promise.all([
            fetch(`/api/host/inspect?server=${encodeURIComponent(serverName)}${freshParam}`),
            fetch(`/api/swarm/status?server=${encodeURIComponent(serverName)}${freshParam}`)
        ]);
        const latency = Math.round(performance.now() - tStart);

        if (currentReq !== telemetryRequestId || serverName !== state.selectedServerName) return;

        if (!hostRes.ok) {
            const errText = await hostRes.text();
            if (deskStatusDot) deskStatusDot.className = 'ops-status-dot status-offline';
            if (tileUptime) tileUptime.textContent = 'Sin conexión';
            if (tileLatencyDisplay) tileLatencyDisplay.textContent = 'Latencia: —';
            if (topActiveName && topActiveName.textContent === serverName) {
                if (topActiveDot) topActiveDot.className = 'status-dot status-offline';
                if (topActiveLatency) topActiveLatency.textContent = 'OFFLINE';
            }
            const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
            if (dotFleet) dotFleet.className = 'status-dot status-offline';
            if (!isSilent) showToast(`Fallo al sondear host '${serverName}': ${errText}`, 'error');
            if (onProcessSwarm) {
                onProcessSwarm(serverName, { active: false }, isSilent);
            }
            if (btnBootstrapSwarm) {
                btnBootstrapSwarm.style.display = 'inline-flex';
                btnBootstrapSwarm.disabled = false;
                btnBootstrapSwarm.classList.remove('swarm-configured');
                btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
                btnBootstrapSwarm.title = `Instalar Framework de orquestación en '${serverName}'`;
            }
            return;
        }

        const data = await hostRes.json();
        if (currentReq !== telemetryRequestId || serverName !== state.selectedServerName) return;

        state.selectedInspection = data;
        if (deskStatusDot) deskStatusDot.className = 'ops-status-dot status-online';
        if (tileLatencyDisplay) tileLatencyDisplay.textContent = `Latencia: ${latency} ms`;

        if (topActiveName && topActiveName.textContent === serverName) {
            if (topActiveDot) topActiveDot.className = 'status-dot status-online';
            if (topActiveLatency) topActiveLatency.textContent = `${latency} ms`;
        }
        const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
        if (dotFleet) dotFleet.className = 'status-dot status-online';

        const m = data.metrics || {};
        if (deskOS) deskOS.textContent = m.os || 'Linux';
        if (tileDistroTag) tileDistroTag.textContent = m.os || 'Linux';
        if (tileHostTag) tileHostTag.textContent = m.hostname || data.host;
        if (tileUptime) tileUptime.textContent = m.uptime || 'En línea';

        const dockerSvc = (data.services || []).find(sv => sv.name && sv.name.toLowerCase().includes('docker'));
        if (dockerSvc && deskDocker) {
            deskDocker.textContent = dockerSvc.subState === 'running' ? 'Activo' : (dockerSvc.activeState || 'Inactivo');
        }

        // CPU Gauge
        if (tileCpuCores) tileCpuCores.textContent = `${m.cpuCores || 1} Cores`;
        const cpuVal = (m.cpuPercent || 0).toFixed(1);
        if (tileCpuPct) tileCpuPct.textContent = `${cpuVal}%`;
        if (tileCpuBar) {
            tileCpuBar.style.width = `${Math.min(100, Math.max(0, m.cpuPercent || 0))}%`;
            tileCpuBar.style.backgroundColor = getGaugeColor(m.cpuPercent || 0);
        }
        if (tileCpuLoad) tileCpuLoad.textContent = `Carga: ${m.loadAvg || '—'}`;

        // RAM Gauge
        if (tileRamUsed) tileRamUsed.textContent = `${m.memoryUsedMb || 0} MB`;
        if (tileRamTotal) tileRamTotal.textContent = `Total: ${m.memoryTotalMb || 0} MB`;
        const ramVal = (m.memoryPercent || 0).toFixed(1);
        if (tileRamPct) tileRamPct.textContent = `${ramVal}%`;
        if (tileRamBar) {
            tileRamBar.style.width = `${Math.min(100, Math.max(0, m.memoryPercent || 0))}%`;
            tileRamBar.style.backgroundColor = getGaugeColor(m.memoryPercent || 0);
        }

        // Disk Gauge
        if (tileDiskUsed) tileDiskUsed.textContent = `${(m.diskUsedGb || 0).toFixed(1)} GB`;
        if (tileDiskTotal) tileDiskTotal.textContent = `Total: ${(m.diskTotalGb || 0).toFixed(1)} GB`;
        const diskVal = (m.diskPercent || 0).toFixed(1);
        if (tileDiskPct) tileDiskPct.textContent = `${diskVal}%`;
        if (tileDiskBar) {
            tileDiskBar.style.width = `${Math.min(100, Math.max(0, m.diskPercent || 0))}%`;
            tileDiskBar.style.backgroundColor = getGaugeColor(m.diskPercent || 0);
        }

        // Host Services list
        const newHostServices = data.services || [];
        if (!isSilent || state.currentHostServices.length !== newHostServices.length) {
            state.currentHostServices = newHostServices;
            if (tabHostCount) tabHostCount.textContent = state.currentHostServices.length;
            if (onRenderHostServices) onRenderHostServices(state.currentHostServices);
        }

        // Procesar estado de Swarm
        if (swarmRes.ok) {
            const swarmData = await swarmRes.json();
            if (currentReq === telemetryRequestId && serverName === state.selectedServerName && onProcessSwarm) {
                onProcessSwarm(serverName, swarmData, isSilent);
            }
        } else {
            if (currentReq === telemetryRequestId && serverName === state.selectedServerName && onProcessSwarm) {
                onProcessSwarm(serverName, { active: false }, isSilent);
            }
        }

    } catch (err) {
        if (currentReq === telemetryRequestId && serverName === state.selectedServerName && onProcessSwarm) {
            onProcessSwarm(serverName, { active: false }, isSilent);
        }
        if (deskStatusDot) deskStatusDot.className = 'ops-status-dot status-offline';
        if (tileLatencyDisplay) tileLatencyDisplay.textContent = 'Latencia: —';
        if (topActiveName && topActiveName.textContent === serverName) {
            if (topActiveDot) topActiveDot.className = 'status-dot status-offline';
            if (topActiveLatency) topActiveLatency.textContent = 'OFFLINE';
        }
        const dotFleet = document.getElementById(`fleet-dot-${serverName}`);
        if (dotFleet) dotFleet.className = 'status-dot status-offline';
        if (btnBootstrapSwarm) {
            btnBootstrapSwarm.style.display = 'inline-flex';
            btnBootstrapSwarm.disabled = false;
            btnBootstrapSwarm.classList.remove('swarm-configured');
            btnBootstrapSwarm.innerHTML = `🚀 <span>Instalar Framework</span>`;
            btnBootstrapSwarm.title = `Instalar Framework de orquestación en '${serverName}'`;
        }
    } finally {
        if (!isSilent && btnDeskRefresh) btnDeskRefresh.disabled = false;
        deactivateInitialSkeletons();
    }
}
