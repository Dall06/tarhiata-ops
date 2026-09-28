/**
 * Tarhiata Cloud Studio — Database Component (opt/databases)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml, getDefaultPort, copyToClipboard } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch } from '/pkg/apiclient/api.js';

export function renderDatabaseCards(databases, callbacks = {}) {
    const swarmDatabasesCardsGrid = document.getElementById('swarmDatabasesCardsGrid');
    const swarmDatabasesEmpty = document.getElementById('swarmDatabasesEmpty');
    if (!swarmDatabasesCardsGrid) return;

    swarmDatabasesCardsGrid.innerHTML = '';

    if (!databases || databases.length === 0) {
        if (swarmDatabasesEmpty) swarmDatabasesEmpty.style.display = 'flex';
        return;
    }
    if (swarmDatabasesEmpty) swarmDatabasesEmpty.style.display = 'none';

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

    // Wire action buttons
    document.querySelectorAll('.btn-logs-db').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onOpenLogs) callbacks.onOpenLogs(name);
        });
    });

    document.querySelectorAll('.btn-restart-db').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onRestart) callbacks.onRestart(name, btn);
        });
    });

    document.querySelectorAll('.btn-backup-db').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            const engine = btn.getAttribute('data-engine');
            if (name) triggerDatabaseBackup(name, engine, btn);
        });
    });

    document.querySelectorAll('.btn-vol-db').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onOpenVolume) callbacks.onOpenVolume('/opt/data/db-storage');
        });
    });

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
            const res = await apiFetch(`/api/databases?name=${encodeURIComponent(name)}&server=${encodeURIComponent(state.selectedServerName || '')}`, { method: 'DELETE' });
            if (res.ok) {
                showToast(`Base de datos '${name}' eliminada.`, 'info');
                if (callbacks.onReloadStatus && state.selectedServerName) {
                    callbacks.onReloadStatus(state.selectedServerName);
                }
            }
        });
    });
}

export function setDBMode(mode) {
    state.dbDeployMode = mode;
    const tabDBSingle = document.getElementById('tabDBSingle');
    const tabDBMulti = document.getElementById('tabDBMulti');
    const tabDBExternal = document.getElementById('tabDBExternal');
    const dbExternalGroup = document.getElementById('dbExternalGroup');
    const dbLocalGroup = document.getElementById('dbLocalGroup');

    if (tabDBSingle) tabDBSingle.classList.toggle('active', mode === 'single-node');
    if (tabDBMulti) tabDBMulti.classList.toggle('active', mode === 'multi-node');
    if (tabDBExternal) tabDBExternal.classList.toggle('active', mode === 'external');

    if (mode === 'external') {
        if (dbExternalGroup) dbExternalGroup.style.display = 'flex';
        if (dbLocalGroup) dbLocalGroup.style.display = 'none';
        return;
    }
    if (dbExternalGroup) dbExternalGroup.style.display = 'none';
    if (dbLocalGroup) dbLocalGroup.style.display = 'flex';
}

export function openDeployDBModal() {
    const dbForm = document.getElementById('dbForm');
    const dbFeedback = document.getElementById('dbFeedback');
    if (dbForm) dbForm.reset();
    if (dbFeedback) dbFeedback.style.display = 'none';
    setDBMode('single-node');
    openModal('dbModal');
}

export function closeDeployDBModal() {
    closeModal('dbModal');
}

export async function triggerDatabaseBackup(name, engine, btnElement) {
    if (!name) return;
    if (btnElement) {
        btnElement.disabled = true;
        btnElement.textContent = '⏳ Snapshot...';
    }
    showToast(`Generando respaldo (dump) de la base de datos '${name}'...`, 'info');

    try {
        const res = await apiFetch('/api/databases/backup', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, engine: engine || 'postgres', server: state.selectedServerName || '' })
        });

        if (res.ok && res.data) {
            showToast(`¡Respaldo de '${name}' completado con éxito!`, 'success');
            await loadBackups();
            openModal('backupsModal');
        }
    } finally {
        if (btnElement) {
            btnElement.disabled = false;
            btnElement.textContent = '💾 Backup';
        }
    }
}

export async function loadBackups() {
    const backupsTableBody = document.getElementById('backupsTableBody');
    const backupsEmptyNotice = document.getElementById('backupsEmptyNotice');
    if (!backupsTableBody) return;

    backupsTableBody.innerHTML = '<tr><td colspan="5" class="t-td-empty">Consultando historial de respaldos...</td></tr>';
    const res = await apiFetch(`/api/backups?server=${encodeURIComponent(state.selectedServerName || '')}`);
    if (!res.ok || !res.data || !Array.isArray(res.data) || res.data.length === 0) {
        backupsTableBody.innerHTML = '';
        if (backupsEmptyNotice) backupsEmptyNotice.style.display = 'block';
        return;
    }

    if (backupsEmptyNotice) backupsEmptyNotice.style.display = 'none';
    backupsTableBody.innerHTML = '';

    res.data.forEach(b => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td><strong style="font-family:var(--font-mono); font-size:0.78rem;">${escapeHtml(b.dbName)}</strong></td>
            <td><span class="t-badge" style="font-size:0.72rem;">${escapeHtml(b.engine)}</span></td>
            <td><code style="font-size:0.75rem; color:var(--text-secondary);">${escapeHtml(b.fileName)}</code></td>
            <td style="font-size:0.75rem; color:var(--text-secondary);">${escapeHtml(b.createdAt || '—')}</td>
            <td>
                <a href="/api/backups/download?file=${encodeURIComponent(b.fileName)}&server=${encodeURIComponent(state.selectedServerName || '')}" class="mini-btn mini-btn-accent" style="text-decoration:none;" download>
                    ⬇ Descargar
                </a>
            </td>
        `;
        backupsTableBody.appendChild(tr);
    });
}

export function setupDatabaseEvents(onReloadStatus) {
    const btnGlobalDB = document.getElementById('btnGlobalDB');
    const btnCloseDBModal = document.getElementById('btnCloseDBModal');
    const tabDBSingle = document.getElementById('tabDBSingle');
    const tabDBMulti = document.getElementById('tabDBMulti');
    const tabDBExternal = document.getElementById('tabDBExternal');
    const dbForm = document.getElementById('dbForm');
    const btnCloseBackupsModal = document.getElementById('btnCloseBackupsModal');
    const btnCloseBackupsModalBottom = document.getElementById('btnCloseBackupsModalBottom');

    if (btnGlobalDB) btnGlobalDB.addEventListener('click', openDeployDBModal);
    if (btnCloseDBModal) btnCloseDBModal.addEventListener('click', closeDeployDBModal);
    if (btnCloseBackupsModal) btnCloseBackupsModal.addEventListener('click', () => closeModal('backupsModal'));
    if (btnCloseBackupsModalBottom) btnCloseBackupsModalBottom.addEventListener('click', () => closeModal('backupsModal'));

    if (tabDBSingle) tabDBSingle.addEventListener('click', () => setDBMode('single-node'));
    if (tabDBMulti) tabDBMulti.addEventListener('click', () => setDBMode('multi-node'));
    if (tabDBExternal) tabDBExternal.addEventListener('click', () => setDBMode('external'));

    if (dbForm) {
        dbForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const dbName = document.getElementById('dbName');
            const dbEngine = document.getElementById('dbEngine');
            const dbExternalURL = document.getElementById('dbExternalURL');
            const dbVolumePath = document.getElementById('dbVolumePath');
            const dbFeedback = document.getElementById('dbFeedback');
            const btnSubmitDB = document.getElementById('btnSubmitDB');

            const payload = {
                name: dbName ? dbName.value.trim() : '',
                engine: dbEngine ? dbEngine.value : 'postgres',
                deployType: state.dbDeployMode,
                externalUrl: dbExternalURL ? dbExternalURL.value.trim() : '',
                volumeHostPath: dbVolumePath ? dbVolumePath.value.trim() : '',
                server: state.selectedServerName || ''
            };

            if (btnSubmitDB) {
                btnSubmitDB.disabled = true;
                btnSubmitDB.textContent = '🚀 Creando Base de Datos...';
            }
            if (dbFeedback) {
                dbFeedback.style.display = 'block';
                dbFeedback.innerHTML = `<span style="color:var(--status-warning);">⏳ Aprovisionando base de datos...</span>`;
            }

            try {
                const res = await apiFetch('/api/databases', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (!res.ok) {
                    if (dbFeedback) {
                        dbFeedback.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    return;
                }
                showToast(`Base de datos '${payload.name}' creada exitosamente.`, 'success');
                closeDeployDBModal();
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
                }
            } finally {
                if (btnSubmitDB) {
                    btnSubmitDB.disabled = false;
                    btnSubmitDB.textContent = 'Crear Base de Datos';
                }
            }
        });
    }
}
