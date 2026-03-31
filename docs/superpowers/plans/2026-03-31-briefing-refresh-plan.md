# 简报刷新逻辑实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修改简报生成逻辑：点击"立即生成简报"时先刷新RSS，只使用本轮新文章生成简报，每轮刷新只生成一次

**Architecture:** 在 BriefingService 添加时间戳记录和刷新前检查，修改 ArticleRepository 添加按时间查询新文章的方法

**Tech Stack:** Go, SQLite

---

## 文件结构

- 修改: `internal/repository/sqlite/article_repository.go` - 添加 GetArticlesAfter 方法
- 修改: `internal/service/briefing_service.go` - 添加时间戳和刷新前检查
- 修改: `cmd/server/main.go` - 添加全局时间戳变量，修改 handleGenerateBriefing
- 修改: `frontend/src/components/Briefing.tsx` - 显示相应提示
- 修改: `frontend/src/api.ts` - 处理错误响应

---

## Task 1: 添加 GetArticlesAfter 方法

**Files:**
- Modify: `internal/repository/sqlite/article_repository.go`

- [ ] **Step 1: 找到 GetRecentForBriefing 方法位置，在其后添加新方法**

在 article_repository.go 文件中约第 197 行，GetRecentForBriefing 方法之后添加：

```go
// GetArticlesAfter returns articles created after the given time
func (r *ArticleRepository) GetArticlesAfter(startTime time.Time) ([]models.Article, error) {
	rows, err := DB.Query(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, embedding, COALESCE(quality_score, 0)
         FROM articles
         WHERE created_at > ?
         ORDER BY created_at DESC
         LIMIT 100`,
		startTime,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticles(rows)
}
```

- [ ] **Step 2: 确保导入了 time 包**

检查文件顶部 import 是否有 `"time"`，如果没有则添加。

- [ ] **Step 3: 编译验证**

```bash
go build ./internal/repository/sqlite/...
```

---

## Task 2: 修改 BriefingService

**Files:**
- Modify: `internal/service/briefing_service.go`

- [ ] **Step 1: 添加导出的全局时间戳变量**

在 briefing_service.go 文件开头，type BriefingService struct 之后添加：

```go
var (
	LastRefreshAt  time.Time // 最后刷新时间（导出供 main.go 使用）
	LastBriefingAt time.Time // 最后生成简报时间
)
```

注意：使用首字母大写使其可被 main.go 访问。

- [ ] **Step 2: 修改 GenerateBriefing 方法**

将原来的：

```go
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
	// 1. Create briefing record
	briefing := &models.Briefing{
		Status: "generating",
	}
	if err := s.briefingRepo.Create(briefing); err != nil {
		return nil, fmt.Errorf("create briefing: %w", err)
	}

	// 2. Get recent articles
	articles, err := s.articleRepo.GetRecentForBriefing()
```

替换为：

```go
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
	// 0. 检查本轮是否已生成
	if !lastBriefingAt.Before(lastRefreshAt) && !lastRefreshAt.IsZero() {
		return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
	}

	// 1. 记录刷新开始时间
	LastRefreshAt = time.Now()

	// 2. 刷新 RSS（通过事件触发或直接调用）
	// 注意：这里需要触发刷新，但当前架构中刷新是在 handler 层调用的
	// 我们暂时跳过这步，假设刷新已在外层完成

	// 3. 获取本轮新文章（created_at > lastRefreshAt）
	articles, err := s.articleRepo.GetArticlesAfter(lastRefreshAt)
	if err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
		return nil, fmt.Errorf("get articles: %w", err)
	}

	// 4. 检查是否有新文章
	if len(articles) == 0 {
		return nil, fmt.Errorf("暂无新文章")
	}

	// 5. Create briefing record
	briefing := &models.Briefing{
		Status: "generating",
	}
	if err := s.briefingRepo.Create(briefing); err != nil {
		return nil, fmt.Errorf("create briefing: %w", err)
	}
```

注意：这里需要调整顺序，简化逻辑。更好的方式是：

```go
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
	// 0. 检查本轮是否已生成
	if !lastBriefingAt.Before(lastRefreshAt) && !lastRefreshAt.IsZero() {
		return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
	}

	// 1. 记录刷新开始时间
	refreshTime := time.Now()

	// 2. Create briefing record
	briefing := &models.Briefing{
		Status: "generating",
	}
	if err := s.briefingRepo.Create(briefing); err != nil {
		return nil, fmt.Errorf("create briefing: %w", err)
	}

	// 3. 获取本轮新文章
	articles, err := s.articleRepo.GetArticlesAfter(refreshTime)
	if err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
		return nil, fmt.Errorf("get articles: %w", err)
	}

	// 4. 检查是否有新文章
	if len(articles) == 0 {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", "暂无新文章")
		return nil, fmt.Errorf("暂无新文章")
	}
```

- [ ] **Step 3: 找到生成成功后记录时间的位置**

在方法最后，return briefing, nil 之前添加：

```go
	// 7. Mark as completed
	s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")
	LastBriefingAt = time.Now()

	return briefing, nil
```

- [ ] **Step 4: 确保导入了 time 包**

检查 import 是否有 `"time"`，如果没有则添加。

- [ ] **Step 5: 编译验证**

```bash
go build ./internal/service/...
```

---

## Task 3: 修改 main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 添加 RSSService 引用**

在 main.go 全局变量处，已有 `rssService *service.RSSService`，无需修改。

- [ ] **Step 2: 修改 handleGenerateBriefing 函数**

将原来的：

```go
func handleGenerateBriefing(w http.ResponseWriter, r *http.Request) {
	// Create briefing in background
	go func() {
		briefingService.GenerateBriefing()
	}()
	w.WriteHeader(http.StatusAccepted)
}
```

替换为：

```go
func handleGenerateBriefing(w http.ResponseWriter, r *http.Request) {
	// 1. 记录刷新时间
	service.LastRefreshAt = time.Now()

	// 2. 刷新所有订阅源
	if err := rssService.RefreshAllFeeds(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. 生成简报
	briefing, err := briefingService.GenerateBriefing()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 4. 返回成功
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"id":      briefing.ID,
	})
}
```

注意：需要确保 service 包中的 lastRefreshAt 是导出的（首字母大写），或者通过方法设置。

更简洁的方案是在 service 包添加一个方法来统一处理：

```go
// 在 briefing_service.go 添加
func (s *BriefingService) GenerateBriefingWithRefresh() (*models.Briefing, error) {
	// 检查本轮是否已生成
	if !lastBriefingAt.Before(lastRefreshAt) && !lastRefreshAt.IsZero() {
		return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
	}

	// 记录刷新开始时间
	LastRefreshAt = time.Now()

	// 刷新 RSS
	if err := rssService.RefreshAllFeeds(); err != nil {
		return nil, fmt.Errorf("刷新订阅源: %w", err)
	}

	// ... 后续逻辑
}
```

但这会引入循环依赖（briefing_service 引用 rss_service，rss_service 可能也引用其他服务）。

保持当前设计在 handler 层调用的方式更简单。

- [ ] **Step 3: 确保导入了 time 包**

检查 import 是否有 `"time"`。

- [ ] **Step 4: 编译验证**

```bash
go build ./cmd/server/...
```

---

## Task 4: 修改前端

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/components/Briefing.tsx`

- [ ] **Step 1: 修改 api.ts 中的 generateBriefing 响应类型**

将原来的：

```typescript
generateBriefing: () => request<void>('/briefings/generate', { method: 'POST' }),
```

替换为：

```typescript
generateBriefing: () => request<{success: boolean; id?: number; error?: string}>('/briefings/generate', { method: 'POST' }),
```

- [ ] **Step 2: 修改 Briefing.tsx 中的 handleGenerate 函数**

将原来的：

```typescript
const handleGenerate = async () => {
    setGenerating(true)
    try {
      await api.generateBriefing()
      await loadBriefings(0)
    } catch (err) {
      console.error('Failed to generate briefing:', err)
    } finally {
      setGenerating(false)
    }
  }
```

替换为：

```typescript
const handleGenerate = async () => {
    setGenerating(true)
    try {
      const result = await api.generateBriefing()
      if (result.success) {
        await loadBriefings(0)
      } else {
        alert(result.error || '生成失败')
      }
    } catch (err) {
      console.error('Failed to generate briefing:', err)
    } finally {
      setGenerating(false)
    }
  }
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npm run build
```

---

## Task 5: 测试

- [ ] **Step 1: 启动服务器测试**

```bash
./server &
```

- [ ] **Step 2: 测试流程**

1. 访问简报页面
2. 点击"立即生成简报"
3. 观察是否正确刷新并生成简报
4. 再次点击应提示"本轮已生成简报"

- [ ] **Step 3: 测试无新文章情况**

停止服务器，手动删除一些文章后重启，再次点击生成简报应提示"暂无新文章"

---

## Task 6: 提交

- [ ] **Step 1: 提交代码**

```bash
git add -A
git commit -m "feat: 简报生成改为每轮刷新后生成，只使用新文章

- 添加 GetArticlesAfter 方法按时间查询文章
- 添加本轮检查防止重复生成
- 修改 handleGenerateBriefing 同步执行刷新和生成
- 前端处理错误提示

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
