package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools registers all MCP tools using correct API
func (s *MCPServer) registerTools() {
	// Tool 1: Search bookmarks
	searchTool := mcp.NewTool("search_bookmarks",
		mcp.WithDescription("搜索书签,支持搜索标题、URL、描述和标签"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索关键词"),
		),
	)
	s.mcpServer.AddTool(searchTool, s.handleSearchBookmarks)

	// Tool 2: Get folders
	foldersTool := mcp.NewTool("get_folders",
		mcp.WithDescription("获取所有书签文件夹列表"),
	)
	s.mcpServer.AddTool(foldersTool, s.handleGetFolders)

	// Tool 3: Get tags
	tagsTool := mcp.NewTool("get_tags",
		mcp.WithDescription("获取所有书签标签列表"),
	)
	s.mcpServer.AddTool(tagsTool, s.handleGetTags)

	// Tool 4: Get bookmarks by folder
	byFolderTool := mcp.NewTool("get_bookmarks_by_folder",
		mcp.WithDescription("获取特定文件夹中的所有书签"),
		mcp.WithNumber("folder_id",
			mcp.Required(),
			mcp.Description("文件夹ID"),
		),
	)
	s.mcpServer.AddTool(byFolderTool, s.handleGetBookmarksByFolder)

	// Tool 5: Get bookmarks by tag
	byTagTool := mcp.NewTool("get_bookmarks_by_tag",
		mcp.WithDescription("获取带有特定标签的所有书签"),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("标签名称"),
		),
	)
	s.mcpServer.AddTool(byTagTool, s.handleGetBookmarksByTag)

	// Tool 6: Fetch bookmark content
	fetchContentTool := mcp.NewTool("fetch_bookmark_content",
		mcp.WithDescription("抓取书签页面的实际内容,用于深入阅读和讨论。注意:此操作可能较慢,请谨慎使用。"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("要抓取内容的书签URL"),
		),
	)
	s.mcpServer.AddTool(fetchContentTool, s.handleFetchBookmarkContent)
}

// Tool handlers - using GetString/GetFloat methods from official example

func (s *MCPServer) handleSearchBookmarks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query parameter required"), nil
	}

	filters := map[string]interface{}{"q": query}
	bookmarks, err := s.bookmarkRepo.List(100, 0, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search bookmarks: %v", err)), nil
	}

	result := formatBookmarks(bookmarks, fmt.Sprintf("搜索结果: '%s'", query))
	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleGetFolders(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	folders, err := s.folderRepo.List()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get folders: %v", err)), nil
	}

	result := formatFolders(folders)
	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleGetTags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags, err := s.tagRepo.List()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get tags: %v", err)), nil
	}

	result := formatTags(tags)
	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleGetBookmarksByFolder(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	folderID := request.GetFloat("folder_id", 0)
	if folderID == 0 {
		return mcp.NewToolResultError("folder_id parameter required"), nil
	}

	filters := map[string]interface{}{"folder_id": int(folderID)}
	bookmarks, err := s.bookmarkRepo.List(100, 0, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get bookmarks by folder: %v", err)), nil
	}

	result := formatBookmarks(bookmarks, fmt.Sprintf("文件夹 ID %d", int(folderID)))
	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleGetBookmarksByTag(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := request.GetString("tag", "")
	if tag == "" {
		return mcp.NewToolResultError("tag parameter required"), nil
	}

	filters := map[string]interface{}{"tag": tag}
	bookmarks, err := s.bookmarkRepo.List(100, 0, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get bookmarks by tag: %v", err)), nil
	}

	result := formatBookmarks(bookmarks, fmt.Sprintf("标签: %s", tag))
	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleFetchBookmarkContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.GetString("url", "")
	if url == "" {
		return mcp.NewToolResultError("url parameter required"), nil
	}

	// 使用 ScraperService 抓取頁面元數據
	metadata, err := s.scraperService.ScrapeWebPage(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to fetch content: %v", err)), nil
	}

	// 格式化返回結果
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# %s\n\n", metadata.Title))
	result.WriteString(fmt.Sprintf("**URL**: %s\n\n", url))
	
	if metadata.Description != "" {
		result.WriteString(fmt.Sprintf("**描述**: %s\n\n", metadata.Description))
	}
	
	if metadata.OGTitle != "" && metadata.OGTitle != metadata.Title {
		result.WriteString(fmt.Sprintf("**Open Graph 標題**: %s\n\n", metadata.OGTitle))
	}
	
	if metadata.OGDesc != "" && metadata.OGDesc != metadata.Description {
		result.WriteString(fmt.Sprintf("**Open Graph 描述**: %s\n\n", metadata.OGDesc))
	}

	result.WriteString("\n---\n\n")
	result.WriteString("⚠️ **注意**: 當前只能抓取頁面的元數據(標題、描述等)。\n")
	result.WriteString("如需完整文章內容,請考慮:\n")
	result.WriteString("1. 使用瀏覽器打開 URL 閱讀\n")
	result.WriteString("2. 將文章內容複製粘貼給我\n")
	result.WriteString("3. 未來可以擴展抓取完整文本內容的功能\n")

	return mcp.NewToolResultText(result.String()), nil
}
