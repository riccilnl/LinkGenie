package services

import (
	"fmt"
	"log"
	"math"

	"ai-bookmark-service/db"
	"ai-bookmark-service/models"
)

// TagOptimizer 标签优化服务
type TagOptimizer struct {
	tagRepo *db.TagRepository
	bmRepo  *db.BookmarkRepository
}

// NewTagOptimizer 创建标签优化服务
func NewTagOptimizer(tagRepo *db.TagRepository, bmRepo *db.BookmarkRepository) *TagOptimizer {
	return &TagOptimizer{
		tagRepo: tagRepo,
		bmRepo:  bmRepo,
	}
}

// OptimizationAction 优化操作
type OptimizationAction struct {
	Type              string  `json:"type"` // "merge" | "promote"
	Source            string  `json:"source,omitempty"`
	Target            string  `json:"target,omitempty"`
	Tag               string  `json:"tag,omitempty"`
	From              string  `json:"from,omitempty"`
	To                string  `json:"to,omitempty"`
	Similarity        float64 `json:"similarity,omitempty"`
	UsageCount        int     `json:"usage_count,omitempty"`
	AffectedBookmarks int     `json:"affected_bookmarks,omitempty"`
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Preview bool                 `json:"preview"`
	Actions []OptimizationAction `json:"actions"`
	Summary OptimizationSummary  `json:"summary"`
}

// OptimizationSummary 优化摘要
type OptimizationSummary struct {
	TotalMerges     int `json:"total_merges"`
	TotalPromotions int `json:"total_promotions"`
	TagsBefore      int `json:"tags_before"`
	TagsAfter       int `json:"tags_after"`
}

// Optimize 执行标签优化
func (o *TagOptimizer) Optimize(dryRun bool, enableMerge bool, enablePromotion bool) (*OptimizationResult, error) {
	result := &OptimizationResult{
		Preview: dryRun,
		Actions: []OptimizationAction{},
		Summary: OptimizationSummary{},
	}

	// 获取所有标签
	allTags, err := o.tagRepo.List()
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}
	result.Summary.TagsBefore = len(allTags)

	// 1. 标签晋升
	if enablePromotion {
		promotions, err := o.checkPromotions(dryRun)
		if err != nil {
			log.Printf("⚠️ 标签晋升检查失败: %v", err)
		} else {
			result.Actions = append(result.Actions, promotions...)
			result.Summary.TotalPromotions = len(promotions)
		}
	}

	// 2. 同义词合并
	if enableMerge {
		merges, err := o.findAndMergeSimilarTags(dryRun)
		if err != nil {
			log.Printf("⚠️ 同义词合并失败: %v", err)
		} else {
			result.Actions = append(result.Actions, merges...)
			result.Summary.TotalMerges = len(merges)
		}
	}

	// 计算优化后的标签数量
	result.Summary.TagsAfter = result.Summary.TagsBefore - result.Summary.TotalMerges

	return result, nil
}

// checkPromotions 检查并执行标签晋升
func (o *TagOptimizer) checkPromotions(dryRun bool) ([]OptimizationAction, error) {
	actions := []OptimizationAction{}

	// 获取所有候选和动态标签
	tags, err := o.tagRepo.List()
	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		// 候选 -> 动态 (使用3次)
		if tag.Category == "candidate" && tag.UsageCount >= 3 {
			actions = append(actions, OptimizationAction{
				Type:       "promote",
				Tag:        tag.Name,
				From:       "candidate",
				To:         "dynamic",
				UsageCount: tag.UsageCount,
			})

			if !dryRun {
				if err := o.tagRepo.UpdateCategory(tag.ID, "dynamic"); err != nil {
					log.Printf("❌ 晋升失败: %s, 错误: %v", tag.Name, err)
				} else {
					log.Printf("✅ 候选→动态: %s (使用%d次)", tag.Name, tag.UsageCount)
				}
			}
		}

		// 动态 -> 固定 (使用10次)
		if tag.Category == "dynamic" && tag.UsageCount >= 10 {
			actions = append(actions, OptimizationAction{
				Type:       "promote",
				Tag:        tag.Name,
				From:       "dynamic",
				To:         "fixed",
				UsageCount: tag.UsageCount,
			})

			if !dryRun {
				if err := o.tagRepo.UpdateCategory(tag.ID, "fixed"); err != nil {
					log.Printf("❌ 晋升失败: %s, 错误: %v", tag.Name, err)
				} else {
					log.Printf("⭐ 动态→固定: %s (使用%d次)", tag.Name, tag.UsageCount)
				}
			}
		}
	}

	return actions, nil
}

// findAndMergeSimilarTags 查找并合并相似标签
func (o *TagOptimizer) findAndMergeSimilarTags(dryRun bool) ([]OptimizationAction, error) {
	actions := []OptimizationAction{}

	// 获取动态和候选标签
	tags, err := o.tagRepo.ListByCategories([]string{"dynamic", "candidate"})
	if err != nil {
		return nil, err
	}

	// 检查是否需要合并 (移除 50 个标签的硬性阈值，改为有标签就检查，提高 AI 响应速度)
	if len(tags) < 2 {
		return actions, nil
	}

	log.Printf("🔍 准备检查 %d 个动态/候选标签的同义词合并", len(tags))

	// 计算两两相似度(简单的字符串相似度)
	merged := make(map[int]bool)
	for i := 0; i < len(tags); i++ {
		if merged[tags[i].ID] {
			continue
		}

		for j := i + 1; j < len(tags); j++ {
			if merged[tags[j].ID] {
				continue
			}

			similarity := o.calculateStringSimilarity(tags[i].Name, tags[j].Name)
			if similarity > 0.80 {
				// 优先保留使用次数多的标签
				var source, target *models.Tag
				if tags[i].UsageCount >= tags[j].UsageCount {
					source = tags[j]
					target = tags[i]
				} else {
					source = tags[i]
					target = tags[j]
				}

				// 获取受影响的书签数量
				affectedCount := o.getTagBookmarkCount(source.ID)

				actions = append(actions, OptimizationAction{
					Type:              "merge",
					Source:            source.Name,
					Target:            target.Name,
					Similarity:        similarity,
					AffectedBookmarks: affectedCount,
				})

				if !dryRun {
					if err := o.mergeTags(source.ID, target.ID); err != nil {
						log.Printf("❌ 合并失败: %s -> %s, 错误: %v", source.Name, target.Name, err)
					} else {
						log.Printf("🔀 自动合并: %s -> %s (相似度%.2f)", source.Name, target.Name, similarity)
						merged[source.ID] = true
					}
				}
			}
		}
	}

	return actions, nil
}

// calculateStringSimilarity 计算字符串相似度(简单版本,使用Levenshtein距离)
func (o *TagOptimizer) calculateStringSimilarity(s1, s2 string) float64 {
	// 如果完全相同
	if s1 == s2 {
		return 1.0
	}

	// 如果一个包含另一个
	if len(s1) > len(s2) {
		if containsSubstring(s1, s2) {
			return 0.85
		}
	} else {
		if containsSubstring(s2, s1) {
			return 0.85
		}
	}

	// 计算Levenshtein距离
	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 0
	}

	return 1.0 - (float64(distance) / maxLen)
}

// containsSubstring 检查是否包含子串
func containsSubstring(s, substr string) bool {
	runes1 := []rune(s)
	runes2 := []rune(substr)

	for i := 0; i <= len(runes1)-len(runes2); i++ {
		match := true
		for j := 0; j < len(runes2); j++ {
			if runes1[i+j] != runes2[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// levenshteinDistance 计算Levenshtein距离
func levenshteinDistance(s1, s2 string) int {
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	m := len(runes1)
	n := len(runes2)

	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 0
			if runes1[i-1] != runes2[j-1] {
				cost = 1
			}

			dp[i][j] = min(
				dp[i-1][j]+1,      // 删除
				dp[i][j-1]+1,      // 插入
				dp[i-1][j-1]+cost, // 替换
			)
		}
	}

	return dp[m][n]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// getTagBookmarkCount 获取标签关联的书签数量
func (o *TagOptimizer) getTagBookmarkCount(tagID int) int {
	count, err := o.tagRepo.GetBookmarkCount(tagID)
	if err != nil {
		log.Printf("⚠️ 获取标签书签数量失败: %v", err)
		return 0
	}
	return count
}

// mergeTags 合并标签
func (o *TagOptimizer) mergeTags(sourceID, targetID int) error {
	// 1. 将source标签的所有书签关联转移到target
	if err := o.tagRepo.MergeBookmarks(sourceID, targetID); err != nil {
		return fmt.Errorf("合并书签关联失败: %w", err)
	}

	// 2. 记录同义词关系
	if err := o.tagRepo.RecordSynonym(targetID, sourceID, 0.0, true); err != nil {
		log.Printf("⚠️ 记录同义词失败: %v", err)
	}

	// 3. 删除source标签
	if err := o.tagRepo.Delete(sourceID); err != nil {
		return fmt.Errorf("删除源标签失败: %w", err)
	}

	// 4. 更新target标签的使用次数
	if err := o.tagRepo.IncrementUsage(targetID); err != nil {
		log.Printf("⚠️ 更新使用次数失败: %v", err)
	}

	return nil
}

// GetStats 获取标签统计信息
func (o *TagOptimizer) GetStats() (map[string]interface{}, error) {
	tags, err := o.tagRepo.List()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total":               len(tags),
		"core":                0,
		"fixed":               0,
		"dynamic":             0,
		"candidate":           0,
		"optimization_needed": false,
		"top_tags":            []map[string]interface{}{},
		"merge_suggestions":   []map[string]interface{}{},
	}

	dynamicCount := 0
	for _, tag := range tags {
		switch tag.Category {
		case "core":
			stats["core"] = stats["core"].(int) + 1
		case "fixed":
			stats["fixed"] = stats["fixed"].(int) + 1
		case "dynamic":
			stats["dynamic"] = stats["dynamic"].(int) + 1
			dynamicCount++
		case "candidate":
			stats["candidate"] = stats["candidate"].(int) + 1
		}
	}

	// 检查是否需要优化
	if dynamicCount > 50 {
		stats["optimization_needed"] = true
	}

	// 获取top标签(按使用次数排序,取前10)
	topTags := o.tagRepo.GetTopTags(10)
	topTagsList := []map[string]interface{}{}
	for _, tag := range topTags {
		topTagsList = append(topTagsList, map[string]interface{}{
			"name":     tag.Name,
			"count":    tag.UsageCount,
			"category": tag.Category,
		})
	}
	stats["top_tags"] = topTagsList

	return stats, nil
}
