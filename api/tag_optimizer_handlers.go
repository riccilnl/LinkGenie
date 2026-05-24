package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/riccilnl/LinkGenie/services"
)

type tagOptimizerHandlers struct {
	optimizer *services.TagOptimizer
}

func newTagOptimizerHandlers(optimizer *services.TagOptimizer) *tagOptimizerHandlers {
	return &tagOptimizerHandlers{optimizer: optimizer}
}

// GET /api/tags/stats - 获取标签统计信息
func (h *tagOptimizerHandlers) handleGetTagStats(w http.ResponseWriter, r *http.Request) {
	if h.optimizer == nil {
		http.Error(w, "标签优化服务未初始化", http.StatusInternalServerError)
		return
	}

	stats, err := h.optimizer.GetStats()
	if err != nil {
		log.Printf("❌ 获取标签统计失败: %v", err)
		http.Error(w, "获取统计失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// POST /api/tags/optimize - 手动触发标签优化
func (h *tagOptimizerHandlers) handleOptimizeTags(w http.ResponseWriter, r *http.Request) {
	if h.optimizer == nil {
		http.Error(w, "标签优化服务未初始化", http.StatusInternalServerError)
		return
	}

	var req struct {
		DryRun          bool `json:"dry_run"`
		EnableMerge     bool `json:"enable_merge"`
		EnablePromotion bool `json:"enable_promotion"`
	}

	req.DryRun = true
	req.EnableMerge = true
	req.EnablePromotion = true

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("⚠️ 解析请求失败,使用默认值: %v", err)
	}

	log.Printf("🔧 开始标签优化: dry_run=%v, merge=%v, promotion=%v",
		req.DryRun, req.EnableMerge, req.EnablePromotion)

	result, err := h.optimizer.Optimize(req.DryRun, req.EnableMerge, req.EnablePromotion)
	if err != nil {
		log.Printf("❌ 标签优化失败: %v", err)
		http.Error(w, "优化失败", http.StatusInternalServerError)
		return
	}

	if req.DryRun {
		log.Printf("👁️ 预览模式完成: 将合并%d个标签, 晋升%d个标签",
			result.Summary.TotalMerges, result.Summary.TotalPromotions)
	} else {
		log.Printf("✅ 优化完成: 合并了%d个标签, 晋升了%d个标签, 标签总数 %d -> %d",
			result.Summary.TotalMerges, result.Summary.TotalPromotions,
			result.Summary.TagsBefore, result.Summary.TagsAfter)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
