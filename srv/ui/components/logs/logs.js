import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, debounce, copyToClipboard } from '/pkg/jsutil/utils.js';

let currentLogsServiceName = '';
let logsPollTimer = null;
let rawLogsText = '';

export function ensureLogsModalMounted() {
    if (document.getElementById('logsModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Visor de Logs de Contenedores y Servicios Docker -->
<div class="t-modal-overlay" id="logsModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card logs-modal-card">
        <div class="t-modal-header">
            <div class="logs-header-title-group">
                <span class="ops-status-dot status-online logs-live-pulse" id="logsStatusDot"></span>
                <div>
                    <h2 class="t-modal-title">Logs: <span id="logsServiceNameTitle" style="color:var(--brand-primary); font-family:var(--font-mono);">—</span></h2>
                    <span class="t-modal-desc">Salida en tiempo real de stdout / stderr del servicio Docker</span>
                </div>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseLogsModal" aria-label="Cerrar">&times;</button>
        </div>

        <!-- Toolbar de Logs -->
        <div class="logs-toolbar">
            <div class="logs-toolbar-left">
                <div class="logs-search-wrap">
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                    <input type="text" id="logsSearchInput" class="logs-search-input" placeholder="Filtrar logs (ej. error, 404, warn)..." aria-label="Filtrar logs">
                </div>
                <div class="logs-select-wrap">
                    <label for="logsTailSelect" class="logs-select-label">Líneas:</label>
                    <select id="logsTailSelect" class="logs-select" aria-label="Número de líneas">
                        <option value="50">50</option>
                        <option value="100" selected>100</option>
                        <option value="250">250</option>
                        <option value="500">500</option>
                        <option value="1000">1000</option>
                    </select>
                </div>
            </div>

            <div class="logs-toolbar-right">
                <label class="t-checkbox-label logs-toggle-label" title="Sondeo automático cada 3 segundos">
                    <input type="checkbox" id="logsLiveToggle" checked>
                    <span class="live-indicator-badge">● En Vivo</span>
                </label>
                <label class="t-checkbox-label logs-toggle-label" title="Desplazar automáticamente hacia el final">
                    <input type="checkbox" id="logsAutoscrollToggle" checked>
                    <span>Auto-scroll</span>
                </label>
                <button type="button" class="mini-btn" id="btnRestartFromLogs" title="Reiniciar este servicio ahora">
                    <span>🔄 Reiniciar</span>
                </button>
                <button type="button" class="mini-btn" id="btnRefreshLogs" title="Actualizar logs ahora">
                    <span>⟳ Refrescar</span>
                </button>
                <button type="button" class="mini-btn" id="btnCopyLogs" title="Copiar logs al portapapeles">
                    <span>📋 Copiar</span>
                </button>
                <button type="button" class="mini-btn mini-btn-accent" id="btnDownloadLogs" title="Descargar archivo de log">
                    <span>⬇️ Descargar</span>
                </button>
            </div>
        </div>

        <!-- Viewport de Terminal de Logs -->
        <div class="logs-terminal-viewport" id="logsTerminalViewport" tabindex="0">
            <pre id="logsTerminalContent" class="logs-terminal-content">Cargando logs del contenedor...</pre>
        </div>

        <!-- Footer info -->
        <div class="logs-footer-bar">
            <div class="logs-footer-stats">
                <span id="logsLineCount">0 líneas</span>
                <span class="meta-dot">·</span>
                <span id="logsLastUpdate">Actualizado: —</span>
            </div>
            <button type="button" class="t-btn t-btn-secondary t-btn-sm" id="btnDismissLogs">Cerrar</button>
        </div>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
}

export function openLogsModal(serviceName) {
    ensureLogsModalMounted();
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
        const res = await apiFetch(`/api/logs?name=${encodeURIComponent(currentLogsServiceName)}&lines=${encodeURIComponent(lines)}&server=${encodeURIComponent(state.selectedServerName || '')}`);
        if (res.ok) {
            const data = res.data;
            rawLogsText = (data && typeof data.logs === 'string') ? data.logs : (typeof data === 'string' ? data : '');
            renderFilteredLogs();
            if (logsLastUpdate) logsLastUpdate.textContent = `Actualizado: ${new Date().toLocaleTimeString()}`;
            if (logsStatusDot) logsStatusDot.className = 'ops-status-dot status-online logs-live-pulse';
        } else {
            const errText = res.error || `Error HTTP ${res.status}`;
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
    ensureLogsModalMounted();

    const btnCloseLogsModal = document.getElementById('btnCloseLogsModal');
    const btnDismissLogs = document.getElementById('btnDismissLogs');
    const logsModal = document.getElementById('logsModal');
    const btnRefreshLogs = document.getElementById('btnRefreshLogs');
    const btnRestartFromLogs = document.getElementById('btnRestartFromLogs');
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

    if (btnRestartFromLogs) {
        btnRestartFromLogs.addEventListener('click', async () => {
            if (!currentLogsServiceName) return;
            const { restartServiceOrContainer } = await import('/components/services/services.js');
            await restartServiceOrContainer(currentLogsServiceName, btnRestartFromLogs);
            fetchAndRenderLogs(false);
        });
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

