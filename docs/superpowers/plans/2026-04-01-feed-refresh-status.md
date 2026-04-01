# 订阅源刷新状态展示实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 刷新完成后，侧边栏 feed 列表显示：失败❌标记、成功+新文章数标签

**Architecture:**
- 数据库：feeds 表新增 3 列存储刷新结果
- 后端：`fetchArticles` 返回新文章数，callback 扩展 per-feed 结果并持久化
- 前端：Feed 列表项显示状态图标/标签

**Tech Stack:** Go + SQLite + React + TypeScript

---

## Task 1: 数据库迁移

**Files:**
- Modify: `internal/repository/sqlite/db.go`

- [ ] **Step 1: 添加迁移 SQL**

```go
// internal/repository/sqlite/db.go
// 在 InitDB 函数中，feeds 表创建语句后添加：

_, err = DB.Exec(`ALTER TABLE feeds ADD COLUMN last_refresh_success INTEGER DEFAULT 0`)
if err != nil && !strings.Contains(err.Error(), "duplicate column") {
    return err
}
_, err = DB.Exec(`ALTER TABLE feeds ADD COLUMN last_refresh_error TEXT DEFAULT ''`)
if err != nil && !strings.Contains(err.Error(), "duplicate column") {
    return err
}
_, err = DB.Exec(`ALTER TABLE feeds ADD COLUMN last_refreshed TEXT`)
if err != nil && !strings.Contains(err.Error(), "duplicate column") {
    return err
}
```

---

## Task 2: Feed 模型更新

**Files:**
- Modify: `internal/models/models.go:5-15`

- [ ] **Step 1: 更新 Feed 结构体**

```go
type Feed struct {
    ID                int64     `json:"id"`
    Title             string    `json:"title"`
    URL               string    `json:"url"`
    Description       string    `json:"description"`
    IconURL           string    `json:"icon_url"`
    LastFetched       time.Time `json:"last_fetched"`
    IsDead            bool      `json:"is_dead"` // true if feed returned 404/410
    CreatedAt         time.Time `json:"created_at"`
    Group             string    `json:"group"` // feed group/folder name, "" means ungrouped
    LastRefreshSuccess int       `json:"last_refresh_success"` // 新文章数，-1=失败
    LastRefreshError   string    `json:"last_refresh_error"`   // 失败错误信息
    LastRefreshed     time.Time `json:"last_refreshed"`       // 最后刷新时间
}
```

---

## Task 3: FeedRepository 更新

**Files:**
- Modify: `internal/repository/sqlite/feed_repository.go`

- [ ] **Step 1: 更新 GetAll 查询**

修改 `GetAll()` 中的 SELECT 语句，添加新列：
```go
`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, COALESCE(group_name, ''),
        last_refresh_success, COALESCE(last_refresh_error, ''), last_refreshed
 FROM feeds ORDER BY created_at DESC`
```

在 Scan 中添加：
```go
var lastRefreshed sql.NullString
err := rows.Scan(..., &f.LastRefreshSuccess, &f.LastRefreshError, &lastRefreshed)
if lastRefreshed.Valid {
    f.LastRefreshed, _ = time.Parse(time.RFC3339, lastRefreshed.String)
}
```

- [ ] **Step 2: 更新 GetByID 查询**

同样在 `GetByID()` 中添加新列扫描。

- [ ] **Step 3: 添加 UpdateRefreshResult 方法**

```go
func (r *FeedRepository) UpdateRefreshResult(id int64, success int, errorMsg string) error {
    _, err := DB.Exec(
        `UPDATE feeds SET last_refresh_success = ?, last_refresh_error = ?, last_refreshed = ? WHERE id = ?`,
        success, errorMsg, time.Now().Format(time.RFC3339), id,
    )
    return err
}
```

---

## Task 4: fetchArticles 返回新文章数

**Files:**
- Modify: `internal/service/rss_service.go:69-117`

- [ ] **Step 1: 修改 fetchArticles 返回类型**

```go
// 返回：新抓取的 article 数量
func (s *RSSService) fetchArticles(feed *models.Feed) (int, error) {
    articles, err := s.parser.ParseURL(feed.URL)
    if err != nil {
        return 0, err
    }

    newCount := 0
    for _, item := range articles.Items {
        // ... 现有逻辑 ...
        exists, _ := s.articleRepo.LinkExists(article.Link)
        if !exists {
            if err := s.articleRepo.Create(article); err != nil {
                log.Printf("warning: failed to save article %s: %v", article.Title, err)
            } else {
                newCount++ // 只在成功创建时计数
            }
        }
    }
    return newCount, nil
}
```

---

## Task 5: RefreshAllFeedsWithProgress callback 扩展

**Files:**
- Modify: `internal/service/rss_service.go:136-191`

- [ ] **Step 1: 更新 callback 签名**

```go
// 原签名：
onProgress(current, total int, feedTitle string, success, failed int)

// 新签名：
onProgress(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string)
```

- [ ] **Step 2: 更新调用处**

在 goroutine 中调用时：
```go
newCount, err := s.fetchArticles(feed)
if err != nil {
    // 标记失败
    onProgress(idx+1, total, f.Title, f.ID, -1, err.Error())
} else {
    // 标记成功
    onProgress(idx+1, total, f.Title, f.ID, newCount, "")
}
```

---

## Task 6: 后端刷新处理和 API 响应

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 扩展 RefreshStatus 存储 per-feed 结果**

```go
// internal/events/events.go
type RefreshStatus struct {
    Mutex      sync.Mutex
    InProgress  bool
    Current    int
    Total      int
    FeedTitle  string
    Success    int
    Failed     int
    Error      string
    // 新增：per-feed 结果
    Results    map[int64]FeedRefreshResult
}

type FeedRefreshResult struct {
    FeedID    int64  `json:"feedId"`
    Title     string `json:"title"`
    Success   bool   `json:"success"`
    NewCount  int    `json:"newCount"`  // -1 表示失败
    Error     string `json:"error"`
}
```

- [ ] **Step 2: 更新 handleRefreshAllFeeds 中的 callback**

```go
err := rssService.RefreshAllFeedsWithProgress(func(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string) {
    events.GlobalRefreshStatus.Mutex.Lock()
    events.GlobalRefreshStatus.Current = success + failed
    events.GlobalRefreshStatus.Total = total
    events.GlobalRefreshStatus.FeedTitle = feedTitle
    if newCount < 0 {
        events.GlobalRefreshStatus.Failed++
        events.GlobalRefreshStatus.Results[feedId] = FeedRefreshResult{
            FeedID: feedId, Title: feedTitle, Success: false, NewCount: -1, Error: errMsg,
        }
        s.feedRepo.UpdateRefreshResult(feedId, -1, errMsg)
    } else {
        events.GlobalRefreshStatus.Success++
        events.GlobalRefreshStatus.Results[feedId] = FeedRefreshResult{
            FeedID: feedId, Title: feedTitle, Success: true, NewCount: newCount, Error: "",
        }
        s.feedRepo.UpdateRefreshResult(feedId, newCount, "")
    }
    events.GlobalRefreshStatus.Mutex.Unlock()
})
```

- [ ] **Step 3: 更新 GET /api/refresh/status 响应**

```go
writeJSON(w, http.StatusOK, map[string]interface{}{
    "inProgress": events.GlobalRefreshStatus.InProgress,
    "current":    events.GlobalRefreshStatus.Current,
    "total":      events.GlobalRefreshStatus.Total,
    "feedTitle":  events.GlobalRefreshStatus.FeedTitle,
    "success":    events.GlobalRefreshStatus.Success,
    "failed":     events.GlobalRefreshStatus.Failed,
    "error":      events.GlobalRefreshStatus.Error,
    // 新增
    "results":    events.GlobalRefreshStatus.Results,
})
```

---

## Task 7: 前端 API 类型更新

**Files:**
- Modify: `frontend/src/api.ts`

- [ ] **Step 1: 添加新类型**

```typescript
export interface FeedRefreshResult {
  feedId: number
  title: string
  success: boolean
  newCount: number
  error: string
}

export interface RefreshStatusResponse {
  inProgress: boolean
  current: number
  total: number
  feedTitle: string
  success: number
  failed: number
  error: string
  results?: FeedRefreshResult[]
}
```

---

## Task 8: FeedList UI 更新

**Files:**
- Modify: `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: 扩展 Feed 类型**

```typescript
interface Feed {
  id: number
  title: string
  url: string
  // ... 现有字段 ...
  last_refresh_success: number  // -1=失败, 0=成功无新文章, >0=成功有新文章
  last_refresh_error: string
  last_refreshed: string
}
```

- [ ] **Step 2: 更新 feed-item 渲染逻辑**

```tsx
// 在 feeds.map 中，feed-item-info 后添加状态显示：
<div className="feed-item-status">
  {feed.last_refresh_success === -1 && (
    <span className="status-failed" title={feed.last_refresh_error}>❌</span>
  )}
  {feed.last_refresh_success > 0 && (
    <span className="status-new">+{feed.last_refresh_success}</span>
  )}
  {feed.last_refresh_success === 0 && (
    <span className="status-ok">✅</span>
  )}
</div>
```

- [ ] **Step 3: 刷新完成后更新本地 feed 状态**

从 `results` 数组中读取每个 feed 的刷新结果，更新对应 feed 的 `last_refresh_success` 和 `last_refresh_error` 字段。

- [ ] **Step 4: 添加 CSS 样式**

```css
.feed-item-status {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}
.status-failed { color: #ef4444; cursor: help; }
.status-new { color: #3b82f6; font-size: 0.75rem; }
.status-ok { color: #22c55e; }
```

---

## 验证步骤

- [ ] 刷新 10 个 feed，验证失败 feed 显示❌
- [ ] 刷新后验证成功 feed 显示 `+N` 标签
- [ ] 刷新 0 新文章的 feed，验证显示✅无标签
- [ ] 查看数据库 feeds 表新列有正确数据
