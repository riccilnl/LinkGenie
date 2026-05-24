package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/models"
)

func setupWorkflowServiceTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "workflow-service-test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("关闭测试数据库失败: %v", err)
		}
	})
}

func TestBookmarkCreatedWorkflowRunsOnRealEvent(t *testing.T) {
	setupWorkflowServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	workflowEngine := NewWorkflowEngine(bookmarkRepo, folderRepo)
	bookmarkService := NewBookmarkService(bookmarkRepo, tagRepo, folderRepo, workflowEngine, nil)

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "Inbox"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	_, err = workflowEngine.CreateWorkflow(&models.WorkflowCreate{
		Name:           "created to folder",
		Enabled:        true,
		ConditionLogic: "OR",
		Triggers: []models.WorkflowTriggerCreate{
			{TriggerType: "bookmark_created", Config: map[string]interface{}{}},
		},
		Actions: []models.WorkflowActionCreate{
			{ActionType: "move_to_folder", Config: map[string]interface{}{"folder_id": float64(folder.ID)}},
		},
	})
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}

	created, err := bookmarkService.CreateBookmark(context.Background(), CreateBookmarkInput{
		URL:   "https://example.com/workflow-created",
		Title: "created",
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	folders, err := folderRepo.GetBookmarkFolders(created.ID)
	if err != nil {
		t.Fatalf("查询书签文件夹失败: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != folder.ID {
		t.Fatalf("预期 bookmark_created 工作流把书签移入目标文件夹，实际为 %+v", folders)
	}
}

func TestApplyWorkflowsDoesNotTreatEventTriggersAsAlwaysTrue(t *testing.T) {
	setupWorkflowServiceTestDB(t)

	bookmarkRepo := db.NewBookmarkRepository()
	folderRepo := db.NewFolderRepository(bookmarkRepo)
	workflowEngine := NewWorkflowEngine(bookmarkRepo, folderRepo)

	folder, err := folderRepo.Create(&models.FolderCreate{Name: "ManualApply"})
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	bookmark, err := bookmarkRepo.Create(&models.BookmarkCreate{
		URL:   "https://example.com/manual-apply",
		Title: "manual",
	})
	if err != nil {
		t.Fatalf("创建书签失败: %v", err)
	}

	workflow, err := workflowEngine.CreateWorkflow(&models.WorkflowCreate{
		Name:           "event-only",
		Enabled:        true,
		ConditionLogic: "OR",
		Triggers: []models.WorkflowTriggerCreate{
			{TriggerType: "bookmark_created", Config: map[string]interface{}{}},
		},
		Actions: []models.WorkflowActionCreate{
			{ActionType: "move_to_folder", Config: map[string]interface{}{"folder_id": float64(folder.ID)}},
		},
	})
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}

	if err := workflowEngine.ApplyWorkflowsToBookmarks([]int{workflow.ID}, []int{bookmark.ID}); err != nil {
		t.Fatalf("手动应用工作流失败: %v", err)
	}

	folders, err := folderRepo.GetBookmarkFolders(bookmark.ID)
	if err != nil {
		t.Fatalf("查询书签文件夹失败: %v", err)
	}
	if len(folders) != 0 {
		t.Fatalf("手动应用时不应将事件型触发器当作恒真条件，实际为 %+v", folders)
	}
}
