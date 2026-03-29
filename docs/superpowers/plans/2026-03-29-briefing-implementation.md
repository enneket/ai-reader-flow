# 简报功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现简报功能 - AI 聚合每日订阅源文章生成主题简报

**Architecture:** 基于现有的 Go backend + React frontend 架构，新增 Briefing 数据模型、Service 和 API，前端新增简报页面

**Tech Stack:** Go, SQLite, React, TypeScript, 现有 AI Provider (OpenAI/Claude/Ollama)

---

## 文件结构

### Backend (Go)

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/models/models.go` | Modify | 新增 Briefing, BriefingItem, BriefingArticle 模型 |
| `internal/repository/sqlite/db.go` | Modify | 新增 3 个表的迁移 SQL |
| `internal/repository/sqlite/briefing_repository.go` | Create | 简报 CRUD Repository |
| `internal/service/briefing_service.go` | Create | 简报生成 Service (GenerateBriefing) |
| `cmd/server/main.go` | Modify | 新增 briefing API endpoints, 修改 cron |

### Frontend (React)

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/components/Briefing.tsx` | Create | 简报主页面组件 |
| `frontend/src/api.ts` | Modify | 新增 briefing API 方法 |
| `frontend/src/App.tsx` | Modify | 路由改为 /, /feeds, /settings |
| `frontend/src/components/ArticleList.tsx` | Delete | 移除（被 Briefing 替代）|
| `frontend/src/components/NoteList.tsx` | Delete | 移除 |

---

## 实施任务

### Task 1: 数据库迁移

**Files:**
- Modify: `internal/repository/sqlite/db.go:90-100`

**Steps:**

- [ ] **Step 1: 添加迁移 SQL**

在 `db.go` 的 `schema` 变量中添加:

```go
// briefings table
`CREATE TABLE IF NOT EXISTS briefings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status TEXT DEFAULT 'pending',
    error TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT
);`,

// briefing_items table
`CREATE TABLE IF NOT EXISTS briefing_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    briefing_id INTEGER NOT NULL,
    topic TEXT NOT NULL,
    summary TEXT,
    sort_order INTEGER DEFAULT 0,
    FOREIGN KEY (briefing_id) REFERENCES briefings(id) ON DELETE CASCADE
);`,

// briefing_articles table
`CREATE TABLE IF NOT EXISTS briefing_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    briefing_item_id INTEGER NOT NULL,
    article_id INTEGER NOT NULL,
    title TEXT,
    FOREIGN KEY (briefing_item_id) REFERENCES briefing_items(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);`,
```

- [ ] **Step 2: 添加索引**

```go
`CREATE INDEX IF NOT EXISTS idx_briefing_items_briefing_id ON briefing_items(briefing_id);`,
`CREATE INDEX IF NOT EXISTS idx_briefing_articles_briefing_item_id ON briefing_articles(briefing_item_id);`,
`CREATE INDEX IF NOT EXISTS idx_briefings_created_at ON briefings(created_at);`,
```

- [ ] **Step 3: 验证数据库**

Run: `go build ./cmd/server/`
Expected: Build success

---

### Task 2: 数据模型

**Files:**
- Modify: `internal/models/models.go`

**Steps:**

- [ ] **Step 1: 添加 Briefing 模型**

在 `models.go` 末尾添加:

```go
// Briefing is an AI-generated daily briefing
type Briefing struct {
    ID          int64      `json:"id"`
    Status      string     `json:"status"` // pending, generating, completed, failed
    Error       string     `json:"error,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// BriefingItem is a topic within a briefing
type BriefingItem struct {
    ID         int64            `json:"id"`
    BriefingID int64           `json:"briefing_id"`
    Topic     string          `json:"topic"`
    Summary   string          `json:"summary"`
    SortOrder int             `json:"sort_order"`
    Articles  []BriefingArticle `json:"articles"`
}

// BriefingArticle is a reference to an article within a briefing item
type BriefingArticle struct {
    ID            int64  `json:"id"`
    BriefingItemID int64 `json:"briefing_item_id"`
    ArticleID     int64  `json:"article_id"`
    Title         string `json:"title"`
}

// BriefingTopic is the AI output format for a topic
type BriefingTopic struct {
    Name       string   `json:"name"`
    ArticleIDs []int64  `json:"article_ids"`
    Summary    string   `json:"summary"`
}

// BriefingResult is the AI output format
type BriefingResult struct {
    Topics []BriefingTopic `json:"topics"`
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/models/`
Expected: Build success

---

### Task 3: Briefing Repository

**Files:**
- Create: `internal/repository/sqlite/briefing_repository.go`

**Steps:**

- [ ] **Step 1: 创建 Repository**

```go
package sqlite

import (
    "ai-rss-reader/internal/models"
    "database/sql"
    "time"
)

type BriefingRepository struct{}

func NewBriefingRepository() *BriefingRepository {
    return &BriefingRepository{}
}

func (r *BriefingRepository) Create(b *models.Briefing) error {
    result, err := DB.Exec(
        `INSERT INTO briefings (status, created_at) VALUES (?, ?)`,
        b.Status, time.Now().Format(time.RFC3339),
    )
    if err != nil {
        return err
    }
    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    b.ID = id
    return nil
}

func (r *BriefingRepository) GetByID(id int64) (*models.Briefing, error) {
    row := DB.QueryRow(
        `SELECT id, status, error, created_at, completed_at FROM briefings WHERE id = ?`,
        id,
    )
    var b models.Briefing
    var createdAt, completedAt string
    err := row.Scan(&b.ID, &b.Status, &b.Error, &createdAt, &completedAt)
    if err != nil {
        return nil, err
    }
    b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
    if completedAt != "" {
        t, _ := time.Parse(time.RFC3339, completedAt)
        b.CompletedAt = &t
    }
    return &b, nil
}

func (r *BriefingRepository) GetAll(limit, offset int) ([]models.Briefing, error) {
    rows, err := DB.Query(
        `SELECT id, status, error, created_at, completed_at FROM briefings ORDER BY created_at DESC LIMIT ? OFFSET ?`,
        limit, offset,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var briefings []models.Briefing
    for rows.Next() {
        var b models.Briefing
        var createdAt, completedAt string
        err := rows.Scan(&b.ID, &b.Status, &b.Error, &createdAt, &completedAt)
        if err != nil {
            continue
        }
        b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
        if completedAt != "" {
            t, _ := time.Parse(time.RFC3339, completedAt)
            b.CompletedAt = &t
        }
        briefings = append(briefings, b)
    }
    return briefings, nil
}

func (r *BriefingRepository) UpdateStatus(id int64, status string, errMsg string) error {
    var completedAt string
    if status == "completed" || status == "failed" {
        completedAt = time.Now().Format(time.RFC3339)
    }
    _, err := DB.Exec(
        `UPDATE briefings SET status = ?, error = ?, completed_at = ? WHERE id = ?`,
        status, errMsg, completedAt, id,
    )
    return err
}

func (r *BriefingRepository) Delete(id int64) error {
    _, err := DB.Exec(`DELETE FROM briefings WHERE id = ?`, id)
    return err
}

func (r *BriefingRepository) CreateItem(item *models.BriefingItem) error {
    result, err := DB.Exec(
        `INSERT INTO briefing_items (briefing_id, topic, summary, sort_order) VALUES (?, ?, ?, ?)`,
        item.BriefingID, item.Topic, item.Summary, item.SortOrder,
    )
    if err != nil {
        return err
    }
    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    item.ID = id
    return nil
}

func (r *BriefingRepository) GetItemsByBriefingID(briefingID int64) ([]models.BriefingItem, error) {
    rows, err := DB.Query(
        `SELECT id, briefing_id, topic, summary, sort_order FROM briefing_items WHERE briefing_id = ? ORDER BY sort_order`,
        briefingID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []models.BriefingItem
    for rows.Next() {
        var item models.BriefingItem
        err := rows.Scan(&item.ID, &item.BriefingID, &item.Topic, &item.Summary, &item.SortOrder)
        if err != nil {
            continue
        }
        items = append(items, item)
    }
    return items, nil
}

func (r *BriefingRepository) CreateArticle(article *models.BriefingArticle) error {
    result, err := DB.Exec(
        `INSERT INTO briefing_articles (briefing_item_id, article_id, title) VALUES (?, ?, ?)`,
        article.BriefingItemID, article.ArticleID, article.Title,
    )
    if err != nil {
        return err
    }
    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    article.ID = id
    return nil
}

func (r *BriefingRepository) GetArticlesByItemID(itemID int64) ([]models.BriefingArticle, error) {
    rows, err := DB.Query(
        `SELECT id, briefing_item_id, article_id, title FROM briefing_articles WHERE briefing_item_id = ?`,
        itemID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var articles []models.BriefingArticle
    for rows.Next() {
        var a models.BriefingArticle
        err := rows.Scan(&a.ID, &a.BriefingItemID, &a.ArticleID, &a.Title)
        if err != nil {
            continue
        }
        articles = append(articles, a)
    }
    return articles, nil
}

// GetLatestBriefingTime returns the created_at of the most recent completed briefing
func (r *BriefingRepository) GetLatestBriefingTime() (*time.Time, error) {
    row := DB.QueryRow(
        `SELECT created_at FROM briefings WHERE status = 'completed' ORDER BY created_at DESC LIMIT 1`,
    )
    var createdAt string
    err := row.Scan(&createdAt)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    t, _ := time.Parse(time.RFC3339, createdAt)
    return &t, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/repository/sqlite/`
Expected: Build success

---

### Task 4: Briefing Service

**Files:**
- Create: `internal/service/briefing_service.go`

**Steps:**

- [ ] **Step 1: 创建 BriefingService**

```go
package service

import (
    "ai-rss-reader/internal/ai"
    "ai-rss-reader/internal/models"
    "ai-rss-reader/internal/repository/sqlite"
    "encoding/json"
    "fmt"
    "log"
    "strings"
)

type BriefingService struct {
    briefingRepo *sqlite.BriefingRepository
    articleRepo  *sqlite.ArticleRepository
    feedRepo     *sqlite.FeedRepository
}

func NewBriefingService() *BriefingService {
    return &BriefingService{
        briefingRepo: sqlite.NewBriefingRepository(),
        articleRepo:  sqlite.NewArticleRepository(),
        feedRepo:     sqlite.NewFeedRepository(),
    }
}

// GenerateBriefing creates a new briefing from recent articles
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
    // 1. Create briefing record
    briefing := &models.Briefing{
        Status: "generating",
    }
    if err := s.briefingRepo.Create(briefing); err != nil {
        return nil, fmt.Errorf("create briefing: %w", err)
    }

    // 2. Get recent articles (since last briefing or last 24 hours)
    articles, err := s.articleRepo.GetRecentForBriefing()
    if err != nil {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
        return nil, fmt.Errorf("get articles: %w", err)
    }

    if len(articles) == 0 {
        s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")
        return briefing, nil
    }

    // 3. Build articles input for AI
    articlesInput := s.buildArticlesInput(articles)

    // 4. Call AI to generate topics
    provider := ai.GetProvider()
    prompt := s.buildPrompt(articlesInput)

    result, err := provider.GenerateBriefing(prompt)
    if err != nil {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
        return nil, fmt.Errorf("AI generation: %w", err)
    }

    // 5. Parse AI result
    var briefingResult models.BriefingResult
    if err := json.Unmarshal([]byte(result), &briefingResult); err != nil {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", "invalid AI response")
        return nil, fmt.Errorf("parse AI result: %w", err)
    }

    // 6. Store briefing items
    for i, topic := range briefingResult.Topics {
        item := &models.BriefingItem{
            BriefingID: briefing.ID,
            Topic:      topic.Name,
            Summary:    topic.Summary,
            SortOrder:  i,
        }
        if err := s.briefingRepo.CreateItem(item); err != nil {
            log.Printf("Warning: failed to create briefing item: %v", err)
            continue
        }

        // Store article references
        for _, articleID := range topic.ArticleIDs {
            // Find article title
            title := ""
            for _, a := range articles {
                if a.ID == articleID {
                    title = a.Title
                    break
                }
            }
            ba := &models.BriefingArticle{
                BriefingItemID: item.ID,
                ArticleID:     articleID,
                Title:         title,
            }
            s.briefingRepo.CreateArticle(ba)
        }
    }

    // 7. Mark as completed
    s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")

    return briefing, nil
}

func (s *BriefingService) buildArticlesInput(articles []models.Article) string {
    var sb strings.Builder
    for _, a := range articles {
        sb.WriteString(fmt.Sprintf("文章 ID: %d\n", a.ID))
        sb.WriteString(fmt.Sprintf("标题: %s\n", a.Title))
        summary := a.Summary
        if summary == "" {
            summary = a.Content
            if len(summary) > 200 {
                summary = summary[:200] + "..."
            }
        }
        sb.WriteString(fmt.Sprintf("摘要: %s\n", summary))
        sb.WriteString("---\n")
    }
    return sb.String()
}

func (s *BriefingService) buildPrompt(articlesInput string) string {
    return fmt.Sprintf(`System: 你是一个内容策划助手。给定一组文章，你需要：
1. 将文章按主题分组（相似内容的文章分到同一组）
2. 为每个主题起一个简短的名字（如"AI"、"创业"、"科技"）
3. 为每个主题提取核心观点（用简洁的 bullets，每条不超过 20 字）

输出格式（严格按 JSON 格式，不要有其他内容）：
{
  "topics": [
    {
      "name": "主题名称",
      "article_ids": [101, 102],
      "summary": "• 核心观点1\n• 核心观点2\n• 核心观点3"
    }
  ]
}

规则：
- 每个简报最多 5 个主题
- 每个主题最多 5 篇核心文章
- 只包含真正有价值的文章，无关内容请忽略
- 主题按文章数量排序（多的在前）
- 如果文章太少或无价值，返回空的 topics 数组

User: 以下是今天的文章：
%s`, articlesInput)
}

// GetBriefingWithItems returns a briefing with all its items and articles
func (s *BriefingService) GetBriefingWithItems(id int64) (*models.Briefing, error) {
    briefing, err := s.briefingRepo.GetByID(id)
    if err != nil {
        return nil, err
    }

    items, err := s.briefingRepo.GetItemsByBriefingID(id)
    if err != nil {
        return nil, err
    }

    for i := range items {
        articles, err := s.briefingRepo.GetArticlesByItemID(items[i].ID)
        if err != nil {
            continue
        }
        items[i].Articles = articles
    }

    briefing.Items = items
    return briefing, nil
}

// GetAllBriefings returns all briefings
func (s *BriefingService) GetAllBriefings(limit, offset int) ([]models.Briefing, error) {
    return s.briefingRepo.GetAll(limit, offset)
}

// DeleteBriefing deletes a briefing
func (s *BriefingService) DeleteBriefing(id int64) error {
    return s.briefingRepo.Delete(id)
}
```

- [ ] **Step 2: 添加 ArticleRepository 方法**

需要在 `internal/repository/sqlite/article_repository.go` 中添加:

```go
// GetRecentForBriefing returns recent unread articles
func (r *ArticleRepository) GetRecentForBriefing() ([]models.Article, error) {
    rows, err := DB.Query(
        `SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, embedding, COALESCE(quality_score, 0)
         FROM articles
         WHERE status = 'unread'
         ORDER BY created_at DESC
         LIMIT 100`,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return r.scanArticles(rows)  // Reuse existing scanner
}
```

- [ ] **Step 3: 添加 AI Provider 方法**

在 `internal/ai/provider.go` 的 AIServiceProvider 接口中添加:

```go
GenerateBriefing(prompt string) (string, error)
```

并在各 Provider 实现中添加方法。以下是 OpenAI 的实现:

```go
func (p *OpenAIProvider) GenerateBriefing(prompt string) (string, error) {
    reqBody := map[string]interface{}{
        "model": p.Model,
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
        "max_tokens": 2000,
    }

    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.APIKey)

    client := &http.Client{Timeout: 120 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return "", err
    }

    if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
        if choice, ok := choices[0].(map[string]interface{}); ok {
            if msg, ok := choice["message"].(map[string]interface{}); ok {
                if content, ok := msg["content"].(string); ok {
                    return content, nil
                }
            }
        }
    }

    return "", fmt.Errorf("unexpected response format")
}
```

同样为 `ClaudeProvider` 和 `OllamaProvider` 添加 `GenerateBriefing` 方法。

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: Build success

---

### Task 5: API Endpoints

**Files:**
- Modify: `cmd/server/main.go`

**Steps:**

- [ ] **Step 1: 添加路由**

在 main.go 中添加:

```go
// Briefings
mux.HandleFunc("GET /api/briefings", handleGetBriefings)
mux.HandleFunc("GET /api/briefings/{id}", handleGetBriefing)
mux.HandleFunc("POST /api/briefings/generate", handleGenerateBriefing)
mux.HandleFunc("DELETE /api/briefings/{id}", handleDeleteBriefing)
mux.HandleFunc("GET /api/briefings/{id}/status", handleGetBriefingStatus)
```

- [ ] **Step 2: 添加 Handler 函数**

```go
// ─── Briefing Handlers ─────────────────────────────────────────────────────────

func handleGetBriefings(w http.ResponseWriter, r *http.Request) {
    limit := int(parseQueryInt(r, "limit", 20))
    offset := int(parseQueryInt(r, "offset", 0))
    briefings, _ := briefingService.GetAllBriefings(limit, offset)
    writeJSON(w, http.StatusOK, briefings)
}

func handleGetBriefing(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/briefings", r)
    if !ok {
        return
    }
    briefing, err := briefingService.GetBriefingWithItems(id)
    if err != nil {
        http.Error(w, "briefing not found", http.StatusNotFound)
        return
    }
    writeJSON(w, http.StatusOK, briefing)
}

func handleGenerateBriefing(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        // SSE or polling for status - just return 202
        w.WriteHeader(http.StatusAccepted)
        return
    }

    // Create briefing in background
    go func() {
        briefingService.GenerateBriefing()
    }()

    w.WriteHeader(http.StatusAccepted)
}

func handleDeleteBriefing(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/briefings", r)
    if !ok {
        return
    }
    briefingService.DeleteBriefing(id)
    w.WriteHeader(http.StatusNoContent)
}

func handleGetBriefingStatus(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/briefings", r)
    if !ok {
        return
    }
    briefing, err := briefingService.GetBriefingWithItems(id)
    if err != nil {
        http.Error(w, "briefing not found", http.StatusNotFound)
        return
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "status":    briefing.Status,
        "error":     briefing.Error,
        "created_at": briefing.CreatedAt,
    })
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./cmd/server/`
Expected: Build success

---

### Task 6: 前端 Briefing 组件

**Files:**
- Create: `frontend/src/components/Briefing.tsx`

**Steps:**

- [ ] **Step 1: 创建 Briefing 组件**

```tsx
import {useState, useEffect} from 'react'
import {FileText, RefreshCw, Settings, LayoutGrid} from 'lucide-react'
import {useNavigate, Link} from 'react-router-dom'
import {api} from '../api'

interface BriefingItem {
  id: number
  topic: string
  summary: string
  articles: {id: number; title: string}[]
}

interface Briefing {
  id: number
  status: string
  created_at: string
  items: BriefingItem[]
}

export function Briefing() {
  const navigate = useNavigate()
  const [briefings, setBriefings] = useState<Briefing[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)

  useEffect(() => {
    loadBriefings()
  }, [])

  const loadBriefings = async () => {
    setLoading(true)
    try {
      const data = await api.getBriefings()
      setBriefings(data || [])
    } catch (err) {
      console.error('Failed to load briefings:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await api.generateBriefing()
      await loadBriefings()
    } catch (err) {
      console.error('Failed to generate briefing:', err)
    } finally {
      setGenerating(false)
    }
  }

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    })
  }

  return (
    <div className="app">
      <div className="app-body">
        <aside className="sidebar">
          {/* Same sidebar as other pages */}
          <nav className="sidebar-nav">
            <Link to="/feeds" className="nav-item">
              <LayoutGrid />
              <span>订阅源</span>
            </Link>
            <Link to="/" className="nav-item active">
              <FileText />
              <span>简报</span>
            </Link>
            <Link to="/settings" className="nav-item">
              <Settings />
              <span>设置</span>
            </Link>
          </nav>
        </aside>

        <main className="app-main">
          <div className="page-content">
            <div className="briefing-header">
              <h1>简报</h1>
              <button
                onClick={handleGenerate}
                disabled={generating}
                className="btn btn-primary"
              >
                <RefreshCw size={16} className={generating ? 'spinning' : ''} />
                {generating ? '生成中...' : '立即生成简报'}
              </button>
            </div>

            {loading ? (
              <div className="loading">加载中...</div>
            ) : briefings.length === 0 ? (
              <div className="empty-state">
                <FileText size={48} />
                <p>暂无简报</p>
                <p style={{fontSize: '0.9rem', color: 'var(--text-secondary)'}}>
                  点击上方按钮立即生成
                </p>
              </div>
            ) : (
              <div className="briefing-list">
                {briefings.map((briefing) => (
                  <div key={briefing.id} className="briefing-card">
                    <div className="briefing-card-header">
                      <span>{formatTime(briefing.created_at)}</span>
                      <span className={`status-badge status-${briefing.status}`}>
                        {briefing.status === 'generating' ? '生成中' :
                         briefing.status === 'completed' ? '已完成' : '失败'}
                      </span>
                    </div>
                    {briefing.status === 'completed' && briefing.items && (
                      <div className="briefing-items">
                        {briefing.items.map((item) => (
                          <div key={item.id} className="briefing-item">
                            <h3>{item.topic} ({item.articles.length}篇)</h3>
                            <p className="briefing-summary">{item.summary}</p>
                            <ul className="briefing-articles">
                              {item.articles.map((article) => (
                                <li key={article.id}>{article.title}</li>
                              ))}
                            </ul>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: 添加 API 方法**

在 `frontend/src/api.ts` 中添加:

```typescript
getBriefings: () => request<Briefing[]>('/briefings'),
generateBriefing: () => request<void>('/briefings/generate', {method: 'POST'}),
```

- [ ] **Step 3: 更新路由**

在 `frontend/src/App.tsx` 中:

```tsx
<Routes>
  <Route path="/" element={<Briefing />} />
  <Route path="/feeds" element={<FeedList />} />
  <Route path="/settings" element={<Settings />} />
</Routes>
```

- [ ] **Step 4: 删除旧组件**

删除:
- `frontend/src/components/ArticleList.tsx`
- `frontend/src/components/NoteList.tsx`

- [ ] **Step 5: 验证编译**

Run: `cd frontend && npm run build`
Expected: Build success

---

### Task 7: Cron 调度

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/config/config.go`

**Steps:**

- [ ] **Step 1: 添加 cron 库依赖**

Run: `go get github.com/robfig/cron/v3`

- [ ] **Step 2: 添加 Hour 字段到 CronConfig**

在 `internal/config/config.go` 中修改 `CronConfig`:

```go
type CronConfig struct {
    Enabled    bool `toml:"enabled"`
    IntervalMins int `toml:"interval_mins"`
    Hour int `toml:"hour"` // 每日触发小时 (0-23)
    Minute int `toml:"minute"` // 每日触发分钟 (0-59)
}
```

并更新默认值:

```go
Cron: CronConfig{
    Enabled:     true,
    IntervalMins: 30,
    Hour: 9, // 默认早上 9 点
    Minute: 0,
},
```

- [ ] **Step 3: 替换现有的 cron 逻辑**

现有的 cron 在 `main.go:152-177`，需要替换为简报生成的 cron:

```go
import "github.com/robfig/cron/v3"

// Schedule briefing generation
if cfg.Cron.Enabled {
    c := cron.New()
    // Run at specific time each day
    schedule := fmt.Sprintf("0 %d %d * * *", cfg.Cron.Minute, cfg.Cron.Hour)
    c.AddFunc(schedule, func() {
        log.Printf("[cron] Generating daily briefing at %02d:%02d", cfg.Cron.Hour, cfg.Cron.Minute)
        briefingService.GenerateBriefing()
    })
    c.Start()
}
```

- [ ] **Step 4: 添加全局 briefingService 变量**

在 `cmd/server/main.go` 的全局变量中添加:

```go
var (
    rssService     *service.RSSService
    filterService  *service.FilterService
    summaryService *service.SummaryService
    noteService    *service.NoteService
    briefingService *service.BriefingService  // ADD THIS
    dataDir       string
)
```

并在初始化时创建:

```go
rssService = service.NewRSSService()
filterService = service.NewFilterService()
summaryService = service.NewSummaryService()
noteService = service.NewNoteService(notesDir)
briefingService = service.NewBriefingService()  // ADD THIS
```

- [ ] **Step 5: 验证编译**

Run: `go build ./cmd/server/`
Expected: Build success

---

### Task 8: 清理旧代码

**Files:**
- Delete: `frontend/src/components/ArticleList.tsx`
- Delete: `frontend/src/components/NoteList.tsx`
- Remove routes from `cmd/server/main.go` (optional, keep for backward compatibility)

**Steps:**

- [ ] **Step 1: 删除旧组件文件**

```bash
rm frontend/src/components/ArticleList.tsx
rm frontend/src/components/NoteList.tsx
```

- [ ] **Step 2: 从 App.tsx 移除导入**

---

## 验证

### Backend 测试

```bash
# 启动服务器
go run ./cmd/server/

# 测试 API
curl http://localhost:8080/api/briefings
curl -X POST http://localhost:8080/api/briefings/generate
```

### Frontend 测试

```bash
cd frontend
npm run dev
# 打开 http://localhost:5173
# 应该看到简报页面
```

---

## 风险和注意事项

1. **AI Prompt 调试**: 首次部署后需要观察 AI 输出质量，可能需要调整 prompt
2. **长文本处理**: 如果文章很多，可能需要分批处理避免 token 超限
3. **现有数据**: 旧数据（articles 表）保留，但不再用于显示
