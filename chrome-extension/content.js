// Content script for floating bookmark button
(function () {
    'use strict';

    // Create floating button container
    const floatingContainer = document.createElement('div');
    floatingContainer.id = 'bookmark-floating-container';
    floatingContainer.classList.add('bookmark-floating-container');
    floatingContainer.innerHTML = `
        <button id="bookmark-add-btn" class="bookmark-floating-btn" title="保存书签">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z" fill="currentColor"/>
            </svg>
        </button>
    `;

    // Create toast notification container
    const toastContainer = document.createElement('div');
    toastContainer.id = 'bookmark-toast-container';

    // Wait for DOM to be ready
    if (document.body) {
        document.body.appendChild(floatingContainer);
        document.body.appendChild(toastContainer);
    } else {
        document.addEventListener('DOMContentLoaded', () => {
            document.body.appendChild(floatingContainer);
            document.body.appendChild(toastContainer);
        });
    }

    // Floating button drag behavior
    const dragState = {
        active: false,
        didDrag: false,
        startX: 0,
        startY: 0,
        startLeft: 0,
        startTop: 0,
        dock: 'right'
    };

    // Add button click handler
    const addButton = floatingContainer.querySelector('#bookmark-add-btn');
    addButton.addEventListener('click', (event) => {
        if (dragState.didDrag) {
            dragState.didDrag = false;
            event.preventDefault();
            return;
        }
        handleAddBookmark();
    });

    floatingContainer.addEventListener('pointerdown', (event) => {
        if (event.button !== 0) {
            return;
        }

        dragState.active = true;
        dragState.didDrag = false;
        dragState.startX = event.clientX;
        dragState.startY = event.clientY;

        const rect = floatingContainer.getBoundingClientRect();
        dragState.startLeft = rect.left;
        dragState.startTop = rect.top;

        floatingContainer.setPointerCapture(event.pointerId);
        floatingContainer.classList.add('dragging');
        floatingContainer.classList.remove('collapsed');
    });

    floatingContainer.addEventListener('pointermove', (event) => {
        if (!dragState.active) {
            return;
        }

        const dx = event.clientX - dragState.startX;
        const dy = event.clientY - dragState.startY;

        if (!dragState.didDrag && (Math.abs(dx) > 4 || Math.abs(dy) > 4)) {
            dragState.didDrag = true;
        }

        const rect = floatingContainer.getBoundingClientRect();
        const width = rect.width;
        const height = rect.height;

        const maxLeft = Math.max(0, window.innerWidth - width);
        const maxTop = Math.max(0, window.innerHeight - height);
        const nextLeft = clamp(dragState.startLeft + dx, 0, maxLeft);
        const nextTop = clamp(dragState.startTop + dy, 0, maxTop);

        floatingContainer.style.left = `${nextLeft}px`;
        floatingContainer.style.top = `${nextTop}px`;
        floatingContainer.style.right = 'auto';
    });

    floatingContainer.addEventListener('pointerup', (event) => {
        if (!dragState.active) {
            return;
        }

        dragState.active = false;
        floatingContainer.releasePointerCapture(event.pointerId);
        floatingContainer.classList.remove('dragging');

        snapToEdge();
    });

    floatingContainer.addEventListener('pointercancel', () => {
        if (!dragState.active) {
            return;
        }
        dragState.active = false;
        floatingContainer.classList.remove('dragging');
        snapToEdge();
    });

    floatingContainer.addEventListener('mouseenter', () => {
        floatingContainer.classList.remove('collapsed');
    });

    floatingContainer.addEventListener('mouseleave', () => {
        if (!dragState.active) {
            floatingContainer.classList.add('collapsed');
        }
    });

    window.addEventListener('resize', () => {
        snapToEdge(true);
    });

    restorePosition();

    // Handle add bookmark
    async function handleAddBookmark() {
        // Disable button to prevent double clicks
        addButton.disabled = true;
        addButton.classList.add('loading');

        try {
            // Get page info with safe truncation
            const rawTitle = document.title || 'Untitled';
            const rawDesc = getPageDescription();

            const pageInfo = {
                url: window.location.href,
                title: rawTitle.substring(0, 197) + (rawTitle.length > 200 ? '...' : ''),
                description: rawDesc.substring(0, 997) + (rawDesc.length > 1000 ? '...' : '')
            };

            // Send message to background script
            const response = await chrome.runtime.sendMessage({
                action: 'saveBookmark',
                data: pageInfo
            });

            if (response && response.success) {
                showToast('✓ 书签保存成功', 'success');
            } else {
                showToast('✗ 保存失败: ' + (response?.error || '未知错误'), 'error');
            }
        } catch (error) {
            console.error('保存书签失败:', error);
            showToast('✗ 保存失败: ' + error.message, 'error');
        } finally {
            // Re-enable button
            addButton.disabled = false;
            addButton.classList.remove('loading');
        }
    }

    // Get page description from meta tags
    function getPageDescription() {
        // Try to get description from meta tags
        const metaDescription = document.querySelector('meta[name="description"]');
        if (metaDescription) {
            return metaDescription.getAttribute('content') || '';
        }

        const ogDescription = document.querySelector('meta[property="og:description"]');
        if (ogDescription) {
            return ogDescription.getAttribute('content') || '';
        }

        // Fallback: get first paragraph text
        const firstParagraph = document.querySelector('p');
        if (firstParagraph) {
            const text = firstParagraph.textContent.trim();
            return text.length > 200 ? text.substring(0, 200) + '...' : text;
        }

        return '';
    }

    // Show toast notification
    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `bookmark-toast bookmark-toast-${type}`;
        toast.textContent = message;

        toastContainer.appendChild(toast);

        // Trigger animation
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);

        // Auto remove after 3 seconds
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                toast.remove();
            }, 300);
        }, 3000);
    }

    // Listen for messages from background script
    chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (message.action === 'bookmarkSaved') {
            showToast('✓ 书签已保存', 'success');
        }
        return true;
    });

    function clamp(value, min, max) {
        return Math.min(Math.max(value, min), max);
    }

    async function restorePosition() {
        try {
            const storage = await chrome.storage.local.get(['floating_btn_pos']);
            const saved = storage.floating_btn_pos;
            if (saved && typeof saved.top === 'number' && typeof saved.left === 'number') {
                dragState.dock = saved.dock === 'left' ? 'left' : 'right';
                floatingContainer.style.left = `${saved.left}px`;
                floatingContainer.style.top = `${saved.top}px`;
                floatingContainer.style.right = 'auto';
            } else {
                setDefaultPosition();
            }
        } catch (error) {
            setDefaultPosition();
        }

        snapToEdge(true);
    }

    function setDefaultPosition() {
        const rect = floatingContainer.getBoundingClientRect();
        const left = Math.max(0, window.innerWidth - rect.width - 20);
        const top = Math.max(0, (window.innerHeight - rect.height) / 2);
        floatingContainer.style.left = `${left}px`;
        floatingContainer.style.top = `${top}px`;
        floatingContainer.style.right = 'auto';
        dragState.dock = 'right';
    }

    function snapToEdge(isResize = false) {
        const rect = floatingContainer.getBoundingClientRect();
        const width = rect.width;
        const height = rect.height;
        const maxLeft = Math.max(0, window.innerWidth - width);
        const maxTop = Math.max(0, window.innerHeight - height);

        let nextLeft = rect.left;
        let nextTop = rect.top;

        if (!isResize || dragState.didDrag) {
            dragState.dock = (rect.left + width / 2) < (window.innerWidth / 2) ? 'left' : 'right';
        }

        nextLeft = dragState.dock === 'left' ? 0 : maxLeft;
        nextTop = clamp(nextTop, 0, maxTop);

        floatingContainer.style.left = `${nextLeft}px`;
        floatingContainer.style.top = `${nextTop}px`;
        floatingContainer.style.right = 'auto';

        floatingContainer.classList.toggle('dock-left', dragState.dock === 'left');
        floatingContainer.classList.toggle('dock-right', dragState.dock === 'right');
        floatingContainer.classList.add('collapsed');

        chrome.storage.local.set({
            floating_btn_pos: {
                left: nextLeft,
                top: nextTop,
                dock: dragState.dock
            }
        });
    }
})();
