/**
 * Tarhiata Cloud Studio — Shared Helper Utilities
 * Leaf module: zero internal dependencies.
 */

export function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

export function getDefaultPort(engine) {
    const eng = (engine || '').toLowerCase();
    if (eng === 'redis') return 6379;
    if (eng === 'mysql' || eng === 'mariadb') return 3306;
    if (eng === 'mongo' || eng === 'mongodb') return 27017;
    if (eng === 'minio') return 9000;
    return 5432;
}

export function debounce(fn, waitMs = 150) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn.apply(this, args), waitMs);
    };
}

export function getGaugeColor(pct) {
    if (pct >= 85) return 'var(--accent-danger, #ef4444)';
    if (pct >= 60) return 'var(--accent-warning, #f59e0b)';
    return 'var(--accent-success, #10b981)';
}

export async function copyToClipboard(text) {
    if (!text) return false;
    if (navigator.clipboard && window.isSecureContext) {
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch (clipErr) {
            console.debug('clipboard.writeText no soportado o denegado:', clipErr);
        }
    }
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    let ok = false;
    try {
        ok = document.execCommand('copy');
    } catch (execErr) {
        console.debug('execCommand copy falló:', execErr);
    }
    document.body.removeChild(textArea);
    return ok;
}

export function formatDockerVersion(raw) {
    if (!raw) return '—';
    const match = String(raw).match(/(\d+\.\d+\.\d+)/);
    if (match) {
        return `Docker v${match[1]}`;
    }
    return String(raw).replace('Docker version ', 'v').replace(/, build .*/, '');
}

export function formatFileSize(bytes) {
    if (bytes === 0 || isNaN(bytes)) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function getFileIcon(name, isDir) {
    if (isDir) return '📁';
    const ext = (name || '').split('.').pop().toLowerCase();
    if (['json', 'yml', 'yaml', 'toml', 'env', 'conf', 'ini'].includes(ext)) return '⚙️';
    if (['js', 'ts', 'go', 'py', 'sh', 'bash', 'rb', 'php'].includes(ext)) return '📜';
    if (['log', 'txt', 'md'].includes(ext)) return '📄';
    if (['db', 'sqlite', 'sql', 'dump'].includes(ext)) return '🗄️';
    if (['tar', 'gz', 'zip', 'tgz'].includes(ext)) return '📦';
    return '📄';
}

export function isSensitiveKey(key) {
    const k = (key || '').toUpperCase();
    return k.includes('PASS') || k.includes('SECRET') || k.includes('TOKEN') || k.includes('KEY') || k.includes('AUTH') || k.includes('PRIVATE');
}
