package services

import (
	"bytes"
	"context"
	"fmt"
	stdhtml "html"
	"io"
	"sort"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"

	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/models"
)

const folderPathSeparator = " / "

// BookmarkImportResult 描述 HTML 导入结果。
type BookmarkImportResult struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// BookmarkExchangeService 负责书签 HTML 导入导出。
type BookmarkExchangeService struct {
	bookmarkSvc  *BookmarkService
	bookmarkRepo *db.BookmarkRepository
	folderRepo   *db.FolderRepository
}

// NewBookmarkExchangeService 创建书签交换服务。
func NewBookmarkExchangeService(bookmarkSvc *BookmarkService, bookmarkRepo *db.BookmarkRepository, folderRepo *db.FolderRepository) *BookmarkExchangeService {
	return &BookmarkExchangeService{
		bookmarkSvc:  bookmarkSvc,
		bookmarkRepo: bookmarkRepo,
		folderRepo:   folderRepo,
	}
}

type bookmarkImportEntry struct {
	URL         string
	Title       string
	Description string
	TagNames    []string
	FolderNames []string
}

type exportBookmarkEntry struct {
	bookmark    *models.Bookmark
	folderNames []string
}

type exportFolderNode struct {
	Name     string
	Children map[string]*exportFolderNode
	Entries  []*exportBookmarkEntry
}

// ImportHTML 导入 Netscape 书签 HTML。
func (s *BookmarkExchangeService) ImportHTML(ctx context.Context, reader io.Reader) (*BookmarkImportResult, error) {
	if s == nil || s.bookmarkSvc == nil {
		return nil, fmt.Errorf("书签导入服务不可用")
	}

	entries, err := parseBookmarkHTML(reader)
	if err != nil {
		return nil, err
	}

	result := &BookmarkImportResult{}
	folderIDsByName, err := s.loadFolderCache()
	if err != nil {
		return nil, err
	}

	for index, entry := range entries {
		if strings.TrimSpace(entry.URL) == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 项缺少 URL", index+1))
			continue
		}

		folderIDs, folderErr := s.ensureFolderIDs(entry.FolderNames, folderIDsByName)
		if folderErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 项文件夹处理失败: %v", index+1, folderErr))
			continue
		}

		_, createErr := s.bookmarkSvc.CreateBookmark(ctx, CreateBookmarkInput{
			URL:         entry.URL,
			Title:       entry.Title,
			Description: entry.Description,
			TagNames:    entry.TagNames,
			FolderIDs:   folderIDs,
		})
		switch {
		case createErr == nil:
			result.Success++
		case IsConflictError(createErr):
			result.Skipped++
		case true:
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("导入 %s 失败: %v", entry.URL, createErr))
		}
	}

	return result, nil
}

// ExportHTML 导出 Netscape 书签 HTML。
func (s *BookmarkExchangeService) ExportHTML(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.bookmarkRepo == nil {
		return nil, fmt.Errorf("书签导出服务不可用")
	}

	count, err := s.bookmarkRepo.Count(nil)
	if err != nil {
		return nil, fmt.Errorf("统计书签数量失败: %w", err)
	}

	bookmarks := []*models.Bookmark{}
	if count > 0 {
		bookmarks, err = s.bookmarkRepo.List(count, 0, map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("查询书签失败: %w", err)
		}
	}

	entries := make([]*exportBookmarkEntry, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		folderNames := []string{}
		if s.folderRepo != nil {
			folders, folderErr := s.folderRepo.GetBookmarkFolders(bookmark.ID)
			if folderErr != nil {
				return nil, fmt.Errorf("查询书签文件夹失败: %w", folderErr)
			}
			for _, folder := range folders {
				folderNames = append(folderNames, folder.Name)
			}
		}
		entries = append(entries, &exportBookmarkEntry{bookmark: bookmark, folderNames: folderNames})
	}

	rootEntries, folderTree := buildExportTree(entries)

	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n")
	buf.WriteString("<!-- This is an automatically generated file.\n")
	buf.WriteString("     It will be read and overwritten.\n")
	buf.WriteString("     DO NOT EDIT! -->\n")
	buf.WriteString("<META HTTP-EQUIV=\"Content-Type\" CONTENT=\"text/html; charset=UTF-8\">\n")
	buf.WriteString("<TITLE>Bookmarks</TITLE>\n")
	buf.WriteString("<H1>Bookmarks</H1>\n")
	buf.WriteString("<DL><p>\n")

	for _, entry := range rootEntries {
		writeExportBookmark(&buf, entry, "    ")
	}
	writeExportFolders(&buf, folderTree, "    ")

	buf.WriteString("</DL><p>\n")
	return buf.Bytes(), nil
}

func (s *BookmarkExchangeService) loadFolderCache() (map[string]int, error) {
	folderIDsByName := make(map[string]int)
	if s.folderRepo == nil {
		return folderIDsByName, nil
	}

	folders, err := s.folderRepo.List()
	if err != nil {
		return nil, fmt.Errorf("查询文件夹失败: %w", err)
	}
	for _, folder := range folders {
		folderIDsByName[folder.Name] = folder.ID
	}
	return folderIDsByName, nil
}

func (s *BookmarkExchangeService) ensureFolderIDs(folderNames []string, folderIDsByName map[string]int) ([]int, error) {
	if s.folderRepo == nil || len(folderNames) == 0 {
		return nil, nil
	}

	folderIDs := make([]int, 0, len(folderNames))
	for _, folderName := range uniqueStrings(folderNames) {
		folderName = strings.TrimSpace(folderName)
		if folderName == "" {
			continue
		}
		if folderID, ok := folderIDsByName[folderName]; ok {
			folderIDs = append(folderIDs, folderID)
			continue
		}

		folder, err := s.folderRepo.Create(&models.FolderCreate{Name: folderName})
		if err != nil {
			return nil, fmt.Errorf("创建文件夹 %q 失败: %w", folderName, err)
		}
		folderIDsByName[folderName] = folder.ID
		folderIDs = append(folderIDs, folder.ID)
	}
	return folderIDs, nil
}

func parseBookmarkHTML(reader io.Reader) ([]bookmarkImportEntry, error) {
	doc, err := xhtml.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("HTML 解析失败: %w", err)
	}

	rootDL := findFirstElement(doc, "dl")
	if rootDL == nil {
		return nil, fmt.Errorf("不是有效的 Netscape 书签 HTML")
	}

	entries := []bookmarkImportEntry{}
	parseBookmarkDL(rootDL, nil, &entries)
	return entries, nil
}

func parseBookmarkDL(dl *xhtml.Node, folderStack []string, entries *[]bookmarkImportEntry) {
	for node := dl.FirstChild; node != nil; {
		next := node.NextSibling
		if node.Type == xhtml.ElementNode && node.Data == "dt" {
			if folderNode := findFirstDescendantElement(node, "h3"); folderNode != nil {
				folderName := strings.TrimSpace(extractNodeText(folderNode))
				if folderName != "" {
					childDL := findFirstDescendantElement(node, "dl")
					if childDL == nil {
						childDL = nextElementSibling(node)
					}
					if childDL != nil && childDL.Data == "dl" {
						parseBookmarkDL(childDL, append(folderStack, folderName), entries)
						if childDL.Parent == node.Parent {
							next = childDL.NextSibling
						}
					}
				}
			} else if linkNode := findFirstDescendantElement(node, "a"); linkNode != nil {
				entry := parseBookmarkLink(linkNode)
				if len(entry.FolderNames) == 0 && len(folderStack) > 0 {
					entry.FolderNames = []string{strings.Join(folderStack, folderPathSeparator)}
				}
				if descriptionNode := nextElementSibling(node); descriptionNode != nil && descriptionNode.Data == "dd" {
					entry.Description = strings.TrimSpace(extractNodeText(descriptionNode))
				}
				*entries = append(*entries, entry)
			}
		} else if node.Type == xhtml.ElementNode {
			parseBookmarkDL(node, folderStack, entries)
		}
		node = next
	}
}

func parseBookmarkLink(linkNode *xhtml.Node) bookmarkImportEntry {
	entry := bookmarkImportEntry{
		Title: strings.TrimSpace(extractNodeText(linkNode)),
	}
	for _, attr := range linkNode.Attr {
		switch strings.ToLower(attr.Key) {
		case "href":
			entry.URL = strings.TrimSpace(attr.Val)
		case "tags":
			entry.TagNames = splitAndTrim(attr.Val, ",")
		case "linkgenie_folders":
			entry.FolderNames = splitAndTrim(attr.Val, "||")
		}
	}
	return entry
}

func buildExportTree(entries []*exportBookmarkEntry) ([]*exportBookmarkEntry, map[string]*exportFolderNode) {
	rootEntries := []*exportBookmarkEntry{}
	folderTree := make(map[string]*exportFolderNode)

	for _, entry := range entries {
		if len(entry.folderNames) == 0 {
			rootEntries = append(rootEntries, entry)
			continue
		}

		primaryFolder := strings.TrimSpace(entry.folderNames[0])
		if primaryFolder == "" {
			rootEntries = append(rootEntries, entry)
			continue
		}

		segments := splitAndTrim(primaryFolder, folderPathSeparator)
		if len(segments) == 0 {
			rootEntries = append(rootEntries, entry)
			continue
		}

		current := folderTree
		var node *exportFolderNode
		for _, segment := range segments {
			nextNode, ok := current[segment]
			if !ok {
				nextNode = &exportFolderNode{Name: segment, Children: make(map[string]*exportFolderNode)}
				current[segment] = nextNode
			}
			node = nextNode
			current = nextNode.Children
		}

		if node != nil {
			node.Entries = append(node.Entries, entry)
		}
	}

	sort.Slice(rootEntries, func(i, j int) bool {
		return rootEntries[i].bookmark.DateAdded.Before(rootEntries[j].bookmark.DateAdded)
	})
	return rootEntries, folderTree
}

func writeExportFolders(buf *bytes.Buffer, nodes map[string]*exportFolderNode, indent string) {
	if len(nodes) == 0 {
		return
	}

	names := make([]string, 0, len(nodes))
	for name := range nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		node := nodes[name]
		buf.WriteString(indent)
		buf.WriteString("<DT><H3>")
		buf.WriteString(stdhtml.EscapeString(node.Name))
		buf.WriteString("</H3>\n")
		buf.WriteString(indent)
		buf.WriteString("<DL><p>\n")

		sort.Slice(node.Entries, func(i, j int) bool {
			return node.Entries[i].bookmark.DateAdded.Before(node.Entries[j].bookmark.DateAdded)
		})
		for _, entry := range node.Entries {
			writeExportBookmark(buf, entry, indent+"    ")
		}
		writeExportFolders(buf, node.Children, indent+"    ")

		buf.WriteString(indent)
		buf.WriteString("</DL><p>\n")
	}
}

func writeExportBookmark(buf *bytes.Buffer, entry *exportBookmarkEntry, indent string) {
	bookmark := entry.bookmark
	buf.WriteString(indent)
	buf.WriteString("<DT><A HREF=\"")
	buf.WriteString(stdhtml.EscapeString(bookmark.URL))
	buf.WriteString("\"")

	if !bookmark.DateAdded.IsZero() {
		buf.WriteString(fmt.Sprintf(" ADD_DATE=\"%d\"", bookmark.DateAdded.Unix()))
	}
	if len(bookmark.TagNames) > 0 {
		buf.WriteString(" TAGS=\"")
		buf.WriteString(stdhtml.EscapeString(strings.Join(uniqueStrings(bookmark.TagNames), ",")))
		buf.WriteString("\"")
	}
	if len(entry.folderNames) > 0 {
		buf.WriteString(" LINKGENIE_FOLDERS=\"")
		buf.WriteString(stdhtml.EscapeString(strings.Join(uniqueStrings(entry.folderNames), "||")))
		buf.WriteString("\"")
	}
	buf.WriteString(">")

	title := strings.TrimSpace(bookmark.Title)
	if title == "" {
		title = bookmark.URL
	}
	buf.WriteString(stdhtml.EscapeString(title))
	buf.WriteString("</A>\n")

	description := strings.TrimSpace(bookmark.Description)
	if description == "" {
		description = strings.TrimSpace(bookmark.Notes)
	}
	if description != "" {
		buf.WriteString(indent)
		buf.WriteString("<DD>")
		buf.WriteString(stdhtml.EscapeString(description))
		buf.WriteString("\n")
	}
}

func nextElementSibling(node *xhtml.Node) *xhtml.Node {
	for sibling := node.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == xhtml.ElementNode {
			return sibling
		}
	}
	return nil
}

func findFirstElement(node *xhtml.Node, tag string) *xhtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == xhtml.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func findFirstDescendantElement(node *xhtml.Node, tag string) *xhtml.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode && child.Data == tag {
			return child
		}
		if found := findFirstDescendantElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractNodeText(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == xhtml.TextNode {
		return node.Data
	}

	var parts []string
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text := strings.TrimSpace(extractNodeText(child))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func splitAndTrim(value, separator string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return uniqueStrings(result)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

// ExportFilename 返回建议下载文件名。
func ExportFilename(now time.Time) string {
	return fmt.Sprintf("bookmarks_%s.html", now.Format("2006-01-02"))
}
