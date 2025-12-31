package db

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"ai-bookmark-service/models"
)

// FolderRepository handles folder database operations
type FolderRepository struct {
	db         *sql.DB
	bookmarkRepo *BookmarkRepository
}

// NewFolderRepository creates a new folder repository
func NewFolderRepository(bookmarkRepo *BookmarkRepository) *FolderRepository {
	return &FolderRepository{
		db:         DB,
		bookmarkRepo: bookmarkRepo,
	}
}

// Create creates a new folder
func (r *FolderRepository) Create(fc *models.FolderCreate) (*models.Folder, error) {
	// Get current max sort_order
	var maxOrder int
	if err := r.db.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM folders").Scan(&maxOrder); err != nil {
		log.Printf("⚠️ 获取最大sort_order失败: %v, 使用默认值0", err)
		maxOrder = 0
	}
	
	now := time.Now().UTC().Format(time.RFC3339Nano)
	
	// Set defaults
	color := fc.Color
	if color == "" {
		color = "#007AFF"
	}
	icon := fc.Icon
	if icon == "" {
		icon = "📁"
	}
	
	result, err := r.db.Exec(
		"INSERT INTO folders (name, color, icon, sort_order, date_added) VALUES (?, ?, ?, ?, ?)",
		fc.Name, color, icon, maxOrder+1, now,
	)
	if err != nil {
		return nil, err
	}
	
	id, _ := result.LastInsertId()
	return r.GetByID(int(id))
}

// GetByID retrieves a folder by ID
func (r *FolderRepository) GetByID(id int) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.QueryRow(
		"SELECT id, name, color, icon, sort_order, date_added FROM folders WHERE id = ?",
		id,
	).Scan(&folder.ID, &folder.Name, &folder.Color, &folder.Icon, &folder.SortOrder, &folder.DateAdded)
	
	if err != nil {
		return nil, err
	}
	
	// Get bookmark count
	if err := r.db.QueryRow("SELECT COUNT(*) FROM bookmark_folders WHERE folder_id = ?", id).Scan(&folder.Count); err != nil {
		log.Printf("⚠️ 获取文件夹书签数量失败: %v", err)
		folder.Count = 0
	}
	
	return &folder, nil
}

// List retrieves all folders
func (r *FolderRepository) List() ([]*models.Folder, error) {
	// Optimized: Use JOIN to get folders and counts in one query, solving N+1 problem
	query := `
		SELECT 
			f.id, f.name, f.color, f.icon, f.sort_order, f.date_added,
			COUNT(bf.bookmark_id) as bookmark_count
		FROM folders f
		LEFT JOIN bookmark_folders bf ON f.id = bf.folder_id
		GROUP BY f.id, f.name, f.color, f.icon, f.sort_order, f.date_added
		ORDER BY f.sort_order ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	folders := []*models.Folder{}
	for rows.Next() {
		var folder models.Folder
		if err := rows.Scan(&folder.ID, &folder.Name, &folder.Color, &folder.Icon, &folder.SortOrder, &folder.DateAdded, &folder.Count); err != nil {
			log.Printf("⚠️ 扫描文件夹失败: %v", err)
			continue
		}
		log.Printf("📁 文件夹: %s (ID: %d), 书签数量: %d", folder.Name, folder.ID, folder.Count)
		folders = append(folders, &folder)
	}
	
	return folders, nil
}

// Update updates a folder
func (r *FolderRepository) Update(id int, fc *models.FolderCreate, sortOrder *int) (*models.Folder, error) {
	// Build dynamic update statement
	updates := []string{}
	args := []interface{}{}
	
	if fc.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, fc.Name)
	}
	if fc.Color != "" {
		updates = append(updates, "color = ?")
		args = append(args, fc.Color)
	}
	if fc.Icon != "" {
		updates = append(updates, "icon = ?")
		args = append(args, fc.Icon)
	}
	if sortOrder != nil {
		updates = append(updates, "sort_order = ?")
		args = append(args, *sortOrder)
	}
	
	if len(updates) == 0 {
		return r.GetByID(id)
	}
	
	query := "UPDATE folders SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, id)
	
	_, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	
	return r.GetByID(id)
}

// Delete deletes a folder
func (r *FolderRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}

// GetBookmarks retrieves bookmarks in a folder
func (r *FolderRepository) GetBookmarks(folderID int, limit, offset int) ([]*models.Bookmark, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM bookmark_folders WHERE folder_id = ?", folderID).Scan(&total); err != nil {
		log.Printf("⚠️ 获取文件夹书签总数失败: %v", err)
		total = 0
	}
	
	// Get bookmark IDs
	rows, err := r.db.Query(`
		SELECT bf.bookmark_id 
		FROM bookmark_folders bf
		JOIN bookmarks b ON bf.bookmark_id = b.id
		WHERE bf.folder_id = ?
		ORDER BY bf.date_added DESC
		LIMIT ? OFFSET ?
	`, folderID, limit, offset)
	
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	bookmarks := []*models.Bookmark{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("⚠️ 扫描书签ID失败: %v", err)
			continue
		}
		if bm, err := r.bookmarkRepo.GetByID(id); err == nil {
			bookmarks = append(bookmarks, bm)
		} else {
			log.Printf("⚠️ 获取书签详情失败 ID=%d: %v", id, err)
		}
	}
	
	return bookmarks, total, nil
}

// AddBookmark adds a bookmark to a folder
func (r *FolderRepository) AddBookmark(bookmarkID, folderID int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(
		"INSERT OR IGNORE INTO bookmark_folders (bookmark_id, folder_id, date_added) VALUES (?, ?, ?)",
		bookmarkID, folderID, now,
	)
	return err
}

// RemoveBookmark removes a bookmark from a folder
func (r *FolderRepository) RemoveBookmark(bookmarkID, folderID int) error {
	_, err := r.db.Exec(
		"DELETE FROM bookmark_folders WHERE bookmark_id = ? AND folder_id = ?",
		bookmarkID, folderID,
	)
	return err
}

// GetBookmarkFolders retrieves folders that contain a bookmark
func (r *FolderRepository) GetBookmarkFolders(bookmarkID int) ([]*models.Folder, error) {
	rows, err := r.db.Query(`
		SELECT f.id FROM folders f
		JOIN bookmark_folders bf ON f.id = bf.folder_id
		WHERE bf.bookmark_id = ?
		ORDER BY f.sort_order
	`, bookmarkID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	folders := []*models.Folder{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("⚠️ 扫描文件夹ID失败: %v", err)
			continue
		}
		if folder, err := r.GetByID(id); err == nil {
			folders = append(folders, folder)
		} else {
			log.Printf("⚠️ 获取文件夹详情失败 ID=%d: %v", id, err)
		}
	}
	
	return folders, nil
}
