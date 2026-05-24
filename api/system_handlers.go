package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/riccilnl/LinkGenie/serverapp"
)

type systemHandlers struct {
	app *serverapp.App
}

func newSystemHandlers(app *serverapp.App) *systemHandlers {
	return &systemHandlers{app: app}
}

func (h *systemHandlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	serverapp.EncodeJSON(w, h.app.MarshalStatus())
}

func (h *systemHandlers) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serverapp.EncodeJSON(w, h.app.MarshalConfig())
	case http.MethodPost:
		var newConfig map[string]string
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.app.SaveSystemConfig(newConfig); err != nil {
			log.Printf("❌ 系统配置保存失败: %v", err)
			status := http.StatusInternalServerError
			if isValidationError(err) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}

		serverapp.EncodeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
