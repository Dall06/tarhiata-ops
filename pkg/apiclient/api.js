/**
 * Tarhiata Cloud Studio — API & Streaming Client
 * Leaf module: zero internal dependencies.
 */

export function setupSecurityInterceptor() {
    try {
        const urlParams = new URLSearchParams(window.location.search);
        const initialKey = urlParams.get('key') || urlParams.get('api_key');
        if (initialKey) {
            sessionStorage.setItem('tarhiata_api_key', initialKey);
        }
        const originalFetch = window.fetch;
        window.fetch = function(input, init) {
            const key = sessionStorage.getItem('tarhiata_api_key') || localStorage.getItem('tarhiata_api_key');
            if (key) {
                init = init || {};
                if (!init.headers) {
                    init.headers = {};
                }
                if (init.headers instanceof Headers) {
                    if (!init.headers.has('X-API-Key')) init.headers.set('X-API-Key', key);
                } else if (Array.isArray(init.headers)) {
                    init.headers.push(['X-API-Key', key]);
                } else {
                    if (!init.headers['X-API-Key']) init.headers['X-API-Key'] = key;
                }
            }
            return originalFetch.call(this, input, init);
        };
    } catch (secErr) {
        console.debug('Error configurando interceptor de seguridad:', secErr);
    }
}

export async function apiFetch(url, options = {}, onError = null) {
    try {
        const res = await fetch(url, options);
        if (!res.ok) {
            const errText = await res.text();
            let msg = errText;
            try {
                const j = JSON.parse(errText);
                if (j.error || j.message) msg = j.error || j.message;
            } catch (_) {}
            if (onError) onError(msg);
            return { ok: false, status: res.status, error: msg, data: null };
        }
        let data = null;
        const contentType = res.headers.get('content-type') || '';
        if (contentType.includes('application/json')) {
            data = await res.json();
        } else {
            data = await res.text();
        }
        return { ok: true, status: res.status, error: null, data };
    } catch (err) {
        if (onError) onError(err.message);
        return { ok: false, status: 0, error: err.message, data: null };
    }
}

export async function consumeNDJSONStream(res, onStep, onError) {
    const reader = res.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop();

        for (const line of lines) {
            if (!line.trim()) continue;
            try {
                const event = JSON.parse(line);
                if (event.type === 'error') {
                    if (onError) onError(event.data);
                } else if (onStep) {
                    onStep(event);
                }
            } catch (e) {
                console.debug('Error parseando NDJSON stream line:', line, e);
            }
        }
    }
    if (buffer.trim()) {
        try {
            const event = JSON.parse(buffer);
            if (event.type === 'error') {
                if (onError) onError(event.data);
            } else if (onStep) {
                onStep(event);
            }
        } catch (_) {}
    }
}
