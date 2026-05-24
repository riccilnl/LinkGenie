package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/riccilnl/LinkGenie/config"
	"github.com/riccilnl/LinkGenie/models"
)

// AIService AI 增强服务
type AIService struct {
	config  *config.Config
	scraper *ScraperService
}

const aiRequestTimeout = 5 * time.Minute
const maxReadmePromptRunes = 12000

// NewAIService 创建 AI 服务
func NewAIService(cfg *config.Config, scraper *ScraperService) *AIService {
	return &AIService{
		config:  cfg,
		scraper: scraper,
	}
}

// Enhance 使用 AI 增强书签
func (s *AIService) Enhance(url, existingTitle, existingDescription string) (*models.AIResponse, error) {
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

	// 构建AI提示词,优先使用抓取的内容,并回退到已有书签信息
	prompt := s.buildPrompt(url, metadata, existingTitle, existingDescription)

	endpoint := s.config.AIEndpoint
	useOllamaNative := shouldUseOllamaNativeEndpoint(endpoint)

	// 调用 AI API
	reqBody := map[string]interface{}{}
	if useOllamaNative {
		endpoint = toOllamaChatEndpoint(endpoint)
		reqBody = map[string]interface{}{
			"model": s.config.AIModel,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream": false,
			"think":  false,
			"format": "json",
			"options": map[string]interface{}{
				"temperature": 0,
				"num_predict": 300,
			},
		}
	} else {
		reqBody = map[string]interface{}{
			"model": s.config.AIModel,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"temperature": 0.2,
			"max_tokens":  300,
			"stream":      false,
		}
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("❌ JSON序列化失败: %v", err)
		return nil, fmt.Errorf("JSON序列化失败: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqJSON))
	if err != nil {
		log.Printf("❌ 创建HTTP请求失败: %v", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: aiRequestTimeout}
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

	// 限制响应体大小为1MB,防止超大响应
	limitedReader := io.LimitReader(resp.Body, 1024*1024)
	if useOllamaNative {
		var result struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}

		if err := json.NewDecoder(limitedReader).Decode(&result); err != nil {
			return nil, fmt.Errorf("解析Ollama响应失败: %w", err)
		}
		if strings.TrimSpace(result.Message.Content) == "" {
			return nil, fmt.Errorf("Ollama无响应")
		}

		return parseAIResponseContent(result.Message.Content)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(limitedReader).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("AI无响应")
	}

	return parseAIResponseContent(result.Choices[0].Message.Content)
}

// buildPrompt 构建 AI 提示词
func (s *AIService) buildPrompt(url string, metadata *models.PageMetadata, existingTitle, existingDescription string) string {
	if metadata == nil {
		metadata = &models.PageMetadata{}
	}

	pageTitle := metadata.OGTitle
	if pageTitle == "" {
		pageTitle = metadata.Title
	}
	pageDesc := metadata.OGDesc
	if pageDesc == "" {
		pageDesc = metadata.Description
	}

	pageTitle = strings.TrimSpace(pageTitle)
	pageDesc = strings.TrimSpace(pageDesc)
	existingTitle = strings.TrimSpace(existingTitle)
	existingDescription = strings.TrimSpace(existingDescription)
	readme := strings.TrimSpace(metadata.Readme)
	if readme != "" {
		readme = truncateReadmePromptText(readme, maxReadmePromptRunes)
	}

	if readme != "" {
		return fmt.Sprintf(`你正在增强一个已有书签。对于 GitHub 仓库链接，请优先根据仓库 README 内容来理解项目，再参考页面标题、页面描述和书签已有信息。如果 README 与页面标题冲突，以 README 为准。

URL: %s
仓库 README（最高优先级）: %s
网页标题: %s
网页描述: %s
当前书签标题: %s
当前书签描述: %s

任务要求:
1. 如果 README 已经足够明确，以 README 为准，页面标题和描述只作为补充。
2. 如果标题或描述是英文，请翻译成自然、简洁的中文，不要机械直译。
3. 如果标题里带有类似 "GitHub - owner/repo: ... · GitHub" 这种站点模板噪音，要去掉外壳，保留核心项目名和中文要点。
4. 描述请输出 80-140 字中文摘要，说明它是做什么的、适合谁、核心价值是什么，不要只改写一句英文原文。
5. 标签输出 3-5 个中文标签，便于后续检索。
6. 只返回 JSON，不要解释，不要 markdown 代码块。

请返回以下JSON格式:
{
  "title": "简洁的中文标题(20字内)",
  "description": "网页核心内容的详细摘要(100-150字)，重点概括该页面的主要观点、功能或核心价值",
  "tags": ["标签1", "标签2", "标签3"]
}

注意:
1. 标题和描述默认输出中文
2. 不要凭空编造功能细节
3. 只返回JSON,不要其他内容`,
			url,
			emptyPromptValue(readme),
			emptyPromptValue(pageTitle),
			emptyPromptValue(pageDesc),
			emptyPromptValue(existingTitle),
			emptyPromptValue(existingDescription),
		)
	}

	if pageTitle != "" || pageDesc != "" || existingTitle != "" || existingDescription != "" {
		return fmt.Sprintf(`你正在增强一个已有书签。请综合网页抓取结果与书签当前已有信息，输出更适合中文用户阅读和检索的书签信息。

URL: %s
网页标题: %s
网页描述: %s
当前书签标题: %s
当前书签描述: %s

任务要求:
1. 如果标题或描述是英文，请翻译成自然、简洁的中文，不要机械直译。
2. 如果标题里带有类似 "GitHub - owner/repo: ... · GitHub" 这种站点模板噪音，要去掉外壳，保留核心项目名和中文要点。
3. 描述请输出 80-140 字中文摘要，说明它是做什么的、适合谁、核心价值是什么，不要只改写一句英文原文。
4. 标签输出 3-5 个中文标签，便于后续检索。
5. 优先采用网页抓取到的信息；如果网页信息不足或抓取失败，再参考当前书签已有标题和描述。
6. 只返回 JSON，不要解释，不要 markdown 代码块。

请返回以下JSON格式:
{
  "title": "简洁的中文标题(20字内)",
  "description": "网页核心内容的详细摘要(100-150字)，重点概括该页面的主要观点、功能或核心价值",
  "tags": ["标签1", "标签2", "标签3"]
}

注意:
1. 标题和描述默认输出中文
2. 不要凭空编造功能细节
3. 只返回JSON,不要其他内容`,
			url,
			emptyPromptValue(pageTitle),
			emptyPromptValue(pageDesc),
			emptyPromptValue(existingTitle),
			emptyPromptValue(existingDescription),
		)
	}

	// 抓取失败且没有已有书签信息,降级为只用URL
	return fmt.Sprintf(`你正在增强一个书签，但当前只能拿到 URL。请根据 URL 给出尽量克制的中文标题、中文摘要和中文标签。

URL: %s

任务要求:
1. 标题默认输出中文，避免直接照搬域名。
2. 描述请尽量概括该链接可能的用途；如果无法确认细节，保持保守，不要编造。
3. 标签输出 3-5 个中文标签。
4. 只返回 JSON，不要解释，不要 markdown 代码块。

请返回以下JSON格式:
{
  "title": "简洁的中文标题(20字内)",
  "description": "网页核心内容的详细摘要(100-150字)，重点概括该页面的主要观点、功能或核心价值",
  "tags": ["标签1", "标签2", "标签3"]
}

注意:
1. 默认输出中文
2. 不要编造
3. 只返回JSON,不要其他内容`, url)
}

func truncateReadmePromptText(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}

	return string(runes[:maxRunes]) + "\n[README 内容已截断]"
}

func emptyPromptValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "（无）"
	}

	return value
}

func shouldUseOllamaNativeEndpoint(endpoint string) bool {
	parsed, err := neturl.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Host)
	path := strings.ToLower(parsed.Path)

	if strings.HasSuffix(path, "/api/chat") {
		return true
	}

	return strings.Contains(host, "11434") && strings.HasSuffix(path, "/v1/chat/completions")
}

func toOllamaChatEndpoint(endpoint string) string {
	parsed, err := neturl.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return endpoint
	}

	parsed.Path = "/api/chat"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func parseAIResponseContent(content string) (*models.AIResponse, error) {
	normalized := normalizeAIResponseContent(content)

	var aiResp models.AIResponse
	if err := json.Unmarshal([]byte(normalized), &aiResp); err == nil {
		return &aiResp, nil
	}

	candidates := extractJSONObjectCandidates(normalized)
	for i := len(candidates) - 1; i >= 0; i-- {
		var candidate models.AIResponse
		if err := json.Unmarshal([]byte(candidates[i]), &candidate); err != nil {
			continue
		}
		if candidate.Title != "" || candidate.Description != "" || len(candidate.Tags) > 0 {
			return &candidate, nil
		}
	}

	return nil, fmt.Errorf("解析AI JSON失败: 未找到有效 JSON 对象")
}

func normalizeAIResponseContent(content string) string {
	content = strings.TrimSpace(content)
	content = regexp.MustCompile(`(?is)<think>.*?</think>`).ReplaceAllString(content, "")
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func extractJSONObjectCandidates(content string) []string {
	candidates := []string{}
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				candidates = append(candidates, strings.TrimSpace(content[start:i+1]))
				start = -1
			}
		}
	}

	return candidates
}
