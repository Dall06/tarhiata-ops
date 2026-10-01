/**
 * Tarhiata Cloud Studio — Audit Log Component (srv/ui/components/audit)
 */

import { apiFetch } from '/pkg/apiclient/api.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';

let auditLogsCache = [];

export function ensureAuditModalMounted() {
    if (document.getElementById('auditModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Explorador de Audit Log -->
<div class="t-modal-overlay" id="auditModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card audit-modal-card" style="max-width:880px; width:94vw;">
        <div class="t-modal-header">
            <div style="display:flex; align-items:center; gap:10px;">
                <div class="modal-header-icon" style="background:rgba(99,102,241,0.15); color:var(--brand-primary-hover);">📜</div>
                <div>
                    <h2 class="t-modal-title">Bitácora de Auditoría (Audit Log)</h2>
                    <p class="t-modal-desc">Registro histórico inmutable de eventos, despliegues, modificaciones de entorno y comandos.</p>
                </div>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseAuditModal" aria-label="Cerrar">&times;</button>
        </div>

        <div class="t-modal-subhead" style="display:flex; justify-content:space-between; align-items:center; padding:10px 20px; background:var(--surface-input); border-bottom:1px solid var(--border-subtle); flex-wrap:wrap; gap:8px;">
            <div style="display:flex; gap:8px; align-items:center; flex-wrap:wrap;">
                <input type="text" id="auditSearchInput" class="t-input" placeholder="Buscar por recurso o detalle..." style="max-width:240px; height:30px; font-size:0.75rem; padding:4px 8px;">
                <select id="auditActionFilter" class="t-input" style="height:30px; font-size:0.75rem; padding:2px 8px; max-width:140px;">
                    <option value="">Todas las acciones</option>
                    <option value="DEPLOY">DEPLOY</option>
                    <option value="EDIT">EDIT</option>
                    <option value="DELETE">DELETE</option>
                    <option value="ROLLBACK">ROLLBACK</option>
                    <option value="LINK">LINK</option>
                    <option value="DRAIN">DRAIN</option>
                    <option value="PROVISION">PROVISION</option>
                </select>
            </div>
            <div style="display:flex; gap:8px;">
                <button type="button" class="mini-btn" id="btnRefreshAuditLogs" title="Actualizar registros">
                    🔄 Recargar
                </button>
            </div>
        </div>

        <div class="form-body" style="padding:16px 20px; max-height:480px; overflow-y:auto;">
            <table class="t-table" id="auditTable" style="width:100%; font-size:0.8rem;">
                <thead>
                    <tr>
                        <th style="width:150px;">Fecha y Hora</th>
                        <th style="width:100px;">Acción</th>
                        <th style="width:100px;">Tipo</th>
                        <th style="width:160px;">Recurso</th>
                        <th>Detalles</th>
                    </tr>
                </thead>
                <tbody id="auditTableBody">
                    <!-- Generado dinámicamente -->
                </tbody>
            </table>
            <div id="auditEmptyNotice" style="display:none; text-align:center; padding:32px 16px; color:var(--text-muted); font-size:0.82rem;">
                No se encontraron registros de auditoría que coincidan con el filtro.
            </div>
        </div>

        <div class="t-modal-footer">
            <span style="font-size:0.75rem; color:var(--text-muted);" id="auditLogsCount">
                0 eventos registrados
            </span>
            <button type="button" class="t-btn t-btn-secondary" id="btnCloseAuditModalBottom">Cerrar</button>
        </div>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
    bindAuditEvents();
}

function bindAuditEvents() {
    const btnCloseAuditModal = document.getElementById('btnCloseAuditModal');
    const btnCloseAuditModalBottom = document.getElementById('btnCloseAuditModalBottom');
    const btnRefreshAuditLogs = document.getElementById('btnRefreshAuditLogs');
    const auditSearchInput = document.getElementById('auditSearchInput');
    const auditActionFilter = document.getElementById('auditActionFilter');

    if (btnCloseAuditModal) btnCloseAuditModal.addEventListener('click', closeAuditModal);
    if (btnCloseAuditModalBottom) btnCloseAuditModalBottom.addEventListener('click', closeAuditModal);
    if (btnRefreshAuditLogs) btnRefreshAuditLogs.addEventListener('click', () => loadAuditLogs(true));

    if (auditSearchInput) {
        auditSearchInput.addEventListener('input', applyAuditFilters);
    }
    if (auditActionFilter) {
        auditActionFilter.addEventListener('change', applyAuditFilters);
    }
}

export async function openAuditModal() {
    ensureAuditModalMounted();
    const modal = document.getElementById('auditModal');
    if (modal) {
        openModal('auditModal');
        await loadAuditLogs();
    }
}

export function closeAuditModal() {
    closeModal('auditModal');
}

export async function loadAuditLogs(force = false) {
    const tbody = document.getElementById('auditTableBody');
    const emptyNotice = document.getElementById('auditEmptyNotice');
    if (tbody) {
        tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; padding:24px; color:var(--text-muted);">Cargando registros de auditoría...</td></tr>`;
    }

    try {
        const res = await apiFetch('/api/audit-logs');
        if (!res.ok) {
            if (tbody) {
                tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; padding:24px; color:var(--status-offline);">Error cargando logs: ${escapeHtml(res.error)}</td></tr>`;
            }
            return;
        }
        auditLogsCache = Array.isArray(res.data) ? res.data : [];
        applyAuditFilters();
    } catch (err) {
        if (tbody) {
            tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; padding:24px; color:var(--status-offline);">Error de red al consultar auditoría.</td></tr>`;
        }
    }
}

function applyAuditFilters() {
    const searchInput = document.getElementById('auditSearchInput');
    const actionFilter = document.getElementById('auditActionFilter');
    const tbody = document.getElementById('auditTableBody');
    const emptyNotice = document.getElementById('auditEmptyNotice');
    const countLabel = document.getElementById('auditLogsCount');

    if (!tbody) return;

    const q = (searchInput ? searchInput.value : '').toLowerCase().trim();
    const action = (actionFilter ? actionFilter.value : '').toUpperCase().trim();

    const filtered = auditLogsCache.filter(item => {
        const matchesAction = !action || (item.action || '').toUpperCase() === action;
        const matchesSearch = !q ||
            (item.resourceName || '').toLowerCase().includes(q) ||
            (item.resourceType || '').toLowerCase().includes(q) ||
            (item.action || '').toLowerCase().includes(q) ||
            (item.details || '').toLowerCase().includes(q);
        return matchesAction && matchesSearch;
    });

    if (countLabel) {
        countLabel.textContent = `${filtered.length} eventos (de ${auditLogsCache.length} totales)`;
    }

    if (filtered.length === 0) {
        tbody.innerHTML = '';
        if (emptyNotice) emptyNotice.style.display = 'block';
        return;
    }

    if (emptyNotice) emptyNotice.style.display = 'none';

    tbody.innerHTML = filtered.map(item => {
        const actionClass = getActionBadgeClass(item.action);
        const dateStr = item.timestamp ? new Date(item.timestamp).toLocaleString() : '—';
        return `
            <tr>
                <td style="color:var(--text-muted); font-size:0.75rem; white-space:nowrap;">${escapeHtml(dateStr)}</td>
                <td><span class="t-badge ${actionClass}">${escapeHtml(item.action || 'INFO')}</span></td>
                <td><span style="font-size:0.75rem; color:var(--text-secondary); text-transform:uppercase; font-weight:600;">${escapeHtml(item.resourceType || '—')}</span></td>
                <td style="font-weight:600; color:#fff;">${escapeHtml(item.resourceName || '—')}</td>
                <td style="color:var(--text-secondary); word-break:break-word; font-size:0.76rem;">${escapeHtml(item.details || '—')}</td>
            </tr>
        `;
    }).join('');
}

function getActionBadgeClass(action) {
    switch ((action || '').toUpperCase()) {
        case 'DEPLOY':
        case 'PROVISION':
            return 't-badge-success';
        case 'DELETE':
        case 'UNLINK':
        case 'DRAIN':
            return 't-badge-danger';
        case 'ROLLBACK':
        case 'EDIT':
            return 't-badge-warning';
        default:
            return 't-badge-default';
    }
}

export function setupAuditListeners() {
    ensureAuditModalMounted();
}
