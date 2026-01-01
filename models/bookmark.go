package models

import "time"

// Bookmark 书签数据模型
type Bookmark struct {
	ID           int       `json:"id"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Notes        string    `json:"notes"`
	IsFavorite   bool      `json:"is_favorite"`
	Unread       bool      `json:"unread"`
	Shared       bool      `json:"shared"`
	TagNames     []string  `json:"tag_names"`
	DateAdded    time.Time `json:"date_added"`
	DateModified time.Time `json:"date_modified"`

	// linkding 兼容字段
	WebArchiveSnapshotURL string  `json:"web_archive_snapshot_url"`
	FaviconURL            *string `json:"favicon_url"`
	PreviewImageURL       *string `json:"preview_image_url"`
	WebsiteTitle          *string `json:"website_title"`
	WebsiteDescription    *string `json:"website_description"`
}

// BookmarkCreate 创建书签请求
type BookmarkCreate struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	IsFavorite  bool     `json:"is_favorite"`
	Unread      bool     `json:"unread"`
	Shared      bool     `json:"shared"`
	TagNames    []string `json:"tag_names"`

	// Linkding 兼容字段
	IsArchived bool `json:"is_archived"` // Linkding 支持归档字段
}
