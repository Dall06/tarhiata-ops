import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, copyToClipboard } from '/pkg/jsutil/utils.js';

let terminalActiveServer = '';
let terminalCommandHistory = [];
let terminalHistoryIndex = -1;
let isTerminalExecuting = false;

export async function launchNativeTerminal(serverName, buttonEl) {
    const btnDeskTerminal = document.getElementById('btnDeskTerminal');
    const btn = buttonEl || btnDeskTerminal;
    let origContent = '';
    if (btn) {
        origContent = btn.innerHTML;
        btn.disabled = true;
        btn.innerHTML = '<span>...</span>';
    }

    try {
        const res = await apiFetch('/api/servers/open-terminal', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: serverName })
        });

        if (!res.ok) {
            showToast(`Error abriendo terminal de ${serverName}: ${res.error || res.status}`, 'error');
            return;
        }

        const data = res.data || {};
        if (data.launched) {
            showToast(`Terminal abierta en tu sistema (${serverName})`, 'success');
            return;
        }
        showToast(data.message || `Terminal lanzada`, 'info');
    } catch (err) {
        showToast(`Fallo al solicitar apertura de terminal: ${err.message}`, 'error');
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.innerHTML = origContent;
        }
    }
}

export function ensureTerminalModalMounted() {
    if (document.getElementById('terminalModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `<div class="t-modal-overlay" id="terminalModal" style="display:none;" role="dialog" aria-modal="true">
        <div class="t-modal-card terminal-modal-card">
            <div class="t-modal-header terminal-modal-header">
                <div style="display:flex; align-items:center; gap:10px;">
                    <div class="terminal-badge-pulse"></div>
                    <div>
                        <h2 class="t-modal-title">💻 Consola Terminal Web</h2>
                        <span class="t-modal-desc">Sesión interactiva en <strong id="terminalHostLabel" style="color:var(--text-pure); font-family:var(--font-mono);">—</strong></span>
                    </div>
                </div>
                <div style="display:flex; align-items:center; gap:8px;">
                    <button type="button" class="mini-btn mini-btn-accent" id="btnTerminalOpenPC" title="Abrir esta sesión en la aplicación de Terminal de tu computadora">
                        🖥️ <span>Terminal del PC (Nativa)</span>
                    </button>
                    <button type="button" class="t-close-btn" id="btnCloseTerminalModal" aria-label="Cerrar">&times;</button>
                </div>
            </div>

            <div class="terminal-toolbar">
                <div class="terminal-toolbar-left">
                    <span class="terminal-target-label">Destino:</span>
                    <select id="terminalTargetSelect" class="t-input terminal-target-select">
                        <option value="">🖥️ Host VPS (Sistema Operativo)</option>
                    </select>
                </div>
                <div class="terminal-toolbar-right">
                    <button type="button" class="mini-btn btn-term-quick" data-cmd="docker ps">docker ps</button>
                    <button type="button" class="mini-btn btn-term-quick" data-cmd="df -h">df -h</button>
                    <button type="button" class="mini-btn btn-term-quick" data-cmd="free -m">free -m</button>
                    <button type="button" class="mini-btn btn-term-quick" data-cmd="uptime">uptime</button>
                    <button type="button" class="mini-btn" id="btnTerminalClear" title="Limpiar pantalla">🧹 Limpiar</button>
                    <button type="button" class="mini-btn" id="btnTerminalCopy" title="Copiar salida de la terminal">📋 Copiar</button>
                </div>
            </div>

            <div id="terminalFallbackBanner" class="terminal-fallback-banner" style="display:none;">
                <div style="display:flex; align-items:center; gap:10px;">
                    <span style="font-size:1.1rem;">⚠️</span>
                    <div>
                        <strong style="color:#fca5a5;">Problema al conectar en el navegador:</strong>
                        <span id="terminalFallbackMsg" style="color:var(--text-secondary); font-size:0.78rem;"></span>
                    </div>
                </div>
                <button type="button" class="t-btn t-btn-primary" id="btnFallbackOpenPC">
                    🖥️ Conectar con Terminal del PC
                </button>
            </div>

            <div class="terminal-viewport" id="terminalViewport" tabindex="0">
                <div id="terminalOutput" class="terminal-output-stream"></div>
            </div>

            <div class="terminal-prompt-bar">
                <span class="terminal-prompt-prefix" id="terminalPromptPrefix">root@vps:~$</span>
                <input type="text" id="terminalCommandInput" class="terminal-cmd-input" placeholder="Escribe un comando bash (ej: docker ps, uname -a, ls -la)..." autocomplete="off" spellcheck="false">
                <button type="button" class="t-btn t-btn-primary" id="btnTerminalSend">
                    <span>Ejecutar</span>
                </button>
            </div>

            <div class="t-modal-footer terminal-modal-footer">
                <span style="font-size:0.72rem; color:var(--text-muted); font-family:var(--font-mono);">
                    Atajos: <code>Enter</code> ejecutar · <code>↑</code> / <code>↓</code> historial · <code>Ctrl+L</code> limpiar
                </span>
                <div style="display:flex; gap:8px;">
                    <button type="button" class="t-btn t-btn-secondary" id="btnTerminalDirectPC">
                        🖥️ Terminal del PC
                    </button>
                    <button type="button" class="t-btn t-btn-secondary" id="btnCloseTerminalModalBottom">
                        Cerrar
                    </button>
                </div>
            </div>
        </div>
    </div>`;
    document.body.appendChild(div.firstElementChild);
    setupTerminalListeners();
}

export function openTerminalModal(serverName, targetContainer = '') {
    ensureTerminalModalMounted();
    const terminalModal = document.getElementById('terminalModal');
    const terminalHostLabel = document.getElementById('terminalHostLabel');
    const terminalFallbackBanner = document.getElementById('terminalFallbackBanner');
    const terminalTargetSelect = document.getElementById('terminalTargetSelect');
    const terminalCommandInput = document.getElementById('terminalCommandInput');
    const terminalOutput = document.getElementById('terminalOutput');

    if (!terminalModal) return;
    terminalActiveServer = serverName || state.selectedServerName || '';
    if (terminalHostLabel) terminalHostLabel.textContent = terminalActiveServer;

    if (terminalFallbackBanner) terminalFallbackBanner.style.display = 'none';

    if (terminalTargetSelect) {
        terminalTargetSelect.innerHTML = `<option value="">🖥️ Host VPS (${escapeHtml(terminalActiveServer)})</option>`;
        
        if (state.swarmServicesCache && state.swarmServicesCache.length > 0) {
            const groupSvc = document.createElement('optgroup');
            groupSvc.label = 'Servicios Swarm / Contenedores';
            state.swarmServicesCache.forEach(s => {
                const opt = document.createElement('option');
                opt.value = s.name;
                opt.textContent = `🐳 ${s.name}`;
                if (targetContainer && targetContainer === s.name) opt.selected = true;
                groupSvc.appendChild(opt);
            });
            terminalTargetSelect.appendChild(groupSvc);
        }

        if (state.swarmDatabasesCache && state.swarmDatabasesCache.length > 0) {
            const groupDb = document.createElement('optgroup');
            groupDb.label = 'Bases de Datos';
            state.swarmDatabasesCache.forEach(db => {
                const opt = document.createElement('option');
                opt.value = `tarhiata-db-${db.name}`;
                opt.textContent = `🗄️ ${db.name} (${db.engine || 'db'})`;
                if (targetContainer && (targetContainer === db.name || targetContainer === `tarhiata-db-${db.name}`)) opt.selected = true;
                groupDb.appendChild(opt);
            });
            terminalTargetSelect.appendChild(groupDb);
        }
    }

    updateTerminalPrompt();

    if (terminalOutput && terminalOutput.children.length === 0) {
        appendTerminalSystemNotice(`✨ Tarhiata-Ops Web Terminal v4.2 iniciada.\nConectado a: ${terminalActiveServer}\nEscribe comandos bash directamente o usa los atajos rápidos.\n👉 Puedes usar la Terminal de tu computadora en cualquier momento con "Terminal del PC".`);
    }

    terminalModal.style.display = 'flex';
    setTimeout(() => {
        if (terminalCommandInput) terminalCommandInput.focus();
    }, 80);
}

export function closeTerminalModal() {
    const terminalModal = document.getElementById('terminalModal');
    if (terminalModal) terminalModal.style.display = 'none';
}

export function updateTerminalPrompt() {
    const terminalPromptPrefix = document.getElementById('terminalPromptPrefix');
    const terminalTargetSelect = document.getElementById('terminalTargetSelect');
    if (!terminalPromptPrefix) return;
    const target = terminalTargetSelect ? terminalTargetSelect.value : '';
    if (target) {
        terminalPromptPrefix.textContent = `root@${target}:#`;
        terminalPromptPrefix.style.color = '#38bdf8';
    } else {
        terminalPromptPrefix.textContent = `root@${terminalActiveServer || 'vps'}:~$`;
        terminalPromptPrefix.style.color = '#10b981';
    }
}

function appendTerminalSystemNotice(text) {
    const terminalOutput = document.getElementById('terminalOutput');
    if (!terminalOutput) return;
    const div = document.createElement('div');
    div.className = 'terminal-line';
    div.style.color = '#94a3b8';
    div.style.fontStyle = 'italic';
    div.textContent = text;
    terminalOutput.appendChild(div);
    scrollTerminalToBottom();
}

function scrollTerminalToBottom() {
    const terminalViewport = document.getElementById('terminalViewport');
    if (terminalViewport) {
        terminalViewport.scrollTop = terminalViewport.scrollHeight;
    }
}

export async function executeTerminalCommand(cmdToRun) {
    const terminalCommandInput = document.getElementById('terminalCommandInput');
    const terminalOutput = document.getElementById('terminalOutput');
    const terminalTargetSelect = document.getElementById('terminalTargetSelect');
    const terminalPromptPrefix = document.getElementById('terminalPromptPrefix');
    const btnTerminalSend = document.getElementById('btnTerminalSend');
    const terminalFallbackBanner = document.getElementById('terminalFallbackBanner');
    const terminalFallbackMsg = document.getElementById('terminalFallbackMsg');

    const rawCmd = (cmdToRun !== undefined ? cmdToRun : (terminalCommandInput ? terminalCommandInput.value : '')).trim();
    if (!rawCmd || isTerminalExecuting) return;

    terminalCommandHistory.push(rawCmd);
    terminalHistoryIndex = terminalCommandHistory.length;

    if (terminalCommandInput) {
        terminalCommandInput.value = '';
    }

    if (rawCmd.toLowerCase() === 'clear') {
        if (terminalOutput) terminalOutput.innerHTML = '';
        return;
    }

    const targetContainer = terminalTargetSelect ? terminalTargetSelect.value : '';
    const promptLabel = terminalPromptPrefix ? terminalPromptPrefix.textContent : '$';

    if (terminalOutput) {
        const cmdLine = document.createElement('div');
        cmdLine.className = 'terminal-line terminal-line-cmd';
        cmdLine.innerHTML = `<span>${escapeHtml(promptLabel)}</span> <span>${escapeHtml(rawCmd)}</span>`;
        terminalOutput.appendChild(cmdLine);

        const runningIndicator = document.createElement('div');
        runningIndicator.className = 'terminal-line';
        runningIndicator.style.color = '#64748b';
        runningIndicator.textContent = '⏳ Ejecutando en servidor remoto...';
        terminalOutput.appendChild(runningIndicator);
        scrollTerminalToBottom();

        isTerminalExecuting = true;
        if (btnTerminalSend) btnTerminalSend.disabled = true;

        try {
            const res = await apiFetch('/api/servers/terminal', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: terminalActiveServer,
                    command: rawCmd,
                    container: targetContainer
                })
            });

            runningIndicator.remove();

            if (!res.ok) {
                throw new Error(res.error || `Error HTTP ${res.status}`);
            }

            const data = res.data || {};

            if (data.connected === false) {
                const errLine = document.createElement('div');
                errLine.className = 'terminal-line terminal-line-err';
                errLine.textContent = `❌ Fallo de conexión web: ${data.output || 'No se pudo conectar por SSH'}\n💡 Puedes conectar directamente abriendo la Terminal del PC.`;
                terminalOutput.appendChild(errLine);

                if (terminalFallbackBanner) {
                    terminalFallbackBanner.style.display = 'flex';
                    if (terminalFallbackMsg) {
                        terminalFallbackMsg.textContent = `${data.output || 'Fallo de conexión SSH'}.`;
                    }
                }
            } else {
                if (terminalFallbackBanner) terminalFallbackBanner.style.display = 'none';

                const outLine = document.createElement('div');
                outLine.className = data.exitCode !== 0 ? 'terminal-line terminal-line-err' : 'terminal-line terminal-line-out';
                outLine.textContent = data.output || '(sin salida)';
                terminalOutput.appendChild(outLine);
            }
        } catch (err) {
            runningIndicator.remove();
            const errLine = document.createElement('div');
            errLine.className = 'terminal-line terminal-line-err';
            errLine.textContent = `❌ Error al ejecutar en navegador: ${err.message}`;
            terminalOutput.appendChild(errLine);

            if (terminalFallbackBanner) {
                terminalFallbackBanner.style.display = 'flex';
                if (terminalFallbackMsg) {
                    terminalFallbackMsg.textContent = `${err.message}.`;
                }
            }
        } finally {
            isTerminalExecuting = false;
            if (btnTerminalSend) btnTerminalSend.disabled = false;
            scrollTerminalToBottom();
            if (terminalCommandInput) terminalCommandInput.focus();
        }
    }
}

export function setupTerminalListeners() {
    const btnTerminalOpenPC = document.getElementById('btnTerminalOpenPC');
    const btnTerminalDirectPC = document.getElementById('btnTerminalDirectPC');
    const btnFallbackOpenPC = document.getElementById('btnFallbackOpenPC');
    const btnDeskTerminal = document.getElementById('btnDeskTerminal');
    const btnDeskNativeTerminal = document.getElementById('btnDeskNativeTerminal');
    const btnTopActiveTerminal = document.getElementById('btnTopActiveTerminal');
    const terminalCommandInput = document.getElementById('terminalCommandInput');
    const btnTerminalSend = document.getElementById('btnTerminalSend');
    const btnTerminalClear = document.getElementById('btnTerminalClear');
    const btnTerminalCopy = document.getElementById('btnTerminalCopy');
    const terminalTargetSelect = document.getElementById('terminalTargetSelect');
    const btnCloseTerminalModal = document.getElementById('btnCloseTerminalModal');

    function triggerPCTerminal() {
        const targetServer = terminalActiveServer || state.selectedServerName || (state.activeServer && state.activeServer.name);
        if (!targetServer) {
            showToast('Selecciona un servidor primero.', 'error');
            return;
        }
        launchNativeTerminal(targetServer, btnTerminalOpenPC);
    }

    if (btnTerminalOpenPC) btnTerminalOpenPC.addEventListener('click', triggerPCTerminal);
    if (btnTerminalDirectPC) btnTerminalDirectPC.addEventListener('click', triggerPCTerminal);
    if (btnFallbackOpenPC) btnFallbackOpenPC.addEventListener('click', triggerPCTerminal);

    if (btnDeskTerminal) btnDeskTerminal.addEventListener('click', () => {
        openTerminalModal(state.selectedServerName);
    });
    if (btnTopActiveTerminal) btnTopActiveTerminal.addEventListener('click', () => {
        openTerminalModal(state.selectedServerName);
    });
    if (btnDeskNativeTerminal) btnDeskNativeTerminal.addEventListener('click', () => {
        const target = state.selectedServerName || (state.activeServer && state.activeServer.name);
        if (target) launchNativeTerminal(target, btnDeskNativeTerminal);
    });

    if (btnCloseTerminalModal) btnCloseTerminalModal.addEventListener('click', closeTerminalModal);

    document.querySelectorAll('.btn-term-quick').forEach(btn => {
        btn.addEventListener('click', () => {
            const cmd = btn.getAttribute('data-cmd');
            if (cmd) executeTerminalCommand(cmd);
        });
    });

    if (terminalCommandInput) {
        terminalCommandInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                executeTerminalCommand();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                if (terminalCommandHistory.length > 0 && terminalHistoryIndex > 0) {
                    terminalHistoryIndex--;
                    terminalCommandInput.value = terminalCommandHistory[terminalHistoryIndex];
                }
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                if (terminalHistoryIndex < terminalCommandHistory.length - 1) {
                    terminalHistoryIndex++;
                    terminalCommandInput.value = terminalCommandHistory[terminalHistoryIndex];
                } else {
                    terminalHistoryIndex = terminalCommandHistory.length;
                    terminalCommandInput.value = '';
                }
            } else if (e.ctrlKey && e.key === 'l') {
                e.preventDefault();
                const terminalOutput = document.getElementById('terminalOutput');
                if (terminalOutput) terminalOutput.innerHTML = '';
            }
        });
    }

    if (btnTerminalSend) {
        btnTerminalSend.addEventListener('click', () => executeTerminalCommand());
    }

    if (btnTerminalClear) {
        btnTerminalClear.addEventListener('click', () => {
            const terminalOutput = document.getElementById('terminalOutput');
            if (terminalOutput) terminalOutput.innerHTML = '';
            if (terminalCommandInput) terminalCommandInput.focus();
        });
    }

    if (btnTerminalCopy) {
        btnTerminalCopy.addEventListener('click', async () => {
            const terminalOutput = document.getElementById('terminalOutput');
            if (!terminalOutput) return;
            const text = terminalOutput.innerText || terminalOutput.textContent || '';
            const ok = await copyToClipboard(text);
            if (ok) {
                showToast('Salida de la terminal copiada al portapapeles', 'success');
            } else {
                showToast('No se pudo copiar el texto', 'error');
            }
        });
    }

    if (terminalTargetSelect) {
        terminalTargetSelect.addEventListener('change', () => {
            updateTerminalPrompt();
        });
    }
}
