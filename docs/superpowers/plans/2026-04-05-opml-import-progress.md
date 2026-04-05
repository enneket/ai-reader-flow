# OPML Import Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add progress tracking to OPML import — user sees "Importing 3/10: Hacker News" instead of just "Importing..."

**Architecture:** Backend stores import job state in memory map, returns 202 immediately, runs import in goroutine. Frontend polls GET endpoint every 200ms.

**Tech Stack:** Go (chi router), React/TypeScript

---

## Task 1: Backend - Add import job state and async handler

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Read current OPML import handler**

Read `cmd/server/main.go` lines ~885-912 to see `handleImportOPML`.

- [ ] **Step 2: Add import job types and global state**

Add near the top of `main.go` (after imports, around line 30):

```go
type importJob struct {
    Total     int
    Current   int
    FeedName  string
    Success   int
    Failed    int
    Done      bool
    CreatedAt time.Time
}

var importJobs = make(map[string]*importJob)
var importJobsMu sync.Mutex
var importOperationMu sync.Mutex
```

- [ ] **Step 3: Add GET handler for import progress**

Add after `handleImportOPML` (around line 912):

```go
func handleGetImportProgress(w http.ResponseWriter, r *http.Request) {
    jobID := strings.TrimPrefix(r.URL.Path, "/api/opml/import/")
    importJobsMu.Lock()
    job, ok := importJobs[jobID]
    importJobsMu.Unlock()
    if !ok {
        http.Error(w, "job not found", http.StatusNotFound)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "current":   job.Current,
        "total":     job.Total,
        "feedName":  job.FeedName,
        "success":   job.Success,
        "failed":    job.Failed,
        "done":      job.Done,
    })
}
```

- [ ] **Step 4: Register new route**

Find the route registration around line 120 (mux.HandleFunc for OPML). Add:

```go
mux.HandleFunc("GET /api/opml/import/{jobId}", handleGetImportProgress)
```

Note: The pattern `/api/opml/import/{jobId}` must be registered BEFORE the `POST /api/opml` handler to avoid path conflicts.

- [ ] **Step 5: Rewrite handleImportOPML for async**

Replace the current `handleImportOPML` body (lines ~895-912) with the async version from the spec. Key points:
- Use `importOperationMu.TryLock()` — if locked, return 409
- Parse OPML first (synchronously), return 400 on parse error
- Return 202 with jobId immediately
- Run import loop in goroutine with proper mutex handling
- Cleanup job after 1 hour

- [ ] **Step 6: Verify Go builds**

```bash
cd /home/dabao/code/ai-reader-flow && go build ./cmd/server/...
```

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(api): add OPML import progress tracking with async handler

- Add importJob struct and in-memory job storage
- POST returns 202 + jobId, runs import in goroutine
- GET /api/opml/import/{jobId} returns progress
- Use operation mutex to prevent concurrent imports

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Frontend - Add API function and polling UI

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/components/Settings.tsx`

- [ ] **Step 1: Add getImportProgress to api.ts**

Find the OPML section in `api.ts`. Add after `importOPML`:

```typescript
getImportProgress: (jobId: string) => {
    return request<{
        current: number
        total: number
        feedName: string
        success: number
        failed: number
        done: boolean
    }>(`/opml/import/${jobId}`)
},
```

Note: This uses the `/opml` prefix (not `/api/opml`), matching the existing pattern in api.ts.

- [ ] **Step 2: Update Settings.tsx import state and add handlers**

In `Settings.tsx`:

Replace the `importing` state (line 34):
```typescript
const [importing, setImporting] = useState(false)
```

With:
```typescript
const [importing, setImporting] = useState(false)
const [importProgress, setImportProgress] = useState<{
    current: number
    total: number
    feedName: string
    success: number
    failed: number
} | null>(null)
```

- [ ] **Step 3: Add polling logic to handleImportOPML**

Replace `handleImportOPML` with:

```typescript
const handleImportOPML = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setImporting(true)
    setImportProgress({ current: 0, total: 0, feedName: '', success: 0, failed: 0 })
    setError('')
    setSuccess('')
    try {
        const result = await api.importOPML(file) as { jobId: string }
        // Poll for progress
        const poll = setInterval(async () => {
            try {
                const progress = await api.getImportProgress(result.jobId)
                setImportProgress({
                    current: progress.current,
                    total: progress.total,
                    feedName: progress.feedName,
                    success: progress.success,
                    failed: progress.failed,
                })
                if (progress.done) {
                    clearInterval(poll)
                    setImporting(false)
                    setImportProgress(null)
                    setSuccess(`Imported ${progress.success} of ${progress.total} feeds`)
                }
            } catch {
                clearInterval(poll)
                setImporting(false)
                setImportProgress(null)
            }
        }, 200)
    } catch (err: any) {
        setImporting(false)
        setImportProgress(null)
        setError(err.message || 'Failed to import OPML')
    } finally {
        if (fileInputRef.current) fileInputRef.current.value = ''
    }
}
```

- [ ] **Step 4: Update import button UI**

Find the import button (around line 385) and add progress display:

```tsx
<button
    onClick={() => fileInputRef.current?.click()}
    disabled={importing}
    className="btn btn-secondary"
>
    <Upload size={16} />
    {importing ? t('settings.importing') : t('settings.importOPML')}
</button>
{importProgress && (
    <div style={{fontSize: '0.85rem', marginTop: '4px'}}>
        导入 {importProgress.current}/{importProgress.total}
        {importProgress.feedName && `: ${importProgress.feedName}`}
        {importProgress.total > 0 && (
            <> — 成功: {importProgress.success}, 失败: {importProgress.failed}</>
        )}
    </div>
)}
```

- [ ] **Step 5: Verify frontend builds**

```bash
cd /home/dabao/code/ai-reader-flow/frontend && npm run build 2>&1 | tail -10
```

Expected: build succeeds

- [ ] **Step 6: Commit**

```bash
git add frontend/src/api.ts frontend/src/components/Settings.tsx
git commit -m "feat(frontend): add OPML import progress polling UI

- Add getImportProgress API function
- Poll every 200ms while importing
- Show current feed name and success/fail counts

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Verification

After both tasks:
1. Run `go test ./...` to ensure no breakage
2. Rebuild Docker and test importing an OPML file
3. Verify progress shows "Importing 3/10: Feed Name" style messages
