import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, debounce, copyToClipboard } from '/pkg/jsutil/utils.js';

let currentLogsServiceName = '';
let logsPollTimer = null;
let rawLogsText = '';

export function openLogsModal(serviceName) {
    const logsModal = document.getElementById('logsModal');
    const logsServiceNameTitle = document.getElementById('logsServiceNameTitle');
    const logsSearchInput = document.getElementById('logsSearchInput');
    const logsTerminalContent = document.getElementById('logsTerminalContent');

    if (!logsModal) return;
    currentLogsServiceName = serviceName;
    if (logsServiceNameTitle) logsServiceNameTitle.textContent = serviceName;
    if (logsSearchInput) logsSearchInput.value = '';
    if (logsTerminalContent) logsTerminalContent.textContent = 'Consultando logs del contenedor...';
    logsModal.style.display = 'flex';
    fetchAndRenderLogs(false);
    startLogsPolling();
}

export function closeLogsModal() {
    stopLogsPolling();
    const logsModal = document.getElementById('logsModal');
    if (logsModal) logsModal.style.display = 'none';
    currentLogsServiceName = '';
    rawLogsText = '';
}

export function startLogsPolling() {
    stopLogsPolling();
    const logsLiveToggle = document.getElementById('logsLiveToggle');
    const logsModal = document.getElementById('logsModal');
    if (logsLiveToggle && logsLiveToggle.checked) {
        logsPollTimer = setInterval(() => {
            if (logsModal && logsModal.style.display !== 'none' && currentLogsServiceName) {
                fetchAndRenderLogs(true);
            } else {
                stopLogsPolling();
            }
        }, 3000);
    }
}

export function stopLogsPolling() {
    if (logsPollTimer) {
        clearInterval(logsPollTimer);
        logsPollTimer = null;
    }
}

export function renderFilteredLogs() {
    const logsTerminalContent = document.getElementById('logsTerminalContent');
    const logsSearchInput = document.getElementById('logsSearchInput');
    const logsLineCount = document.getElementById('logsLineCount');
    const logsAutoscrollToggle = document.getElementById('logsAutoscrollToggle');
    const logsTerminalViewport = document.getElementById('logsTerminalViewport');

    if (!logsTerminalContent) return;
    const query = (logsSearchInput ? logsSearchInput.value : '').trim().toLowerCase();
    if (!rawLogsText) {
        logsTerminalContent.textContent = '(Sin registros recibidos)';
        if (logsLineCount) logsLineCount.textContent = '0 líneas';
        return;
    }

    const lines = rawLogsText.split('\n');
    if (logsLineCount) {
        logsLineCount.textContent = `${lines.length} líneas`;
    }

    if (!query) {
        logsTerminalContent.textContent = rawLogsText;
    } else {
        const matched = lines.filter(line => line.toLowerCase().includes(query));
        if (matched.length === 0) {
            logsTerminalContent.innerHTML = `<span style="color:var(--text-muted); font-style:italic;">No se encontraron registros para "${escapeHtml(query)}"</span>`;
            return;
        }
        const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(${escapedQuery})`, 'gi');
        const highlighted = matched.map(line => {
            return escapeHtml(line).replace(regex, '<mark>$1</mark>');
        }).join('\n');
        logsTerminalContent.innerHTML = highlighted;
    }

    if (logsAutoscrollToggle && logsAutoscrollToggle.checked && logsTerminalViewport) {
        logsTerminalViewport.scrollTop = logsTerminalViewport.scrollHeight;
    }
}

export async function fetchAndRenderLogs(isPoll) {
    if (!currentLogsServiceName) return;
    const logsTailSelect = document.getElementById('logsTailSelect');
    const logsTerminalContent = document.getElementById('logsTerminalContent');
    const logsLastUpdate = document.getElementById('logsLastUpdate');
    const logsStatusDot = document.getElementById('logsStatusDot');

    const lines = logsTailSelect ? logsTailSelect.value : '100';

    try {
        const res = await apiFetch(`/api/logs?name=${encodeURIComponent(currentLogsServiceName)}&lines=${encodeURIComponent(lines)}`);
        if (res.ok) {
            const data = await res.json();
            rawLogsText = (data && typeof data.logs === 'string') ? data.logs : '';
            renderFilteredLogs();
            if (logsLastUpdate) logsLastUpdate.textContent = `Actualizado: ${new Date().toLocaleTimeString()}`;
            if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-online logs-live-pulse';
        } else {
            const errText = await res.text();
            if (!isPoll && logsTerminalContent) {
                logsTerminalContent.textContent = `Error al consultar logs: ${errText}`;
            }
            if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-offline';
        }
    } catch (err) {
        if (!isPoll && logsTerminalContent) {
            logsTerminalContent.textContent = `Fallo de conexión: ${err.message}`;
        }
        if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-offline';
    }
}

export function setupLogsListeners() {
    const btnCloseLogsModal = document.getElementById('btnCloseLogsModal');
    const btnDismissLogs = document.getElementById('btnDismissLogs');
    const logsModal = document.getElementById('logsModal');
    const btnRefreshLogs = document.getElementById('btnRefreshLogs');
    const logsTailSelect = document.getElementById('logsTailSelect');
    const logsLiveToggle = document.getElementById('logsLiveToggle');
    const logsSearchInput = document.getElementById('logsSearchInput');
    const btnCopyLogs = document.getElementById('btnCopyLogs');
    const btnDownloadLogs = document.getElementById('btnDownloadLogs');

    if (btnCloseLogsModal) btnCloseLogsModal.addEventListener('click', closeLogsModal);
    if (btnDismissLogs) btnDismissLogs.addEventListener('click', closeLogsModal);
    if (logsModal) {
        logsModal.addEventListener('click', (e) => {
            if (e.target === logsModal) closeLogsModal();
        });
    }

    if (btnRefreshLogs) {
        btnRefreshLogs.addEventListener('click', () => fetchAndRenderLogs(false));
    }

    if (logsTailSelect) {
        logsTailSelect.addEventListener('change', () => fetchAndRenderLogs(false));
    }

    if (logsLiveToggle) {
        logsLiveToggle.addEventListener('change', () => {
            if (logsLiveToggle.checked) {
                startLogsPolling();
            } else {
                stopLogsPolling();
            }
        });
    }

    if (logsSearchInput) {
        logsSearchInput.addEventListener('input', debounce(() => renderFilteredLogs(), 120));
    }

    if (btnCopyLogs) {
        btnCopyLogs.addEventListener('click', async () => {
            if (!rawLogsText) {
                showToast('No hay logs para copiar', 'info');
                return;
            }
            const ok = await copyToClipboard(rawLogsText);
            if (ok) {
                showToast('Logs copiados al portapapeles', 'success');
                return;
            }
            showToast('No se pudo copiar automáticamente', 'error');
        });
    }

    if (btnDownloadLogs) {
        btnDownloadLogs.addEventListener('click', () => {
            if (!rawLogsText) {
                showToast('No hay logs para descargar', 'info');
                return;
            }
            const blob = new Blob([rawLogsText], { type: 'text/plain;charset=utf-8' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${currentLogsServiceName || 'service'}_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        });
    }
}
