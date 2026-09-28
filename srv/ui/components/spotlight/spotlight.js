/**
 * Tarhiata Cloud Studio — Spotlight Command Palette Component (⌘K / Ctrl+K)
 * Fast, keyboard-driven action dispatcher and entity search.
 */

import { state } from '/pkg/store/state.js';
import { openModal, closeModal } from '/pkg/modal/modal.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';
import { selectServer } from '/components/fleet/fleet.js';
import { openLogsModal } from '/components/logs/logs.js';
import { openEnvModal } from '/components/env/env.js';
import { requestNotificationPermission } from '/pkg/notify/notify.js';

let selectedIndex = 0;
let currentItems = [];

export function ensureSpotlightMounted() {
    if (document.getElementById('spotlightModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
        <div class="t-modal-overlay spotlight-overlay" id="spotlightModal" style="display:none;" role="dialog" aria-modal="true" aria-label="Paleta de Comandos">
            <div class="spotlight-card">
                <div class="spotlight-search-header">
                    <svg class="spotlight-search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                        <circle cx="11" cy="11" r="8"></circle>
                        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                    </svg>
                    <input type="text" id="spotlightInput" class="spotlight-input" placeholder="Buscar servidores, aplicaciones, bases de datos o comandos..." autocomplete="off" spellcheck="false">
                    <kbd class="spotlight-esc-hint">ESC</kbd>
                </div>

                <div class="spotlight-results" id="spotlightResults"></div>

                <div class="spotlight-footer">
                    <div class="spotlight-hints">
                        <span><kbd>↑</kbd> <kbd>↓</kbd> Navegar</span>
                        <span><kbd>↵</kbd> Ejecutar</span>
                        <span><kbd>ESC</kbd> Cerrar</span>
                    </div>
                    <div class="spotlight-brand">
                        <span>Tarhiata Spotlight (⌘K)</span>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.appendChild(div.firstElementChild);
}

export function openSpotlight() {
    ensureSpotlightMounted();
    const modal = document.getElementById('spotlightModal');
    const input = document.getElementById('spotlightInput');
    if (!modal || !input) return;

    modal.style.display = 'flex';
    input.value = '';
    selectedIndex = 0;
    renderSpotlightResults('');
    setTimeout(() => input.focus(), 50);
}

export function closeSpotlight() {
    const modal = document.getElementById('spotlightModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

export function buildSpotlightRegistry() {
    const items = [];

    // 1. Acciones Rápidas
    items.push({
        id: 'action-deploy-app',
        category: 'Acciones Rápidas',
        icon: '🚀',
        title: 'Desplegar Aplicación Docker',
        subtitle: 'Crear o actualizar un contenedor con proxy Traefik y SSL',
        shortcut: 'D',
        handler: () => {
            const btn = document.getElementById('btnOpenDeployModal') || document.getElementById('btnGlobalDeploy');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-create-db',
        category: 'Acciones Rápidas',
        icon: '🗄️',
        title: 'Crear Base de Datos',
        subtitle: 'PostgreSQL, MySQL, Redis, MongoDB o MinIO S3',
        shortcut: 'B',
        handler: () => {
            const btn = document.getElementById('btnOpenDeployDBModal');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-connect-vps',
        category: 'Acciones Rápidas',
        icon: '🖥️',
        title: 'Conectar Servidor VPS',
        subtitle: 'Añadir un nuevo host por SSH o arrancar local',
        shortcut: 'N',
        handler: () => {
            const btn = document.getElementById('btnOpenAddModal');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-cloud-worker',
        category: 'Acciones Rápidas',
        icon: '☁️',
        title: 'Añadir Cloud Worker (Vultr / OpenTofu)',
        subtitle: 'Aprovisionar nuevo nodo worker automáticamente',
        handler: () => {
            const btn = document.getElementById('btnSidebarOpenWorker') || document.getElementById('btnOpenWorkerModal');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-open-terminal',
        category: 'Acciones Rápidas',
        icon: '💻',
        title: 'Abrir Terminal SSH Interactiva',
        subtitle: 'Consola web en el servidor activo',
        shortcut: 'T',
        handler: () => {
            const btn = document.getElementById('btnTopActiveTerminal') || document.getElementById('btnDeskTerminal');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-open-ssl',
        category: 'Acciones Rápidas',
        icon: '🔒',
        title: 'SSL & Modo Mantenimiento',
        subtitle: 'Diagnóstico de certificados y recarga de Traefik',
        handler: () => {
            const btn = document.getElementById('btnTopSSL') || document.getElementById('btnDeskSSL');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-test-all',
        category: 'Acciones Rápidas',
        icon: '📡',
        title: 'Sondear Red (Test Conectividad)',
        subtitle: 'Verificar latencia y estado de todos los servidores',
        handler: () => {
            const btn = document.getElementById('btnTestAll');
            if (btn) btn.click();
        }
    });

    items.push({
        id: 'action-system-report',
        category: 'Acciones Rápidas',
        icon: '📊',
        title: 'Descargar Reporte de Diagnóstico (.md)',
        subtitle: 'Generar volcado de telemetría, seguridad UFW, contenedores y BDs',
        handler: () => {
            downloadSystemReport();
        }
    });

    items.push({
        id: 'action-enable-notif',
        category: 'Acciones Rápidas',
        icon: '🔔',
        title: 'Habilitar Notificaciones de Escritorio',
        subtitle: 'Alertas del sistema cuando finalicen despliegues',
        handler: async () => {
            await requestNotificationPermission();
        }
    });

    // 2. Navegación entre Pestañas
    items.push({
        id: 'nav-tab-services',
        category: 'Navegación',
        icon: '📦',
        title: 'Ver Aplicaciones y Clúster Swarm',
        subtitle: 'Pestaña principal de contenedores y nodos',
        handler: () => {
            if (window.activateTab) window.activateTab('services');
        }
    });

    items.push({
        id: 'nav-tab-databases',
        category: 'Navegación',
        icon: '🗄️',
        title: 'Ver Bases de Datos',
        subtitle: 'Instancias de BD y copias de seguridad',
        handler: () => {
            if (window.activateTab) window.activateTab('databases');
        }
    });

    items.push({
        id: 'nav-tab-host',
        category: 'Navegación',
        icon: '⚙️',
        title: 'Ver Servicios del VPS (Systemd)',
        subtitle: 'Demonios y procesos del sistema operativo',
        handler: () => {
            if (window.activateTab) window.activateTab('host');
        }
    });

    items.push({
        id: 'nav-tab-devices',
        category: 'Navegación',
        icon: '🔌',
        title: 'Ver Dispositivos & Hardware',
        subtitle: 'Almacenamiento, GPUs, PCI y periféricos del host',
        handler: () => {
            if (window.activateTab) window.activateTab('devices');
        }
    });

    // 3. Servidores VPS (Flota)
    if (Array.isArray(state.servers)) {
        state.servers.forEach(s => {
            const isSelected = s.name === state.selectedServerName;
            items.push({
                id: `server-${s.name}`,
                category: 'Servidores VPS',
                icon: '🖥️',
                title: `Conmutar a servidor: ${s.name}`,
                subtitle: `Host: ${s.host} ${s.isActive ? '★ PRINCIPAL' : ''} ${isSelected ? '(Seleccionado)' : ''}`,
                handler: () => {
                    selectServer(s.name);
                }
            });
        });
    }

    // 4. Servicios Swarm en Ejecución
    if (Array.isArray(state.swarmServicesCache)) {
        state.swarmServicesCache.forEach(svc => {
            items.push({
                id: `svc-logs-${svc.Name}`,
                category: 'Servicios Docker',
                icon: '📜',
                title: `Ver Logs: ${svc.Name}`,
                subtitle: `Imagen: ${svc.Image || 'docker'} · Replicas: ${svc.Replicas || '1/1'}`,
                handler: () => {
                    openLogsModal(svc.Name, 'service');
                }
            });

            items.push({
                id: `svc-env-${svc.Name}`,
                category: 'Servicios Docker',
                icon: '🔑',
                title: `Editar Variables .env: ${svc.Name}`,
                subtitle: `Variables de entorno en caliente`,
                handler: () => {
                    openEnvModal(svc.Name);
                }
            });
        });
    }

    // 5. Bases de Datos en Ejecución
    if (Array.isArray(state.swarmDatabasesCache)) {
        state.swarmDatabasesCache.forEach(db => {
            items.push({
                id: `db-logs-${db.Name}`,
                category: 'Bases de Datos',
                icon: '🗄️',
                title: `Ver Logs BD: ${db.Name}`,
                subtitle: `Motor: ${(db.Engine || 'postgres').toUpperCase()} · Puerto: ${db.InternalPort || 5432}`,
                handler: () => {
                    openLogsModal(`tarhiata-db-${db.Name}`, 'database');
                }
            });
        });
    }

    return items;
}

export function renderSpotlightResults(query = '') {
    const container = document.getElementById('spotlightResults');
    if (!container) return;

    const allItems = buildSpotlightRegistry();
    const cleanQ = query.toLowerCase().trim();

    currentItems = allItems.filter(item => {
        if (!cleanQ) return true;
        return item.title.toLowerCase().includes(cleanQ) ||
               item.subtitle.toLowerCase().includes(cleanQ) ||
               item.category.toLowerCase().includes(cleanQ);
    });

    if (currentItems.length === 0) {
        container.innerHTML = `
            <div class="spotlight-empty">
                <span>No se encontraron comandos o servicios para "<strong>${escapeHtml(query)}</strong>"</span>
            </div>
        `;
        return;
    }

    if (selectedIndex >= currentItems.length) {
        selectedIndex = 0;
    }

    // Agrupar por categoría
    const groups = {};
    currentItems.forEach((item, idx) => {
        if (!groups[item.category]) groups[item.category] = [];
        groups[item.category].push({ item, globalIndex: idx });
    });

    let html = '';
    for (const [catName, entries] of Object.entries(groups)) {
        html += `<div class="spotlight-category-title">${escapeHtml(catName)}</div>`;
        entries.forEach(({ item, globalIndex }) => {
            const isSel = (globalIndex === selectedIndex);
            html += `
                <div class="spotlight-item ${isSel ? 'selected' : ''}" data-index="${globalIndex}">
                    <div class="spotlight-item-left">
                        <span class="spotlight-item-icon">${item.icon}</span>
                        <div class="spotlight-item-meta">
                            <span class="spotlight-item-title">${escapeHtml(item.title)}</span>
                            <span class="spotlight-item-sub">${escapeHtml(item.subtitle)}</span>
                        </div>
                    </div>
                    ${item.shortcut ? `<kbd class="spotlight-item-shortcut">${escapeHtml(item.shortcut)}</kbd>` : ''}
                </div>
            `;
        });
    }

    container.innerHTML = html;

    // Scroll elemento seleccionado a la vista
    const activeEl = container.querySelector('.spotlight-item.selected');
    if (activeEl) {
        activeEl.scrollIntoView({ block: 'nearest' });
    }

    // Eventos de click en items
    container.querySelectorAll('.spotlight-item').forEach(el => {
        el.addEventListener('click', () => {
            const idx = parseInt(el.getAttribute('data-index'), 10);
            executeSpotlightIndex(idx);
        });
        el.addEventListener('mouseenter', () => {
            selectedIndex = parseInt(el.getAttribute('data-index'), 10);
            updateSpotlightSelection();
        });
    });
}

function updateSpotlightSelection() {
    const container = document.getElementById('spotlightResults');
    if (!container) return;
    container.querySelectorAll('.spotlight-item').forEach(el => {
        const idx = parseInt(el.getAttribute('data-index'), 10);
        el.classList.toggle('selected', idx === selectedIndex);
    });
    const activeEl = container.querySelector('.spotlight-item.selected');
    if (activeEl) {
        activeEl.scrollIntoView({ block: 'nearest' });
    }
}

function executeSpotlightIndex(idx) {
    if (idx >= 0 && idx < currentItems.length) {
        const item = currentItems[idx];
        closeSpotlight();
        if (item.handler) {
            item.handler();
        }
    }
}

export function downloadSystemReport() {
    const serverName = state.selectedServerName || '';
    showToast('Generando reporte de diagnóstico del sistema...', 'info');
    const url = `/api/system/report?server=${encodeURIComponent(serverName)}&format=markdown`;
    
    // Disparar descarga directa del archivo markdown
    const a = document.createElement('a');
    a.href = url;
    a.download = `tarhiata-diagnostic-${serverName || 'host'}.md`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    showToast('Reporte de diagnóstico descargado con éxito.', 'success');
}

export function setupSpotlightListeners() {
    ensureSpotlightMounted();

    // 1. Teclas globales (⌘K / Ctrl+K)
    window.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
            e.preventDefault();
            const modal = document.getElementById('spotlightModal');
            if (modal && modal.style.display !== 'none') {
                closeSpotlight();
            } else {
                openSpotlight();
            }
        }
    });

    // 2. Input y navegación en Spotlight
    const modal = document.getElementById('spotlightModal');
    const input = document.getElementById('spotlightInput');

    if (input) {
        input.addEventListener('input', (e) => {
            selectedIndex = 0;
            renderSpotlightResults(e.target.value);
        });

        input.addEventListener('keydown', (e) => {
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                if (currentItems.length > 0) {
                    selectedIndex = (selectedIndex + 1) % currentItems.length;
                    updateSpotlightSelection();
                }
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                if (currentItems.length > 0) {
                    selectedIndex = (selectedIndex - 1 + currentItems.length) % currentItems.length;
                    updateSpotlightSelection();
                }
            } else if (e.key === 'Enter') {
                e.preventDefault();
                executeSpotlightIndex(selectedIndex);
            } else if (e.key === 'Escape') {
                e.preventDefault();
                closeSpotlight();
            }
        });
    }

    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                closeSpotlight();
            }
        });
    }

    // Botón de trigger en navbar si existe
    const btnNavSpotlight = document.getElementById('btnNavSpotlight');
    if (btnNavSpotlight) {
        btnNavSpotlight.addEventListener('click', () => openSpotlight());
    }
}
