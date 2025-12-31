package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"ai-bookmark-service/db"
	"ai-bookmark-service/models"
)

// 文件夹API处理函数
// 这些函数将被添加到main.go的HTTP处理部分

var folderRepo *db.FolderRepository

// SetFolderRepository sets the folder repository
func SetFolderRepository(repo *db.FolderRepository) {
	folderRepo = repo
}

// ============ 文件夹API处理函数 ============

// GET /api/folders/ - 获取所有文件夹
func HandleGetFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := folderRepo.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 确保 folders 不是 nil
	if folders == nil {
		folders = []*models.Folder{}
	}
	
	// 添加日志以调试
	log.Printf("📤 API返回文件夹数量: %d", len(folders))
	for _, f := range folders {
		log.Printf("  - %s (ID: %d): count=%d", f.Name, f.ID, f.Count)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

// POST /api/folders/ - 创建文件夹
func HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var fc models.FolderCreate
	if err := json.NewDecoder(r.Body).Decode(&fc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if fc.Name == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}
	
	folder, err := folderRepo.Create(&fc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

// PUT /api/folders/{id} - 更新文件夹
func HandleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	// 从URL路径提取ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
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
	
	folder, err := folderRepo.Update(id, fc, updateData.SortOrder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folder)
}

// DELETE /api/folders/{id} - 删除文件夹
func HandleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}
	
	if err := folderRepo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/folders/{id}/bookmarks - 获取文件夹内的书签
func HandleGetFolderBookmarks(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}
	
	// 分页参数
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
	
	bookmarks, total, err := folderRepo.GetBookmarks(id, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"count":   total,
		"results": bookmarks,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /api/bookmarks/{id}/folders - 添加书签到文件夹
func HandleAddBookmarkToFolders(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}
	
	bookmarkID, err := strconv.Atoi(parts[3])
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
	
	// 添加到所有指定的文件夹
	for _, folderID := range data.FolderIDs {
		if err := folderRepo.AddBookmark(bookmarkID, folderID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/bookmarks/{bookmarkId}/folders/{folderId} - 从文件夹移除书签
func HandleRemoveBookmarkFromFolder(w http.ResponseWriter, r *http.Request) {
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
	
	if err := folderRepo.RemoveBookmark(bookmarkID, folderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}
