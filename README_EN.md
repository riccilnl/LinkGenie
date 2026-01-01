# 🧞 LinkGenie: Your AI-Powered Bookmark Elf, Boost Productivity by 3x! 🚀

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Supported-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

[中文说明](./README.md) | **English**

### Are you struggling with these "Bookmark Hoarding" issues?
- ❌ **Information Black Hole**: Bookmarks saved without titles or descriptions are forgotten within days.
- ❌ **Categorization Hell**: Manual tagging is tedious, and your library eventually becomes a mess.
- ❌ **Knowledge Gap**: You save great content, but your AI tools (like Claude or Cursor) have no way of accessing your curated knowledge.

**LinkGenie** is built to solve this. It's not just a bookmark manager; it's an **Automated Indexer for your Second Brain**.

---

## ✨ Why LinkGenie?

- 🔍 **AI-Native Organization**
  Built-in **Asynchronous AI Processing**. You just "save," and LinkGenie works in the background to fetch titles, summarize content, and suggest smart tags. No more nameless bookmarks; only structured knowledge.

- ⚡ **Industrial-Grade Workflow Engine**
  An **event-driven** automation system. Supports various triggers (create, update, metadata changes) and chained logic. You can automate archiving, tagging, or moving folders just like configuring GitHub Actions.

- 🧹 **Intelligent Tag Optimizer**
  Say goodbye to fragmented tags. The optimization engine supports **synonym merging** and **tag promotion**. Frequent tags are auto-promoted, and similar ones are unified.

- 🔋 **Extreme Minimalism**
  Optimized with Go, consuming only ~ **10MB** of RAM. Perfect for NAS, micro-servers, or budget VPS.

- 🧠 **Future-Ready: Native MCP Support**
  Fully supports the **Model Context Protocol (MCP)**. Your AI assistants (Claude, Cursor, etc.) can directly retrieve your bookmarks as context for a true "knowledge-base" experience.

- 📂 **Visual Folders & Seamless Migration**
  Visual folder system with **Emoji support** and color coding. Fully compatible with Netscape HTML standards for 1-second migration from Chrome or Linkding.

---

## 🚀 Quick Start

This project is optimized for **Containerized Deployment**.

### 1. Deploy with Docker Compose

1. **Create `docker-compose.yml`**:
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
2. **Configure `.env`**: Create a `.env` file based on the settings below.
3. **Run**: `docker-compose up -d`

---

## 🧩 Chrome Extension (Manual Installation)

1. **Download Source**: Clone this repository.
2. **Open Extensions**: Go to `chrome://extensions/` in Chrome.
3. **Enable Developer Mode**: Toggle the switch in the top right.
4. **Load Unpacked**: Click "Load unpacked" and select the `chrome-extension` folder.
5. **Pin**: Pin LinkGenie to your toolbar for quick access.

---

## 🔧 Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `API_TOKEN` | Token for client authentication | `your-secret-token-here` |
| `AI_ENABLED` | Enable AI features | `true` |
| `ENABLE_ASYNC_AI` | Enable async AI processing | `true` |
| `AI_API_KEY` | API Key for OpenAI compatible interface | - |
| `AI_ENDPOINT` | AI API Endpoint | `https://api.openai.com/v1/...` |
| `AI_MODEL` | AI Model name | `gpt-3.5-turbo` |
| `DATABASE_URL` | SQLite database path | `./data/bookmarks.db` |

---

## 📡 API Overview

All requests require `Authorization: Token YOUR_TOKEN`.

* `POST /api/bookmarks` - Create bookmark (Triggers AI & Workflows)
* `POST /api/tags/optimize` - Trigger tag optimization
* `GET /mcp/` - MCP Protocol endpoint

---

## 🤝 Compatibility & Clients

LinkGenie is designed to be a lightweight, enhanced alternative to Linkding.
- **Supported Clients**: [Linkdy (iOS/Android)](https://github.com/JGeek00/linkdy), [Linkding Web Extension](https://github.com/sissbruecker/linkding-extension).
- **Export**: Full Netscape HTML standard support.

---

## 📜 Release History

### v1.1.0 (2026-01-01)
- **✨ New: Extension Onboarding Flow**
  - Interactive setup for API Base, Token, and AI parameters.
- **⚡ Enhanced Compatibility: Linkdy (iOS/Flutter)**
  - Full support for `multipart/form-data`.
  - Smart tag format recognition (CSV/JSON).
  - Auto-mapping of `is_archived` to `is_favorite`.
- **🛡️ Security: Hardened Auth Middleware**
  - Fixed root path bypass vulnerability.
  - Standardized `Token`/`Bearer` parsing.
- **🧪 Quality: Full Regression Suite**
  - Added integration tests for high-concurrency and recovery scenarios.

---

## 📄 License

MIT License.

**⭐ If this project helps you, please give it a Star!**
