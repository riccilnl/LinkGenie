package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-bookmark-service/config"
	"ai-bookmark-service/models"
)

// AIService AI 增强服务
type AIService struct {
	config  *config.Config
	scraper *ScraperService
}

// NewAIService 创建 AI 服务
func NewAIService(cfg *config.Config, scraper *ScraperService) *AIService {
	return &AIService{
		config:  cfg,
		scraper: scraper,
	}
}

// Enhance 使用 AI 增强书签
func (s *AIService) Enhance(url string) (*models.AIResponse, error) {
	// 详细日志：显示 AI 配置状态（脱敏）
	apiKeyPreview := "未设置"
	if len(s.config.AIAPIKey) > 4 {
		apiKeyPreview = "***" + s.config.AIAPIKey[len(s.config.AIAPIKey)-4:]
	}
	
	log.Printf("🔍 AI配置检查: AIEnabled=%v, AIAPIKey=%s, AIEndpoint=%s", 
		s.config.AIEnabled, apiKeyPreview, s.config.AIEndpoint)
	
	if !s.config.AIEnabled || s.config.AIAPIKey == "" {
		return nil, fmt.Errorf("AI未启用")
	}

	// 先尝试抓取网页内容
	metadata, err := s.scraper.ScrapeWebPage(url)
	if err != nil {
		log.Printf("⚠️ 网页抓取失败: %v, 降级为只用URL", err)
		metadata = &models.PageMetadata{}
	}

	// 构建AI提示词,优先使用抓取的内容
	prompt := s.buildPrompt(url, metadata)

	// 调用 AI API
	reqBody := map[string]interface{}{
		"model": s.config.AIModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("❌ JSON序列化失败: %v", err)
		return nil, fmt.Errorf("JSON序列化失败: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.AIEndpoint, bytes.NewReader(reqJSON))
	if err != nil {
		log.Printf("❌ 创建HTTP请求失败: %v", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ AI服务返回错误状态: %d %s", resp.StatusCode, resp.Status)
		
		// 特殊处理认证错误
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("AI API认证失败: 请检查AI_API_KEY是否正确 (状态码: %d)", resp.StatusCode)
		}
		
		return nil, fmt.Errorf("AI服务错误: %s (状态码: %d)", resp.Status, resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// 限制响应体大小为1MB,防止超大响应
	limitedReader := io.LimitReader(resp.Body, 1024*1024)
	if err := json.NewDecoder(limitedReader).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("AI无响应")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var aiResp models.AIResponse
	if err := json.Unmarshal([]byte(content), &aiResp); err != nil {
		return nil, fmt.Errorf("解析AI JSON失败: %w", err)
	}

	return &aiResp, nil
}

// buildPrompt 构建 AI 提示词
func (s *AIService) buildPrompt(url string, metadata *models.PageMetadata) string {
	pageTitle := metadata.OGTitle
	if pageTitle == "" {
		pageTitle = metadata.Title
	}
	pageDesc := metadata.OGDesc
	if pageDesc == "" {
		pageDesc = metadata.Description
	}

	if pageTitle != "" || pageDesc != "" {
		// 有抓取内容,使用真实信息
		return fmt.Sprintf(`分析这个网页并返回JSON格式的书签信息:

URL: %s
网页标题: %s
网页描述: %s

请基于以上真实内容返回以下JSON格式(不要包含markdown代码块标记):
{
  "title": "简洁的中文标题(20字内)",
  "description": "网页核心内容的详细摘要(100-150字)，重点概括该页面的主要观点、功能或核心价值",
  "tags": ["标签1", "标签2", "标签3"]
}

要求:
1. 标题要简洁明了,基于网页真实标题
2. 描述要详实深邃，不要记流水账，要能体现网页的核心价值
3. 标签要准确分类(3-5个)
4. 只返回JSON,不要其他内容`, url, pageTitle, pageDesc)
	}

	// 抓取失败,降级为只用URL
	return fmt.Sprintf(`分析这个网页URL并返回JSON格式的书签信息:

URL: %s

请返回以下JSON格式(不要包含markdown代码块标记):
{
  "title": "简洁的中文标题(20字内)",
  "description": "网页核心内容的详细摘要(100-150字)，重点概括该页面的主要观点、功能或核心价值",
  "tags": ["标签1", "标签2", "标签3"]
}

要求:
1. 标题要简洁明了
2. 描述要详实深邃，不要记流水账，要能体现网页的核心价值
3. 标签要准确分类(3-5个)
4. 只返回JSON,不要其他内容`, url)
}
