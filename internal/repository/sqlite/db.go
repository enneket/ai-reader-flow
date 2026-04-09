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
	// Migration: add viewpoint fields to briefing_items
	_ = migrateAddColumn("briefing_items", "consensus", "TEXT DEFAULT ''")
	_ = migrateAddColumn("briefing_items", "disputes", "TEXT DEFAULT ''")
	// Migration: add viewpoint fields to briefing_articles
	_ = migrateAddColumn("briefing_articles", "stance", "TEXT DEFAULT ''")
	_ = migrateAddColumn("briefing_articles", "key_argument", "TEXT DEFAULT ''")
	_ = migrateAddColumn("briefing_articles", "source_url", "TEXT DEFAULT ''")
	// Migration: add title/lead/closing to briefings (新闻整合简报 format)
	_ = migrateAddColumn("briefings", "title", "TEXT DEFAULT ''")
	_ = migrateAddColumn("briefings", "lead", "TEXT DEFAULT ''")
	_ = migrateAddColumn("briefings", "closing", "TEXT DEFAULT ''")

	// Migration: update existing briefing prompt to 新闻整合简报 format
	// This ensures existing DBs (which already have the old prompt) get the new version
	newBriefingPrompt := `根据以下文章，生成一份新闻整合简报。

【核心任务】
1. 阅读每篇文章，提炼核心事件（时间+主体+事件+结果）
2. 将文章按领域/主题分为若干分节（最多 {topicLimit} 个分节）
3. 每条只保留核心事实，不添加主观评论

【输出格式】（严格 JSON，不要有其他内容）
{
  "title": "XX领域新闻整合简报",
  "lead": "整合周期+新闻领域+核心总览（1-2句）",
  "sections": [
    {
      "name": "分节名称，如"AI领域"",
      "summary": "本节要目（1-2句），如"涵盖模型进展与行业争议"",
      "articles": [
        {
          "id": 101,
          "insight": "一句话核心事件（时间+主体+核心事件+关键结果）",
          "key_argument": "关键结果或影响（1-2句）",
          "source_url": "https://..."
        }
      ]
    }
  ],
  "closing": "整体趋势概括或后续关注重点（可选，若有）"
}

【规则】
- 每节约 2-5 条新闻，新闻太少则合并到其他节
- 分节按新闻条数排序（多的在前）
- 只包含真正有价值的新闻，无关内容请忽略
- 标题不要带时间，AI 根据文章日期推断时间范围（{dateRange}）

以下是文章（共 {totalArticles} 篇，第 {batchIndex}/{totalBatches} 批）：
{articles}`
	newBriefingSystem := `你是科技新闻分析师，负责提炼文章的核心事件。输出必须是中文 JSON，严格按格式来，不要有任何额外解释。`
	DB.Exec(`UPDATE prompt_configs SET prompt = ?, system = ? WHERE type = 'briefing'`,
		newBriefingPrompt, newBriefingSystem)

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
			prompt: `根据以下文章，生成一份"观点提炼"简报。不是摘要文章讲什么，而是提炼每篇文章的主张/立场/观点。

【核心任务】
1. 阅读每篇文章，提炼其核心观点、立场（支持/反对/中立/信息补充）
2. 将文章按主题分组（最多 5 个主题，每个主题至少2篇文章）
3. 每个主题内：找出共识点，分析分歧/争议点

【立场定义】
- 支持：作者明确赞同该观点/主张
- 反对：作者明确反对该观点/主张
- 中立：文章报道事件但不选边
- 信息补充：文章提供额外证据或视角（无明确立场时使用此值）

【零观点处理】
- 如果文章无明确立场（如纯新闻报道），stance 默认为"信息补充"
- 不要强行给每篇文章都提炼"立场"

【观点提炼要求】
- 观点 = 文章的立场/主张，不是摘要
- 每条观点必须说清楚：这篇文章认为什么/主张什么
- 标注来源文章ID和链接

【求同存异要求】
- 求同：这些文章共同指向的结论。例如："三篇文章都认为AI监管应该基于风险分级"
- 存异：分歧在哪里。例如："A认为应该政府主导，B认为应该行业自律，C认为现有法规足够"
- 如果只有一篇文章，保持 consensus = ""，disputes = ""

【示例】

主题：AI 监管政策争议

Articles:
- id:1, stance:支持, insight:"AI监管应该基于风险分级，高风险应用需要强制审查", key_argument:"现行安全标准不足以覆盖AGI风险"
- id:2, stance:反对, insight:"过度监管会扼杀创新，应该让行业自律", key_argument:"监管成本最终转嫁给小企业"
- id:3, stance:中立, insight:"欧盟AI法案本周通过，细节仍在讨论", key_argument:"法案将于2026年生效"

consensus: "各方都认同AI需要某种形式的监管"
disputes: "分歧在于监管主体（政府vs行业）和监管方式（强制审查vs自律）"

【输出格式】（严格 JSON，不要有其他内容）
{
  "topics": [
    {
      "name": "主题名称",
      "summary": "一句话概括这个主题在讨论什么（20字以内）",
      "articles": [
        {
          "id": 101,
          "insight": "一句话核心观点（独立可读）",
          "stance": "支持|反对|中立|信息补充",
          "key_argument": "核心论点（1-2句）",
          "source_url": "https://..."
        }
      ],
      "consensus": "这些文章的共识",
      "disputes": "分歧点"
    }
  ]
}

规则：
- 每个简报最多 5 个主题
- 每个主题至少 2 篇核心文章
- 只包含真正有价值的文章，无关内容请忽略
- 主题按文章数量排序（多的在前）
- 如果文章太少或无价值，返回空的 topics 数组

以下是今天的文章（共 {totalArticles} 篇，第 {batchIndex}/{totalBatches} 批）：
{articles}`,
			system:    `你是科技新闻分析师，负责提炼文章的核心观点和立场。输出必须是中文 JSON，严格按格式来，不要有任何额外解释。`,
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
