# Remove SSE + Add Progress Polling · Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove SSE infrastructure and replace with REST polling for progress status.

**Architecture:** Replace SSE push model (`/api/events`) with REST pull model (`GET /api/progress`). The `GlobalRefreshStatus` and `GlobalOperationState` structures already hold the needed state — SSE broadcasts are redundant since the state is already being written. Frontend polls `/api/progress` every second instead of listening to SSE events.

**Tech Stack:** Go (Chi router), React + TypeScript

---

## File Map

| File | Change |
|------|--------|
| `cmd/server/main.go` | Add `GET /api/progress` handler; remove `GET /api/events` route + `handleSSEvents`; strip all `GlobalBroadcaster.Broadcast` calls from `handleRefreshAllFeeds` and `handleGenerateBriefing` |
| `internal/events/events.go` | Check if any references remain; likely fully unused — delete if so |
| `frontend/src/api.ts` | Add `getProgress()` method |
| `frontend/src/components/Briefing.tsx` | Replace SSE EventSource with `progressPollTimer` polling `GET /api/progress` |
| `frontend/src/components/BriefingDetail.tsx` | Remove SSE EventSource entirely; keep 3s polling |

---

## Task 1: Add `GET /api/progress` endpoint (Backend)

**Files:**
- Modify: `cmd/server/main.go` (add handler + route)
- Modify: `internal/events/events.go` (add exported helper structs)

- [ ] **Step 1: In `internal/events/events.go`, add exported progress response type**

Add after `type Event struct` (around line 109):

```go
// ProgressResponse is the JSON payload returned by GET /api/progress
type ProgressResponse struct {
	Operation string `json:"operation"` // "idle" | "refreshing" | "generating"
	Refresh  *RefreshStatusDTO `json:"refresh,omitempty"`
}

// RefreshStatusDTO mirrors GlobalRefreshStatus for JSON serialization
type RefreshStatusDTO struct {
	InProgress bool   `json:"inProgress"`
	Current    int    `json:"current"`
	Total      int    `json:"total"`
	FeedTitle  string `json:"feedTitle"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	Error      string `json:"error"`
}
```

- [ ] **Step 2: In `cmd/server/main.go`, add `GET /api/progress` handler**

Add before `handleSSEvents` (around line 1241):

```go
// ─── Progress Polling ──────────────────────────────────────────────────────────

func handleProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current operation
	events.GlobalOperationState.mutex.Lock()
	op := events.GlobalOperationState.current
	events.GlobalOperationState.mutex.Unlock()

	resp := events.ProgressResponse{Operation: op}

	if op == "refreshing" || op == "generating" {
		events.GlobalRefreshStatus.Mutex.Lock()
		resp.Refresh = &events.RefreshStatusDTO{
			InProgress: events.GlobalRefreshStatus.InProgress,
			Current:    events.GlobalRefreshStatus.Current,
			Total:      events.GlobalRefreshStatus.Total,
			FeedTitle:  events.GlobalRefreshStatus.FeedTitle,
			Success:    events.GlobalRefreshStatus.Success,
			Failed:     events.GlobalRefreshStatus.Failed,
			Error:      events.GlobalRefreshStatus.Error,
		}
		events.GlobalRefreshStatus.Mutex.Unlock()
	}

	writeJSON(w, http.StatusOK, resp)
}
```

Note: `OperationState.mutex` is unexported; add a getter method in `events.go` instead:

Add to `events.go` after `Unlock()`:

```go
// Current returns the current operation name ("idle", "refreshing", "generating")
func (s *OperationState) Current() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.current
}
```

Then update `handleProgress` to use `events.GlobalOperationState.Current()` instead of direct mutex access.

- [ ] **Step 3: Register the route in `main.go`**

Find the section around line 155-159:
```go
// SSE events stream
mux.HandleFunc("GET /api/events", handleSSEvents)
```

Replace with:
```go
// Progress polling (replaces SSE)
mux.HandleFunc("GET /api/progress", handleProgress)
```

- [ ] **Step 4: Build to verify**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go build ./cmd/server/...`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go internal/events/events.go
git commit -m "feat(api): add GET /api/progress endpoint for polling"
```

---

## Task 2: Strip SSE Broadcast calls from `handleRefreshAllFeeds`

**Files:**
- Modify: `cmd/server/main.go:404-500`

- [ ] **Step 1: Review current handleRefreshAllFeeds SSE broadcast lines**

Lines needing removal in `handleRefreshAllFeeds`:
- Line 430: `events.GlobalBroadcaster.Broadcast(events.EventRefreshStart, ...)`
- Line 447-454: `events.GlobalBroadcaster.Broadcast(events.EventRefreshProgress, ...)`
- Line 486: `events.GlobalBroadcaster.Broadcast(events.EventRefreshError, ...)`
- Line 495: `events.GlobalBroadcaster.Broadcast(events.EventRefreshComplete, ...)`

Keep all `GlobalRefreshStatus` updates (they feed the polling endpoint).

- [ ] **Step 2: Remove the 4 Broadcast calls**

Leave all `GlobalRefreshStatus.Mutex.Lock() ... .Unlock()` blocks intact.

Run: `go build ./cmd/server/...`
Expected: Clean build

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "refactor(api): remove SSE Broadcast from handleRefreshAllFeeds"
```

---

## Task 3: Strip SSE Broadcast calls from `handleGenerateBriefing`

**Files:**
- Modify: `cmd/server/main.go:725-803`

- [ ] **Step 1: Review handleGenerateBriefing SSE broadcast lines**

Lines needing removal:
- Line 757: `events.GlobalBroadcaster.Broadcast(events.EventBriefingStart, ...)`
- Line 762: `events.GlobalBroadcaster.Broadcast(events.EventRefreshStart, ...)`
- Line 765-772: `events.GlobalBroadcaster.Broadcast(events.EventRefreshProgress, ...)`
- Line 776: `events.GlobalBroadcaster.Broadcast(events.EventRefreshError, ...)`
- Line 777: `events.GlobalBroadcaster.Broadcast(events.EventBriefingError, ...)`
- Line 781: `events.GlobalBroadcaster.Broadcast(events.EventRefreshComplete, ...)`
- Line 785-788: `events.GlobalBroadcaster.Broadcast(events.EventBriefingProgress, ...)`
- Line 792: `events.GlobalBroadcaster.Broadcast(events.EventBriefingError, ...)`
- Line 800-802: `events.GlobalBroadcaster.Broadcast(events.EventBriefingComplete, ...)`

- [ ] **Step 2: Update GlobalRefreshStatus during refresh phase in handleGenerateBriefing**

The refresh phase inside `handleGenerateBriefing` (lines 759-773) does NOT update `GlobalRefreshStatus` — only SSE broadcasts. Add status updates to match what `handleRefreshAllFeeds` does.

In the goroutine after `feeds, _ := rssService.GetFeeds()` (around line 760), add before the `RefreshAllFeedsWithProgress` call:

```go
events.GlobalRefreshStatus.Mutex.Lock()
events.GlobalRefreshStatus.InProgress = true
events.GlobalRefreshStatus.Current = 0
events.GlobalRefreshStatus.Total = total
events.GlobalRefreshStatus.FeedTitle = ""
events.GlobalRefreshStatus.Success = 0
events.GlobalRefreshStatus.Failed = 0
events.GlobalRefreshStatus.Error = ""
events.GlobalRefreshStatus.Results = make(map[int64]events.FeedRefreshResult)
events.GlobalRefreshStatus.Mutex.Unlock()
```

Inside the `RefreshAllFeedsWithProgress` callback, add `GlobalRefreshStatus` updates mirroring `handleRefreshAllFeeds` (same logic as lines 456-480).

After refresh completes (success or error), set `InProgress = false`.

- [ ] **Step 3: Remove all Broadcast calls**

Run: `go build ./cmd/server/...`
Expected: Clean build

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "refactor(api): remove SSE Broadcast from handleGenerateBriefing"
```

---

## Task 4: Remove SSE endpoint and handler

**Files:**
- Modify: `cmd/server/main.go`
- Delete: `internal/events/events.go` (if fully unused after Tasks 1-3)

- [ ] **Step 1: Remove `GET /api/events` route (line 159)**

Delete the line:
```go
mux.HandleFunc("GET /api/events", handleSSEvents)
```

- [ ] **Step 2: Remove `handleSSEvents` function (lines 1241-1280+)**

Delete the entire `// ─── SSE Events ───` section.

- [ ] **Step 3: Check if events.go is still referenced**

Run: `grep -r "events\." cmd/server/main.go | grep -v "events.GlobalOperationState\|events.GlobalRefreshStatus\|events.FeedRefreshResult\|events.RefreshProgress\|events.BriefingProgress\|events.ProgressResponse\|events.RefreshStatusDTO\|events.RefreshComplete\|events.NewBroadcaster\|events.NewBroadcaster\|events.BriefingComplete\|events.BriefingProgress\|events.OperationState\|events.Event"`
Expected: Only `events.GlobalOperationState` and `events.GlobalRefreshStatus` remain (used for status)

If only those two remain, `events.go` is still needed for the type definitions and `OperationState` struct — do NOT delete it. The file contains types used by the progress polling path.

- [ ] **Step 4: Build and commit**

Run: `go build ./cmd/server/...`
Expected: Clean build

```bash
git add cmd/server/main.go
git commit -m "refactor(api): remove SSE /api/events endpoint and handler"
```

---

## Task 5: Add `getProgress()` to frontend API client

**Files:**
- Modify: `frontend/src/api.ts`

- [ ] **Step 1: Add ProgressResponse type and getProgress method**

Find the `Briefing` type definition and add nearby:

```typescript
export type ProgressResponse = {
  operation: 'idle' | 'refreshing' | 'generating'
  refresh?: {
    inProgress: boolean
    current: number
    total: number
    feedTitle: string
    success: number
    failed: number
    error: string
  } | null
}
```

Add method to `api` object:

```typescript
async getProgress(): Promise<ProgressResponse> {
  const res = await fetch('/api/progress')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}
```

- [ ] **Step 2: Build frontend to verify**

Run: `cd /home/zjx/code/mine/ai-reader-flow/frontend && npm run build 2>&1 | tail -20`
Expected: No TypeScript errors related to new type/method

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api.ts
git commit -m "feat(frontend): add getProgress() API method"
```

---

## Task 6: Replace SSE with polling in Briefing.tsx

**Files:**
- Modify: `frontend/src/components/Briefing.tsx`

- [ ] **Step 1: Remove SSE EventSource useEffect (lines 49-92)**

Delete the entire `useEffect` block that creates `new EventSource('/api/events')`. Replace with:

```typescript
// Progress polling (replaces SSE)
useEffect(() => {
  if (progress.type === 'idle') return

  const poll = async () => {
    try {
      const data = await api.getProgress()
      if (data.operation === 'idle') {
        setProgress({type: 'idle', message: ''})
        if (progress.type === 'refreshing') loadBriefings(0)
        return
      }
      if (data.operation === 'refreshing' && data.refresh) {
        setProgress({
          type: 'refreshing',
          message: `正在刷新 ${data.refresh.current}/${data.refresh.total} 个订阅源: ${data.refresh.feedTitle || ''}`,
          current: data.refresh.current,
          total: data.refresh.total,
        })
      }
      if (data.operation === 'generating' && data.refresh) {
        // Show refresh progress even during briefing generation
        setProgress({
          type: 'refreshing',
          message: `正在刷新 ${data.refresh.current}/${data.refresh.total} 个订阅源...`,
          current: data.refresh.current,
          total: data.refresh.total,
        })
      }
    } catch {
      // Non-critical, keep polling
    }
  }

  const timer = setInterval(poll, 1000)
  poll()
  return () => clearInterval(timer)
}, [progress.type])
```

Note: The existing `generating` state polling useEffect (lines 94-154) handles briefing completion detection. This new effect only handles `refreshing` state progress display.

- [ ] **Step 2: Fix stale closure in briefing completion polling (lines 131-133)**

The `if (generating)` inside the poll callback references a stale closure value. Change:

```typescript
// Still generating - keep polling
if (generating) {
  briefingPollTimer.current = setTimeout(poll, 1000)
}
```

To:

```typescript
// Keep polling regardless — the API status drives completion
briefingPollTimer.current = setTimeout(poll, 1000)
```

And update the cleanup and dependency to avoid double-polling when `progress.type === 'refreshing'` transitions to `generating`.

Actually, simplify: keep the existing `generating` polling for completion detection, but make the SSE `briefing:progress` listener obsolete. Remove lines 79-89 (the SSE briefing:progress listener) entirely since that event will no longer be fired.

- [ ] **Step 3: Update handleGenerate to not reference SSE**

Line 185 comment says "SSE will handle setting generating=false on completion/error" — update to reflect polling:

Change comment to: `// Polling will handle setting generating=false on completion/error`

- [ ] **Step 4: Verify build**

Run: `cd /home/zjx/code/mine/ai-reader-flow/frontend && npm run build 2>&1 | tail -20`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Briefing.tsx
git commit -m "refactor(frontend): replace SSE with polling for progress updates"
```

---

## Task 7: Remove SSE from BriefingDetail.tsx

**Files:**
- Modify: `frontend/src/components/BriefingDetail.tsx`

- [ ] **Step 1: Remove SSE EventSource (lines 36-50)**

Keep the `setInterval` polling (lines 29-34). Remove:
- The `new EventSource('/api/events')` line
- The `briefing:complete` listener
- The `briefing:error` listener

The 3s polling already handles completion detection. The SSE was only for real-time completion which is now redundant.

After:
```typescript
useEffect(() => {
  // Poll every 3s while briefing is generating
  const pollInterval = setInterval(() => {
    if (briefing?.status === 'generating') {
      loadBriefing()
    }
  }, 3000)

  return () => clearInterval(pollInterval)
}, [briefing?.status])
```

- [ ] **Step 2: Build and commit**

Run: `cd /home/zjx/code/mine/ai-reader-flow/frontend && npm run build 2>&1 | tail -20`

```bash
git add frontend/src/components/BriefingDetail.tsx
git commit -m "refactor(frontend): remove SSE from BriefingDetail"
```

---

## Task 8: Final verification

- [ ] **Step 1: Full backend build**

Run: `cd /home/zjx/code/mine/ai-reader-flow && go build ./cmd/server/... && go build ./...`

- [ ] **Step 2: Full frontend build**

Run: `cd /home/zjx/code/mine/ai-reader-flow/frontend && npm run build`

- [ ] **Step 3: API smoke test**

Run: `curl -s http://localhost:8080/api/progress | python3 -m json.tool`
Expected: `{"operation": "idle", "refresh": null}`

- [ ] **Step 4: Commit all remaining changes**

```bash
git status
# Should show only modified files from Tasks 1-7
git add -A
git commit -m "feat: remove SSE, replace with /api/progress polling"
```

---

## Spec Self-Review

1. **Placeholder scan:** No TBD/TODOs. All code is complete and runnable.
2. **Internal consistency:** Types in `events.go` (ProgressResponse, RefreshStatusDTO) match what frontend `api.ts` expects. `handleProgress` writes the same fields that `Briefing.tsx` polls.
3. **Scope check:** Single focused goal — remove SSE, replace with polling. All 7 tasks trace to this goal.
4. **Boundary clarity:** Each task is isolated: backend adds endpoint → strips SSE calls → frontend adopts new endpoint. No cross-task dependencies.
