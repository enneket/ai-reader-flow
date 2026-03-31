# 简报生成逻辑修改设计

## 背景

当前简报生成使用所有未读文章，用户希望改为：每轮 RSS 刷新后生成简报，只使用本轮刷新获取的新文章。

## 目标

1. 用户点击"立即生成简报"时，先刷新 RSS，再用本轮新文章生成简报
2. 每轮刷新只生成一次简报，防止重复
3. 手动触发刷新和生成简报一体化

## 设计方案

### 数据模型

在配置或全局状态中记录两个时间戳：

```go
var (
    lastRefreshAt    time.Time // 最后刷新时间
    lastBriefingAt   time.Time // 最后生成简报时间
)
```

### 流程

1. 用户点击"立即生成简报"
2. 检查 `lastBriefingAt >= lastRefreshAt`：
   - 是 → 返回错误"本轮已生成简报，请稍后再试"
   - 否 → 继续
3. 记录 `lastRefreshAt = time.Now()`
4. 执行 RSS 刷新，获取新文章
5. 用本轮新文章（`created_at > lastRefreshAt`）生成简报
6. 记录 `lastBriefingAt = time.Now()`

### 数据层修改

**ArticleRepository 新增方法：**

```go
// GetArticlesAfter returns articles created after the given time
func (r *ArticleRepository) GetArticlesAfter(startTime time.Time) ([]models.Article, error)
```

SQL:
```sql
SELECT ... FROM articles
WHERE created_at > ?
ORDER BY created_at DESC
LIMIT 100
```

### 服务层修改

**BriefingService.GenerateBriefing() 修改：**

```go
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
    // 1. 检查本轮是否已生成
    if !lastBriefingAt.Before(lastRefreshAt) {
        return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
    }

    // 2. 记录刷新开始时间
    lastRefreshAt = time.Now()

    // 3. 刷新 RSS
    if err := rssService.RefreshAllFeeds(); err != nil {
        return nil, fmt.Errorf("刷新订阅源: %w", err)
    }

    // 4. 获取本轮新文章
    articles, err := s.articleRepo.GetArticlesAfter(lastRefreshAt)
    if err != nil {
        return nil, fmt.Errorf("获取新文章: %w", err)
    }

    // 5. 生成简报...
}
```

### API 层修改

`handleGenerateBriefing` 改为同步执行（不再后台 goroutine），因为需要返回错误给前端。

### 前端修改

- 生成成功后显示"简报生成成功"
- 如果返回"本轮已生成"错误，显示提示"本轮已生成简报，请稍后再试"

## 实现步骤

1. 修改 `ArticleRepository` 添加 `GetArticlesAfter` 方法
2. 在 `BriefingService` 添加刷新前检查和时间记录
3. 修改 `GenerateBriefing` 流程：刷新 → 获取新文章 → 生成
4. 修改 API handler 同步执行
5. 前端显示相应提示

## 错误处理

- 刷新失败 → 返回错误，本轮仍可重试
- 无新文章 → 返回"暂无新文章"
- 本轮已生成 → 返回"本轮已生成简报"
- AI 生成失败 → 记录失败状态，可重试
