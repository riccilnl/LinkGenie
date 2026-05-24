package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riccilnl/LinkGenie/internal/testsupport"
	"github.com/riccilnl/LinkGenie/models"
)

func TestSystemStatus(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/status", nil)
	fixture.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("预期状态码 200，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("预期 status=ok，实际为 %#v", body["status"])
	}
	if body["database"] != "connected" {
		t.Fatalf("预期 database=connected，实际为 %#v", body["database"])
	}
	if initialized, _ := body["initialized"].(bool); initialized {
		t.Fatalf("空数据库初始化状态应为 false")
	}
}

func TestSystemConfigSaveSuccessAndFailure(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	badPayload := map[string]string{"AI_ENABLED": "true"}
	badRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/system/config", badPayload, fixture.Config.APIToken)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("预期校验失败返回 400，实际为 %d，body=%s", badRec.Code, badRec.Body.String())
	}

	goodPayload := map[string]string{
		"AI_ENABLED":         "true",
		"ENABLE_ASYNC_AI":    "false",
		"AI_API_KEY":         "demo-key",
		"AI_ENDPOINT":        "http://127.0.0.1:18080/v1/chat/completions",
		"AI_MODEL":           "demo-model",
		"AI_WORKER_COUNT":    "2",
		"RATE_LIMIT_ENABLED": "false",
	}
	goodRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/system/config", goodPayload, fixture.Config.APIToken)
	if goodRec.Code != http.StatusOK {
		t.Fatalf("预期保存成功返回 200，实际为 %d，body=%s", goodRec.Code, goodRec.Body.String())
	}

	getRec := performJSONRequest(t, fixture.Handler, http.MethodGet, "/api/system/config", nil, fixture.Config.APIToken)
	if getRec.Code != http.StatusOK {
		t.Fatalf("读取配置失败: %d body=%s", getRec.Code, getRec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&body); err != nil {
		t.Fatalf("解析配置响应失败: %v", err)
	}
	if enabled, _ := body["ai_enabled"].(bool); !enabled {
		t.Fatalf("预期 ai_enabled=true，实际响应: %#v", body)
	}
	if model, _ := body["ai_model"].(string); model != "demo-model" {
		t.Fatalf("预期 ai_model=demo-model，实际为 %q", model)
	}
	if count, _ := body["ai_worker_count"].(float64); int(count) != 2 {
		t.Fatalf("预期 ai_worker_count=2，实际为 %#v", body["ai_worker_count"])
	}
}

func TestCreateBookmarkRoute(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	payload := map[string]interface{}{
		"url":         "example.com/create-route",
		"title":       "Route Test",
		"description": "created by integration test",
		"tag_names":   []string{"go", "api"},
	}
	rec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/", payload, fixture.Config.APIToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("预期创建返回 201，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		ID          int      `json:"id"`
		URL         string   `json:"url"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		TagNames    []string `json:"tag_names"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析创建响应失败: %v", err)
	}

	if body.ID <= 0 {
		t.Fatalf("预期创建后有有效 ID，实际为 %d", body.ID)
	}
	if body.URL != "https://example.com/create-route" {
		t.Fatalf("预期 URL 被规范化，实际为 %q", body.URL)
	}
	if len(body.TagNames) != 2 {
		t.Fatalf("预期标签被保存，实际为 %+v", body.TagNames)
	}
}

func TestCreateBookmarkRouteSupportsLinkdyFormEncoded(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	form := url.Values{}
	form.Set("url", "example.com/linkdy-form")
	form.Set("title", "Linkdy Form")
	form.Set("description", "saved from form client")
	form.Set("tag_names", "ios,mobile")
	form.Set("unread", "true")
	form.Set("is_archived", "1")

	rec := performFormRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/", form, fixture.Config.APIToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("预期 Linkdy 表单创建返回 201，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		ID          int      `json:"id"`
		URL         string   `json:"url"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		IsFavorite  bool     `json:"is_favorite"`
		Unread      bool     `json:"unread"`
		TagNames    []string `json:"tag_names"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析 Linkdy 表单创建响应失败: %v", err)
	}

	if body.ID <= 0 {
		t.Fatalf("预期创建后有有效 ID，实际为 %d", body.ID)
	}
	if body.URL != "https://example.com/linkdy-form" {
		t.Fatalf("预期 URL 被规范化，实际为 %q", body.URL)
	}
	if !body.Unread {
		t.Fatalf("预期 unread=true，实际为 false")
	}
	if !body.IsFavorite {
		t.Fatalf("预期 is_archived 被映射为 is_favorite=true")
	}
	if len(body.TagNames) != 2 || body.TagNames[0] != "ios" || body.TagNames[1] != "mobile" {
		t.Fatalf("预期标签被正确解析，实际为 %+v", body.TagNames)
	}
}

func TestCreateBookmarkRouteReturnsExistingBookmarkForDuplicateMultipartClient(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	existingID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":         "https://example.com/linkdy-duplicate",
		"title":       "Existing Bookmark",
		"description": "already saved",
	})

	form := map[string]string{
		"url":         "example.com/linkdy-duplicate",
		"title":       "Duplicate Attempt",
		"description": "should not overwrite",
	}
	rec := performMultipartFormRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/", form, fixture.Config.APIToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("预期重复表单创建返回 200 幂等成功，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		ID          int    `json:"id"`
		URL         string `json:"url"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析重复创建响应失败: %v", err)
	}

	if body.ID != existingID {
		t.Fatalf("预期返回已存在书签 ID=%d，实际为 %d", existingID, body.ID)
	}
	if body.Title != "Existing Bookmark" {
		t.Fatalf("预期返回原始书签标题，实际为 %q", body.Title)
	}
	if body.Description != "already saved" {
		t.Fatalf("预期返回原始书签描述，实际为 %q", body.Description)
	}

	listRec := performRequest(t, fixture.Handler, http.MethodGet, "/api/bookmarks/?limit=10", nil, fixture.Config.APIToken, "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("查询书签列表失败: %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listBody struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listBody); err != nil {
		t.Fatalf("解析书签列表失败: %v", err)
	}
	if listBody.Count != 1 {
		t.Fatalf("预期重复创建后书签总数仍为 1，实际为 %d", listBody.Count)
	}
}

func TestCheckBookmarkRouteReturnsLinkdyContractForExistingBookmark(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	bookmarkID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":         "https://example.com/already-saved",
		"title":       "Already Saved",
		"description": "existing bookmark description",
		"tag_names":   []string{"existing"},
	})

	rec := performRequest(
		t,
		fixture.Handler,
		http.MethodGet,
		"/api/bookmarks/check/?url="+url.QueryEscape("https://example.com/already-saved"),
		nil,
		fixture.Config.APIToken,
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("预期检查书签返回 200，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Bookmark *struct {
			ID          int      `json:"id"`
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			TagNames    []string `json:"tag_names"`
		} `json:"bookmark"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析检查书签响应失败: %v", err)
	}

	if body.Bookmark == nil {
		t.Fatalf("预期返回 bookmark 对象，实际为空: %+v", body)
	}
	if body.Bookmark.ID != bookmarkID {
		t.Fatalf("预期返回已存在书签 ID=%d，实际为 %d", bookmarkID, body.Bookmark.ID)
	}
	if body.Bookmark.URL != "https://example.com/already-saved" {
		t.Fatalf("预期返回规范化 URL，实际为 %q", body.Bookmark.URL)
	}
	if body.Metadata["description"] != "existing bookmark description" {
		t.Fatalf("预期 metadata.description 被返回，实际为 %+v", body.Metadata)
	}
}

func TestTagsRouteSupportsTrailingSlashAndWrappedResponse(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":       "https://example.com/tag-a",
		"title":     "Tag A",
		"tag_names": []string{"apple", "banana"},
	})
	createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":       "https://example.com/tag-b",
		"title":     "Tag B",
		"tag_names": []string{"banana", "citrus"},
	})

	rec := performRequest(t, fixture.Handler, http.MethodGet, "/api/tags/?q=an&limit=1", nil, fixture.Config.APIToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("预期查询标签返回 200，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Count   int `json:"count"`
		Results []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析标签响应失败: %v", err)
	}

	if body.Count != 1 {
		t.Fatalf("预期模糊过滤后 count=1，实际为 %d", body.Count)
	}
	if len(body.Results) != 1 || body.Results[0].Name != "banana" {
		t.Fatalf("预期返回 banana 标签，实际为 %+v", body.Results)
	}
}

func TestImportBookmarksHTMLRoute(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	html := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
    <DT><H3>Inbox</H3>
    <DL><p>
        <DT><A HREF="https://example.com/imported" TAGS="ios,reading">Imported Bookmark</A>
        <DD>Imported from integration test
    </DL><p>
</DL><p>`

	rec := performMultipartFileRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/import/", "file", "bookmarks.html", html, fixture.Config.APIToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("导入书签失败: %d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Success int      `json:"success"`
		Failed  int      `json:"failed"`
		Skipped int      `json:"skipped"`
		Errors  []string `json:"errors"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析导入响应失败: %v", err)
	}
	if body.Success != 1 || body.Failed != 0 {
		t.Fatalf("导入统计不符合预期: %+v", body)
	}

	bookmarks, err := fixture.App.BookmarkRepository().List(10, 0, map[string]interface{}{})
	if err != nil {
		t.Fatalf("查询导入后的书签失败: %v", err)
	}
	if len(bookmarks) != 1 || bookmarks[0].URL != "https://example.com/imported" {
		t.Fatalf("导入后的书签不符合预期: %+v", bookmarks)
	}

	folders, err := fixture.App.FolderRepository().List()
	if err != nil {
		t.Fatalf("查询导入后的文件夹失败: %v", err)
	}
	if len(folders) != 1 || folders[0].Name != "Inbox" {
		t.Fatalf("导入后的文件夹不符合预期: %+v", folders)
	}
}

func TestExportBookmarksHTMLRoute(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	folderID := createFolderViaAPI(t, fixture, map[string]interface{}{
		"name": "Export Folder",
	})
	bookmarkID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":         "https://example.com/exported",
		"title":       "Export Title",
		"description": "Export Description",
		"tag_names":   []string{"export", "html"},
	})

	addRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/folders", map[string]interface{}{
		"folder_ids": []int{folderID},
	}, fixture.Config.APIToken)
	if addRec.Code != http.StatusNoContent {
		t.Fatalf("准备导出数据时添加文件夹失败: %d body=%s", addRec.Code, addRec.Body.String())
	}

	rec := performRequest(t, fixture.Handler, http.MethodGet, "/api/bookmarks/export/", nil, fixture.Config.APIToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("导出书签失败: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("预期导出 content-type 为 text/html，实际为 %q", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE NETSCAPE-Bookmark-file-1>") {
		t.Fatalf("导出结果缺少 Netscape 头: %s", body)
	}
	if !strings.Contains(body, "https://example.com/exported") {
		t.Fatalf("导出结果缺少书签 URL: %s", body)
	}
	if !strings.Contains(body, "Export Folder") {
		t.Fatalf("导出结果缺少文件夹名称: %s", body)
	}
}

func TestEnhanceBookmarkReturns503WhenAIDisabled(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})
	bookmarkID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":   "https://example.com/no-ai",
		"title": "no ai",
	})

	rec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/enhance/", nil, fixture.Config.APIToken)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("预期 AI 未启用返回 503，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}
}

func TestEnhanceBookmarkSucceedsWithFakeAI(t *testing.T) {
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>GitHub - demo/project: English title · GitHub</title><meta name="description" content="Original English description from page"></head><body>demo</body></html>`))
	}))
	defer pageServer.Close()

	var (
		mu          sync.Mutex
		capturedMsg string
	)
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("解析 fake AI 请求失败: %v", err)
		}
		if len(reqBody.Messages) > 0 {
			mu.Lock()
			capturedMsg = reqBody.Messages[0].Content
			mu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"项目中文标题\",\"description\":\"这是经过 AI 整理后的中文摘要，用于验证异步增强链路是否成功回写。\",\"tags\":[\"开发工具\",\"GitHub\",\"项目\"]}"}}]}`))
	}))
	defer aiServer.Close()

	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{
		AIEnabled:     true,
		EnableAsyncAI: true,
		AIAPIKey:      "demo-key",
		AIEndpoint:    aiServer.URL + "/v1/chat/completions",
		AIModel:       "fake-model",
		AIWorkerCount: 1,
	})

	bookmark, err := fixture.App.BookmarkRepository().Create(&models.BookmarkCreate{
		URL:         pageServer.URL,
		Title:       "Old English Title",
		Description: "Old English Description",
	})
	if err != nil {
		t.Fatalf("创建初始书签失败: %v", err)
	}
	bookmarkID := bookmark.ID

	rec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/enhance/", nil, fixture.Config.APIToken)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("预期触发成功返回 202，实际为 %d，body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		bookmark, err := fixture.App.BookmarkRepository().GetByID(bookmarkID)
		if err == nil && bookmark.Title == "项目中文标题" && strings.Contains(bookmark.Description, "中文摘要") {
			if len(bookmark.TagNames) < 3 {
				t.Fatalf("预期 AI 标签已回写，实际为 %+v", bookmark.TagNames)
			}

			mu.Lock()
			prompt := capturedMsg
			mu.Unlock()
			if !strings.Contains(prompt, "当前书签标题: Old English Title") {
				t.Fatalf("预期 prompt 包含旧标题，实际为: %s", prompt)
			}
			if !strings.Contains(prompt, "当前书签描述: Old English Description") {
				t.Fatalf("预期 prompt 包含旧描述，实际为: %s", prompt)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	bookmark, err = fixture.App.BookmarkRepository().GetByID(bookmarkID)
	if err != nil {
		t.Fatalf("查询增强后的书签失败: %v", err)
	}
	t.Fatalf("在超时前未观察到 AI 回写，当前书签=%+v", bookmark)
}

func TestFolderRoutesCRUDAndMembership(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	folderPayload := map[string]interface{}{
		"name":  "Inbox",
		"color": "#0a84ff",
		"icon":  "📥",
	}
	folderID := createFolderViaAPI(t, fixture, folderPayload)

	listRec := performJSONRequest(t, fixture.Handler, http.MethodGet, "/api/folders/", nil, fixture.Config.APIToken)
	if listRec.Code != http.StatusOK {
		t.Fatalf("获取文件夹列表失败: %d body=%s", listRec.Code, listRec.Body.String())
	}

	var folders []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&folders); err != nil {
		t.Fatalf("解析文件夹列表失败: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("预期 1 个文件夹，实际为 %d", len(folders))
	}

	updatePayload := map[string]interface{}{
		"name":  "Inbox Updated",
		"color": "#34c759",
		"icon":  "🗂️",
	}
	updateRec := performJSONRequest(t, fixture.Handler, http.MethodPut, "/api/folders/"+strconv.Itoa(folderID), updatePayload, fixture.Config.APIToken)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("更新文件夹失败: %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	bookmarkID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":   "https://example.com/folder-member",
		"title": "folder member",
	})
	addRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/folders", map[string]interface{}{
		"folder_ids": []int{folderID},
	}, fixture.Config.APIToken)
	if addRec.Code != http.StatusNoContent {
		t.Fatalf("添加书签到文件夹失败: %d body=%s", addRec.Code, addRec.Body.String())
	}

	bookmarksRec := performJSONRequest(t, fixture.Handler, http.MethodGet, "/api/folders/"+strconv.Itoa(folderID)+"/bookmarks", nil, fixture.Config.APIToken)
	if bookmarksRec.Code != http.StatusOK {
		t.Fatalf("获取文件夹书签失败: %d body=%s", bookmarksRec.Code, bookmarksRec.Body.String())
	}

	var bookmarksBody struct {
		Count   int `json:"count"`
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}
	if err := json.NewDecoder(bookmarksRec.Body).Decode(&bookmarksBody); err != nil {
		t.Fatalf("解析文件夹书签响应失败: %v", err)
	}
	if bookmarksBody.Count != 1 || len(bookmarksBody.Results) != 1 || bookmarksBody.Results[0].ID != bookmarkID {
		t.Fatalf("文件夹书签结果不符合预期: %+v", bookmarksBody)
	}

	removeRec := performJSONRequest(t, fixture.Handler, http.MethodDelete, "/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/folders/"+strconv.Itoa(folderID), nil, fixture.Config.APIToken)
	if removeRec.Code != http.StatusNoContent {
		t.Fatalf("移除文件夹关系失败: %d body=%s", removeRec.Code, removeRec.Body.String())
	}

	deleteRec := performJSONRequest(t, fixture.Handler, http.MethodDelete, "/api/folders/"+strconv.Itoa(folderID), nil, fixture.Config.APIToken)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("删除文件夹失败: %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestWorkflowRoutesCreateToggleApplyAndDelete(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	folderID := createFolderViaAPI(t, fixture, map[string]interface{}{
		"name": "Workflow Target",
	})
	bookmarkID := createBookmarkViaAPI(t, fixture, map[string]interface{}{
		"url":   "https://example.com/workflow-route",
		"title": "workflow route",
	})

	createWorkflowPayload := map[string]interface{}{
		"name":            "move matched bookmark",
		"description":     "integration test workflow",
		"enabled":         true,
		"condition_logic": "OR",
		"triggers": []map[string]interface{}{
			{
				"trigger_type": "url_match",
				"config": map[string]interface{}{
					"match_mode": "contains",
					"value":      "workflow-route",
				},
			},
		},
		"actions": []map[string]interface{}{
			{
				"action_type": "move_to_folder",
				"config": map[string]interface{}{
					"folder_id": float64(folderID),
				},
			},
		},
	}

	createRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/workflows/", createWorkflowPayload, fixture.Config.APIToken)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("创建工作流失败: %d body=%s", createRec.Code, createRec.Body.String())
	}

	var workflowBody struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&workflowBody); err != nil {
		t.Fatalf("解析创建工作流响应失败: %v", err)
	}
	if workflowBody.ID <= 0 {
		t.Fatalf("预期工作流 ID 有效，实际为 %d", workflowBody.ID)
	}

	listRec := performJSONRequest(t, fixture.Handler, http.MethodGet, "/api/workflows/", nil, fixture.Config.APIToken)
	if listRec.Code != http.StatusOK {
		t.Fatalf("获取工作流列表失败: %d body=%s", listRec.Code, listRec.Body.String())
	}

	toggleRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/workflows/"+strconv.Itoa(workflowBody.ID)+"/toggle", nil, fixture.Config.APIToken)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("切换工作流失败: %d body=%s", toggleRec.Code, toggleRec.Body.String())
	}
	var toggledBody struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(toggleRec.Body).Decode(&toggledBody); err != nil {
		t.Fatalf("解析切换工作流响应失败: %v", err)
	}
	if toggledBody.Enabled {
		t.Fatalf("预期切换后工作流被禁用")
	}

	// 重新开启后再执行 apply。
	toggleRec = performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/workflows/"+strconv.Itoa(workflowBody.ID)+"/toggle", nil, fixture.Config.APIToken)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("重新启用工作流失败: %d body=%s", toggleRec.Code, toggleRec.Body.String())
	}

	applyRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/workflows/apply", map[string]interface{}{
		"workflow_ids": []int{workflowBody.ID},
		"bookmark_ids": []int{bookmarkID},
	}, fixture.Config.APIToken)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("应用工作流失败: %d body=%s", applyRec.Code, applyRec.Body.String())
	}

	folders, err := fixture.App.FolderRepository().GetBookmarkFolders(bookmarkID)
	if err != nil {
		t.Fatalf("查询工作流执行后的文件夹失败: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != folderID {
		t.Fatalf("预期工作流把书签移入目标文件夹，实际为 %+v", folders)
	}

	deleteRec := performJSONRequest(t, fixture.Handler, http.MethodDelete, "/api/workflows/"+strconv.Itoa(workflowBody.ID), nil, fixture.Config.APIToken)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("删除工作流失败: %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestTagStatsAndOptimizePreview(t *testing.T) {
	fixture := testsupport.NewAppFixture(t, testsupport.AppOptions{})

	for i := 0; i < 3; i++ {
		createBookmarkViaAPI(t, fixture, map[string]interface{}{
			"url":       "https://example.com/tag-preview-" + strconv.Itoa(i),
			"title":     "tag preview",
			"tag_names": []string{"go"},
		})
	}

	statsRec := performJSONRequest(t, fixture.Handler, http.MethodGet, "/api/tags/stats", nil, fixture.Config.APIToken)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("获取标签统计失败: %d body=%s", statsRec.Code, statsRec.Body.String())
	}

	var statsBody map[string]interface{}
	if err := json.NewDecoder(statsRec.Body).Decode(&statsBody); err != nil {
		t.Fatalf("解析标签统计失败: %v", err)
	}
	if total, _ := statsBody["total"].(float64); int(total) != 1 {
		t.Fatalf("预期总标签数为 1，实际为 %#v", statsBody["total"])
	}

	optimizeRec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/tags/optimize", map[string]interface{}{
		"dry_run":          true,
		"enable_merge":     false,
		"enable_promotion": true,
	}, fixture.Config.APIToken)
	if optimizeRec.Code != http.StatusOK {
		t.Fatalf("预览标签优化失败: %d body=%s", optimizeRec.Code, optimizeRec.Body.String())
	}

	var optimizeBody struct {
		Preview bool `json:"preview"`
		Actions []struct {
			Type string `json:"type"`
			Tag  string `json:"tag"`
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"actions"`
		Summary struct {
			TotalPromotions int `json:"total_promotions"`
		} `json:"summary"`
	}
	if err := json.NewDecoder(optimizeRec.Body).Decode(&optimizeBody); err != nil {
		t.Fatalf("解析标签优化响应失败: %v", err)
	}
	if !optimizeBody.Preview {
		t.Fatalf("预期 dry_run 返回 preview=true")
	}
	if optimizeBody.Summary.TotalPromotions != 1 {
		t.Fatalf("预期有 1 个晋升动作，实际为 %d", optimizeBody.Summary.TotalPromotions)
	}
	if len(optimizeBody.Actions) == 0 || optimizeBody.Actions[0].Tag != "go" || optimizeBody.Actions[0].To != "dynamic" {
		t.Fatalf("标签优化动作不符合预期: %+v", optimizeBody.Actions)
	}
}

func createBookmarkViaAPI(t *testing.T, fixture *testsupport.AppFixture, payload map[string]interface{}) int {
	t.Helper()
	rec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/bookmarks/", payload, fixture.Config.APIToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建测试书签失败: %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析创建书签响应失败: %v", err)
	}
	return body.ID
}

func createFolderViaAPI(t *testing.T, fixture *testsupport.AppFixture, payload map[string]interface{}) int {
	t.Helper()
	rec := performJSONRequest(t, fixture.Handler, http.MethodPost, "/api/folders/", payload, fixture.Config.APIToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建测试文件夹失败: %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("解析创建文件夹响应失败: %v", err)
	}
	return body.ID
}

func performJSONRequest(t *testing.T, handler http.Handler, method, target string, payload interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("序列化请求失败: %v", err)
		}
	}

	var reader io.Reader
	if payload != nil {
		reader = &body
	}
	return performRequest(t, handler, method, target, reader, token, "application/json")
}

func performFormRequest(t *testing.T, handler http.Handler, method, target string, form url.Values, token string) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(form.Encode())
	return performRequest(t, handler, method, target, body, token, "application/x-www-form-urlencoded")
}

func performMultipartFormRequest(t *testing.T, handler http.Handler, method, target string, form map[string]string, token string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range form {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("写入 multipart 字段失败: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}

	return performRequest(t, handler, method, target, &body, token, writer.FormDataContentType())
}

func performMultipartFileRequest(t *testing.T, handler http.Handler, method, target, fieldName, filename, content, token string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("创建 multipart 文件字段失败: %v", err)
	}
	if _, err := fileWriter.Write([]byte(content)); err != nil {
		t.Fatalf("写入 multipart 文件内容失败: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}

	return performRequest(t, handler, method, target, &body, token, writer.FormDataContentType())
}

func performRequest(t *testing.T, handler http.Handler, method, target string, body io.Reader, token, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, body)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
