# Remove Embedding Vector Logic - Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans or superpowers:subagent-driven-development.

**Goal:** Remove all embedding vector computation logic from the codebase.

**Approach:** Delete dead code following dependency order (provider → repository → service → main).

---

## Task 1: Remove background goroutine call

**Files:**
- Modify: `cmd/server/main.go:476-486`

Remove the `go func()` block containing `FilterAllArticlesNew()` inside `handleRefreshAllFeeds`.

---

## Task 2: Remove Embedding field from Article model

**Files:**
- Modify: `internal/models/models.go`

Remove `Embedding []float32` field from `Article` struct.

---

## Task 3: Remove embedding methods from ArticleRepository

**Files:**
- Modify: `internal/repository/sqlite/article_repository.go`

Remove:
- `GetUnreadWithoutEmbedding()` method
- `SaveEmbedding()` method
- Remove `embedding` column from all SELECT queries
- Remove JSON unmarshal for embedding in article scan methods

---

## Task 4: Add database migration

**Files:**
- Modify: `internal/repository/sqlite/db.go`

Add migration to drop embedding and quality_score columns from articles table:
```go
_ = migrateDropColumn("articles", "embedding")
_ = migrateDropColumn("articles", "quality_score")
```

---

## Task 5: Remove embedding code from FilterService

**Files:**
- Modify: `internal/service/filter_service.go`

Remove:
- `semanticDedupBatch()` method
- `FilterAllArticlesNew()` method
- Update interface if applicable

---

## Task 6: Remove EmbeddingProvider from AI Provider

**Files:**
- Modify: `internal/ai/provider.go`

Remove:
- `EmbeddingProvider` interface
- `GetEmbedding()` from `OpenAIProvider`
- `GetEmbedding()` from `ClaudeProvider`
- `GetEmbedding()` from `OllamaProvider`

---

## Task 7: Remove embedding tests

**Files:**
- Modify: `internal/repository/sqlite/article_repository_test.go`
- Modify: `internal/service/filter_service_test.go`
- Modify: `internal/ai/ai_test.go`

Remove all test functions and helper types related to embeddings.

---

## Task 8: Verify build and tests

**Files:**
- Run: `go build ./cmd/server/...`
- Run: `go test ./...`
- Run: `cd frontend && npm run build`

---

## Task 9: Commit

```bash
git add -A
git commit -m "feat(api): remove embedding vector logic

Remove all embedding computation from:
- FilterAllArticlesNew and semanticDedupBatch
- GetUnreadWithoutEmbedding and SaveEmbedding
- EmbeddingProvider interface and GetEmbedding
- Database columns (embedding, quality_score)
- Related tests

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
