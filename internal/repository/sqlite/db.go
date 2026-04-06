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

	CREATE TABLE IF NOT EXISTS prompt_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		prompt TEXT NOT NULL,
		system TEXT NOT NULL DEFAULT '',
		max_tokens INTEGER NOT NULL DEFAULT 500,
		is_default INTEGER NOT NULL DEFAULT 0
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
	_ = migrateAddColumn("briefing_articles", "insight", "TEXT DEFAULT ''")
	// Migration: drop embedding and quality_score columns (embedding feature removed)
	_ = migrateDropColumn("articles", "embedding")
	_ = migrateDropColumn("articles", "quality_score")
	// Migration: add translation columns for English article translation feature
	_ = migrateAddColumn("articles", "is_translated", "INTEGER DEFAULT 0")
	_ = migrateAddColumn("articles", "translated_content", "TEXT")

	// Seed default prompts if table is empty
	_ = seedDefaultPrompts()

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

func migrateDropColumn(table, column string) error {
	// Check if column exists
	var count int
	err := DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", table, column)).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil // column doesn't exist
	}
	_, err = DB.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column))
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

func seedDefaultPrompts() error {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM prompt_configs").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultPrompts := []struct {
		promptType  string
		name       string
		prompt     string
		system     string
		maxTokens  int
		isDefault  int
	}{
		{
			promptType: "summary",
			name:       "快读摘要",
			prompt: `为提供的文本创作一份"快读摘要"，旨在让读者在30秒内掌握核心情报。

要求：
1) 极简主义：剔除背景铺垫、案例细节、营销话术及修饰性词汇，直奔主题。
2) 内容密度：必须包含核心主体、关键动作/事件、最终影响/结论。
3) 篇幅：严格控制在50-150字之间。

待摘要内容：
{content}`,
			system:    `你是一名资深内容分析师，擅长用最极简的语言精准捕捉文章灵魂。输出必须为中文、客观、单段长句（可用逗号、句号，禁止分段/换行），禁止任何列表符号，禁止出现"这篇文章讲了/摘要如下"等前置废话。`,
			maxTokens: 400,
			isDefault: 1,
		},
		{
			promptType: "briefing",
			name:       "简报生成",
			prompt: `你是一个内容策划助手。请根据以下文章内容，生成一份简报。

要求：
1) 将文章按主题分组
2) 每个主题给出简短总结
3) 标注文章来源

{content}`,
			system:    `你是一个内容策划助手。`,
			maxTokens: 16384,
			isDefault: 1,
		},
		{
			promptType: "translation",
			name:       "中英翻译",
			prompt: `将以下英文文章翻译成中文。

要求：
1) 严格保留原始Markdown格式
2) 专业术语使用业界通用中文表达
3) 语言风格地道、通顺

{content}`,
			system:    `你是一位精通中英文互译的专业翻译官。必须仅输出中文译文，禁止任何额外话语。`,
			maxTokens: 14000,
			isDefault: 1,
		},
	}

	for _, p := range defaultPrompts {
		_, err := DB.Exec(
			`INSERT INTO prompt_configs (type, name, prompt, system, max_tokens, is_default) VALUES (?, ?, ?, ?, ?, ?)`,
			p.promptType, p.name, p.prompt, p.system, p.maxTokens, p.isDefault,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
