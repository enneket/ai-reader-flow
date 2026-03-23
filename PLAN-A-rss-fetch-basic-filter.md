# Plan A: RSS Fetch + Basic Filtering (This Week)

## Goal
RSS → AI 过滤 → 文章列表 端到端 Demo，验收标准：5 feeds + 100 articles，首篇到末篇 < 5min，单次刷新增量 < 30s。

## Scope

### In Scope
- [x] 并发 Feed Fetch（最多 5 个并行）
- [x] Feed 失效处理（404/410 → 标记 + 通知用户）
- [x] 重试策略（3 次，指数退避）
- [x] 文章状态：未读/已接受/已拒绝/稍后读
- [x] 前端：接受/拒绝/稍后读按钮
- [x] 3 栏布局（Feeds / Articles / Preview）

### NOT In Scope (→ Plan B)
- 语义去重（embedding + cosine similarity）
- 质量评分（0-100 分公式）
- batch embedding 优化
- AI 偏好学习

---

## Implementation

### Step 1: Model — 添加文章状态字段

**File:** `internal/models/models.go`

```go
// Article 新增字段
type Article struct {
    // ... existing fields ...
    Status    string `json:"status"` // "unread", "accepted", "rejected", "snoozed"
}
```

**Note:** `Status` 兼容现有 `IsFiltered`/`IsSaved` bool 字段，Phase 1 结束后统一清理。

### Step 2: RSS Service — 并发 Fetch + 失败处理

**File:** `internal/service/rss_service.go`

```go
func (s *RSSService) RefreshAllFeeds() error {
    feeds, err := s.feedRepo.GetAll()
    if err != nil {
        return err
    }

    // 并发：最多 5 个并行
    sem := make(chan struct{}, 5)
    var wg sync.WaitGroup
    var mu sync.Mutex
    errors := []error{}

    for _, feed := range feeds {
        wg.Add(1)
        go func(feed models.Feed) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            err := s.refreshFeedWithRetry(feed.ID)
            if err != nil {
                mu.Lock()
                errors = append(errors, fmt.Errorf("feed %s: %w", feed.Title, err))
                mu.Unlock()
            }
        }(feed)
    }

    wg.Wait()
    if len(errors) > 0 {
        return errors.Join(errors...) // Go 1.21+，用户一次看到所有失败
    }
    return nil
}

func (s *RSSService) refreshFeedWithRetry(feedID int64) error {
    const maxRetries = 3
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        err := s.RefreshFeed(feedID)
        if err == nil {
            return nil
        }
        lastErr = err
        // 不可恢复错误：404/410 → 标记 dead，跳出重试
        if isHTTPNotFound(err) {
            s.feedRepo.MarkDead(feedID)
            return fmt.Errorf("feed %d dead (404/410): %w", feedID, err)
        }
        time.Sleep(time.Duration(1<<uint(i)) * time.Second) // 指数退避: 1s, 2s, 4s
    }
    return lastErr
}

// isHTTPNotFound 检查 err 是否为 404/410
func isHTTPNotFound(err error) bool {
    if err == nil { return false }
    s := err.Error()
    return strings.Contains(s, "404") || strings.Contains(s, "410") ||
           strings.Contains(s, "not found") || strings.Contains(s, "Gone")
}
```

**Repository 新增方法** `internal/repository/sqlite/feed_repository.go`:
```go
func (r *FeedRepository) MarkDead(id int64) error
func (r *FeedRepository) GetDeadFeeds() ([]models.Feed, error)
```

### Step 3: Article Repository — 状态更新

**File:** `internal/repository/sqlite/article_repository.go`

```go
func (r *ArticleRepository) SetStatus(id int64, status string) error
func (r *ArticleRepository) GetByStatus(status string) ([]models.Article, error)
func (r *ArticleRepository) GetUnreadByFeed(feedID int64) ([]models.Article, error)
```

### Step 4: RSSService — 添加 SetArticleStatus

**File:** `internal/service/rss_service.go`

```go
func (s *RSSService) SetArticleStatus(id int64, status string) error {
    return s.articleRepo.SetStatus(id, status)
}
```

### Step 5: App 绑定 — 新方法

**File:** `app.go`

```go
// AcceptArticle marks article as accepted
func (a *App) AcceptArticle(id int64) error {
    return rssService.SetArticleStatus(id, "accepted")
}

// RejectArticle marks article as rejected
func (a *App) RejectArticle(id int64) error {
    return rssService.SetArticleStatus(id, "rejected")
}

// SnoozeArticle marks article as snoozed
func (a *App) SnoozeArticle(id int64) error {
    return rssService.SetArticleStatus(id, "snoozed")
}

// GetDeadFeeds returns feeds that returned 404/410
func (a *App) GetDeadFeeds() []models.Feed {
    feeds, _ := rssService.GetDeadFeeds()
    return feeds
}

// DeleteDeadFeed deletes a dead feed
func (a *App) DeleteDeadFeed(id int64) error {
    return rssService.DeleteFeed(id)
}
```

### Step 6: 前端 — 接受/拒绝/稍后读按钮

**File:** `frontend/src/components/ArticleList.tsx`

在 article content actions 区添加：

```tsx
<div className="article-content-actions">
  <button
    onClick={() => handleAccept(selectedArticle.id)}
    className="btn btn-primary"
    disabled={selectedArticle.status === 'accepted'}
  >
    <Check size={16} />
    {t('articles.accept')}
  </button>
  <button
    onClick={() => handleReject(selectedArticle.id)}
    className="btn btn-danger"
    disabled={selectedArticle.status === 'rejected'}
  >
    <X size={16} />
    {t('articles.reject')}
  </button>
  <button
    onClick={() => handleSnooze(selectedArticle.id)}
    className="btn btn-secondary"
    disabled={selectedArticle.status === 'snoozed'}
  >
    <Clock size={16} />
    {t('articles.snooze')}
  </button>
</div>
```

添加 handlers：
```tsx
const handleAccept = async (id: number) => {
  try {
    await AcceptArticle(id)
    await loadArticles()
  } catch (err) { setError(err.message) }
}
const handleReject = async (id: number) => { ... }
const handleSnooze = async (id: number) => { ... }
```

状态 badge 更新：
```tsx
<span className={`badge badge-${article.status}`}>
  {t(`articles.status.${article.status}`)}
</span>
```

### Step 7: 前端 — 失效 Feed Banner

**File:** `frontend/src/components/ArticleList.tsx`

在顶部添加：

```tsx
const [deadFeeds, setDeadFeeds] = useState<models.Feed[]>([])

useEffect(() => {
  const dead = GetDeadFeeds()
  setDeadFeeds(dead || [])
}, [])
```

```tsx
{deadFeeds.length > 0 && (
  <div className="dead-feeds-banner">
    <span>{t('feeds.deadWarning', { count: deadFeeds.length })}</span>
    <button onClick={() => showDeadFeedsModal()}>{t('feeds.viewDead')}</button>
  </div>
)}
```

### Step 8: i18n 文案

**Files:** `frontend/src/i18n/en.ts`, `frontend/src/i18n/zh.ts`

```ts
// Articles
articles: {
  accept: 'Accept',
  reject: 'Reject',
  snooze: 'Snooze',
  status: {
    unread: 'Unread',
    accepted: 'Accepted',
    rejected: 'Rejected',
    snoozed: 'Snoozed',
  }
},
// Feeds
feeds: {
  deadWarning: '{{count}} feed(s) no longer available (404/410)',
  viewDead: 'View',
}
```

---

## Data Flow

```
User clicks "Refresh All"
       │
       ▼
RSSService.RefreshAllFeeds()
  │ ← 5 feeds in parallel
  ├── Feed A → gofeed.Parse → articles → articleRepo.Create()
  ├── Feed B → 404 → MarkDead() → skip
  └── ...
       │
       ▼
FilterService.FilterAllArticles()
  │
  ├── For each article: AI yes/no (or error → log + show all)
  └── articleRepo.SetFiltered(article.ID, !shouldShow)
       │
       ▼
Frontend: articles displayed with status badges
       │
       ▼
User: accept / reject / snooze
       │
       ├── accept → article.status = "accepted"
       ├── reject → article.status = "rejected"
       └── snooze → article.status = "snoozed"
```

---

## Tests to Write

### Unit Tests

| # | File | What to Test |
|---|------|-------------|
| 1 | `rss_service_test.go` | `RefreshAllFeeds` — 5 feeds 并发执行 |
| 2 | `rss_service_test.go` | `refreshFeedWithRetry` — 成功不重试 |
| 3 | `rss_service_test.go` | `refreshFeedWithRetry` — 指数退避 1s→2s→4s |
| 4 | `rss_service_test.go` | `refreshFeedWithRetry` — 404 → `MarkDead()` 被调用 |
| 5 | `rss_service_test.go` | `refreshFeedWithRetry` — 3 次全失败返回最后 error |
| 6 | `rss_service_test.go` | `isHTTPNotFound` — 各种 404/410 错误字符串 |

### Integration Tests

| # | File | What to Test |
|---|------|-------------|
| 7 | `article_repo_test.go` | `SetStatus` → `GetByStatus` 往返 |

---

## File Changes Summary

| File | Action | Notes |
|------|--------|-------|
| `internal/models/models.go` | Modify | 新增 `Status string` 字段 |
| `internal/service/rss_service.go` | Modify | 并发 fetch + retry + `isHTTPNotFound` + `refreshFeedWithRetry` |
| `internal/repository/sqlite/feed_repository.go` | Modify | `MarkDead`, `GetDeadFeeds` |
| `internal/repository/sqlite/article_repository.go` | Modify | `SetStatus`, `GetByStatus` |
| `app.go` | Modify | 5 个新绑定方法 |
| `frontend/src/components/ArticleList.tsx` | Modify | accept/reject/snooze + dead feeds banner |
| `frontend/src/i18n/en.ts` | Modify | 新文案 |
| `frontend/src/i18n/zh.ts` | Modify | 新文案 |
| `internal/service/rss_service_test.go` | Modify | 6 个新测试 |
| `internal/repository/sqlite/article_repo_test.go` | **New** | 集成测试 |

**Total: 10 files (1 new, 9 modified)**

---

## Open Questions

1. **稍后读超时时间？** 建议 24h 后自动变回"未读"（可在 Settings 里配置）

---

## Open Questions (deferred to Plan B)

1. embedding 模型：`nomic-embed-text` vs `bge-m3`？
2. 质量评分权重是否可配置？
3. embedding cache 策略？

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | issues_open | 8 issues, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

**VERDICT:** ENG REVIEW REQUIRED — issues found but resolved, implementation can proceed
