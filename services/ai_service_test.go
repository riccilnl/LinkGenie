package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riccilnl/LinkGenie/config"
	"github.com/riccilnl/LinkGenie/models"
)

func TestParseAIResponseContentSupportsThinkBlocks(t *testing.T) {
	content := `<think>
Thinking...
{"title":"模板","description":"模板","tags":["模板"]}
</think>

{"title":"最终标题","description":"最终描述","tags":["go","ollama"]}`

	resp, err := parseAIResponseContent(content)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if resp.Title != "最终标题" {
		t.Fatalf("预期解析最终 JSON，实际 title=%q", resp.Title)
	}
	if resp.Description != "最终描述" {
		t.Fatalf("预期解析最终 JSON，实际 description=%q", resp.Description)
	}
	if len(resp.Tags) != 2 || resp.Tags[0] != "go" || resp.Tags[1] != "ollama" {
		t.Fatalf("标签解析错误: %+v", resp.Tags)
	}
}

func TestParseAIResponseContentSupportsMarkdownFence(t *testing.T) {
	content := "```json\n{\"title\":\"标题\",\"description\":\"描述\",\"tags\":[\"a\",\"b\"]}\n```"

	resp, err := parseAIResponseContent(content)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if resp.Title != "标题" || resp.Description != "描述" {
		t.Fatalf("解析结果不符合预期: %+v", resp)
	}
}

func TestAIServiceEnhanceSupportsOllamaNativeEndpoint(t *testing.T) {
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>GitHub - owner/repo: English Title · GitHub</title><meta name="description" content="Original English description"></head><body>demo</body></html>`))
	}))
	defer pageServer.Close()

	var captured struct {
		Path    string
		Model   string
		Stream  bool
		Think   bool
		Format  string
		Content string
	}

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path

		var reqBody struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Think    bool   `json:"think"`
			Format   string `json:"format"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("解析 Ollama 请求失败: %v", err)
		}

		captured.Model = reqBody.Model
		captured.Stream = reqBody.Stream
		captured.Think = reqBody.Think
		captured.Format = reqBody.Format
		if len(reqBody.Messages) > 0 {
			captured.Content = reqBody.Messages[0].Content
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"{\"title\":\"中文标题\",\"description\":\"中文摘要\",\"tags\":[\"标签1\",\"标签2\"]}"}}`))
	}))
	defer aiServer.Close()

	cfg := &config.Config{
		AIEnabled:  true,
		AIAPIKey:   "demo-key",
		AIEndpoint: aiServer.URL + "/api/chat",
		AIModel:    "qwen3.5:9b",
	}
	service := NewAIService(cfg, NewScraperService())

	resp, err := service.Enhance(pageServer.URL, "Old English Title", "Old English Description")
	if err != nil {
		t.Fatalf("调用 Ollama 原生协议失败: %v", err)
	}

	if captured.Path != "/api/chat" {
		t.Fatalf("预期切换到 /api/chat，实际为 %q", captured.Path)
	}
	if captured.Model != "qwen3.5:9b" {
		t.Fatalf("预期模型为 qwen3.5:9b，实际为 %q", captured.Model)
	}
	if captured.Stream {
		t.Fatalf("预期 stream=false")
	}
	if captured.Think {
		t.Fatalf("预期 think=false")
	}
	if captured.Format != "json" {
		t.Fatalf("预期 format=json，实际为 %q", captured.Format)
	}
	if !strings.Contains(captured.Content, "当前书签标题: Old English Title") {
		t.Fatalf("预期 prompt 包含旧标题，实际为: %s", captured.Content)
	}
	if !strings.Contains(captured.Content, "当前书签描述: Old English Description") {
		t.Fatalf("预期 prompt 包含旧描述，实际为: %s", captured.Content)
	}
	if resp.Title != "中文标题" || resp.Description != "中文摘要" {
		t.Fatalf("预期解析 Ollama 返回的 JSON，实际为 %+v", resp)
	}
}

func TestBuildPromptFallsBackToExistingBookmarkContent(t *testing.T) {
	service := NewAIService(&config.Config{}, NewScraperService())

	prompt := service.buildPrompt(
		"https://example.com/old-bookmark",
		nil,
		"Legacy English Title",
		"Legacy English Description",
	)

	if !strings.Contains(prompt, "当前书签标题: Legacy English Title") {
		t.Fatalf("预期 prompt 包含旧标题，实际为: %s", prompt)
	}
	if !strings.Contains(prompt, "当前书签描述: Legacy English Description") {
		t.Fatalf("预期 prompt 包含旧描述，实际为: %s", prompt)
	}
	if !strings.Contains(prompt, "网页标题: （无）") {
		t.Fatalf("预期网页标题缺失时使用占位符，实际为: %s", prompt)
	}
}

func TestBuildPromptPrioritizesGitHubReadme(t *testing.T) {
	service := NewAIService(&config.Config{}, NewScraperService())

	prompt := service.buildPrompt(
		"https://github.com/owner/repo",
		&models.PageMetadata{
			Title:       "GitHub - owner/repo: English Title · GitHub",
			Description: "GitHub project page description",
			Readme:      "# Project\nThis is the README content.",
		},
		"Legacy Title",
		"Legacy Description",
	)

	readmeIndex := strings.Index(prompt, "仓库 README（最高优先级）")
	titleIndex := strings.Index(prompt, "网页标题:")
	if readmeIndex == -1 {
		t.Fatalf("预期 prompt 包含 README 段落，实际为: %s", prompt)
	}
	if titleIndex == -1 {
		t.Fatalf("预期 prompt 包含网页标题段落，实际为: %s", prompt)
	}
	if readmeIndex > titleIndex {
		t.Fatalf("预期 README 优先于网页标题，实际 prompt 为: %s", prompt)
	}
	if !strings.Contains(prompt, "This is the README content.") {
		t.Fatalf("预期 prompt 包含 README 内容，实际为: %s", prompt)
	}
}
