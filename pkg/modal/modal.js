/**
 * Tarhiata Cloud Studio — Modal Lifecycle & Accessibility
 * Leaf module: zero internal dependencies.
 */

export function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.add('active');
    modal.style.display = 'flex';
    document.body.classList.add('modal-open');
}

export function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.remove('active');
    modal.style.display = 'none';
    if (!document.querySelector('.t-modal-overlay.active')) {
        document.body.classList.remove('modal-open');
    }
}

export function setupModalDismissals() {
    document.querySelectorAll('.t-modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                overlay.classList.remove('active');
                overlay.style.display = 'none';
                if (!document.querySelector('.t-modal-overlay.active')) {
                    document.body.classList.remove('modal-open');
                }
            }
        });
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            document.querySelectorAll('.t-modal-overlay.active').forEach(modal => {
                modal.classList.remove('active');
                modal.style.display = 'none';
            });
            document.body.classList.remove('modal-open');
        }
    });
}
