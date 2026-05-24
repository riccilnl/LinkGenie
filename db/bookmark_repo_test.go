package db

import (
	"path/filepath"
	"testing"

	"github.com/riccilnl/LinkGenie/models"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	t.Cleanup(func() {
		if err := Close(); err != nil {
			t.Fatalf("关闭测试数据库失败: %v", err)
		}
	})
}

func TestInitEnablesForeignKeys(t *testing.T) {
	setupTestDB(t)

	var enabled int
	if err := DB.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("查询 foreign_keys 失败: %v", err)
	}

	if enabled != 1 {
		t.Fatalf("foreign_keys 未启用，实际值: %d", enabled)
	}
}

func TestDeleteBookmarkCascadesRelations(t *testing.T) {
	setupTestDB(t)

	bookmarkRepo := NewBookmarkRepository()
	folderRepo := NewFolderRepository(bookmarkRepo)

	bookmark, err := bookmarkRepo.Create(&models.BookmarkCreate{
		URL:      "https://example.com/go",
		Title:    "Go",
		TagNames: []string{"go"},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "Tech"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	if err := folderRepo.AddBookmark(bookmark.ID, folder.ID); err != nil {
		t.Fatalf("关联文件夹失败: %v", err)
	}

	var tagCountBefore int
	if err := DB.QueryRow("SELECT COUNT(*) FROM bookmark_tags WHERE bookmark_id = ?", bookmark.ID).Scan(&tagCountBefore); err != nil {
		t.Fatalf("查询 bookmark_tags 失败: %v", err)
	}
	if tagCountBefore != 1 {
		t.Fatalf("预期 bookmark_tags 为 1，实际为 %d", tagCountBefore)
	}

	var folderCountBefore int
	if err := DB.QueryRow("SELECT COUNT(*) FROM bookmark_folders WHERE bookmark_id = ?", bookmark.ID).Scan(&folderCountBefore); err != nil {
		t.Fatalf("查询 bookmark_folders 失败: %v", err)
	}
	if folderCountBefore != 1 {
		t.Fatalf("预期 bookmark_folders 为 1，实际为 %d", folderCountBefore)
	}

	if err := bookmarkRepo.Delete(bookmark.ID); err != nil {
		t.Fatalf("删除书签失败: %v", err)
	}

	var tagCountAfter int
	if err := DB.QueryRow("SELECT COUNT(*) FROM bookmark_tags WHERE bookmark_id = ?", bookmark.ID).Scan(&tagCountAfter); err != nil {
		t.Fatalf("查询删除后 bookmark_tags 失败: %v", err)
	}
	if tagCountAfter != 0 {
		t.Fatalf("预期删除后 bookmark_tags 为 0，实际为 %d", tagCountAfter)
	}

	var folderCountAfter int
	if err := DB.QueryRow("SELECT COUNT(*) FROM bookmark_folders WHERE bookmark_id = ?", bookmark.ID).Scan(&folderCountAfter); err != nil {
		t.Fatalf("查询删除后 bookmark_folders 失败: %v", err)
	}
	if folderCountAfter != 0 {
		t.Fatalf("预期删除后 bookmark_folders 为 0，实际为 %d", folderCountAfter)
	}
}

func TestListAndCountRespectFolderAndTagFilters(t *testing.T) {
	setupTestDB(t)

	bookmarkRepo := NewBookmarkRepository()
	folderRepo := NewFolderRepository(bookmarkRepo)

	goBookmark, err := bookmarkRepo.Create(&models.BookmarkCreate{
		URL:      "https://example.com/go",
		Title:    "Go",
		TagNames: []string{"go"},
		Unread:   true,
		Shared:   true,
	})
	if err != nil {
		t.Fatalf("创建 go 书签失败: %v", err)
	}

	rustBookmark, err := bookmarkRepo.Create(&models.BookmarkCreate{
		URL:      "https://example.com/rust",
		Title:    "Rust",
		TagNames: []string{"rust"},
	})
	if err != nil {
		t.Fatalf("创建 rust 书签失败: %v", err)
	}

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "Language"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	if err := folderRepo.AddBookmark(goBookmark.ID, folder.ID); err != nil {
		t.Fatalf("关联 go 书签到文件夹失败: %v", err)
	}

	filteredByFolder, err := bookmarkRepo.List(10, 0, map[string]interface{}{"folder_id": folder.ID})
	if err != nil {
		t.Fatalf("按文件夹查询失败: %v", err)
	}
	if len(filteredByFolder) != 1 || filteredByFolder[0].ID != goBookmark.ID {
		t.Fatalf("按文件夹过滤结果不正确: %+v", filteredByFolder)
	}

	filteredByTag, err := bookmarkRepo.List(10, 0, map[string]interface{}{"tag": "rust"})
	if err != nil {
		t.Fatalf("按标签查询失败: %v", err)
	}
	if len(filteredByTag) != 1 || filteredByTag[0].ID != rustBookmark.ID {
		t.Fatalf("按标签过滤结果不正确: %+v", filteredByTag)
	}

	count, err := bookmarkRepo.Count(map[string]interface{}{"unread": true, "shared": true})
	if err != nil {
		t.Fatalf("按 unread/shared 计数失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("预期 unread/shared 计数为 1，实际为 %d", count)
	}
}
