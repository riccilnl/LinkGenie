package mcp

import (
	"fmt"
	"strings"

	"ai-bookmark-service/db"
	"ai-bookmark-service/models"
	"ai-bookmark-service/services"

	"github.com/mark3labs/mcp-go/server"
)

// MCPServer wraps the MCP server with bookmark service repositories
type MCPServer struct {
	bookmarkRepo   *db.BookmarkRepository
	tagRepo        *db.TagRepository
	folderRepo     *db.FolderRepository
	scraperService *services.ScraperService
	mcpServer      *server.MCPServer
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(
	bookmarkRepo *db.BookmarkRepository,
	tagRepo *db.TagRepository,
	folderRepo *db.FolderRepository,
	scraperService *services.ScraperService,
) *MCPServer {
	s := &MCPServer{
		bookmarkRepo:   bookmarkRepo,
		tagRepo:        tagRepo,
		folderRepo:     folderRepo,
		scraperService: scraperService,
	}

	// Create MCP server with latest API
	s.mcpServer = server.NewMCPServer(
		"ai-bookmark-service",
		"1.0.0",
		server.WithResourceCapabilities(true, false),
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Register tools and resources
	s.registerTools()
	s.registerResources()

	return s
}

// Server returns the underlying MCP server
func (s *MCPServer) Server() *server.MCPServer {
	return s.mcpServer
}

// formatBookmarks formats bookmarks as markdown
func formatBookmarks(bookmarks []*models.Bookmark, title string) string {
	if len(bookmarks) == 0 {
		return fmt.Sprintf("# %s\n\n没有找到书签。", title)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("# %s\n\n", title))
	result.WriteString(fmt.Sprintf("共 %d 个书签\n", len(bookmarks)))

	for _, bookmark := range bookmarks {
		result.WriteString(fmt.Sprintf("\n## %s\n", bookmark.Title))
		result.WriteString(fmt.Sprintf("- **URL**: %s\n", bookmark.URL))
		
		if bookmark.Description != "" {
			result.WriteString(fmt.Sprintf("- **描述**: %s\n", bookmark.Description))
		}
		
		if len(bookmark.TagNames) > 0 {
			result.WriteString(fmt.Sprintf("- **标签**: %s\n", strings.Join(bookmark.TagNames, ", ")))
		}
	}

	return result.String()
}

// formatFolders formats folders as markdown
func formatFolders(folders []*models.Folder) string {
	if len(folders) == 0 {
		return "# 文件夹列表\n\n没有找到文件夹。"
	}

	var result strings.Builder
	result.WriteString("# 文件夹列表\n\n")
	result.WriteString(fmt.Sprintf("共 %d 个文件夹\n", len(folders)))

	for _, folder := range folders {
		result.WriteString(fmt.Sprintf("\n## %s\n", folder.Name))
		result.WriteString(fmt.Sprintf("- **ID**: %d\n", folder.ID))
		result.WriteString(fmt.Sprintf("- **书签数量**: %d\n", folder.Count))
	}

	return result.String()
}

// formatTags formats tags as markdown
func formatTags(tags []*models.Tag) string {
	if len(tags) == 0 {
		return "# 标签列表\n\n没有找到标签。"
	}

	var result strings.Builder
	result.WriteString("# 标签列表\n\n")
	result.WriteString(fmt.Sprintf("共 %d 个标签\n\n", len(tags)))
	
	// Extract tag names
	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}
	result.WriteString(strings.Join(tagNames, ", "))

	return result.String()
}
