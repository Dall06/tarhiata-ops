/**
 * Tarhiata Cloud Studio — Swarm Nodes Component (opt/nodes)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';
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

        let assignedHtml = '<span style="color:var(--text-muted); font-size:0.75rem;">Sin cargas asignadas</span>';
        if (assignedList.length > 0) {
            assignedHtml = `<div style="display:flex; gap:4px; flex-wrap:wrap;">` +
                assignedList.map(item => {
                    const icon = item.type === 'db' ? '🗄️' : '🚀';
                    return `<span class="t-badge t-badge-active" style="font-size:0.72rem; padding:1px 6px;">${icon} ${escapeHtml(item.name)}</span>`;
                }).join('') +
                `</div>`;
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

export function openWorkerModal() {
    const workerForm = document.getElementById('workerForm');
    const workerStreamLog = document.getElementById('workerStreamLog');
    const workerTerminalContainer = document.getElementById('workerTerminalContainer');
    if (workerForm) workerForm.reset();
    if (workerStreamLog) workerStreamLog.innerHTML = '';
    if (workerTerminalContainer) workerTerminalContainer.style.display = 'none';
    openModal('workerModal');
}

export function closeWorkerModal() {
    closeModal('workerModal');
}

export function setupNodesEvents(onReloadStatus) {
    const btnOpenWorkerModal = document.getElementById('btnOpenWorkerModal');
    const btnCloseWorkerModal = document.getElementById('btnCloseWorkerModal');
    const btnCloseWorkerModalBottom = document.getElementById('btnCloseWorkerModalBottom');
    const workerForm = document.getElementById('workerForm');

    if (btnOpenWorkerModal) btnOpenWorkerModal.addEventListener('click', openWorkerModal);
    if (btnCloseWorkerModal) btnCloseWorkerModal.addEventListener('click', closeWorkerModal);
    if (btnCloseWorkerModalBottom) btnCloseWorkerModalBottom.addEventListener('click', closeWorkerModal);

    if (workerForm) {
        workerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const workerNodeName = document.getElementById('workerNodeName');
            const workerLabel = document.getElementById('workerLabel');
            const workerRegion = document.getElementById('workerRegion');
            const workerPlan = document.getElementById('workerPlan');
            const btnSubmitWorker = document.getElementById('btnSubmitWorker');
            const workerTerminalContainer = document.getElementById('workerTerminalContainer');
            const workerStreamLog = document.getElementById('workerStreamLog');

            const payload = {
                nodeName: workerNodeName ? workerNodeName.value.trim() : '',
                label: workerLabel ? workerLabel.value : 'worker',
                region: workerRegion ? workerRegion.value.trim() : 'mex',
                plan: workerPlan ? workerPlan.value.trim() : 'vc2-1c-1gb',
                server: state.selectedServerName || ''
            };

            if (btnSubmitWorker) {
                btnSubmitWorker.disabled = true;
                btnSubmitWorker.textContent = '⏳ Aprovisionando...';
            }
            if (workerTerminalContainer) workerTerminalContainer.style.display = 'block';
            if (workerStreamLog) workerStreamLog.innerHTML = `<span style="color:var(--text-muted);">Iniciando orquestación de VM en la nube...</span>\n`;

            try {
                const res = await fetch('/api/swarm/provision-worker', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    const errText = await res.text();
                    if (workerStreamLog) workerStreamLog.innerHTML += `<span style="color:var(--status-offline);">✕ Error (${res.status}): ${escapeHtml(errText)}</span>\n`;
                    showToast(`Fallo al provisionar: ${errText}`, 'error');
                    return;
                }

                await consumeNDJSONStream(res, (event) => {
                    if (workerStreamLog) {
                        workerStreamLog.innerHTML += `<span>${escapeHtml(event.data || '')}</span>\n`;
                        workerStreamLog.scrollTop = workerStreamLog.scrollHeight;
                    }
                }, (errMsg) => {
                    if (workerStreamLog) {
                        workerStreamLog.innerHTML += `<span style="color:var(--status-offline);">✕ ${escapeHtml(errMsg)}</span>\n`;
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
