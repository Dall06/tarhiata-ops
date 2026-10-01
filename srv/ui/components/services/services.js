/**
 * Tarhiata Cloud Studio — Swarm Services Component (opt/services)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml, getDefaultPort } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { sendDesktopNotification } from '/pkg/notify/notify.js';

export function renderMasterServicesTable(services, databases, links, callbacks = {}) {
    const tableBody = document.getElementById('masterServicesTableBody');
    const emptyBox = document.getElementById('swarmServicesEmpty');
    const filterInput = document.getElementById('servicesFilterInput');
    const tabServicesCount = document.getElementById('tabServicesCount');
    const tabDatabasesCount = document.getElementById('tabDatabasesCount');

    const svcs = services || state.swarmServicesCache || [];
    const dbs = databases || state.swarmDatabasesCache || [];
    const lnks = links || state.currentServiceLinks || [];

    if (tabServicesCount) tabServicesCount.textContent = svcs.length;
    if (tabDatabasesCount) tabDatabasesCount.textContent = dbs.length;

    if (!tableBody) return;

    const query = (filterInput ? filterInput.value : '').toLowerCase().trim();

    // Consolidar todos los items (Apps, BDs, Framework)
    const items = [];

    // 1. Aplicaciones y Framework
    svcs.forEach(s => {
        const isFramework = ['tarhiata_proxy_traefik', 'tarhiata_obs_portainer', 'tarhiata_obs_dozzle'].includes(s.name) ||
                            (s.image && (s.image.includes('traefik') || s.image.includes('portainer') || s.image.includes('dozzle')));
        items.push({
            name: s.name,
            image: s.image || 'imagen docker',
            type: isFramework ? 'framework' : 'app',
            replicas: s.replicas || '1/1',
            port: s.port || (s.ports ? (String(s.ports).match(/(\d+)/) || ['','80'])[1] : '80'),
            domain: s.domain || '',
            expose: s.expose || (s.domain && s.domain !== ''),
            targetNode: s.targetNode || '',
            raw: s
        });
    });

    // 2. Bases de Datos
    dbs.forEach(db => {
        const port = db.internalPort || getDefaultPort(db.engine);
        items.push({
            name: db.name,
            image: `${db.engine || 'database'}:${db.version || 'latest'}`,
            type: 'db',
            engine: db.engine || 'postgres',
            replicas: db.status === 'running' ? '1/1' : (db.status ? '0/1' : '1/1'),
            port: port,
            domain: db.externalUrl || '',
            expose: false,
            targetNode: db.targetNode || 'Manager / Primario',
            raw: db
        });
    });

    // Ordenar: Aplicaciones primero, Bases de datos segundo, Framework al final
    items.sort((a, b) => {
        const priority = { app: 1, db: 2, framework: 3 };
        const diff = (priority[a.type] || 2) - (priority[b.type] || 2);
        if (diff !== 0) return diff;
        return a.name.localeCompare(b.name);
    });

    // Filtrar en vivo
    const filtered = items.filter(it => {
        if (!query) return true;
        return it.name.toLowerCase().includes(query) ||
               it.image.toLowerCase().includes(query) ||
               it.type.toLowerCase().includes(query) ||
               (it.domain && it.domain.toLowerCase().includes(query));
    });

    if (items.length === 0) {
        tableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty">Sin aplicaciones ni contenedores desplegados. Haz clic en '+ Desplegar App'.</td></tr>`;
        if (emptyBox) emptyBox.style.display = 'flex';
        return;
    }

    if (emptyBox) emptyBox.style.display = 'none';

    if (filtered.length === 0) {
        tableBody.innerHTML = `<tr><td colspan="6" class="t-td-empty">No se encontraron servicios que coincidan con "${escapeHtml(query)}".</td></tr>`;
        return;
    }

    tableBody.innerHTML = '';

    filtered.forEach(it => {
        const tr = document.createElement('tr');
        const isOnline = it.replicas && !it.replicas.startsWith('0/');

        let icon = '🚀';
        let typeBadgeHtml = '<span class="t-badge t-badge-active" style="font-size:0.72rem;">App Docker</span>';
        if (it.type === 'framework') {
            icon = '⚡';
            typeBadgeHtml = '<span class="t-badge" style="background:rgba(99,102,241,0.12); color:#a5b4fc; border-color:rgba(99,102,241,0.25); font-size:0.72rem;">⚡ Framework</span>';
        } else if (it.type === 'db') {
            icon = '🗄️';
            typeBadgeHtml = `<span class="t-badge" style="background:rgba(16,185,129,0.12); color:#34d399; border-color:rgba(16,185,129,0.25); font-size:0.72rem;">🗄️ ${escapeHtml((it.engine || 'DB').toUpperCase())}</span>`;
        }

        // Links badges
        const relatedLinks = lnks.filter(l => (l.source_svc || l.SourceSvc) === it.name || (l.target_svc || l.TargetSvc) === it.name);
        let linkBadgesHtml = '';
        if (relatedLinks.length > 0) {
            linkBadgesHtml = `<div style="display:flex; gap:4px; flex-wrap:wrap; margin-top:4px;">` +
                relatedLinks.map(l => {
                    const isSrc = (l.source_svc || l.SourceSvc) === it.name;
                    const other = isSrc ? (l.target_svc || l.TargetSvc) : (l.source_svc || l.SourceSvc);
                    const tag = isSrc ? `🔗 ${other}` : `⬅️ ${other}`;
                    return `<span class="t-badge" style="font-size:0.68rem; padding:1px 5px; border-color:rgba(99,102,241,0.3); background:rgba(99,102,241,0.1); color:#a5b4fc;" title="Enlace con ${escapeHtml(other)}">${escapeHtml(tag)}</span>`;
                }).join('') +
                `</div>`;
        }

        const publicRouteHtml = it.expose && it.domain
            ? `<div style="display:flex; align-items:center; gap:6px; flex-wrap:wrap; margin-top:2px;">
                 <a href="https://${escapeHtml(it.domain)}" target="_blank" style="color:var(--brand-primary); text-decoration:none; font-weight:600; font-size:0.80rem; display:inline-flex; align-items:center; gap:4px;">
                   🌐 https://${escapeHtml(it.domain)} ↗
                 </a>
                 <span class="card-ssl-pill" style="background:rgba(16,185,129,0.12); color:#10b981; border:1px solid rgba(16,185,129,0.25); font-size:0.65rem; padding:1px 4px; border-radius:3px;">SSL</span>
               </div>`
            : `<div style="color:var(--text-muted); font-size:0.75rem; margin-top:2px;">🔒 Red Interna Swarm</div>`;

        const targetNodeHtml = it.targetNode
            ? `<span class="t-badge" style="font-size:0.74rem;">${escapeHtml(it.targetNode)}</span>`
            : `<span style="color:var(--text-muted); font-size:0.74rem;">Cualquiera (Global)</span>`;

        let actionsHtml = '';
        const n = escapeHtml(it.name);
        const isTraefik = it.name.includes('traefik') || it.image.includes('traefik');
        if (it.type === 'framework') {
            actionsHtml = `
                <div class="row-actions-wrap">
                    <button type="button" class="row-action-primary btn-logs-svc" data-name="${n}">📜 Logs</button>
                    <button type="button" class="row-action-chevron" data-dropdown="${n}-fw" title="Más acciones" aria-haspopup="menu" aria-expanded="false" aria-controls="dd-${n}-fw">▾</button>
                    <div class="row-actions-dropdown" id="dd-${n}-fw" role="menu">
                        <button type="button" class="dd-item btn-metrics-svc" data-name="${n}" role="menuitem">📊 Métricas</button>
                        <button type="button" class="dd-item btn-restart-svc" data-name="${n}" role="menuitem">🔄 Reiniciar</button>
                        ${isTraefik ? `<button type="button" class="dd-item btn-repair-traefik" data-name="${n}" role="menuitem">🔧 Reparar Traefik</button>` : ''}
                        <button type="button" class="dd-item btn-vol-svc" data-name="${n}" role="menuitem">📁 Archivos</button>
                    </div>
                </div>
            `;
        } else if (it.type === 'db') {
            actionsHtml = `
                <div class="row-actions-wrap">
                    <button type="button" class="row-action-primary btn-logs-db" data-name="${n}">📜 Logs</button>
                    <button type="button" class="row-action-chevron" data-dropdown="${n}-db" title="Más acciones" aria-haspopup="menu" aria-expanded="false" aria-controls="dd-${n}-db">▾</button>
                    <div class="row-actions-dropdown" id="dd-${n}-db" role="menu">
                        <button type="button" class="dd-item btn-metrics-svc" data-name="${n}" role="menuitem">📊 Métricas</button>
                        <button type="button" class="dd-item btn-restart-db" data-name="${n}" role="menuitem">🔄 Reiniciar</button>
                        <button type="button" class="dd-item btn-backup-db" data-name="${n}" data-engine="${escapeHtml(it.engine || 'postgres')}" role="menuitem">💾 Backup</button>
                        <button type="button" class="dd-item btn-vol-db" data-name="${n}" role="menuitem">📁 Archivos</button>
                        <div class="dd-sep"></div>
                        <button type="button" class="dd-item dd-danger btn-delete-db" data-name="${n}" role="menuitem">✕ Eliminar BD</button>
                    </div>
                </div>
            `;
        } else {
            actionsHtml = `
                <div class="row-actions-wrap">
                    <button type="button" class="row-action-primary btn-logs-svc" data-name="${n}">📜 Logs</button>
                    <button type="button" class="row-action-chevron" data-dropdown="${n}-app" title="Más acciones" aria-haspopup="menu" aria-expanded="false" aria-controls="dd-${n}-app">▾</button>
                    <div class="row-actions-dropdown" id="dd-${n}-app" role="menu">
                        <button type="button" class="dd-item btn-metrics-svc" data-name="${n}" role="menuitem">📊 Métricas</button>
                        <button type="button" class="dd-item btn-env-svc" data-name="${n}" role="menuitem">🔑 Variables Env</button>
                        <button type="button" class="dd-item btn-restart-svc" data-name="${n}" role="menuitem">🔄 Reiniciar</button>
                        <button type="button" class="dd-item btn-history-svc" data-name="${n}" role="menuitem">⏳ Versiones</button>
                        <button type="button" class="dd-item btn-vol-svc" data-name="${n}" role="menuitem">📁 Archivos</button>
                        <button type="button" class="dd-item btn-rebuild-svc" data-name="${n}" role="menuitem">🔧 Rebuild &amp; Deploy</button>
                        <button type="button" class="dd-item btn-edit-svc" data-name="${n}" data-expose="${it.expose}" data-domain="${escapeHtml(it.domain)}" role="menuitem">⚙️ Configurar</button>
                        <div class="dd-sep"></div>
                        <button type="button" class="dd-item dd-danger btn-del-svc" data-name="${n}" role="menuitem">✕ Eliminar</button>
                    </div>
                </div>
            `;
        }

        tr.innerHTML = `
            <td>
                <div style="display:flex; align-items:center; gap:8px;">
                    <span style="font-size:1.15rem; line-height:1; flex-shrink:0;">${icon}</span>
                    <div style="min-width:0; overflow:hidden;">
                        <strong style="color:var(--text-pure); font-size:0.86rem; display:block; text-overflow:ellipsis; overflow:hidden; white-space:nowrap;" title="${escapeHtml(it.name)}">${escapeHtml(it.name)}</strong>
                        <span style="font-size:0.72rem; color:var(--text-muted); font-family:var(--font-mono); display:block; text-overflow:ellipsis; overflow:hidden; white-space:nowrap;" title="${escapeHtml(it.image)}">${escapeHtml(it.image)}</span>
                    </div>
                </div>
            </td>
            <td>${typeBadgeHtml}</td>
            <td>
                <span class="svc-pill ${isOnline ? 'svc-pill-active' : ''}" style="font-size:0.74rem;">
                    <span class="status-dot ${isOnline ? 'status-online' : 'status-offline'}" style="width:6px; height:6px;"></span>
                    ${escapeHtml(it.replicas)}
                </span>
            </td>
            <td>
                <div>
                    <code style="font-family:var(--font-mono); font-size:0.78rem; color:var(--brand-primary);">${escapeHtml(it.name)}:${escapeHtml(it.port)}</code>
                    ${publicRouteHtml}
                    ${linkBadgesHtml}
                </div>
            </td>
            <td>${targetNodeHtml}</td>
            <td>${actionsHtml}</td>
        `;

        tableBody.appendChild(tr);
    });

    wireMasterTableActions(callbacks);
}

export function wireMasterTableActions(callbacks = {}) {
    // ─── Dropdown chevron toggle ───────────────────────────────────────────
    const closeAllRowDropdowns = () => {
        document.querySelectorAll('.row-actions-dropdown.open').forEach(el => el.classList.remove('open'));
        document.querySelectorAll('.row-action-chevron.open').forEach(el => {
            el.classList.remove('open');
            el.setAttribute('aria-expanded', 'false');
        });
    };

    document.querySelectorAll('.row-action-chevron').forEach(chevron => {
        chevron.onclick = (e) => {
            e.stopPropagation();
            const ddId = 'dd-' + chevron.getAttribute('data-dropdown');
            const dd = document.getElementById(ddId);
            if (!dd) return;
            const isOpen = dd.classList.contains('open');
            // Cerrar todos los dropdowns abiertos primero
            closeAllRowDropdowns();
            if (!isOpen) {
                dd.classList.add('open');
                chevron.classList.add('open');
                chevron.setAttribute('aria-expanded', 'true');
            }
        };
    });

    // Cerrar dropdowns al hacer click afuera o con Escape
    if (!document._rowActionsClickOutside) {
        document._rowActionsClickOutside = true;
        document.addEventListener('click', closeAllRowDropdowns);
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') closeAllRowDropdowns();
        });
    }

    // ─── Métricas ──────────────────────────────────────────────────────────
    document.querySelectorAll('.btn-metrics-svc').forEach(btn => {
        btn.onclick = () => openServiceMetricsModal(btn.getAttribute('data-name'));
    });

    // ─── Logs ──────────────────────────────────────────────────────────────
    document.querySelectorAll('.btn-logs-svc, .btn-logs-db').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            const { openLogsModal } = window;
            if (openLogsModal) openLogsModal(name);
            else if (callbacks.onOpenLogs) callbacks.onOpenLogs(name);
        };
    });

    document.querySelectorAll('.btn-restart-svc').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            if (name) restartServiceOrContainer(name, btn);
        };
    });

    document.querySelectorAll('.btn-repair-traefik').forEach(btn => {
        btn.onclick = async () => {
            const acmeEmail = prompt('¿Reparar Traefik? Ingresa el email ACME para SSL/HTTPS (déjalo vacío para omitir HTTPS):', '');
            if (acmeEmail === null) return;
            btn.disabled = true;
            btn.textContent = '⏳...';
            showToast('Reparando Traefik...', 'info');
            try {
                const res = await apiFetch('/api/tools/repair-traefik', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ acmeEmail: acmeEmail.trim() })
                });
                if (res.ok) {
                    showToast('Traefik reparado y redesplegado.', 'success');
                }
            } finally {
                btn.disabled = false;
                btn.textContent = '🔧 Reparar Traefik';
            }
        };
    });

    document.querySelectorAll('.btn-restart-db').forEach(btn => {
        btn.onclick = async () => {
            const name = btn.getAttribute('data-name');
            if (!name) return;
            btn.disabled = true;
            btn.textContent = '⏳...';
            showToast(`Reiniciando base de datos '${name}'...`, 'info');
            try {
                await apiFetch('/api/databases/restart', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: name, server: state.selectedServerName || '' })
                });
                showToast(`Base de datos '${name}' reiniciada.`, 'success');
            } finally {
                btn.disabled = false;
                btn.textContent = '🔄';
            }
        };
    });

    document.querySelectorAll('.btn-env-svc').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            const { openEnvModal } = window;
            if (openEnvModal) openEnvModal(name);
            else if (callbacks.onOpenEnv) callbacks.onOpenEnv(name);
        };
    });

    document.querySelectorAll('.btn-history-svc').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            if (name) openHistoryModal(name, () => loadSwarmStatus(state.selectedServerName));
        };
    });

    document.querySelectorAll('.btn-backup-db').forEach(btn => {
        btn.onclick = async () => {
            const name = btn.getAttribute('data-name');
            const engine = btn.getAttribute('data-engine') || 'postgres';
            if (!name) return;
            btn.disabled = true;
            btn.textContent = '💾...';
            showToast(`Generando backup para '${name}' (${engine})...`, 'info');
            try {
                const res = await apiFetch(`/api/databases/backup?name=${encodeURIComponent(name)}&engine=${encodeURIComponent(engine)}&server=${encodeURIComponent(state.selectedServerName || '')}`);
                if (res.ok && res.data) {
                    showToast(`Backup de '${name}' generado (${res.data.size || 'OK'}).`, 'success');
                }
            } finally {
                btn.disabled = false;
                btn.textContent = '💾';
            }
        };
    });

    document.querySelectorAll('.btn-vol-svc, .btn-vol-db').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            const { openVolumeModal } = window;
            if (openVolumeModal) openVolumeModal(`/opt/data/${name}`);
            else if (callbacks.onOpenVolume) callbacks.onOpenVolume(`/opt/data/${name}`);
        };
    });

    document.querySelectorAll('.btn-rebuild-svc').forEach(btn => {
        btn.onclick = async () => {
            const name = btn.getAttribute('data-name');
            if (!name) return;
            if (!confirm(`¿Reconstruir '${name}' desde su repo git y redesplegarlo? Esto solo funciona si el servicio tiene origen "git" configurado.`)) return;

            ensureServicesModalsMounted();
            const titleEl = document.getElementById('buildLogsServiceName');
            const contentEl = document.getElementById('buildLogsContent');
            if (titleEl) titleEl.textContent = name;
            if (contentEl) contentEl.textContent = '';
            openModal('buildLogsModal');

            const appendLog = (line) => {
                if (!contentEl) return;
                contentEl.textContent += line + '\n';
                contentEl.scrollTop = contentEl.scrollHeight;
            };

            btn.disabled = true;
            try {
                // fetch() directo (no apiFetch): apiFetch consume el body entero antes de
                // devolverlo, lo que rompería el streaming línea por línea en vivo.
                const res = await fetch(`/api/services/rebuild?name=${encodeURIComponent(name)}&server=${encodeURIComponent(state.selectedServerName || '')}`, {
                    method: 'POST'
                });
                if (!res.ok) {
                    const errText = await res.text();
                    appendLog(`✕ Error: ${errText || 'no se pudo iniciar el rebuild'}`);
                    showToast(`Error en rebuild de '${name}'`, 'error');
                    return;
                }
                // Parser propio en vez de consumeNDJSONStream: el backend emite {t,m}/
                // {t:"done",d} (mismo formato que streamJSON/WriteEvent en Go), no
                // {type,data} como espera ese helper genérico.
                let failed = false;
                const reader = res.body.getReader();
                const decoder = new TextDecoder('utf-8');
                let buffer = '';
                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;
                    buffer += decoder.decode(value, { stream: true });
                    const parts = buffer.split('\n');
                    buffer = parts.pop();
                    for (const line of parts) {
                        if (!line.trim()) continue;
                        try {
                            const event = JSON.parse(line);
                            if (event.t === 'error') {
                                failed = true;
                                appendLog(`✕ ${event.m}`);
                            } else if (event.t === 'done') {
                                // nada que imprimir: el resumen final lo da el toast
                            } else {
                                appendLog(event.m || line);
                            }
                        } catch (e) {
                            console.debug('línea NDJSON no parseable:', line, e);
                        }
                    }
                }

                if (failed) {
                    showToast(`Error en rebuild de '${name}'`, 'error');
                } else {
                    showToast(`'${name}' reconstruido y redesplegado.`, 'success');
                    if (state.selectedServerName) {
                        const { refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
                        refreshServerTelemetry(state.selectedServerName, false);
                    }
                }
            } finally {
                btn.disabled = false;
            }
        };
    });

    document.querySelectorAll('.btn-edit-svc').forEach(btn => {
        btn.onclick = () => {
            const name = btn.getAttribute('data-name');
            const expose = btn.getAttribute('data-expose') === 'true';
            const domain = btn.getAttribute('data-domain');
            openEditServiceModal(name, expose, domain);
        };
    });

    document.querySelectorAll('.btn-del-svc').forEach(btn => {
        btn.onclick = async () => {
            const name = btn.getAttribute('data-name');
            if (!confirm(`¿Estás seguro de eliminar el servicio '${name}' de Docker Swarm?`)) return;
            const res = await apiFetch(`/api/services/${encodeURIComponent(name)}?server=${encodeURIComponent(state.selectedServerName || '')}`, { method: 'DELETE' });
            if (res.ok) {
                showToast(`Servicio '${name}' eliminado.`, 'info');
                if (state.selectedServerName) {
                    await loadSwarmStatus(state.selectedServerName);
                }
            }
        };
    });

    document.querySelectorAll('.btn-delete-db').forEach(btn => {
        btn.onclick = async () => {
            const name = btn.getAttribute('data-name');
            if (!confirm(`¿Estás seguro de eliminar la base de datos '${name}'? ¡Se perderán los datos locales no respaldados!`)) return;
            const res = await apiFetch(`/api/databases?name=${encodeURIComponent(name)}&server=${encodeURIComponent(state.selectedServerName || '')}`, { method: 'DELETE' });
            if (res.ok) {
                showToast(`Base de datos '${name}' eliminada.`, 'info');
                if (state.selectedServerName) {
                    await loadSwarmStatus(state.selectedServerName);
                }
            }
        };
    });
}

export function renderAppCards(services, callbacks = {}) {
    renderMasterServicesTable(services, state.swarmDatabasesCache, state.currentServiceLinks, callbacks);
}

export function renderServicesTable(services) {
    const hostServicesTableBody = document.getElementById('servicesTableBody') || document.getElementById('hostServicesTableBody');
    if (!hostServicesTableBody) return;
    hostServicesTableBody.innerHTML = '';

    if (!services || services.length === 0) {
        hostServicesTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">No se encontraron servicios del sistema corriendo en este servidor.</td></tr>`;
        return;
    }

    services.forEach(svc => {
        const tr = document.createElement('tr');
        const isRunning = svc.subState === 'running' || svc.activeState === 'active';
        tr.innerHTML = `
            <td>
                <span class="status-dot ${isRunning ? 'status-online' : 'status-offline'}" style="margin-right:6px;"></span>
                <span style="font-weight:600; font-family:var(--font-mono); font-size:0.78rem;">${escapeHtml(svc.name)}</span>
            </td>
            <td><span class="t-badge" style="font-size:0.72rem;">${escapeHtml(svc.activeState)}</span></td>
            <td><span style="font-size:0.75rem; color:var(--text-secondary); font-family:var(--font-mono);">${escapeHtml(svc.subState)}</span></td>
            <td style="font-size:0.78rem; color:var(--text-secondary);">${escapeHtml(svc.description || '—')}</td>
        `;
        hostServicesTableBody.appendChild(tr);
    });
}

export async function restartServiceOrContainer(name, btnElement) {
    if (!name) return;
    if (btnElement) {
        btnElement.disabled = true;
        btnElement.textContent = '⏳...';
    }
    showToast(`Reiniciando servicio '${name}'...`, 'info');

    try {
        const res = await apiFetch('/api/services/restart', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, server: state.selectedServerName || '' })
        });
        if (res.ok) {
            showToast(`Servicio '${name}' reiniciado con éxito.`, 'success');
        }
    } finally {
        if (btnElement) {
            btnElement.disabled = false;
            btnElement.textContent = '🔄 Reiniciar';
        }
    }
}

export function ensureServicesModalsMounted() {
    if (document.getElementById('deployModal') && document.getElementById('editServiceModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Desplegar Nueva Aplicación -->
<div class="t-modal-overlay" id="deployModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card">
        <div class="t-modal-header">
            <div class="t-modal-title">
                <span>🚀 Desplegar Nueva Aplicación</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseDeployModal" aria-label="Cerrar">×</button>
        </div>
        <form id="formDeploy">
            <div class="form-body">
                <div class="fast-notice">
                    <span class="notice-icon">🐋</span>
                    <div>
                        <strong>Despliegue Rápido Asistido</strong>
                        <p>Tu contenedor se iniciará en Docker Swarm con proxy inverso Traefik y certificado SSL HTTPS automático.</p>
                    </div>
                </div>

                <div class="form-row-grid">
                    <div class="form-field">
                        <label for="depName">Nombre de la App</label>
                        <input type="text" id="depName" class="t-input" placeholder="ej: mi-backend" required>
                    </div>
                    <div class="form-field">
                        <label for="depPort">Puerto del Contenedor</label>
                        <input type="number" id="depPort" class="t-input" placeholder="ej: 3000 o 80" value="80" required>
                    </div>
                    <div class="form-field full-span">
                        <label for="depImage">Imagen Docker</label>
                        <input type="text" id="depImage" class="t-input" placeholder="ej: nginx:alpine o usuario/repo:tag" required>
                    </div>
                    <div class="form-field full-span">
                        <label for="depDomain">Dominio / Subdominio Web</label>
                        <input type="text" id="depDomain" class="t-input" placeholder="ej: api.tudominio.com">
                        <div id="depDomainDnsFeedback" class="dns-feedback-hint" style="display:none;"></div>
                    </div>
                    <div class="form-field">
                        <label for="depDB">Vincular Base de Datos</label>
                        <select id="depDB" class="t-input">
                            <option value="">Ninguna (Autónoma)</option>
                            <option value="postgres">PostgreSQL</option>
                            <option value="mysql">MySQL</option>
                            <option value="mongodb">MongoDB</option>
                            <option value="redis">Redis</option>
                        </select>
                    </div>
                    <div class="form-field">
                        <label for="depEnv">Nombre Variable de Entorno</label>
                        <input type="text" id="depEnv" class="t-input" value="DATABASE_URL">
                    </div>
                </div>
            </div>
            <div class="t-modal-footer">
                <button type="button" class="t-btn t-btn-secondary" id="btnCancelDeploy">Cancelar</button>
                <button type="submit" class="t-btn t-btn-primary" id="btnSubmitDeploy">
                    <span>Desplegar en Swarm</span>
                </button>
            </div>
        </form>
    </div>
</div>

<!-- Modal: Configurar Acceso de Servicio -->
<div class="t-modal-overlay" id="editServiceModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card">
        <div class="t-modal-header">
            <div class="t-modal-title">
                <span>⚙️ Configurar Acceso de Servicio</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseEditServiceModal" aria-label="Cerrar">×</button>
        </div>
        <form id="formEditService">
            <input type="hidden" id="editServiceName">
            <div class="form-body">
                <div class="fast-notice">
                    <span class="notice-icon">🌐</span>
                    <div>
                        <strong id="editServiceTitle">Ajustes de Visibilidad</strong>
                        <p>Configura si este servicio debe responder públicamente en Internet o solo dentro de la red del clúster.</p>
                    </div>
                </div>

                <div class="form-row-grid">
                    <div class="form-field full-span">
                        <label for="editServiceExpose">Tipo de Visibilidad</label>
                        <select id="editServiceExpose" class="t-input">
                            <option value="true">🌐 Público (Accesible en Internet vía Traefik)</option>
                            <option value="false">🔒 Privado (Solo accesible dentro del clúster)</option>
                        </select>
                    </div>

                    <div class="form-field full-span" id="editDomainField">
                        <label for="editServiceDomain">Dominio / Subdominio Web</label>
                        <input type="text" id="editServiceDomain" class="t-input" placeholder="ej: api.tudominio.com">
                        <div id="editDomainDnsFeedback" class="dns-feedback-hint" style="display:none;"></div>
                    </div>

                    <div class="form-field full-span">
                        <label for="editServicePort">Puerto Interno del Contenedor</label>
                        <input type="number" id="editServicePort" class="t-input" value="80">
                    </div>

                    <div class="form-field full-span">
                        <label for="editServiceSourceType">Origen de la Imagen</label>
                        <select id="editServiceSourceType" class="t-input">
                            <option value="image">🐋 Imagen Docker (Hub/registry)</option>
                            <option value="archive">📦 Zip de imagen pre-armada</option>
                            <option value="git">🔧 Construir desde repo Git (Dockerfile)</option>
                        </select>
                    </div>

                    <div id="editGitSourceFields" style="display:none;">
                        <div class="form-field full-span">
                            <label for="editServiceGitRepoURL">URL del repo</label>
                            <input type="text" id="editServiceGitRepoURL" class="t-input" placeholder="https://github.com/org/repo.git">
                        </div>
                        <div class="form-field">
                            <label for="editServiceGitBranch">Branch</label>
                            <input type="text" id="editServiceGitBranch" class="t-input" placeholder="main">
                        </div>
                        <div class="form-field">
                            <label for="editServiceDockerfilePath">Ruta del Dockerfile</label>
                            <input type="text" id="editServiceDockerfilePath" class="t-input" placeholder="Dockerfile">
                        </div>
                        <div class="form-field full-span">
                            <label for="editServiceGitAccessToken">Token de acceso (solo repos privados)</label>
                            <input type="password" id="editServiceGitAccessToken" class="t-input" placeholder="Dejar vacío para no cambiar / repo público" autocomplete="new-password">
                        </div>
                    </div>
                </div>
            </div>
            <div class="t-modal-footer">
                <button type="button" class="mini-btn mini-btn-accent" id="btnEditServiceOpenEnv" style="margin-right:auto;">
                    <span>🔑 Variables de Entorno (.env)</span>
                </button>
                <button type="button" class="t-btn t-btn-secondary" id="btnCancelEditService">Cancelar</button>
                <button type="submit" class="t-btn t-btn-primary" id="btnSubmitEditService">
                    <span>Guardar Cambios</span>
                </button>
            </div>
        </form>
    </div>
</div>

<!-- Modal: Logs de Build (build-from-source) -->
<div class="t-modal-overlay" id="buildLogsModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card" style="max-width: 720px;">
        <div class="t-modal-header">
            <div>
                <h2 class="t-modal-title">🔧 Build: <span id="buildLogsServiceName" style="color:var(--brand-primary); font-family:var(--font-mono);">—</span></h2>
                <span class="t-modal-desc">Clone → build → push → deploy, en vivo</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseBuildLogsModal" aria-label="Cerrar">×</button>
        </div>
        <div class="form-body">
            <pre id="buildLogsContent" style="background:#0a0a0a; color:#d1d5db; padding:14px; border-radius:8px; max-height:420px; overflow-y:auto; font-family:var(--font-mono); font-size:0.78rem; white-space:pre-wrap;"></pre>
        </div>
        <div class="t-modal-footer">
            <button type="button" class="t-btn t-btn-secondary" id="btnCloseBuildLogs2">Cerrar</button>
        </div>
    </div>
</div>

<!-- Modal: Historial de Versiones y Rollback -->
<div class="t-modal-overlay" id="historyModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card" style="max-width: 720px;">
        <div class="t-modal-header">
            <div>
                <h2 class="t-modal-title">⏳ Historial: <span id="historyServiceName" style="color:var(--brand-primary); font-family:var(--font-mono);">—</span></h2>
                <span class="t-modal-desc">Versiones y snapshots de despliegues previos con restauración 1-click</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseHistoryModal" aria-label="Cerrar">×</button>
        </div>
        <div class="form-body">
            <div class="fast-notice">
                <span class="notice-icon">🔄</span>
                <div>
                    <strong>Rollback Instantáneo</strong>
                    <p>Restaura cualquier versión histórica de tu aplicación en Docker Swarm con toda su configuración anterior intacta.</p>
                </div>
            </div>
            <div style="margin-top:14px; max-height:360px; overflow-y:auto;">
                <table class="t-table" style="width:100%; font-size:0.8rem;">
                    <thead>
                        <tr>
                            <th style="width:25%;">Fecha</th>
                            <th style="width:35%;">Imagen Docker</th>
                            <th style="width:20%;">Puerto / Dominio</th>
                            <th style="width:20%; text-align:center;">Acción</th>
                        </tr>
                    </thead>
                    <tbody id="historyTableBody">
                        <tr><td colspan="4" class="t-td-empty">Cargando versiones...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>
        <div class="t-modal-footer">
            <button type="button" class="t-btn t-btn-secondary" id="btnCancelHistory">Cerrar</button>
        </div>
    </div>
</div>`;
    while (div.firstElementChild) {
        document.body.appendChild(div.firstElementChild);
    }
}

export async function loadSwarmStatus(serverName) {
    const { refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
    return refreshServerTelemetry(serverName, false);
}

export function openDeployModal() {
    ensureServicesModalsMounted();
    const deployForm = document.getElementById('formDeploy') || document.getElementById('deployForm');
    const deployFeedback = document.getElementById('deployFeedback');
    if (deployForm) deployForm.reset();
    if (deployFeedback) deployFeedback.style.display = 'none';
    openModal('deployModal');
}

export function closeDeployModal() {
    closeModal('deployModal');
}

function toggleEditGitSourceFields() {
    const sourceType = document.getElementById('editServiceSourceType');
    const gitFields = document.getElementById('editGitSourceFields');
    if (gitFields) gitFields.style.display = (sourceType && sourceType.value === 'git') ? 'contents' : 'none';
}

export async function openEditServiceModal(name, expose, domain) {
    ensureServicesModalsMounted();
    const editSvcName = document.getElementById('editServiceName') || document.getElementById('editSvcName');
    const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
    const editSvcDomain = document.getElementById('editServiceDomain') || document.getElementById('editSvcDomain');
    const editDomainField = document.getElementById('editDomainField') || document.getElementById('editDomainGroup');
    const editFeedback = document.getElementById('editFeedback');
    const sourceType = document.getElementById('editServiceSourceType');
    const gitRepoURL = document.getElementById('editServiceGitRepoURL');
    const gitBranch = document.getElementById('editServiceGitBranch');
    const dockerfilePath = document.getElementById('editServiceDockerfilePath');
    const gitAccessToken = document.getElementById('editServiceGitAccessToken');

    if (editSvcName) editSvcName.value = name || '';
    if (editSvcExpose) editSvcExpose.value = (expose ? 'true' : 'false');
    if (editSvcDomain) editSvcDomain.value = domain || '';
    if (editDomainField) editDomainField.style.display = expose ? 'flex' : 'none';
    if (editFeedback) editFeedback.style.display = 'none';
    if (sourceType) sourceType.value = 'image';
    if (gitRepoURL) gitRepoURL.value = '';
    if (gitBranch) gitBranch.value = '';
    if (dockerfilePath) dockerfilePath.value = '';
    if (gitAccessToken) gitAccessToken.value = '';
    toggleEditGitSourceFields();

    openModal('editServiceModal');

    // Cargar el resto de la config (origen git) de forma asíncrona: el llamador solo
    // tiene name/expose/domain a mano (vienen de la tabla ya renderizada).
    try {
        const res = await apiFetch(`/api/services/${encodeURIComponent(name)}`);
        if (res.ok && res.data) {
            if (sourceType) sourceType.value = res.data.sourceType || 'image';
            if (gitRepoURL) gitRepoURL.value = res.data.gitRepoUrl || '';
            if (gitBranch) gitBranch.value = res.data.gitBranch || '';
            if (dockerfilePath) dockerfilePath.value = res.data.dockerfilePath || '';
            toggleEditGitSourceFields();
        }
    } catch (e) {
        console.debug('No se pudo precargar el origen git del servicio:', e);
    }
}

export function closeEditServiceModal() {
    closeModal('editServiceModal');
}

export async function openHistoryModal(serviceName, onReloadCallback) {
    ensureServicesModalsMounted();
    const historyServiceName = document.getElementById('historyServiceName');
    const historyTableBody = document.getElementById('historyTableBody');
    if (historyServiceName) historyServiceName.textContent = serviceName;
    if (historyTableBody) {
        historyTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">Consultando historial de '${escapeHtml(serviceName)}'...</td></tr>`;
    }
    openModal('historyModal');

    try {
        const srv = state.selectedServerName || '';
        const res = await apiFetch(`/api/services/history?name=${encodeURIComponent(serviceName)}&server=${encodeURIComponent(srv)}`);
        if (!res.ok) {
            if (historyTableBody) {
                historyTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty" style="color:var(--status-offline);">Error al cargar historial: ${escapeHtml(res.error)}</td></tr>`;
            }
            return;
        }

        const list = res.data || [];
        if (list.length === 0) {
            if (historyTableBody) {
                historyTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty">No hay snapshots históricos registrados aún para esta app.</td></tr>`;
            }
            return;
        }

        if (historyTableBody) {
            historyTableBody.innerHTML = '';
            list.forEach(rec => {
                const tr = document.createElement('tr');
                const dateStr = rec.createdAt ? new Date(rec.createdAt).toLocaleString('es-ES', { dateStyle: 'short', timeStyle: 'short' }) : '—';
                tr.innerHTML = `
                    <td>
                        <span style="font-size:0.75rem; color:var(--text-secondary); font-family:var(--font-mono);">${escapeHtml(dateStr)}</span>
                        ${rec.status === 'active' ? '<br><span class="t-badge" style="background:rgba(16,185,129,0.15); color:#10b981; font-size:0.65rem;">ACTIVA</span>' : ''}
                    </td>
                    <td>
                        <span style="font-family:var(--font-mono); font-size:0.78rem; font-weight:600; color:var(--text-primary);" title="${escapeHtml(rec.imageSource || '')}">${escapeHtml(rec.imageSource || '—')}</span>
                    </td>
                    <td>
                        <span style="font-size:0.75rem; color:var(--text-secondary); font-family:var(--font-mono);">${escapeHtml(rec.domain || 'Puerto ' + rec.port)}</span>
                    </td>
                    <td style="text-align:center;">
                        <button type="button" class="mini-btn mini-btn-accent btn-rollback-rec" data-id="${rec.id}" title="Restaurar esta versión">
                            🔄 Rollback
                        </button>
                    </td>
                `;

                const btnRollback = tr.querySelector('.btn-rollback-rec');
                if (btnRollback) {
                    btnRollback.addEventListener('click', async () => {
                        await rollbackToVersion(rec.id, serviceName, onReloadCallback, btnRollback);
                    });
                }

                historyTableBody.appendChild(tr);
            });
        }
    } catch (err) {
        if (historyTableBody) {
            historyTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty" style="color:var(--status-offline);">Fallo de conexión: ${escapeHtml(err.message)}</td></tr>`;
        }
    }
}

export function closeHistoryModal() {
    closeModal('historyModal');
}

export async function rollbackToVersion(recordId, serviceName, onReloadCallback, btnElement) {
    if (!recordId) return;
    if (!confirm(`¿Deseas restaurar la versión #${recordId} de '${serviceName}' en Docker Swarm?`)) return;

    if (btnElement) {
        btnElement.disabled = true;
        btnElement.textContent = '⏳...';
    }
    showToast(`Ejecutando rollback de '${serviceName}' a versión #${recordId}...`, 'info');

    try {
        const srv = state.selectedServerName || '';
        const res = await apiFetch('/api/services/rollback-version', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: recordId, server: srv })
        });

        if (!res.ok) {
            showToast(`Error en rollback: ${res.error}`, 'error');
            return;
        }

        showToast(`¡Servicio '${serviceName}' restaurado con éxito!`, 'success');
        closeHistoryModal();
        if (onReloadCallback && state.selectedServerName) {
            onReloadCallback(state.selectedServerName);
        } else if (state.selectedServerName) {
            const { refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
            refreshServerTelemetry(state.selectedServerName, false);
        }
    } catch (err) {
        showToast(`Fallo en rollback: ${err.message}`, 'error');
    } finally {
        if (btnElement) {
            btnElement.disabled = false;
            btnElement.textContent = '🔄 Rollback';
        }
    }
}

export function setupServicesEvents(onReloadStatus) {
    ensureServicesModalsMounted();

    const btnGlobalDeploy = document.getElementById('btnGlobalDeploy');
    const btnOpenDeployModal = document.getElementById('btnOpenDeployModal');
    const btnZeroStateDeployApp = document.getElementById('btnZeroStateDeployApp');
    const btnCloseDeployModal = document.getElementById('btnCloseDeployModal');
    const btnCancelDeploy = document.getElementById('btnCancelDeploy');
    const btnCloseEditServiceModal = document.getElementById('btnCloseEditServiceModal');
    const btnCancelEditService = document.getElementById('btnCancelEditService');
    const btnCloseHistoryModal = document.getElementById('btnCloseHistoryModal');
    const btnCancelHistory = document.getElementById('btnCancelHistory');
    const btnEditServiceOpenEnv = document.getElementById('btnEditServiceOpenEnv');
    const serviceSearchInput = document.getElementById('serviceSearchInput');
    const deployForm = document.getElementById('formDeploy') || document.getElementById('deployForm');
    const editServiceForm = document.getElementById('formEditService') || document.getElementById('editServiceForm');
    const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
    const editDomainField = document.getElementById('editDomainField') || document.getElementById('editDomainGroup');

    if (btnGlobalDeploy) btnGlobalDeploy.addEventListener('click', openDeployModal);
    if (btnOpenDeployModal) btnOpenDeployModal.addEventListener('click', openDeployModal);
    if (btnZeroStateDeployApp) btnZeroStateDeployApp.addEventListener('click', openDeployModal);
    if (btnCloseDeployModal) btnCloseDeployModal.addEventListener('click', closeDeployModal);
    if (btnCancelDeploy) btnCancelDeploy.addEventListener('click', closeDeployModal);
    if (btnCloseEditServiceModal) btnCloseEditServiceModal.addEventListener('click', closeEditServiceModal);
    if (btnCancelEditService) btnCancelEditService.addEventListener('click', closeEditServiceModal);
    if (btnCloseHistoryModal) btnCloseHistoryModal.addEventListener('click', closeHistoryModal);
    if (btnCancelHistory) btnCancelHistory.addEventListener('click', closeHistoryModal);

    const editServiceSourceType = document.getElementById('editServiceSourceType');
    if (editServiceSourceType) editServiceSourceType.addEventListener('change', toggleEditGitSourceFields);

    const btnCloseBuildLogsModal = document.getElementById('btnCloseBuildLogsModal');
    const btnCloseBuildLogs2 = document.getElementById('btnCloseBuildLogs2');
    const closeBuildLogs = () => closeModal('buildLogsModal');
    if (btnCloseBuildLogsModal) btnCloseBuildLogsModal.addEventListener('click', closeBuildLogs);
    if (btnCloseBuildLogs2) btnCloseBuildLogs2.addEventListener('click', closeBuildLogs);

    if (btnEditServiceOpenEnv) {
        btnEditServiceOpenEnv.addEventListener('click', async () => {
            const editSvcName = document.getElementById('editServiceName') || document.getElementById('editSvcName');
            const name = editSvcName ? editSvcName.value.trim() : '';
            if (name) {
                closeEditServiceModal();
                const { openEnvModal } = await import('/components/env/env.js');
                openEnvModal(name);
            }
        });
    }

    const servicesFilterInput = document.getElementById('servicesFilterInput');
    if (servicesFilterInput) {
        servicesFilterInput.addEventListener('input', () => {
            renderMasterServicesTable(state.swarmServicesCache, state.swarmDatabasesCache, state.currentServiceLinks);
        });
    }

    if (serviceSearchInput) {
        serviceSearchInput.addEventListener('input', () => {
            const q = serviceSearchInput.value.toLowerCase().trim();
            const filtered = (state.currentHostServices || []).filter(s =>
                (s.name || '').toLowerCase().includes(q) ||
                (s.description || '').toLowerCase().includes(q)
            );
            renderServicesTable(filtered);
        });
    }

    if (editSvcExpose && editDomainField) {
        editSvcExpose.addEventListener('change', () => {
            const isPub = (editSvcExpose.value === 'true' || editSvcExpose.checked);
            editDomainField.style.display = isPub ? 'flex' : 'none';
        });
    }

    if (deployForm) {
        deployForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const depName = document.getElementById('depName');
            const depImage = document.getElementById('depImage');
            const depPort = document.getElementById('depPort');
            const depDomain = document.getElementById('depDomain');
            const depPreHook = document.getElementById('depPreHook');
            const depAutoMigrate = document.getElementById('depAutoMigrate');
            const btnSubmitDeploy = document.getElementById('btnSubmitDeploy');
            const deployFeedback = document.getElementById('deployFeedback');

            const payload = {
                name: depName ? depName.value.trim() : '',
                imageSource: depImage ? depImage.value.trim() : '',
                port: parseInt(depPort ? depPort.value || '80' : '80', 10),
                domain: depDomain ? depDomain.value.trim() : '',
                preDeployHook: depPreHook ? depPreHook.value.trim() : '',
                autoMigrate: depAutoMigrate ? depAutoMigrate.checked : false,
                server: state.selectedServerName || ''
            };

            if (btnSubmitDeploy) {
                btnSubmitDeploy.disabled = true;
                btnSubmitDeploy.textContent = '🚀 Desplegando...';
            }
            if (deployFeedback) {
                deployFeedback.style.display = 'block';
                deployFeedback.innerHTML = `<span style="color:var(--status-warning);">⏳ Desplegando servicio en Swarm...</span>`;
            }

            try {
                const srv = state.selectedServerName || '';
                const res = await apiFetch(`/api/services?server=${encodeURIComponent(srv)}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    if (deployFeedback) {
                        deployFeedback.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    showToast(`Error al desplegar: ${res.error}`, 'error');
                    sendDesktopNotification('Fallo de Despliegue', `Error al desplegar '${payload.name}': ${res.error}`);
                    return;
                }

                showToast(`¡Servicio '${payload.name}' desplegado con éxito!`, 'success');
                sendDesktopNotification('Despliegue Exitoso', `El servicio '${payload.name}' se encuentra activo.`);
                closeDeployModal();
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
                } else if (state.selectedServerName) {
                    const { refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
                    refreshServerTelemetry(state.selectedServerName, false);
                }
            } finally {
                if (btnSubmitDeploy) {
                    btnSubmitDeploy.disabled = false;
                    btnSubmitDeploy.textContent = '🚀 Desplegar Servicio';
                }
            }
        });
    }

    if (editServiceForm) {
        editServiceForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const editSvcName = document.getElementById('editServiceName') || document.getElementById('editSvcName');
            const editSvcDomain = document.getElementById('editServiceDomain') || document.getElementById('editSvcDomain');
            const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
            const editFeedback = document.getElementById('editFeedback');
            const btnSubmitEdit = document.getElementById('btnSubmitEditService') || document.getElementById('btnSubmitEdit');
            const sourceType = document.getElementById('editServiceSourceType');
            const gitRepoURL = document.getElementById('editServiceGitRepoURL');
            const gitBranch = document.getElementById('editServiceGitBranch');
            const dockerfilePath = document.getElementById('editServiceDockerfilePath');
            const gitAccessToken = document.getElementById('editServiceGitAccessToken');

            const isExpose = editSvcExpose ? (editSvcExpose.value === 'true' || editSvcExpose.checked) : false;
            const payload = {
                name: editSvcName ? editSvcName.value.trim() : '',
                domain: editSvcDomain ? editSvcDomain.value.trim() : '',
                expose: isExpose,
                server: state.selectedServerName || '',
                sourceType: sourceType ? sourceType.value : 'image',
                gitRepoUrl: gitRepoURL ? gitRepoURL.value.trim() : '',
                gitBranch: gitBranch ? gitBranch.value.trim() : '',
                dockerfilePath: dockerfilePath ? dockerfilePath.value.trim() : '',
                gitAccessToken: gitAccessToken ? gitAccessToken.value : ''
            };

            if (btnSubmitEdit) btnSubmitEdit.disabled = true;

            try {
                const srv = state.selectedServerName || '';
                const res = await apiFetch(`/api/services/${encodeURIComponent(payload.name)}?server=${encodeURIComponent(srv)}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (!res.ok) {
                    if (editFeedback) {
                        editFeedback.style.display = 'block';
                        editFeedback.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    showToast(`Error al actualizar servicio: ${res.error}`, 'error');
                    return;
                }
                showToast(`Servicio '${payload.name}' actualizado.`, 'success');
                closeEditServiceModal();
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
                } else if (state.selectedServerName) {
                    const { refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
                    refreshServerTelemetry(state.selectedServerName, false);
                }
            } finally {
                if (btnSubmitEdit) btnSubmitEdit.disabled = false;
            }
        });
    }
}

export const setupServicesListeners = setupServicesEvents;

// ─── Metrics Modal ──────────────────────────────────────────────────────────

function buildSparklineSVG(values, color) {
    if (!values || values.length === 0) return '<text x="50%" y="50%" text-anchor="middle" fill="#6b7280" font-size="10">Sin datos</text>';
    const W = 240;
    const H = 48;
    const min = Math.min(...values);
    const max = Math.max(...values);
    const range = max - min || 1;
    const pts = values.map((v, i) => {
        const x = (i / (values.length - 1)) * W;
        const y = H - ((v - min) / range) * (H - 4) - 2;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
    const lastVal = values[values.length - 1];
    const lastY = H - ((lastVal - min) / range) * (H - 4) - 2;
    return `
        <polyline points="${pts}" fill="none" stroke="${color}" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/>
        <circle cx="${W}" cy="${lastY.toFixed(1)}" r="2.5" fill="${color}"/>
    `;
}

export async function openServiceMetricsModal(serviceName) {
    if (!serviceName) return;

    let overlay = document.getElementById('serviceMetricsModal');
    if (!overlay) {
        overlay = document.createElement('div');
        overlay.id = 'serviceMetricsModal';
        overlay.className = 't-modal-overlay';
        overlay.innerHTML = `
            <div class="t-modal-card metrics-modal-card" role="dialog" aria-modal="true">
                <div class="t-modal-header">
                    <span class="t-modal-title">📊 Métricas — <span id="metricsModalName"></span></span>
                    <button type="button" class="t-close-btn" id="btnCloseMetricsModal" aria-label="Cerrar">✕</button>
                </div>
                <div id="metricsModalBody" class="metrics-grid">
                    <div style="grid-column:1/-1; text-align:center; padding:32px; color:var(--text-muted);">⏳ Cargando métricas...</div>
                </div>
                <div class="t-modal-footer" style="padding:12px 20px; border-top:1px solid var(--border-subtle); display:flex; justify-content:flex-end; gap:8px;">
                    <select id="metricsRangeSelect" style="background:var(--surface-input); border:1px solid var(--border-subtle); color:var(--text-secondary); padding:4px 8px; border-radius:var(--radius-sm); font-size:0.80rem;">
                        <option value="1h">Última 1h</option>
                        <option value="6h">Últimas 6h</option>
                        <option value="24h">Últimas 24h</option>
                    </select>
                    <button type="button" class="t-btn t-btn-secondary" id="btnCloseMetricsModal2">Cerrar</button>
                </div>
            </div>
        `;
        document.body.appendChild(overlay);

        document.getElementById('btnCloseMetricsModal').onclick = () => { overlay.style.display = 'none'; };
        document.getElementById('btnCloseMetricsModal2').onclick = () => { overlay.style.display = 'none'; };
        overlay.addEventListener('click', (e) => { if (e.target === overlay) overlay.style.display = 'none'; });
        document.getElementById('metricsRangeSelect').addEventListener('change', () => {
            const name = document.getElementById('metricsModalName').textContent;
            const range = document.getElementById('metricsRangeSelect').value;
            loadMetrics(name, range);
        });
    }

    const nameEl = document.getElementById('metricsModalName');
    const body = document.getElementById('metricsModalBody');
    if (nameEl) nameEl.textContent = serviceName;
    if (body) body.innerHTML = '<div style="grid-column:1/-1; text-align:center; padding:32px; color:var(--text-muted);">⏳ Cargando métricas...</div>';
    overlay.style.display = 'flex';

    await loadMetrics(serviceName, document.getElementById('metricsRangeSelect')?.value || '1h');
}

async function loadMetrics(serviceName, range) {
    const body = document.getElementById('metricsModalBody');
    if (!body) return;
    try {
        const res = await apiFetch(`/api/observability/metrics?service=${encodeURIComponent(serviceName)}&range=${encodeURIComponent(range)}&server=${encodeURIComponent(state.selectedServerName || '')}`);
        if (!res.ok || !res.data) {
            body.innerHTML = '<div style="grid-column:1/-1; text-align:center; padding:32px; color:var(--status-offline);">No se pudieron cargar las métricas.</div>';
            return;
        }
        const pts = res.data.points || [];
        const cpuVals    = pts.map(p => parseFloat(p.cpu)    || 0);
        const memVals    = pts.map(p => parseFloat(p.memory)  || 0);
        const netVals    = pts.map(p => parseFloat(p.network) || 0);
        const diskVals   = pts.map(p => parseFloat(p.disk)    || 0);

        const lastCpu  = cpuVals.length  ? cpuVals[cpuVals.length - 1].toFixed(1)   : '—';
        const lastMem  = memVals.length  ? memVals[memVals.length - 1].toFixed(1)   : '—';
        const lastNet  = netVals.length  ? netVals[netVals.length - 1].toFixed(1)   : '—';
        const lastDisk = diskVals.length ? diskVals[diskVals.length - 1].toFixed(1) : '—';

        const charts = [
            { label: 'CPU', unit: '%',   vals: cpuVals,  last: lastCpu,  color: '#818cf8' },
            { label: 'Memoria', unit: 'MB', vals: memVals, last: lastMem,  color: '#34d399' },
            { label: 'Red',     unit: 'KB/s', vals: netVals, last: lastNet, color: '#fbbf24' },
            { label: 'Disco',   unit: 'MB', vals: diskVals, last: lastDisk, color: '#f87171' },
        ];

        body.innerHTML = charts.map(c => `
            <div class="metric-chart-block">
                <div class="metric-chart-label">
                    <span>${c.label}</span>
                    <span style="font-size:0.68rem; color:var(--text-muted); font-weight:400;">${pts.length} pts</span>
                </div>
                <div class="metric-chart-value">${c.last}<span style="font-size:0.70rem; font-weight:400; color:var(--text-muted); margin-left:3px;">${c.unit}</span></div>
                <svg class="metric-sparkline" viewBox="0 0 240 48" preserveAspectRatio="none">
                    ${buildSparklineSVG(c.vals, c.color)}
                </svg>
            </div>
        `).join('');
    } catch (err) {
        body.innerHTML = `<div style="grid-column:1/-1; text-align:center; padding:32px; color:var(--status-offline);">Error: ${escapeHtml(String(err))}</div>`;
    }
}

