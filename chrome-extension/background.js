// Background script for side panel
chrome.action.onClicked.addListener((tab) => {
    // Open the side panel for the current tab
    chrome.sidePanel.open({ tabId: tab.id });
});

// Handle messages from content script
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'saveBookmark') {
        handleSaveBookmark(message.data)
            .then(result => sendResponse(result))
            .catch(error => sendResponse({ success: false, error: error.message }));
        return true; // Keep message channel open for async response
    }
});

// Save bookmark to backend API
async function handleSaveBookmark(pageInfo) {
    try {
        // Get API configuration
        const storage = await chrome.storage.local.get(['api_base', 'api_token']);
        const API_BASE = (storage.api_base || 'http://localhost:8080').replace(/\/$/, "");
        const API_TOKEN = storage.api_token || 'your-secret-token-here';

        // Ensure data is within limits (double-check)
        const title = (pageInfo.title || '').substring(0, 200);
        const description = (pageInfo.description || '').substring(0, 1000);

        const bookmarkData = {
            url: pageInfo.url,
            title: title,
            description: description
        };

        console.log('📤 发送书签:', {
            url: bookmarkData.url,
            titleLength: title.length,
            descLength: description.length
        });

        // Send request to backend
        const response = await fetch(`${API_BASE}/api/bookmarks/`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(bookmarkData)
        });

        if (!response.ok) {
            const errorText = await response.text();
            console.error('❌ API 错误:', response.status, errorText);
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const result = await response.json();
        console.log('✅ 保存成功:', result.id);

        return { success: true, data: result };
    } catch (error) {
        console.error('❌ 保存失败:', error);
        return { success: false, error: error.message };
    }
}
