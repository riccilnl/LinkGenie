package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/riccilnl/LinkGenie/serverapp"
)

type tagHandlers struct {
	app *serverapp.App
}

func newTagHandlers(app *serverapp.App) *tagHandlers {
	return &tagHandlers{app: app}
}

func (h *tagHandlers) handleTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTags(w, r)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *tagHandlers) listTags(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	search := strings.TrimSpace(strings.ToLower(query.Get("q")))

	tags, err := h.app.TagRepository().List()
	if err != nil {
		log.Printf("❌ 查询标签失败: %v", err)
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	filtered := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		if search != "" && !strings.Contains(strings.ToLower(tag.Name), search) {
			continue
		}
		filtered = append(filtered, tag)
	}

	total := len(filtered)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	end := offset + limit
	if end > total {
		end = total
	}

	serverapp.EncodeJSON(w, map[string]interface{}{
		"count":    total,
		"next":     nil,
		"previous": nil,
		"results":  filtered[offset:end],
	})
}
