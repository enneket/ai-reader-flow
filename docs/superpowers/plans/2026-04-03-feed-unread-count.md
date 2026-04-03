# Feed Unread Count Badge 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 订阅源列表 badge 显示真实未读数，点击文章后计数自动减 1

**Architecture:** `Feed.unread_count` 是 article status 的聚合缓存字段。article status 变更时（unread → accepted/rejected/snoozed），Repository 层同步更新对应 feed 的 `unread_count`。

**Tech Stack:** Go (backend), React + TypeScript (frontend), SQLite

---

## Task 1: DB Migration

**Files:**
- Modify: `internal/repository/sqlite/db.go`

- [ ] **Step 1: 添加 DB migration**

在 `db.go` 的 `migrations` map 中找到已有的迁移块，在其下方添加：

```go
if err := migrateTable(db, "feeds", "add_unread_count", func(db *sql.DB) error {
    _, err := db.Exec("ALTER TABLE feeds ADD COLUMN unread_count INTEGER NOT NULL DEFAULT 0")
    if err != nil && !strings.Contains(err.Error(), "duplicate column") {
        return err
    }
    return nil
}); err != nil {
    return err
}
```

- [ ] **Step 2: 验证 migration 执行不报错**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/repository/sqlite/db.go
git commit -m "feat(db): add unread_count column to feeds table"
```

---

## Task 2: Feed Model 新增字段

**Files:**
- Modify: `internal/models/models.go:5-18`

- [ ] **Step 1: 在 Feed struct 中添加 UnreadCount 字段**

```go
type Feed struct {
    // ... existing fields (lines 5-17) ...
    UnreadCount     int       `json:"unread_count"`  // 新增
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/models/models.go
git commit -m "feat(models): add UnreadCount to Feed struct"
```

---

## Task 3: FeedRepository SELECT 更新

**Files:**
- Modify: `internal/repository/sqlite/feed_repository.go`

- [ ] **Step 1: 更新 GetAll() 的 SELECT 语句**

在 `GetAll()` 方法中，找到 SELECT 语句，添加 `unread_count` 字段：

```go
// 原来的 SELECT（第 33 行附近）：
SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, `+"`group`"+`, last_refresh_success, last_refresh_error, last_refreshed

// 改为：
SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, `+"`group`"+`, last_refresh_success, last_refresh_error, last_refreshed, unread_count
```

同样在 `scanFeeds` 的 `row.Scan` 中添加 `&f.UnreadCount`（在 `&f.LastRefreshed` 之后）。

- [ ] **Step 2: 更新 GetByID() 的 SELECT 和 Scan**

在 `GetByID()` 中同样添加 `unread_count` 到 SELECT 和 `row.Scan`。

- [ ] **Step 3: 验证编译**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 4: Commit**

```bash
git add internal/repository/sqlite/feed_repository.go
git commit -m "feat(repo): add unread_count to Feed SELECT queries"
```

---

## Task 4: FeedRepository 新增计数方法

**Files:**
- Modify: `internal/repository/sqlite/feed_repository.go`

- [ ] **Step 1: 添加 UpdateUnreadCount 方法**

在文件末尾（`GetDeadFeeds` 之后）添加：

```go
// UpdateUnreadCount increments or decrements the unread count for a feed
func (r *FeedRepository) UpdateUnreadCount(feedId int64, delta int) error {
    _, err := DB.Exec(`UPDATE feeds SET unread_count = MAX(0, unread_count + ?) WHERE id = ?`, delta, feedId)
    return err
}

// RecalcUnreadCount recalculates unread_count from article table
func (r *FeedRepository) RecalcUnreadCount(feedId int64) error {
    var count int
    err := DB.QueryRow(`SELECT COUNT(*) FROM articles WHERE feed_id = ? AND status = 'unread'`, feedId).Scan(&count)
    if err != nil {
        return err
    }
    _, err = DB.Exec(`UPDATE feeds SET unread_count = ? WHERE id = ?`, count, feedId)
    return err
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/repository/sqlite/feed_repository.go
git commit -m "feat(repo): add UpdateUnreadCount and RecalcUnreadCount methods"
```

---

## Task 5: ArticleRepository SetStatus/Create 同步更新 unread_count

**Files:**
- Modify: `internal/repository/sqlite/article_repository.go`

- [ ] **Step 1: 修改 SetStatus 方法，同步更新 unread_count**

当前 `SetStatus`（第 165-168 行）：
```go
func (r *ArticleRepository) SetStatus(id int64, status string) error {
    _, err := DB.Exec(`UPDATE articles SET status = ? WHERE id = ?`, status, id)
    return err
}
```

替换为：
```go
func (r *ArticleRepository) SetStatus(id int64, status string) error {
    // 获取旧 status 和 feed_id，用于更新 unread_count
    var oldStatus string
    var feedId int64
    err := DB.QueryRow(`SELECT status, feed_id FROM articles WHERE id = ?`, id).Scan(&oldStatus, &feedId)
    if err != nil {
        return err
    }

    _, err = DB.Exec(`UPDATE articles SET status = ? WHERE id = ?`, status, id)
    if err != nil {
        return err
    }

    // 同步更新 feed 的 unread_count
    if oldStatus == "unread" && status != "unread" {
        DB.Exec(`UPDATE feeds SET unread_count = MAX(0, unread_count - 1) WHERE id = ?`, feedId)
    } else if oldStatus != "unread" && status == "unread" {
        DB.Exec(`UPDATE feeds SET unread_count = unread_count + 1 WHERE id = ?`, feedId)
    }

    return nil
}
```

- [ ] **Step 2: 修改 Create 方法，article 初始为 unread 时更新 feed 的 unread_count**

当前 `Create` 方法结尾（第 32-36 行）：
```go
    id, _ := result.LastInsertId()
    article.ID = id
    // Index in FTS for full-text search
    _ = IndexArticle(article.ID, article.Title, article.Content)
    return nil
```

在 `return nil` 前添加：
```go
    // 同步更新 feed 的 unread_count
    if status == "unread" {
        DB.Exec(`UPDATE feeds SET unread_count = unread_count + 1 WHERE id = ?`, article.FeedID)
    }
```

- [ ] **Step 3: 验证编译**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 4: Commit**

```bash
git add internal/repository/sqlite/article_repository.go
git commit -m "feat(repo): sync unread_count on article status changes"
```

---

## Task 6: 前端 Feed Badge 联动

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: Feed interface 添加 unread_count**

在 `api.ts` 的 `Feed` interface 中添加：

```ts
export interface Feed {
  // ... existing fields ...
  unread_count: number  // 新增
}
```

- [ ] **Step 2: FeedList badge 改为 unread_count**

在 `FeedList.tsx` 中找到 `last_refresh_success` badge 逻辑（约第 389-393 行）：

原来的代码：
```tsx
{feed.last_refresh_success === -1 && (
  <span className="status-failed" title={feed.last_refresh_error}>❌</span>
)}
{feed.last_refresh_success > 0 && (
  <span className="status-new">+{feed.last_refresh_success}</span>
)}
```

替换为：
```tsx
{feed.last_refresh_success === -1 && (
  <span className="status-failed" title={feed.last_refresh_error}>❌</span>
)}
{feed.unread_count > 0 && (
  <span className="status-new">+{feed.unread_count}</span>
)}
```

- [ ] **Step 3: 点击文章时调用 acceptArticle，乐观更新 badge**

在 `handleArticleClick` 方法中（原来只是 setSelectedArticle），增加：

```tsx
const handleArticleClick = async (article: Article) => {
  setSelectedArticle(article)
  // 如果是未读文章，标记为已读并更新 badge
  if (article.status === 'unread') {
    try {
      await api.acceptArticle(article.id)
      setFeeds(prev => prev.map(f =>
        f.id === article.feed_id
          ? {...f, unread_count: Math.max(0, f.unread_count - 1)}
          : f
      ))
    } catch (err) {
      console.error('Failed to accept article:', err)
    }
  }
}
```

注意：确认 `Feed` 类型在 `FeedList.tsx` import 中包含 `unread_count`，以及 `Feed` 是 `api` 中导入的 `Feed` 类型。

- [ ] **Step 4: 验证构建**

Run: `cd frontend && npm run build`
Expected: 构建成功，无 TS 错误

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api.ts frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): show unread_count in feed badge and mark read on click"
```

---

## 验证步骤

部署后测试：
1. 刷新订阅源，badge 显示新增文章数
2. 点击一篇文章，badge 减 1
3. badge 为 0 时不显示标记
4. 刷新页面后计数与后端一致
