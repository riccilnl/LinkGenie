package services

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestScrapeWebPageExtractsGitHubReadme(t *testing.T) {
	htmlBody := `<!doctype html>
<html>
  <head>
    <title>GitHub - owner/repo: English Title · GitHub</title>
    <meta name="description" content="Original English description">
  </head>
  <body>
    <div id="readme">
      <article class="markdown-body entry-content container-lg">
        <h1>Project README</h1>
        <p>This project helps users do something useful.</p>
        <ul>
          <li>Item one</li>
          <li>Item two</li>
        </ul>
      </article>
    </div>
  </body>
</html>`

	scraper := &ScraperService{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "github.com" {
					t.Fatalf("预期请求 github.com，实际为 %s", req.URL.Host)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(htmlBody)),
					Request:    req,
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
				}, nil
			}),
		},
	}

	metadata, err := scraper.ScrapeWebPage("https://github.com/owner/repo")
	if err != nil {
		t.Fatalf("抓取失败: %v", err)
	}

	if metadata.Readme == "" {
		t.Fatalf("预期提取 README 内容，实际为空")
	}
	if !strings.Contains(metadata.Readme, "Project README") {
		t.Fatalf("README 内容缺少标题，实际为: %s", metadata.Readme)
	}
	if !strings.Contains(metadata.Readme, "This project helps users do something useful.") {
		t.Fatalf("README 内容缺少正文，实际为: %s", metadata.Readme)
	}
	if !strings.Contains(metadata.Readme, "Item one") || !strings.Contains(metadata.Readme, "Item two") {
		t.Fatalf("README 内容缺少列表项，实际为: %s", metadata.Readme)
	}
}
