/**
 * Tarhiata Cloud Studio — Desktop Notifications Module (pkg/notify)
 * Integrates Web Notifications API with graceful toast fallback.
 */

import { showToast } from '/pkg/toast/toast.js';

let notificationPermissionGranted = false;

export async function requestNotificationPermission() {
    if (!('Notification' in window)) {
        return false;
    }
    if (Notification.permission === 'granted') {
        notificationPermissionGranted = true;
        return true;
    }
    if (Notification.permission !== 'denied') {
        const perm = await Notification.requestPermission();
        notificationPermissionGranted = (perm === 'granted');
        if (notificationPermissionGranted) {
            showToast('Notificaciones de escritorio habilitadas.', 'success');
        }
        return notificationPermissionGranted;
    }
    return false;
}

export function sendDesktopNotification(title, body = '', icon = null) {
    if (!('Notification' in window) || Notification.permission !== 'granted') {
        return;
    }

    try {
        const notif = new Notification(title, {
            body: body,
            icon: icon || 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" rx="12" fill="%236366f1"/><polygon points="50,15 20,55 50,55 45,85 80,45 50,45" fill="white"/></svg>',
            silent: false
        });

        notif.onclick = () => {
            window.focus();
            notif.close();
        };

        setTimeout(() => notif.close(), 6000);
    } catch (err) {
        console.debug('Error enviando notificación nativa:', err);
    }
}
