# Plan B: Embedding + Semantic Deduplication + Quality Scoring

## Goal

Phase 2 核心功能：AI 语义去重 + 质量评分，过滤低质量内容，减少重复阅读。

验收标准：100 篇文章 embedding + 评分 < 2 分钟，重复内容减少 > 80%。

---

## Scope

### In Scope
- [x] `AIServiceProvider.GetEmbedding()` — Ollama / Claude / OpenAI 三种 provider 都实现
- [x] Article 表新增 `embedding` 字段（SQLite 存为 JSON 数组 or BLOB）
- [x] 语义去重：cosine similarity > 0.85 → 标记重复，保留高分文章
- [x] 精确去重：URL 完全相同 → 直接跳过
- [x] 质量评分公式：`score = 标题分(0-25) + 长度分(0-15)`，阈值 30（可配置）
- [x] `FilterAllArticles` 并行化（goroutine pool，10 并发）
- [x] `FilterArticle` 错误正确返回（不再静默吞掉）
- [x] Ollama HTTP timeout（30s explicit）
- [x] 单元测试覆盖

### NOT In Scope (→ Phase 2 后续)
- AI 偏好学习（基于用户 accept/reject 行为调整权重）
- 多语言翻译
- Batch embedding 优化（> 100 篇的批处理）

---

## Architecture — Data Flow

```
RSS Refresh Cycle
================

Phase 1: RSS Fetch（fetchArticles 并行，goroutine pool 5）
  │
  └── 每批 feed fetch 完成 → articles 入库
      ├── URL exact dedup（LinkExists）→ 已存在则 skip
      └── 新文章入库，embedding = nil, quality_score = 0

Phase 2: FilterAllArticles（`RefreshAllFeeds` 完成后自动调用）
  │
  ├── 获取所有 unread 且无 embedding 的文章
  │      （同一批 fetches 产生的新文章）
  │
  ├── 语义去重（仅在这批新文章内部互相比较）
  │      ├── 各自计算 embedding（并行，goroutine pool 10）
  │      ├── 互相算 cosine similarity
  │      └── similarity > 0.85 → 保留高分，低分标记 is_filtered=true
  │
  └── 质量评分（与 embedding 计算并行）
         score = 标题分(0-25) + 长度分(0-15)
         │
         ├── score ≥ 30 ──→ is_filtered = false
         └── score < 30 ──→ is_filtered = true
```

**关键约束：语义去重仅在同一批新文章内部，不查历史全量 embedding（内存可控）**

### Data Model Changes

**`internal/models/models.go`** — Article 新增字段：
```go
type Article struct {
    // ... existing fields ...
    Embedding     []float32 `json:"embedding,omitempty"` // nullable，JSON 存储
    QualityScore  int       `json:"quality_score"`       // 0-75，0 = 未评分
}
```

**`internal/repository/sqlite/article_repository.go`**:
```go
// GetUnreadWithoutEmbedding returns unread articles that haven't been embedded yet
func (r *ArticleRepository) GetUnreadWithoutEmbedding() ([]models.Article, error)

// SaveEmbedding saves the embedding vector for an article
func (r *ArticleRepository) SaveEmbedding(articleID int64, embedding []float32) error

// UpdateQualityScore updates the quality score for an article
func (r *ArticleRepository) UpdateQualityScore(id int64, score int) error

// GetByIDs returns articles by their IDs (for batch dedup lookup)
func (r *ArticleRepository) GetByIDs(ids []int64) ([]models.Article, error)
```

### New AI Provider Interface

**`internal/ai/provider.go`**:
```go
type AIServiceProvider interface {
    GenerateSummary(content string) (string, error)
    FilterArticle(content string, rules []string) (bool, error)
    GetEmbedding(text string) ([]float32, error)  // NEW
}

// Cosine similarity
func CosineSimilarity(a, b []float32) float64 {
    // ...
}

// SemanticDedup returns articles that are semantically duplicate of newArticle
// Threshold: 0.85 similarity
func SemanticDedup(newEmbedding []float32, existing map[int64][]float32, threshold float64) (duplicateIDs []int64, bestMatchID int64)
```

### Quality Scoring Formula

```go
// QualityScore returns 0-40 based on title clarity + content length.
// Source credibility removed: hardcoded whitelist has near-zero coverage on RSS blogs.
// Threshold: 30/40 (75% pass-through — articles below are auto-hidden)
func QualityScore(article *models.Article) int {
    // Title clarity: 0-25
    // Long titles (10-100 chars) with caps/numbers = high score
    // Too short (<5) or too long (>150) = 0
    titleScore := scoreTitle(article.Title)

    // Content length: 0-15
    // > 2000 chars: 15, 1000-2000: 10, 500-1000: 5, < 500: 0
    lengthScore := scoreLength(article.Content)

    return titleScore + lengthScore  // max 40
}
```

### FilterAllArticles Full Pipeline

```go
func (s *FilterService) FilterAllArticles() error {
    // Step 1: 获取未评分的文章（同一批新文章）
    newArticles, err := s.articleRepo.GetUnreadWithoutEmbedding()
    if err != nil {
        return err
    }
    if len(newArticles) == 0 {
        return nil
    }

    // Step 2: 并行计算 embedding（goroutine pool 10）
    embeddings := make(map[int64][]float32)
    var mu sync.Mutex
    sem := make(chan struct{}, 10)
    var wg sync.WaitGroup
    var errs []error

    for i := range newArticles {
        wg.Add(1)
        go func(a *models.Article) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            emb, err := s.provider.GetEmbedding(a.Title + "\n\n" + a.Summary)
            if err != nil {
                mu.Lock()
                errs = append(errs, fmt.Errorf("embedding article %d: %w", a.ID, err))
                mu.Unlock()
                return
            }
            s.articleRepo.SaveEmbedding(a.ID, emb)
            mu.Lock()
            embeddings[a.ID] = emb
            mu.Unlock()
        }(&newArticles[i])
    }
    wg.Wait()
    if len(errs) > 0 {
        return errors.Join(errs...) // embedding 失败则整体失败
    }

    // Step 3: 同一批内语义去重（O(n²)，仅限本批文章）
    // 互相 cosine similarity > 0.85 → 保留 quality_score 更高的，低分者标记 filtered=true
    toFilter := s.semanticDedupBatch(newArticles, embeddings)

    // Step 4: 剩余文章并行质量评分 + AI 过滤
    sem = make(chan struct{}, 10)
    var mu2 sync.Mutex
    errs = nil

    for _, a := range newArticles {
        if toFilter[a.ID] {
            s.articleRepo.SetFiltered(a.ID, true)
            continue
        }
        wg.Add(1)
        go func(a models.Article) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            score := s.QualityScore(&a)
            s.articleRepo.UpdateQualityScore(a.ID, score)

            shouldShow := score >= s.qualityThreshold
            if !shouldShow {
                s.articleRepo.SetFiltered(a.ID, true)
            }
        }(a)
    }
    wg.Wait()
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}
```

---

## File Changes Summary

| File | Action | Notes |
|------|--------|-------|
| `internal/models/models.go` | Modify | 新增 `Embedding []float32`, `QualityScore int` 字段 |
| `internal/repository/sqlite/article_repository.go` | Modify | 新增 3 个方法：`GetUnreadWithoutEmbedding`, `SaveEmbedding`, `UpdateQualityScore` |
| `internal/repository/sqlite/db.go` | Modify | 迁移：新增 `embedding` TEXT 列、`quality_score` INT DEFAULT 0 |
| `internal/ai/provider.go` | Modify | 新增 `GetEmbedding()` Ollama/Claude/OpenAI 三种实现 + `CosineSimilarity()` |
| `internal/service/filter_service.go` | Modify | 重写 `FilterAllArticles`（embedding→去重→评分完整 pipeline）+ `QualityScore()` + `semanticDedupBatch()` |
| `internal/service/filter_service_test.go` | Modify | 新增 8 个测试（embedding/score/dedup/error handling）|
| `app.go` | Modify | `RefreshAllFeeds` 完成后自动调用 `FilterService.FilterAllArticles` |

**Total: 7 files (0 new, 7 modified)**

---

## Tests to Write

| # | File | What to Test | Quality |
|---|------|-------------|---------|
| 1 | `filter_service_test.go` | `CosineSimilarity` — 相同向量=1.0, 正交=0, 相反≈-1 | ★★★ |
| 2 | `filter_service_test.go` | `CosineSimilarity` — nil/empty → returns 0 | ★★★ |
| 3 | `filter_service_test.go` | `QualityScore` — content length boundaries: <500→0, 500→5, 1000→10, 2000→15 | ★★★ |
| 4 | `filter_service_test.go` | `QualityScore` — title boundaries: <5→0, 5→5, 10→20, 100→25, >150→0 | ★★★ |
| 5 | `filter_service_test.go` | `QualityScore` — title with numbers/caps → +5 bonus | ★★ |
| 6 | `filter_service_test.go` | `semanticDedupBatch` — 2篇相似>0.85 → 低分标记 filtered=true | ★★★ |
| 7 | `filter_service_test.go` | `semanticDedupBatch` — 3篇互不相像 → 都不标记 filtered | ★★★ |
| 8 | `filter_service_test.go` | `semanticDedupBatch` — 3篇中2篇相似 → 仅那对中低分者标记 | ★★ |
| 9 | `filter_service_test.go` | `FilterAllArticles` — embedding 失败任意一篇 → errors.Join 聚合所有错误 | ★★★ |
| 10 | `filter_service_test.go` | `FilterAllArticles` — 全部已有 embedding → 跳过 embedding 步骤 | ★★ |
| 11 | `filter_service_test.go` | `FilterAllArticles` — 零 article → 立即返回 nil | ★ |
| 12 | `article_repository_test.go` | `GetUnreadWithoutEmbedding` — SQL `IS NULL` 正确（不是 `= NULL`）| ★★★ |
| 13 | `article_repository_test.go` | `SaveEmbedding` → JSON encode/decode round-trip | ★★ |
| 14 | `article_repository_test.go` | `UpdateQualityScore` → 更新已存在的 score | ★ |

---

## NOT In Scope

- AI 偏好学习（基于用户 accept/reject 行为调整权重）— Phase 2
- 多语言翻译管道 — Phase 2
- Embedding 批处理优化（>100 篇/批次）— Phase 2
- 用户可配置 source credibility 白名单 — 移除（hardcoded 白名单覆盖率低）
- 前端 quality score badge 显示 — 低优先级，Phase 2 或后续

---

## What Already Exists

| 已有代码 | 复用于 Plan B |
|---------|--------------|
| `RSSService.RefreshAllFeeds` 的 semaphore 并发模式 | 直接复制到 `FilterAllArticles`（sem chan 10） |
| `errors.Join` 错误聚合 | `FilterAllArticles` 错误聚合直接使用 |
| `LinkExists` URL exact dedup | Plan B 去重逻辑复用此检查 |
| `gofeed.Parser` | 继续使用，无改动 |
| AI provider 接口 | `GetEmbedding()` 作为新方法加入同一 interface |

---

## Open Questions

1. **Embedding 模型选择**：Ollama `nomi-embed-text` (768维)，OpenAI `text-embedding-3-small` (1536维)，Claude 用内嵌模型 — CosineSimilarity 对不同维度向量仍然有效（先归一化）
2. **Embedding 存储格式**：SQLite TEXT 存 JSON `[]float32` — 方便调试，Go `encoding/json` 直接 marshal/unmarshal

---

## Failure Modes

| Failure | Impact | Mitigation |
|---------|---------|------------|
| Ollama `/api/embed` endpoint unavailable (v < 0.1.15) | FilterAllArticles fails | 检查 Ollama 版本，提示用户升级或降级到 `/api/generate` fallback |
| Embedding 计算部分失败（2/10 文章失败） | `errors.Join` 返回聚合错误，整体失败 | 设计上如此：任何 embedding 失败则整体重试，用户需重新触发 |
| SQLite TEXT 存储 JSON 数组超长（文章 > 50000 chars embedding） | 写入失败 | 截断 content/summary 再 embedding（max 50000 chars） |
| 用户无 AI 配置（未设置 Ollama/API） | FilterAllArticles 调用失败 | 自动降级：跳过 AI 过滤，只做 URL exact dedup + quality_score |
| Ollama 服务 hang（进程 alive 但不响应） | embedding 请求无限等待 | `http.Client{Timeout: 30s}` — Plan B 已包含 |

## Plan B 优先级（建议顺序）

1. **Step 1**: `GetEmbedding()` 三种 provider 实现 + `CosineSimilarity()` + 单元测试（Test #1-2）
2. **Step 2**: Article model + repository 新字段 + 迁移（`embedding` TEXT, `quality_score` INT）
3. **Step 3**: `QualityScore()` 公式 + 单元测试（Test #3-5）
4. **Step 4**: `semanticDedupBatch()` + 单元测试（Test #6-8）
5. **Step 5**: `FilterAllArticles` 完整 pipeline + 错误聚合 + 单元测试（Test #9-11）
6. **Step 6**: Ollama HTTP timeout（`http.Client{Timeout: 30s}`）
7. **Step 7**: 前端 quality score badge 显示（可选，低优先级）

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | CLEAR | 5 decisions made, 14 tests planned |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

**VERDICT:** ENG REVIEW CLEARED — 5 architecture decisions made (embedding timing, dedup scope, source credibility removed, auto-trigger, error aggregation), plan ready to implement
