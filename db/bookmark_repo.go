package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-bookmark-service/models"
)

// BookmarkRepository 书签数据库操作
type BookmarkRepository struct {
	db *sql.DB
}

// NewBookmarkRepository 创建书签仓库
func NewBookmarkRepository() *BookmarkRepository {
	return &BookmarkRepository{db: DB}
}

// Create 创建书签（带事务处理）
func (r *BookmarkRepository) Create(bm *models.BookmarkCreate) (*models.Bookmark, error) {
	// 检查是否已存在
	var existingID int
	err := r.db.QueryRow("SELECT id FROM bookmarks WHERE url = ?", bm.URL).Scan(&existingID)
	if err == nil {
		log.Printf("🔄 URL已存在(ID=%d)，转为更新操作", existingID)
		return r.Update(existingID, bm)
	}

	// 开始事务
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	log.Printf("📝 执行INSERT: URL=%s Title=%s Shared=%v", bm.URL, bm.Title, bm.Shared)

	// 插入书签
	result, err := tx.Exec(
		"INSERT INTO bookmarks (url, title, description, notes, is_favorite, unread, shared, date_added, date_modified) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		bm.URL, bm.Title, bm.Description, bm.Notes, bm.IsFavorite, bm.Unread, bm.Shared, now, now,
	)
	if err != nil {
		log.Printf("❌ INSERT失败: %v", err)
		return nil, fmt.Errorf("插入书签失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取插入ID失败: %w", err)
	}

	// 添加标签（在同一事务中）
	for _, tagName := range bm.TagNames {
		tagID, err := r.getOrCreateTagTx(tx, tagName)
		if err != nil {
			log.Printf("⚠️ 创建标签失败: %s, 错误: %v", tagName, err)
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
			log.Printf("⚠️ 关联标签失败: %s, 错误: %v", tagName, err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 获取创建的书签
	return r.GetByID(int(id))
}

// Update 更新书签（带事务处理）
func (r *BookmarkRepository) Update(id int, bm *models.BookmarkCreate) (*models.Bookmark, error) {
	// 开始事务
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	log.Printf("🔄 执行UPDATE: ID=%d Title=%s Shared=%v", id, bm.Title, bm.Shared)

	_, err = tx.Exec(
		"UPDATE bookmarks SET url=?, title=?, description=?, notes=?, is_favorite=?, unread=?, shared=?, date_modified=? WHERE id=?",
		bm.URL, bm.Title, bm.Description, bm.Notes, bm.IsFavorite, bm.Unread, bm.Shared, now, id,
	)
	if err != nil {
		log.Printf("❌ UPDATE失败: %v", err)
		return nil, fmt.Errorf("更新书签失败: %w", err)
	}

	// 更新标签（删除旧的，添加新的）
	if _, err := tx.Exec("DELETE FROM bookmark_tags WHERE bookmark_id = ?", id); err != nil {
		log.Printf("⚠️ 删除旧标签失败: %v", err)
	}

	for _, tagName := range bm.TagNames {
		tagID, err := r.getOrCreateTagTx(tx, tagName)
		if err != nil {
			log.Printf("⚠️ 创建标签失败: %s, 错误: %v", tagName, err)
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
			log.Printf("⚠️ 关联标签失败: %s, 错误: %v", tagName, err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return r.GetByID(id)
}

// GetByID 根据ID获取书签
func (r *BookmarkRepository) GetByID(id int) (*models.Bookmark, error) {
	// 使用 LEFT JOIN 一次性获取书签和标签（解决 N+1 问题）
	query := `
		SELECT 
			b.id, b.url, b.title, b.description, b.notes,
			b.is_favorite, b.unread, b.shared,
			b.date_added, b.date_modified,
			GROUP_CONCAT(t.name, ',') as tag_names
		FROM bookmarks b
		LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
		LEFT JOIN tags t ON bt.tag_id = t.id
		WHERE b.id = ?
		GROUP BY b.id
	`

	var bm models.Bookmark
	var tagNamesStr sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&bm.ID, &bm.URL, &bm.Title, &bm.Description, &bm.Notes,
		&bm.IsFavorite, &bm.Unread, &bm.Shared,
		&bm.DateAdded, &bm.DateModified,
		&tagNamesStr,
	)
	if err != nil {
		return nil, err
	}

	// 解析标签
	if tagNamesStr.Valid && tagNamesStr.String != "" {
		bm.TagNames = strings.Split(tagNamesStr.String, ",")
	} else {
		bm.TagNames = []string{}
	}

	return &bm, nil
}

// GetByURL 根据URL获取书签
func (r *BookmarkRepository) GetByURL(url string) (*models.Bookmark, error) {
	var id int
	err := r.db.QueryRow("SELECT id FROM bookmarks WHERE url = ?", url).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

// List 获取书签列表（优化版，解决 N+1 查询问题）
func (r *BookmarkRepository) List(limit, offset int, filters map[string]interface{}) ([]*models.Bookmark, error) {
	// 使用 LEFT JOIN 一次性获取所有数据
	query := `
		SELECT 
			b.id, b.url, b.title, b.description, b.notes,
			b.is_favorite, b.unread, b.shared,
			b.date_added, b.date_modified,
			GROUP_CONCAT(t.name, ',') as tag_names
		FROM bookmarks b
		LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
		LEFT JOIN tags t ON bt.tag_id = t.id
	`

	// 构建 WHERE 条件
	whereClauses := []string{}
	args := []interface{}{}

	if q, ok := filters["q"].(string); ok && q != "" {
		whereClauses = append(whereClauses, "(b.title LIKE ? OR b.description LIKE ? OR b.url LIKE ?)")
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if unread, ok := filters["unread"].(bool); ok {
		whereClauses = append(whereClauses, "b.unread = ?")
		args = append(args, unread)
	}

	if shared, ok := filters["shared"].(bool); ok {
		whereClauses = append(whereClauses, "b.shared = ?")
		args = append(args, shared)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " GROUP BY b.id ORDER BY b.date_added DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询书签列表失败: %w", err)
	}
	defer rows.Close()

	bookmarks := []*models.Bookmark{}
	for rows.Next() {
		var bm models.Bookmark
		var tagNamesStr sql.NullString

		err := rows.Scan(
			&bm.ID, &bm.URL, &bm.Title, &bm.Description, &bm.Notes,
			&bm.IsFavorite, &bm.Unread, &bm.Shared,
			&bm.DateAdded, &bm.DateModified,
			&tagNamesStr,
		)
		if err != nil {
			log.Printf("⚠️ 扫描书签失败: %v", err)
			continue
		}

		// 解析标签
		if tagNamesStr.Valid && tagNamesStr.String != "" {
			bm.TagNames = strings.Split(tagNamesStr.String, ",")
		} else {
			bm.TagNames = []string{}
		}

		bookmarks = append(bookmarks, &bm)
	}

	return bookmarks, nil
}

// Delete 删除书签
func (r *BookmarkRepository) Delete(id int) error {
	result, err := r.db.Exec("DELETE FROM bookmarks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除书签失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("书签不存在: ID=%d", id)
	}

	return nil
}

// Count 统计书签数量
func (r *BookmarkRepository) Count(filters map[string]interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM bookmarks"

	whereClauses := []string{}
	args := []interface{}{}

	if q, ok := filters["q"].(string); ok && q != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR description LIKE ? OR url LIKE ?)")
		searchTerm := "%" + q + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// getOrCreateTagTx 在事务中获取或创建标签
func (r *BookmarkRepository) getOrCreateTagTx(tx *sql.Tx, tagName string) (int, error) {
	// 先尝试获取
	var tagID int
	err := tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err == nil {
		return tagID, nil
	}

	// 不存在则创建
	result, err := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
	if err != nil {
		return 0, fmt.Errorf("创建标签失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取标签ID失败: %w", err)
	}

	return int(id), nil
}
