/**
 * Tarhiata Cloud Studio — Swarm Nodes Component (opt/nodes)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml, copyToClipboard } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch, consumeNDJSONStream } from '/pkg/apiclient/api.js';

export function renderNodesTable(nodes, callbacks = {}) {
    const swarmNodesTableBody = document.getElementById('swarmNodesTableBody');
    if (!swarmNodesTableBody) return;
    if (!nodes || nodes.length === 0) {
        swarmNodesTableBody.innerHTML = `<tr><td colspan="7" class="t-td-empty">No se detectaron nodos adicionales.</td></tr>`;
        return;
    }

    swarmNodesTableBody.innerHTML = '';
    nodes.forEach(node => {
        const tr = document.createElement('tr');
        const isLeader = node.managerStatus && node.managerStatus.toLowerCase().includes('leader');
        const isManager = isLeader || (node.managerStatus && node.managerStatus.toLowerCase().includes('reachable'));
        const avail = (node.availability || 'active').toLowerCase();
        const isDrain = avail === 'drain';
        const nodeHost = (node.hostname || '').toLowerCase();

        const assignedList = [];
        (state.swarmServicesCache || []).forEach(s => {
            const tgt = (s.targetNode || '').toLowerCase();
            if (!tgt || tgt === 'all' || tgt === nodeHost || (isManager && (tgt === 'manager' || tgt === 'lider' || tgt === 'leader'))) {
                assignedList.push({ name: s.name, type: 'app' });
            }
        });
        (state.swarmDatabasesCache || []).forEach(db => {
            const tgt = (db.targetNode || '').toLowerCase();
            if (!tgt || tgt === 'all' || tgt === nodeHost || (isManager && (tgt === 'manager' || tgt === 'lider' || tgt === 'leader'))) {
                assignedList.push({ name: db.name, type: 'db' });
            }
        });

        let assignedHtml = '<span style="color:var(--text-muted); font-size:0.75rem;">0 cargas activas</span>';
        if (assignedList.length > 0) {
            assignedHtml = `<span class="t-badge" style="background:rgba(99,102,241,0.12); color:#a5b4fc; border-color:rgba(99,102,241,0.25); font-weight:600; font-size:0.75rem;">⚡ ${assignedList.length} ${assignedList.length === 1 ? 'carga activa' : 'cargas activas'}</span>`;
        }

        let actionsHtml = '';
        if (isLeader) {
            actionsHtml = `<span style="color:var(--text-muted); font-size:0.75rem;">Líder de Clúster</span>`;
        } else {
            const availBtnHtml = isDrain
                ? `<button type="button" class="mini-btn btn-node-avail" data-id="${escapeHtml(node.id)}" data-avail="active" title="Activar nodo para recibir contenedores" style="color:var(--status-online);">▶️ Activar</button>`
                : `<button type="button" class="mini-btn btn-node-avail" data-id="${escapeHtml(node.id)}" data-avail="drain" title="Drenar nodo (desalojar tareas)" style="color:var(--accent-warning, #f59e0b);">⏸️ Drenar</button>`;

            const deleteBtnHtml = `<button type="button" class="mini-btn btn-node-delete" data-id="${escapeHtml(node.id)}" data-hostname="${escapeHtml(node.hostname)}" title="Expulsar nodo del clúster" style="color:var(--status-offline);">🗑️ Expulsar</button>`;

            actionsHtml = `<div style="display:flex; gap:6px; align-items:center;">${availBtnHtml}${deleteBtnHtml}</div>`;
        }

        tr.innerHTML = `
            <td style="font-weight:700; color:#fff;">${escapeHtml(node.hostname)}</td>
            <td><span class="svc-pill ${node.status && node.status.toLowerCase() === 'ready' ? 'svc-pill-active' : ''}">${escapeHtml(node.status)}</span></td>
            <td style="color:${isDrain ? 'var(--accent-warning, #f59e0b)' : 'var(--text-muted)'};">${escapeHtml(node.availability)}</td>
            <td><span class="t-badge ${isLeader ? 't-badge-active' : ''}">${escapeHtml(node.managerStatus || 'Worker')}</span></td>
            <td>${assignedHtml}</td>
            <td style="color:var(--text-muted); font-family:var(--font-mono); font-size:0.75rem;">${escapeHtml(node.engineVersion || '—')}</td>
            <td>${actionsHtml}</td>
        `;
        swarmNodesTableBody.appendChild(tr);
    });

    document.querySelectorAll('.btn-node-avail').forEach(btn => {
        btn.addEventListener('click', async () => {
            const id = btn.getAttribute('data-id');
            const targetAvail = btn.getAttribute('data-avail');
            if (!id || !targetAvail) return;
            btn.disabled = true;
            try {
                const res = await apiFetch(`/api/nodes/update?server=${encodeURIComponent(state.selectedServerName || '')}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ id, availability: targetAvail })
                });
                if (res.ok) {
                    showToast(`Disponibilidad de nodo actualizada a '${targetAvail}'`, 'success');
                    if (callbacks.onReloadStatus && state.selectedServerName) {
                        callbacks.onReloadStatus(state.selectedServerName);
                    }
                }
            } finally {
                btn.disabled = false;
            }
        });
    });

    document.querySelectorAll('.btn-node-delete').forEach(btn => {
        btn.addEventListener('click', async () => {
            const id = btn.getAttribute('data-id');
            const hostname = btn.getAttribute('data-hostname') || id;
            if (!id) return;
            if (!confirm(`¿Estás seguro de expulsar el nodo '${hostname}' del clúster Swarm?`)) return;
            btn.disabled = true;
            try {
                const res = await apiFetch(`/api/nodes?id=${encodeURIComponent(id)}&server=${encodeURIComponent(state.selectedServerName || '')}`, {
                    method: 'DELETE'
                });
                if (res.ok) {
                    showToast(`Nodo '${hostname}' expulsado exitosamente del clúster`, 'success');
                    if (callbacks.onReloadStatus && state.selectedServerName) {
                        callbacks.onReloadStatus(state.selectedServerName);
                    }
                }
            } finally {
                btn.disabled = false;
            }
        });
    });
}

export function ensureNodesModalMounted() {
    if (document.getElementById('workerModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Aprovisionar Nodo Worker en la Nube -->
<div class="t-modal-overlay" id="workerModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card">
        <div class="t-modal-header">
            <div class="t-modal-title">
                <span>☁️ Aprovisionar Nodo Worker Cloud</span>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseWorkerModal" aria-label="Cerrar">×</button>
        </div>
        <form id="formWorker">
            <div class="form-body">
                <div class="fast-notice">
                    <span class="notice-icon">🚀</span>
                    <div>
                        <strong>Escalado Automático de Clúster</strong>
                        <p>Crea un VPS secundario en la nube y lo une automáticamente como nodo worker a tu clúster Swarm.</p>
                    </div>
                </div>

                <div class="form-row-grid">
                    <div class="form-field">
                        <label for="workerName">Nombre del Nodo</label>
                        <input type="text" id="workerName" class="t-input" placeholder="ej: worker-1" value="worker-1" required>
                    </div>

                    <div class="form-field">
                        <label for="workerProvider">Proveedor Cloud</label>
                        <select id="workerProvider" class="t-input">
                            <option value="vultr">Vultr Cloud Compute</option>
                            <option value="digitalocean">DigitalOcean Droplet</option>
                        </select>
                    </div>

                    <div class="form-field full-span">
                        <label for="workerApiKey">API Key del Proveedor Cloud</label>
                        <input type="password" id="workerApiKey" class="t-input" placeholder="Ingresa tu API Key de Vultr o DigitalOcean" required>
                    </div>

                    <div class="form-field">
                        <label for="workerRegion">Región del Datacenter</label>
                        <input type="text" id="workerRegion" class="t-input" placeholder="mex, nyc1, ewr" value="mex">
                    </div>

                    <div class="form-field">
                        <label for="workerPlan">Plan de Servidor</label>
                        <input type="text" id="workerPlan" class="t-input" placeholder="vc2-1c-1gb o s-1vcpu-1gb" value="vc2-1c-1gb">
                    </div>

                    <div class="form-field full-span">
                        <label for="workerLabel">Etiqueta de Carga</label>
                        <select id="workerLabel" class="t-input">
                            <option value="worker">Propósito General (worker)</option>
                            <option value="database">Dedicado a Bases de Datos (database)</option>
                            <option value="edge">Proxy / Edge (edge)</option>
                        </select>
                    </div>
                </div>

                <div id="workerLogsBox" class="t-diagnostic-box" style="display:none; max-height:160px; overflow-y:auto; margin-top:8px;">
                    <div id="workerLogsContent" style="font-size:0.75rem; color:var(--text-muted); line-height:1.5;"></div>
                </div>
            </div>
            <div class="t-modal-footer">
                <button type="button" class="t-btn t-btn-secondary" id="btnCancelWorker">Cancelar</button>
                <button type="submit" class="t-btn t-btn-primary" id="btnSubmitWorker">
                    <span id="btnSubmitWorkerText">Crear y Unir al Clúster</span>
                </button>
            </div>
        </form>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
}

export function openWorkerModal() {
    ensureNodesModalMounted();
    const workerForm = document.getElementById('formWorker') || document.getElementById('workerForm');
    const workerLogsBox = document.getElementById('workerLogsBox') || document.getElementById('workerTerminalContainer');
    const workerLogsContent = document.getElementById('workerLogsContent') || document.getElementById('workerStreamLog');
    if (workerForm) workerForm.reset();
    if (workerLogsContent) workerLogsContent.innerHTML = '';
    if (workerLogsBox) workerLogsBox.style.display = 'none';
    openModal('workerModal');
}

export function closeWorkerModal() {
    closeModal('workerModal');
}

export function setupNodesEvents(onReloadStatus) {
    ensureNodesModalMounted();

    const btnOpenWorkerModal = document.getElementById('btnOpenWorkerModal');
    const btnCloseWorkerModal = document.getElementById('btnCloseWorkerModal');
    const btnCancelWorker = document.getElementById('btnCancelWorker') || document.getElementById('btnCloseWorkerModalBottom');
    const btnCopyJoinToken = document.getElementById('btnCopyJoinToken');
    const workerForm = document.getElementById('formWorker') || document.getElementById('workerForm');

    if (btnOpenWorkerModal) btnOpenWorkerModal.addEventListener('click', openWorkerModal);
    if (btnCloseWorkerModal) btnCloseWorkerModal.addEventListener('click', closeWorkerModal);
    if (btnCancelWorker) btnCancelWorker.addEventListener('click', closeWorkerModal);

    if (btnCopyJoinToken) {
        btnCopyJoinToken.addEventListener('click', async () => {
            const srv = state.selectedServerName;
            const res = await apiFetch(`/api/nodes/join-token?server=${encodeURIComponent(srv || '')}`);
            if (res.ok && res.data && res.data.joinCommand) {
                await copyToClipboard(res.data.joinCommand);
                showToast('Comando docker swarm join copiado al portapapeles', 'success');
            } else {
                showToast(res.error || 'No se pudo obtener el token de Swarm', 'error');
            }
        });
    }

    if (workerForm) {
        workerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const workerNodeName = document.getElementById('workerName') || document.getElementById('workerNodeName');
            const workerProvider = document.getElementById('workerProvider');
            const workerApiKey = document.getElementById('workerApiKey');
            const workerLabel = document.getElementById('workerLabel');
            const workerRegion = document.getElementById('workerRegion');
            const workerPlan = document.getElementById('workerPlan');
            const btnSubmitWorker = document.getElementById('btnSubmitWorker');
            const workerLogsBox = document.getElementById('workerLogsBox') || document.getElementById('workerTerminalContainer');
            const workerLogsContent = document.getElementById('workerLogsContent') || document.getElementById('workerStreamLog');

            const payload = {
                nodeName: workerNodeName ? workerNodeName.value.trim() : '',
                provider: workerProvider ? workerProvider.value : 'vultr',
                apiToken: workerApiKey ? workerApiKey.value.trim() : '',
                label: workerLabel ? workerLabel.value : 'worker',
                region: workerRegion ? workerRegion.value.trim() : 'mex',
                plan: workerPlan ? workerPlan.value.trim() : 'vc2-1c-1gb',
                server: state.selectedServerName || ''
            };

            if (btnSubmitWorker) {
                btnSubmitWorker.disabled = true;
                btnSubmitWorker.textContent = '⏳ Aprovisionando...';
            }
            if (workerLogsBox) workerLogsBox.style.display = 'block';
            if (workerLogsContent) workerLogsContent.innerHTML = `<span style="color:var(--text-muted);">Iniciando orquestación de VM en la nube...</span>\n`;

            try {
                const res = await fetch('/api/provision-worker', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    const errText = await res.text();
                    if (workerLogsContent) workerLogsContent.innerHTML += `<span style="color:var(--status-offline);">✕ Error (${res.status}): ${escapeHtml(errText)}</span>\n`;
                    showToast(`Fallo al provisionar: ${errText}`, 'error');
                    return;
                }

                await consumeNDJSONStream(res, (event) => {
                    if (workerLogsContent) {
                        workerLogsContent.innerHTML += `<div>${escapeHtml(event.data || event.message || JSON.stringify(event))}</div>`;
                        workerLogsContent.scrollTop = workerLogsContent.scrollHeight;
                    }
                }, (errMsg) => {
                    if (workerLogsContent) {
                        workerLogsContent.innerHTML += `<div style="color:var(--status-offline);">✕ ${escapeHtml(errMsg)}</div>`;
                    }
                    showToast(`Error: ${errMsg}`, 'error');
                });

                showToast(`¡Worker '${payload.nodeName}' unido exitosamente al clúster!`, 'success');
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
                }
            } catch (err) {
                showToast(`Error de red: ${err.message}`, 'error');
            } finally {
                if (btnSubmitWorker) {
                    btnSubmitWorker.disabled = false;
                    btnSubmitWorker.textContent = '⚡ Aprovisionar y Unir al Clúster';
                }
            }
        });
    }
}

export const setupNodesListeners = setupNodesEvents;

