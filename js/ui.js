// ========== UI 控制 ==========

// 显示提示消息
function showToast(message) {
    const toast = document.createElement('div');
    toast.textContent = message;
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #0a84ff;
        color: white;
        padding: 12px 20px;
        border-radius: 8px;
        font-size: 14px;
        z-index: 10000;
        animation: slideIn 0.3s ease;
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 2500);
}

// 显示复制成功提示
function showCopySuccess() {
    const toast = document.createElement('div');
    toast.textContent = '✓ 链接已复制';
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #34c759;
        color: white;
        padding: 12px 20px;
        border-radius: 8px;
        font-size: 14px;
        z-index: 10000;
        animation: slideIn 0.3s ease;
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 2000);
}

// 关闭弹层
function closeModal() {
    document.getElementById('editModal').classList.remove('show');
    document.getElementById('errorMessage').classList.remove('show');
}

function closeDeleteModal() {
    document.getElementById('deleteModal').classList.remove('show');
}

function showSettings() {
    // 自动填充当前 API 配置
    document.getElementById('apiBaseInput').value = API_BASE;
    document.getElementById('apiTokenInput').value = API_TOKEN;
    document.getElementById('settingsModal').classList.add('show');
}

function closeSettingsModal() {
    document.getElementById('settingsModal').classList.remove('show');
    document.getElementById('importFileName').textContent = '';
    document.getElementById('importProgress').style.display = 'none';
}

// 搜索框清除按钮
function toggleClearBtn() {
    const input = document.getElementById('searchInput');
    const btn = document.getElementById('searchClearBtn');
    if (btn) btn.style.display = input.value ? 'flex' : 'none';
}

function clearSearch() {
    const input = document.getElementById('searchInput');
    input.value = '';
    toggleClearBtn();
    searchBookmarks();
}

// 抽屉菜单控制
function toggleDrawer() {
    // 仅在移动端有效
    if (window.innerWidth > 768) return;

    const list = document.getElementById('categoryListSidebar');
    const overlay = document.getElementById('drawerOverlay');

    list.classList.toggle('show');
    overlay.classList.toggle('show');
}

function closeDrawer() {
    const list = document.getElementById('categoryListSidebar');
    const overlay = document.getElementById('drawerOverlay');

    list.classList.remove('show');
    overlay.classList.remove('show');
}

// UI 函数已定义为全局函数,可在其他模块中直接使用
