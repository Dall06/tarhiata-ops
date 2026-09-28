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

export function openEditServiceModal(name, expose, domain) {
    ensureServicesModalsMounted();
    const editSvcName = document.getElementById('editServiceName') || document.getElementById('editSvcName');
    const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
    const editSvcDomain = document.getElementById('editServiceDomain') || document.getElementById('editSvcDomain');
    const editDomainField = document.getElementById('editDomainField') || document.getElementById('editDomainGroup');
    const editFeedback = document.getElementById('editFeedback');

    if (editSvcName) editSvcName.value = name || '';
    if (editSvcExpose) editSvcExpose.value = (expose ? 'true' : 'false');
    if (editSvcDomain) editSvcDomain.value = domain || '';
    if (editDomainField) editDomainField.style.display = expose ? 'flex' : 'none';
    if (editFeedback) editFeedback.style.display = 'none';

    openModal('editServiceModal');
}

export function closeEditServiceModal() {
    closeModal('editServiceModal');
}

export function setupServicesEvents(onReloadStatus) {
    ensureServicesModalsMounted();

    const btnGlobalDeploy = document.getElementById('btnGlobalDeploy');
    const btnCloseDeployModal = document.getElementById('btnCloseDeployModal');
    const btnCancelDeploy = document.getElementById('btnCancelDeploy');
    const btnCloseEditServiceModal = document.getElementById('btnCloseEditServiceModal');
    const btnCancelEditService = document.getElementById('btnCancelEditService');
    const deployForm = document.getElementById('formDeploy') || document.getElementById('deployForm');
    const editServiceForm = document.getElementById('formEditService') || document.getElementById('editServiceForm');
    const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
    const editDomainField = document.getElementById('editDomainField') || document.getElementById('editDomainGroup');

    if (btnGlobalDeploy) btnGlobalDeploy.addEventListener('click', openDeployModal);
    if (btnCloseDeployModal) btnCloseDeployModal.addEventListener('click', closeDeployModal);
    if (btnCancelDeploy) btnCancelDeploy.addEventListener('click', closeDeployModal);
    if (btnCloseEditServiceModal) btnCloseEditServiceModal.addEventListener('click', closeEditServiceModal);
    if (btnCancelEditService) btnCancelEditService.addEventListener('click', closeEditServiceModal);

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
                    showToast(`Error al desplegar: ${res.error}`, 'error');
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
            const editSvcName = document.getElementById('editServiceName') || document.getElementById('editSvcName');
            const editSvcDomain = document.getElementById('editServiceDomain') || document.getElementById('editSvcDomain');
            const editSvcExpose = document.getElementById('editServiceExpose') || document.getElementById('editSvcExpose');
            const editFeedback = document.getElementById('editFeedback');
            const btnSubmitEdit = document.getElementById('btnSubmitEditService') || document.getElementById('btnSubmitEdit');

            const isExpose = editSvcExpose ? (editSvcExpose.value === 'true' || editSvcExpose.checked) : false;
            const payload = {
                name: editSvcName ? editSvcName.value.trim() : '',
                domain: editSvcDomain ? editSvcDomain.value.trim() : '',
                expose: isExpose,
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
                    showToast(`Error al actualizar servicio: ${res.error}`, 'error');
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

export const setupServicesListeners = setupServicesEvents;

