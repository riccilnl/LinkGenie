package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/riccilnl/LinkGenie/models"
	"github.com/riccilnl/LinkGenie/serverapp"
	"github.com/riccilnl/LinkGenie/services"
	"github.com/riccilnl/LinkGenie/utils"
)

type bookmarkHandlers struct {
	app *serverapp.App
}

func newBookmarkHandlers(app *serverapp.App) *bookmarkHandlers {
	return &bookmarkHandlers{app: app}
}

func (h *bookmarkHandlers) handleBookmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBookmarks(w, r)
	case http.MethodPost:
		h.createBookmark(w, r)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *bookmarkHandlers) listBookmarks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	filters := make(map[string]interface{})
	if q := query.Get("q"); q != "" {
		filters["q"] = q
	}
	if query.Get("unread") == "true" {
		filters["unread"] = true
	}
	if query.Get("shared") == "true" {
		filters["shared"] = true
	}
	if tag := query.Get("tag"); tag != "" {
		filters["tag"] = tag
	}
	if folderID, err := strconv.Atoi(query.Get("folder_id")); err == nil && folderID > 0 {
		filters["folder_id"] = folderID
	}

	bookmarks, err := h.app.BookmarkRepository().List(limit, offset, filters)
	if err != nil {
		log.Printf("❌ 查询书签失败: %v", err)
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	if bookmarks == nil {
		bookmarks = []*models.Bookmark{}
	}

	count, _ := h.app.BookmarkRepository().Count(filters)
	serverapp.EncodeJSON(w, map[string]interface{}{
		"count":    count,
		"next":     nil,
		"previous": nil,
		"results":  bookmarks,
	})
}

func (h *bookmarkHandlers) createBookmark(w http.ResponseWriter, r *http.Request) {
	input, err := parseCreateBookmarkInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.app.BookmarkService().CreateBookmark(r.Context(), input)
	if err != nil {
		if services.IsConflictError(err) && isBookmarkFormRequest(r) {
			normalizedURL, normalizeErr := utils.NormalizeURL(input.URL)
			if normalizeErr == nil {
				existing, getErr := h.app.BookmarkRepository().GetByURL(normalizedURL)
				if getErr == nil {
					log.Printf("ℹ️ 表单客户端重复创建按幂等成功返回: bookmark_id=%d", existing.ID)
					serverapp.EncodeJSON(w, existing)
					return
				}
			}
		}

		log.Printf("❌ 创建书签失败: %v", err)
		writeBookmarkServiceError(w, err, "创建失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func isBookmarkFormRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded")
}

func (h *bookmarkHandlers) handleImportBookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "无效的导入文件", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "缺少导入文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	service := h.app.BookmarkExchangeService()
	if service == nil {
		http.Error(w, "导入服务不可用", http.StatusInternalServerError)
		return
	}

	result, err := service.ImportHTML(r.Context(), file)
	if err != nil {
		log.Printf("❌ 导入书签失败: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverapp.EncodeJSON(w, result)
}

func (h *bookmarkHandlers) handleExportBookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	service := h.app.BookmarkExchangeService()
	if service == nil {
		http.Error(w, "导出服务不可用", http.StatusInternalServerError)
		return
	}

	content, err := service.ExportHTML(r.Context())
	if err != nil {
		log.Printf("❌ 导出书签失败: %v", err)
		http.Error(w, "导出失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"bookmarks_"+time.Now().UTC().Format("2006-01-02")+".html\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *bookmarkHandlers) handleCheckBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		http.Error(w, "缺少url参数", http.StatusBadRequest)
		return
	}

	normalizedURL, err := utils.NormalizeURL(urlStr)
	if err != nil {
		serverapp.EncodeJSON(w, map[string]interface{}{
			"bookmark": nil,
			"metadata": map[string]string{
				"url": urlStr,
			},
		})
		return
	}

	bm, err := h.app.BookmarkRepository().GetByURL(normalizedURL)
	if err == nil {
		serverapp.EncodeJSON(w, map[string]interface{}{
			"bookmark": bm,
			"metadata": map[string]string{
				"url":         bm.URL,
				"title":       bm.Title,
				"description": bm.Description,
			},
		})
		return
	}

	metadata, err := h.app.ScraperService().ScrapeWebPage(normalizedURL)
	if err != nil {
		serverapp.EncodeJSON(w, map[string]interface{}{
			"bookmark": nil,
			"metadata": map[string]string{
				"url": normalizedURL,
			},
		})
		return
	}

	serverapp.EncodeJSON(w, map[string]interface{}{
		"bookmark": nil,
		"metadata": map[string]string{
			"url":         normalizedURL,
			"title":       metadata.Title,
			"description": metadata.Description,
		},
	})
}

func (h *bookmarkHandlers) handleBookmarkByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseBookmarkID(r.URL.Path)
	if err != nil {
		http.Error(w, "无效的ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBookmark(w, r, id)
	case http.MethodPatch, http.MethodPut:
		h.updateBookmark(w, r, id)
	case http.MethodDelete:
		h.deleteBookmark(w, r, id)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *bookmarkHandlers) getBookmark(w http.ResponseWriter, r *http.Request, id int) {
	bookmark, err := h.app.BookmarkRepository().GetByID(id)
	if err != nil {
		http.Error(w, "书签不存在", http.StatusNotFound)
		return
	}

	serverapp.EncodeJSON(w, bookmark)
}

func (h *bookmarkHandlers) updateBookmark(w http.ResponseWriter, r *http.Request, id int) {
	var (
		updated *models.Bookmark
		err     error
	)

	if r.Method == http.MethodPut {
		input, parseErr := parseReplaceBookmarkInput(r)
		if parseErr != nil {
			http.Error(w, parseErr.Error(), http.StatusBadRequest)
			return
		}
		updated, err = h.app.BookmarkService().UpdateBookmark(r.Context(), id, input)
	} else {
		input, parseErr := parsePatchBookmarkInput(r)
		if parseErr != nil {
			http.Error(w, parseErr.Error(), http.StatusBadRequest)
			return
		}
		updated, err = h.app.BookmarkService().PatchBookmark(r.Context(), id, input)
	}

	if err != nil {
		log.Printf("❌ 更新书签失败: %v", err)
		writeBookmarkServiceError(w, err, "更新失败")
		return
	}

	serverapp.EncodeJSON(w, updated)
}

func (h *bookmarkHandlers) deleteBookmark(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.app.BookmarkService().DeleteBookmark(r.Context(), id); err != nil {
		log.Printf("❌ 删除书签失败: %v", err)
		writeBookmarkServiceError(w, err, "删除失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *bookmarkHandlers) handleEnhanceBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	trimmedPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/bookmarks/"), "/")
	parts := strings.Split(trimmedPath, "/")
	if len(parts) != 2 || parts[1] != "enhance" {
		http.Error(w, "无效的路径", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "无效的ID", http.StatusBadRequest)
		return
	}

	cfg := h.app.ConfigSnapshot()
	if !cfg.AIEnabled {
		http.Error(w, "AI功能未启用", http.StatusServiceUnavailable)
		return
	}

	if err := h.app.BookmarkService().EnqueueEnhancement(r.Context(), id); err != nil {
		writeBookmarkServiceError(w, err, "AI增强触发失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "AI增强已开始处理",
		"id":      id,
	})
}

func parseBookmarkID(path string) (int, error) {
	idStr := strings.Trim(strings.TrimPrefix(path, "/api/bookmarks/"), "/")
	return strconv.Atoi(idStr)
}
