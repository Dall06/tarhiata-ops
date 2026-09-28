/**
 * Tarhiata Cloud Studio — Fleet & Server Component (opt/fleet)
 * Composite module: uses pkg/store, pkg/toast, pkg/jsutil, pkg/modal, pkg/apiclient.
 */

import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml, formatDockerVersion } from '/pkg/jsutil/utils.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { apiFetch } from '/pkg/apiclient/api.js';

export function renderFleetDirectory(onSelectServer, onSwitchActive, onDeleteServer) {
    const fleetList = document.getElementById('fleetList');
    const fleetCountBadge = document.getElementById('fleetCountBadge');
    const fleetSearchInput = document.getElementById('fleetSearchInput');
    if (!fleetList) return;

    fleetList.innerHTML = '';
    const query = (fleetSearchInput ? fleetSearchInput.value : '').toLowerCase().trim();
    const filtered = state.servers.filter(s =>
        s.name.toLowerCase().includes(query) ||
        s.host.toLowerCase().includes(query) ||
        (s.cloudProvider || '').toLowerCase().includes(query)
    );

    if (fleetCountBadge) {
        fleetCountBadge.textContent = `${state.servers.length} VPS`;
    }

    if (filtered.length === 0) {
        fleetList.innerHTML = `<div style="padding:16px; text-align:center; color:var(--text-muted); font-size:0.75rem;">No se encontraron servidores.</div>`;
        return;
    }

    filtered.forEach(s => {
        const isSelected = (s.name === state.selectedServerName);
        const isActive = s.isActive;
        const item = document.createElement('div');
        item.className = `fleet-item ${isSelected ? 'selected' : ''} ${isActive ? 'is-active-master' : ''}`;

        const isLoc = (s.host === 'localhost' || s.host === '127.0.0.1');
        const badgeProv = isLoc ? 'LOCAL' : (s.cloudProvider || 'VPS').toUpperCase();

        item.innerHTML = `
            <div class="fleet-item-info">
                <span class="status-dot status-pending" id="fleet-dot-${escapeHtml(s.name)}"></span>
                <span class="fleet-item-name">${escapeHtml(s.name)}</span>
                <span class="fleet-badge">${badgeProv}</span>
                ${isActive ? '<span class="fleet-badge" style="background:rgba(16,185,129,0.15); color:var(--accent-success); border-color:rgba(16,185,129,0.3);">ACTIVO</span>' : ''}
            </div>
            <div class="fleet-item-meta">
                <span>${escapeHtml(s.host)}</span>
                <div class="fleet-item-actions">
                    ${!isActive ? `<button class="mini-btn mini-btn-accent btn-make-active" title="Establecer como servidor principal">Principal</button>` : ''}
                    <button class="mini-btn btn-delete-srv" title="Eliminar servidor">✕</button>
                </div>
            </div>
        `;

        item.addEventListener('click', (e) => {
            if (e.target.closest('.fleet-item-actions')) return;
            if (onSelectServer) onSelectServer(s.name);
        });

        const btnMakeActive = item.querySelector('.btn-make-active');
        if (btnMakeActive) {
            btnMakeActive.addEventListener('click', (e) => {
                e.stopPropagation();
                if (onSwitchActive) onSwitchActive(s.name);
            });
        }

        const btnDelete = item.querySelector('.btn-delete-srv');
        if (btnDelete) {
            btnDelete.addEventListener('click', (e) => {
                e.stopPropagation();
                if (onDeleteServer) onDeleteServer(s.name);
            });
        }

        fleetList.appendChild(item);
    });
}

export async function switchActiveServer(name, loadHubStateCallback) {
    showToast(`Cambiando a servidor activo: '${name}'...`, 'info');
    const res = await apiFetch('/api/servers/active', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name })
    });
    if (!res.ok) return;
    showToast(`Servidor principal: '${name}'.`, 'success');
    state.selectedServerName = name;
    if (loadHubStateCallback) await loadHubStateCallback();
}

export async function deleteServer(name, loadHubStateCallback) {
    if (!confirm(`¿Eliminar definitivamente el servidor '${name}'?`)) return;

    const res = await apiFetch(`/api/servers?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
    if (!res.ok) return;
    showToast(`Servidor '${name}' eliminado.`, 'success');
    if (state.selectedServerName === name) {
        state.selectedServerName = null;
    }
    if (loadHubStateCallback) await loadHubStateCallback();
}

export async function testSingleServer(name) {
    const dotEl = document.getElementById(`fleet-dot-${name}`);
    const chipDot = document.getElementById(`chip-dot-${name}`);
    if (dotEl) dotEl.className = 'status-dot status-pending';
    if (chipDot) chipDot.className = 'status-dot status-pending';

    const s = state.servers.find(x => x.name === name);
    if (!s) return;

    const res = await apiFetch('/api/connect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(s)
    });

    if (!res.ok || !res.data || !res.data.connected) {
        if (dotEl) dotEl.className = 'status-dot status-offline';
        if (chipDot) chipDot.className = 'status-dot status-offline';
        if (name === state.selectedServerName) {
            const topActiveDot = document.getElementById('topActiveDot');
            const topActiveLatency = document.getElementById('topActiveLatency');
            if (topActiveDot) topActiveDot.className = 'status-dot status-offline';
            if (topActiveLatency) topActiveLatency.textContent = 'OFFLINE';
        }
        return;
    }

    const diag = res.data;
    if (dotEl) dotEl.className = 'status-dot status-online';
    if (chipDot) chipDot.className = 'status-dot status-online';

    if (name === state.selectedServerName) {
        const deskDocker = document.getElementById('deskDocker');
        const deskSwarm = document.getElementById('deskSwarm');
        const tileLatencyDisplay = document.getElementById('tileLatencyDisplay');
        const topActiveDot = document.getElementById('topActiveDot');
        const topActiveLatency = document.getElementById('topActiveLatency');

        if (deskDocker) {
            deskDocker.textContent = diag.dockerActive ? formatDockerVersion(diag.dockerVersion) : 'Inactivo';
            deskDocker.title = diag.dockerActive ? (diag.dockerVersion || '') : 'Inactivo';
        }
        if (deskSwarm) deskSwarm.textContent = diag.swarmActive ? 'Operacional' : 'Inactivo';
        if (tileLatencyDisplay) tileLatencyDisplay.textContent = `Latencia: ${diag.latencyMs} ms`;
        if (topActiveDot) topActiveDot.className = 'status-dot status-online';
        if (topActiveLatency) topActiveLatency.textContent = `${diag.latencyMs} ms`;
    }
}

export async function testAllServers(isManual = false) {
    const btnTestAll = document.getElementById('btnTestAll');
    if (btnTestAll) {
        btnTestAll.disabled = true;
        btnTestAll.innerHTML = '<span>⟳ Sondeando...</span>';
    }

    try {
        const res = await apiFetch('/api/connect/all', { method: 'POST' });
        if (res.ok && Array.isArray(res.data)) {
            res.data.forEach(diag => {
                const dotEl = document.getElementById(`fleet-dot-${diag.name}`);
                if (dotEl) {
                    dotEl.className = diag.connected ? 'status-dot status-online' : 'status-dot status-offline';
                }
                const chipDot = document.getElementById(`chip-dot-${diag.name}`);
                if (chipDot) {
                    chipDot.className = diag.connected ? 'status-dot status-online' : 'status-dot status-offline';
                }
                if (diag.name === state.selectedServerName && diag.connected) {
                    const deskDocker = document.getElementById('deskDocker');
                    const deskSwarm = document.getElementById('deskSwarm');
                    const tileLatencyDisplay = document.getElementById('tileLatencyDisplay');
                    const topActiveDot = document.getElementById('topActiveDot');
                    const topActiveLatency = document.getElementById('topActiveLatency');

                    if (deskDocker) {
                        deskDocker.textContent = diag.dockerActive ? formatDockerVersion(diag.dockerVersion) : 'Inactivo';
                        deskDocker.title = diag.dockerActive ? (diag.dockerVersion || '') : 'Inactivo';
                    }
                    if (deskSwarm) deskSwarm.textContent = diag.swarmActive ? 'Operacional' : 'Inactivo';
                    if (tileLatencyDisplay) tileLatencyDisplay.textContent = `Latencia: ${diag.latencyMs} ms`;
                    if (topActiveDot) topActiveDot.className = 'status-dot status-online';
                    if (topActiveLatency) topActiveLatency.textContent = `${diag.latencyMs} ms`;
                }
            });
        } else {
            await Promise.all(state.servers.map(s => testSingleServer(s.name)));
        }
    } finally {
        if (btnTestAll) {
            btnTestAll.disabled = false;
            btnTestAll.innerHTML = `
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6"></path><path d="M1 20v-6h6"></path><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
                <span>Sondear Red</span>
            `;
        }
        if (isManual) {
            showToast('Sondeo de conectividad completado.', 'info');
        }
    }
}

export function setModalMode(mode) {
    state.modalMode = mode;
    const tabLocal = document.getElementById('tabLocal');
    const tabRemote = document.getElementById('tabRemote');
    const tabCloud = document.getElementById('tabCloud');
    const modalTestResult = document.getElementById('modalTestResult');
    const localFastNotice = document.getElementById('localFastNotice');
    const cloudFields = document.getElementById('cloudFields');
    const standardFields = document.getElementById('standardFields');
    const nameField = document.getElementById('nameField');
    const hostField = document.getElementById('hostField');
    const userField = document.getElementById('userField');
    const portField = document.getElementById('portField');
    const keyField = document.getElementById('keyField');
    const cfgName = document.getElementById('cfgName');
    const cfgHost = document.getElementById('cfgHost');
    const cfgUser = document.getElementById('cfgUser');
    const cfgPort = document.getElementById('cfgPort');
    const cfgKey = document.getElementById('cfgKey');
    const btnModalTest = document.getElementById('btnModalTest');
    const btnModalSaveText = document.getElementById('btnModalSaveText');

    if (tabLocal) tabLocal.classList.toggle('active', mode === 'local');
    if (tabRemote) tabRemote.classList.toggle('active', mode === 'remote');
    if (tabCloud) tabCloud.classList.toggle('active', mode === 'cloud');
    if (modalTestResult) modalTestResult.style.display = 'none';

    if (mode === 'local') {
        if (localFastNotice) localFastNotice.style.display = 'flex';
        if (cloudFields) cloudFields.style.display = 'none';
        if (standardFields) standardFields.style.display = 'grid';
        if (nameField) nameField.style.display = 'flex';
        if (hostField) hostField.style.display = 'none';
        if (userField) userField.style.display = 'none';
        if (portField) portField.style.display = 'none';
        if (keyField) keyField.style.display = 'none';

        if (cfgName) cfgName.value = 'local';
        if (cfgHost) cfgHost.value = 'localhost';
        if (cfgUser) cfgUser.value = 'local';
        if (cfgPort) cfgPort.value = '0';
        if (btnModalTest) btnModalTest.style.display = 'inline-flex';
        if (btnModalSaveText) btnModalSaveText.textContent = 'Conectar Local';
        return;
    }
    if (mode === 'remote') {
        if (localFastNotice) localFastNotice.style.display = 'none';
        if (cloudFields) cloudFields.style.display = 'none';
        if (standardFields) standardFields.style.display = 'grid';
        if (nameField) nameField.style.display = 'flex';
        if (hostField) hostField.style.display = 'flex';
        if (userField) userField.style.display = 'flex';
        if (portField) portField.style.display = 'flex';
        if (keyField) keyField.style.display = 'flex';

        if (cfgName && (cfgName.value === 'local' || cfgName.value.startsWith('cloud-'))) {
            cfgName.value = 'vps-servidor';
        }
        if (cfgHost && (cfgHost.value === 'localhost' || cfgHost.value === '127.0.0.1')) {
            cfgHost.value = '';
        }
        if (cfgUser) cfgUser.value = 'root';
        if (cfgPort) cfgPort.value = '22';
        if (cfgKey) cfgKey.value = '~/.ssh/id_rsa';
        if (btnModalTest) btnModalTest.style.display = 'inline-flex';
        if (btnModalSaveText) btnModalSaveText.textContent = 'Guardar VPS';
        return;
    }
    if (mode === 'cloud') {
        if (localFastNotice) localFastNotice.style.display = 'none';
        if (cloudFields) cloudFields.style.display = 'grid';
        if (standardFields) standardFields.style.display = 'grid';
        if (nameField) nameField.style.display = 'flex';
        if (hostField) hostField.style.display = 'none';
        if (userField) userField.style.display = 'none';
        if (portField) portField.style.display = 'none';
        if (keyField) keyField.style.display = 'none';

        if (cfgName && (!cfgName.value || cfgName.value === 'local')) {
            cfgName.value = 'cloud-node-1';
        }
        if (btnModalTest) btnModalTest.style.display = 'none';
        if (btnModalSaveText) btnModalSaveText.textContent = 'Crear con OpenTofu';
    }
}

export function ensureServerModalMounted() {
    if (document.getElementById('serverModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `<div class="t-modal-overlay" id="serverModal" style="display:none;" role="dialog" aria-modal="true">
        <div class="t-modal-card">
            <div class="t-modal-header">
                <div class="t-modal-title">
                    <span>🖥️ Conectar Servidor</span>
                </div>
                <button type="button" class="t-close-btn" id="btnCloseModal" aria-label="Cerrar">×</button>
            </div>

            <div class="mode-selector">
                <button type="button" class="mode-btn active" id="tabLocal">
                    <span>Máquina Local</span>
                </button>
                <button type="button" class="mode-btn" id="tabRemote">
                    <span>VPS Remoto (SSH)</span>
                </button>
                <button type="button" class="mode-btn" id="tabCloud">
                    <span>Crear en la Nube</span>
                </button>
            </div>

            <form id="formServer">
                <div class="form-body">
                    <div id="localFastNotice" class="fast-notice">
                        <span class="notice-icon">⚡</span>
                        <div>
                            <strong>Conexión Local Directa</strong>
                            <p>Monitorea y opera este mismo equipo sin requerir llaves SSH. Se vinculará de inmediato.</p>
                        </div>
                    </div>

                    <div class="form-row-grid" id="cloudFields" style="display:none;">
                        <div class="form-field">
                            <label for="cfgProvider">Proveedor Cloud</label>
                            <select id="cfgProvider" class="t-input">
                                <option value="vultr">Vultr Cloud Compute</option>
                                <option value="digitalocean">DigitalOcean Droplet</option>
                            </select>
                        </div>
                        <div class="form-field">
                            <label for="cfgRegion">Región del Datacenter</label>
                            <input type="text" id="cfgRegion" class="t-input" placeholder="mex, nyc1, ewr" value="mex">
                        </div>
                        <div class="form-field full-span">
                            <label for="cfgToken">API Key / Token del Proveedor</label>
                            <input type="password" id="cfgToken" class="t-input" placeholder="Token secreto de tu cuenta">
                        </div>
                        <div class="form-field full-span">
                            <label for="cfgPlan">Plan de Servidor</label>
                            <input type="text" id="cfgPlan" class="t-input" placeholder="vc2-1c-1gb o s-1vcpu-1gb" value="vc2-1c-1gb">
                        </div>
                    </div>

                    <div class="form-row-grid" id="standardFields">
                        <div class="form-field" id="nameField">
                            <label for="cfgName">Nombre / Alias</label>
                            <input type="text" id="cfgName" class="t-input" placeholder="ej: vps-produccion" required>
                        </div>
                        <div class="form-field" id="hostField">
                            <label for="cfgHost">Host o Dirección IP</label>
                            <input type="text" id="cfgHost" class="t-input" placeholder="ej: 108.61.33.61 o dominio" required>
                        </div>
                        <div class="form-field" id="userField" style="display:none;">
                            <label for="cfgUser">Usuario SSH</label>
                            <input type="text" id="cfgUser" class="t-input" value="root">
                        </div>
                        <div class="form-field" id="portField" style="display:none;">
                            <label for="cfgPort">Puerto SSH</label>
                            <input type="number" id="cfgPort" class="t-input" value="22">
                        </div>
                        <div class="form-field full-span" id="keyField" style="display:none;">
                            <label for="cfgKey">Ruta de Llave Privada SSH</label>
                            <input type="text" id="cfgKey" class="t-input" placeholder="~/.ssh/id_rsa o ruta absoluta" value="~/.ssh/id_rsa">
                        </div>
                    </div>

                    <div class="form-field checkbox-field">
                        <label class="t-checkbox-label">
                            <input type="checkbox" id="cfgIsActive" checked>
                            <span>Establecer como servidor activo de inmediato</span>
                        </label>
                    </div>

                    <div id="modalTestResult" class="t-diagnostic-box" style="display:none;"></div>
                </div>

                <div class="t-modal-footer">
                    <button type="button" class="t-btn t-btn-secondary" id="btnCancelServer">Cancelar</button>
                    <button type="button" class="t-btn t-btn-secondary" id="btnModalTest">
                        <span id="btnModalTestText">Probar Conexión</span>
                    </button>
                    <button type="submit" class="t-btn t-btn-primary" id="btnModalSave">
                        <span id="btnModalSaveText">Guardar Servidor</span>
                    </button>
                </div>
            </form>
        </div>
    </div>`;
    document.body.appendChild(div.firstElementChild);
}

export function openAddModal() {
    ensureServerModalMounted();
    const formServer = document.getElementById('formServer');
    const serverPopover = document.getElementById('serverPopover');
    if (formServer) formServer.reset();
    setModalMode('remote');
    openModal('serverModal');
    if (serverPopover) {
        serverPopover.classList.remove('active');
        serverPopover.style.display = 'none';
    }
}

export function closeServerModal() {
    const formServer = document.getElementById('formServer');
    const modalTestResult = document.getElementById('modalTestResult');
    if (formServer) formServer.reset();
    if (modalTestResult) modalTestResult.innerHTML = '';
    closeModal('serverModal');
}

export function setupFleetEvents(loadHubStateCallback) {
    ensureServerModalMounted();
    const btnOpenAddModal = document.getElementById('btnOpenAddModal');
    const btnFleetAddServer = document.getElementById('btnFleetAddServer');
    const btnCloseModal = document.getElementById('btnCloseModal');
    const btnCancelServer = document.getElementById('btnCancelServer');
    const tabLocal = document.getElementById('tabLocal');
    const tabRemote = document.getElementById('tabRemote');
    const tabCloud = document.getElementById('tabCloud');
    const btnTestAll = document.getElementById('btnTestAll');
    const formServer = document.getElementById('formServer');
    const btnModalTest = document.getElementById('btnModalTest');
    const fleetSearchInput = document.getElementById('fleetSearchInput');
    const serverSwitcherBtn = document.getElementById('serverSwitcherBtn');
    const serverPopover = document.getElementById('serverPopover');
    const serverSwitcherWrap = document.getElementById('serverSwitcherWrap');
    const btnDeskActivate = document.getElementById('btnDeskActivate');
    const btnSidebarOpenWorker = document.getElementById('btnSidebarOpenWorker');

    if (serverSwitcherBtn && serverPopover) {
        serverSwitcherBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            const isOpen = serverPopover.style.display !== 'none';
            serverPopover.style.display = isOpen ? 'none' : 'flex';
            serverPopover.classList.toggle('active', !isOpen);
            serverSwitcherBtn.classList.toggle('open', !isOpen);
        });
    }

    document.addEventListener('click', (e) => {
        if (serverPopover && serverSwitcherWrap && !serverSwitcherWrap.contains(e.target)) {
            serverPopover.style.display = 'none';
            serverPopover.classList.remove('active');
            if (serverSwitcherBtn) serverSwitcherBtn.classList.remove('open');
        }
    });

    if (btnDeskActivate) {
        btnDeskActivate.addEventListener('click', () => {
            if (state.selectedServerName) switchActiveServer(state.selectedServerName, () => loadHubState());
        });
    }

    if (btnSidebarOpenWorker) {
        btnSidebarOpenWorker.addEventListener('click', async () => {
            if (serverPopover) {
                serverPopover.style.display = 'none';
                serverPopover.classList.remove('active');
            }
            const { openWorkerModal } = await import('/components/nodes/nodes.js');
            openWorkerModal();
        });
    }

    if (btnOpenAddModal) btnOpenAddModal.addEventListener('click', openAddModal);
    if (btnFleetAddServer) btnFleetAddServer.addEventListener('click', openAddModal);
    if (btnCloseModal) btnCloseModal.addEventListener('click', closeServerModal);
    if (btnCancelServer) btnCancelServer.addEventListener('click', closeServerModal);

    if (tabLocal) tabLocal.addEventListener('click', () => setModalMode('local'));
    if (tabRemote) tabRemote.addEventListener('click', () => setModalMode('remote'));
    if (tabCloud) tabCloud.addEventListener('click', () => setModalMode('cloud'));

    if (btnTestAll) btnTestAll.addEventListener('click', () => testAllServers(true));

    if (fleetSearchInput) {
        fleetSearchInput.addEventListener('input', () => {
            renderFleetDirectory(
                (name) => {
                    state.selectedServerName = name;
                    renderFleetDirectory(null, null, null);
                    if (loadHubStateCallback) loadHubStateCallback();
                },
                (name) => switchActiveServer(name, loadHubStateCallback),
                (name) => deleteServer(name, loadHubStateCallback)
            );
        });
    }

    if (formServer) {
        formServer.addEventListener('submit', async (e) => {
            e.preventDefault();
            const isCloud = (state.modalMode === 'cloud');
            const isLoc = (state.modalMode === 'local');
            const cfgName = document.getElementById('cfgName');
            const cfgHost = document.getElementById('cfgHost');
            const cfgPort = document.getElementById('cfgPort');
            const cfgUser = document.getElementById('cfgUser');
            const cfgKey = document.getElementById('cfgKey');
            const cfgProvider = document.getElementById('cfgProvider');
            const cfgToken = document.getElementById('cfgToken');
            const cfgRegion = document.getElementById('cfgRegion');
            const cfgPlan = document.getElementById('cfgPlan');
            const cfgIsActive = document.getElementById('cfgIsActive');
            const btnModalSave = document.getElementById('btnModalSave');
            const btnModalSaveText = document.getElementById('btnModalSaveText');
            const modalTestResult = document.getElementById('modalTestResult');

            let payload = {
                name: cfgName ? cfgName.value.trim() : '',
                host: isLoc ? 'localhost' : (cfgHost ? cfgHost.value.trim() : ''),
                port: isLoc ? 0 : parseInt(cfgPort ? cfgPort.value || '22' : '22', 10),
                user: isLoc ? 'local' : (cfgUser ? cfgUser.value.trim() : ''),
                privateKey: isLoc ? '' : (cfgKey ? cfgKey.value.trim() : ''),
                cloudProvider: isLoc ? 'local' : (isCloud && cfgProvider ? cfgProvider.value : 'custom'),
                isActive: cfgIsActive ? cfgIsActive.checked : false
            };

            if (isCloud && cfgProvider) {
                payload.provider = cfgProvider.value;
                payload.apiToken = cfgToken ? cfgToken.value.trim() : '';
                payload.region = cfgRegion ? cfgRegion.value.trim() || 'mex' : 'mex';
                payload.plan = cfgPlan ? cfgPlan.value.trim() || 'vc2-1c-1gb' : 'vc2-1c-1gb';
            }

            if (btnModalSave) btnModalSave.disabled = true;
            if (btnModalSaveText) btnModalSaveText.textContent = 'Guardando...';

            try {
                const endpoint = isCloud ? '/api/servers/provision' : '/api/servers';
                const res = await apiFetch(endpoint, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (!res.ok) {
                    if (modalTestResult) {
                        modalTestResult.style.display = 'block';
                        modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ ${escapeHtml(res.error)}</span>`;
                    }
                    return;
                }

                showToast(`Servidor '${payload.name}' conectado con éxito!`, 'success');
                closeServerModal();
                state.selectedServerName = payload.name;
                const cb = loadHubStateCallback || (() => loadHubState());
                await cb();
            } finally {
                if (btnModalSave) btnModalSave.disabled = false;
                if (btnModalSaveText) btnModalSaveText.textContent = isCloud ? 'Crear con OpenTofu' : 'Guardar Servidor';
            }
        });
    }

    if (btnModalTest) {
        btnModalTest.addEventListener('click', async () => {
            const btnModalTestText = document.getElementById('btnModalTestText');
            const modalTestResult = document.getElementById('modalTestResult');
            const cfgName = document.getElementById('cfgName');
            const cfgHost = document.getElementById('cfgHost');
            const cfgPort = document.getElementById('cfgPort');
            const cfgUser = document.getElementById('cfgUser');
            const cfgKey = document.getElementById('cfgKey');

            btnModalTest.disabled = true;
            if (btnModalTestText) btnModalTestText.textContent = 'Probando...';
            if (modalTestResult) {
                modalTestResult.style.display = 'block';
                modalTestResult.innerHTML = `<span style="color:var(--status-warning);">⏳ Verificando conexión...</span>`;
            }

            const isLoc = (state.modalMode === 'local');
            const testPayload = {
                name: cfgName ? cfgName.value.trim() || 'test' : 'test',
                host: isLoc ? 'localhost' : (cfgHost ? cfgHost.value.trim() : ''),
                port: isLoc ? 0 : parseInt(cfgPort ? cfgPort.value || '22' : '22', 10),
                user: isLoc ? 'local' : (cfgUser ? cfgUser.value.trim() : ''),
                privateKey: isLoc ? '' : (cfgKey ? cfgKey.value.trim() : ''),
                cloudProvider: isLoc ? 'local' : 'custom'
            };

            try {
                const res = await apiFetch('/api/connect', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(testPayload)
                });
                if (!res.ok || !res.data) {
                    if (modalTestResult) modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ Error: ${escapeHtml(res.error)}</span>`;
                    return;
                }
                const data = res.data;
                if (!data.connected) {
                    if (modalTestResult) modalTestResult.innerHTML = `<span style="color:var(--status-offline);">✕ Fallo de conexión SSH: ${escapeHtml(data.error || 'Host inalcanzable')}</span>`;
                    return;
                }
                const docVer = data.dockerVersion ? formatDockerVersion(data.dockerVersion) : 'No detectado';
                const swarmStatus = data.swarmActive ? 'Operacional' : 'No iniciado';
                if (modalTestResult) {
                    modalTestResult.innerHTML = `<span style="color:var(--status-online);">✓ Conexión exitosa · Latencia: ${data.latencyMs}ms · Docker: ${docVer} · Swarm: ${swarmStatus}</span>`;
                }
            } finally {
                btnModalTest.disabled = false;
                if (btnModalTestText) btnModalTestText.textContent = 'Probar Conexión';
            }
        });
    }
}

export async function selectServer(serverName) {
    state.selectedServerName = serverName;
    try {
        localStorage.setItem('tarhiata_last_server', serverName);
    } catch (storageErr) {
        console.debug('No se pudo guardar último servidor en localStorage:', storageErr);
    }

    let s = state.servers.find(x => x.name === serverName);
    if (!s) {
        if (state.activeServer && state.activeServer.name === serverName) {
            s = state.activeServer;
        } else if (state.servers.length > 0) {
            s = state.servers[0];
            serverName = s.name;
            state.selectedServerName = s.name;
        }
    }

    renderFleetDirectory(
        (name) => selectServer(name),
        (name) => switchActiveServer(name, () => loadHubState()),
        (name) => deleteServer(name, () => loadHubState())
    );

    if (!s) return;

    const { activateServerLoadingSkeletons, refreshServerTelemetry } = await import('/components/telemetry/telemetry.js');
    const { loadHostDevices } = await import('/components/hardware/hardware.js');

    activateServerLoadingSkeletons(s);
    state.currentHostDevices = null;

    await Promise.all([
        refreshServerTelemetry(serverName),
        loadHostDevices()
    ]);
}

export async function loadHubState() {
    const topActiveName = document.getElementById('topActiveName');
    const topActiveHost = document.getElementById('topActiveHost');
    const topActiveDot = document.getElementById('topActiveDot');
    const topActiveLatency = document.getElementById('topActiveLatency');

    try {
        const res = await apiFetch('/api/status');
        if (!res.ok) {
            if (topActiveName) topActiveName.textContent = 'Error de API';
            showToast(`Error al consultar estado (${res.status})`, 'error');
            const { deactivateInitialSkeletons } = await import('/components/telemetry/telemetry.js');
            deactivateInitialSkeletons();
            return;
        }

        const data = res.data || (typeof res.json === 'function' ? await res.json() : {});
        state.servers = data.servers || [];
        state.activeServer = data.config || null;

        if (state.servers.length === 0 && state.activeServer && state.activeServer.host) {
            state.servers = [state.activeServer];
        }

        if (topActiveName) topActiveName.textContent = 'Ninguno';
        if (topActiveHost) topActiveHost.textContent = '—';
        if (topActiveDot) topActiveDot.className = 'status-dot status-offline';
        if (topActiveLatency) topActiveLatency.textContent = '—';

        if (state.activeServer && state.activeServer.name) {
            if (topActiveName) topActiveName.textContent = state.activeServer.name;
            if (topActiveHost) topActiveHost.textContent = state.activeServer.host || 'localhost';
            if (topActiveDot) topActiveDot.className = data.isOnline ? 'status-dot status-online' : 'status-dot status-offline';
            if (topActiveLatency) topActiveLatency.textContent = data.isOnline ? 'ONLINE' : 'OFFLINE';
        }

        let savedServer = null;
        try {
            savedServer = localStorage.getItem('tarhiata_last_server');
        } catch (e) {}

        if (savedServer && state.servers.some(s => s.name === savedServer)) {
            state.selectedServerName = savedServer;
        } else if (!state.selectedServerName || !state.servers.some(s => s.name === state.selectedServerName)) {
            if (state.activeServer && state.activeServer.name) {
                state.selectedServerName = state.activeServer.name;
            } else if (state.servers.length > 0) {
                state.selectedServerName = state.servers[0].name;
            }
        }

        renderFleetDirectory(
            (name) => selectServer(name),
            (name) => switchActiveServer(name, () => loadHubState()),
            (name) => deleteServer(name, () => loadHubState())
        );

        if (state.selectedServerName) {
            await selectServer(state.selectedServerName);
        } else if (state.servers.length === 0) {
            const { deactivateInitialSkeletons } = await import('/components/telemetry/telemetry.js');
            deactivateInitialSkeletons();
        }
    } catch (err) {
        console.debug('Error en loadHubState:', err);
    }
}

export const setupFleetListeners = setupFleetEvents;

