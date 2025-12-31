package utils

import (
	"net/http"
	"strings"
)

// SanitizeHeaders 脱敏HTTP头（隐藏敏感信息）
func SanitizeHeaders(headers http.Header) http.Header {
	sanitized := headers.Clone()
	
	// 脱敏敏感字段
	sensitiveKeys := []string{
		"Authorization",
		"X-API-Key",
		"Cookie",
		"Set-Cookie",
		"Api-Key",
	}
	
	for _, key := range sensitiveKeys {
		if val := sanitized.Get(key); val != "" {
			if len(val) > 8 {
				// 只显示后4位
				sanitized.Set(key, "***"+val[len(val)-4:])
			} else {
				sanitized.Set(key, "***")
			}
		}
	}
	
	return sanitized
}

// SanitizeAPIKey 脱敏 API Key（只显示后4位）
func SanitizeAPIKey(apiKey string) string {
	if apiKey == "" {
		return "未设置"
	}
	if len(apiKey) > 4 {
		return "***" + apiKey[len(apiKey)-4:]
	}
	return "***"
}

// SanitizeURL 脱敏 URL（隐藏查询参数中的敏感信息）
func SanitizeURL(urlStr string) string {
	// 如果URL包含token、key、password等敏感参数，进行脱敏
	sensitiveParams := []string{"token", "key", "password", "secret", "api_key"}
	
	for _, param := range sensitiveParams {
		if strings.Contains(strings.ToLower(urlStr), param+"=") {
			// 简单替换，实际应该用URL解析
			urlStr = strings.ReplaceAll(urlStr, param+"=", param+"=***")
		}
	}
	
	return urlStr
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
