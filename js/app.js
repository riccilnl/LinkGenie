// ========== 系统状态检查与引导 (Onboarding) ==========

async function checkSystemStatus() {
    try {
        const response = await fetch(`${API_BASE}/api/system/status`, { headers });
        if (!response.ok) {
            if (response.status === 401) {
                const isDefault = (API_TOKEN === 'your-secret-token-change-me' || !API_TOKEN);
                const msg = isDefault
                    ? '欢迎使用 LinkGenie，请先设置您的访问密钥。'
                    : '访问认证已失效，请重新输入正确的 Token 密钥以接入。';
                showOnboarding(msg);
                return;
            }
            throw new Error('无法连接到服务器');
        }
        const data = await response.json();

        // 如果数据库没有初始化(无书签)，显示引导页
        if (!data.initialized) {
            showOnboarding('欢迎使用！您的数据库还是空的，让我们开启第一次收藏。');
        } else {
            // 正常加载
            loadBookmarks();
            loadFolders();
        }
    } catch (error) {
        console.error('系统状态检查失败:', error);
        showOnboarding('无法连接到后端服务器，请检查地址是否正确。');
    }
}

function showOnboarding(message) {
    const overlay = document.getElementById('onboardingOverlay');
    if (overlay) {
        overlay.style.display = 'flex';
        const status = document.getElementById('onboardingStatus');
        if (status && message) {
            status.textContent = message;
            status.style.color = '#8e8e93';
        }
    }

    // 自动填充当前的连接配置
    const apiBaseInput = document.getElementById('onboardingApiBase');
    const apiTokenInput = document.getElementById('onboardingApiToken');
    if (apiBaseInput) apiBaseInput.value = API_BASE || '';
    if (apiTokenInput) apiTokenInput.value = (API_TOKEN === 'your-secret-token-change-me') ? '' : API_TOKEN;

    // 获取并填充 AI 配置 (仅用于回填已有的配置，报错不影响新用户填写)
    fetch(`${API_BASE}/api/system/config`, { headers }).then(r => {
        if (!r.ok) return {}; // 报错直接返回空
        return r.json();
    }).then(config => {
        if (config.ai_endpoint) document.getElementById('onboardingAiEndpoint').value = config.ai_endpoint;
        if (config.ai_model) document.getElementById('onboardingAiModel').value = config.ai_model;
    }).catch(() => { });
}

function toggleOnboardingAi() {
    const content = document.getElementById('onboardingAiFields');
    const icon = document.getElementById('aiToggleIcon');
    content.classList.toggle('show');
    if (content.classList.contains('show')) {
        icon.style.transform = 'rotate(180deg)';
    } else {
        icon.style.transform = 'rotate(0deg)';
    }
}

async function testAndSaveOnboarding() {
    const baseInput = document.getElementById('onboardingApiBase');
    const tokenInput = document.getElementById('onboardingApiToken');
    const aiKeyInput = document.getElementById('onboardingAiKey');
    const aiEndpointInput = document.getElementById('onboardingAiEndpoint');
    const aiModelInput = document.getElementById('onboardingAiModel');
    const status = document.getElementById('onboardingStatus');

    let base = baseInput.value.trim();
    if (base && !base.startsWith('http')) {
        base = 'http://' + base;
        baseInput.value = base; // 同步回输入框让用户看见
    }
    const token = tokenInput.value.trim();

    if (!base || !token) {
        status.textContent = '❌ 请填写服务器地址和 Token';
        status.style.color = '#ff453a';
        return;
    }

    status.textContent = '⏳ 正在同步配置...';
    status.style.color = '#0a84ff';

    try {
        const testHeaders = {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        };

        // 1. 先尝试同步 AI 配置到后端 (热重载)
        const aiConfig = {};
        if (aiKeyInput.value.trim()) aiConfig['AI_API_KEY'] = aiKeyInput.value.trim();
        if (aiEndpointInput.value.trim()) aiConfig['AI_ENDPOINT'] = aiEndpointInput.value.trim();
        if (aiModelInput.value.trim()) aiConfig['AI_MODEL'] = aiModelInput.value.trim();
        aiConfig['API_TOKEN'] = token;

        const configResp = await fetch(`${base}/api/system/config`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }, // 此时可能还没 Token，后端已在中间件中放行此路径
            body: JSON.stringify(aiConfig)
        });

        if (!configResp.ok) throw new Error('同步到服务器失败');

        // 2. 验证状态
        const response = await fetch(`${base}/api/system/status`, { headers: testHeaders });

        if (response.ok) {
            // 保存到本地
            localStorage.setItem('api_base', base);
            localStorage.setItem('api_token', token);

            // 更新全局
            API_BASE = base;
            API_TOKEN = token;
            headers = testHeaders;

            status.textContent = '✅ 配置已注入并成功连接！';
            status.style.color = '#34c759';

            setTimeout(() => {
                document.getElementById('onboardingOverlay').style.display = 'none';
                loadBookmarks();
                loadFolders();
            }, 1000);
        } else {
            status.textContent = '❌ 连接验证失败';
            status.style.color = '#ff453a';
        }
    } catch (error) {
        console.error('Onboarding failed:', error);
        status.textContent = '❌ 服务器同步失败，请检查地址';
        status.style.color = '#ff453a';
    }
}

// 保存 API 配置 (原本的设置面板函数)
function saveApiConfig() {
    const base = document.getElementById('apiBaseInput').value.trim();
    const token = document.getElementById('apiTokenInput').value.trim();

    if (!base || !token) {
        alert('请填写完整的 API 配置');
        return;
    }

    // 保存到 localStorage
    localStorage.setItem('api_base', base);
    localStorage.setItem('api_token', token);

    // 更新全局变量
    API_BASE = base;
    API_TOKEN = token;
    headers = {
        'Authorization': `Bearer ${API_TOKEN}`,
        'Content-Type': 'application/json'
    };

    // 显示保存成功提示
    const status = document.getElementById('apiConfigStatus');
    status.style.display = 'inline';
    setTimeout(() => {
        status.style.display = 'none';
    }, 2000);

    // 重新加载书签以验证配置
    loadBookmarks();
}

// ========== 文件夹管理 ==========
let folders = [];
let currentFolderId = null;

// 加载文件夹列表
async function loadFolders() {
    try {
        const response = await fetch(`${API_BASE}/api/folders/`, { headers });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();

        // 验证数据格式
        if (!Array.isArray(data)) {
            console.error('Invalid folder data format:', data);
            folders = [];
        } else {
            folders = data;
        }

        renderFolders();
    } catch (error) {
        console.error('加载文件夹失败:', error);
        folders = [];
        renderFolders();
    }
}

// 渲染文件夹列表
function renderFolders() {
    const container = document.getElementById('folderList');
    if (!container) return;

    const html = folders.map(folder => `
        <div class="category-item ${currentFolderId === folder.id ? 'active' : ''}" 
             onclick="selectFolder(${folder.id}, event)"
             style="position: relative; padding-right: 60px;">
            <span>${folder.icon} ${folder.name}</span>
            <div style="position: absolute; right: 8px; top: 50%; transform: translateY(-50%); display: flex; gap: 4px; align-items: center;">
                <span class="tag-count">${folder.count || 0}</span>
                <button onclick="editFolder(${folder.id}, event)" style="background: transparent; border: none; color: #0a84ff; cursor: pointer; padding: 2px 4px; font-size: 14px;" title="编辑">✏️</button>
                <button onclick="deleteFolder(${folder.id}, event)" style="background: transparent; border: none; color: #ff453a; cursor: pointer; padding: 2px 4px; font-size: 14px;" title="删除">🗑️</button>
            </div>
        </div>
    `).join('');

    container.innerHTML = html;
}

// 选择文件夹
async function selectFolder(folderId, event) {
    currentFolderId = folderId;
    currentCategory = 'folder';

    // 更新UI
    document.querySelectorAll('.category-item').forEach(item => {
        item.classList.remove('active');
    });
    if (event) {
        event.target.closest('.category-item').classList.add('active');
    }

    // 加载文件夹内的书签
    try {
        const response = await fetch(`${API_BASE}/api/folders/${folderId}/bookmarks`, { headers });
        const data = await response.json();
        displayBookmarks(data.results);
    } catch (error) {
        console.error('加载文件夹书签失败:', error);
    }
}

function renderFolderPickers() {
    const colorContainer = document.getElementById('colorPicker');
    if (colorContainer.children.length === 0) {
        colorContainer.innerHTML = presetColors.map(color => `
            <div class="color-option" style="background-color: ${color}" 
                 onclick="selectFolderColor('${color}')" data-value="${color}"></div>
        `).join('');
    }

    const iconContainer = document.getElementById('iconPicker');
    if (iconContainer.children.length === 0) {
        iconContainer.innerHTML = presetIcons.map(icon => `
            <div class="icon-option" onclick="selectFolderIcon('${icon}')" data-value="${icon}">
                ${icon}
            </div>
        `).join('');
    }
}

function selectFolderColor(color) {
    document.getElementById('folderColor').value = color;
    document.querySelectorAll('.color-option').forEach(el => {
        if (el.dataset.value === color) el.classList.add('selected');
        else el.classList.remove('selected');
    });
}

function selectFolderIcon(icon) {
    document.getElementById('folderIcon').value = icon;
    document.querySelectorAll('.icon-option').forEach(el => {
        if (el.dataset.value === icon) el.classList.add('selected');
        else el.classList.remove('selected');
    });
}

// 显示新建文件夹对话框
function showCreateFolderModal() {
    document.getElementById('folderModal').style.display = 'flex';
    document.getElementById('folderModalTitle').textContent = '新建文件夹';
    document.getElementById('folderName').value = '';

    renderFolderPickers();
    selectFolderColor('#0a84ff');
    selectFolderIcon('📁');

    document.getElementById('folderModalSave').onclick = createFolder;
}

// 创建文件夹
async function createFolder() {
    const name = document.getElementById('folderName').value.trim();
    if (!name) {
        alert('请输入文件夹名称');
        return;
    }

    const color = document.getElementById('folderColor').value;
    const icon = document.getElementById('folderIcon').value || '📁';

    try {
        const response = await fetch(`${API_BASE}/api/folders/`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ name, color, icon })
        });

        if (response.ok) {
            closeFolderModal();
            loadFolders();
        }
    } catch (error) {
        console.error('创建文件夹失败:', error);
        alert('创建失败');
    }
}

// 编辑文件夹
function editFolder(id, event) {
    event.stopPropagation();
    const folder = folders.find(f => f.id === id);
    if (!folder) return;

    document.getElementById('folderModal').style.display = 'flex';
    document.getElementById('folderModalTitle').textContent = '编辑文件夹';
    document.getElementById('folderName').value = folder.name;

    renderFolderPickers();
    selectFolderColor(folder.color || '#0a84ff');
    selectFolderIcon(folder.icon || '📁');

    document.getElementById('folderModalSave').onclick = () => updateFolder(id);
}

// 更新文件夹
async function updateFolder(id) {
    const name = document.getElementById('folderName').value.trim();
    if (!name) {
        alert('请输入文件夹名称');
        return;
    }

    const color = document.getElementById('folderColor').value;
    const icon = document.getElementById('folderIcon').value || '📁';

    try {
        const response = await fetch(`${API_BASE}/api/folders/${id}`, {
            method: 'PUT',
            headers,
            body: JSON.stringify({ name, color, icon })
        });

        if (response.ok) {
            closeFolderModal();
            loadFolders();
        }
    } catch (error) {
        console.error('更新文件夹失败:', error);
        alert('更新失败');
    }
}

// 删除文件夹
async function deleteFolder(id, event) {
    event.stopPropagation();
    const folder = folders.find(f => f.id === id);
    if (!confirm(`确定要删除文件夹"${folder.name}"吗？\n\n书签不会被删除，只是移出此文件夹。`)) return;

    try {
        const response = await fetch(`${API_BASE}/api/folders/${id}`, {
            method: 'DELETE',
            headers
        });

        if (response.ok) {
            loadFolders();
            if (currentFolderId === id) {
                currentFolderId = null;
                loadBookmarks();
            }
        }
    } catch (error) {
        console.error('删除文件夹失败:', error);
        alert('删除失败');
    }
}

// 关闭文件夹对话框
function closeFolderModal() {
    document.getElementById('folderModal').style.display = 'none';
}

// 切换文件夹折叠状态
function toggleFolders() {
    const folderSection = document.getElementById('folderSection');
    const isCollapsed = folderSection.classList.toggle('collapsed');
    localStorage.setItem('foldersCollapsed', isCollapsed);
}

// ========== 工作流管理 ==========
let workflows = [];
let currentWorkflowId = null;
let workflowTriggers = [];
let workflowActions = [];

// 显示工作流管理
function showWorkflows() {
    document.getElementById('workflowModal').style.display = 'flex';
    loadWorkflows();
}

// 关闭工作流管理
function closeWorkflowModal() {
    document.getElementById('workflowModal').style.display = 'none';
}

// 加载工作流列表
async function loadWorkflows() {
    try {
        const response = await fetch(`${API_BASE}/api/workflows/`, { headers });
        workflows = await response.json();
        renderWorkflows();
    } catch (error) {
        console.error('加载工作流失败:', error);
    }
}

// 渲染工作流列表
function renderWorkflows() {
    const container = document.getElementById('workflowList');
    if (!workflows.length) {
        container.innerHTML = '<div style="color: #8e8e93; text-align: center; padding: 20px;">暂无工作流，点击上方按钮创建</div>';
        return;
    }

    const html = workflows.map((wf, index) => `
        <div draggable="true" 
             data-workflow-id="${wf.id}" 
             data-index="${index}"
             ondragstart="handleDragStart(event)" 
             ondragover="handleDragOver(event)" 
             ondrop="handleDrop(event)"
             ondragend="handleDragEnd(event)"
             style="background: #2c2c2e; border-radius: 12px; padding: 16px; margin-bottom: 12px; cursor: move;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                <div style="font-weight: 500;">
                    <span style="color: #8e8e93; margin-right: 8px;">☰</span>
                    ${wf.enabled ? '●' : '○'} ${wf.name}
                </div>
                <label style="position: relative; display: inline-block; width: 44px; height: 24px;">
                    <input type="checkbox" ${wf.enabled ? 'checked' : ''} onchange="toggleWorkflow(${wf.id})" style="opacity: 0; width: 0; height: 0;">
                    <span style="position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: ${wf.enabled ? '#34c759' : '#48484a'}; transition: .4s; border-radius: 24px;"></span>
                    <span style="position: absolute; content: ''; height: 18px; width: 18px; left: ${wf.enabled ? '23px' : '3px'}; bottom: 3px; background-color: white; transition: .4s; border-radius: 50%;"></span>
                </label>
            </div>
            <div style="color: #8e8e93; font-size: 13px; margin-bottom: 8px;">${wf.description || '无描述'}</div>
            <div style="color: #8e8e93; font-size: 12px;">
                触发: ${wf.triggers.length} 个条件 | 动作: ${wf.actions.length} 个
            </div>
            <div style="margin-top: 12px; display: flex; gap: 8px;">
                <button class="modal-btn secondary" onclick="runWorkflow(${wf.id})" style="flex: 1; font-size: 13px; padding: 6px 12px; background: #0a84ff;">▶️ 立即运行</button>
                <button class="modal-btn secondary" onclick="editWorkflow(${wf.id})" style="font-size: 13px; padding: 6px 12px;">编辑</button>
                <button class="modal-btn secondary" onclick="deleteWorkflow(${wf.id})" style="font-size: 13px; padding: 6px 12px; background: #ff453a;">删除</button>
            </div>
        </div>
    `).join('');

    container.innerHTML = html;
}

// 拖拽相关变量
let draggedWorkflow = null;

function handleDragStart(e) {
    draggedWorkflow = e.target;
    e.target.style.opacity = '0.4';
}

function handleDragOver(e) {
    if (e.preventDefault) {
        e.preventDefault();
    }
    e.dataTransfer.dropEffect = 'move';
    return false;
}

function handleDrop(e) {
    if (e.stopPropagation) {
        e.stopPropagation();
    }

    if (draggedWorkflow !== e.currentTarget) {
        const draggedIndex = parseInt(draggedWorkflow.dataset.index);
        const targetIndex = parseInt(e.currentTarget.dataset.index);

        // 交换工作流顺序
        const temp = workflows[draggedIndex];
        workflows.splice(draggedIndex, 1);
        workflows.splice(targetIndex, 0, temp);

        // 更新优先级并保存
        updateWorkflowPriorities();
    }

    return false;
}

function handleDragEnd(e) {
    e.target.style.opacity = '1';
}

// 更新工作流优先级
async function updateWorkflowPriorities() {
    try {
        // 批量更新优先级
        for (let i = 0; i < workflows.length; i++) {
            workflows[i].priority = i;
            await fetch(`${API_BASE}/api/workflows/${workflows[i].id}`, {
                method: 'PUT',
                headers,
                body: JSON.stringify({
                    name: workflows[i].name,
                    description: workflows[i].description,
                    enabled: workflows[i].enabled,
                    condition_logic: workflows[i].condition_logic,
                    triggers: workflows[i].triggers.map(t => ({ trigger_type: t.trigger_type, config: t.config })),
                    actions: workflows[i].actions.map(a => ({ action_type: a.action_type, config: a.config }))
                })
            });
        }
        renderWorkflows();
    } catch (error) {
        console.error('更新优先级失败:', error);
    }
}

// 切换工作流启用状态
async function toggleWorkflow(id) {
    try {
        await fetch(`${API_BASE}/api/workflows/${id}/toggle`, {
            method: 'POST',
            headers
        });
        loadWorkflows();
    } catch (error) {
        console.error('切换工作流失败:', error);
    }
}

// 显示工作流编辑器
function showWorkflowEditor(id = null) {
    currentWorkflowId = id;
    workflowTriggers = [];
    workflowActions = [{ type: 'move_to_folder', config: { folder_id: folders[0]?.id || 1 } }];

    document.getElementById('workflowListView').style.display = 'none';
    document.getElementById('workflowListActions').style.display = 'none';
    document.getElementById('workflowEditorView').style.display = 'block';

    if (id) {
        const wf = workflows.find(w => w.id === id);
        document.getElementById('workflowName').value = wf.name;
        document.getElementById('workflowDesc').value = wf.description;
        document.getElementById('workflowLogic').value = wf.condition_logic;
        workflowTriggers = wf.triggers.map(t => ({ type: t.trigger_type, config: t.config }));
        workflowActions = wf.actions.map(a => ({ type: a.action_type, config: a.config }));
    } else {
        document.getElementById('workflowName').value = '';
        document.getElementById('workflowDesc').value = '';
        document.getElementById('workflowLogic').value = 'OR';
    }

    renderTriggers();
    renderActions();
    document.getElementById('saveWorkflowBtn').onclick = saveWorkflow;
}

// 返回工作流列表
function backToWorkflowList() {
    document.getElementById('workflowListView').style.display = 'block';
    document.getElementById('workflowListActions').style.display = 'block';
    document.getElementById('workflowEditorView').style.display = 'none';
}

// 添加触发条件
function addTrigger() {
    workflowTriggers.push({ type: 'bookmark_created', config: {} });
    renderTriggers();
}

// 渲染触发条件
function renderTriggers() {
    const container = document.getElementById('triggerList');
    const html = workflowTriggers.map((trigger, index) => `
        <div style="background: #2c2c2e; padding: 10px; border-radius: 8px; margin-bottom: 8px;">
            <div style="display: flex; gap: 8px; align-items: center;">
                <select onchange="updateTriggerType(${index}, this.value)" class="form-input" style="flex: 0 0 150px; padding: 8px 10px;">
                    <optgroup label="条件触发器">
                        <option value="url_match" ${trigger.type === 'url_match' ? 'selected' : ''}>URL匹配</option>
                        <option value="keyword_match" ${trigger.type === 'keyword_match' ? 'selected' : ''}>关键字匹配</option>
                    </optgroup>
                    <optgroup label="事件触发器">
                        <option value="bookmark_created" ${trigger.type === 'bookmark_created' ? 'selected' : ''}>📌 书签已添加</option>
                        <option value="bookmark_updated" ${trigger.type === 'bookmark_updated' ? 'selected' : ''}>✏️ 书签已更新</option>
                        <option value="bookmark_deleted" ${trigger.type === 'bookmark_deleted' ? 'selected' : ''}>🗑️ 书签已删除</option>
                        <option value="title_changed" ${trigger.type === 'title_changed' ? 'selected' : ''}>📝 标题已更改</option>
                        <option value="description_added" ${trigger.type === 'description_added' ? 'selected' : ''}>📄 描述已添加</option>
                        <option value="bookmark_tagged" ${trigger.type === 'bookmark_tagged' ? 'selected' : ''}>🏷️ 书签已标记</option>
                    </optgroup>
                </select>
                ${['url_match', 'keyword_match'].includes(trigger.type) ? `
                    <input type="text" class="form-input" placeholder="${trigger.type === 'url_match' ? '例如：github.com' : '例如：API'}" value="${trigger.config.value || ''}" onchange="updateTriggerConfig(${index}, 'value', this.value)" style="flex: 1; padding: 8px 10px;">
                ` : `
                    <div style="flex: 1; padding: 8px 10px; color: #8e8e93; font-size: 13px;">
                        ${trigger.type === 'bookmark_created' ? '当创建新书签时触发' :
            trigger.type === 'bookmark_updated' ? '当更新书签时触发' :
                trigger.type === 'bookmark_deleted' ? '当删除书签时触发' :
                    trigger.type === 'title_changed' ? '当书签标题被修改时触发' :
                        trigger.type === 'description_added' ? '当向书签添加描述时触发' :
                            trigger.type === 'bookmark_tagged' ? '当向书签添加标签时触发' : '事件触发器'}
                    </div>
                `}
                ${trigger.type === 'keyword_match' ? `
                    <select class="form-input" onchange="updateTriggerConfig(${index}, 'field', this.value)" style="flex: 0 0 100px; padding: 8px 10px;">
                        <option value="title" ${trigger.config.field === 'title' ? 'selected' : ''}>标题</option>
                        <option value="description" ${trigger.config.field === 'description' ? 'selected' : ''}>描述</option>
                        <option value="both" ${trigger.config.field === 'both' ? 'selected' : ''}>标题+描述</option>
                    </select>
                ` : ''}
                <button onclick="removeTrigger(${index})" style="background: #ff453a; border: none; color: white; padding: 8px 12px; border-radius: 6px; cursor: pointer; flex-shrink: 0;">🗑️</button>
            </div>
        </div>
    `).join('');
    container.innerHTML = html || '<div style="color: #8e8e93;">暂无触发条件</div>';

    // 更新匹配预览
    updateMatchPreview();
}

// 更新匹配书签预览
async function updateMatchPreview() {
    if (!workflowTriggers.length) {
        document.getElementById('matchPreview').innerHTML = '';
        return;
    }

    // 简化版：统计包含关键字的书签
    let matchCount = 0;
    const logic = document.getElementById('workflowLogic').value;

    allBookmarksData.forEach(bookmark => {
        const results = workflowTriggers.map(trigger => {
            if (trigger.type === 'url_match') {
                return bookmark.url.includes(trigger.config.value || '');
            } else if (trigger.type === 'keyword_match') {
                const field = trigger.config.field || 'title';
                const value = (trigger.config.value || '').toLowerCase();
                if (!value) return false;

                if (field === 'title') {
                    return (bookmark.title || '').toLowerCase().includes(value);
                } else if (field === 'description') {
                    return (bookmark.description || '').toLowerCase().includes(value);
                } else {
                    return ((bookmark.title || '') + ' ' + (bookmark.description || '')).toLowerCase().includes(value);
                }
            }
            return false;
        });

        const matched = logic === 'AND' ? results.every(r => r) : results.some(r => r);
        if (matched) matchCount++;
    });

    const previewEl = document.getElementById('matchPreview');
    if (previewEl) {
        previewEl.innerHTML = `<div style="color: #0a84ff; font-size: 13px; margin-top: 8px;">📊 预计匹配 ${matchCount} 个书签</div>`;
    }
}

// 更新触发条件类型
function updateTriggerType(index, type) {
    workflowTriggers[index] = {
        type,
        config: type === 'url_match' ? { match_mode: 'contains', value: '' } : { field: 'title', match_mode: 'contains', value: '', case_sensitive: false }
    };
    renderTriggers();
}

// 更新触发条件配置
function updateTriggerConfig(index, key, value) {
    workflowTriggers[index].config[key] = value;
}

// 删除触发条件
function removeTrigger(index) {
    workflowTriggers.splice(index, 1);
    renderTriggers();
}

// 渲染执行动作
function renderActions() {
    const container = document.getElementById('actionList');

    if (!folders || folders.length === 0) {
        container.innerHTML = `
            <div style="background: #2c2c2e; padding: 12px; border-radius: 8px; text-align: center;">
                <div style="color: #ff9f0a; margin-bottom: 8px;">⚠️ 暂无可用文件夹</div>
                <div style="color: #8e8e93; font-size: 13px;">请先在左侧创建文件夹</div>
            </div>
        `;
        return;
    }

    const html = `
        <div style="background: #2c2c2e; padding: 12px; border-radius: 8px;">
            <div style="margin-bottom: 8px;">移动到文件夹</div>
            <select class="form-input" onchange="updateActionFolder(this.value)">
                ${folders.map(f => `<option value="${f.id}" ${workflowActions[0]?.config.folder_id === f.id ? 'selected' : ''}>${f.icon} ${f.name}</option>`).join('')}
            </select>
        </div>
    `;
    container.innerHTML = html;
}

// 更新动作文件夹
function updateActionFolder(folderId) {
    workflowActions[0].config.folder_id = parseInt(folderId);
}

// 保存工作流
async function saveWorkflow() {
    const name = document.getElementById('workflowName').value.trim();
    if (!name) {
        alert('请输入工作流名称');
        return;
    }

    if (!workflowTriggers.length) {
        alert('请至少添加一个触发条件');
        return;
    }

    const data = {
        name,
        description: document.getElementById('workflowDesc').value.trim(),
        enabled: true,
        condition_logic: document.getElementById('workflowLogic').value,
        triggers: workflowTriggers.map(t => ({ trigger_type: t.type, config: t.config })),
        actions: workflowActions.map(a => ({ action_type: a.type, config: a.config }))
    };

    try {
        const url = currentWorkflowId ? `${API_BASE}/api/workflows/${currentWorkflowId}` : `${API_BASE}/api/workflows/`;
        const method = currentWorkflowId ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method,
            headers,
            body: JSON.stringify(data)
        });

        if (response.ok) {
            backToWorkflowList();
            loadWorkflows();
        }
    } catch (error) {
        console.error('保存工作流失败:', error);
        alert('保存失败');
    }
}

// 编辑工作流
function editWorkflow(id) {
    showWorkflowEditor(id);
}

// 删除工作流
async function deleteWorkflow(id) {
    if (!confirm('确定要删除此工作流吗？')) return;

    try {
        await fetch(`${API_BASE}/api/workflows/${id}`, {
            method: 'DELETE',
            headers
        });
        loadWorkflows();
    } catch (error) {
        console.error('删除工作流失败:', error);
    }
}

// 立即运行工作流
async function runWorkflow(id) {
    const workflow = workflows.find(w => w.id === id);
    if (!workflow) return;

    if (!confirm(`确定要对所有书签运行工作流"${workflow.name}"吗？`)) return;

    try {
        // 获取所有书签ID
        const bookmarkIds = allBookmarksData.map(bm => bm.id);

        const response = await fetch(`${API_BASE}/api/workflows/apply`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
                workflow_ids: [id],
                bookmark_ids: bookmarkIds
            })
        });

        if (response.ok) {
            // 显示成功提示
            const toast = document.createElement('div');
            toast.textContent = `✓ 工作流"${workflow.name}"已执行完成`;
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

            // 重新加载书签以显示更新
            loadBookmarks();
        } else {
            alert('执行失败');
        }
    } catch (error) {
        console.error('运行工作流失败:', error);
        alert('执行失败');
    }
}

let currentCategory = 'all';
let currentBookmarkId = null;
let deleteBookmarkId = null;
let bookmarkUrls = new Set(); // URL缓存,用于快速检查重复
let allBookmarksData = []; // 所有书签数据,用于前端过滤

// 加载书签
async function loadBookmarks(search = '') {
    try {
        const url = search
            ? `${API_BASE}/api/bookmarks/?q=${encodeURIComponent(search)}`
            : `${API_BASE}/api/bookmarks/`;

        const response = await fetch(url, { headers });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();

        // 验证数据格式
        if (!data || !data.results || !Array.isArray(data.results)) {
            console.error('Invalid data format:', data);
            allBookmarksData = [];
            displayBookmarks([]);
            return;
        }

        // 保存数据用于过滤
        allBookmarksData = data.results;

        displayBookmarks(data.results);
        loadTags(data.results);

        // 更新URL缓存
        bookmarkUrls.clear();
        data.results.forEach(bm => bookmarkUrls.add(bm.url));
    } catch (error) {
        console.error('加载失败:', error);
        allBookmarksData = [];
        displayBookmarks([]);
    }
}

// 跟踪正在AI处理的书签
const aiProcessingBookmarks = new Set();

// 显示书签
function displayBookmarks(bookmarks) {
    const list = document.getElementById('bookmarkList');

    // 安全检查: 确保bookmarks是数组
    if (!bookmarks || !Array.isArray(bookmarks)) {
        console.error('displayBookmarks received invalid data:', bookmarks);
        bookmarks = [];
    }

    let html = bookmarks.map(bm => {
        const isProcessing = aiProcessingBookmarks.has(bm.id);
        const processingClass = isProcessing ? 'ai-processing' : '';
        const aiBadge = isProcessing ? '<span class="ai-badge"><span class="spinner"></span>AI处理中</span>' : '';

        return `
        <div class="bookmark-item ${processingClass}" data-id="${bm.id}" onclick="openBookmark('${bm.url}')">
            <div class="bookmark-header">
                <div class="bookmark-title">${bm.title || '无标题'}${aiBadge}</div>
                <div class="bookmark-actions" onclick="event.stopPropagation()">
                    <button class="action-btn" onclick="event.stopPropagation(); copyLink('${bm.url}')" title="复制链接">🔗</button>
                    <button class="action-btn" onclick="event.stopPropagation(); triggerAI(${bm.id})" title="AI处理">🤖</button>
                    <button class="action-btn" onclick="event.stopPropagation(); editBookmark(${bm.id})" title="编辑">✏️</button>
                    <button class="action-btn delete" onclick="event.stopPropagation(); deleteBookmark(${bm.id})" title="删除">🗑️</button>
                </div>
            </div>
            <div class="bookmark-desc">${bm.description || ''}</div>
            <div class="bookmark-time">${new Date(bm.date_added).toLocaleString('zh-CN')}</div>
        </div>
    `;
    }).join('');

    // 追加底线提示
    if (bookmarks.length > 0) {
        html += `<div class="list-footer">啊！我也是有点底线的</div>`;
    } else {
        html = `<div class="list-footer">暂无书签</div>`;
    }

    list.innerHTML = html;
}

// 更新单个书签卡片(不重新渲染整个列表)
function updateSingleBookmarkCard(bookmark) {
    const list = document.getElementById('bookmarkList');
    // 使用 data-id 查找卡片，而不是依赖索引
    const card = list.querySelector(`.bookmark-item[data-id="${bookmark.id}"]`);

    // 如果卡片不在当前视图中(可能被过滤掉了)，则更新数据但不更新UI
    const index = allBookmarksData.findIndex(bm => bm.id === bookmark.id);
    if (index !== -1) {
        allBookmarksData[index] = bookmark;
    }

    if (!card) return;

    // 生成新的卡片HTML
    const isProcessing = aiProcessingBookmarks.has(bookmark.id);
    const processingClass = isProcessing ? 'ai-processing' : '';
    const aiBadge = isProcessing ? '<span class="ai-badge"><span class="spinner"></span>AI处理中</span>' : '';

    const newCardHTML = `
        <div class="bookmark-item ${processingClass}" data-id="${bookmark.id}" onclick="openBookmark('${bookmark.url}')">
            <div class="bookmark-header">
                <div class="bookmark-title">${bookmark.title || '无标题'}${aiBadge}</div>
                <div class="bookmark-actions" onclick="event.stopPropagation()">
                    <button class="action-btn" onclick="event.stopPropagation(); copyLink('${bookmark.url}')" title="复制链接">🔗</button>
                    <button class="action-btn" onclick="event.stopPropagation(); triggerAI(${bookmark.id})" title="AI处理">🤖</button>
                    <button class="action-btn" onclick="event.stopPropagation(); editBookmark(${bookmark.id})" title="编辑">✏️</button>
                    <button class="action-btn delete" onclick="event.stopPropagation(); deleteBookmark(${bookmark.id})" title="删除">🗑️</button>
                </div>
            </div>
            <div class="bookmark-desc">${bookmark.description || ''}</div>
            <div class="bookmark-time">${new Date(bookmark.date_added).toLocaleString('zh-CN')}</div>
        </div>
    `;

    // 使用 insertAdjacentHTML + remove 替代 outerHTML
    // 这种方法在所有浏览器中都能正确触发 DOM 更新
    card.insertAdjacentHTML('afterend', newCardHTML);
    card.remove();

    console.log(`✅ 已更新书签卡片 ID=${bookmark.id}, 标题="${bookmark.title}"`);
}

// 加载类别
function loadCategories(bookmarks) {
    const tags = new Set();
    bookmarks.forEach(bm => bm.tag_names.forEach(tag => tags.add(tag)));

    const list = document.getElementById('categoryList');
    list.innerHTML = Array.from(tags).map(tag =>
        `<div class="category-item" onclick="selectCategory('${tag}')">${tag}</div>`
    ).join('');
}

// 加载标签
function loadTags(bookmarks) {
    const tagCounts = {};
    bookmarks.forEach(bm => {
        bm.tag_names.forEach(tag => {
            tagCounts[tag] = (tagCounts[tag] || 0) + 1;
        });
    });

    const list = document.getElementById('tagList');
    list.innerHTML = Object.entries(tagCounts)
        .sort((a, b) => b[1] - a[1])
        .map(([tag, count]) =>
            `<div class="category-item" onclick="selectCategory('${tag}', 'tag', event)">
                <span>${tag}</span>
                <span class="tag-count">${count}</span>
            </div>`
        ).join('');

    // 更新分类计数
    document.getElementById('totalCount').textContent = bookmarks.length;
    document.getElementById('unreadCount').textContent = bookmarks.filter(bm => bm.unread).length;
    document.getElementById('favoriteCount').textContent = bookmarks.filter(bm => bm.is_favorite).length;
}


// 选择类别或标签
function selectCategory(value, type, event) {
    currentCategory = value;
    document.querySelectorAll('.category-item').forEach(el => el.classList.remove('active'));
    if (event) {
        event.target.closest('.category-item').classList.add('active');
    }

    // 根据类型过滤
    if (type === 'category') {
        if (value === 'all') {
            // 显示所有书签
            displayBookmarks(allBookmarksData);
        } else if (value === 'unread') {
            // 过滤未读
            const filtered = allBookmarksData.filter(bm => bm.unread);
            displayBookmarks(filtered);
        } else if (value === 'favorite') {
            // 过滤收藏
            const filtered = allBookmarksData.filter(bm => bm.is_favorite);
            displayBookmarks(filtered);
        }
    } else if (type === 'tag') {
        // 按标签过滤
        const filtered = allBookmarksData.filter(bm => bm.tag_names.includes(value));
        displayBookmarks(filtered);
    }
}

// 打开书签
function openBookmark(url) {
    window.open(url, '_blank');
}

// 复制链接
function copyLink(url) {
    // 检查 Clipboard API 是否可用 (HTTPS 或 localhost)
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(() => {
            showCopySuccess();
        }).catch(err => {
            console.error('Clipboard API 失败:', err);
            fallbackCopy(url);
        });
    } else {
        // 降级方案：使用传统方法
        fallbackCopy(url);
    }
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

// 降级复制方法 (兼容非HTTPS环境)
function fallbackCopy(url) {
    const textArea = document.createElement('textarea');
    textArea.value = url;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    try {
        const successful = document.execCommand('copy');
        if (successful) {
            showCopySuccess();
        } else {
            alert('复制失败，请手动复制:\n' + url);
        }
    } catch (err) {
        console.error('复制失败:', err);
        alert('复制失败，请手动复制:\n' + url);
    }

    document.body.removeChild(textArea);
}

// 显示添加弹层
function showAddModal() {
    document.getElementById('modalTitle').textContent = '添加书签';
    document.getElementById('bookmarkForm').reset();
    document.getElementById('errorMessage').classList.remove('show');
    currentBookmarkId = null;

    // 显示所有字段,保持与编辑模式一致
    document.getElementById('titleGroup').style.display = 'block';
    document.getElementById('descGroup').style.display = 'block';
    document.getElementById('tagsGroup').style.display = 'block';

    document.getElementById('editModal').classList.add('show');
}

// 编辑书签
async function editBookmark(id) {
    try {
        const response = await fetch(`${API_BASE}/api/bookmarks/${id}/`, { headers });
        const bookmark = await response.json();

        currentBookmarkId = id;
        document.getElementById('modalTitle').textContent = '编辑书签';
        document.getElementById('errorMessage').classList.remove('show');

        // 编辑模式显示所有字段
        document.getElementById('titleGroup').style.display = 'block';
        document.getElementById('descGroup').style.display = 'block';
        document.getElementById('tagsGroup').style.display = 'block';

        document.getElementById('urlInput').value = bookmark.url;
        document.getElementById('titleInput').value = bookmark.title || '';
        document.getElementById('descInput').value = bookmark.description || '';
        document.getElementById('tagsInput').value = bookmark.tag_names.join(', ');
        document.getElementById('editModal').classList.add('show');
    } catch (error) {
        alert('加载书签失败');
    }
}

// 删除书签
function deleteBookmark(id) {
    deleteBookmarkId = id;
    document.getElementById('deleteModal').classList.add('show');
}

// 手动触发AI处理
async function triggerAI(id) {
    let checkInterval = null; // 在外部声明,以便在 catch 中访问

    try {
        // 添加到处理中列表
        aiProcessingBookmarks.add(id);
        showToast('🤖 AI处理中,请稍候...');

        // 立即更新卡片显示处理动画
        const currentBm = allBookmarksData.find(b => b.id === id);
        if (currentBm) {
            updateSingleBookmarkCard(currentBm);
        }

        // 获取当前状态作为对比
        let previousTitle = currentBm ? (currentBm.title || '') : '';
        let previousDesc = currentBm ? (currentBm.description || '') : '';

        // 标记API请求是否完成
        let apiRequestFinished = false;
        let apiRequestFailed = false; // 新增:标记请求是否失败

        // 发起请求(不阻塞UI)
        fetch(`${API_BASE}/api/bookmarks/${id}/enhance/`, {
            method: 'POST',
            headers
        }).then(async (response) => {
            if (!response.ok) throw new Error('AI request failed');
            // 请求完成后,稍微延迟一下标记,确保轮询能捕捉到
            setTimeout(() => { apiRequestFinished = true; }, 500);
        }).catch(error => {
            console.error(error);
            apiRequestFailed = true; // 标记为失败
            apiRequestFinished = true; // 出错也算完成
            showToast('❌ AI请求失败');

            // 立即清除轮询并移除处理状态
            if (checkInterval) {
                clearInterval(checkInterval);
                aiProcessingBookmarks.delete(id);
                // 刷新卡片以移除闪光效果
                const bookmark = allBookmarksData.find(b => b.id === id);
                if (bookmark) {
                    updateSingleBookmarkCard(bookmark);
                }
            }
        });

        // 等待AI处理完成(轮询检查)
        let attempts = 0;
        const maxAttempts = 30; // 30秒超时

        checkInterval = setInterval(async () => {
            attempts++;

            // 如果请求已失败,立即停止轮询
            if (apiRequestFailed) {
                clearInterval(checkInterval);
                aiProcessingBookmarks.delete(id);
                return;
            }

            try {
                // 重新加载书签数据
                const response = await fetch(`${API_BASE}/api/bookmarks/${id}/`, { headers });
                if (!response.ok) throw new Error('Failed to fetch bookmark');
                const bookmark = await response.json();

                // 🔍 调试日志
                console.log(`[AI轮询 #${attempts}] 书签ID=${id}`);
                console.log(`  当前标题: "${bookmark.title}"`);
                console.log(`  当前描述: "${bookmark.description?.substring(0, 50)}..."`);
                console.log(`  之前标题: "${previousTitle}"`);
                console.log(`  之前描述: "${previousDesc?.substring(0, 50)}..."`);

                // 检查变化
                const titleChanged = bookmark.title !== previousTitle;
                const descChanged = bookmark.description !== previousDesc;

                console.log(`  标题变化: ${titleChanged}, 描述变化: ${descChanged}`);

                // 如果有变化,更新内存数据并立即刷新UI
                if (titleChanged || descChanged) {
                    console.log(`  ✅ 检测到变化,更新UI`);
                    previousTitle = bookmark.title;
                    previousDesc = bookmark.description;

                    // 更新内存中的数据
                    const index = allBookmarksData.findIndex(bm => bm.id === id);
                    if (index !== -1) {
                        allBookmarksData[index] = bookmark;
                        console.log(`  ✅ 已更新内存数据 index=${index}`);
                    }

                    // ✅ 立即更新卡片UI,让用户看到实时变化
                    updateSingleBookmarkCard(bookmark);
                    console.log(`  ✅ 已调用 updateSingleBookmarkCard`);
                } else {
                    console.log(`  ⏳ 无变化,继续等待...`);
                }

                // 结束条件: 
                // 1. API请求已完成 AND 书签有完整内容
                // 2. 或者超时
                const hasContent = bookmark.title && bookmark.description;

                console.log(`  API完成: ${apiRequestFinished}, 有内容: ${hasContent}, 尝试次数: ${attempts}/${maxAttempts}`);

                if ((apiRequestFinished && hasContent) || attempts >= maxAttempts) {
                    console.log(`  🎉 轮询结束`);
                    clearInterval(checkInterval);
                    aiProcessingBookmarks.delete(id);

                    // 最终更新完整卡片
                    updateSingleBookmarkCard(bookmark);
                    console.log(`  ✅ 最终更新卡片`);

                    // 刷新文件夹列表(计数可能已变化)
                    loadFolders();

                    if (attempts < maxAttempts) {
                        showToast('✅ AI处理完成!');
                    } else {
                        showToast('⏱️ AI处理超时(但已保存)');
                    }
                }
            } catch (error) {
                console.error('状态检查失败:', error);
                // 如果状态检查连续失败,也应该停止轮询
                if (attempts >= 5) {
                    clearInterval(checkInterval);
                    aiProcessingBookmarks.delete(id);
                    showToast('❌ 状态检查失败,已停止轮询');
                }
            }
        }, 1000);

    } catch (error) {
        if (checkInterval) {
            clearInterval(checkInterval);
        }
        aiProcessingBookmarks.delete(id);
        alert('启动AI失败: ' + error.message);
    }
}

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

// 确认删除
async function confirmDelete() {
    try {
        await fetch(`${API_BASE}/api/bookmarks/${deleteBookmarkId}/`, {
            method: 'DELETE',
            headers
        });
        closeDeleteModal();
        loadBookmarks();
    } catch (error) {
        alert('删除失败');
    }
}


// 搜索
function searchBookmarks() {
    const search = document.getElementById('searchInput').value;
    loadBookmarks(search);
}

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

// ========== 标签优化功能 ==========

// 加载标签统计
async function loadTagStats() {
    try {
        const response = await fetch(`${API_BASE}/api/tags/stats`, { headers });
        if (!response.ok) throw new Error('获取统计失败');

        const stats = await response.json();

        // 显示统计容器
        document.getElementById('tagStatsContainer').style.display = 'block';

        // 更新统计数据
        document.getElementById('tagStatsTotal').textContent = stats.total || 0;
        document.getElementById('tagStatsCore').textContent = stats.core || 0;
        document.getElementById('tagStatsFixed').textContent = stats.fixed || 0;
        document.getElementById('tagStatsDynamic').textContent = stats.dynamic || 0;

        // 显示优化建议
        const needsOptimization = stats.optimization_needed || false;
        document.getElementById('tagOptimizationNeeded').style.display = needsOptimization ? 'block' : 'none';

    } catch (error) {
        console.error('加载标签统计失败:', error);
        alert('加载统计失败: ' + error.message);
    }
}

// 预览标签优化
async function previewTagOptimization() {
    try {
        const response = await fetch(`${API_BASE}/api/tags/optimize`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
                dry_run: true,
                enable_merge: true,
                enable_promotion: true
            })
        });

        if (!response.ok) throw new Error('预览失败');

        const result = await response.json();

        // 显示预览容器
        document.getElementById('optimizationPreview').style.display = 'block';

        // 渲染优化操作列表
        const actionsContainer = document.getElementById('optimizationActions');
        if (result.actions && result.actions.length > 0) {
            actionsContainer.innerHTML = result.actions.map(action => {
                if (action.type === 'merge') {
                    return `<div style="padding: 6px 0; border-bottom: 1px solid #3a3a3c;">
                        🔀 合并: <span style="color: #ff453a;">${action.source}</span> → 
                        <span style="color: #34c759;">${action.target}</span>
                        <span style="color: #636366; margin-left: 8px;">(相似度: ${(action.similarity * 100).toFixed(0)}%)</span>
                    </div>`;
                } else if (action.type === 'promote') {
                    const color = action.to === 'fixed' ? '#0a84ff' : '#ff9f0a';
                    return `<div style="padding: 6px 0; border-bottom: 1px solid #3a3a3c;">
                        ⬆️ 晋升: <span style="color: ${color};">${action.tag}</span>
                        <span style="color: #636366; margin-left: 8px;">(${action.from} → ${action.to}, 使用${action.usage_count}次)</span>
                    </div>`;
                }
                return '';
            }).join('');
        } else {
            actionsContainer.innerHTML = '<div style="color: #34c759; text-align: center; padding: 20px;">✓ 标签已优化,无需调整</div>';
        }

        // 显示摘要
        const summary = result.summary;
        document.getElementById('optimizationSummary').innerHTML = `
            <div style="color: #fff;">
                将合并 <span style="color: #ff9f0a;">${summary.total_merges}</span> 个标签,
                晋升 <span style="color: #0a84ff;">${summary.total_promotions}</span> 个标签
            </div>
            <div style="color: #8e8e93; margin-top: 4px;">
                标签总数: ${summary.tags_before} → ${summary.tags_after}
            </div>
        `;

    } catch (error) {
        console.error('预览优化失败:', error);
        alert('预览失败: ' + error.message);
    }
}

// 执行标签优化
async function executeTagOptimization() {
    if (!confirm('确定要执行标签优化吗?\n\n此操作会合并同义词标签,建议先预览查看优化计划。')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/tags/optimize`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
                dry_run: false,
                enable_merge: true,
                enable_promotion: true
            })
        });

        if (!response.ok) throw new Error('优化失败');

        const result = await response.json();
        const summary = result.summary;

        alert(`✅ 优化完成!\n\n合并了 ${summary.total_merges} 个标签\n晋升了 ${summary.total_promotions} 个标签\n标签总数: ${summary.tags_before} → ${summary.tags_after}`);

        // 刷新统计和标签列表
        loadTagStats();
        loadTags();

        // 隐藏预览
        document.getElementById('optimizationPreview').style.display = 'none';

    } catch (error) {
        console.error('执行优化失败:', error);
        alert('优化失败: ' + error.message);
    }
}


// 处理导入文件选择
function handleImportFile() {
    const fileInput = document.getElementById('importFile');
    const file = fileInput.files[0];
    if (file) {
        document.getElementById('importFileName').textContent = file.name;
        importBookmarksFile(file);
    }
}

// 导入书签
async function importBookmarksFile(file) {
    const formData = new FormData();
    formData.append('file', file);

    document.getElementById('importProgress').style.display = 'block';
    document.getElementById('importProgressBar').style.width = '0%';

    try {
        const response = await fetch(`${API_BASE}/api/bookmarks/import/`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${API_TOKEN}`
            },
            body: formData
        });

        const result = await response.json();
        document.getElementById('importProgressBar').style.width = '100%';

        setTimeout(() => {
            alert(`导入完成!\n成功: ${result.success}\n失败: ${result.failed}`);
            closeSettingsModal();
            loadBookmarks();
        }, 500);
    } catch (error) {
        alert('导入失败: ' + error.message);
        document.getElementById('importProgress').style.display = 'none';
    }
}

// 导出书签
async function exportBookmarks() {
    try {
        const response = await fetch(`${API_BASE}/api/bookmarks/export/`, { headers });
        const blob = await response.blob();

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `bookmarks_${new Date().toISOString().split('T')[0]}.html`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
    } catch (error) {
        alert('导出失败: ' + error.message);
    }
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

// 注册 Service Worker
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js')
            .then(registration => {
                console.log('✅ ServiceWorker 注册成功:', registration.scope);
            })
            .catch(error => {
                console.log('❌ ServiceWorker 注册失败:', error);
            });
    });
}

// 初始系统检查
checkSystemStatus();

// 绑定书签表单提交事件
const bookmarkForm = document.getElementById('bookmarkForm');
if (bookmarkForm) {
    bookmarkForm.addEventListener('submit', async function (e) {
        e.preventDefault();

        const url = document.getElementById('urlInput').value.trim();
        const title = document.getElementById('titleInput').value.trim();
        const description = document.getElementById('descInput').value.trim();
        const tags = document.getElementById('tagsInput').value.trim();

        const data = {
            url,
            title: title || undefined,
            description: description || undefined,
            tag_names: tags ? tags.split(',').map(t => t.trim()).filter(t => t) : []
        };

        try {
            let response;
            if (currentBookmarkId) {
                // 编辑模式 - 更新书签
                response = await fetch(`${API_BASE}/api/bookmarks/${currentBookmarkId}/`, {
                    method: 'PUT',
                    headers,
                    body: JSON.stringify(data)
                });
            } else {
                // 添加模式 - 创建新书签
                response = await fetch(`${API_BASE}/api/bookmarks/`, {
                    method: 'POST',
                    headers,
                    body: JSON.stringify(data)
                });
            }

            if (response.ok) {
                closeModal();
                loadBookmarks();
                loadFolders();
                showToast(currentBookmarkId ? '✅ 更新成功!' : '✅ 添加成功!');
            } else {
                const error = await response.text();
                document.getElementById('errorMessage').textContent = error || '保存失败';
                document.getElementById('errorMessage').classList.add('show');
            }
        } catch (error) {
            console.error('保存书签失败:', error);
            document.getElementById('errorMessage').textContent = '保存失败: ' + error.message;
            document.getElementById('errorMessage').classList.add('show');
        }
    });
}

// 绑定搜索框回车事件
const searchInput = document.getElementById('searchInput');
if (searchInput) {
    searchInput.addEventListener('keypress', function (e) {
        if (e.key === 'Enter') {
            searchBookmarks();
        }
    });
}

// 恢复文件夹折叠状态
const foldersCollapsed = localStorage.getItem('foldersCollapsed') === 'true';
if (foldersCollapsed) {
    document.getElementById('folderSection').classList.add('collapsed');
}
