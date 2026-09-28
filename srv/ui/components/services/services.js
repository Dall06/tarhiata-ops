/**
 * Tarhiata Cloud Studio — Swarm Services Component (opt/services)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch, consumeNDJSONStream } from '/pkg/apiclient/api.js';

export function renderAppCards(services, callbacks = {}) {
    const swarmServicesCardsGrid = document.getElementById('swarmServicesCardsGrid');
    const swarmServicesEmpty = document.getElementById('swarmServicesEmpty');
    if (!swarmServicesCardsGrid) return;

    swarmServicesCardsGrid.innerHTML = '';

    if (!services || services.length === 0) {
        if (swarmServicesEmpty) swarmServicesEmpty.style.display = 'flex';
        return;
    }
    if (swarmServicesEmpty) swarmServicesEmpty.style.display = 'none';

    services.forEach(svc => {
        const isPublic = svc.expose || (svc.domain && svc.domain !== '');
        const card = document.createElement('div');
        card.className = 'app-card';

        const domainHtml = svc.domain ? `
            <div class="app-card-url-box">
                <a href="http://${escapeHtml(svc.domain)}" target="_blank" class="app-card-url-link">
                    🌐 https://${escapeHtml(svc.domain)} ↗
                </a>
            </div>
        ` : `
            <div class="app-card-url-box" style="color:var(--text-dim); font-size:0.75rem;">
                🔒 Red Interna Swarm
            </div>
        `;

        card.innerHTML = `
            <div>
                <div class="app-card-header">
                    <div class="app-card-title-group">
                        <div class="app-card-icon">🚀</div>
                        <div style="min-width:0; flex:1; overflow:hidden;">
                            <h3 class="app-card-name" title="${escapeHtml(svc.name)}">${escapeHtml(svc.name)}</h3>
                            <div class="app-card-image" title="${escapeHtml(svc.image)}">${escapeHtml(svc.image)}</div>
                        </div>
                    </div>
                    <div class="app-card-badges">
                        ${isPublic && svc.domain ? '<span class="card-ssl-pill" style="background:rgba(16,185,129,0.12); color:#10b981; border:1px solid rgba(16,185,129,0.25);" title="Certificado SSL Activo vía Traefik">🔒 SSL</span>' : ''}
                        <span class="svc-pill svc-pill-active" style="flex-shrink:0;">
                            <span class="status-dot status-online" style="width:6px; height:6px;"></span>
                            ${escapeHtml(svc.replicas)} Réplicas
                        </span>
                    </div>
                </div>
                <div style="margin-top:14px;">
                    ${domainHtml}
                </div>
            </div>

            <div class="app-card-actions">
                <div class="app-card-actions-meta">
                    <span class="t-badge" style="font-size:0.72rem;">${isPublic ? '🌐 Público SSL' : '🔒 Privado'}</span>
                </div>
                <div class="app-card-actions-buttons">
                    <button type="button" class="mini-btn btn-logs-svc" data-name="${escapeHtml(svc.name)}" title="Ver logs en tiempo real">
                        📜 Logs
                    </button>
                    <button type="button" class="mini-btn btn-restart-svc" data-name="${escapeHtml(svc.name)}" title="Reiniciar servicio en Docker">
                        🔄 Reiniciar
                    </button>
                    <button type="button" class="mini-btn btn-env-svc" data-name="${escapeHtml(svc.name)}" title="Gestionar variables de entorno .env">
                        🔑 Env
                    </button>
                    <button type="button" class="mini-btn btn-vol-svc" data-name="${escapeHtml(svc.name)}" title="Explorar archivos del volumen de almacenamiento">
                        📁 Archivos
                    </button>
                    <button type="button" class="mini-btn btn-edit-svc" data-name="${escapeHtml(svc.name)}" data-expose="${isPublic}" data-domain="${escapeHtml(svc.domain || '')}">
                        ⚙️ Configurar
                    </button>
                    <button type="button" class="mini-btn btn-del-svc" data-name="${escapeHtml(svc.name)}" style="color:var(--status-offline);" title="Eliminar servicio">
                        ✕
                    </button>
                </div>
            </div>
        `;

        swarmServicesCardsGrid.appendChild(card);
    });

    // Wire action buttons
    document.querySelectorAll('.btn-logs-svc').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onOpenLogs) callbacks.onOpenLogs(name);
        });
    });

    document.querySelectorAll('.btn-restart-svc').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name) restartServiceOrContainer(name, btn);
        });
    });

    document.querySelectorAll('.btn-env-svc').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onOpenEnv) callbacks.onOpenEnv(name);
        });
    });

    document.querySelectorAll('.btn-vol-svc').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            if (name && callbacks.onOpenVolume) callbacks.onOpenVolume(`/opt/data/${name}`);
        });
    });

    document.querySelectorAll('.btn-edit-svc').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = btn.getAttribute('data-name');
            const expose = btn.getAttribute('data-expose') === 'true';
            const domain = btn.getAttribute('data-domain');
            openEditServiceModal(name, expose, domain);
        });
    });

    document.querySelectorAll('.btn-del-svc').forEach(btn => {
        btn.addEventListener('click', async () => {
            const name = btn.getAttribute('data-name');
            if (!confirm(`¿Estás seguro de eliminar el servicio '${name}' de Docker Swarm?`)) return;
            const res = await apiFetch(`/api/services/${encodeURIComponent(name)}?server=${encodeURIComponent(state.selectedServerName || '')}`, { method: 'DELETE' });
            if (res.ok) {
                showToast(`Servicio '${name}' eliminado.`, 'info');
                if (callbacks.onReloadStatus && state.selectedServerName) {
                    callbacks.onReloadStatus(state.selectedServerName);
                }
            }
        });
    });
}

export function renderServicesTable(services) {
    const hostServicesTableBody = document.getElementById('hostServicesTableBody');
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

export function openDeployModal() {
    const deployForm = document.getElementById('deployForm');
    const deployFeedback = document.getElementById('deployFeedback');
    if (deployForm) deployForm.reset();
    if (deployFeedback) deployFeedback.style.display = 'none';
    openModal('deployModal');
}

export function closeDeployModal() {
    closeModal('deployModal');
}

export function openEditServiceModal(name, expose, domain) {
    const editSvcName = document.getElementById('editSvcName');
    const editSvcExpose = document.getElementById('editSvcExpose');
    const editSvcDomain = document.getElementById('editSvcDomain');
    const editDomainGroup = document.getElementById('editDomainGroup');
    const editFeedback = document.getElementById('editFeedback');

    if (editSvcName) editSvcName.value = name || '';
    if (editSvcExpose) editSvcExpose.checked = !!expose;
    if (editSvcDomain) editSvcDomain.value = domain || '';
    if (editDomainGroup) editDomainGroup.style.display = expose ? 'flex' : 'none';
    if (editFeedback) editFeedback.style.display = 'none';

    openModal('editServiceModal');
}

export function closeEditServiceModal() {
    closeModal('editServiceModal');
}

export function setupServicesEvents(onReloadStatus) {
    const btnGlobalDeploy = document.getElementById('btnGlobalDeploy');
    const btnCloseDeployModal = document.getElementById('btnCloseDeployModal');
    const btnCloseEditServiceModal = document.getElementById('btnCloseEditServiceModal');
    const deployForm = document.getElementById('deployForm');
    const editServiceForm = document.getElementById('editServiceForm');
    const editSvcExpose = document.getElementById('editSvcExpose');
    const editDomainGroup = document.getElementById('editDomainGroup');

    if (btnGlobalDeploy) btnGlobalDeploy.addEventListener('click', openDeployModal);
    if (btnCloseDeployModal) btnCloseDeployModal.addEventListener('click', closeDeployModal);
    if (btnCloseEditServiceModal) btnCloseEditServiceModal.addEventListener('click', closeEditServiceModal);

    if (editSvcExpose && editDomainGroup) {
        editSvcExpose.addEventListener('change', () => {
            editDomainGroup.style.display = editSvcExpose.checked ? 'flex' : 'none';
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
                image: depImage ? depImage.value.trim() : '',
                port: parseInt(depPort ? depPort.value || '80' : '80', 10),
                domain: depDomain ? depDomain.value.trim() : '',
                preHook: depPreHook ? depPreHook.value.trim() : '',
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
                const res = await apiFetch('/api/deploy', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    if (deployFeedback) {
                        deployFeedback.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    return;
                }

                showToast(`¡Servicio '${payload.name}' desplegado con éxito!`, 'success');
                closeDeployModal();
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
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
            const editSvcName = document.getElementById('editSvcName');
            const editSvcDomain = document.getElementById('editSvcDomain');
            const editSvcExpose = document.getElementById('editSvcExpose');
            const editFeedback = document.getElementById('editFeedback');
            const btnSubmitEdit = document.getElementById('btnSubmitEdit');

            const payload = {
                name: editSvcName ? editSvcName.value.trim() : '',
                domain: editSvcDomain ? editSvcDomain.value.trim() : '',
                expose: editSvcExpose ? editSvcExpose.checked : false,
                server: state.selectedServerName || ''
            };

            if (btnSubmitEdit) btnSubmitEdit.disabled = true;

            try {
                const res = await apiFetch('/api/services/update', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (!res.ok) {
                    if (editFeedback) {
                        editFeedback.style.display = 'block';
                        editFeedback.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    return;
                }
                showToast(`Servicio '${payload.name}' actualizado.`, 'success');
                closeEditServiceModal();
                if (onReloadStatus && state.selectedServerName) {
                    onReloadStatus(state.selectedServerName);
                }
            } finally {
                if (btnSubmitEdit) btnSubmitEdit.disabled = false;
            }
        });
    }
}
