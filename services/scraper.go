package services

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-bookmark-service/models"

	"golang.org/x/net/html"
)

// ScraperService 网页抓取服务
type ScraperService struct {
	timeout time.Duration
}

// NewScraperService 创建抓取服务
func NewScraperService() *ScraperService {
	return &ScraperService{
		timeout: 30 * time.Second,
	}
}

// ScrapeWebPage 抓取网页元数据
func (s *ScraperService) ScrapeWebPage(url string) (*models.PageMetadata, error) {
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
	client := &http.Client{
		Timeout: s.timeout,
	}
	
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
	
	return metadata, nil
}
