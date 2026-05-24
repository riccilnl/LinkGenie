package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/riccilnl/LinkGenie/services"
)

func parseCreateBookmarkInput(r *http.Request) (services.CreateBookmarkInput, error) {
	var input services.CreateBookmarkInput
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := parseBookmarkForm(r, contentType); err != nil {
			log.Printf("❌ 解析表单失败: %v", err)
			return input, errors.New("无效的表单数据")
		}

		input.URL = firstNonEmptyFormValue(r, "url")
		input.Title = firstNonEmptyFormValue(r, "title")
		input.Description = firstNonEmptyFormValue(r, "description")
		input.Notes = firstNonEmptyFormValue(r, "notes")
		input.Unread = isTruthyFormValue(firstNonEmptyFormValue(r, "unread"))
		input.Shared = isTruthyFormValue(firstNonEmptyFormValue(r, "shared"))
		input.IsArchived = isTruthyFormValue(firstNonEmptyFormValue(r, "is_archived"))
		input.IsFavorite = isTruthyFormValue(firstNonEmptyFormValue(r, "is_favorite"))
		input.TagNames = parseFlexibleTagValues(formValues(r, "tag_names", "tag_names[]", "tags", "tags[]"))
		input.FolderIDs = parseFlexibleIntValues(formValues(r, "folder_ids", "folder_ids[]", "folder_id"))
		return input, nil
	}

	raw, err := readRawJSONMap(r)
	if err != nil {
		return input, err
	}

	input.URL = readString(raw, "url")
	input.Title = readString(raw, "title")
	input.Description = readString(raw, "description")
	input.Notes = readString(raw, "notes")
	input.IsFavorite = readBool(raw, "is_favorite")
	input.Unread = readBool(raw, "unread")
	input.Shared = readBool(raw, "shared")
	input.IsArchived = readBool(raw, "is_archived")
	input.TagNames = readStringSlice(raw, "tag_names")
	if len(input.TagNames) == 0 {
		input.TagNames = readStringSlice(raw, "tags")
	}
	input.FolderIDs = readIntSlice(raw, "folder_ids")
	if len(input.FolderIDs) == 0 {
		if folderID, ok := readInt(raw, "folder_id"); ok && folderID > 0 {
			input.FolderIDs = []int{folderID}
		}
	}

	return input, nil
}

func parseReplaceBookmarkInput(r *http.Request) (services.ReplaceBookmarkInput, error) {
	var input services.ReplaceBookmarkInput
	raw, err := readRawJSONMap(r)
	if err != nil {
		return input, err
	}

	input.URL = readString(raw, "url")
	input.Title = readString(raw, "title")
	input.Description = readString(raw, "description")
	input.Notes = readString(raw, "notes")
	input.IsFavorite = readBool(raw, "is_favorite")
	input.Unread = readBool(raw, "unread")
	input.Shared = readBool(raw, "shared")
	input.IsArchived = readBool(raw, "is_archived")
	input.TagNames = readStringSlice(raw, "tag_names")
	if len(input.TagNames) == 0 {
		input.TagNames = readStringSlice(raw, "tags")
	}
	input.FolderIDs = readIntSlice(raw, "folder_ids")
	if len(input.FolderIDs) == 0 {
		if folderID, ok := readInt(raw, "folder_id"); ok && folderID > 0 {
			input.FolderIDs = []int{folderID}
		}
	}

	return input, nil
}

func parsePatchBookmarkInput(r *http.Request) (services.PatchBookmarkInput, error) {
	raw, err := readRawJSONMap(r)
	if err != nil {
		return services.PatchBookmarkInput{}, err
	}

	input := services.PatchBookmarkInput{}

	if v, ok := raw["url"]; ok {
		value, parseErr := decodeString(v)
		if parseErr != nil {
			return input, errors.New("url 字段格式无效")
		}
		input.URL = &value
	}
	if v, ok := raw["title"]; ok {
		value, parseErr := decodeString(v)
		if parseErr != nil {
			return input, errors.New("title 字段格式无效")
		}
		input.Title = &value
	}
	if v, ok := raw["description"]; ok {
		value, parseErr := decodeString(v)
		if parseErr != nil {
			return input, errors.New("description 字段格式无效")
		}
		input.Description = &value
	}
	if v, ok := raw["notes"]; ok {
		value, parseErr := decodeString(v)
		if parseErr != nil {
			return input, errors.New("notes 字段格式无效")
		}
		input.Notes = &value
	}
	if v, ok := raw["is_favorite"]; ok {
		value, parseErr := decodeBool(v)
		if parseErr != nil {
			return input, errors.New("is_favorite 字段格式无效")
		}
		input.IsFavorite = &value
	}
	if v, ok := raw["unread"]; ok {
		value, parseErr := decodeBool(v)
		if parseErr != nil {
			return input, errors.New("unread 字段格式无效")
		}
		input.Unread = &value
	}
	if v, ok := raw["shared"]; ok {
		value, parseErr := decodeBool(v)
		if parseErr != nil {
			return input, errors.New("shared 字段格式无效")
		}
		input.Shared = &value
	}
	if v, ok := raw["is_archived"]; ok {
		value, parseErr := decodeBool(v)
		if parseErr != nil {
			return input, errors.New("is_archived 字段格式无效")
		}
		input.IsArchived = &value
	}
	if v, ok := raw["tag_names"]; ok {
		value, parseErr := decodeStringSlice(v)
		if parseErr != nil {
			return input, errors.New("tag_names 字段格式无效")
		}
		input.TagNames = &value
	} else if v, ok := raw["tags"]; ok {
		value, parseErr := decodeStringSlice(v)
		if parseErr != nil {
			return input, errors.New("tags 字段格式无效")
		}
		input.TagNames = &value
	}
	if v, ok := raw["folder_ids"]; ok {
		value, parseErr := decodeIntSlice(v)
		if parseErr != nil {
			return input, errors.New("folder_ids 字段格式无效")
		}
		input.FolderIDs = &value
	} else if v, ok := raw["folder_id"]; ok {
		value, parseErr := decodeInt(v)
		if parseErr != nil {
			return input, errors.New("folder_id 字段格式无效")
		}
		folderIDs := []int{value}
		input.FolderIDs = &folderIDs
	}

	return input, nil
}

func readRawJSONMap(r *http.Request) (map[string]json.RawMessage, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 限制 1MB
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("请求体过大或读取失败")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		log.Printf("❌ JSON解析失败: %v, Body: %s", err, string(bodyBytes))
		return nil, errors.New("无效的JSON数据")
	}

	return raw, nil
}

func readString(raw map[string]json.RawMessage, key string) string {
	if v, ok := raw[key]; ok {
		value, err := decodeString(v)
		if err == nil {
			return value
		}
	}
	return ""
}

func readBool(raw map[string]json.RawMessage, key string) bool {
	if v, ok := raw[key]; ok {
		value, err := decodeBool(v)
		if err == nil {
			return value
		}
	}
	return false
}

func readStringSlice(raw map[string]json.RawMessage, key string) []string {
	if v, ok := raw[key]; ok {
		value, err := decodeStringSlice(v)
		if err == nil {
			return value
		}
	}
	return nil
}

func readIntSlice(raw map[string]json.RawMessage, key string) []int {
	if v, ok := raw[key]; ok {
		value, err := decodeIntSlice(v)
		if err == nil {
			return value
		}
	}
	return nil
}

func readInt(raw map[string]json.RawMessage, key string) (int, bool) {
	if v, ok := raw[key]; ok {
		value, err := decodeInt(v)
		if err == nil {
			return value, true
		}
	}
	return 0, false
}

func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func decodeBool(raw json.RawMessage) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, err
	}
	return value, nil
}

func decodeStringSlice(raw json.RawMessage) ([]string, error) {
	var slice []string
	if err := json.Unmarshal(raw, &slice); err == nil {
		return slice, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return parseCommaSeparatedTags(value), nil
}

func decodeIntSlice(raw json.RawMessage) ([]int, error) {
	var slice []int
	if err := json.Unmarshal(raw, &slice); err == nil {
		return slice, nil
	}

	var stringSlice []string
	if err := json.Unmarshal(raw, &stringSlice); err == nil {
		result := make([]int, 0, len(stringSlice))
		for _, item := range stringSlice {
			value, err := strconv.Atoi(strings.TrimSpace(item))
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	}

	var single int
	if err := json.Unmarshal(raw, &single); err == nil {
		return []int{single}, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return parseCommaSeparatedInts(value), nil
}

func decodeInt(raw json.RawMessage) (int, error) {
	var value int
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}

	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(stringValue))
}

func parseCommaSeparatedTags(tagNames string) []string {
	if tagNames == "" {
		return nil
	}

	parts := strings.Split(tagNames, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

func parseCommaSeparatedInts(values string) []int {
	if values == "" {
		return nil
	}

	parts := strings.Split(values, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		value, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		result = append(result, value)
	}

	return result
}

func isTruthyFormValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseBookmarkForm(r *http.Request, contentType string) error {
	if strings.Contains(contentType, "multipart/form-data") {
		return r.ParseMultipartForm(32 << 20)
	}
	return r.ParseForm()
}

func firstNonEmptyFormValue(r *http.Request, keys ...string) string {
	for _, value := range formValues(r, keys...) {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formValues(r *http.Request, keys ...string) []string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, r.Form[key]...)
	}
	return values
}

func parseFlexibleTagValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if strings.HasPrefix(value, "[") {
			var jsonValues []string
			if err := json.Unmarshal([]byte(value), &jsonValues); err == nil {
				result = append(result, jsonValues...)
				continue
			}
		}

		if strings.Contains(value, ",") {
			result = append(result, parseCommaSeparatedTags(value)...)
			continue
		}

		result = append(result, value)
	}
	return result
}

func parseFlexibleIntValues(values []string) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if strings.HasPrefix(value, "[") {
			var jsonInts []int
			if err := json.Unmarshal([]byte(value), &jsonInts); err == nil {
				result = append(result, jsonInts...)
				continue
			}

			var jsonStrings []string
			if err := json.Unmarshal([]byte(value), &jsonStrings); err == nil {
				for _, item := range jsonStrings {
					parsed, err := strconv.Atoi(strings.TrimSpace(item))
					if err == nil {
						result = append(result, parsed)
					}
				}
				continue
			}
		}

		if strings.Contains(value, ",") {
			result = append(result, parseCommaSeparatedInts(value)...)
			continue
		}

		parsed, err := strconv.Atoi(value)
		if err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

func writeBookmarkServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case services.IsConflictError(err):
		http.Error(w, err.Error(), http.StatusConflict)
	case services.IsNotFoundError(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case services.IsUnavailableError(err):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fallback, http.StatusInternalServerError)
	}
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "无效") ||
		strings.Contains(message, "不能为空") ||
		strings.Contains(message, "过长") ||
		strings.Contains(message, "未设置") ||
		strings.Contains(message, "请设置")
}
