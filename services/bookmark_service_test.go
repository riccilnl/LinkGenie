package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/models"
)

func setupBookmarkServiceTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "bookmark-service-test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("关闭测试数据库失败: %v", err)
		}
	})
}

func TestCreateBookmarkRejectsDuplicateURL(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)

	submitCount := 0
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, func(int) error {
		submitCount++
		return nil
	})

	ctx := context.Background()

	created, err := service.CreateBookmark(ctx, CreateBookmarkInput{
		URL:         "example.com/dup",
		Title:       "first",
		Description: "desc",
		TagNames:    []string{"go"},
	})
	if err != nil {
		t.Fatalf("第一次创建失败: %v", err)
	}

	if submitCount != 1 {
		t.Fatalf("预期创建后投递 1 次增强任务，实际为 %d", submitCount)
	}

	_, err = service.CreateBookmark(ctx, CreateBookmarkInput{
		URL:       "https://example.com/dup",
		Title:     "second",
		TagNames:  []string{"rust"},
		FolderIDs: []int{1},
	})
	if !errors.Is(err, ErrBookmarkAlreadyExists) {
		t.Fatalf("预期重复 URL 返回 ErrBookmarkAlreadyExists，实际为: %v", err)
	}

	stored, err := bookmarkRepo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("重新获取书签失败: %v", err)
	}

	if stored.Title != "first" {
		t.Fatalf("重复创建不应覆盖原标题，实际为 %s", stored.Title)
	}
}

func TestPatchBookmarkPreservesUnspecifiedFields(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)

	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)
	ctx := context.Background()

	created, err := service.CreateBookmark(ctx, CreateBookmarkInput{
		URL:         "https://example.com/patch",
		Title:       "before",
		Description: "original description",
		Notes:       "notes",
		TagNames:    []string{"go", "backend"},
		Shared:      true,
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	newTitle := "after"
	patched, err := service.PatchBookmark(ctx, created.ID, PatchBookmarkInput{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("PATCH 失败: %v", err)
	}

	if patched.Title != "after" {
		t.Fatalf("预期标题更新为 after，实际为 %s", patched.Title)
	}
	if patched.Description != "original description" {
		t.Fatalf("未提交字段 description 不应被覆盖，实际为 %s", patched.Description)
	}
	if patched.Notes != "notes" {
		t.Fatalf("未提交字段 notes 不应被覆盖，实际为 %s", patched.Notes)
	}
	if !patched.Shared {
		t.Fatalf("未提交字段 shared 不应被覆盖")
	}
	if len(patched.TagNames) != 2 || patched.TagNames[0] != "go" || patched.TagNames[1] != "backend" {
		t.Fatalf("未提交字段 tag_names 不应被覆盖，实际为 %+v", patched.TagNames)
	}
}

func TestUpdateBookmarkReplacesFields(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)

	ctx := context.Background()

	created, err := service.CreateBookmark(ctx, CreateBookmarkInput{
		URL:         "https://example.com/replace",
		Title:       "before",
		Description: "desc",
		Notes:       "notes",
		TagNames:    []string{"old"},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	updated, err := service.UpdateBookmark(ctx, created.ID, ReplaceBookmarkInput{
		URL:         "https://example.com/replace",
		Title:       "after",
		Description: "",
		Notes:       "",
		TagNames:    []string{"new"},
	})
	if err != nil {
		t.Fatalf("PUT 失败: %v", err)
	}

	if updated.Title != "after" {
		t.Fatalf("预期标题更新为 after，实际为 %s", updated.Title)
	}
	if updated.Description != "" || updated.Notes != "" {
		t.Fatalf("PUT 应执行全量替换，实际 description=%q notes=%q", updated.Description, updated.Notes)
	}
	if len(updated.TagNames) != 1 || updated.TagNames[0] != "new" {
		t.Fatalf("PUT 应替换标签，实际为 %+v", updated.TagNames)
	}
}

func TestUpdateBookmarkCanClearFolders(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "ToClear"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	created, err := service.CreateBookmark(context.Background(), CreateBookmarkInput{
		URL:       "https://example.com/replace-folders",
		Title:     "replace folders",
		FolderIDs: []int{folder.ID},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	_, err = service.UpdateBookmark(context.Background(), created.ID, ReplaceBookmarkInput{
		URL:   created.URL,
		Title: "replace folders",
	})
	if err != nil {
		t.Fatalf("PUT 清空文件夹失败: %v", err)
	}

	folders, err := folderRepo.GetBookmarkFolders(created.ID)
	if err != nil {
		t.Fatalf("查询书签文件夹失败: %v", err)
	}
	if len(folders) != 0 {
		t.Fatalf("PUT 传空文件夹时应清空关联，实际为 %+v", folders)
	}
}

func TestPatchBookmarkCanClearFoldersWhenFieldProvided(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "PatchClear"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	created, err := service.CreateBookmark(context.Background(), CreateBookmarkInput{
		URL:       "https://example.com/patch-folders",
		Title:     "patch folders",
		FolderIDs: []int{folder.ID},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	folderIDs := []int{}
	_, err = service.PatchBookmark(context.Background(), created.ID, PatchBookmarkInput{
		FolderIDs: &folderIDs,
	})
	if err != nil {
		t.Fatalf("PATCH 清空文件夹失败: %v", err)
	}

	folders, err := folderRepo.GetBookmarkFolders(created.ID)
	if err != nil {
		t.Fatalf("查询书签文件夹失败: %v", err)
	}
	if len(folders) != 0 {
		t.Fatalf("PATCH 显式传空文件夹时应清空关联，实际为 %+v", folders)
	}
}

func TestEnqueueEnhancementRejectsMissingBookmark(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)

	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)
	err := service.EnqueueEnhancement(context.Background(), 999)
	if !errors.Is(err, ErrBookmarkNotFound) {
		t.Fatalf("预期不存在时返回 ErrBookmarkNotFound，实际为 %v", err)
	}
}

func TestCreateBookmarkCanReplaceFolders(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "Tech"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	created, err := service.CreateBookmark(context.Background(), CreateBookmarkInput{
		URL:       "https://example.com/folder",
		Title:     "folder",
		FolderIDs: []int{folder.ID},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	folders, err := folderRepo.GetBookmarkFolders(created.ID)
	if err != nil {
		t.Fatalf("查询书签文件夹失败: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != folder.ID {
		t.Fatalf("预期书签归属到指定文件夹，实际为 %+v", folders)
	}
}

func TestTagUsageCountsAreRecalculatedOnWritePaths(t *testing.T) {
	setupBookmarkServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	service := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, nil, nil)

	created, err := service.CreateBookmark(context.Background(), CreateBookmarkInput{
		URL:      "https://example.com/tag-stats",
		Title:    "stats",
		TagNames: []string{"go", "backend"},
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	tags, err := tagRepo.List()
	if err != nil {
		t.Fatalf("查询标签失败: %v", err)
	}

	assertTagUsageCount(t, tags, "go", 1)
	assertTagUsageCount(t, tags, "backend", 1)

	newTags := []string{"go"}
	_, err = service.PatchBookmark(context.Background(), created.ID, PatchBookmarkInput{
		TagNames: &newTags,
	})
	if err != nil {
		t.Fatalf("更新标签失败: %v", err)
	}

	tags, err = tagRepo.List()
	if err != nil {
		t.Fatalf("查询更新后标签失败: %v", err)
	}

	assertTagUsageCount(t, tags, "go", 1)
	assertTagUsageCount(t, tags, "backend", 0)

	if err := service.DeleteBookmark(context.Background(), created.ID); err != nil {
		t.Fatalf("删除书签失败: %v", err)
	}

	tags, err = tagRepo.List()
	if err != nil {
		t.Fatalf("查询删除后标签失败: %v", err)
	}

	assertTagUsageCount(t, tags, "go", 0)
	assertTagUsageCount(t, tags, "backend", 0)
}

func assertTagUsageCount(t *testing.T, tags []*models.Tag, name string, expected int) {
	t.Helper()

	for _, tag := range tags {
		if tag.Name == name {
			if tag.UsageCount != expected {
				t.Fatalf("标签 %s 的 usage_count 预期为 %d，实际为 %d", name, expected, tag.UsageCount)
			}
			return
		}
	}

	t.Fatalf("未找到标签 %s", name)
}
