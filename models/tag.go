package models

// Tag 标签数据模型
type Tag struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`         // core | fixed | dynamic | candidate
	UsageCount      int     `json:"usage_count"`      // 使用次数
	LastUsed        string  `json:"last_used"`        // 最后使用时间
	ParentTagID     *int    `json:"parent_tag_id"`    // 同义词映射
	ConfidenceScore float64 `json:"confidence_score"` // AI生成置信度
	DateAdded       string  `json:"date_added"`
}
