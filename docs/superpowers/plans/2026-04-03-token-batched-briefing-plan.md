# Token 预算感知多批次简报生成实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 BriefingService 增加 token 估算和自动分批能力，超限时自动拆成多批次调用再合并结果。

**Architecture:** 新增 TokenEstimator（轻量字符比算法） + 分批分组逻辑 + TopicMerge 合并逻辑。Provider 接口不变，仅改造 BriefingService。

**Tech Stack:** Go 1.21+, SQLite, Ollama/OpenAI/Claude (Volcano)

---

## 文件变更总览

| 文件 | 动作 |
|------|------|
| `internal/ai/token.go` | 新建 |
| `internal/models/models.go` | 修改：AIProviderConfig 新增 2 字段 |
| `internal/config/config.go` | 修改：AIProviderConfig 新增 2 字段 |
| `cmd/server/main.go` | 修改：handleGetAIConfig / handleSaveAIConfig 透传新字段 |
| `internal/service/briefing_service.go` | 重构：分批 + merge |
| `internal/service/briefing_service_test.go` | 新建：测试 |

---

### Task 1: 添加 ContextWindow / OutputReserve 配置字段

**Files:**
- Modify: `internal/models/models.go:214-221` (AIProviderConfig struct)
- Modify: `internal/config/config.go:16-22` (AIProviderConfig struct)
- Modify: `internal/config/config.go:40-45` (LoadConfig default values)
- Modify: `cmd/server/main.go:845-851` (handleGetAIConfig)
- Modify: `cmd/server/main.go:870-876` (handleSaveAIConfig)
- Modify: `cmd/server/main.go:856-861` (handleSaveAIConfig request struct)

- [ ] **Step 1: 修改 models.go** — AIProviderConfig struct 新增：

```go
// internal/models/models.go
type AIProviderConfig struct {
    Provider      string `json:"provider"`
    APIKey        string `json:"api_key"`
    BaseURL       string `json:"base_url"`
    Model         string `json:"model"`
    MaxTokens     int    `json:"max_tokens"`
    ContextWindow int    `json:"context_window"`  // 新增
    OutputReserve int    `json:"output_reserve"`  // 新增
}
```

- [ ] **Step 2: 修改 config.go** — AIProviderConfig struct 新增：

```go
// internal/config/config.go
type AIProviderConfig struct {
    Provider      string `toml:"provider"`
    APIKey        string `toml:"api_key"`
    BaseURL       string `toml:"base_url"`
    Model         string `toml:"model"`
    MaxTokens     int    `toml:"max_tokens"`
    ContextWindow int    `toml:"context_window"`  // 新增
    OutputReserve int    `toml:"output_reserve"` // 新增
}
```

- [ ] **Step 3: 修改 LoadConfig 默认值** — 在 `AIProvider: AIProviderConfig{...}` 块中添加：

```go
AIProvider: AIProviderConfig{
    Provider:      "openai",
    BaseURL:       "https://api.openai.com/v1",
    Model:         "gpt-3.5-turbo",
    MaxTokens:     500,
    ContextWindow: 32768,   // 新增
    OutputReserve: 2048,     // 新增
},
```

- [ ] **Step 4: 修改 handleGetAIConfig** — 透传新字段：

```go
// cmd/server/main.go:845-851
aiConfig := models.AIProviderConfig{
    Provider:      cfg.AIProvider.Provider,
    APIKey:        cfg.AIProvider.APIKey,
    BaseURL:       cfg.AIProvider.BaseURL,
    Model:         cfg.AIProvider.Model,
    MaxTokens:     cfg.AIProvider.MaxTokens,
    ContextWindow: cfg.AIProvider.ContextWindow,  // 新增
    OutputReserve: cfg.AIProvider.OutputReserve,  // 新增
}
```

- [ ] **Step 5: 修改 handleSaveAIConfig 请求结构体** — 添加新字段：

```go
// cmd/server/main.go:856-862
var req struct {
    Provider      string `json:"provider"`
    APIKey        string `json:"api_key"`
    BaseURL       string `json:"base_url"`
    Model         string `json:"model"`
    MaxTokens     int    `json:"max_tokens"`
    ContextWindow int    `json:"context_window"`  // 新增
    OutputReserve int    `json:"output_reserve"`  // 新增
}
```

- [ ] **Step 6: 修改 handleSaveAIConfig 赋值** — 透传新字段：

```go
// cmd/server/main.go:870-876
cfg.AIProvider = config.AIProviderConfig{
    Provider:      req.Provider,
    APIKey:       req.APIKey,
    BaseURL:      req.BaseURL,
    Model:        req.Model,
    MaxTokens:    req.MaxTokens,
    ContextWindow: req.ContextWindow,  // 新增
    OutputReserve: req.OutputReserve,   // 新增
}
```

- [ ] **Step 7: 验证构建**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go build ./...`
Expected: 0 errors (only interface{} → any warnings, ignore)

- [ ] **Step 8: Commit**

```bash
git add internal/models/models.go internal/config/config.go cmd/server/main.go
git commit -m "feat(config): add ContextWindow and OutputReserve to AIProviderConfig

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: 创建 TokenEstimator (`internal/ai/token.go`)

**Files:**
- Create: `internal/ai/token.go`
- Test: `internal/ai/token_test.go`

- [ ] **Step 1: 写测试**

```go
// internal/ai/token_test.go
package ai

import "testing"

func TestEstimate(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        wantMin  int
        wantMax  int
    }{
        {
            name:    "empty string",
            input:   "",
            wantMin: 0,
            wantMax: 0,
        },
        {
            name:    "ASCII text",
            input:   "Hello world this is a test",
            wantMin: 5,  // 5 words ~ 5 tokens
            wantMax: 25, // conservative upper bound
        },
        {
            name:    "CJK text",
            input:   "这是一段测试文本用于测试Token估算",
            wantMin: 10,
            wantMax: 30,
        },
        {
            name:    "mixed content",
            input:   "Hello 你好 World 世界",
            wantMin: 4,
            wantMax: 16,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Estimate(tt.input)
            if got < tt.wantMin || got > tt.wantMax {
                t.Errorf("Estimate(%q) = %d, want between %d and %d", tt.input, got, tt.wantMin, tt.wantMax)
            }
        })
    }
}
```

- [ ] **Step 2: Run test — verify it fails (token.go doesn't exist yet)**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go test ./internal/ai/... -run TestEstimate -v`
Expected: FAIL (undefined: Estimate)

- [ ] **Step 3: 写 TokenEstimator 实现**

```go
// internal/ai/token.go
package ai

import "unicode"

// Estimate estimates token count for a string using a simple ratio model.
// CJK characters are estimated at ~2.5 tokens each.
// ASCII characters (letters, digits, punctuation) are estimated at ~4 per token.
// Whitespace is counted at ~5 chars per token.
// This is a rough approximation suitable for budget planning, not billing accuracy.
func Estimate(text string) int {
    if text == "" {
        return 0
    }

    cjkCount := 0
    asciiCount := 0
    wsCount := 0

    for _, r := range text {
        switch {
        case isCJK(r):
            cjkCount++
        case r == ' ' || r == '\t' || r == '\n' || r == '\r':
            wsCount++
        default:
            asciiCount++
        }
    }

    // CJK: ~2.5 chars per token (each CJK char is ~1 token in typical encodings)
    // ASCII: ~4 chars per token
    // Whitespace: ~5 chars per token (overhead)
    cjkTokens := float64(cjkCount) / 2.5
    asciiTokens := float64(asciiCount) / 4.0
    wsTokens := float64(wsCount) / 5.0

    return int(cjkTokens + asciiTokens + wsTokens)
}

// isCJK returns true if r is a CJK character (Chinese, Japanese, Korean, etc.)
func isCJK(r rune) bool {
    return (r >= 0x4E00 && r <= 0x9FFF) ||  // CJK Unified Ideographs
           (r >= 0x3000 && r <= 0x303F) ||  // CJK Symbols
           (r >= 0xFF00 && r <= 0xFFEF) ||  // Halfwidth/Fullwidth Forms
           (r >= 0x3040 && r <= 0x309F) ||  // Hiragana
           (r >= 0x30A0 && r <= 0x30FF) ||  // Katakana
           (r >= 0xAC00 && r <= 0xD7AF)      // Hangul Syllables
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go test ./internal/ai/... -run TestEstimate -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/token.go internal/ai/token_test.go
git commit -m "feat(ai): add TokenEstimator with CJK/ASCII ratio algorithm

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: BriefingService 增加分批逻辑

**Files:**
- Modify: `internal/service/briefing_service.go`
- Create: `internal/service/briefing_service_test.go`

**Constants (新增到 briefing_service.go 顶部):**

```go
const (
    DefaultContextWindow = 32768
    DefaultOutputReserve = 2048
    DefaultPromptOverhead = 500
)
```

**BriefingService struct 新增字段:**

```go
type BriefingService struct {
    briefingRepo    *sqlite.BriefingRepository
    articleRepo     *sqlite.ArticleRepository
    feedRepo        *sqlite.FeedRepository
    LastRefreshAt   time.Time
    LastBriefingAt  time.Time
    aiConfig        *models.AIProviderConfig  // 新增
}
```

**NewBriefingService 签名变更（接受 aiConfig）:**

```go
func NewBriefingService(aiConfig *models.AIProviderConfig) *BriefingService {
    return &BriefingService{
        briefingRepo: sqlite.NewBriefingRepository(),
        articleRepo:  sqlite.NewArticleRepository(),
        feedRepo:     sqlite.NewFeedRepository(),
        aiConfig:     aiConfig,
    }
}
```

**新增方法 ( BriefingService):**

```go
// computeBudget returns the max tokens available for article content per batch.
// budget = contextWindow * 0.6 - promptOverhead - outputReserve
func (s *BriefingService) computeBudget() int {
    cw := DefaultContextWindow
    or := DefaultOutputReserve
    if s.aiConfig != nil {
        if s.aiConfig.ContextWindow > 0 {
            cw = s.aiConfig.ContextWindow
        }
        if s.aiConfig.OutputReserve > 0 {
            or = s.aiConfig.OutputReserve
        }
    }
    return cw*6/10 - DefaultPromptOverhead - or
}

// buildArticleStringForEstimate builds the full article string for token estimation
// (same format as buildArticlesInput but returns string, not written to sb)
func (s *BriefingService) buildArticleStringForEstimate(a models.Article) string {
    content := a.Content
    if content == "" {
        content = a.Summary
    }
    return fmt.Sprintf("文章 ID: %d\n标题: %s\n内容:\n%s\n---\n", a.ID, a.Title, content)
}

// splitIntoBatches splits articles into token-budgeted batches.
// Returns a slice of article slices, each within the token budget.
func (s *BriefingService) splitIntoBatches(articles []models.Article) [][]models.Article {
    budget := s.computeBudget()
    var batches [][]models.Article
    var currentBatch []models.Article
    currentTokens := 0

    for _, a := range articles {
        articleStr := s.buildArticleStringForEstimate(a)
        articleTokens := ai.Estimate(articleStr)

        if currentTokens+articleTokens > budget && len(currentBatch) > 0 {
            batches = append(batches, currentBatch)
            currentBatch = nil
            currentTokens = 0
        }

        if articleTokens > budget {
            // Single article exceeds budget: truncate content
            // We accept it as-is; the truncated version will be used
            // in buildArticlesInput which also truncates per-article content
        }

        currentBatch = append(currentBatch, a)
        currentTokens += articleTokens
    }

    if len(currentBatch) > 0 {
        batches = append(batches, currentBatch)
    }

    return batches
}
```

**buildArticleString 方法更新（基于截断 content）:**

```go
// buildArticlesInput builds the articles section of the prompt, truncating per-article
// content to maxContentChars characters to stay within token budget.
func (s *BriefingService) buildArticlesInput(articles []models.Article, budget int) string {
    var sb strings.Builder
    remaining := budget
    for _, a := range articles {
        articleStr := s.buildArticleStringForEstimate(a)
        articleTokens := ai.Estimate(articleStr)
        // Use proportional content limit
        maxChars := (remaining / articleTokens) * 4 // rough char/token ratio
        if maxChars < 200 {
            maxChars = 200
        }

        content := a.Content
        if content == "" {
            content = a.Summary
        }
        if len(content) > maxChars {
            content = content[:maxChars] + "..."
        }

        sb.WriteString(fmt.Sprintf("文章 ID: %d\n", a.ID))
        sb.WriteString(fmt.Sprintf("标题: %s\n", a.Title))
        sb.WriteString(fmt.Sprintf("内容:\n%s\n", content))
        sb.WriteString("---\n")
        remaining -= articleTokens
    }
    return sb.String()
}
```

- [ ] **Step 1: 修改 BriefingService struct 和构造函数** — 添加 aiConfig 字段
- [ ] **Step 2: 更新 cmd/server/main.go 中 NewBriefingService 调用** — 传入 aiConfig

```bash
grep -n "NewBriefingService" /home/zjx/code/mine/ai-reader-flow/cmd/server/main.go
# 找到调用处，修改为 NewBriefingService(&config.AppConfig_.AIProvider)
```

- [ ] **Step 3: 添加 computeBudget 和 splitIntoBatches 方法**
- [ ] **Step 4: 更新 buildArticlesInput 方法**
- [ ] **Step 5: Run build**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go build ./...`
Expected: 0 errors

- [ ] **Step 6: Commit**

```bash
git add internal/service/briefing_service.go cmd/server/main.go
git commit -m "refactor(briefing): add token budget and batch splitting to BriefingService

- BriefingService now accepts aiConfig for context window
- computeBudget(): contextWindow * 0.6 - overhead
- splitIntoBatches(): article grouping by token budget
- buildArticlesInput(): per-article content truncation based on budget

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: 创建 mergeBriefingResults 函数

**Files:**
- Modify: `internal/service/briefing_service.go`

**新增函数:**

```go
import "sort"
import "unicode"

// normalizeTopicName normalizes a topic name for merge matching.
// Converts to lowercase and removes spaces.
func normalizeTopicName(name string) string {
    return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "", -1), "\t", "", -1))
}

// mergeBriefingResults merges multiple BriefingResult batches into one.
// Topics with the same normalized name are merged (articles deduplicated by ID).
// Topics are sorted by article count (descending).
func mergeBriefingResults(batches []models.BriefingResult) models.BriefingResult {
    topicMap := make(map[string]*models.BriefingTopic)

    for _, batch := range batches {
        for i := range batch.Topics {
            topic := &batch.Topics[i]
            key := normalizeTopicName(topic.Name)
            existing, ok := topicMap[key]
            if !ok {
                // Clone the topic to avoid aliasing
                cloned := *topic
                topicMap[key] = &cloned
                continue
            }
            // Merge: deduplicate articles by ID
            seen := make(map[int64]bool)
            for _, a := range existing.Articles {
                seen[a.ID] = true
            }
            for _, a := range topic.Articles {
                if !seen[a.ID] {
                    existing.Articles = append(existing.Articles, a)
                    seen[a.ID] = true
                }
            }
        }
    }

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

- [ ] **Step 1: Write test for mergeBriefingResults**

```go
// internal/service/briefing_service_test.go
package service

import (
    "testing"
    "ai-rss-reader/internal/models"
)

func TestMergeBriefingResults(t *testing.T) {
    batch1 := models.BriefingResult{
        Topics: []models.BriefingTopic{
            {
                Name: "AI 大模型",
                Summary: "summary1",
                Articles: []models.BriefingTopicArticle{
                    {ID: 1, Insight: "insight1"},
                    {ID: 2, Insight: "insight2"},
                },
            },
        },
    }
    batch2 := models.BriefingResult{
        Topics: []models.BriefingTopic{
            {
                Name: "AI大模型", // same as batch1 topic (no space)
                Summary: "summary2",
                Articles: []models.BriefingTopicArticle{
                    {ID: 2, Insight: "insight2 longer"}, // duplicate ID 2
                    {ID: 3, Insight: "insight3"},
                },
            },
            {
                Name: "机器人",
                Summary: "summary3",
                Articles: []models.BriefingTopicArticle{
                    {ID: 4, Insight: "insight4"},
                },
            },
        },
    }

    result := mergeBriefingResults([]models.BriefingResult{batch1, batch2})

    // Should have 2 topics: "AI大模型" (merged, 3 articles) and "机器人" (1 article)
    if len(result.Topics) != 2 {
        t.Errorf("expected 2 topics, got %d", len(result.Topics))
    }

    // Find AI topic
    var aiTopic *models.BriefingTopic
    for i := range result.Topics {
        if normalizeTopicName(result.Topics[i].Name) == normalizeTopicName("AI大模型") {
            aiTopic = &result.Topics[i]
            break
        }
    }
    if aiTopic == nil {
        t.Fatal("AI topic not found after merge")
    }
    // Should have 3 unique articles (1, 2, 3)
    if len(aiTopic.Articles) != 3 {
        t.Errorf("AI topic should have 3 articles after dedup, got %d", len(aiTopic.Articles))
    }
}

func TestNormalizeTopicName(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"AI 大模型", "ai大模型"},
        {"AI大模型", "ai大模型"},
        {"  AI  大模型  ", "ai大模型"},
        {"robotics", "robotics"},
    }
    for _, tt := range tests {
        got := normalizeTopicName(tt.input)
        if got != tt.expected {
            t.Errorf("normalizeTopicName(%q) = %q, want %q", tt.input, got, tt.expected)
        }
    }
}
```

- [ ] **Step 2: Run tests — verify they fail (mergeBriefingResults not defined)**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go test ./internal/service/... -run "TestMerge|TestNormalize" -v`
Expected: FAIL (undefined: mergeBriefingResults)

- [ ] **Step 3: Add mergeBriefingResults and normalizeTopicName to briefing_service.go**

- [ ] **Step 4: Run tests — verify they pass**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go test ./internal/service/... -run "TestMerge|TestNormalize" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/briefing_service.go internal/service/briefing_service_test.go
git commit -m "feat(briefing): add mergeBriefingResults with topic deduplication

- normalizeTopicName(): lowercase + remove spaces for matching
- mergeBriefingResults(): merge batches, dedupe articles by ID
- Topics sorted by article count descending

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: 重构 GenerateBriefingWithProgress 使用分批

**Files:**
- Modify: `internal/service/briefing_service.go`

**修改 GenerateBriefingWithProgress 方法（替换原来的单次 AI 调用逻辑）:**

```go
// GenerateBriefingWithProgress creates a briefing with optional progress callback.
func (s *BriefingService) GenerateBriefingWithProgress(onProgress func(stage, detail string)) (*models.Briefing, error) {
    // 0. Check if already generated this round
    if onProgress != nil {
        onProgress("checking", "检查生成状态...")
    }
    if !s.LastBriefingAt.Before(s.LastRefreshAt) && !s.LastRefreshAt.IsZero() {
        return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
    }

    // 1. Create briefing record
    briefing := &models.Briefing{
        Status: "generating",
    }
    if err := s.briefingRepo.Create(briefing); err != nil {
        return nil, fmt.Errorf("create briefing: %w", err)
    }

    // 2. Get articles after last refresh
    if onProgress != nil {
        onProgress("fetching", "正在获取文章...")
    }
    articles, err := s.articleRepo.GetArticlesAfter(s.LastRefreshAt)
    if err != nil {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
        return nil, fmt.Errorf("get articles: %w", err)
    }
    if len(articles) == 0 {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", "暂无新文章")
        return nil, fmt.Errorf("暂无新文章")
    }

    // 3. Split into token-budgeted batches
    batches := s.splitIntoBatches(articles)
    totalBatches := len(batches)
    budget := s.computeBudget()

    // 4. Call AI for each batch
    provider := ai.GetProvider()
    var allResults []models.BriefingResult
    var lastErr error

    for i, batch := range batches {
        stage := fmt.Sprintf("batch %d/%d", i+1, totalBatches)
        if onProgress != nil {
            onProgress("analyzing", fmt.Sprintf("正在分析文章批次 %d/%d（共%d篇）...", i+1, totalBatches, len(batch)))
        }

        articlesInput := s.buildArticlesInput(batch, budget)
        prompt := s.buildPrompt(articlesInput, len(articles), i, totalBatches)

        result, err := provider.GenerateBriefing(prompt)
        if err != nil {
            lastErr = err
            log.Printf("[briefing] batch %d/%d failed: %v", i+1, totalBatches, err)
            continue
        }

        // 5. Parse AI result
        var br models.BriefingResult
        parseErr := s.parseAIResult(result, &br)
        if parseErr != nil {
            lastErr = parseErr
            log.Printf("[briefing] batch %d/%d parse failed: %v", i+1, totalBatches, parseErr)
            continue
        }

        allResults = append(allResults, br)
    }

    // 6. Check if any batch succeeded
    if len(allResults) == 0 {
        errMsg := fmt.Sprintf("所有批次均失败: %v", lastErr)
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", errMsg)
        return nil, fmt.Errorf("briefing generation failed: %w", lastErr)
    }

    // 7. Merge results
    if onProgress != nil {
        onProgress("generating", "正在合并简报...")
    }
    merged := mergeBriefingResults(allResults)

    if len(merged.Topics) == 0 {
        s.briefingRepo.UpdateStatus(briefing.ID, "failed", "无有效简报内容")
        return nil, fmt.Errorf("no valid topics after merge")
    }

    // 8. Store merged briefing items
    if onProgress != nil {
        onProgress("generating", "正在存储简报...")
    }
    for i, topic := range merged.Topics {
        item := &models.BriefingItem{
            BriefingID: briefing.ID,
            Topic:      topic.Name,
            Summary:    topic.Summary,
            SortOrder:  i,
        }
        if err := s.briefingRepo.CreateItem(item); err != nil {
            log.Printf("Warning: failed to create briefing item: %v", err)
            continue
        }
        for _, ta := range topic.Articles {
            title := ""
            for _, a := range articles {
                if a.ID == ta.ID {
                    title = a.Title
                    break
                }
            }
            ba := &models.BriefingArticle{
                BriefingItemID: item.ID,
                ArticleID:     ta.ID,
                Title:         title,
                Insight:       ta.Insight,
            }
            s.briefingRepo.CreateArticle(ba)
        }
    }

    // 9. Mark as completed
    s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")
    s.LastBriefingAt = time.Now()

    return briefing, nil
}
```

**parseAIResult 方法（从原 ExtractJSONBlock 改造）:**

```go
// parseAIResult parses AI response into BriefingResult, trying direct JSON
// first, then extracting from markdown code block.
func (s *BriefingService) parseAIResult(result string, out *models.BriefingResult) error {
    parseErr := json.Unmarshal([]byte(result), out)
    if parseErr == nil {
        return nil
    }

    // Try to extract JSON from markdown code block
    idx := strings.Index(result, "{")
    if idx == -1 {
        return fmt.Errorf("parse failed: no JSON found in response: %w", parseErr)
    }
    jsonStr := strings.TrimSpace(result[idx:])
    endIdx := strings.LastIndex(jsonStr, "}")
    if endIdx == -1 {
        return fmt.Errorf("parse failed: unclosed JSON in response: %w", parseErr)
    }
    jsonStr = jsonStr[:endIdx+1]
    if err := json.Unmarshal([]byte(jsonStr), out); err != nil {
        return fmt.Errorf("parse failed: %w | raw: %s", err, result)
    }
    return nil
}
```

**buildPrompt 方法签名变更:**

```go
// buildPrompt builds the full prompt for briefing generation.
// batchIndex and totalBatches are used to adjust topic limits.
func (s *BriefingService) buildPrompt(articlesInput string, totalArticles, batchIndex, totalBatches int) string {
    isSingleBatch := totalBatches == 1
    topicLimit := 5
    if !isSingleBatch {
        topicLimit = 3
    }
    // topicLimitStr used in prompt
    ...
}
```

**修改 prompt 中的 topic 数量限制:**

在 `buildPrompt` 的 JSON 示例部分，将 `"topics": []` 改为反映实际的每批限制数量，并添加引导词说明这是第 N 批。

- [ ] **Step 1: 添加 parseAIResult 方法**
- [ ] **Step 2: 修改 buildPrompt 方法签名和内容**
- [ ] **Step 3: 重写 GenerateBriefingWithProgress 使用分批逻辑**
- [ ] **Step 4: 更新 main.go 中 NewBriefingService 调用** — 传入 aiConfig

```go
// cmd/server/main.go - find NewBriefingService calls and update:
// briefingService := service.NewBriefingService(&config.AppConfig_.AIProvider)
```

- [ ] **Step 5: Run build**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go build ./...`
Expected: 0 errors

- [ ] **Step 6: Commit**

```bash
git add internal/service/briefing_service.go
git commit -m "refactor(briefing): implement batched AI calling with merge

- GenerateBriefingWithProgress: loop batches, call AI per batch
- parseAIResult: defensive JSON parsing with markdown block extraction
- Multi-batch aware buildPrompt with reduced topic limits
- mergeBriefingResults integration

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: 端到端验证

- [ ] **Step 1: 构建并部署**

```bash
cd /home/zjx/code/mine/ai-reader-flow
go build -o server ./cmd/server
docker compose build --no-cache api && docker compose up -d api
```

- [ ] **Step 2: 检查新 binary 是否包含新代码**

```bash
docker exec ai-reader-flow-api-1 grep -c "splitIntoBatches" /server
# Expected: 1 (found)
```

- [ ] **Step 3: 触发简报生成**

```bash
curl -s -X POST http://localhost:18562/api/briefings/generate
```

- [ ] **Step 4: 等待结果并检查状态**

```bash
sleep 60 && curl -s "http://localhost:18562/api/briefings?limit=3"
```

Expected: 新简报 status=completed，items 非空

- [ ] **Step 5: 验证内容质量**

```bash
curl -s "http://localhost:18562/api/briefings/$(id)/items" | python3 -m json.tool | head -50
```

检查：每篇 article 有 insight，每个 topic 有 summary，不是简单标题罗列

---

## 验收标准

1. `go build ./...` 通过，0 errors
2. `go test ./internal/ai/... -run TestEstimate` 通过
3. `go test ./internal/service/... -run "TestMerge|TestNormalize"` 通过
4. Docker 部署后，简报生成成功（status=completed）
5. 简报内容包含 per-article insight 和 topic summary（不是标题列表）
6. 多次生成验证分批逻辑稳定
