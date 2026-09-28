import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, debounce } from '/pkg/jsutil/utils.js';

let sslItemsCache = [];

export async function openSSLModal() {
    const sslModal = document.getElementById('sslModal');
    const sslSearchInput = document.getElementById('sslSearchInput');
    if (!sslModal) return;
    sslModal.style.display = 'flex';
    if (sslSearchInput) sslSearchInput.value = '';
    await fetchAndRenderSSL();
}

export function closeSSLModal() {
    const sslModal = document.getElementById('sslModal');
    if (sslModal) sslModal.style.display = 'none';
}

export async function fetchAndRenderSSL() {
    const sslTableBody = document.getElementById('sslTableBody');
    const sslEmptyNotice = document.getElementById('sslEmptyNotice');
    if (!sslTableBody) return;
    sslTableBody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding:24px; color:var(--text-muted);">Inspeccionando certificados TLS en puerto 443...</td></tr>`;
    if (sslEmptyNotice) sslEmptyNotice.style.display = 'none';

    try {
        const srv = state.selectedServerName;
        const serverQuery = srv ? `?server=${encodeURIComponent(srv)}` : '';
        const res = await apiFetch(`/api/ssl/inspect${serverQuery}`);
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

export function renderSSLTable(items) {
    const sslTableBody = document.getElementById('sslTableBody');
    const sslEmptyNotice = document.getElementById('sslEmptyNotice');
    const sslSearchInput = document.getElementById('sslSearchInput');
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

        const isMaint = state.activeMaintenanceServices.has(item.serviceName);

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

    sslTableBody.querySelectorAll('.btn-modal-maint').forEach(btn => {
        btn.addEventListener('click', () => {
            const svcName = btn.getAttribute('data-svc');
            if (svcName) {
                const currentlyActive = state.activeMaintenanceServices.has(svcName);
                toggleMaintenanceMode(svcName, !currentlyActive, btn);
            }
        });
    });
}

export async function toggleMaintenanceMode(serviceName, enable, triggerBtn) {
    if (!serviceName) return;
    const origText = triggerBtn ? triggerBtn.textContent : '';
    if (triggerBtn) {
        triggerBtn.disabled = true;
        triggerBtn.textContent = enable ? 'Activando...' : 'Desactivando...';
    }

    try {
        const srv = state.selectedServerName;
        const serverQuery = srv ? `?server=${encodeURIComponent(srv)}` : '';
        const res = await apiFetch(`/api/maintenance/toggle${serverQuery}`, {
            method: 'POST',
            body: JSON.stringify({ serviceName, enable })
        });

        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'Error al cambiar modo mantenimiento');
        }

        if (enable) {
            state.activeMaintenanceServices.add(serviceName);
            showToast(`Servicio '${serviceName}' ahora devuelve 503 Maintenance`, 'info');
        } else {
            state.activeMaintenanceServices.delete(serviceName);
            showToast(`Modo mantenimiento desactivado para '${serviceName}'`, 'success');
        }

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

export async function reloadTraefikProxy(btn) {
    if (btn) {
        btn.disabled = true;
        btn.textContent = '⏳ Recargando...';
    }
    try {
        const srv = state.selectedServerName;
        const serverQuery = srv ? `?server=${encodeURIComponent(srv)}` : '';
        const res = await apiFetch(`/api/ssl/reload${serverQuery}`, {
            method: 'POST'
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

export function setupSSLListeners() {
    const btnTopSSL = document.getElementById('btnTopSSL');
    const btnRefreshSSL = document.getElementById('btnRefreshSSL');
    const btnReloadTraefik = document.getElementById('btnReloadTraefik');
    const btnCloseSSLModal = document.getElementById('btnCloseSSLModal');
    const btnCloseSSLModalBottom = document.getElementById('btnCloseSSLModalBottom');
    const sslModal = document.getElementById('sslModal');
    const sslSearchInput = document.getElementById('sslSearchInput');

    if (btnTopSSL) btnTopSSL.addEventListener('click', openSSLModal);
    if (btnRefreshSSL) btnRefreshSSL.addEventListener('click', fetchAndRenderSSL);
    if (btnReloadTraefik) btnReloadTraefik.addEventListener('click', () => reloadTraefikProxy(btnReloadTraefik));
    if (btnCloseSSLModal) btnCloseSSLModal.addEventListener('click', closeSSLModal);
    if (btnCloseSSLModalBottom) btnCloseSSLModalBottom.addEventListener('click', closeSSLModal);
    if (sslModal) {
        sslModal.addEventListener('click', (e) => {
            if (e.target === sslModal) closeSSLModal();
        });
    }
    if (sslSearchInput) sslSearchInput.addEventListener('input', debounce(() => renderSSLTable(sslItemsCache), 150));
}
