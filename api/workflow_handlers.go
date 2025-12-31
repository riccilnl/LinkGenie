package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ai-bookmark-service/models"
	"ai-bookmark-service/services"
)

var workflowEngine *services.WorkflowEngine

// SetWorkflowEngine sets the workflow engine
func SetWorkflowEngine(engine *services.WorkflowEngine) {
	workflowEngine = engine
}

// ============ 工作流API处理函数 ============

// GET /api/workflows/ - 获取所有工作流
func HandleGetWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := workflowEngine.ListWorkflows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

// POST /api/workflows/ - 创建工作流
func HandleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var wc models.WorkflowCreate
	if err := json.NewDecoder(r.Body).Decode(&wc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if wc.Name == "" {
		http.Error(w, "Workflow name is required", http.StatusBadRequest)
		return
	}
	
	workflow, err := workflowEngine.CreateWorkflow(&wc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(workflow)
}

// PUT /api/workflows/{id} - 更新工作流
func HandleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	var wc models.WorkflowCreate
	if err := json.NewDecoder(r.Body).Decode(&wc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	workflow, err := workflowEngine.UpdateWorkflow(id, &wc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}

// DELETE /api/workflows/{id} - 删除工作流
func HandleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	if err := workflowEngine.DeleteWorkflow(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/workflows/{id}/toggle - 切换启用状态
func HandleToggleWorkflow(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	
	workflow, err := workflowEngine.ToggleWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}

// POST /api/workflows/apply - 批量应用工作流
func HandleApplyWorkflows(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WorkflowIDs []int `json:"workflow_ids"`
		BookmarkIDs []int `json:"bookmark_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := workflowEngine.ApplyWorkflowsToBookmarks(data.WorkflowIDs, data.BookmarkIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
