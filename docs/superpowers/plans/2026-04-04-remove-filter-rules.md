# Remove Filter Rules Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all filter rule CRUD logic and UI from the codebase.

**Architecture:** Remove dead code following dependency order (frontend → handlers → service → repository → model).

---

### Task 1: Remove backend filter rule routes and handlers

**Files:**
- Modify: `cmd/server/main.go`

- [ ] Remove filter rule route registrations (lines ~101-103)
- [ ] Remove `handleGetFilterRules` function
- [ ] Remove `handleAddFilterRule` function
- [ ] Remove `handleDeleteFilterRule` function
- [ ] Remove `FilterRuleRepository` from `NewFeedService` dependency

---

### Task 2: Remove FilterRuleRepository

**Files:**
- DELETE: `internal/repository/sqlite/filter_rule_repository.go`

---

### Task 3: Remove FilterRule model and filter service methods

**Files:**
- Modify: `internal/models/models.go` - Remove `FilterRule` struct
- Modify: `internal/service/filter_service.go` - Remove `AddRule`, `GetRules`, `UpdateRule`, `DeleteRule`, `evaluateRule`, `filterWithAI` methods

---

### Task 4: Remove frontend filter rules UI and API

**Files:**
- Modify: `frontend/src/components/Settings.tsx` - Remove filter rules section
- Modify: `frontend/src/api.ts` - Remove filter rules API functions
- Modify: `frontend/src/i18n/en.ts` - Remove filter rules i18n
- Modify: `frontend/src/i18n/zh.ts` - Remove filter rules i18n

---

### Task 5: Remove filter rule tests

**Files:**
- Modify: `internal/service/filter_service_test.go` - Remove filter rule tests
- Modify: `internal/models/models_test.go` - Remove `TestFilterRuleModel`

---

### Task 6: Verify build and tests

**Files:**
- Run: `go build ./cmd/server/...`
- Run: `go test ./...`
- Run: `cd frontend && npm run build`

---

### Task 7: Commit

```bash
git add -A
git commit -m "feat(api): remove filter rules CRUD

Remove all filter rule logic:
- FilterRuleRepository
- FilterService.AddRule/GetRules/UpdateRule/DeleteRule
- FilterRule model
- API handlers and routes
- Frontend settings UI
- Related tests

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
