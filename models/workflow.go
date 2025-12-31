package models

import "time"

// Workflow 工作流数据模型
type Workflow struct {
	ID             int               `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Enabled        bool              `json:"enabled"`
	Priority       int               `json:"priority"`
	ConditionLogic string            `json:"condition_logic"` // OR or AND
	Triggers       []WorkflowTrigger `json:"triggers"`
	Actions        []WorkflowAction  `json:"actions"`
	DateAdded      time.Time         `json:"date_added"`
	DateModified   time.Time         `json:"date_modified"`
	MatchCount     int               `json:"match_count"` // 匹配的书签数量
}

// WorkflowCreate 创建工作流请求
type WorkflowCreate struct {
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Enabled        bool                     `json:"enabled"`
	ConditionLogic string                   `json:"condition_logic"`
	Triggers       []WorkflowTriggerCreate  `json:"triggers"`
	Actions        []WorkflowActionCreate   `json:"actions"`
}

// WorkflowTrigger 工作流触发器
type WorkflowTrigger struct {
	ID          int                    `json:"id"`
	WorkflowID  int                    `json:"workflow_id"`
	TriggerType string                 `json:"trigger_type"` // url_match, keyword_match
	Config      map[string]interface{} `json:"config"`
}

// WorkflowTriggerCreate 创建触发器请求
type WorkflowTriggerCreate struct {
	TriggerType string                 `json:"trigger_type"`
	Config      map[string]interface{} `json:"config"`
}

// WorkflowAction 工作流动作
type WorkflowAction struct {
	ID         int                    `json:"id"`
	WorkflowID int                    `json:"workflow_id"`
	ActionType string                 `json:"action_type"` // move_to_folder
	Config     map[string]interface{} `json:"config"`
}

// WorkflowActionCreate 创建动作请求
type WorkflowActionCreate struct {
	ActionType string                 `json:"action_type"`
	Config     map[string]interface{} `json:"config"`
}
