package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ai-bookmark-service/models"
)

// ValidateBookmarkCreate 验证书签创建请求
func ValidateBookmarkCreate(bm *models.BookmarkCreate) error {
	// 验证 URL
	if bm.URL == "" {
		return fmt.Errorf("URL不能为空")
	}

	// 规范化URL（自动添加协议）
	normalizedURL, err := NormalizeURL(bm.URL)
	if err != nil {
		return fmt.Errorf("无效的URL: %s", bm.URL)
	}
	bm.URL = normalizedURL

	// 验证标题长度
	if len(bm.Title) > 200 {
		return fmt.Errorf("标题过长（最多200字符）")
	}

	// 验证描述长度
	if len(bm.Description) > 1000 {
		return fmt.Errorf("描述过长（最多1000字符）")
	}

	// 验证笔记长度
	if len(bm.Notes) > 2000 {
		return fmt.Errorf("笔记过长（最多2000字符）")
	}

	// 验证标签
	if len(bm.TagNames) > 50 {
		return fmt.Errorf("标签过多（最多50个）")
	}

	for i, tag := range bm.TagNames {
		if len(tag) > 100 {
			return fmt.Errorf("标签名过长: %s（最多100字符）", tag)
		}
		// 自动清理标签首尾空格
		bm.TagNames[i] = strings.TrimSpace(tag)
	}

	return nil
}

// ValidateFolderCreate 验证文件夹创建请求
func ValidateFolderCreate(folder *models.FolderCreate) error {
	if folder.Name == "" {
		return fmt.Errorf("文件夹名称不能为空")
	}

	if len(folder.Name) > 100 {
		return fmt.Errorf("文件夹名称过长（最多100字符）")
	}

	// 验证颜色格式（可选）
	if folder.Color != "" && !isValidColor(folder.Color) {
		return fmt.Errorf("无效的颜色格式: %s", folder.Color)
	}

	return nil
}

// NormalizeURL 规范化URL，自动添加协议前缀
func NormalizeURL(urlStr string) (string, error) {
	// 去除首尾空格
	urlStr = regexp.MustCompile(`^\s+|\s+$`).ReplaceAllString(urlStr, "")

	// 如果URL为空，返回错误
	if urlStr == "" {
		return "", fmt.Errorf("URL不能为空")
	}

	// 检查是否已经包含协议
	hasScheme := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`).MatchString(urlStr)

	if !hasScheme {
		// 自动添加 https:// 协议
		urlStr = "https://" + urlStr
	}

	// 解析URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 验证协议必须是 http 或 https
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("不支持的协议: %s（仅支持http和https）", u.Scheme)
	}

	// 验证必须有主机名
	if u.Host == "" {
		return "", fmt.Errorf("URL缺少主机名")
	}

	return u.String(), nil
}

// isValidColor 验证颜色格式（#RRGGBB）
func isValidColor(color string) bool {
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
	return matched
}
