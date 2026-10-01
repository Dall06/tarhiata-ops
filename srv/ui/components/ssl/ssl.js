import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, debounce } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';

let sslItemsCache = [];

export function ensureSSLModalMounted() {
    if (document.getElementById('sslModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Certificados SSL & Mantenimiento -->
<div class="t-modal-overlay" id="sslModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card ssl-modal-card">
        <div class="t-modal-header">
            <div style="display:flex; align-items:center; gap:10px;">
                <div class="modal-header-icon" style="background:rgba(16,185,129,0.12); color:var(--status-online);">🔐</div>
                <div>
                    <h2 class="t-modal-title">Certificados SSL &amp; Mantenimiento Traefik</h2>
                    <p class="t-modal-desc">Supervisión en tiempo real de certificados TLS (Let's Encrypt / ACME) y drenado de tráfico 503 por servicio.</p>
                </div>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseSSLModal" aria-label="Cerrar">&times;</button>
        </div>

        <div class="t-modal-subhead" style="display:flex; justify-content:space-between; align-items:center; padding:10px 20px; background:var(--surface-input); border-bottom:1px solid var(--border-subtle); flex-wrap:wrap; gap:8px;">
            <div style="display:flex; gap:8px; align-items:center;">
                <span style="font-size:0.78rem; color:var(--text-secondary);">Filtrar:</span>
                <input type="text" id="sslSearchInput" class="t-input" placeholder="Buscar dominio o servicio..." style="max-width:220px; height:28px; font-size:0.75rem; padding:4px 8px;">
            </div>
            <div style="display:flex; gap:8px;">
                <button type="button" class="mini-btn" id="btnRefreshSSL" title="Comprobar certificados SSL">
                    🔄 Inspeccionar SSL
                </button>
                <button type="button" class="mini-btn mini-btn-accent" id="btnReloadTraefik" title="Recargar configuración y renovar Traefik">
                    🚀 Recargar Traefik
                </button>
            </div>
        </div>

        <div class="form-body" style="padding:16px 20px; max-height:480px; overflow-y:auto;">
            <table class="t-table" id="sslTable" style="width:100%; font-size:0.8rem;">
                <thead>
                    <tr>
                        <th>Dominio</th>
                        <th>Servicio</th>
                        <th>Estado SSL</th>
                        <th>Emisor</th>
                        <th>Expira en</th>
                        <th>Acciones</th>
                    </tr>
                </thead>
                <tbody id="sslTableBody">
                    <!-- Generado dinámicamente -->
                </tbody>
            </table>
            <div id="sslEmptyNotice" style="display:none; text-align:center; padding:32px 16px; color:var(--text-muted); font-size:0.82rem;">
                No hay servicios públicos con dominio configurado en este servidor.
            </div>

            <div class="fast-notice" style="margin-top:16px;">
                <span class="notice-icon">💡</span>
                <span>Los certificados SSL son emitidos y renovados automáticamente por Traefik v3 vía ACME TLS challenge al apuntar el registro DNS A del dominio a la IP pública del servidor.</span>
            </div>
        </div>

        <div class="t-modal-footer">
            <span style="font-size:0.75rem; color:var(--text-muted);">
                Traefik Reverse Proxy · Modo Mantenimiento 503 HTTP Drain
            </span>
            <button type="button" class="t-btn t-btn-secondary" id="btnCloseSSLModalBottom">Cerrar</button>
        </div>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
}

export async function openSSLModal() {
    ensureSSLModalMounted();
    const sslModal = document.getElementById('sslModal');
    const sslSearchInput = document.getElementById('sslSearchInput');
    if (!sslModal) return;
    openModal('sslModal');
    if (sslSearchInput) sslSearchInput.value = '';
    await fetchAndRenderSSL();
}

export function closeSSLModal() {
    closeModal('sslModal');
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
            throw new Error(res.error || `Error HTTP ${res.status}`);
        }
        const data = res.data;
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
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ serviceName, enable })
        });

        if (!res.ok) {
            throw new Error(res.error || 'Error al cambiar modo mantenimiento');
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
            throw new Error(res.error || 'Error al recargar Traefik');
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
    ensureSSLModalMounted();
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
    if (sslSearchInput) sslSearchInput.addEventListener('input', debounce(() => renderSSLTable(sslItemsCache), 150));
}
