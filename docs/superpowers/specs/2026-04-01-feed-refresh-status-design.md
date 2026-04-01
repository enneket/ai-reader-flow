# 订阅源刷新状态精细化展示

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 刷新订阅源后，在侧边栏列表显示每个 feed 的刷新结果：失败标记、新文章数

**Architecture:**
- 后端：`fetchArticles` 返回新文章数，`RefreshAllFeedsWithProgress` callback 扩展 per-feed 结果
- 前端：Feed 列表项显示状态图标 + 新文章数标签
- API：新增 per-feed 结果存储，`/api/refresh/status` 返回明细

---

## 1. 数据模型变更

### Feed 模型新增字段
```go
// internal/models/models.go
LastRefreshSuccess int    // 最后刷新新文章数，-1 表示失败
LastRefreshError   string // 失败时的错误信息
LastRefreshed      time.Time
```

### 数据库迁移
SQLite 新增列：
```sql
ALTER TABLE feeds ADD COLUMN last_refresh_success INTEGER DEFAULT 0;
ALTER TABLE feeds ADD COLUMN last_refresh_error TEXT DEFAULT '';
ALTER TABLE feeds ADD COLUMN last_refreshed DATETIME;
```

---

## 2. 后端实现

### 2.1 fetchArticles 返回新文章数
```go
// internal/service/rss_service.go
// 返回新抓取的 article 数量
func (s *RSSService) fetchArticles(feed *models.Feed) (int, error)
```

### 2.2 RefreshAllFeedsWithProgress 扩展 callback
```go
// 新 callback 签名
onProgress(idx, total int, feedTitle string, feedId int64, success bool, newCount int, errMsg string)
```

### 2.3 FeedRepository 更新方法
```go
UpdateRefreshResult(id int64, success int, errorMsg string) error
```

### 2.4 API 端点
`GET /api/refresh/status` 响应：
```json
{
  "inProgress": false,
  "results": [
    {"feedId": 1, "title": "量子位", "success": true, "newCount": 5},
    {"feedId": 2, "title": "机器之心", "success": false, "error": "connection refused"}
  ]
}
```

---

## 3. 前端实现

### 3.1 Feed 列表项显示逻辑
```
✅ 标题 (+N)   ← 成功，新文章数 N>0
✅ 标题        ← 成功，0 新文章（不显示标签）
❌ 标题        ← 失败，hover 显示错误信息
```

### 3.2 组件接口
```tsx
// FeedItem 新增 props
interface FeedItemProps {
  lastRefreshSuccess: number  // -1=失败, 0=成功无新文章, >0=成功有新文章
  lastRefreshError: string    // 失败时的错误信息
}
```

### 3.3 刷新完成后更新本地 feed 状态
响应 `/api/refresh/status` 的 `results` 数组，更新对应 feed 的显示状态

---

## 4. 刷新完成弹框内容

不再显示 `成功刷新 X 个订阅源`，改为：
- 显示失败 feed 列表（如有）
- 显示成功 feed 的新文章数明细
- 0 新文章的 feed 不列出

---

## 5. 文件清单

| 文件 | 改动 |
|------|------|
| `internal/models/models.go` | Feed 新增 3 字段 |
| `internal/repository/sqlite/feed.go` | 新增 `UpdateRefreshResult` |
| `internal/service/rss_service.go` | `fetchArticles` 返回 count，callback 扩展 |
| `cmd/server/main.go` | 存储 per-feed 结果，API 响应调整 |
| `frontend/src/api.ts` | 适配新 API 响应 |
| `frontend/src/components/FeedList.tsx` | FeedItem 显示状态图标/标签 |

---

## 6. 测试验证

- [ ] 刷新 10 个 feed，部分失败，验证失败标记显示
- [ ] 刷新后验证成功 feed 显示 `+N` 标签
- [ ] 刷新 0 新文章的 feed，验证无标签
- [ ] 刷新完成后弹框内容正确
