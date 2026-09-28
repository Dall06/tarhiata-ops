import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';

let isLoadingDevices = false;

export async function loadHostDevices(forceFresh = false) {
    const serverName = state.selectedServerName;
    if (!serverName || isLoadingDevices) return;
    isLoadingDevices = true;

    const btnRefreshDevices = document.getElementById('btnRefreshDevices');
    const hwStorageTableBody = document.getElementById('hwStorageTableBody');
    const hwGpuCardsContainer = document.getElementById('hwGpuCardsContainer');
    const hwUsbTableBody = document.getElementById('hwUsbTableBody');
    const hwDisplaysContainer = document.getElementById('hwDisplaysContainer');
    const hwPciTableBody = document.getElementById('hwPciTableBody');

    if (btnRefreshDevices) btnRefreshDevices.disabled = true;

    if (!state.currentHostDevices || forceFresh) {
        if (hwStorageTableBody) hwStorageTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">Consultando unidades de disco...</td></tr>`;
        if (hwGpuCardsContainer) hwGpuCardsContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">Consultando tarjetas gráficas...</div>`;
        if (hwUsbTableBody) hwUsbTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Consultando periféricos USB...</td></tr>`;
        if (hwDisplaysContainer) hwDisplaysContainer.innerHTML = `<div class="t-td-empty" style="padding: 24px; text-align: center; width: 100%;">Consultando salidas de video...</div>`;
        if (hwPciTableBody) hwPciTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Consultando controladores PCI...</td></tr>`;
    }

    try {
        const freshParam = forceFresh ? '&fresh=true' : '';
        const res = await apiFetch(`/api/host/devices?server=${encodeURIComponent(serverName)}${freshParam}`);
        if (!res.ok) {
            const errText = await res.text();
            showToast(`Error al obtener dispositivos: ${errText}`, 'error');
            return;
        }
        const data = await res.json();
        state.currentHostDevices = data;
        renderHostDevices(data);
    } catch (err) {
        showToast(`Fallo de conexión al inspeccionar dispositivos: ${err.message}`, 'error');
    } finally {
        isLoadingDevices = false;
        if (btnRefreshDevices) btnRefreshDevices.disabled = false;
    }
}

export function renderHostDevices(data) {
    if (!data) return;

    const storage = data.storage || [];
    const gpus = data.gpus || [];
    const usb = data.usb || [];
    const displays = data.displays || [];
    const pci = data.pci || [];

    const totalDevices = storage.length + gpus.length + usb.length + displays.length + pci.length;

    const tabDevicesCount = document.getElementById('tabDevicesCount');
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
