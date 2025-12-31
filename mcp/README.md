# MCP 服務器 (Go 實現)

## 概述

本目錄包含使用 Go 實現的 Model Context Protocol (MCP) 服務器,直接嵌入到 AI 書簽服務中。

## 架構

```
mcp/
├── mcp_server.go      # MCP 服務器主結構和格式化函數
├── tools.go           # 工具實現 (6個工具)
├── resources.go       # 資源實現 (3個資源)
└── README.md          # 本文檔
```

## 功能

### 工具 (Tools)

1. **search_bookmarks** - 搜索書簽
   - 參數: `query` (string)
   - 返回: 匹配的書簽列表

2. **get_folders** - 獲取所有文件夾
   - 無參數
   - 返回: 文件夾列表及書簽數量

3. **get_tags** - 獲取所有標簽
   - 無參數
   - 返回: 標簽列表

4. **get_bookmarks_by_folder** - 按文件夾查詢書簽
   - 參數: `folder_id` (number)
   - 返回: 該文件夾中的書簽

5. **get_bookmarks_by_tag** - 按標簽查詢書簽
   - 參數: `tag` (string)
   - 返回: 帶有該標簽的書簽

6. **fetch_bookmark_content** - 抓取書簽內容
   - 參數: `url` (string)
   - 返回: 頁面元數據(標題、描述、Open Graph 數據)

### 資源 (Resources)

1. **bookmarks://all** - 所有書簽
2. **bookmarks://folders** - 文件夾列表
3. **bookmarks://tags** - 標簽列表

## 使用方式

### 在 OpenCat 中配置

```
MCP 服務器 URL: http://your-server-ip:8080/mcp/sse
```

### 測試連接

```bash
# 測試 SSE 端點
curl http://localhost:8080/mcp/sse

# 列出工具
curl -X POST http://localhost:8080/mcp/message?sessionId=test \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'
```

## 技術細節

- **庫**: `github.com/mark3labs/mcp-go` v0.43.2
- **傳輸**: SSE (Server-Sent Events)
- **協議**: JSON-RPC 2.0
- **集成**: 直接嵌入 Go API,共享內存和數據庫連接

## 優勢

- ✅ **單容器部署**: 無需額外的 Python 容器
- ✅ **低資源占用**: ~100MB vs 之前的 ~300MB
- ✅ **高性能**: 無需 HTTP 調用,直接訪問數據庫
- ✅ **類型安全**: Go 的靜態類型檢查
- ✅ **易維護**: 統一的代碼庫

## 開發

### 添加新工具

1. 在 `tools.go` 的 `registerTools()` 中註冊工具
2. 實現處理函數 `handle<ToolName>()`
3. 重新編譯並測試

### 添加新資源

1. 在 `resources.go` 的 `registerResources()` 中註冊資源
2. 實現處理函數 `handle<ResourceName>()`
3. 重新編譯並測試

## 遷移說明

**從 Python MCP 遷移**:
- ❌ 移除 `mcp/*.py` 文件
- ❌ 移除 `mcp/*.bat` 文件
- ❌ 移除 Python requirements 文件
- ✅ 使用新的端點: `/mcp/sse` (而非舊的 `:8081/`)
- ✅ 更新 OpenCat 配置

## 參考資料

- [MCP 官方文檔](https://modelcontextprotocol.io)
- [mcp-go GitHub](https://github.com/mark3labs/mcp-go)
- [MCP Go 教程](https://ganhua.wang/mcp-go)
