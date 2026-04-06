# Mark Read Implementation Plan

## Goal

Add two mark-read features:
1. **Mark all as read** — button in top header, marks all unread articles as read
2. **Mark feed as read** — button in articles list header when viewing a specific feed

## Architecture

**Backend:**
- `ArticleRepository`: Add `SetFeedArticlesStatus(feedId int64, status string)` and `SetAllArticlesStatus(status string)`
- API endpoints: `POST /api/feeds/{id}/mark-read` and `POST /api/articles/mark-all-read`

**Frontend:**
- Header: Add "Mark all read" button next to refresh-all button
- Articles list header: Add "Mark as read" button when a feed is selected

**Data flow:**
1. User clicks button → optimistic UI update → API call
2. On success: articles list refreshes with updated counts
3. On failure: revert UI, show error toast

## Tech Stack

Go (Chi router), React+TypeScript, SQLite

---

## Files to Modify

| File | Change |
|------|--------|
| `internal/repository/sqlite/article_repository.go` | Add `SetFeedArticlesStatus()` and `SetAllArticlesStatus()` |
| `cmd/server/main.go` | Add `POST /api/feeds/{id}/mark-read` and `POST /api/articles/mark-all-read` handlers + routes |
| `frontend/src/api.ts` | Add `markFeedRead(feedId)` and `markAllRead()` |
| `frontend/src/components/FeedList.tsx` | Add UI buttons for both features |
| `frontend/src/i18n/zh.ts` | Add translations |
| `frontend/src/i18n/en.ts` | Add translations |
