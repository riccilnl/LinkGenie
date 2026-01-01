# 🧞 LinkGenie：你的智能书签精灵，效率提升 3 倍！🚀

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Supported-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

### 作为一个“收藏狂魔”，你是否也常被这些问题困扰？
- ❌ **信息黑洞**：随手存的网页没有标题和描述，过两天就忘了它是干嘛的。
- ❌ **整理地狱**：手动打标签太累，书签库最后乱成一团，成了“收藏从未停止，查找从未开始”。
- ❌ **知识断层**：存了一堆干货，但在用 Claude 或 Cursor 协作时，AI 却无法感知你的知识沉淀。

**LinkGenie** 正是为此而生。它不仅是一个书签工具，更是你的**“第二大脑”自动化索引器**。

---

## ✨ 为什么选择它？

- 🔍 **懂整理的“AI 助手”**
  内置**异步 AI 处理机制**。你只管“收藏”，它在后台默默帮你补全标题、概括网页精髓并智能打上标签。不再有无名书签，只有井井有条的知识库。

- ⚡ **工业级的自动化流水线 (Workflow Engine)**
  基于**事件驱动**的自动化系统。支持 6 种事件触发（创建、更新、标题/描述变更、标签变动）及链式逻辑组合。您可以像配置 GitHub Actions 一样，让书签根据规则自动归档、打标或移动文件夹。

- 🧹 **标签资产的“智能洗护” (Tag Optimizer)**
  告别碎片化标签！内置优化引擎支持**同义词智能合并**与**标签晋升机制**。高频标签自动升级，相似标签一键合并，确保您的知识图谱始终精准、纯净。

- 🔋 **极致的极简主义**
  基于 Go 语言极致优化，运行内存占用仅约 **10MB**。无论是放在 NAS、微型主机还是廉价 VPS 上，它都能轻快运行，几乎不占资源。

- 🧠 **面向未来：原生 MCP 支持**
  支持 **Model Context Protocol (MCP)** 协议。这意味着当你使用 Claude、Cursor 等 AI 助手时，它们可以直接检索你保存的书签作为上下文，实现真正的“知识库级”AI 协作。

- 📂 **视觉化文件夹与无痛迁移**
  支持 **Emoji 图标**与**色彩编码**的文件夹系统。深度兼容 Netscape HTML 标准导入/导出，从 Chrome 或 Linkding 搬家只需一秒。

---

## 🚀 快速开始

本项目**仅支持容器化部署**，以确保运行环境的一致性与稳定性。

### 1. 使用 Docker Compose 部署

1. **创建配置文件**：在目录中创建 `docker-compose.yml`：
```yaml
version: '3.8'
services:
  bookmarks:
    image: riccilnl/linkgenie:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    env_file: .env
    restart: always
```
2. **配置环境变量**：创建 `.env` 文件并填入您的配置（参考下方“环境变量配置”章节）。
3. **启动服务**：`docker-compose up -d`

---

## 🧩 Chrome 扩展安装 (手动安装指南)

由于扩展程序尚未发布至应用商店，请按照以下步骤手动安装：

1. **下载源码**：克隆本项目到本地。
2. **打开扩展页面**：在 Chrome 地址栏访问 `chrome://extensions/`。
3. **开启开发者模式**：点击右上角的“**开发者模式**”开关。
4. **加载插件**：点击左上角的“**加载已解压的扩展程序**”。
5. **选择文件夹**：在文件选择器中选中本项目根目录下的 `chrome-extension` 文件夹。
6. **固定插件**：点击浏览器右上角的拼图图标，将 **LinkGenie** 固定到工具栏。

*提示：安装后在设置中填入您的后端 API 地址和 Token 即可开始使用！*

---

## 🔧 环境变量配置

| 变量名 | 说明 | 默认值 |
| :--- | :--- | :--- |
| `API_TOKEN` | 用于客户端认证的 Token | `your-secret-token-here` |
| `AI_ENABLED` | 是否启用 AI 增强功能 | `true` |
| `ENABLE_ASYNC_AI` | 是否开启异步 AI 处理 (推荐) | `true` |
| `AI_API_KEY` | OpenAI 兼容接口的 API Key | - |
| `AI_ENDPOINT` | AI 接口地址 | `https://api.openai.com/v1/...` |
| `AI_MODEL` | 使用的 AI 模型名称 | `gpt-3.5-turbo` |
| `DATABASE_URL` | SQLite 数据库路径 | `./data/bookmarks.db` |

---

## 📡 API 列表预览

所有 API 需在 Header 中携带 `Authorization: Token YOUR_TOKEN`。

*   `POST /api/bookmarks` - 创建新书签（触发 AI 异步增强及工作流）
*   `POST /api/tags/optimize` - 触发全局标签清洗与规范化
*   `POST /api/workflows/apply` - 对存量书签手动应用工作流规则
*   `GET /mcp/` - MCP 协议交互端点

---

## 🤝 兼容性与客户端

本服务旨在成为 Linkding 的轻量化、增强版替代方案。
- **支持客户端**：[Linkdy (iOS/Android)](https://github.com/JGeek00/linkdy), [Linkding Web Extension](https://github.com/sissbruecker/linkding-extension).
- **导出支持**：完全支持 Netscape HTML 标准格式导出。

---

## 📜 版本更新历史

### v1.1.0 (2026-01-01)
- **✨ 重大特性：新增 Chrome 扩展引导流程**
  - 为新用户提供一键引导界面，支持快速配置后端服务器地址、认证 Token 及 AI 参数。
  - 支持配置折叠与状态即时反馈，极大降低上手门槛。
- **⚡ 兼容性飞跃：深度适配 Linkdy (iOS/Flutter)**
  - 支持 `multipart/form-data` 请求解析，完美解决 Linkdy 与原生 Linkding API 的协议细微差异。
  - 智能识别标签格式，支持逗号分隔字符串与 JSON 数组自动转换。
  - 自动处理 Linkdy 的 `is_archived` 归档标志，并自动映射为系统的 `is_favorite` 收藏状态。
- **🛡️ 安全增强：加固认证中间件**
  - 修复了一个可能导致根路径认证绕过的边界漏洞。
  - 强化了对 `Authorization: Token` 与 `Bearer` 头的规范化解析。
- **🧪 质量保障：全量回归测试**
  - 补充了针对 Linkdy 数据结构、畸形 JSON、并发压力及幸存者自愈（Panic Recovery）的全量集成测试用例。
- **🧹 代码优化**：清理了已废弃的标签校验逻辑和模型冗余字段，保持后端极致轻量。

---

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 协议开源。

**⭐ 如果这个项目帮到了您，请给一个 Star 以示支持！**
