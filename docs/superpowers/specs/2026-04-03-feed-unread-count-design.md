# 订阅源未读数 badge 设计

## 1. 概述

订阅源列表中每个源右侧的 `+N` badge 表示该订阅源有多少篇未读文章。用户打开文章详情后，badge 计数应自动减 1，直到归零后 badge 消失。

## 2. 架构

**数据流**：article status 是唯一数据源，`unread_count` 是 article status 的聚合缓存。

```
article.status = "unread"  →  feed.unread_count++
article.status = "accepted/rejected/snoozed"  →  feed.unread_count--
```

## 3. 数据层

### 3.1 Feed 模型变更

**文件**: `internal/models/models.go`

```go
// Feed struct 新增字段
UnreadCount   int       `json:"unread_count"`
```

### 3.2 数据库迁移

**文件**: `internal/repository/sqlite/db.go`

```sql
ALTER TABLE feeds ADD COLUMN unread_count INTEGER NOT NULL DEFAULT 0;
```

### 3.3 Feed Repository

**文件**: `internal/repository/sqlite/feed_repository.go`

**更新 `GetAll()` SELECT**：加入 `unread_count` 字段。

**新增方法**：
```go
// UpdateUnreadCount 更新指定订阅源的未读数
func (r *FeedRepository) UpdateUnreadCount(feedId int64, delta int) error

// RecalcUnreadCount 重新计算并更新指定订阅源的未读数（从 article 表 count）
func (r *FeedRepository) RecalcUnreadCount(feedId int64) error
```

### 3.4 Article Repository 状态变更时同步更新

**文件**: `internal/repository/sqlite/article_repository.go`

`SetStatus(id int64, status string)` 方法中，当 status 从 `"unread"` 变为非 `"unread"` 时，调用 `feedRepo.UpdateUnreadCount(feedId, -1)`。

`Create(article *models.Article)` 时，若 `status = "unread"`，调用 `feedRepo.UpdateUnreadCount(feedId, 1)`。

## 4. API 层

无需新增 endpoint。现有 `GET /api/feeds` 返回的 Feed 列表已包含 `unread_count`（由 `GetAll()` 填充）。

## 5. 前端层

### 5.1 Feed 模型

**文件**: `frontend/src/api.ts`

```ts
export interface Feed {
  // ...existing fields
  unread_count: number  // 新增
}
```

### 5.2 Badge 显示逻辑

**文件**: `frontend/src/components/FeedList.tsx`

```tsx
{feed.unread_count > 0 && (
  <span className="status-new">+{feed.unread_count}</span>
)}
// 移除 last_refresh_success 的 badge 显示逻辑
```

### 5.3 点击文章时标记已读

**文件**: `frontend/src/components/FeedList.tsx`

`handleArticleClick` 中，选中文章后调用 `api.acceptArticle(article.id)` 标记已读，前端乐观更新 `setFeeds(prev => prev.map(f => f.id === article.feed_id ? {...f, unread_count: Math.max(0, f.unread_count - 1)} : f))`。

> 注意：仅在文章当前 status 为 `"unread"` 时才递减，避免重复递减。

## 6. 刷新订阅源时的处理

**文件**: `internal/service/rss_service.go` / `cmd/server/main.go`

`RefreshAllFeedsWithProgress` 回调中，新增 article 初始 status 为 `"unread"`（已在 `Create` 时由 Repository 层处理），`unread_count` 增量已由 Repository 层自动维护，无需额外逻辑。

## 7. 初始化已有数据

对数据库中已有 feed，第一次读取时调用 `RecalcUnreadCount` 一次性对齐（通过 `GetAll()` 时检测 `unread_count = 0` 且 `last_refresh_success > 0` 的行进行补算）。

## 8. 验收标准

- [ ] 订阅源刷新后，badge 显示新文章数
- [ ] 点击一篇文章，badge 计数 -1
- [ ] badge 为 0 时不显示任何标记
- [ ] 刷新页面后计数与后端一致（不丢失）
- [ ] `last_refresh_success` 的 `❌` 失败标记逻辑保持不变
