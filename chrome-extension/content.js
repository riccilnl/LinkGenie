// Content script for floating bookmark button
(function () {
    'use strict';

    // Create floating button container
    const floatingContainer = document.createElement('div');
    floatingContainer.id = 'bookmark-floating-container';
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

    // Add button click handler
    const addButton = floatingContainer.querySelector('#bookmark-add-btn');
    addButton.addEventListener('click', handleAddBookmark);

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
})();
