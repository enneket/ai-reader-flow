package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "reader.db")
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	if err = DB.Ping(); err != nil {
		return err
	}

	if err = createTables(); err != nil {
		return err
	}

	log.Println("Database initialized at:", dbPath)
	return nil
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		description TEXT,
		icon_url TEXT,
		last_fetched TEXT,
		is_dead INTEGER DEFAULT 0,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		group_name TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		link TEXT UNIQUE NOT NULL,
		content TEXT,
		summary TEXT,
		author TEXT,
		published TEXT,
		is_filtered INTEGER DEFAULT 0,
		is_saved INTEGER DEFAULT 0,
		status TEXT DEFAULT 'unread',
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS filter_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		value TEXT NOT NULL,
		action TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		title TEXT NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS briefings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status TEXT DEFAULT 'pending',
		error TEXT,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		completed_at TEXT
	);

	CREATE TABLE IF NOT EXISTS briefing_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		briefing_id INTEGER NOT NULL,
		topic TEXT NOT NULL,
		summary TEXT,
		sort_order INTEGER DEFAULT 0,
		FOREIGN KEY (briefing_id) REFERENCES briefings(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS briefing_articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		briefing_item_id INTEGER NOT NULL,
		article_id INTEGER NOT NULL,
		title TEXT,
		FOREIGN KEY (briefing_item_id) REFERENCES briefing_items(id) ON DELETE CASCADE,
		FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	CREATE INDEX IF NOT EXISTS idx_articles_is_filtered ON articles(is_filtered);
	CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
	CREATE INDEX IF NOT EXISTS idx_notes_article_id ON notes(article_id);
	CREATE INDEX IF NOT EXISTS idx_briefing_items_briefing_id ON briefing_items(briefing_id);
	CREATE INDEX IF NOT EXISTS idx_briefing_articles_briefing_item_id ON briefing_articles(briefing_item_id);
	CREATE INDEX IF NOT EXISTS idx_briefings_created_at ON briefings(created_at);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: add is_dead column if it doesn't exist (for existing databases)
	_ = migrateAddColumn("feeds", "is_dead", "INTEGER DEFAULT 0")
	// Migration: add group column for feed folders
	_ = migrateAddColumn("feeds", "group_name", "TEXT DEFAULT ''")
	// Migration: add status column if it doesn't exist (for existing databases)
	_ = migrateAddColumn("articles", "status", "TEXT DEFAULT 'unread'")
	// Migration: add embedding and quality_score columns for Plan B
	_ = migrateAddColumn("articles", "embedding", "TEXT")
	_ = migrateAddColumn("articles", "quality_score", "INTEGER DEFAULT 0")
	// Migration: add feed refresh status columns
	_ = migrateAddColumn("feeds", "last_refresh_success", "INTEGER DEFAULT 0")
	_ = migrateAddColumn("feeds", "last_refresh_error", "TEXT DEFAULT ''")
	_ = migrateAddColumn("feeds", "last_refreshed", "TEXT")
	// Migration: add unread_count column to feeds
	_ = migrateAddColumn("feeds", "unread_count", "INTEGER NOT NULL DEFAULT 0")

	// Create FTS5 virtual table for full-text search
	_ = createFTSTable()

	return nil
}

func migrateAddColumn(table, column, definition string) error {
	// Check if column exists
	var count int
	err := DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", table, column)).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // column already exists
	}
	_, err = DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func createFTSTable() error {
	// Create FTS5 virtual table for full-text search on article title + content
	_, err := DB.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS articles_fts USING fts5(
			article_id UNINDEXED,
			title,
			content,
			content='articles',
			content_rowid='id'
		)
	`)
	if err != nil {
		return err
	}
	// Populate FTS table from existing articles (idempotent — just upsert)
	_, _ = DB.Exec(`
		INSERT OR IGNORE INTO articles_fts(rowid, article_id, title, content)
		SELECT id, id, title, COALESCE(content, '') FROM articles
		WHERE id NOT IN (SELECT article_id FROM articles_fts)
	`)
	return nil
}

// IndexArticle adds an article to the FTS index (called after article is created)
func IndexArticle(articleID int64, title, content string) error {
	_, err := DB.Exec(
		`INSERT OR REPLACE INTO articles_fts(rowid, article_id, title, content) VALUES (?, ?, ?, ?)`,
		articleID, articleID, title, content,
	)
	return err
}

// RemoveArticleFTS removes an article from the FTS index
func RemoveArticleFTS(articleID int64) error {
	_, err := DB.Exec(`DELETE FROM articles_fts WHERE article_id = ?`, articleID)
	return err
}
