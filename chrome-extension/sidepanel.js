// API 配置 - 从 chrome.storage 读取，如果没有则使用默认值
let API_BASE = 'http://localhost:8080';
let API_TOKEN = 'your-secret-token-here';

// Load API config from chrome.storage
async function loadApiConfigFromStorage() {
    const storage = await chrome.storage.local.get(['api_base', 'api_token']);
    API_BASE = (storage.api_base || 'http://localhost:8080').replace(/\/$/, "");
    API_TOKEN = storage.api_token || 'your-secret-token-here';
}

let currentTab = 'all';
let allBookmarks = [];
let filteredBookmarks = [];
let allFolders = [];
let currentFolderId = null;

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    await loadApiConfigFromStorage();
    setupEventListeners();
    loadBookmarksFromAPI();
});

// Load bookmarks from API
async function loadBookmarksFromAPI() {
    try {
        const response = await fetch(`${API_BASE}/api/bookmarks/`, {
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        allBookmarks = data.results || [];
        filteredBookmarks = [...allBookmarks];

        renderBookmarks();

        console.log(`✅ 成功加载 ${allBookmarks.length} 个书签`);
    } catch (error) {
        console.error('❌ 加载书签失败:', error);

        // 显示错误提示
        const bookmarkList = document.getElementById('bookmarkList');
        bookmarkList.innerHTML = `
            <div class="empty-state">
                <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" fill="currentColor"/>
                </svg>
                <p>无法连接到服务器</p>
                <small>请检查API配置: ${API_BASE}</small>
                <small style="display: block; margin-top: 8px; color: #ef4444;">${error.message}</small>
            </div>
        `;
    }
}

// Setup event listeners
function setupEventListeners() {
    // Tab switching
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const tab = e.currentTarget.dataset.tab;
            switchTab(tab);
        });
    });

    // Settings button in top bar
    const settingsBtn = document.getElementById('settingsBtn');
    if (settingsBtn) {
        settingsBtn.addEventListener('click', () => {
            showSettingsInContentArea();
            // Update tab states - deactivate all tabs
            document.querySelectorAll('.tab-btn').forEach(btn => {
                btn.classList.remove('active');
            });
        });
    }

    // Search functionality
    const searchInput = document.getElementById('searchInput');
    const searchClear = document.getElementById('searchClear');

    searchInput.addEventListener('input', (e) => {
        const value = e.target.value;

        // Show/hide clear button
        if (value) {
            searchClear.style.display = 'flex';
        } else {
            searchClear.style.display = 'none';
        }

        handleSearch(value);
    });

    // Clear search button
    searchClear.addEventListener('click', () => {
        searchInput.value = '';
        searchClear.style.display = 'none';
        loadBookmarksFromAPI(); // Reset to all bookmarks
    });

    // Folder popup close button
    const folderPopupClose = document.getElementById('folderPopupClose');
    if (folderPopupClose) {
        folderPopupClose.addEventListener('click', closeFolderPopup);
    }

    // Close folder popup when clicking outside
    const folderPopup = document.getElementById('folderPopup');
    if (folderPopup) {
        folderPopup.addEventListener('click', (e) => {
            if (e.target === folderPopup) {
                closeFolderPopup();
            }
        });
    }

    // Load saved theme
    loadTheme();
}

// Toggle theme (called from settings)
async function setTheme(theme) {
    const root = document.documentElement;
    const body = document.body;

    if (theme === 'dark') {
        root.classList.add('dark-theme');
        body.classList.add('dark-theme');
    } else {
        root.classList.remove('dark-theme');
        body.classList.remove('dark-theme');
    }

    // Save preference
    await chrome.storage.local.set({ theme });

    // Update button states in settings if visible
    updateThemeButtonStates();
}

// Update theme button states
function updateThemeButtonStates() {
    const lightBtn = document.getElementById('lightThemeBtn');
    const darkBtn = document.getElementById('darkThemeBtn');

    if (lightBtn && darkBtn) {
        const isDark = document.body.classList.contains('dark-theme');

        if (isDark) {
            lightBtn.classList.remove('active');
            darkBtn.classList.add('active');
        } else {
            lightBtn.classList.add('active');
            darkBtn.classList.remove('active');
        }
    }
}

// Load theme
async function loadTheme() {
    const storage = await chrome.storage.local.get(['theme']);
    if (storage.theme === 'dark') {
        document.documentElement.classList.add('dark-theme');
        document.body.classList.add('dark-theme');
    }
}

// Switch tabs
function switchTab(tab) {
    currentTab = tab;

    // Update active state
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.tab === tab) {
            btn.classList.add('active');
        }
    });

    const bookmarkList = document.getElementById('bookmarkList');

    if (tab === 'folders') {
        // Show folder popup
        showFolderPopup();
    } else {
        // Show bookmarks
        bookmarkList.style.display = 'grid';

        // Filter bookmarks based on tab
        filterBookmarksByTab(tab);
    }
}

// Show settings in content area
function showSettingsInContentArea() {
    const bookmarkList = document.getElementById('bookmarkList');
    const template = document.getElementById('settingsTemplate');

    // Clone template content
    const settingsContent = template.content.cloneNode(true);

    // Clear bookmark list and insert settings
    bookmarkList.innerHTML = '';
    bookmarkList.style.display = 'block';
    bookmarkList.appendChild(settingsContent);

    // Load saved API config
    loadApiConfig();

    // Add event listeners for settings
    setupSettingsEventListeners();

    // Update theme button states
    updateThemeButtonStates();
}

// Close settings and return to bookmarks
function closeSettings() {
    // Switch back to "all" tab
    switchTab('all');
}

// Setup settings event listeners
function setupSettingsEventListeners() {
    // Close button
    const closeBtn = document.getElementById('settingsCloseBtn');
    if (closeBtn) {
        closeBtn.addEventListener('click', closeSettings);
    }

    // Save API config button
    const saveBtn = document.querySelector('.settings-btn-primary');
    if (saveBtn) {
        saveBtn.addEventListener('click', saveApiConfig);
    }

    // Theme toggle buttons
    const lightThemeBtn = document.getElementById('lightThemeBtn');
    const darkThemeBtn = document.getElementById('darkThemeBtn');

    if (lightThemeBtn) {
        lightThemeBtn.addEventListener('click', () => setTheme('light'));
    }

    if (darkThemeBtn) {
        darkThemeBtn.addEventListener('click', () => setTheme('dark'));
    }

    // Import file button
    const importBtn = document.querySelector('.settings-btn-secondary');
    if (importBtn) {
        importBtn.addEventListener('click', () => {
            document.getElementById('importFile').click();
        });
    }

    // Import file input
    const importFile = document.getElementById('importFile');
    if (importFile) {
        importFile.addEventListener('change', handleImportFile);
    }

    // Export button
    const exportBtns = document.querySelectorAll('.settings-btn-secondary');
    if (exportBtns.length > 1) {
        exportBtns[1].addEventListener('click', exportBookmarks);
    }
}

// Load API config into form
function loadApiConfig() {
    const apiBaseInput = document.getElementById('apiBaseInput');
    const apiTokenInput = document.getElementById('apiTokenInput');

    if (apiBaseInput && apiTokenInput) {
        apiBaseInput.value = API_BASE;
        apiTokenInput.value = API_TOKEN;
    }
}

// Save API config
async function saveApiConfig() {
    const base = document.getElementById('apiBaseInput').value.trim();
    const token = document.getElementById('apiTokenInput').value.trim();

    if (!base || !token) {
        alert('请填写完整的 API 配置');
        return;
    }

    // Save to chrome.storage
    await chrome.storage.local.set({ api_base: base, api_token: token });

    // Update global variables
    API_BASE = base;
    API_TOKEN = token;

    // Show success message
    const status = document.getElementById('apiConfigStatus');
    status.style.display = 'inline';
    setTimeout(() => {
        status.style.display = 'none';
    }, 2000);

    // Reload bookmarks with new config
    console.log('🔄 使用新配置重新加载书签...');
    loadBookmarksFromAPI();
}

// Handle import file
function handleImportFile() {
    const fileInput = document.getElementById('importFile');
    const fileName = document.getElementById('importFileName');

    if (fileInput.files.length > 0) {
        fileName.textContent = fileInput.files[0].name;
        // TODO: Implement actual import logic
        alert('导入功能开发中...');
    }
}

// Export bookmarks
function exportBookmarks() {
    // TODO: Implement actual export logic
    alert('导出功能开发中...');
}

// Filter bookmarks by tab
function filterBookmarksByTab(tab) {
    switch (tab) {
        case 'all':
            currentFolderId = null;
            filteredBookmarks = [...allBookmarks];
            break;
        case 'unread':
            currentFolderId = null;
            // Filter bookmarks that are not read (assuming is_read field exists)
            filteredBookmarks = allBookmarks.filter(b => !b.is_read);
            break;
        case 'favorites':
            currentFolderId = null;
            // Filter bookmarks that are favorited (assuming is_favorited field exists)
            filteredBookmarks = allBookmarks.filter(b => b.is_favorited);
            break;
        default:
            filteredBookmarks = [...allBookmarks];
    }

    renderBookmarks();
}

// Handle search
async function handleSearch(query) {
    if (!query.trim()) {
        // Empty search, reload all bookmarks
        loadBookmarksFromAPI();
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/bookmarks/?q=${encodeURIComponent(query)}`, {
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        filteredBookmarks = data.results || [];

        renderBookmarks();

        console.log(`🔍 搜索 "${query}" 找到 ${filteredBookmarks.length} 个结果`);
    } catch (error) {
        console.error('❌ 搜索失败:', error);

        // 搜索失败时降级为前端过滤
        const lowerQuery = query.toLowerCase();
        filteredBookmarks = allBookmarks.filter(bookmark => {
            return (bookmark.title || '').toLowerCase().includes(lowerQuery) ||
                (bookmark.description || '').toLowerCase().includes(lowerQuery) ||
                (bookmark.tag_names || []).some(tag => tag.toLowerCase().includes(lowerQuery));
        });

        renderBookmarks();
        console.log(`⚠️ 使用前端过滤，找到 ${filteredBookmarks.length} 个结果`);
    }
}

// Render bookmarks
function renderBookmarks() {
    const bookmarkList = document.getElementById('bookmarkList');

    if (filteredBookmarks.length === 0) {
        bookmarkList.innerHTML = `
      <div class="empty-state">
        <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z" fill="currentColor"/>
        </svg>
        <p>暂无书签</p>
        <small>${currentTab === 'unread' ? '未读功能即将推出' : currentTab === 'archive' ? '归档功能即将推出' : '开始添加您的第一个书签吧'}</small>
      </div>
    `;
        return;
    }

    bookmarkList.innerHTML = filteredBookmarks.map(bookmark => {
        // Format date
        let timeStr = '未知时间';
        if (bookmark.date_added) {
            const date = new Date(bookmark.date_added);
            const now = new Date();
            const diffMs = now - date;
            const diffMins = Math.floor(diffMs / 60000);
            const diffHours = Math.floor(diffMs / 3600000);
            const diffDays = Math.floor(diffMs / 86400000);

            if (diffMins < 60) {
                timeStr = `${diffMins}分钟前`;
            } else if (diffHours < 24) {
                timeStr = `${diffHours}小时前`;
            } else if (diffDays < 7) {
                timeStr = `${diffDays}天前`;
            } else {
                timeStr = date.toLocaleDateString('zh-CN');
            }
        }

        return `
    <div class="bookmark-card" data-id="${bookmark.id}" data-url="${bookmark.url}">
      <div class="bookmark-title" data-action="open">
        <svg class="bookmark-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M17 3H7c-1.1 0-1.99.9-1.99 2L5 21l7-3 7 3V5c0-1.1-.9-2-2-2z" fill="currentColor"/>
        </svg>
        ${bookmark.title || '无标题'}
      </div>
      <div class="bookmark-description" data-action="expand">${bookmark.description || ''}</div>
      <div class="bookmark-tags">
        ${(bookmark.tag_names || []).map(tag => `<span class="tag">#${tag}</span>`).join('')}
      </div>
      <div class="bookmark-time">
        <svg class="time-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z" fill="currentColor"/>
          <path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z" fill="currentColor"/>
        </svg>
        ${timeStr}
      </div>
    </div>
  `;
    }).join('');

    // Add click handlers
    document.querySelectorAll('.bookmark-card').forEach(card => {
        const url = card.dataset.url;

        // Click title to open bookmark
        const title = card.querySelector('[data-action="open"]');
        title.addEventListener('click', (e) => {
            e.stopPropagation();
            if (url) {
                chrome.tabs.create({ url });
            }
        });

        // Click description to expand/collapse card
        const description = card.querySelector('[data-action="expand"]');
        description.addEventListener('click', (e) => {
            e.stopPropagation();
            card.classList.toggle('expanded');
        });

        // Click tags to search
        const tags = card.querySelectorAll('.tag');
        tags.forEach(tag => {
            tag.addEventListener('click', (e) => {
                e.stopPropagation();
                const tagText = tag.textContent; // 包含 # 号
                const searchInput = document.getElementById('searchInput');
                const searchClear = document.getElementById('searchClear');

                searchInput.value = tagText;

                // Show clear button
                if (tagText) {
                    searchClear.style.display = 'flex';
                }

                handleSearch(tagText);
            });
        });
    });
}

// Folder popup functions
async function showFolderPopup() {
    const folderPopup = document.getElementById('folderPopup');
    const folderList = document.getElementById('folderList');

    // Load folders from API
    await loadFoldersFromAPI();

    // Render folders
    if (allFolders.length === 0) {
        folderList.innerHTML = `
            <div class="empty-state" style="padding: 40px 20px;">
                <p>暂无文件夹</p>
                <small>您还没有创建任何文件夹</small>
            </div>
        `;
    } else {
        folderList.innerHTML = allFolders.map(folder => `
            <div class="folder-item" data-folder-id="${folder.id}">
                <svg class="folder-item-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" fill="currentColor"/>
                </svg>
                <span class="folder-item-name">${folder.name}</span>
                <span class="folder-item-count">${folder.count || 0}</span>
            </div>
        `).join('');

        // Add click handlers
        document.querySelectorAll('.folder-item').forEach(item => {
            item.addEventListener('click', () => {
                const folderId = parseInt(item.dataset.folderId);
                selectFolder(folderId);
            });
        });
    }

    // Show popup
    folderPopup.style.display = 'flex';
}

function closeFolderPopup() {
    const folderPopup = document.getElementById('folderPopup');
    folderPopup.style.display = 'none';
}

async function loadFoldersFromAPI() {
    try {
        const response = await fetch(`${API_BASE}/api/folders/`, {
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        allFolders = data.results || data || [];

        console.log(`✅ 成功加载 ${allFolders.length} 个文件夹`);
    } catch (error) {
        console.error('❌ 加载文件夹失败:', error);
        allFolders = [];
    }
}

async function selectFolder(folderId) {
    currentFolderId = folderId;
    closeFolderPopup();

    // Switch to folders tab
    currentTab = 'folders';
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.tab === 'folders') {
            btn.classList.add('active');
        }
    });

    // Load bookmarks for this folder
    await loadBookmarksByFolder(folderId);
}

async function loadBookmarksByFolder(folderId) {
    try {
        const response = await fetch(`${API_BASE}/api/folders/${folderId}/bookmarks/`, {
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        filteredBookmarks = data.results || data || [];

        renderBookmarks();

        console.log(`✅ 成功加载文件夹 ${folderId} 的 ${filteredBookmarks.length} 个书签`);
    } catch (error) {
        console.error('❌ 加载文件夹书签失败:', error);

        // Fallback: filter bookmarks by folder_id
        filteredBookmarks = allBookmarks.filter(b => b.folder_id === folderId);
        renderBookmarks();
    }
}
