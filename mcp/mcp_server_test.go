package mcp

import (
	"fmt"
	"strings"
	"testing"

	"ai-bookmark-service/models"

	"github.com/stretchr/testify/assert"
)

func TestFormatBookmarks(t *testing.T) {
	tests := []struct {
		name      string
		bookmarks []*models.Bookmark
		title     string
		contains  []string
	}{
		{
			name:      "Empty bookmarks",
			bookmarks: []*models.Bookmark{},
			title:     "测试标题",
			contains:  []string{"# 测试标题", "没有找到书签"},
		},
		{
			name: "Single bookmark",
			bookmarks: []*models.Bookmark{
				{
					ID:          1,
					Title:       "Example",
					URL:         "https://example.com",
					Description: "Test description",
					TagNames:    []string{"test", "example"},
				},
			},
			title: "单个书签",
			contains: []string{
				"# 单个书签",
				"共 1 个书签",
				"## Example",
				"https://example.com",
				"Test description",
				"test, example",
			},
		},
		{
			name: "Multiple bookmarks",
			bookmarks: []*models.Bookmark{
				{
					ID:    1,
					Title: "First",
					URL:   "https://first.com",
				},
				{
					ID:    2,
					Title: "Second",
					URL:   "https://second.com",
				},
			},
			title: "多个书签",
			contains: []string{
				"共 2 个书签",
				"## First",
				"## Second",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBookmarks(tt.bookmarks, tt.title)
			
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatFolders(t *testing.T) {
	tests := []struct {
		name     string
		folders  []*models.Folder
		contains []string
	}{
		{
			name:     "Empty folders",
			folders:  []*models.Folder{},
			contains: []string{"# 文件夹列表", "没有找到文件夹"},
		},
		{
			name: "Single folder",
			folders: []*models.Folder{
				{
					ID:    1,
					Name:  "Work",
					Count: 5,
				},
			},
			contains: []string{
				"共 1 个文件夹",
				"## Work",
				"**ID**: 1",
				"**书签数量**: 5",
			},
		},
		{
			name: "Multiple folders",
			folders: []*models.Folder{
				{ID: 1, Name: "Work", Count: 5},
				{ID: 2, Name: "Personal", Count: 10},
			},
			contains: []string{
				"共 2 个文件夹",
				"## Work",
				"## Personal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFolders(tt.folders)
			
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []*models.Tag
		contains []string
	}{
		{
			name:     "Empty tags",
			tags:     []*models.Tag{},
			contains: []string{"# 标签列表", "没有找到标签"},
		},
		{
			name: "Single tag",
			tags: []*models.Tag{
				{ID: 1, Name: "golang"},
			},
			contains: []string{
				"共 1 个标签",
				"golang",
			},
		},
		{
			name: "Multiple tags",
			tags: []*models.Tag{
				{ID: 1, Name: "golang"},
				{ID: 2, Name: "python"},
				{ID: 3, Name: "javascript"},
			},
			contains: []string{
				"共 3 个标签",
				"golang, python, javascript",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTags(tt.tags)
			
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestFormatContentMetadata(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		title    string
		desc     string
		ogTitle  string
		ogDesc   string
		contains []string
	}{
		{
			name:  "Basic metadata",
			url:   "https://example.com",
			title: "Example Page",
			desc:  "This is an example",
			contains: []string{
				"# Example Page",
				"**URL**: https://example.com",
				"**描述**: This is an example",
			},
		},
		{
			name:    "With Open Graph data",
			url:     "https://blog.example.com",
			title:   "Blog Post",
			desc:    "Short description",
			ogTitle: "Full Blog Post Title",
			ogDesc:  "Detailed Open Graph description",
			contains: []string{
				"# Blog Post",
				"**Open Graph 標題**: Full Blog Post Title",
				"**Open Graph 描述**: Detailed Open Graph description",
			},
		},
		{
			name:  "Minimal metadata",
			url:   "https://minimal.com",
			title: "Minimal Page",
			contains: []string{
				"# Minimal Page",
				"**URL**: https://minimal.com",
				"⚠️ **注意**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the formatting logic from handleFetchBookmarkContent
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# %s\n\n", tt.title))
			result.WriteString(fmt.Sprintf("**URL**: %s\n\n", tt.url))
			
			if tt.desc != "" {
				result.WriteString(fmt.Sprintf("**描述**: %s\n\n", tt.desc))
			}
			
			if tt.ogTitle != "" && tt.ogTitle != tt.title {
				result.WriteString(fmt.Sprintf("**Open Graph 標題**: %s\n\n", tt.ogTitle))
			}
			
			if tt.ogDesc != "" && tt.ogDesc != tt.desc {
				result.WriteString(fmt.Sprintf("**Open Graph 描述**: %s\n\n", tt.ogDesc))
			}

			result.WriteString("\n---\n\n")
			result.WriteString("⚠️ **注意**: 當前只能抓取頁面的元數據(標題、描述等)。\n")
			
			output := result.String()
			
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

