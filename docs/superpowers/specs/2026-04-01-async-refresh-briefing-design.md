# 异步刷新订阅与生成简报设计

## 背景

当前 `handleGenerateBriefing` 是同步阻塞的：先刷新所有订阅源（很慢，TLS证书问题导致超时），再生成简报，全程不返回直到完成。用户看到的是 504 超时，体验很差。

## 目标

改为异步模式：
- 后端立即返回，前端显示进度
- 刷新和生成通过 SSE 推送实时进度
- 操作互斥：刷新进行中不能生成简报，反之亦然

## 技术方案

### 1. 复用现有 SSE 通道

扩展 `internal/events/events.go` 的事件类型，复用现有 `/api/events` 通道推送进度。

### 2. 事件类型

```go
// Refresh events
EventRefreshStart     = "refresh:start"      // {total: int}
EventRefreshProgress   = "refresh:progress"    // {current: int, total: int, feedTitle: string}
EventRefreshComplete  = "refresh:complete"   // {success: int, failed: int}
EventRefreshError     = "refresh:error"       // {message: string}

// Briefing events
EventBriefingStart     = "briefing:start"     // {}
EventBriefingProgress = "briefing:progress"  // {stage: string, detail: string}
EventBriefingComplete = "briefing:complete"  // {briefingId: int}
EventBriefingError     = "briefing:error"     // {message: string}
```

### 3. API 改动

#### POST /api/refresh
- 立即返回 202 Accepted + {taskId: string}
- goroutine 执行刷新，通过 SSE 推送进度
- 操作互斥：刷新进行中时返回 409 Conflict

#### POST /api/briefings/generate
- 立即返回 202 Accepted + {taskId: string}
- goroutine 执行生成，通过 SSE 推送进度
- 操作互斥：生成进行中时返回 409 Conflict

### 4. 操作互斥

全局状态：
```go
type OperationState struct {
    mutex   sync.Mutex
    current string  // "idle" | "refreshing" | "generating"
}
var operationState = &OperationState{}
```

检查逻辑：
```go
if !operationState.tryLock("refreshing") {
    writeJSON(w, 409, {"success": false, "error": "正在刷新订阅源，请稍候", "code": "OPERATION_IN_PROGRESS"})
    return
}
defer operationState.unlock()
```

### 5. RSSService 进度回调

```go
func (s *RSSService) RefreshAllFeeds(onProgress func(current, total int, feedTitle string)) error {
    feeds, _ := s.feedRepo.GetAllFeeds()
    for i, feed := range feeds {
        if err := s.fetchFeed(feed); err != nil {
            // log but continue
        }
        onProgress(i+1, len(feeds), feed.Title)
    }
}
```

### 6. BriefingService 进度回调

```go
func (s *BriefingService) GenerateBriefing(onProgress func(stage, detail string)) (*models.Briefing, error) {
    onProgress("fetching", "正在获取文章...")
    articles, _ := s.articleRepo.GetArticlesAfter(s.LastRefreshAt)

    onProgress("analyzing", "正在分析文章主题...")
    // AI 调用

    onProgress("generating", "正在生成简报...")
    // 存储 items

    return briefing, nil
}
```

### 7. 前端改动

- 复用现有 SSE EventSource（`/api/events`）
- 监听新事件类型，解析并更新 UI 状态
- 进度条显示：刷新显示 "正在刷新 5/77 个订阅源: Hacker News"，生成显示 "正在生成简报: 分析文章主题"
- 操作进行中时按钮 disabled
- 完成后自动刷新列表

## 数据流

```
用户点击 → POST /api/refresh → 202 Accepted
         → SSE 推送 refresh:start
         → SSE 推送 refresh:progress (多次)
         → SSE 推送 refresh:complete/error
```

```
用户点击 → POST /api/briefings/generate → 202 Accepted
         → SSE 推送 briefing:start
         → SSE 推送 briefing:progress (多次)
         → SSE 推送 briefing:complete/error
```

## 验收标准

- [ ] POST /api/refresh 立即返回 202，不阻塞
- [ ] POST /api/briefings/generate 立即返回 202，不阻塞
- [ ] 刷新进行中时，生成简报返回 409
- [ ] 生成进行中时，刷新订阅返回 409
- [ ] SSE 推送 refresh:progress 包含当前进度 (current/total)
- [ ] SSE 推送 briefing:progress 包含当前阶段 (stage/detail)
- [ ] 前端显示实时进度条
- [ ] 操作完成后自动刷新列表
