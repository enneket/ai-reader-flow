# Token 预算感知的多批次简报生成设计

> **目标**：为 BriefingService 增加 token 估算和自动分批能力，确保 AI 输入始终在 context window 内，超限时自动拆成多批次调用再合并结果。

## 1. 背景与问题

**现状问题**：
- `BriefingService.buildArticlesInput` 仅用字符数截断（`len(content) > 2000`），无法感知实际 token 消耗
- AI 输出超限时 JSON 被截断，导致 `parse failed: unexpected end of JSON input`
- 不同 AI provider 的 context window 差异大（Ollama 32k、Volcano 224k），无统一预算控制

**根本原因**：缺乏 token 预算感知机制，输入和输出均无边界控制。

## 2. 设计原则

- **Token 估算用轻量算法**：中文字符 ≈ 2.5 tokens/字，英文 ≈ 4 chars/token，不引入 tiktoken 等外部依赖
- **保守预算**：每次调用使用 context window 的 60%，为系统 prompt、输出和 JSON 结构留足空间
- **破坏性最小**：仅改造 `BriefingService`，不修改 AI Provider 接口
- **向后兼容**：不支持分块的 provider（GetEmbedding、FilterArticle）保持现状

## 3. 架构

```
BuildArticlesInput(tokenBudget)
  → 逐篇加入，累加 token 估算
  → 超过 budget → 开启新 batch
  → 单篇超 budget → 截断该篇 content

GenerateBatchedBriefing()
  → 获取所有新文章
  → 构建 N 个 batch（各自 < tokenBudget）
  → 串行调用 AI（每批次独立生成一组 topics）
  → 按 topic name merge 结果（article ID 去重）
  → 存储 merged 结果
```

## 4. 新增模块

### 4.1 TokenEstimator (`internal/ai/token.go`)

```go
// Estimate estimates token count for a string.
// Uses simple ratio: ~2.5 for CJK, ~4 for ASCII whitespace.
func Estimate(text string) int

// EstimatePrompt estimates total tokens for a briefing prompt:
// system + user prompt template + all articles
func (s *BriefingService) EstimatePrompt(articles []models.Article) int
```

### 4.2 AIConfig 新增字段

**`internal/config/config.go`** 和 **`internal/models/models.go`**：

```go
type AIProviderConfig struct {
    // ... existing fields ...
    ContextWindow int `toml:"context_window"` // context window size in tokens
    OutputReserve int `toml:"output_reserve"` // reserve for output (default 2048)
}
```

**默认值**：
- `ContextWindow`: 32768（通用默认值）
- `OutputReserve`: 2048

**各 provider 参考值**（可配置）：

| Provider | ContextWindow |
|----------|--------------|
| openai (GPT-4o) | 128000 |
| volcano (豆包) | 224000 |
| claude | 200000 |
| ollama (local) | 32768 |

### 4.3 Batch 分块策略

```go
const (
    DefaultContextWindow = 32768
    DefaultOutputReserve = 2048
    DefaultPromptOverhead = 500 // system prompt + JSON template
)

func (s *BriefingService) computeBudget() int {
    contextWindow := s.getContextWindow() // from config
    return contextWindow*6/10 - PromptOverhead - OutputReserve
}
```

**分块算法**：

```
batches = []
currentBatch = []
currentTokens = 0
budget = computeBudget()

for each article:
    articleTokens = Estimate(buildArticleString(article))
    if currentTokens + articleTokens > budget:
        if len(currentBatch) > 0:
            batches = append(batches, currentBatch)
        currentBatch = []
        currentTokens = 0
    if articleTokens > budget:
        // single article exceeds budget → truncate content
        truncated = truncateContent(article, budget - currentTokens)
        currentBatch = append(currentBatch, truncated)
    else:
        currentBatch = append(currentBatch, article)
    currentTokens += articleTokens

if len(currentBatch) > 0:
    batches = append(batches, currentBatch)
```

### 4.4 Merge 策略

```go
func mergeBriefingResults(batches []models.BriefingResult) models.BriefingResult {
    topicMap := make(map[string]*models.BriefingTopic)

    for _, batch := range batches {
        for _, topic := range batch.Topics {
            key := normalizeTopicName(topic.Name) // 去除空格、大小写
            if existing, ok := topicMap[key]; ok {
                // Merge articles: append new article IDs, skip duplicates
                existingArticles := make(map[int64]bool)
                for _, a := range existing.Articles {
                    existingArticles[a.ID] = true
                }
                for _, a := range topic.Articles {
                    if !existingArticles[a.ID] {
                        existing.Articles = append(existing.Articles, a)
                        existingArticles[a.ID] = true
                    }
                }
            } else {
                topicMap[key] = &topic
            }
        }
    }

    // Sort by article count (descending)
    topics := make([]models.BriefingTopic, 0, len(topicMap))
    for _, t := range topicMap {
        topics = append(topics, *t)
    }
    sort.Slice(topics, func(i, j int) bool {
        return len(topics[i].Articles) > len(topics[j].Articles)
    })
    return models.BriefingResult{Topics: topics}
}
```

**Merge 去重规则**：
- Topic name 归一化后匹配（大写转小写、去除空格）
- 同一 article ID 不重复出现在同一 topic 下
- Insight 字段：同一 article ID 保留更长的 insight

### 4.5 Prompt Budget 调整

`buildPrompt` 需要感知当前 batch 的 article 数量，调整输出要求：

```go
func (s *BriefingService) buildPrompt(articlesInput string, totalArticles, batchIndex, totalBatches int) string {
    isSingleBatch := totalBatches == 1
    topicLimit := 5
    if !isSingleBatch {
        topicLimit = 3 // 多批次时每批减少 topic 数，保证总输出可接受
    }
    // ... prompt with adjusted expectations
}
```

## 5. 改动范围

| 文件 | 改动 |
|------|------|
| `internal/ai/token.go` | 新建：TokenEstimator |
| `internal/models/models.go` | AIProviderConfig 新增 ContextWindow、OutputReserve 字段 |
| `internal/config/config.go` | 同上 |
| `internal/repository/sqlite/db.go` | 新增列迁移（已有 migrateAddColumn） |
| `internal/service/briefing_service.go` | 重构：分批逻辑 + merge 逻辑 |

**不受影响**：
- AI Provider 接口不变
- `GenerateSummary`、`FilterArticle`、`GetEmbedding` 不变
- Frontend 无改动

## 6. 错误处理

| 场景 | 处理 |
|------|------|
| 单篇文章 > budget | 截断 content 到 budget 内，仍可处理 |
| 所有批次均失败 | briefing 状态 → failed |
| 部分批次失败 | 已成功的批次正常 merge，失败的记录 error 日志 |
| Merge 后 topics 为空 | 返回 failed + "无有效简报内容" |

## 7. 测试场景

1. **正常场景**：文章数量 < budget，单批次完成
2. **超 budget**：文章数量大，触发多批次，验证 merge 正确
3. **单篇超 budget**：某篇极长，截断后仍可生成
4. **去重验证**：同一 article 出现在多个 batch，merge 后只保留一个
5. **Context window 配置**：不同 provider 配置下，budget 计算正确

## 8. 配置示例

```toml
[ai]
provider = "openai"
api_key = "sk-xxx"
base_url = "https://api.openai.com/v1"
model = "gpt-4o"
max_tokens = 500
context_window = 128000
output_reserve = 2048
```
