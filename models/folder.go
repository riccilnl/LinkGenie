package models

import "time"

// Folder 文件夹数据模型
type Folder struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	DateAdded time.Time `json:"date_added"`
	Count     int       `json:"count"` // 书签数量
}

// FolderCreate 创建文件夹请求
type FolderCreate struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}
