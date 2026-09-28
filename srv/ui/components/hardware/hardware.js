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
        const [resDev, resSec] = await Promise.all([
            apiFetch(`/api/host/devices?server=${encodeURIComponent(serverName)}${freshParam}`),
            apiFetch(`/api/host/security?server=${encodeURIComponent(serverName)}${freshParam}`)
        ]);

        if (resDev.ok && resDev.data) {
            state.currentHostDevices = resDev.data;
            renderHostDevices(resDev.data);
        } else {
            showToast(`Error al obtener dispositivos: ${resDev.error || resDev.status}`, 'error');
        }

        if (resSec.ok && resSec.data) {
            renderSecurityReport(resSec.data);
        }
    } catch (err) {
        showToast(`Fallo de conexión al inspeccionar host: ${err.message}`, 'error');
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

export function renderSecurityReport(sec) {
    if (!sec) return;
    const secUfwBadge = document.getElementById('secUfwBadge');
    const secF2bBadge = document.getElementById('secF2bBadge');
    const secUfwTableBody = document.getElementById('secUfwTableBody');
    const secF2bStatusPill = document.getElementById('secF2bStatusPill');
    const secF2bBody = document.getElementById('secF2bBody');

    // 1. UFW Rules
    if (secUfwBadge) {
        secUfwBadge.textContent = sec.ufwActive ? 'UFW: ACTIVO' : 'UFW: INACTIVO';
        secUfwBadge.style.color = sec.ufwActive ? 'var(--status-online)' : 'var(--text-muted)';
    }

    if (secUfwTableBody) {
        const rules = sec.ufwRules || [];
        if (rules.length === 0) {
            secUfwTableBody.innerHTML = `<tr><td colspan="5" class="t-td-empty">${sec.ufwActive ? 'Sin reglas de puertos configuradas' : 'Cortafuegos UFW inactivo o sin reglas'}</td></tr>`;
        } else {
            secUfwTableBody.innerHTML = rules.map(r => {
                const isAllow = (r.action || '').toUpperCase().includes('ALLOW');
                const actionBadge = isAllow
                    ? `<span class="t-badge" style="background:rgba(16,185,129,0.15); color:var(--status-online); border-color:rgba(16,185,129,0.3);">${escapeHtml(r.action)}</span>`
                    : `<span class="t-badge" style="background:rgba(244,63,94,0.15); color:var(--status-offline); border-color:rgba(244,63,94,0.3);">${escapeHtml(r.action)}</span>`;
                return `
                    <tr>
                        <td><code>${escapeHtml(r.number || '—')}</code></td>
                        <td><strong style="color:var(--text-pure);">${escapeHtml(r.to || '—')}</strong></td>
                        <td>${actionBadge}</td>
                        <td><code>${escapeHtml(r.from || 'Anywhere')}</code></td>
                        <td><span class="t-badge">${escapeHtml(r.proto || 'any')}</span></td>
                    </tr>
                `;
            }).join('');
        }
    }

    // 2. Fail2Ban
    const f2b = sec.fail2ban || {};
    if (secF2bBadge) {
        secF2bBadge.textContent = f2b.active ? `Fail2Ban: ACTIVO (${(f2b.jails || []).length} jaulas)` : 'Fail2Ban: INACTIVO';
        secF2bBadge.style.color = f2b.active ? 'var(--status-online)' : 'var(--text-muted)';
    }

    if (secF2bStatusPill) {
        secF2bStatusPill.textContent = f2b.active ? 'Operacional' : 'No Detectado / Inactivo';
        secF2bStatusPill.style.color = f2b.active ? 'var(--status-online)' : 'var(--text-muted)';
    }

    if (secF2bBody) {
        const jails = f2b.jails || [];
        const banned = f2b.bannedIps || [];
        secF2bBody.innerHTML = `
            <div style="display:flex; flex-wrap:wrap; gap:16px; margin-bottom:10px;">
                <div style="font-size:0.80rem;">
                    <span style="color:var(--text-muted);">Jaulas Activas:</span>
                    <strong style="color:var(--text-pure); margin-left:6px;">${jails.length > 0 ? jails.map(j => `<span class="t-badge">${escapeHtml(j)}</span>`).join(' ') : 'Ninguna'}</strong>
                </div>
                <div style="font-size:0.80rem;">
                    <span style="color:var(--text-muted);">Total IPs Bloqueadas:</span>
                    <strong style="color:${f2b.totalBanned > 0 ? 'var(--status-warning)' : 'var(--status-online)'}; margin-left:6px;">${f2b.totalBanned || 0}</strong>
                </div>
            </div>
            ${banned.length > 0 ? `
                <div style="font-size:0.75rem; color:var(--text-muted); margin-bottom:6px;">IPs Bloqueadas Recientes:</div>
                <div style="display:flex; flex-wrap:wrap; gap:6px;">
                    ${banned.map(ip => `<code style="background:rgba(244,63,94,0.12); color:#fca5a5; padding:2px 6px; border-radius:3px; border:1px solid rgba(244,63,94,0.25); font-family:var(--font-mono); font-size:0.72rem;">${escapeHtml(ip)}</code>`).join('')}
                </div>
            ` : '<div style="font-size:0.76rem; color:var(--text-muted);">No hay IPs bloqueadas actualmente en la lista negra.</div>'}
        `;
    }
}
