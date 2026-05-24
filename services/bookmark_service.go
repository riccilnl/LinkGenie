package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/models"
	"github.com/riccilnl/LinkGenie/utils"
)

var (
	ErrBookmarkNotFound       = errors.New("书签不存在")
	ErrBookmarkAlreadyExists  = errors.New("书签已存在")
	ErrEnhancementUnavailable = errors.New("AI增强队列不可用")
	ErrEnhancementQueueFull   = errors.New("AI增强队列已满")
)

// CreateBookmarkInput 创建书签输入
type CreateBookmarkInput struct {
	URL         string
	Title       string
	Description string
	Notes       string
	IsFavorite  bool
	Unread      bool
	Shared      bool
	TagNames    []string
	FolderIDs   []int
	IsArchived  bool
}

// ReplaceBookmarkInput 全量替换输入
type ReplaceBookmarkInput struct {
	URL         string
	Title       string
	Description string
	Notes       string
	IsFavorite  bool
	Unread      bool
	Shared      bool
	TagNames    []string
	FolderIDs   []int
	IsArchived  bool
}

// PatchBookmarkInput 局部更新输入
type PatchBookmarkInput struct {
	URL         *string
	Title       *string
	Description *string
	Notes       *string
	IsFavorite  *bool
	Unread      *bool
	Shared      *bool
	TagNames    *[]string
	FolderIDs   *[]int
	IsArchived  *bool
}

// BookmarkService 统一书签主写路径
type BookmarkService struct {
	bookmarkRepo       *db.BookmarkRepository
	tagRepo            *db.TagRepository
	folderRepo         *db.FolderRepository
	workflowEngine     *WorkflowEngine
	enqueueEnhancement func(int) error
}

// NewBookmarkService 创建书签应用服务
func NewBookmarkService(
	bookmarkRepo *db.BookmarkRepository,
	tagRepo *db.TagRepository,
	folderRepo *db.FolderRepository,
	workflowEngine *WorkflowEngine,
	enqueueEnhancement func(int) error,
) *BookmarkService {
	return &BookmarkService{
		bookmarkRepo:       bookmarkRepo,
		tagRepo:            tagRepo,
		folderRepo:         folderRepo,
		workflowEngine:     workflowEngine,
		enqueueEnhancement: enqueueEnhancement,
	}
}

// CreateBookmark 创建书签
func (s *BookmarkService) CreateBookmark(ctx context.Context, input CreateBookmarkInput) (*models.Bookmark, error) {
	_ = ctx

	req := toBookmarkCreate(
		input.URL,
		input.Title,
		input.Description,
		input.Notes,
		input.IsFavorite,
		input.Unread,
		input.Shared,
		input.TagNames,
		input.IsArchived,
	)

	if err := sanitizeBookmarkCreate(req); err != nil {
		return nil, err
	}

	exists, existingID, err := s.bookmarkRepo.ExistsByURL(req.URL)
	if err != nil {
		return nil, fmt.Errorf("检查重复URL失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: ID=%d", ErrBookmarkAlreadyExists, existingID)
	}

	created, err := s.bookmarkRepo.Create(req)
	if err != nil {
		return nil, err
	}

	if err := s.replaceBookmarkFolders(created.ID, input.FolderIDs); err != nil {
		if rollbackErr := s.bookmarkRepo.Delete(created.ID); rollbackErr != nil {
			return nil, fmt.Errorf("关联文件夹失败且回滚创建失败: %v / %w", rollbackErr, err)
		}
		return nil, err
	}

	if err := s.refreshTagUsageCounts(); err != nil {
		return nil, err
	}

	if s.enqueueEnhancement != nil {
		if err := s.enqueueEnhancement(created.ID); err != nil {
			fmt.Printf("⚠️ 自动投递 AI 增强失败: bookmark_id=%d err=%v\n", created.ID, err)
		}
	}

	finalBookmark, err := s.bookmarkRepo.GetByID(created.ID)
	if err != nil {
		return nil, err
	}

	s.dispatchWorkflowEvent("bookmark_created", nil, finalBookmark)

	return finalBookmark, nil
}

// UpdateBookmark 全量替换书签
func (s *BookmarkService) UpdateBookmark(ctx context.Context, id int, input ReplaceBookmarkInput) (*models.Bookmark, error) {
	_ = ctx

	before, err := s.bookmarkRepo.GetByID(id)
	if err != nil {
		return nil, ErrBookmarkNotFound
	}

	req := toBookmarkCreate(
		input.URL,
		input.Title,
		input.Description,
		input.Notes,
		input.IsFavorite,
		input.Unread,
		input.Shared,
		input.TagNames,
		input.IsArchived,
	)

	if err := sanitizeBookmarkCreate(req); err != nil {
		return nil, err
	}

	if err := s.ensureUniqueURL(id, req.URL); err != nil {
		return nil, err
	}

	updated, err := s.bookmarkRepo.Update(id, req)
	if err != nil {
		return nil, err
	}

	if err := s.replaceBookmarkFolders(id, input.FolderIDs); err != nil {
		return nil, err
	}
	updated, err = s.bookmarkRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.refreshTagUsageCounts(); err != nil {
		return nil, err
	}

	s.dispatchWorkflowEvent("bookmark_updated", before, updated)

	return updated, nil
}

// PatchBookmark 局部更新书签
func (s *BookmarkService) PatchBookmark(ctx context.Context, id int, input PatchBookmarkInput) (*models.Bookmark, error) {
	_ = ctx

	current, err := s.bookmarkRepo.GetByID(id)
	if err != nil {
		return nil, ErrBookmarkNotFound
	}

	req := &models.BookmarkCreate{
		URL:         current.URL,
		Title:       current.Title,
		Description: current.Description,
		Notes:       current.Notes,
		IsFavorite:  current.IsFavorite,
		Unread:      current.Unread,
		Shared:      current.Shared,
		TagNames:    append([]string{}, current.TagNames...),
	}

	if input.URL != nil {
		req.URL = *input.URL
	}
	if input.Title != nil {
		req.Title = *input.Title
	}
	if input.Description != nil {
		req.Description = *input.Description
	}
	if input.Notes != nil {
		req.Notes = *input.Notes
	}
	if input.IsFavorite != nil {
		req.IsFavorite = *input.IsFavorite
	}
	if input.Unread != nil {
		req.Unread = *input.Unread
	}
	if input.Shared != nil {
		req.Shared = *input.Shared
	}
	if input.TagNames != nil {
		req.TagNames = append([]string{}, (*input.TagNames)...)
	}
	if input.IsArchived != nil {
		req.IsArchived = *input.IsArchived
		if req.IsArchived {
			req.IsFavorite = true
		}
	}

	if err := sanitizeBookmarkCreate(req); err != nil {
		return nil, err
	}

	if err := s.ensureUniqueURL(id, req.URL); err != nil {
		return nil, err
	}

	updated, err := s.bookmarkRepo.Update(id, req)
	if err != nil {
		return nil, err
	}

	if input.FolderIDs != nil {
		if err := s.replaceBookmarkFolders(id, *input.FolderIDs); err != nil {
			return nil, err
		}
		updated, err = s.bookmarkRepo.GetByID(id)
		if err != nil {
			return nil, err
		}
	}

	if err := s.refreshTagUsageCounts(); err != nil {
		return nil, err
	}

	s.dispatchWorkflowEvent("bookmark_updated", current, updated)

	return updated, nil
}

// DeleteBookmark 删除书签
func (s *BookmarkService) DeleteBookmark(ctx context.Context, id int) error {
	_ = ctx

	before, err := s.bookmarkRepo.GetByID(id)
	if err != nil {
		return ErrBookmarkNotFound
	}

	if err := s.bookmarkRepo.Delete(id); err != nil {
		return err
	}

	if err := s.refreshTagUsageCounts(); err != nil {
		return err
	}

	s.dispatchWorkflowEvent("bookmark_deleted", before, nil)

	return nil
}

// EnqueueEnhancement 投递AI增强任务
func (s *BookmarkService) EnqueueEnhancement(ctx context.Context, id int) error {
	_ = ctx

	if _, err := s.bookmarkRepo.GetByID(id); err != nil {
		return ErrBookmarkNotFound
	}

	if s.enqueueEnhancement == nil {
		return ErrEnhancementUnavailable
	}

	return s.enqueueEnhancement(id)
}

func (s *BookmarkService) ensureUniqueURL(bookmarkID int, url string) error {
	exists, existingID, err := s.bookmarkRepo.ExistsByURL(url)
	if err != nil {
		return fmt.Errorf("检查重复URL失败: %w", err)
	}
	if exists && existingID != bookmarkID {
		return fmt.Errorf("%w: ID=%d", ErrBookmarkAlreadyExists, existingID)
	}
	return nil
}

func (s *BookmarkService) replaceBookmarkFolders(bookmarkID int, folderIDs []int) error {
	if s.folderRepo == nil {
		return nil
	}

	existingFolders, err := s.folderRepo.GetBookmarkFolders(bookmarkID)
	if err != nil {
		return fmt.Errorf("获取书签文件夹失败: %w", err)
	}

	target := make(map[int]bool, len(folderIDs))
	for _, folderID := range folderIDs {
		if folderID > 0 {
			target[folderID] = true
		}
	}

	for _, folder := range existingFolders {
		if !target[folder.ID] {
			if err := s.folderRepo.RemoveBookmark(bookmarkID, folder.ID); err != nil {
				return fmt.Errorf("移除旧文件夹失败: %w", err)
			}
		}
	}

	for folderID := range target {
		if err := s.folderRepo.AddBookmark(bookmarkID, folderID); err != nil {
			return fmt.Errorf("关联文件夹失败: %w", err)
		}
	}

	return nil
}

func (s *BookmarkService) refreshTagUsageCounts() error {
	if s.tagRepo == nil {
		return nil
	}
	return s.tagRepo.RecalculateUsageCounts()
}

func (s *BookmarkService) dispatchWorkflowEvent(eventType string, before *models.Bookmark, after *models.Bookmark) {
	if s.workflowEngine == nil {
		return
	}
	if err := s.workflowEngine.HandleBookmarkEvent(eventType, before, after); err != nil {
		fmt.Printf("⚠️ 工作流事件执行失败: event=%s err=%v\n", eventType, err)
	}
}

func toBookmarkCreate(url, title, description, notes string, favorite, unread, shared bool, tagNames []string, archived bool) *models.BookmarkCreate {
	return &models.BookmarkCreate{
		URL:         url,
		Title:       title,
		Description: description,
		Notes:       notes,
		IsFavorite:  favorite,
		Unread:      unread,
		Shared:      shared,
		TagNames:    append([]string{}, tagNames...),
		IsArchived:  archived,
	}
}

func sanitizeBookmarkCreate(bm *models.BookmarkCreate) error {
	if bm.IsArchived {
		bm.IsFavorite = true
	}

	bm.TagNames = normalizeTagNames(bm.TagNames)

	if len(bm.Title) > 200 {
		bm.Title = bm.Title[:197] + "..."
	}
	if len(bm.Description) > 1000 {
		bm.Description = bm.Description[:997] + "..."
	}

	if err := utils.ValidateBookmarkCreate(bm); err != nil {
		return err
	}

	return nil
}

func normalizeTagNames(tagNames []string) []string {
	seen := make(map[string]bool, len(tagNames))
	result := make([]string, 0, len(tagNames))

	for _, tag := range tagNames {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}

	return result
}

// IsNotFoundError 判断是否为不存在错误
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrBookmarkNotFound) || errors.Is(err, sql.ErrNoRows)
}

// IsConflictError 判断是否为冲突错误
func IsConflictError(err error) bool {
	return errors.Is(err, ErrBookmarkAlreadyExists)
}

// IsUnavailableError 判断是否为服务不可用错误
func IsUnavailableError(err error) bool {
	return errors.Is(err, ErrEnhancementUnavailable) || errors.Is(err, ErrEnhancementQueueFull)
}
