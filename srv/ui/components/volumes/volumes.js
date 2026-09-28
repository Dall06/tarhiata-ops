import { state } from '/pkg/store/state.js';
import { showToast } from '/pkg/toast/toast.js';
import { apiFetch } from '/pkg/apiclient/api.js';
import { escapeHtml, formatFileSize, getFileIcon } from '/pkg/jsutil/utils.js';

let currentVolumePath = '/opt/data';
let currentVolumeFiles = [];
let currentEditingFilePath = '';

export function openVolumeModal(initialPath) {
    const volumeModal = document.getElementById('volumeModal');
    const volumeSearchInput = document.getElementById('volumeSearchInput');
    const volumeFilesView = document.getElementById('volumeFilesView');
    const volumeEditorView = document.getElementById('volumeEditorView');

    if (!volumeModal) return;
    currentVolumePath = initialPath || '/opt/data';
    if (volumeSearchInput) volumeSearchInput.value = '';
    if (volumeFilesView) volumeFilesView.style.display = 'block';
    if (volumeEditorView) volumeEditorView.style.display = 'none';
    volumeModal.style.display = 'flex';
    fetchAndRenderVolumeFiles(currentVolumePath);
}

export function closeVolumeModal() {
    const volumeModal = document.getElementById('volumeModal');
    if (volumeModal) volumeModal.style.display = 'none';
    currentEditingFilePath = '';
}

export function renderVolumeBreadcrumbs(targetPath) {
    const volumeBreadcrumbs = document.getElementById('volumeBreadcrumbs');
    if (!volumeBreadcrumbs) return;
    volumeBreadcrumbs.innerHTML = '';

    const base = '/opt/data';
    let rel = targetPath.startsWith(base) ? targetPath.slice(base.length) : targetPath;
    if (rel.startsWith('/')) rel = rel.slice(1);
    const parts = rel ? rel.split('/').filter(Boolean) : [];

    const rootCrumb = document.createElement('span');
    rootCrumb.className = `breadcrumb-crumb ${parts.length === 0 ? 'active' : ''}`;
    rootCrumb.textContent = '🏠 /opt/data';
    rootCrumb.addEventListener('click', () => {
        if (currentVolumePath !== base) {
            currentVolumePath = base;
            fetchAndRenderVolumeFiles(base);
        }
    });
    volumeBreadcrumbs.appendChild(rootCrumb);

    let accumulated = base;
    parts.forEach((p, idx) => {
        const sep = document.createElement('span');
        sep.className = 'breadcrumb-sep';
        sep.textContent = '/';
        volumeBreadcrumbs.appendChild(sep);

        accumulated += '/' + p;
        const crumb = document.createElement('span');
        const isLast = idx === parts.length - 1;
        crumb.className = `breadcrumb-crumb ${isLast ? 'active' : ''}`;
        crumb.textContent = p;
        const target = accumulated;
        if (!isLast) {
            crumb.addEventListener('click', () => {
                currentVolumePath = target;
                fetchAndRenderVolumeFiles(target);
            });
        }
        volumeBreadcrumbs.appendChild(crumb);
    });
}

export async function fetchAndRenderVolumeFiles(targetPath) {
    const volumeTableBody = document.getElementById('volumeTableBody');
    const volumePathDisplay = document.getElementById('volumePathDisplay');
    const volumeModalStatus = document.getElementById('volumeModalStatus');

    if (!volumeTableBody) return;
    renderVolumeBreadcrumbs(targetPath);
    if (volumePathDisplay) volumePathDisplay.textContent = targetPath;
    if (volumeModalStatus) volumeModalStatus.textContent = 'Explorando directorio remoto...';

    volumeTableBody.innerHTML = '<tr><td colspan="4" class="t-td-empty">Explorando directorio remoto...</td></tr>';

    try {
        const srv = state.selectedServerName;
        const srvParam = srv ? `&server=${encodeURIComponent(srv)}` : '';
        const res = await apiFetch(`/api/volumes/files?path=${encodeURIComponent(targetPath)}${srvParam}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'Error al obtener archivos');
        }
        const items = await res.json();
        currentVolumeFiles = Array.isArray(items) ? items : [];

        currentVolumeFiles.sort((a, b) => {
            if (a.isDir && !b.isDir) return -1;
            if (!a.isDir && b.isDir) return 1;
            return (a.name || '').localeCompare(b.name || '');
        });

        renderVolumeTableRows(currentVolumeFiles);
        if (volumeModalStatus) {
            const dirs = currentVolumeFiles.filter(f => f.isDir).length;
            const files = currentVolumeFiles.filter(f => !f.isDir).length;
            volumeModalStatus.textContent = `${dirs} carpetas, ${files} archivos en ${targetPath}`;
        }
    } catch (err) {
        volumeTableBody.innerHTML = `<tr><td colspan="4" class="t-td-empty" style="color:var(--status-offline);">Error: ${escapeHtml(err.message)}</td></tr>`;
        if (volumeModalStatus) volumeModalStatus.textContent = `Error: ${err.message}`;
    }
}

export function renderVolumeTableRows(items) {
    const volumeTableBody = document.getElementById('volumeTableBody');
    if (!volumeTableBody) return;
    volumeTableBody.innerHTML = '';

    if (!items || items.length === 0) {
        volumeTableBody.innerHTML = '<tr><td colspan="4" class="t-td-empty">Directorio vacío. Puedes crear carpetas o subir archivos.</td></tr>';
        return;
    }

    items.forEach(item => {
        const tr = document.createElement('tr');
        tr.className = 'volume-item-row';

        const icon = getFileIcon(item.name, item.isDir);
        const sizeStr = item.isDir ? '—' : formatFileSize(item.size);
        const nameClass = item.isDir ? 'volume-item-folder' : 'volume-item-file';

        tr.innerHTML = `
            <td>
                <div class="volume-item-name">
                    <span>${icon}</span>
                    <span class="${nameClass}" data-path="${escapeHtml(item.path)}" data-dir="${item.isDir}">${escapeHtml(item.name)}</span>
                </div>
            </td>
            <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-secondary);">${escapeHtml(sizeStr)}</td>
            <td style="font-family:var(--font-mono); font-size:0.75rem; color:var(--text-muted);">${escapeHtml(item.modTime || '—')}</td>
            <td style="text-align:right;">
                <div style="display:flex; justify-content:flex-end; gap:4px;">
                    ${!item.isDir ? `
                        <button type="button" class="mini-btn btn-vol-read" data-path="${escapeHtml(item.path)}" title="Ver o editar texto">
                            👁️
                        </button>
                        <button type="button" class="mini-btn btn-vol-dl" data-path="${escapeHtml(item.path)}" title="Descargar archivo">
                            ⬇️
                        </button>
                    ` : ''}
                    <button type="button" class="mini-btn btn-vol-del" data-path="${escapeHtml(item.path)}" data-name="${escapeHtml(item.name)}" style="color:var(--status-offline);" title="Eliminar">
                        🗑️
                    </button>
                </div>
            </td>
        `;

        if (item.isDir) {
            const folderLink = tr.querySelector('.volume-item-folder');
            if (folderLink) {
                folderLink.addEventListener('click', () => {
                    currentVolumePath = item.path;
                    fetchAndRenderVolumeFiles(item.path);
                });
            }
        }

        volumeTableBody.appendChild(tr);
    });

    volumeTableBody.querySelectorAll('.btn-vol-read').forEach(btn => {
        btn.addEventListener('click', () => {
            const path = btn.getAttribute('data-path');
            if (path) openVolumeFileEditor(path);
        });
    });

    volumeTableBody.querySelectorAll('.btn-vol-dl').forEach(btn => {
        btn.addEventListener('click', () => {
            const path = btn.getAttribute('data-path');
            if (path) {
                const srv = state.selectedServerName;
                const srvParam = srv ? `&server=${encodeURIComponent(srv)}` : '';
                window.location.href = `/api/volumes/download?path=${encodeURIComponent(path)}${srvParam}`;
            }
        });
    });

    volumeTableBody.querySelectorAll('.btn-vol-del').forEach(btn => {
        btn.addEventListener('click', async () => {
            const path = btn.getAttribute('data-path');
            const name = btn.getAttribute('data-name');
            if (!path) return;
            if (!confirm(`¿Estás seguro de eliminar permanentemente "${name}" del VPS?`)) return;

            try {
                btn.disabled = true;
                const srv = state.selectedServerName;
                const srvParam = srv ? `&server=${encodeURIComponent(srv)}` : '';
                const res = await apiFetch(`/api/volumes/delete?path=${encodeURIComponent(path)}${srvParam}`, {
                    method: 'DELETE'
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al eliminar');
                }
                showToast(`"${name}" eliminado correctamente`, 'success');
                fetchAndRenderVolumeFiles(currentVolumePath);
            } catch (err) {
                showToast(`Error al eliminar: ${err.message}`, 'error');
                btn.disabled = false;
            }
        });
    });
}

export async function openVolumeFileEditor(filePath) {
    currentEditingFilePath = filePath;
    const volumeFilesView = document.getElementById('volumeFilesView');
    const volumeEditorView = document.getElementById('volumeEditorView');
    const volumeEditorFilePath = document.getElementById('volumeEditorFilePath');
    const volumeEditorTextarea = document.getElementById('volumeEditorTextarea');

    if (volumeFilesView) volumeFilesView.style.display = 'none';
    if (volumeEditorView) volumeEditorView.style.display = 'block';
    if (volumeEditorFilePath) volumeEditorFilePath.textContent = filePath;
    if (volumeEditorTextarea) volumeEditorTextarea.value = 'Cargando contenido del archivo...';

    try {
        const srv = state.selectedServerName;
        const srvParam = srv ? `&server=${encodeURIComponent(srv)}` : '';
        const res = await apiFetch(`/api/volumes/read?path=${encodeURIComponent(filePath)}${srvParam}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'Error al leer archivo');
        }
        const data = await res.json();
        if (volumeEditorTextarea) volumeEditorTextarea.value = data.content || '';
    } catch (err) {
        if (volumeEditorTextarea) volumeEditorTextarea.value = `Error al leer archivo: ${err.message}`;
        showToast(`Error: ${err.message}`, 'error');
    }
}

export function setupVolumeListeners() {
    const btnVolumeBackToList = document.getElementById('btnVolumeBackToList');
    const btnVolumeEditorDownload = document.getElementById('btnVolumeEditorDownload');
    const btnVolumeEditorSave = document.getElementById('btnVolumeEditorSave');
    const btnVolumeNewFolder = document.getElementById('btnVolumeNewFolder');
    const volumeFileInput = document.getElementById('volumeFileInput');
    const btnVolumeRefresh = document.getElementById('btnVolumeRefresh');
    const volumeSearchInput = document.getElementById('volumeSearchInput');
    const btnDeskVolumes = document.getElementById('btnDeskVolumes');
    const btnCloseVolumeModal = document.getElementById('btnCloseVolumeModal');
    const btnCloseVolumeModalBottom = document.getElementById('btnCloseVolumeModalBottom');
    const volumeModal = document.getElementById('volumeModal');
    const volumeFilesView = document.getElementById('volumeFilesView');
    const volumeEditorView = document.getElementById('volumeEditorView');
    const volumeEditorTextarea = document.getElementById('volumeEditorTextarea');

    if (btnVolumeBackToList) {
        btnVolumeBackToList.addEventListener('click', () => {
            if (volumeEditorView) volumeEditorView.style.display = 'none';
            if (volumeFilesView) volumeFilesView.style.display = 'block';
            currentEditingFilePath = '';
        });
    }

    if (btnVolumeEditorDownload) {
        btnVolumeEditorDownload.addEventListener('click', () => {
            if (currentEditingFilePath) {
                const srv = state.selectedServerName;
                const srvParam = srv ? `&server=${encodeURIComponent(srv)}` : '';
                window.location.href = `/api/volumes/download?path=${encodeURIComponent(currentEditingFilePath)}${srvParam}`;
            }
        });
    }

    if (btnVolumeEditorSave) {
        btnVolumeEditorSave.addEventListener('click', async () => {
            if (!currentEditingFilePath || !volumeEditorTextarea) return;
            const originalText = btnVolumeEditorSave.innerHTML;
            btnVolumeEditorSave.disabled = true;
            btnVolumeEditorSave.innerHTML = '<span>Guardando...</span>';

            try {
                const srv = state.selectedServerName;
                const srvParam = srv ? `?server=${encodeURIComponent(srv)}` : '';
                const res = await apiFetch(`/api/volumes/write${srvParam}`, {
                    method: 'POST',
                    body: JSON.stringify({
                        path: currentEditingFilePath,
                        content: volumeEditorTextarea.value
                    })
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al guardar archivo');
                }
                showToast('Archivo guardado correctamente en el VPS', 'success');
            } catch (err) {
                showToast(`Error al guardar: ${err.message}`, 'error');
            } finally {
                btnVolumeEditorSave.disabled = false;
                btnVolumeEditorSave.innerHTML = originalText;
            }
        });
    }

    if (btnVolumeNewFolder) {
        btnVolumeNewFolder.addEventListener('click', async () => {
            const folderName = prompt('Nombre de la nueva carpeta:');
            if (!folderName || !folderName.trim()) return;

            const cleanName = folderName.trim().replace(/[/\\]/g, '');
            const newFolderPath = `${currentVolumePath}/${cleanName}`;

            try {
                const srv = state.selectedServerName;
                const srvParam = srv ? `?server=${encodeURIComponent(srv)}` : '';
                const res = await apiFetch(`/api/volumes/mkdir${srvParam}`, {
                    method: 'POST',
                    body: JSON.stringify({ path: newFolderPath })
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al crear carpeta');
                }
                showToast(`Carpeta "${cleanName}" creada`, 'success');
                fetchAndRenderVolumeFiles(currentVolumePath);
            } catch (err) {
                showToast(`Error: ${err.message}`, 'error');
            }
        });
    }

    if (volumeFileInput) {
        volumeFileInput.addEventListener('change', async (e) => {
            const file = e.target.files && e.target.files[0];
            if (!file) return;

            const formData = new FormData();
            formData.append('file', file);
            formData.append('dir', currentVolumePath);

            showToast(`Subiendo "${file.name}"...`, 'info');
            try {
                const srv = state.selectedServerName;
                const srvParam = srv ? `?server=${encodeURIComponent(srv)}` : '';
                const res = await fetch(`/api/volumes/upload${srvParam}`, {
                    method: 'POST',
                    body: formData
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(errText || 'Error al subir archivo');
                }
                showToast(`"${file.name}" subido con éxito`, 'success');
                fetchAndRenderVolumeFiles(currentVolumePath);
            } catch (err) {
                showToast(`Error al subir: ${err.message}`, 'error');
            } finally {
                volumeFileInput.value = '';
            }
        });
    }

    if (btnVolumeRefresh) {
        btnVolumeRefresh.addEventListener('click', () => {
            fetchAndRenderVolumeFiles(currentVolumePath);
        });
    }

    if (volumeSearchInput) {
        volumeSearchInput.addEventListener('input', () => {
            const query = volumeSearchInput.value.toLowerCase().trim();
            if (!query) {
                renderVolumeTableRows(currentVolumeFiles);
                return;
            }
            const filtered = currentVolumeFiles.filter(item => (item.name || '').toLowerCase().includes(query));
            renderVolumeTableRows(filtered);
        });
    }

    if (btnDeskVolumes) {
        btnDeskVolumes.addEventListener('click', () => {
            openVolumeModal('/opt/data');
        });
    }

    if (btnCloseVolumeModal) btnCloseVolumeModal.addEventListener('click', closeVolumeModal);
    if (btnCloseVolumeModalBottom) btnCloseVolumeModalBottom.addEventListener('click', closeVolumeModal);
    if (volumeModal) {
        volumeModal.addEventListener('click', (e) => {
            if (e.target === volumeModal) closeVolumeModal();
        });
    }
}
