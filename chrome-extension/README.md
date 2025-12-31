# Chrome书签管理插件

一个现代化的Chrome浏览器书签管理扩展程序。

## 功能特点

- **80/20布局**: 内容区域占80%，Tab区域占20%，最大化书签展示空间
- **可爱的小熊Logo**: 独特的品牌标识
- **实时搜索**: 快速搜索书签标题、描述和标签
- **现代化卡片设计**: 美观的书签卡片展示
- **多Tab导航**: 全部、未读、归档、设置四个功能区
- **设置面板**: 深色模式、自动同步等个性化设置

## 安装方法

1. 打开Chrome浏览器
2. 访问 `chrome://extensions/`
3. 开启右上角的"开发者模式"
4. 点击"加载已解压的扩展程序"
5. 选择 `chrome-extension` 文件夹

## 项目结构

```
chrome-extension/
├── manifest.json       # 扩展配置文件
├── popup.html         # 弹出窗口HTML
├── popup.js           # 功能逻辑脚本
├── styles.css         # 样式文件
├── icons/             # 图标文件夹
│   ├── icon16.svg
│   ├── icon48.svg
│   └── icon128.svg
└── README.md          # 说明文档
```

## 使用说明

### 主界面
- **搜索框**: 输入关键词实时过滤书签
- **书签卡片**: 点击卡片在新标签页打开链接
- **标签**: 快速识别书签分类

### Features

- **Side Panel Interface**: Modern, clean UI for managing bookmarks
- **Search Functionality**: Quickly find bookmarks by title, description, or tags
- **Tag Support**: Organize bookmarks with tags
- **Theme Toggle**: Switch between light and dark modes
- **API Integration**: Connects to your bookmark service backend
- **Floating Bookmark Button**: Quick-save any webpage with a single click from a floating button

### Floating Bookmark Button

A convenient floating button appears on the right side of every webpage, allowing you to instantly save the current page as a bookmark.

**Features:**
- 🎯 One-click bookmark saving
- 📍 Fixed position on the right side (doesn't interfere with page content)
- 🎨 Beautiful gradient design with smooth animations
- 📱 Responsive design (adapts to mobile screens)
- ✅ Success/error notifications with toast messages
- 🌙 Dark mode support
- 🔄 Loading state during save operation

**How to use:**
1. Browse to any webpage you want to bookmark
2. Click the purple floating button with the "+" icon on the right side
3. Wait for the success notification
4. Open the side panel to view your saved bookmark

### Tab功能
- **全部**: 显示所有书签(默认)
- **未读**: 未读书签管理(预留功能)
- **归档**: 已归档书签(预留功能)
- **设置**: 个性化配置选项

### 设置选项
- **深色模式**: 切换界面主题
- **自动同步**: 自动同步书签数据
- **每页显示数量**: 自定义显示条数

## 技术栈

- HTML5
- CSS3 (现代化设计、Flexbox布局)
- JavaScript (原生ES6+)
- Chrome Extension API

## 开发说明

当前版本使用模拟数据进行演示。后续可以集成Chrome Bookmarks API实现真实书签管理功能。

### 扩展功能建议
- 集成Chrome Bookmarks API
- 实现书签添加/编辑/删除
- 书签导入/导出
- 云端同步
- 标签管理系统
- 未读/归档状态管理

## 版本

v1.0.0 - 初始版本

## 许可

MIT License
