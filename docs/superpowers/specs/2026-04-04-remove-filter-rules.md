# Remove Filter Rules Logic

## Status

Approved for implementation.

---

## Background

The system has a filter rules feature (keyword/source-based article filtering) that is no longer needed. All filter rule CRUD logic and UI need to be removed.

---

## Scope

**Backend - Remove:**

### API Layer (main.go)
- Route `POST /api/articles/{id}/filter` and handler `handleFilterArticle`
- Routes `GET/POST/DELETE /api/filter-rules` and handlers
- Comment block `// ─── Filter Rule Handlers ───`

### Service Layer (filter_service.go)
- `FilterService.AddRule`, `GetRules`, `UpdateRule`, `DeleteRule` methods
- `FilterService.FilterArticle`, `evaluateRule`, `filterWithAI` methods
- `FilterService.FilterAllArticles` method (uses FilterArticle)
- `ruleRepo` field and `sqlite.NewFilterRuleRepository()` call
- Remove `ai` import if only used for filterWithAI

### Provider Layer (ai/provider.go)
- `FilterArticle(content string, rules []string)` from `AIServiceProvider` interface
- `FilterArticle` implementation from `OpenAIProvider`, `ClaudeProvider`, `OllamaProvider`

### Repository Layer
- DELETE: `internal/repository/sqlite/filter_rule_repository.go`

### Model Layer
- Remove `FilterRule` struct from `models.go`

**Frontend - Remove:**
- Filter rules section from `Settings.tsx`
- Filter rules API functions from `api.ts`
- i18n entries (en.ts, zh.ts)

**Tests - Remove:**
- Filter rule tests from `filter_service_test.go`
- `TestFilterRuleModel` from `models_test.go`
- FilterArticle tests from `ai_test.go`

---

## Files to Modify/Delete

| File | Changes |
|------|---------|
| `cmd/server/main.go` | Remove filter rule routes, handlers, handleFilterArticle |
| `internal/service/filter_service.go` | Remove filter rule methods, FilterArticle, FilterAllArticles |
| `internal/ai/provider.go` | Remove FilterArticle from interface and implementations |
| `internal/repository/sqlite/filter_rule_repository.go` | DELETE entire file |
| `internal/models/models.go` | Remove FilterRule struct |
| `frontend/src/components/Settings.tsx` | Remove filter rules UI section |
| `frontend/src/api.ts` | Remove filter rules API functions |
| `frontend/src/i18n/en.ts` | Remove filter rules i18n |
| `frontend/src/i18n/zh.ts` | Remove filter rules i18n |
| `internal/service/filter_service_test.go` | Remove filter rule tests |
| `internal/models/models_test.go` | Remove TestFilterRuleModel |
| `internal/ai/ai_test.go` | Remove FilterArticle tests |

---

## Implementation Order

1. Remove backend API routes and handlers (main.go)
2. Remove provider FilterArticle method
3. Remove filter service methods
4. Delete FilterRuleRepository
5. Remove FilterRule model
6. Remove frontend filter rules UI and API
7. Remove tests
8. Verify build and tests

---

## Verification

1. `go build ./cmd/server/...` succeeds
2. `go test ./...` passes
3. `cd frontend && npm run build` succeeds
