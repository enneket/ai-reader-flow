# Feed Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-feed settings button that lets users edit the feed's title and URL.

**Architecture:** Backend extends the existing `PATCH /api/feeds/{id}` endpoint to accept `title` and `url` fields. Frontend adds a settings button (gear icon) on each feed list item that opens a modal with editable title and URL fields.

**Tech Stack:** Go (Chi router), React + TypeScript, Ant Design Modal

---

## Task 1: Extend FeedRepository.Update to support URL

**Files:**
- Modify: `internal/repository/sqlite/feed_repository.go:88-94`

- [ ] **Step 1: Read current FeedRepository.Update**

```go
func (r *FeedRepository) Update(feed *models.Feed) error {
    _, err := DB.Exec(
        `UPDATE feeds SET title = ?, description = ?, icon_url = ?, last_fetched = ?, group_name = ? WHERE id = ?`,
        feed.Title, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), feed.Group, feed.ID,
    )
    return err
}
```

- [ ] **Step 2: Add url to UPDATE SQL**

Replace the SQL with:

```go
func (r *FeedRepository) Update(feed *models.Feed) error {
    _, err := DB.Exec(
        `UPDATE feeds SET title = ?, url = ?, description = ?, icon_url = ?, last_fetched = ?, group_name = ? WHERE id = ?`,
        feed.Title, feed.URL, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), feed.Group, feed.ID,
    )
    return err
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/repository/sqlite/feed_repository.go
git commit -m "feat(repo): add url field to FeedRepository.Update"
```

---

## Task 2: Extend handleUpdateFeed to accept Title and URL

**Files:**
- Modify: `cmd/server/main.go:294-317`

- [ ] **Step 1: Read current handleUpdateFeed**

```go
func handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/feeds", r)
    if !ok {
        http.Error(w, "invalid feed id", http.StatusBadRequest)
        return
    }
    var req struct {
        Group string `json:"group"`
    }
    if !readJSON(w, r, &req) {
        return
    }
    feed, err := rssService.GetFeed(id)
    if err != nil {
        http.Error(w, "feed not found", http.StatusNotFound)
        return
    }
    feed.Group = req.Group
    if err := rssService.UpdateFeed(feed); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, feed)
}
```

- [ ] **Step 2: Replace req struct to include Title and URL**

```go
func handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID("/api/feeds", r)
    if !ok {
        http.Error(w, "invalid feed id", http.StatusBadRequest)
        return
    }
    var req struct {
        Title string `json:"title"`
        URL   string `json:"url"`
        Group string `json:"group"`
    }
    if !readJSON(w, r, &req) {
        return
    }
    feed, err := rssService.GetFeed(id)
    if err != nil {
        http.Error(w, "feed not found", http.StatusNotFound)
        return
    }
    if req.Title != "" {
        feed.Title = req.Title
    }
    if req.URL != "" {
        feed.URL = req.URL
    }
    feed.Group = req.Group
    if err := rssService.UpdateFeed(feed); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, feed)
}
```

- [ ] **Step 3: Build to verify**

```bash
go build ./...
```

Expected: no output (clean build)

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(api): handleUpdateFeed accepts title and url fields"
```

---

## Task 3: Add feed settings button and edit modal in FeedList

**Files:**
- Modify: `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: Read current feed-item rendering (lines ~391-417)**

The feed item renders with:
- title, url display
- unread badge
- delete button

Add a settings button (gear icon) next to the delete button.

- [ ] **Step 2: Add Settings import and state**

In the imports from lucide-react, add `Settings` (already imported at line 4, used in masthead). Reuse it.

Add state for the edit modal:
```tsx
const [editModalOpen, setEditModalOpen] = useState(false)
const [editFeed, setEditFeed] = useState<{id: number; title: string; url: string} | null>(null)
```

Add handler:
```tsx
const handleEditFeed = (feed: Feed, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setEditFeed({id: feed.id, title: feed.title || '', url: feed.url})
    setEditModalOpen(true)
}
```

Add Modal after the progressModal (around line 292):
```tsx
<Modal
    open={editModalOpen}
    title="订阅源设置"
    onOk={async () => {
        if (!editFeed) return
        try {
            await api.updateFeed(editFeed.id, {title: editFeed.title, url: editFeed.url, group: ''})
            setEditModalOpen(false)
            await loadFeeds()
        } catch (err: any) {
            setError(err.message || '更新失败')
        }
    }}
    onCancel={() => setEditModalOpen(false)}
    okText="保存"
    cancelText="取消"
>
    <div style={{display: 'flex', flexDirection: 'column', gap: 12}}>
        <div>
            <label style={{fontSize: '0.85rem', marginBottom: 4, display: 'block'}}>标题</label>
            <input
                className="form-input"
                value={editFeed?.title || ''}
                onChange={e => setEditFeed(prev => prev ? {...prev, title: e.target.value} : null)}
            />
        </div>
        <div>
            <label style={{fontSize: '0.85rem', marginBottom: 4, display: 'block'}}>订阅源链接</label>
            <input
                className="form-input"
                value={editFeed?.url || ''}
                onChange={e => setEditFeed(prev => prev ? {...prev, url: e.target.value} : null)}
            />
        </div>
    </div>
</Modal>
```

- [ ] **Step 3: Add settings button in feed-item**

In the feed-item div (around line 409), after the delete button, add:
```tsx
<button
    onClick={(e) => handleEditFeed(feed, e)}
    className="btn btn-ghost btn-sm btn-icon"
    aria-label="Edit feed"
>
    <Settings size={12} />
</button>
```

- [ ] **Step 4: Check api.updateFeed signature**

In `frontend/src/api.ts`, find `updateFeed`. It should be:
```typescript
updateFeed: (id: number, feed: Partial<Feed>): Promise<void>
```
Verify it calls `PATCH /api/feeds/{id}` with the feed body.

- [ ] **Step 5: Test frontend build**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): add feed settings button with title and url editing"
```

---

## Task 4: E2E verification

- [ ] **Step 1: Build and restart API**

```bash
docker compose build --no-cache api && docker compose up -d api
```

- [ ] **Step 2: Open app, find a feed, click settings gear**

Verify:
- Gear icon appears on feed items
- Modal opens with current title and URL
- Edit title/url and save → feed updates in list
- Cancel → modal closes without changes
