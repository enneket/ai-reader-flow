# Remove Embedding Vector Logic

## Status

Approved for implementation.

---

## Background

The system currently computes embedding vectors for articles using AI providers and uses them for semantic deduplication. The user wants to remove this feature entirely.

---

## Scope

**Remove:**
- `FilterAllArticlesNew()` function (filter_service.go)
- `semanticDedupBatch()` function (filter_service.go)
- `GetUnreadWithoutEmbedding()` method (article_repository.go)
- `SaveEmbedding()` method (article_repository.go)
- `embedding` and `quality_score` columns from articles table
- `Embedding` field from Article model (models.go)
- `EmbeddingProvider` interface and `GetEmbedding()` implementations (ai/provider.go)
- Background goroutine call to `FilterAllArticlesNew` (main.go)
- Related tests

**Keep (for filter rules phase):**
- `filterService.FilterArticle()`, `GetRules()`, `AddRule()`, `DeleteRule()`
- Filter rule API handlers and frontend settings

---

## Implementation

### Step 1: Database Migration

Add migration to drop embedding and quality_score columns:

```go
// internal/repository/sqlite/db.go
migrateDropColumns()
```

SQLite: `ALTER TABLE articles DROP COLUMN embedding, DROP COLUMN quality_score`

### Step 2: Article Model

Remove `Embedding []float32` field from `Article` struct in models.go.

### Step 3: ArticleRepository

Remove:
- `GetUnreadWithoutEmbedding()`
- `SaveEmbedding()`
- Remove `embedding` from all SELECT queries
- Remove embedding-related code from other methods

### Step 4: FilterService

Remove:
- `FilterAllArticlesNew()`
- `semanticDedupBatch()`
- Remove `ArticleRepository` embedding methods from interface if defined

### Step 5: AI Provider

Remove:
- `EmbeddingProvider` interface
- `GetEmbedding()` from `OpenAIProvider`
- `GetEmbedding()` from `ClaudeProvider`
- `GetEmbedding()` from `OllamaProvider`

### Step 6: API Server

Remove the background goroutine call to `FilterAllArticlesNew` in `handleRefreshAllFeeds`.

### Step 7: Tests

Remove embedding-related tests from:
- `article_repository_test.go`
- `filter_service_test.go`
- `ai_test.go`

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/repository/sqlite/db.go` | Add DROP COLUMN migration |
| `internal/models/models.go` | Remove Embedding field |
| `internal/repository/sqlite/article_repository.go` | Remove embedding methods and columns |
| `internal/service/filter_service.go` | Remove FilterAllArticlesNew, semanticDedupBatch |
| `internal/ai/provider.go` | Remove EmbeddingProvider, GetEmbedding |
| `cmd/server/main.go` | Remove FilterAllArticlesNew goroutine call |
| `internal/repository/sqlite/article_repository_test.go` | Remove embedding tests |
| `internal/service/filter_service_test.go` | Remove embedding tests |
| `internal/ai/ai_test.go` | Remove embedding tests |

---

## Verification

1. `go build ./cmd/server/...` succeeds
2. `go test ./...` passes
3. `cd frontend && npm run build` succeeds
