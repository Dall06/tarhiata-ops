import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, copyToClipboard, isSensitiveKey } from '/pkg/jsutil/utils.js';

let currentEnvServiceName = '';
let currentEnvMode = 'table';

export function openEnvModal(serviceName) {
    const envModal = document.getElementById('envModal');
    const envModalServiceName = document.getElementById('envModalServiceName');
    const envModalStatus = document.getElementById('envModalStatus');
    const envTabTable = document.getElementById('envTabTable');
    const envTabRaw = document.getElementById('envTabRaw');
    const envTableView = document.getElementById('envTableView');
    const envRawView = document.getElementById('envRawView');
    const envTableBody = document.getElementById('envTableBody');
    const envRawTextarea = document.getElementById('envRawTextarea');

    if (!envModal) return;
    currentEnvServiceName = serviceName;
    currentEnvMode = 'table';
    if (envModalServiceName) envModalServiceName.textContent = serviceName;
    if (envModalStatus) envModalStatus.textContent = '';
    if (envTabTable) envTabTable.classList.add('active');
    if (envTabRaw) envTabRaw.classList.remove('active');
    if (envTableView) envTableView.style.display = 'block';
    if (envRawView) envRawView.style.display = 'none';

    if (envTableBody) {
        envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Consultando variables de '${escapeHtml(serviceName)}'...</td></tr>`;
    }
    if (envRawTextarea) envRawTextarea.value = '';

    envModal.style.display = 'flex';
    fetchAndRenderEnvVars(serviceName);
}

export function closeEnvModal() {
    const envModal = document.getElementById('envModal');
    if (envModal) envModal.style.display = 'none';
    currentEnvServiceName = '';
}

export async function fetchAndRenderEnvVars(serviceName) {
    const envTableBody = document.getElementById('envTableBody');
    const envRawTextarea = document.getElementById('envRawTextarea');

    try {
        const res = await apiFetch(`/api/env?service=${encodeURIComponent(serviceName)}`);
        if (!res.ok) {
            if (envTableBody) {
                envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas aún. Pulsa '+ Agregar Variable'.</td></tr>`;
            }
            return;
        }
        const data = await res.json();
        const raw = data.rawContent || '';
        if (envRawTextarea) envRawTextarea.value = raw;
        renderEnvTableFromRaw(raw);
    } catch (err) {
        if (envTableBody) {
            envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty" style="color:var(--status-offline);">Error de conexión: ${escapeHtml(err.message)}</td></tr>`;
        }
    }
}

export function renderEnvTableFromRaw(rawContent) {
    const envTableBody = document.getElementById('envTableBody');
    if (!envTableBody) return;
    envTableBody.innerHTML = '';

    const lines = (rawContent || '').split('\n');
    let count = 0;

    lines.forEach(line => {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) return;
        const idx = trimmed.indexOf('=');
        if (idx === -1) return;

        const key = trimmed.slice(0, idx).trim();
        let val = trimmed.slice(idx + 1).trim();
        if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
            val = val.slice(1, -1);
        }

        createEnvTableRow(key, val);
        count++;
    });

    if (count === 0) {
        envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas. Pulsa '+ Agregar Variable'.</td></tr>`;
    }
}

export function createEnvTableRow(key, val) {
    const envTableBody = document.getElementById('envTableBody');
    if (!envTableBody) return;

    const emptyTr = envTableBody.querySelector('.t-td-empty');
    if (emptyTr && emptyTr.parentElement) emptyTr.parentElement.remove();

    const tr = document.createElement('tr');
    const sensitive = isSensitiveKey(key);

    tr.innerHTML = `
        <td>
            <input type="text" class="env-key-input" placeholder="VARIABLE_NAME" value="${escapeHtml(key || '')}" spellcheck="false">
        </td>
        <td>
            <div class="env-val-wrap">
                <input type="${sensitive ? 'password' : 'text'}" class="env-val-input" placeholder="valor" value="${escapeHtml(val || '')}" spellcheck="false">
                <button type="button" class="env-toggle-mask" title="${sensitive ? 'Mostrar valor' : 'Ocultar valor'}">
                    ${sensitive ? '👁️' : '🔒'}
                </button>
            </div>
        </td>
        <td style="text-align: center;">
            <button type="button" class="env-del-btn" title="Eliminar variable">✕</button>
        </td>
    `;

    const maskBtn = tr.querySelector('.env-toggle-mask');
    const valInput = tr.querySelector('.env-val-input');
    maskBtn.addEventListener('click', () => {
        const isPass = valInput.type === 'password';
        valInput.type = isPass ? 'text' : 'password';
        maskBtn.textContent = isPass ? '🔒' : '👁️';
        maskBtn.title = isPass ? 'Ocultar valor' : 'Mostrar valor';
    });

    const delBtn = tr.querySelector('.env-del-btn');
    delBtn.addEventListener('click', () => {
        tr.remove();
        if (envTableBody.children.length === 0) {
            envTableBody.innerHTML = `<tr><td colspan="3" class="t-td-empty">Sin variables configuradas. Pulsa '+ Agregar Variable'.</td></tr>`;
        }
    });

    envTableBody.appendChild(tr);
}

export function collectEnvFromTable() {
    const envTableBody = document.getElementById('envTableBody');
    if (!envTableBody) return '';
    const rows = envTableBody.querySelectorAll('tr');
    const lines = [];

    rows.forEach(tr => {
        const keyInput = tr.querySelector('.env-key-input');
        const valInput = tr.querySelector('.env-val-input');
        if (keyInput && valInput) {
            const k = keyInput.value.trim();
            const v = valInput.value.trim();
            if (k) {
                lines.push(`${k}=${v}`);
            }
        }
    });

    return lines.join('\n');
}

export function setupEnvListeners(onEnvSavedCallback) {
    const envModal = document.getElementById('envModal');
    const envTabTable = document.getElementById('envTabTable');
    const envTabRaw = document.getElementById('envTabRaw');
    const envTableView = document.getElementById('envTableView');
    const envRawView = document.getElementById('envRawView');
    const envRawTextarea = document.getElementById('envRawTextarea');
    const envTableBody = document.getElementById('envTableBody');
    const btnAddEnvRow = document.getElementById('btnAddEnvRow');
    const btnCopyEnv = document.getElementById('btnCopyEnv');
    const btnSaveEnv = document.getElementById('btnSaveEnv');
    const btnCloseEnvModal = document.getElementById('btnCloseEnvModal');
    const btnCancelEnv = document.getElementById('btnCancelEnv');
    const envModalStatus = document.getElementById('envModalStatus');

    if (envTabTable) {
        envTabTable.addEventListener('click', () => {
            if (currentEnvMode === 'raw' && envRawTextarea) {
                renderEnvTableFromRaw(envRawTextarea.value);
            }
            currentEnvMode = 'table';
            envTabTable.classList.add('active');
            if (envTabRaw) envTabRaw.classList.remove('active');
            if (envTableView) envTableView.style.display = 'block';
            if (envRawView) envRawView.style.display = 'none';
        });
    }

    if (envTabRaw) {
        envTabRaw.addEventListener('click', () => {
            if (currentEnvMode === 'table' && envRawTextarea) {
                envRawTextarea.value = collectEnvFromTable();
            }
            currentEnvMode = 'raw';
            envTabRaw.classList.add('active');
            if (envTabTable) envTabTable.classList.remove('active');
            if (envRawView) envRawView.style.display = 'block';
            if (envTableView) envTableView.style.display = 'none';
        });
    }

    if (btnAddEnvRow) {
        btnAddEnvRow.addEventListener('click', () => {
            if (currentEnvMode === 'raw' && envTabTable) {
                envTabTable.click();
            }
            createEnvTableRow('', '');
            if (envTableBody) {
                const inputs = envTableBody.querySelectorAll('.env-key-input');
                if (inputs.length > 0) {
                    inputs[inputs.length - 1].focus();
                }
            }
        });
    }

    if (btnCopyEnv) {
        btnCopyEnv.addEventListener('click', async () => {
            const textToCopy = currentEnvMode === 'raw' && envRawTextarea ? envRawTextarea.value : collectEnvFromTable();
            if (!textToCopy) {
                showToast('No hay variables para copiar', 'info');
                return;
            }
            const ok = await copyToClipboard(textToCopy);
            if (ok) {
                showToast('Variables .env copiadas al portapapeles', 'success');
                return;
            }
            showToast('No se pudo copiar automáticamente', 'error');
        });
    }

    if (btnSaveEnv) {
        btnSaveEnv.addEventListener('click', async () => {
            if (!currentEnvServiceName) return;
            const content = currentEnvMode === 'raw' && envRawTextarea ? envRawTextarea.value : collectEnvFromTable();

            btnSaveEnv.disabled = true;
            const originalText = btnSaveEnv.innerHTML;
            btnSaveEnv.innerHTML = '<span>⏳ Guardando...</span>';
            if (envModalStatus) envModalStatus.textContent = 'Actualizando servicio en Swarm...';

            try {
                const res = await apiFetch('/api/env', {
                    method: 'POST',
                    body: JSON.stringify({
                        serviceName: currentEnvServiceName,
                        rawContent: content
                    })
                });

                if (!res.ok) {
                    const errText = await res.text();
                    showToast(`Error al guardar variables: ${errText}`, 'error');
                    if (envModalStatus) envModalStatus.textContent = `Error: ${errText}`;
                    return;
                }

                showToast(`¡Variables de entorno de '${currentEnvServiceName}' actualizadas!`, 'success');
                closeEnvModal();
                if (onEnvSavedCallback) {
                    await onEnvSavedCallback();
                }
            } catch (err) {
                showToast(`Fallo de conexión: ${err.message}`, 'error');
                if (envModalStatus) envModalStatus.textContent = `Fallo: ${err.message}`;
            } finally {
                btnSaveEnv.disabled = false;
                btnSaveEnv.innerHTML = originalText;
            }
        });
    }

    if (btnCloseEnvModal) btnCloseEnvModal.addEventListener('click', closeEnvModal);
    if (btnCancelEnv) btnCancelEnv.addEventListener('click', closeEnvModal);
    if (envModal) {
        envModal.addEventListener('click', (e) => {
            if (e.target === envModal) closeEnvModal();
        });
    }
}
