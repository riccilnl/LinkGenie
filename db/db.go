package db

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// DB 全局数据库连接
var DB *sql.DB

// Init 初始化数据库
func Init(dbPath string) error {
	var err error
	// 使用 DSN 参数配置 WAL 模式和超时，确保连接池中的所有连接都生效
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	// 限制连接数以避免在极高并发下触发 SQLite 锁定
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(time.Hour)

	// 创建表
	schema := `
	CREATE TABLE IF NOT EXISTS bookmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		title TEXT DEFAULT '',
		description TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		is_favorite INTEGER DEFAULT 0,
		unread INTEGER DEFAULT 0,
		shared INTEGER DEFAULT 0,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
		date_modified DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		category TEXT DEFAULT 'candidate',
		usage_count INTEGER DEFAULT 0,
		last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tag_synonyms (
		main_tag_id INTEGER,
		synonym_tag_id INTEGER,
		similarity_score REAL DEFAULT 0.0,
		auto_merged INTEGER DEFAULT 0,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (main_tag_id, synonym_tag_id),
		FOREIGN KEY (main_tag_id) REFERENCES tags(id) ON DELETE CASCADE,
		FOREIGN KEY (synonym_tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS bookmark_tags (
		bookmark_id INTEGER,
		tag_id INTEGER,
		FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (bookmark_id, tag_id)
	);

	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		color TEXT DEFAULT '#007AFF',
		icon TEXT DEFAULT '📁',
		sort_order INTEGER DEFAULT 0,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bookmark_folders (
		bookmark_id INTEGER NOT NULL,
		folder_id INTEGER NOT NULL,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id) ON DELETE CASCADE,
		FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
		PRIMARY KEY (bookmark_id, folder_id)
	);

	CREATE TABLE IF NOT EXISTS workflows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		enabled INTEGER DEFAULT 1,
		priority INTEGER DEFAULT 0,
		condition_logic TEXT DEFAULT 'OR',
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
		date_modified DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS workflow_triggers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL,
		trigger_type TEXT NOT NULL,
		config TEXT NOT NULL,
		FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS workflow_actions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL,
		action_type TEXT NOT NULL,
		config TEXT NOT NULL,
		FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS system_configs (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		date_modified DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_bookmarks_url ON bookmarks(url);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_date_added ON bookmarks(date_added DESC);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_is_favorite ON bookmarks(is_favorite);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_unread ON bookmarks(unread);
	CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
	CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category);
	CREATE INDEX IF NOT EXISTS idx_tags_usage ON tags(usage_count);
	CREATE INDEX IF NOT EXISTS idx_folders_sort ON folders(sort_order);
	CREATE INDEX IF NOT EXISTS idx_bookmark_folders_bookmark ON bookmark_folders(bookmark_id);
	CREATE INDEX IF NOT EXISTS idx_bookmark_folders_folder ON bookmark_folders(folder_id);
	CREATE INDEX IF NOT EXISTS idx_workflows_enabled ON workflows(enabled);
	CREATE INDEX IF NOT EXISTS idx_workflows_priority ON workflows(priority);
	CREATE INDEX IF NOT EXISTS idx_workflow_triggers_workflow ON workflow_triggers(workflow_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_actions_workflow ON workflow_actions(workflow_id);
	`

	_, err = DB.Exec(schema)
	if err != nil {
		return err
	}

	log.Printf("✅ 数据库初始化成功 (WAL模式): %s", dbPath)
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
