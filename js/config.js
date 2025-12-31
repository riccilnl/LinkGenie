// ========== 配置管理 ==========

// API 配置 - 从 localStorage 读取，如果没有则使用默认值
let API_BASE = localStorage.getItem('api_base') || 'http://localhost:8080';
let API_TOKEN = localStorage.getItem('api_token') || 'your-secret-token-change-me';

// 更新 headers
let headers = {
    'Authorization': `Bearer ${API_TOKEN}`,
    'Content-Type': 'application/json'
};

// 预定义颜色和图标
const presetColors = [
    '#ff453a', '#ff9f0a', '#ffd60a', '#32d74b', '#64d2ff', '#0a84ff', '#5e5ce6', '#bf5af2', '#ff375f', '#8e8e93',
    '#d70015', '#ff7f50', '#c9b700', '#00882b', '#40c8e0', '#0040dd', '#3634a3', '#ac44ce', '#ac8e68', '#636366'
];

const presetIcons = ['📁', '📂', '🗂️', '📚', '📃', '📑', '🔖', '🏷️', '📦', '📥', '💼', '🏠', '🎬', '🎮', '🎵', '💻', '📱', '⭐', '❤️', '🔥'];

// 配置已定义为全局变量,可在其他模块中直接使用
