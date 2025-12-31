package db

import (
	"database/sql"
	"fmt"

	"ai-bookmark-service/models"
)

// TagRepository 标签数据库操作
type TagRepository struct {
	db *sql.DB
}

// NewTagRepository 创建标签仓库
func NewTagRepository() *TagRepository {
	return &TagRepository{db: DB}
}

// GetByID 根据 ID 获取标签
func (r *TagRepository) GetByID(id int) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.QueryRow(`
		SELECT id, name, COALESCE(category, 'candidate'), COALESCE(usage_count, 0), 
		       COALESCE(last_used, date_added), date_added 
		FROM tags WHERE id = ?
	`, id).Scan(&tag.ID, &tag.Name, &tag.Category, &tag.UsageCount, &tag.LastUsed, &tag.DateAdded)

	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetOrCreate 获取或创建标签
func (r *TagRepository) GetOrCreate(tagName string) (int, error) {
	// 先尝试获取
	var tagID int
	err := r.db.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err == nil {
		return tagID, nil
	}

	// 不存在则创建
	result, err := r.db.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
	if err != nil {
		return 0, fmt.Errorf("创建标签失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取标签ID失败: %w", err)
	}

	return int(id), nil
}

// List 获取所有标签
func (r *TagRepository) List() ([]*models.Tag, error) {
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(category, 'candidate'), COALESCE(usage_count, 0), 
		       COALESCE(last_used, date_added), date_added 
		FROM tags ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer rows.Close()

	tags := []*models.Tag{}
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.UsageCount, &tag.LastUsed, &tag.DateAdded); err != nil {
			fmt.Printf("❌ Scan错误: %v\n", err)
			continue
		}
		tags = append(tags, &tag)
	}

	fmt.Printf("🔍 TagRepository.List() 返回 %d 个标签\n", len(tags))
	return tags, nil
}

// ListByCategories 根据分类获取标签
func (r *TagRepository) ListByCategories(categories []string) ([]*models.Tag, error) {
	if len(categories) == 0 {
		return []*models.Tag{}, nil
	}

	// 构建占位符
	placeholders := ""
	args := []interface{}{}
	for i, cat := range categories {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, cat)
	}

	query := fmt.Sprintf(`
		SELECT id, name, COALESCE(category, 'candidate'), COALESCE(usage_count, 0), 
		       COALESCE(last_used, date_added), date_added 
		FROM tags 
		WHERE category IN (%s)
		ORDER BY usage_count DESC, name
	`, placeholders)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer rows.Close()

	tags := []*models.Tag{}
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.UsageCount, &tag.LastUsed, &tag.DateAdded); err != nil {
			continue
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}

// UpdateCategory 更新标签分类
func (r *TagRepository) UpdateCategory(tagID int, category string) error {
	_, err := r.db.Exec("UPDATE tags SET category = ? WHERE id = ?", category, tagID)
	if err != nil {
		return fmt.Errorf("更新标签分类失败: %w", err)
	}
	return nil
}

// GetBookmarkCount 获取标签关联的书签数量
func (r *TagRepository) GetBookmarkCount(tagID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM bookmark_tags WHERE tag_id = ?", tagID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询书签数量失败: %w", err)
	}
	return count, nil
}

// MergeBookmarks 将源标签的所有书签关联转移到目标标签
func (r *TagRepository) MergeBookmarks(sourceID, targetID int) error {
	// 1. 获取源标签的所有书签
	rows, err := r.db.Query("SELECT bookmark_id FROM bookmark_tags WHERE tag_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("查询源标签书签失败: %w", err)
	}
	defer rows.Close()

	bookmarkIDs := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		bookmarkIDs = append(bookmarkIDs, id)
	}

	// 2. 为每个书签添加目标标签关联(忽略重复)
	for _, bmID := range bookmarkIDs {
		_, err := r.db.Exec(`
			INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) 
			VALUES (?, ?)
		`, bmID, targetID)
		if err != nil {
			return fmt.Errorf("添加目标标签关联失败: %w", err)
		}
	}

	// 3. 删除源标签的所有关联
	_, err = r.db.Exec("DELETE FROM bookmark_tags WHERE tag_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("删除源标签关联失败: %w", err)
	}

	return nil
}

// RecordSynonym 记录同义词关系
func (r *TagRepository) RecordSynonym(mainTagID, synonymTagID int, similarity float64, autoMerged bool) error {
	autoMergedInt := 0
	if autoMerged {
		autoMergedInt = 1
	}

	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO tag_synonyms (main_tag_id, synonym_tag_id, similarity_score, auto_merged) 
		VALUES (?, ?, ?, ?)
	`, mainTagID, synonymTagID, similarity, autoMergedInt)

	if err != nil {
		return fmt.Errorf("记录同义词失败: %w", err)
	}
	return nil
}

// Delete 删除标签
func (r *TagRepository) Delete(tagID int) error {
	_, err := r.db.Exec("DELETE FROM tags WHERE id = ?", tagID)
	if err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	return nil
}

// IncrementUsage 增加标签使用次数
func (r *TagRepository) IncrementUsage(tagID int) error {
	_, err := r.db.Exec(`
		UPDATE tags 
		SET usage_count = usage_count + 1, last_used = CURRENT_TIMESTAMP 
		WHERE id = ?
	`, tagID)
	if err != nil {
		return fmt.Errorf("更新使用次数失败: %w", err)
	}
	return nil
}

// GetTopTags 获取使用次数最多的标签
func (r *TagRepository) GetTopTags(limit int) []*models.Tag {
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(category, 'candidate'), COALESCE(usage_count, 0), 
		       COALESCE(last_used, date_added), date_added 
		FROM tags 
		WHERE usage_count > 0
		ORDER BY usage_count DESC, name 
		LIMIT ?
	`, limit)
	if err != nil {
		return []*models.Tag{}
	}
	defer rows.Close()

	tags := []*models.Tag{}
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.UsageCount, &tag.LastUsed, &tag.DateAdded); err != nil {
			continue
		}
		tags = append(tags, &tag)
	}

	return tags
}

// CountByCategory 统计指定分类的标签数量
func (r *TagRepository) CountByCategory(category string) int {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM tags WHERE category = ?", category).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}
