package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerResources registers all MCP resources
func (s *MCPServer) registerResources() {
	// Resource 1: All bookmarks
	allResource := mcp.NewResource("bookmarks://all",
		"所有书签",
		mcp.WithMIMEType("text/markdown"),
		mcp.WithResourceDescription("获取所有书签列表"),
	)
	s.mcpServer.AddResource(allResource, s.handleAllBookmarks)

	// Resource 2: Folders
	foldersResource := mcp.NewResource("bookmarks://folders",
		"文件夹列表",
		mcp.WithMIMEType("text/markdown"),
		mcp.WithResourceDescription("所有书签文件夹"),
	)
	s.mcpServer.AddResource(foldersResource, s.handleFoldersResource)

	// Resource 3: Tags
	tagsResource := mcp.NewResource("bookmarks://tags",
		"标签列表",
		mcp.WithMIMEType("text/markdown"),
		mcp.WithResourceDescription("所有书签标签"),
	)
	s.mcpServer.AddResource(tagsResource, s.handleTagsResource)
}

// Resource handlers - correct signature from official example

func (s *MCPServer) handleAllBookmarks(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	bookmarks, err := s.bookmarkRepo.List(100, 0, nil)
	if err != nil {
		return nil, err
	}

	result := formatBookmarks(bookmarks, "所有书签")
	
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "bookmarks://all",
			MIMEType: "text/markdown",
			Text:     result,
		},
	}, nil
}

func (s *MCPServer) handleFoldersResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	folders, err := s.folderRepo.List()
	if err != nil {
		return nil, err
	}

	result := formatFolders(folders)
	
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "bookmarks://folders",
			MIMEType: "text/markdown",
			Text:     result,
		},
	}, nil
}

func (s *MCPServer) handleTagsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	tags, err := s.tagRepo.List()
	if err != nil {
		return nil, err
	}

	result := formatTags(tags)
	
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "bookmarks://tags",
			MIMEType: "text/markdown",
			Text:     result,
		},
	}, nil
}
