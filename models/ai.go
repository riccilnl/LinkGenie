package models

// AIResponse AI 响应数据模型
type AIResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// PageMetadata 网页元数据
type PageMetadata struct {
	Title       string
	Description string
	OGTitle     string
	OGDesc      string
}
