package services

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	"ai-bookmark-service/db"
	"ai-bookmark-service/models"
)

// WorkflowEngine handles workflow operations
type WorkflowEngine struct {
	bookmarkRepo *db.BookmarkRepository
	folderRepo   *db.FolderRepository
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(bookmarkRepo *db.BookmarkRepository, folderRepo *db.FolderRepository) *WorkflowEngine {
	return &WorkflowEngine{
		bookmarkRepo: bookmarkRepo,
		folderRepo:   folderRepo,
	}
}

// ============ 工作流数据库操作 ============

// CreateWorkflow creates a new workflow
func (e *WorkflowEngine) CreateWorkflow(wc *models.WorkflowCreate) (*models.Workflow, error) {
	// 获取当前最大priority
	var maxPriority int
	db.DB.QueryRow("SELECT COALESCE(MAX(priority), 0) FROM workflows").Scan(&maxPriority)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	enabled := 1
	if !wc.Enabled {
		enabled = 0
	}

	conditionLogic := wc.ConditionLogic
	if conditionLogic == "" {
		conditionLogic = "OR"
	}

	// 插入工作流
	result, err := db.DB.Exec(
		"INSERT INTO workflows (name, description, enabled, priority, condition_logic, date_added, date_modified) VALUES (?, ?, ?, ?, ?, ?, ?)",
		wc.Name, wc.Description, enabled, maxPriority+1, conditionLogic, now, now,
	)
	if err != nil {
		return nil, err
	}

	workflowID, _ := result.LastInsertId()

	// 插入触发条件
	for _, trigger := range wc.Triggers {
		configJSON, _ := json.Marshal(trigger.Config)
		db.DB.Exec(
			"INSERT INTO workflow_triggers (workflow_id, trigger_type, config) VALUES (?, ?, ?)",
			workflowID, trigger.TriggerType, string(configJSON),
		)
	}

	// 插入执行动作
	for _, action := range wc.Actions {
		configJSON, _ := json.Marshal(action.Config)
		db.DB.Exec(
			"INSERT INTO workflow_actions (workflow_id, action_type, config) VALUES (?, ?, ?)",
			workflowID, action.ActionType, string(configJSON),
		)
	}

	return e.GetWorkflow(int(workflowID))
}

// GetWorkflow retrieves a workflow by ID
func (e *WorkflowEngine) GetWorkflow(id int) (*models.Workflow, error) {
	var workflow models.Workflow
	var enabled int

	err := db.DB.QueryRow(
		"SELECT id, name, description, enabled, priority, condition_logic, date_added, date_modified FROM workflows WHERE id = ?",
		id,
	).Scan(&workflow.ID, &workflow.Name, &workflow.Description, &enabled, &workflow.Priority, &workflow.ConditionLogic, &workflow.DateAdded, &workflow.DateModified)

	if err != nil {
		return nil, err
	}

	workflow.Enabled = enabled == 1

	// 获取触发条件
	rows, err := db.DB.Query("SELECT id, trigger_type, config FROM workflow_triggers WHERE workflow_id = ?", id)
	if err != nil {
		log.Printf("⚠️ 查询工作流触发器失败: %v", err)
		workflow.Triggers = []models.WorkflowTrigger{}
	} else {
		defer rows.Close()

		for rows.Next() {
			var trigger models.WorkflowTrigger
			var configJSON string
			if err := rows.Scan(&trigger.ID, &trigger.TriggerType, &configJSON); err != nil {
				log.Printf("⚠️ 扫描触发器失败: %v", err)
				continue
			}
			trigger.WorkflowID = id
			if err := json.Unmarshal([]byte(configJSON), &trigger.Config); err != nil {
				log.Printf("⚠️ 解析触发器配置失败: %v", err)
				continue
			}
			workflow.Triggers = append(workflow.Triggers, trigger)
		}
	}

	// 获取执行动作
	rows2, err := db.DB.Query("SELECT id, action_type, config FROM workflow_actions WHERE workflow_id = ?", id)
	if err != nil {
		log.Printf("⚠️ 查询工作流动作失败: %v", err)
		workflow.Actions = []models.WorkflowAction{}
	} else {
		defer rows2.Close()

		for rows2.Next() {
			var action models.WorkflowAction
			var configJSON string
			if err := rows2.Scan(&action.ID, &action.ActionType, &configJSON); err != nil {
				log.Printf("⚠️ 扫描动作失败: %v", err)
				continue
			}
			action.WorkflowID = id
			if err := json.Unmarshal([]byte(configJSON), &action.Config); err != nil {
				log.Printf("⚠️ 解析动作配置失败: %v", err)
				continue
			}
			workflow.Actions = append(workflow.Actions, action)
		}
	}

	// 计算匹配数量（简化版，实际应该评估所有书签）
	workflow.MatchCount = 0

	return &workflow, nil
}

// ListWorkflows retrieves all workflows
func (e *WorkflowEngine) ListWorkflows() ([]*models.Workflow, error) {
	rows, err := db.DB.Query("SELECT id FROM workflows ORDER BY priority ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workflows := []*models.Workflow{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		if workflow, err := e.GetWorkflow(id); err == nil {
			workflows = append(workflows, workflow)
		}
	}

	return workflows, nil
}

// UpdateWorkflow updates a workflow
func (e *WorkflowEngine) UpdateWorkflow(id int, wc *models.WorkflowCreate) (*models.Workflow, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	enabled := 1
	if !wc.Enabled {
		enabled = 0
	}

	// 更新基本信息
	var err error
	_, err = db.DB.Exec(
		"UPDATE workflows SET name = ?, description = ?, enabled = ?, condition_logic = ?, date_modified = ? WHERE id = ?",
		wc.Name, wc.Description, enabled, wc.ConditionLogic, now, id,
	)
	if err != nil {
		return nil, err
	}

	// 删除旧的触发条件和动作
	db.DB.Exec("DELETE FROM workflow_triggers WHERE workflow_id = ?", id)
	db.DB.Exec("DELETE FROM workflow_actions WHERE workflow_id = ?", id)

	// 插入新的触发条件
	for _, trigger := range wc.Triggers {
		configJSON, _ := json.Marshal(trigger.Config)
		db.DB.Exec(
			"INSERT INTO workflow_triggers (workflow_id, trigger_type, config) VALUES (?, ?, ?)",
			id, trigger.TriggerType, string(configJSON),
		)
	}

	// 插入新的执行动作
	for _, action := range wc.Actions {
		configJSON, _ := json.Marshal(action.Config)
		db.DB.Exec(
			"INSERT INTO workflow_actions (workflow_id, action_type, config) VALUES (?, ?, ?)",
			id, action.ActionType, string(configJSON),
		)
	}

	return e.GetWorkflow(id)
}

// DeleteWorkflow deletes a workflow
func (e *WorkflowEngine) DeleteWorkflow(id int) error {
	_, err := db.DB.Exec("DELETE FROM workflows WHERE id = ?", id)
	return err
}

// ToggleWorkflow toggles workflow enabled status
func (e *WorkflowEngine) ToggleWorkflow(id int) (*models.Workflow, error) {
	// 使用原子操作避免竞态条件
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.DB.Exec(
		"UPDATE workflows SET enabled = CASE WHEN enabled = 1 THEN 0 ELSE 1 END, date_modified = ? WHERE id = ?",
		now, id,
	)
	if err != nil {
		return nil, err
	}

	return e.GetWorkflow(id)
}

// ============ 工作流引擎 ============

// 评估URL匹配触发器
func evaluateURLMatch(bookmark *models.Bookmark, config map[string]interface{}) bool {
	matchMode, _ := config["match_mode"].(string)
	value, _ := config["value"].(string)

	if matchMode == "" {
		matchMode = "contains"
	}

	switch matchMode {
	case "contains":
		return strings.Contains(bookmark.URL, value)
	case "equals":
		return bookmark.URL == value
	case "regex":
		matched, _ := regexp.MatchString(value, bookmark.URL)
		return matched
	}
	return false
}

// 评估关键字匹配触发器
func evaluateKeywordMatch(bookmark *models.Bookmark, config map[string]interface{}) bool {
	field, _ := config["field"].(string)
	matchMode, _ := config["match_mode"].(string)
	value, _ := config["value"].(string)
	caseSensitive, _ := config["case_sensitive"].(bool)

	if field == "" {
		field = "title"
	}
	if matchMode == "" {
		matchMode = "contains"
	}

	var text string
	switch field {
	case "title":
		text = bookmark.Title
	case "description":
		text = bookmark.Description
	case "both":
		text = bookmark.Title + " " + bookmark.Description
	}

	if !caseSensitive {
		text = strings.ToLower(text)
		value = strings.ToLower(value)
	}

	switch matchMode {
	case "contains":
		return strings.Contains(text, value)
	case "equals":
		return text == value
	case "regex":
		matched, _ := regexp.MatchString(value, text)
		return matched
	}
	return false
}

// 评估单个触发器
func evaluateTrigger(bookmark *models.Bookmark, trigger models.WorkflowTrigger) bool {
	switch trigger.TriggerType {
	case "url_match":
		return evaluateURLMatch(bookmark, trigger.Config)
	case "keyword_match":
		return evaluateKeywordMatch(bookmark, trigger.Config)
	// 事件触发器在事件发生时已经匹配,这里总是返回true
	case "bookmark_created", "bookmark_updated", "bookmark_deleted",
		"title_changed", "description_added", "bookmark_tagged":
		return true
	default:
		return false
	}
}

// 评估工作流是否匹配
func evaluateWorkflow(bookmark *models.Bookmark, workflow *models.Workflow) bool {
	if len(workflow.Triggers) == 0 {
		return false
	}

	results := make([]bool, len(workflow.Triggers))
	for i, trigger := range workflow.Triggers {
		results[i] = evaluateTrigger(bookmark, trigger)
	}

	// 根据condition_logic组合结果
	if workflow.ConditionLogic == "AND" {
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	} else {
		// OR
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	}
}

// 执行工作流动作
func (e *WorkflowEngine) executeWorkflowActions(bookmark *models.Bookmark, actions []models.WorkflowAction) {
	for _, action := range actions {
		switch action.ActionType {
		case "move_to_folder":
			if folderID, ok := action.Config["folder_id"].(float64); ok {
				e.folderRepo.AddBookmark(bookmark.ID, int(folderID))
			}
		}
	}
}

// 对书签执行所有启用的工作流
func (e *WorkflowEngine) executeWorkflowsForBookmark(bookmark *models.Bookmark) {
	workflows, err := e.ListWorkflows()
	if err != nil {
		return
	}

	for _, workflow := range workflows {
		if !workflow.Enabled {
			continue
		}

		if evaluateWorkflow(bookmark, workflow) {
			e.executeWorkflowActions(bookmark, workflow.Actions)
		}
	}
}

// ApplyWorkflowsToBookmarks applies workflows to bookmarks
func (e *WorkflowEngine) ApplyWorkflowsToBookmarks(workflowIDs []int, bookmarkIDs []int) error {
	var workflows []*models.Workflow

	// 获取要应用的工作流
	if len(workflowIDs) == 0 {
		// 应用所有启用的工作流
		allWorkflows, _ := e.ListWorkflows()
		for _, w := range allWorkflows {
			if w.Enabled {
				workflows = append(workflows, w)
			}
		}
	} else {
		// 应用指定的工作流
		for _, id := range workflowIDs {
			if w, err := e.GetWorkflow(id); err == nil {
				workflows = append(workflows, w)
			}
		}
	}

	// 获取要处理的书签
	var bookmarks []*models.Bookmark
	if len(bookmarkIDs) == 0 {
		// 处理所有书签
		filters := make(map[string]interface{})
		bookmarks, _ = e.bookmarkRepo.List(10000, 0, filters)
	} else {
		// 处理指定的书签
		for _, id := range bookmarkIDs {
			if bm, err := e.bookmarkRepo.GetByID(id); err == nil {
				bookmarks = append(bookmarks, bm)
			}
		}
	}

	// 应用工作流
	for _, bookmark := range bookmarks {
		for _, workflow := range workflows {
			if evaluateWorkflow(bookmark, workflow) {
				e.executeWorkflowActions(bookmark, workflow.Actions)
			}
		}
	}

	return nil
}
