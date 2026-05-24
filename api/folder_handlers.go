package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/models"
)

type folderHandlers struct {
	repo *db.FolderRepository
}

func newFolderHandlers(repo *db.FolderRepository) *folderHandlers {
	return &folderHandlers{repo: repo}
}

// GET /api/folders/ - 获取所有文件夹
func (h *folderHandlers) handleGetFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := h.repo.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if folders == nil {
		folders = []*models.Folder{}
	}

	log.Printf("📤 API返回文件夹数量: %d", len(folders))
	for _, f := range folders {
		log.Printf("  - %s (ID: %d): count=%d", f.Name, f.ID, f.Count)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(folders)
}

// POST /api/folders/ - 创建文件夹
func (h *folderHandlers) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var fc models.FolderCreate
	if err := json.NewDecoder(r.Body).Decode(&fc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if fc.Name == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}

	folder, err := h.repo.Create(&fc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(folder)
}

// PUT /api/folders/{id} - 更新文件夹
func (h *folderHandlers) handleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	id, err := parseFolderIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	var updateData struct {
		Name      string `json:"name"`
		Color     string `json:"color"`
		Icon      string `json:"icon"`
		SortOrder *int   `json:"sort_order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fc := &models.FolderCreate{
		Name:  updateData.Name,
		Color: updateData.Color,
		Icon:  updateData.Icon,
	}

	folder, err := h.repo.Update(id, fc, updateData.SortOrder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(folder)
}

// DELETE /api/folders/{id} - 删除文件夹
func (h *folderHandlers) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	id, err := parseFolderIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/folders/{id}/bookmarks - 获取文件夹内的书签
func (h *folderHandlers) handleGetFolderBookmarks(w http.ResponseWriter, r *http.Request) {
	id, err := parseFolderIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	limit := 100
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	bookmarks, total, err := h.repo.GetBookmarks(id, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"count":   total,
		"results": bookmarks,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// POST /api/bookmarks/{id}/folders - 添加书签到文件夹
func (h *folderHandlers) handleAddBookmarkToFolders(w http.ResponseWriter, r *http.Request) {
	bookmarkID, err := parseBookmarkIDForFolderPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	var data struct {
		FolderIDs []int `json:"folder_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _, folderID := range data.FolderIDs {
		if err := h.repo.AddBookmark(bookmarkID, folderID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/bookmarks/{bookmarkId}/folders/{folderId} - 从文件夹移除书签
func (h *folderHandlers) handleRemoveBookmarkFromFolder(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	bookmarkID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	folderID, err := strconv.Atoi(parts[5])
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.RemoveBookmark(bookmarkID, folderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseFolderIDFromPath(path string) (int, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(parts[3])
}

func parseBookmarkIDForFolderPath(path string) (int, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(parts[3])
}
