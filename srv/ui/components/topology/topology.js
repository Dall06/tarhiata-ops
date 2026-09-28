/**
 * Tarhiata Cloud Studio — Topology Component (opt/topology)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml, getDefaultPort } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch } from '/pkg/apiclient/api.js';

export function renderTopologyServicesTable(services, databases, links) {
    const topologyServicesTableBody = document.getElementById('topologyServicesTableBody');
    const topologyServicesCountBadge = document.getElementById('topologyServicesCountBadge');
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

    svcs.forEach(s => {
        const isPublic = s.expose || (s.domain && s.domain !== '');
        let port = '80';
        if (s.port) {
            port = s.port;
        } else if (s.ports) {
            const match = String(s.ports).match(/(\d+)/);
            if (match) port = match[1];
        }

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

    dbs.forEach(db => {
        const dbPort = db.port || getDefaultPort(db.engine);
        const targetNodeHtml = db.targetNode
            ? `<span class="t-badge" style="font-size:0.75rem;">${escapeHtml(db.targetNode)}</span>`
            : `<span style="color:var(--text-muted); font-size:0.75rem;">Manager / Primario</span>`;

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

export function populateLinkModalDropdowns() {
    const serviceListFrom = document.getElementById('serviceListFrom');
    const serviceListTo = document.getElementById('serviceListTo');

    if (serviceListFrom) {
        serviceListFrom.innerHTML = '';
        (state.swarmServicesCache || []).forEach(svc => {
            const opt = document.createElement('option');
            opt.value = svc.name;
            opt.label = `${svc.name} (App)`;
            serviceListFrom.appendChild(opt);
        });
    }
    if (serviceListTo) {
        serviceListTo.innerHTML = '';
        (state.swarmDatabasesCache || []).forEach(db => {
            const opt = document.createElement('option');
            opt.value = db.name;
            opt.label = `${db.name} (${db.engine || 'BD'})`;
            serviceListTo.appendChild(opt);
        });
        (state.swarmServicesCache || []).forEach(svc => {
            const opt = document.createElement('option');
            opt.value = svc.name;
            opt.label = `${svc.name} (App)`;
            serviceListTo.appendChild(opt);
        });
    }
}

export async function loadServiceLinks() {
    const linksTableBody = document.getElementById('linksTableBody');
    if (!linksTableBody) return;

    try {
        const res = await apiFetch('/api/links');
        if (!res.ok || !res.data) return;

        const links = res.data;
        state.currentServiceLinks = Array.isArray(links) ? links : [];
        renderTopologyServicesTable(state.swarmServicesCache, state.swarmDatabasesCache, state.currentServiceLinks);

        if (!state.currentServiceLinks || state.currentServiceLinks.length === 0) {
            linksTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Sin enlaces activos. Haz clic en 'Enlazar Servicios'.</td></tr>`;
            return;
        }

        linksTableBody.innerHTML = '';
        state.currentServiceLinks.forEach(l => {
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

                const delRes = await apiFetch(`/api/links?source_svc=${encodeURIComponent(fromSvc)}&target_svc=${encodeURIComponent(toSvc)}`, { method: 'DELETE' });
                if (delRes.ok) {
                    showToast(`Enlace eliminado: ${fromSvc} ⤬ ${toSvc}`, 'info');
                    loadServiceLinks();
                }
            });
        });
    } catch (err) {
        console.debug('Error cargando enlaces:', err);
    }
}

export function ensureTopologyModalMounted() {
    if (document.getElementById('linkModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Conectar Servicios (Service Linking) -->
<div class="t-modal-overlay" id="linkModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card">
        <div class="t-modal-header">
            <div class="t-modal-title">
                <span>🔗 Conectar Servicios entre sí</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseLinkModal" aria-label="Cerrar">×</button>
        </div>
        <form id="formLink">
            <div class="form-body">
                <div class="fast-notice">
                    <span class="notice-icon">💡</span>
                    <div>
                        <strong>Inyección Segura de Credenciales</strong>
                        <p>Comunica una aplicación con su base de datos usando la red interna del clúster.</p>
                    </div>
                </div>

                <div class="form-row-grid">
                    <div class="form-field">
                        <label for="linkFrom">App Origen (Consumidora)</label>
                        <input type="text" id="linkFrom" list="serviceListFrom" class="t-input" placeholder="Selecciona o escribe app (ej: mi-backend)" required autocomplete="off">
                        <datalist id="serviceListFrom"></datalist>
                    </div>
                    <div class="form-field">
                        <label for="linkTo">Destino (BD o Servicio)</label>
                        <input type="text" id="linkTo" list="serviceListTo" class="t-input" placeholder="Selecciona o escribe destino (ej: mi-postgres)" required autocomplete="off">
                        <datalist id="serviceListTo"></datalist>
                    </div>
                    <div class="form-field full-span">
                        <label for="linkVar">Variable de Entorno Inyectada</label>
                        <input type="text" id="linkVar" class="t-input" value="DATABASE_URL" required>
                    </div>
                </div>
            </div>
            <div class="t-modal-footer">
                <button type="button" class="t-btn t-btn-secondary" id="btnCancelLink">Cancelar</button>
                <button type="submit" class="t-btn t-btn-accent" id="btnSubmitLink">
                    <span>🔗 Conectar</span>
                </button>
            </div>
        </form>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
}

export function openLinkModal() {
    ensureTopologyModalMounted();
    populateLinkModalDropdowns();
    const linkForm = document.getElementById('formLink') || document.getElementById('linkForm');
    if (linkForm) linkForm.reset();
    openModal('linkModal');
}

export function closeLinkModal() {
    closeModal('linkModal');
}

export function setupTopologyEvents() {
    ensureTopologyModalMounted();

    const btnOpenLinkModal = document.getElementById('btnOpenLinkModal');
    const btnCloseLinkModal = document.getElementById('btnCloseLinkModal');
    const btnCancelLink = document.getElementById('btnCancelLink');
    const linkForm = document.getElementById('formLink') || document.getElementById('linkForm');

    if (btnOpenLinkModal) btnOpenLinkModal.addEventListener('click', openLinkModal);
    if (btnCloseLinkModal) btnCloseLinkModal.addEventListener('click', closeLinkModal);
    if (btnCancelLink) btnCancelLink.addEventListener('click', closeLinkModal);

    if (linkForm) {
        linkForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const linkSource = document.getElementById('linkFrom') || document.getElementById('linkSource');
            const linkTarget = document.getElementById('linkTo') || document.getElementById('linkTarget');
            const linkVar = document.getElementById('linkVar');
            const btnSubmitLink = document.getElementById('btnSubmitLink');

            const payload = {
                source_svc: linkSource ? linkSource.value.trim() : '',
                target_svc: linkTarget ? linkTarget.value.trim() : '',
                env_var_name: linkVar ? linkVar.value.trim() : ''
            };

            if (btnSubmitLink) btnSubmitLink.disabled = true;

            try {
                const res = await apiFetch('/api/links', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    showToast(`Error al crear enlace: ${res.error}`, 'error');
                    return;
                }

                showToast(`Enlace creado: ${payload.source_svc} ➔ ${payload.target_svc}`, 'success');
                closeLinkModal();
                loadServiceLinks();
            } finally {
                if (btnSubmitLink) btnSubmitLink.disabled = false;
            }
        });
    }
}

export const setupTopologyListeners = setupTopologyEvents;

