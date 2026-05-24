package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/riccilnl/LinkGenie/models"
	"github.com/riccilnl/LinkGenie/services"
)

type workflowHandlers struct {
	engine *services.WorkflowEngine
}

func newWorkflowHandlers(engine *services.WorkflowEngine) *workflowHandlers {
	return &workflowHandlers{engine: engine}
}

// GET /api/workflows/ - 获取所有工作流
func (h *workflowHandlers) handleGetWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := h.engine.ListWorkflows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(workflows)
}

// POST /api/workflows/ - 创建工作流
func (h *workflowHandlers) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var wc models.WorkflowCreate
	if err := json.NewDecoder(r.Body).Decode(&wc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if wc.Name == "" {
		http.Error(w, "Workflow name is required", http.StatusBadRequest)
		return
	}

	workflow, err := h.engine.CreateWorkflow(&wc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(workflow)
}

// PUT /api/workflows/{id} - 更新工作流
func (h *workflowHandlers) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := parseWorkflowIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	var wc models.WorkflowCreate
	if err := json.NewDecoder(r.Body).Decode(&wc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	workflow, err := h.engine.UpdateWorkflow(id, &wc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(workflow)
}

// DELETE /api/workflows/{id} - 删除工作流
func (h *workflowHandlers) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := parseWorkflowIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	if err := h.engine.DeleteWorkflow(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/workflows/{id}/toggle - 切换启用状态
func (h *workflowHandlers) handleToggleWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := parseWorkflowIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	workflow, err := h.engine.ToggleWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(workflow)
}

// POST /api/workflows/apply - 批量应用工作流
func (h *workflowHandlers) handleApplyWorkflows(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WorkflowIDs []int `json:"workflow_ids"`
		BookmarkIDs []int `json:"bookmark_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.engine.ApplyWorkflowsToBookmarks(data.WorkflowIDs, data.BookmarkIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func parseWorkflowIDFromPath(path string) (int, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(parts[3])
}
