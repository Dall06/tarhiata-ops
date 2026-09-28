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

export function ensureDatabasesModalsMounted() {
    if (document.getElementById('dbModal') && document.getElementById('backupsModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Crear Base de Datos -->
<div class="t-modal-overlay" id="dbModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card">
        <div class="t-modal-header">
            <div class="t-modal-title">
                <span>🗄️ Crear Base de Datos</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseDBModal" aria-label="Cerrar">×</button>
        </div>
        <form id="formDeployDB">
            <div class="form-body">
                <div class="mode-selector" style="margin: 0 0 14px 0;">
                    <button type="button" class="mode-btn active" id="tabDBLocal">
                        <span>Local en Swarm</span>
                    </button>
                    <button type="button" class="mode-btn" id="tabDBNode">
                        <span>Nodo Dedicado</span>
                    </button>
                    <button type="button" class="mode-btn" id="tabDBExternal">
                        <span>URL Externa</span>
                    </button>
                </div>

                <div class="form-row-grid">
                    <div class="form-field">
                        <label for="dbName">Nombre del Servicio de BD</label>
                        <input type="text" id="dbName" class="t-input" placeholder="ej: bd-clientes" required>
                    </div>
                    <div class="form-field">
                        <label for="dbEngine">Motor de Base de Datos</label>
                        <select id="dbEngine" class="t-input">
                            <option value="postgres">PostgreSQL</option>
                            <option value="mongodb">MongoDB</option>
                            <option value="mysql">MySQL</option>
                            <option value="redis">Redis</option>
                            <option value="minio">MinIO (S3 Storage)</option>
                        </select>
                    </div>

                    <div class="form-field full-span" id="dbUrlField" style="display:none;">
                        <label for="dbExternalURL">URL de Conexión (Supabase, Neon, etc.)</label>
                        <input type="text" id="dbExternalURL" class="t-input" placeholder="ej: postgresql://usuario:clave@db.neon.tech:5432/bd">
                    </div>

                    <div class="form-field full-span" id="dbPathField">
                        <label for="dbVolumePath">Ruta Persistente en Disco (Volumen)</label>
                        <input type="text" id="dbVolumePath" class="t-input" value="/opt/data/db-storage">
                    </div>

                    <div class="form-field" id="dbPortField">
                        <label for="dbPort">Puerto Interno</label>
                        <input type="number" id="dbPort" class="t-input" value="5432">
                    </div>

                    <div class="form-field" id="dbTargetNodeField" style="display:none;">
                        <label for="dbTargetNode">Afinidad de Nodo Target</label>
                        <input type="text" id="dbTargetNode" class="t-input" placeholder="worker o hostname" value="worker">
                    </div>
                </div>
            </div>
            <div class="t-modal-footer">
                <button type="button" class="t-btn t-btn-secondary" id="btnCancelDB">Cancelar</button>
                <button type="submit" class="t-btn t-btn-primary" id="btnSubmitDB">
                    <span id="btnSubmitDBText">Crear Base de Datos</span>
                </button>
            </div>
        </form>
    </div>
</div>

<!-- Modal: Historial y Gestión de Backups / Snapshots -->
<div class="t-modal-overlay" id="backupsModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card" style="max-width: 840px; width: 92vw;">
        <div class="t-modal-header">
            <div>
                <h3 class="t-modal-title">📦 Historial de Snapshots y Backups</h3>
                <p class="t-modal-subtitle">Respaldos SQL, volcados binarios y archivos comprimidos almacenados en el servidor.</p>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseBackupsModal" aria-label="Cerrar">&times;</button>
        </div>
        <div class="form-body" style="padding: 16px 20px;">
            <div class="tactical-table-wrap" style="max-height: 420px; overflow-y: auto;">
                <table class="t-table">
                    <thead>
                        <tr>
                            <th>Target / Base de Datos</th>
                            <th>Motor</th>
                            <th>Archivo</th>
                            <th>Tamaño</th>
                            <th>Fecha</th>
                            <th>Acciones</th>
                        </tr>
                    </thead>
                    <tbody id="backupsTableBody">
                        <tr>
                            <td colspan="6" class="t-td-empty">Consultando copias de seguridad...</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        <div class="t-modal-footer">
            <button type="button" class="t-btn t-btn-secondary" id="btnDismissBackupsModal">Cerrar</button>
            <button type="button" class="t-btn t-btn-primary" id="btnRefreshBackupsModal">🔄 Actualizar</button>
        </div>
    </div>
</div>`;
    while (div.firstElementChild) {
        document.body.appendChild(div.firstElementChild);
    }
}

export function setDBMode(mode) {
    state.dbDeployMode = mode;
    const tabDBLocal = document.getElementById('tabDBLocal') || document.getElementById('tabDBSingle');
    const tabDBNode = document.getElementById('tabDBNode') || document.getElementById('tabDBMulti');
    const tabDBExternal = document.getElementById('tabDBExternal');

    const dbUrlField = document.getElementById('dbUrlField') || document.getElementById('dbExternalGroup');
    const dbPathField = document.getElementById('dbPathField') || document.getElementById('dbLocalGroup');
    const dbPortField = document.getElementById('dbPortField');
    const dbTargetNodeField = document.getElementById('dbTargetNodeField');

    if (tabDBLocal) tabDBLocal.classList.toggle('active', mode === 'single-node' || mode === 'local');
    if (tabDBNode) tabDBNode.classList.toggle('active', mode === 'multi-node' || mode === 'node');
    if (tabDBExternal) tabDBExternal.classList.toggle('active', mode === 'external');

    if (mode === 'external') {
        if (dbUrlField) dbUrlField.style.display = 'flex';
        if (dbPathField) dbPathField.style.display = 'none';
        if (dbPortField) dbPortField.style.display = 'none';
        if (dbTargetNodeField) dbTargetNodeField.style.display = 'none';
        return;
    }
    if (mode === 'node' || mode === 'multi-node') {
        if (dbUrlField) dbUrlField.style.display = 'none';
        if (dbPathField) dbPathField.style.display = 'flex';
        if (dbPortField) dbPortField.style.display = 'flex';
        if (dbTargetNodeField) dbTargetNodeField.style.display = 'flex';
        return;
    }
    // local / single-node
    if (dbUrlField) dbUrlField.style.display = 'none';
    if (dbPathField) dbPathField.style.display = 'flex';
    if (dbPortField) dbPortField.style.display = 'flex';
    if (dbTargetNodeField) dbTargetNodeField.style.display = 'none';
}

export function openDeployDBModal() {
    ensureDatabasesModalsMounted();
    const dbForm = document.getElementById('formDeployDB') || document.getElementById('dbForm');
    const dbFeedback = document.getElementById('dbFeedback');
    if (dbForm) dbForm.reset();
    if (dbFeedback) dbFeedback.style.display = 'none';
    setDBMode('local');
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

        if (res.ok) {
            showToast(`¡Respaldo de '${name}' completado con éxito!`, 'success');
            await loadBackups();
            openModal('backupsModal');
        } else {
            showToast(`Error al respaldar base de datos: ${res.error}`, 'error');
        }
    } finally {
        if (btnElement) {
            btnElement.disabled = false;
            btnElement.textContent = '💾 Backup';
        }
    }
}

export async function loadBackups(onReloadStatus) {
    ensureDatabasesModalsMounted();
    const backupsTableBody = document.getElementById('backupsTableBody');
    if (!backupsTableBody) return;

    backupsTableBody.innerHTML = '<tr><td colspan="6" class="t-td-empty">Consultando copias de seguridad...</td></tr>';
    const res = await apiFetch(`/api/backups?server=${encodeURIComponent(state.selectedServerName || '')}`);
    if (!res.ok || !res.data || !Array.isArray(res.data) || res.data.length === 0) {
        backupsTableBody.innerHTML = '<tr><td colspan="6" class="t-td-empty">No se encontraron copias de seguridad registradas. Genera una desde tus bases de datos.</td></tr>';
        return;
    }

    backupsTableBody.innerHTML = '';
    res.data.forEach(b => {
        const tr = document.createElement('tr');
        const bId = b.id || b.ID;
        const targetName = b.targetName || b.TargetName || b.dbName || '—';
        const engine = b.engine || b.Engine || 'database';
        const filename = b.filename || b.Filename || b.fileName || '—';
        const sizeStr = b.sizeBytes || b.SizeBytes ? `${Math.round((b.sizeBytes || b.SizeBytes) / 1024)} KB` : (b.size || '—');
        const createdAt = b.createdAt || b.CreatedAt || '—';

        tr.innerHTML = `
            <td style="font-weight:700; color:#fff;">${escapeHtml(targetName)}</td>
            <td><span class="svc-pill svc-pill-active">${escapeHtml(engine)}</span></td>
            <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-muted); max-width:200px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="${escapeHtml(filename)}">${escapeHtml(filename)}</td>
            <td style="color:var(--text-secondary); font-size:0.8rem;">${escapeHtml(sizeStr)}</td>
            <td style="color:var(--text-muted); font-size:0.75rem;">${escapeHtml(createdAt)}</td>
            <td>
                <div style="display:flex; gap:6px; align-items:center;">
                    <a class="mini-btn btn-dl-backup" href="/api/backups/download?id=${bId}&file=${encodeURIComponent(filename)}&server=${encodeURIComponent(state.selectedServerName || '')}" download="${escapeHtml(filename)}" title="Descargar snapshot SQL/archivo" style="text-decoration:none;">
                        📥 Descargar
                    </a>
                    <button type="button" class="mini-btn btn-restore-backup" data-id="${bId}" data-target="${escapeHtml(targetName)}" title="Restaurar base de datos desde este snapshot" style="color:var(--accent-warning, #f59e0b);">
                        ♻️ Restaurar
                    </button>
                    <button type="button" class="mini-btn btn-del-backup" data-id="${bId}" data-file="${escapeHtml(filename)}" title="Eliminar respaldo" style="color:var(--status-offline);">
                        🗑️
                    </button>
                </div>
            </td>
        `;
        backupsTableBody.appendChild(tr);
    });

    // Bind restore events
    document.querySelectorAll('.btn-restore-backup').forEach(btn => {
        btn.addEventListener('click', async () => {
            const id = btn.getAttribute('data-id');
            const target = btn.getAttribute('data-target');
            if (!id) return;
            if (!confirm(`¿Restaurar la base de datos '${target}' desde este respaldo? ADVERTENCIA: Esta operación sobrescribirá los datos actuales con el contenido del snapshot.`)) return;
            btn.disabled = true;
            showToast(`Restaurando '${target}' desde snapshot...`, 'info');
            try {
                const res = await apiFetch(`/api/backups/restore?server=${encodeURIComponent(state.selectedServerName || '')}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ backupId: parseInt(id, 10), server: state.selectedServerName || '' })
                });
                if (res.ok) {
                    showToast(`¡Base de datos '${target}' restaurada con éxito!`, 'success');
                    if (onReloadStatus && state.selectedServerName) {
                        onReloadStatus(state.selectedServerName);
                    }
                } else {
                    showToast(`Error al restaurar: ${res.error}`, 'error');
                }
            } finally {
                btn.disabled = false;
            }
        });
    });

    // Bind delete events
    document.querySelectorAll('.btn-del-backup').forEach(btn => {
        btn.addEventListener('click', async () => {
            const id = btn.getAttribute('data-id');
            const file = btn.getAttribute('data-file');
            if (!id) return;
            if (!confirm(`¿Eliminar definitivamente el respaldo '${file}'?`)) return;
            btn.disabled = true;
            try {
                const res = await apiFetch(`/api/backups?id=${encodeURIComponent(id)}&file=${encodeURIComponent(file)}&server=${encodeURIComponent(state.selectedServerName || '')}`, {
                    method: 'DELETE'
                });
                if (res.ok) {
                    showToast(`Respaldo eliminado`, 'info');
                    await loadBackups(onReloadStatus);
                } else {
                    showToast(`Error al eliminar: ${res.error}`, 'error');
                }
            } finally {
                btn.disabled = false;
            }
        });
    });
}

export function setupDatabaseEvents(onReloadStatus) {
    ensureDatabasesModalsMounted();

    const btnGlobalDB = document.getElementById('btnGlobalDB');
    const btnCloseDBModal = document.getElementById('btnCloseDBModal');
    const btnCancelDB = document.getElementById('btnCancelDB');
    const tabDBLocal = document.getElementById('tabDBLocal') || document.getElementById('tabDBSingle');
    const tabDBNode = document.getElementById('tabDBNode') || document.getElementById('tabDBMulti');
    const tabDBExternal = document.getElementById('tabDBExternal');
    const dbForm = document.getElementById('formDeployDB') || document.getElementById('dbForm');
    const btnCloseBackupsModal = document.getElementById('btnCloseBackupsModal');
    const btnDismissBackupsModal = document.getElementById('btnDismissBackupsModal');
    const btnRefreshBackupsModal = document.getElementById('btnRefreshBackupsModal');

    if (btnGlobalDB) btnGlobalDB.addEventListener('click', openDeployDBModal);
    if (btnCloseDBModal) btnCloseDBModal.addEventListener('click', closeDeployDBModal);
    if (btnCancelDB) btnCancelDB.addEventListener('click', closeDeployDBModal);
    if (btnCloseBackupsModal) btnCloseBackupsModal.addEventListener('click', () => closeModal('backupsModal'));
    if (btnDismissBackupsModal) btnDismissBackupsModal.addEventListener('click', () => closeModal('backupsModal'));
    if (btnRefreshBackupsModal) btnRefreshBackupsModal.addEventListener('click', () => loadBackups(onReloadStatus));

    if (tabDBLocal) tabDBLocal.addEventListener('click', () => setDBMode('local'));
    if (tabDBNode) tabDBNode.addEventListener('click', () => setDBMode('node'));
    if (tabDBExternal) tabDBExternal.addEventListener('click', () => setDBMode('external'));

    if (dbForm) {
        dbForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const dbName = document.getElementById('dbName');
            const dbEngine = document.getElementById('dbEngine');
            const dbExternalURL = document.getElementById('dbExternalURL');
            const dbVolumePath = document.getElementById('dbVolumePath');
            const dbPort = document.getElementById('dbPort');
            const dbTargetNode = document.getElementById('dbTargetNode');
            const dbFeedback = document.getElementById('dbFeedback');
            const btnSubmitDB = document.getElementById('btnSubmitDB');

            const payload = {
                name: dbName ? dbName.value.trim() : '',
                engine: dbEngine ? dbEngine.value : 'postgres',
                deployType: state.dbDeployMode || 'local',
                externalUrl: dbExternalURL ? dbExternalURL.value.trim() : '',
                volumeHostPath: dbVolumePath ? dbVolumePath.value.trim() : '',
                internalPort: dbPort ? parseInt(dbPort.value || '5432', 10) : 5432,
                targetNode: dbTargetNode ? dbTargetNode.value.trim() : '',
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
                    showToast(`Error al crear BD: ${res.error}`, 'error');
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

export const setupDatabasesListeners = setupDatabaseEvents;

