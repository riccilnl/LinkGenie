package services

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/riccilnl/LinkGenie/models"

	"golang.org/x/net/html"
)

var privateIPPrefixes = []struct {
	network *net.IPNet
}{
	{network: mustParseCIDR("10.0.0.0/8")},
	{network: mustParseCIDR("172.16.0.0/12")},
	{network: mustParseCIDR("192.168.0.0/16")},
	{network: mustParseCIDR("169.254.0.0/16")},
	{network: mustParseCIDR("127.0.0.0/8")},
	{network: mustParseCIDR("0.0.0.0/32")},
	{network: mustParseCIDR("::1/128")},
	{network: mustParseCIDR("fc00::/7")},
	{network: mustParseCIDR("fe80::/10")},
}

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic("无效 CIDR: " + s)
	}
	return n
}

var privateHostnames = map[string]bool{
	"localhost":       true,
	"localhost.localdomain": true,
	"localhost6":      true,
	"localhost6.localdomain6": true,
}

func isPrivateTarget(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if privateHostnames[host] {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return false
	}
	for _, ipStr := range ips {
		if ip := net.ParseIP(ipStr); ip != nil && isPrivateIP(ip) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	for _, p := range privateIPPrefixes {
		if p.network.Contains(ip) {
			return true
		}
	}
	return false
}

// ScraperService 网页抓取服务
type ScraperService struct {
	timeout time.Duration
	client  *http.Client
}

// NewScraperService 创建抓取服务
func NewScraperService() *ScraperService {
	timeout := 30 * time.Second
	return &ScraperService{
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *ScraperService) httpClient() *http.Client {
	if s != nil && s.client != nil {
		return s.client
	}

	timeout := 30 * time.Second
	if s != nil && s.timeout > 0 {
		timeout = s.timeout
	}

	return &http.Client{Timeout: timeout}
}

// ScrapeWebPage 抓取网页元数据
func (s *ScraperService) ScrapeWebPage(url string) (*models.PageMetadata, error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("无效 URL: %w", err)
	}
	if parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("URL 缺少主机名")
	}
	if isPrivateTarget(parsedURL.Hostname()) {
		return nil, fmt.Errorf("拒绝访问内网或回环地址: %s", parsedURL.Hostname())
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置User-Agent,模拟浏览器访问,避免被反爬虫拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://www.google.com/")

	// 发送请求
	client := s.httpClient()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("网页返回错误状态: %d %s", resp.StatusCode, resp.Status)
	}

	// 限制读取大小为128KB (增加到128KB以获取更多内容)
	limitedReader := io.LimitReader(resp.Body, 128*1024)

	// 解析HTML
	doc, err := html.Parse(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("HTML解析失败: %w", err)
	}

	metadata := &models.PageMetadata{}

	// 遍历HTML节点提取信息
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					metadata.Title = n.FirstChild.Data
				}
			case "meta":
				var name, property, content string
				for _, attr := range n.Attr {
					switch attr.Key {
					case "name":
						name = attr.Val
					case "property":
						property = attr.Val
					case "content":
						content = attr.Val
					}
				}

				// 提取description
				if name == "description" {
					metadata.Description = content
				}

				// 提取Open Graph标签
				if property == "og:title" {
					metadata.OGTitle = content
				}
				if property == "og:description" {
					metadata.OGDesc = content
				}

				// 提取Twitter Card标签
				if name == "twitter:title" && metadata.OGTitle == "" {
					metadata.OGTitle = content
				}
				if name == "twitter:description" && metadata.OGDesc == "" {
					metadata.OGDesc = content
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if isGitHubRepositoryURL(url) {
		if readme := extractGitHubReadme(doc); readme != "" {
			metadata.Readme = readme
		}
	}

	return metadata, nil
}

func isGitHubRepositoryURL(rawURL string) bool {
	parsed, err := neturl.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return false
	}

	if parts[0] == "" || isGitHubReservedPathSegment(parts[0]) {
		return false
	}

	return parts[1] != ""
}

func isGitHubReservedPathSegment(segment string) bool {
	switch strings.ToLower(strings.TrimSpace(segment)) {
	case "about", "apps", "collections", "contact", "events", "explore", "features", "github-copilot",
		"login", "marketplace", "notifications", "pricing", "security", "sessions", "site", "signup",
		"solutions", "sponsors", "stars", "trending", "topics":
		return true
	default:
		return false
	}
}

func extractGitHubReadme(doc *html.Node) string {
	if doc == nil {
		return ""
	}

	readmeRoot := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && hasAttrValue(n, "id", "readme")
	})
	if readmeRoot == nil {
		return ""
	}

	readmeContent := findFirstNode(readmeRoot, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "article" && hasClass(n, "markdown-body")
	})
	if readmeContent == nil {
		readmeContent = findFirstNode(readmeRoot, func(n *html.Node) bool {
			return n.Type == html.ElementNode && hasClass(n, "markdown-body")
		})
	}
	if readmeContent == nil {
		readmeContent = readmeRoot
	}

	readme := renderReadableText(readmeContent)
	readme = normalizeReadmeText(readme)
	return truncateReadmeText(readme, 12000)
}

func findFirstNode(root *html.Node, match func(*html.Node) bool) *html.Node {
	if root == nil {
		return nil
	}

	if match(root) {
		return root
	}

	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstNode(child, match); found != nil {
			return found
		}
	}

	return nil
}

func hasAttrValue(n *html.Node, key, value string) bool {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) && strings.EqualFold(strings.TrimSpace(attr.Val), value) {
			return true
		}
	}

	return false
}

func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if !strings.EqualFold(attr.Key, "class") {
			continue
		}

		for _, part := range strings.Fields(attr.Val) {
			if strings.EqualFold(part, className) {
				return true
			}
		}
	}

	return false
}

func renderReadableText(root *html.Node) string {
	if root == nil {
		return ""
	}

	var buf bytes.Buffer
	renderReadableTextInto(&buf, root, false)
	return buf.String()
}

func renderReadableTextInto(buf *bytes.Buffer, node *html.Node, inPre bool) {
	if node == nil {
		return
	}

	switch node.Type {
	case html.TextNode:
		text := node.Data
		if inPre {
			text = strings.ReplaceAll(text, "\r", "")
			text = strings.TrimRight(text, "\n")
		} else {
			text = strings.Join(strings.Fields(text), " ")
		}

		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		if buf.Len() > 0 {
			last := buf.Bytes()[buf.Len()-1]
			if last != '\n' && last != ' ' {
				buf.WriteByte(' ')
			}
		}

		buf.WriteString(text)
		return
	case html.ElementNode:
		if shouldSkipReadableTextNode(node.Data) {
			return
		}

		nextInPre := inPre || node.Data == "pre"
		block := isReadmeBlockElement(node.Data)
		if node.Data == "br" {
			trimTrailingSpaces(buf)
			appendReadableNewline(buf)
			return
		}

		if block {
			trimTrailingSpaces(buf)
			appendReadableNewline(buf)
		}

		if node.Data == "li" {
			trimTrailingSpaces(buf)
			appendReadableNewline(buf)
			buf.WriteString("- ")
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderReadableTextInto(buf, child, nextInPre)
		}

		if block {
			trimTrailingSpaces(buf)
			appendReadableNewline(buf)
		}
	}
}

func shouldSkipReadableTextNode(tagName string) bool {
	switch strings.ToLower(tagName) {
	case "script", "style", "noscript":
		return true
	default:
		return false
	}
}

func isReadmeBlockElement(tagName string) bool {
	switch strings.ToLower(tagName) {
	case "article", "blockquote", "details", "div", "footer", "h1", "h2", "h3", "h4", "h5", "h6",
		"header", "li", "main", "ol", "p", "pre", "section", "table", "tbody", "td", "tfoot", "th",
		"thead", "tr", "ul":
		return true
	default:
		return false
	}
}

func trimTrailingSpaces(buf *bytes.Buffer) {
	b := buf.Bytes()
	end := len(b)
	for end > 0 {
		switch b[end-1] {
		case ' ', '\t', '\r':
			end--
		default:
			goto done
		}
	}
done:
	if end < len(b) {
		buf.Truncate(end)
	}
}

func appendReadableNewline(buf *bytes.Buffer) {
	if buf.Len() == 0 {
		return
	}

	if last := buf.Bytes()[buf.Len()-1]; last != '\n' {
		buf.WriteByte('\n')
	}
}

func normalizeReadmeText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r", ""))
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if blank {
				continue
			}
			blank = true
			cleaned = append(cleaned, "")
			continue
		}

		blank = false
		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func truncateReadmeText(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}

	return string(runes[:maxRunes]) + "\n[README 内容已截断]"
}
