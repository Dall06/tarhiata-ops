/**
 * Tarhiata Cloud Studio — Alert Settings Component (srv/ui/components/alerts)
 */

import { apiFetch } from '/pkg/apiclient/api.js';
import { showToast } from '/pkg/toast/toast.js';
import { escapeHtml } from '/pkg/jsutil/utils.js';

export function ensureAlertsModalMounted() {
    if (document.getElementById('alertsModal')) return;
    const div = document.createElement('div');
    div.innerHTML = `
<!-- Modal: Configuración de Alertas Salientes -->
<div class="t-modal-overlay" id="alertsModal" style="display:none;" role="dialog" aria-modal="true">
    <div class="t-modal-card alerts-modal-card" style="max-width:680px; width:92vw;">
        <div class="t-modal-header">
            <div style="display:flex; align-items:center; gap:10px;">
                <div class="modal-header-icon" style="background:rgba(245,158,11,0.15); color:var(--accent-warning, #f59e0b);">🔔</div>
                <div>
                    <h2 class="t-modal-title">Notificaciones y Alertas Automáticas</h2>
                    <p class="t-modal-desc">Despacho de alertas en tiempo real ante caídas de servicios, saturación de hardware y fallos de healthcheck.</p>
                </div>
            </div>
            <button type="button" class="t-close-btn" id="btnCloseAlertsModal" aria-label="Cerrar">&times;</button>
        </div>

        <form id="formAlertsSettings">
            <div class="form-body" style="padding:16px 20px; max-height:480px; overflow-y:auto;">
                <div class="form-field checkbox-field" style="margin-bottom:16px; background:var(--surface-input); padding:10px 14px; border-radius:var(--radius-sm); border:1px solid var(--border-subtle); display:flex; align-items:center; justify-content:space-between;">
                    <div>
                        <strong style="font-size:0.85rem; color:#fff;">Activar Motor de Alertas</strong>
                        <p style="font-size:0.75rem; color:var(--text-muted); margin:2px 0 0;">Emitir alertas salientes cuando ocurran eventos críticos en el cluster</p>
                    </div>
                    <label class="t-checkbox-label" style="margin:0;">
                        <input type="checkbox" id="alertsEnabled" style="width:18px; height:18px;">
                    </label>
                </div>

                <div class="form-row-grid">
                    <div class="form-field full-span">
                        <label for="alertDiscordURL">🎮 Discord Webhook URL</label>
                        <input type="url" id="alertDiscordURL" class="t-input" placeholder="https://discord.com/api/webhooks/...">
                        <span style="font-size:0.72rem; color:var(--text-muted); margin-top:3px;">Envía rich embeds con colores de severidad (Info, Advertencia, Crítico).</span>
                    </div>

                    <div class="form-field">
                        <label for="alertTelegramToken">✈️ Telegram Bot Token</label>
                        <input type="text" id="alertTelegramToken" class="t-input" placeholder="123456:ABC-DEF1234...">
                    </div>

                    <div class="form-field">
                        <label for="alertTelegramChat">💬 Telegram Chat ID / Channel</label>
                        <input type="text" id="alertTelegramChat" class="t-input" placeholder="ej: -1001234567890 o @micanal">
                    </div>

                    <div class="form-field full-span">
                        <label for="alertSlackURL">💼 Slack Incoming Webhook URL</label>
                        <input type="url" id="alertSlackURL" class="t-input" placeholder="https://hooks.slack.com/services/...">
                    </div>

                    <div class="form-field full-span">
                        <label for="alertGenericURL">🌐 Webhook HTTP Genérico (POST JSON)</label>
                        <input type="url" id="alertGenericURL" class="t-input" placeholder="https://mi-api.com/webhooks/tarhiata">
                    </div>
                </div>

                <div class="fast-notice" style="margin-top:14px;">
                    <span class="notice-icon">💡</span>
                    <span>Puedes probar la conectividad de los canales configurados en cualquier momento con el botón "Probar Alerta".</span>
                </div>
            </div>

            <div class="t-modal-footer" style="display:flex; justify-content:space-between; align-items:center;">
                <button type="button" class="mini-btn mini-btn-accent" id="btnTestAlerts">
                    ⚡ Probar Alerta
                </button>
                <div style="display:flex; gap:8px;">
                    <button type="button" class="t-btn t-btn-secondary" id="btnCancelAlerts">Cancelar</button>
                    <button type="submit" class="t-btn t-btn-primary" id="btnSaveAlerts">
                        <span>Guardar Ajustes</span>
                    </button>
                </div>
            </div>
        </form>
    </div>
</div>`;
    document.body.appendChild(div.firstElementChild);
    bindAlertsEvents();
}

function bindAlertsEvents() {
    const btnCloseAlertsModal = document.getElementById('btnCloseAlertsModal');
    const btnCancelAlerts = document.getElementById('btnCancelAlerts');
    const btnTestAlerts = document.getElementById('btnTestAlerts');
    const formAlertsSettings = document.getElementById('formAlertsSettings');

    if (btnCloseAlertsModal) btnCloseAlertsModal.addEventListener('click', closeAlertsModal);
    if (btnCancelAlerts) btnCancelAlerts.addEventListener('click', closeAlertsModal);

    if (btnTestAlerts) {
        btnTestAlerts.addEventListener('click', async () => {
            const payload = collectFormData();
            btnTestAlerts.disabled = true;
            btnTestAlerts.textContent = '⏳ Probando...';
            try {
                const res = await apiFetch('/api/alerts/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ settings: payload })
                });
                if (res.ok) {
                    showToast('¡Alerta de prueba enviada exitosamente!', 'success');
                } else {
                    showToast(`Aviso en prueba de alerta: ${res.error || 'Revise URLs configuradas'}`, 'error');
                }
            } catch (err) {
                showToast(`Error al probar alertas: ${err.message}`, 'error');
            } finally {
                btnTestAlerts.disabled = false;
                btnTestAlerts.textContent = '⚡ Probar Alerta';
            }
        });
    }

    if (formAlertsSettings) {
        formAlertsSettings.addEventListener('submit', async (e) => {
            e.preventDefault();
            const btnSaveAlerts = document.getElementById('btnSaveAlerts');
            const payload = collectFormData();

            if (btnSaveAlerts) btnSaveAlerts.disabled = true;
            try {
                const res = await apiFetch('/api/settings/alerts', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('Configuración de alertas guardada con éxito.', 'success');
                    closeAlertsModal();
                } else {
                    showToast(`Error guardando alertas: ${res.error}`, 'error');
                }
            } catch (err) {
                showToast(`Fallo de conexión: ${err.message}`, 'error');
            } finally {
                if (btnSaveAlerts) btnSaveAlerts.disabled = false;
            }
        });
    }
}

function collectFormData() {
    return {
        enabled: document.getElementById('alertsEnabled') ? document.getElementById('alertsEnabled').checked : false,
        discordUrl: document.getElementById('alertDiscordURL') ? document.getElementById('alertDiscordURL').value.trim() : '',
        telegramToken: document.getElementById('alertTelegramToken') ? document.getElementById('alertTelegramToken').value.trim() : '',
        telegramChat: document.getElementById('alertTelegramChat') ? document.getElementById('alertTelegramChat').value.trim() : '',
        slackUrl: document.getElementById('alertSlackURL') ? document.getElementById('alertSlackURL').value.trim() : '',
        genericUrl: document.getElementById('alertGenericURL') ? document.getElementById('alertGenericURL').value.trim() : ''
    };
}

export async function openAlertsModal() {
    ensureAlertsModalMounted();
    const modal = document.getElementById('alertsModal');
    if (!modal) return;
    modal.style.display = 'flex';
    await loadAlertsSettings();
}

export function closeAlertsModal() {
    const modal = document.getElementById('alertsModal');
    if (modal) modal.style.display = 'none';
}

export async function loadAlertsSettings() {
    try {
        const res = await apiFetch('/api/settings/alerts');
        if (res.ok && res.data) {
            const d = res.data;
            if (document.getElementById('alertsEnabled')) document.getElementById('alertsEnabled').checked = !!d.enabled;
            if (document.getElementById('alertDiscordURL')) document.getElementById('alertDiscordURL').value = d.discordUrl || '';
            if (document.getElementById('alertTelegramToken')) document.getElementById('alertTelegramToken').value = d.telegramToken || '';
            if (document.getElementById('alertTelegramChat')) document.getElementById('alertTelegramChat').value = d.telegramChat || '';
            if (document.getElementById('alertSlackURL')) document.getElementById('alertSlackURL').value = d.slackUrl || '';
            if (document.getElementById('alertGenericURL')) document.getElementById('alertGenericURL').value = d.genericUrl || '';
        }
    } catch (err) {
        console.debug('Error cargando configuración de alertas:', err);
    }
}

export function setupAlertsListeners() {
    ensureAlertsModalMounted();
}
