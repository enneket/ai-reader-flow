# Per-Feed Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-feed refresh button so users can refresh individual feeds without refreshing all.

**Architecture:** Backend exposes `POST /api/feeds/{id}/refresh` which calls the existing `RSSService.RefreshFeed(feedID)`. Frontend adds a refresh button on each feed item in the sidebar.

**Tech Stack:** Go (Chi router), React + TypeScript

---

## Task 1: Add POST /api/feeds/{id}/refresh endpoint

**Files:**
- Modify: `cmd/server/main.go`
- No backend service changes needed — `RSSService.RefreshFeed(feedID)` already exists

- [ ] **Step 1: Read main.go routes to find where to add (around line 78-82)**

Current routes:
```go
mux.HandleFunc("GET /api/feeds", handleGetFeeds)
mux.HandleFunc("POST /api/feeds", handleAddFeed)
mux.HandleFunc("PATCH /api/feeds/{id}", handleUpdateFeed)
mux.HandleFunc("DELETE /api/feeds/{id}", handleDeleteFeed)
mux.HandleFunc("GET /api/feeds/dead", handleGetDeadFeeds)
```

- [ ] **Step 2: Add new route after DELETE**

```go
mux.HandleFunc("POST /api/feeds/{id}/refresh", handleRefreshFeed)
```

- [ ] **Step 3: Add handler function (after handleDeleteFeed ~line 270)**

```go
func handleRefreshFeed(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/feeds", r)
    if !ok {
        http.Error(w, "invalid feed id", http.StatusBadRequest)
        return
    }

    if err := rssService.RefreshFeed(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    feed, _ := rssService.GetFeed(id)
    writeJSON(w, http.StatusOK, feed)
}
```

- [ ] **Step 4: Build to verify**

```bash
go build ./...
```

Expected: no output (clean build)

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(api): add POST /api/feeds/{id}/refresh endpoint"
```

---

## Task 2: Add per-feed refresh button in FeedList

**Files:**
- Modify: `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: Add refreshOneFeed handler (around line 234)**

```tsx
const handleRefreshOneFeed = async (feedId: number, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    try {
        await api.refreshFeed(feedId)
        await loadFeeds()
        if (selectedFeed?.id === feedId) {
            await loadArticles(feedId)
        }
    } catch (err: any) {
        setError(err.message || '刷新失败')
    }
}
```

- [ ] **Step 2: Add to api.ts if not already present**

Check `frontend/src/api.ts` for `refreshFeed`. If not present, add:
```typescript
refreshFeed: (id: number): Promise<void> => {
    return fetch(`/api/feeds/${id}/refresh`, {method: 'POST'}).then(r => {
        if (!r.ok) throw new Error('refresh failed')
    })
},
```

- [ ] **Step 3: Add refresh button in feed-item (before the delete button, around line 454)**

Add a small refresh button next to each feed. Use the existing `RefreshCw` icon from lucide-react (already imported at line 4).

```tsx
<button
    onClick={(e) => handleRefreshOneFeed(feed.id, e)}
    className="btn btn-ghost btn-sm btn-icon"
    aria-label="Refresh feed"
    title="刷新"
>
    <RefreshCw size={12} />
</button>
```

- [ ] **Step 4: Test frontend build**

```bash
cd frontend && npm run build 2>&1 | tail -10
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/FeedList.tsx frontend/src/api.ts
git commit -m "feat(frontend): add per-feed refresh button"
```

---

## Task 3: E2E verification

- [ ] **Step 1: Build and restart API**

```bash
go build -o server ./cmd/server && docker compose build --no-cache api && docker compose up -d api
```

- [ ] **Step 2: Rebuild and restart Web**

```bash
docker compose build web && docker compose up -d web
```

- [ ] **Step 3: Test via curl**

```bash
curl -s -X POST http://localhost:18562/api/feeds/22/refresh && echo ""
curl -s http://localhost:18562/api/feeds/22 | python3 -c "import sys,json; d=json.load(sys.stdin); print('last_refreshed:', d.get('last_refreshed',''))"
```

Expected: refresh succeeds, last_refreshed updated
