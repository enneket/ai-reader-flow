# Briefing Multi-Time Schedule Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 支持配置多个简报执行时间（从单一时间点扩展为时间列表），通过 Settings UI 管理，动态生效无需重启。

**Architecture:** 单一 cron 每分钟触发，handler 内检查当前时间是否在 `config.toml` 的 `Times` 列表中。新增 `GET/PUT /api/cron-times` API。

---

## File Map

### Backend
- `internal/config/config.go` — `CronConfig` 改 `Times []string`，废弃 `Hour`/`Minute`/`IntervalMins`
- `cmd/server/main.go` — cron 改 `@hourly`（每分钟），handler 内做时间匹配；新增 API handlers

### Frontend
- `frontend/src/api.ts` — 新增 `getCronTimes`, `setCronTimes`
- `frontend/src/components/Settings.tsx` — 新增简报定时区块 UI

---

## Task 1: Update CronConfig struct

**Files:**
- Modify: `internal/config/config.go:31-36`

- [ ] **Step 1: Read current CronConfig**

```go
type CronConfig struct {
    Enabled      bool `toml:"enabled"`
    IntervalMins int  `toml:"interval_mins"`
    Hour        int  `toml:"hour"`
    Minute      int  `toml:"minute"`
}
```

- [ ] **Step 2: Replace CronConfig struct**

```go
type CronConfig struct {
    Enabled bool     `toml:"enabled"`
    Times   []string `toml:"times"` // e.g. ["09:00", "18:00", "21:00"]
}
```

- [ ] **Step 3: Update default values in LoadConfig()**

```go
Cron: CronConfig{
    Enabled: true,
    Times:   []string{"09:00"},
},
```

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor: CronConfig.Times []string replacing Hour/Minute/IntervalMins"
```

---

## Task 2: Update cron handler to use time list

**Files:**
- Modify: `cmd/server/main.go:191-212`

- [ ] **Step 1: Read current cron block**

```go
schedule := fmt.Sprintf("%d %d * * *", cfg.Cron.Minute, cfg.Cron.Hour)
c.AddFunc(schedule, func() {
    log.Printf("[cron] Daily briefing at %02d:%02d", cfg.Cron.Hour, cfg.Cron.Minute)
    ...
})
```

- [ ] **Step 2: Replace with minute-level cron**

```go
c.AddFunc("@hourly", func() {
    now := time.Now()
    t := now.Format("15:04") // "HH:MM" in local timezone
    // Check if current time is in the Times list
    matched := false
    for _, tm := range cfg.Cron.Times {
        if tm == t {
            matched = true
            break
        }
    }
    if !matched {
        return // Not a scheduled time, skip silently
    }
    log.Printf("[cron] Briefing trigger at %s - refreshing feeds first", t)
    ...
})
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: cron checks Times list instead of single time"
```

---

## Task 3: Add cron-times API handlers

**Files:**
- Modify: `cmd/server/main.go` — add route and two handlers

- [ ] **Step 1: Add route after briefing routes**

```go
mux.HandleFunc("GET /api/cron-times", handleGetCronTimes)
mux.HandleFunc("PUT /api/cron-times", handleSetCronTimes)
```

- [ ] **Step 2: Add handlers before Briefing handlers section (~line 667)**

```go
func handleGetCronTimes(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, cfg.Cron.Times)
}

func handleSetCronTimes(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Times []string `json:"times"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    // Validate HH:MM format
    for _, t := range req.Times {
        if len(t) != 5 || t[2] != ':' {
            http.Error(w, "invalid time format: "+t, http.StatusBadRequest)
            return
        }
    }
    cfg.Cron.Times = req.Times
    writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add GET/PUT /api/cron-times handlers"
```

---

## Task 4: Add frontend API functions

**Files:**
- Modify: `frontend/src/api.ts`

- [ ] **Step 1: Find location near briefing API calls (line ~173)**

Add after `deleteAllBriefings`:

```typescript
getCronTimes: () => request<string[]>('/cron-times'),

setCronTimes: (times: string[]) =>
    request<{success: boolean}>('/cron-times', {
        method: 'PUT',
        body: JSON.stringify({times}),
    }),
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api.ts
git commit -m "feat: add getCronTimes and setCronTimes to frontend API"
```

---

## Task 5: Add Settings UI for cron times

**Files:**
- Modify: `frontend/src/components/Settings.tsx`

- [ ] **Step 1: Add state after other state declarations (~line 23)**

```typescript
const [cronTimes, setCronTimes] = useState<string[]>([])
const [cronInput, setCronInput] = useState('')
const [cronSaving, setCronSaving] = useState(false)
```

- [ ] **Step 2: Add loadCronTimes in useEffect (~line 88)**

```typescript
api.getCronTimes().then(times => setCronTimes(times)).catch(console.error)
```

- [ ] **Step 3: Add save handler**

```typescript
const handleSaveCronTimes = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!cronInput.trim()) return
    const newTime = cronInput.trim()
    if (cronTimes.includes(newTime)) {
        setModal({type: 'warning', title: '重复', content: '该时间已存在'})
        return
    }
    setCronSaving(true)
    try {
        await api.setCronTimes([...cronTimes, newTime])
        setCronTimes([...cronTimes, newTime])
        setCronInput('')
    } catch (err: any) {
        setModal({type: 'error', title: '错误', content: err.message})
    } finally {
        setCronSaving(false)
    }
}
```

- [ ] **Step 4: Add remove handler**

```typescript
const handleRemoveCronTime = async (time: string) => {
    try {
        await api.setCronTimes(cronTimes.filter(t => t !== time))
        setCronTimes(cronTimes.filter(t => t !== time))
    } catch (err: any) {
        setModal({type: 'error', title: '错误', content: err.message})
    }
}
```

- [ ] **Step 5: Add UI section after AI Config section (~line 470, before notes section)**

```tsx
<section className="settings-section">
  <h3>简报定时</h3>
  <p style={{fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: 'var(--space-3)'}}>
    配置简报生成时间（北京时间），可添加多个时间点。
  </p>
  <div className="cron-times-list">
    {cronTimes.length === 0 ? (
      <p style={{color: 'var(--text-secondary)', fontSize: '0.85rem'}}>暂无定时</p>
    ) : (
      cronTimes.sort().map(time => (
        <div key={time} className="cron-time-item">
          <span>{time}</span>
          <button
            onClick={() => handleRemoveCronTime(time)}
            className="btn btn-ghost btn-sm"
            style={{padding: '4px', color: 'var(--danger)'}}
          >
            ×
          </button>
        </div>
      ))
    )}
  </div>
  <form onSubmit={handleSaveCronTimes} style={{display: 'flex', gap: 'var(--space-2)', marginTop: 'var(--space-2)'}}>
    <input
      type="text"
      value={cronInput}
      onChange={e => setCronInput(e.target.value)}
      placeholder="HH:MM 如 09:00"
      pattern="[0-2][0-9]:[0-5][0-9]"
      className="form-input"
      style={{width: '120px'}}
    />
    <button type="submit" disabled={cronSaving} className="btn btn-secondary btn-sm">
      {cronSaving ? '...' : '添加'}
    </button>
  </form>
</section>
```

- [ ] **Step 6: Add CSS for cron-time-item in style.css**

```css
.cron-times-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
.cron-time-item {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  background: var(--bg-secondary);
  border-radius: var(--radius);
  font-size: 0.9rem;
}
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/Settings.tsx frontend/src/style.css
git commit -m "feat: add cron times UI to Settings page"
```

---

## Task 6: Build and verify

- [ ] **Step 1: Build backend**

```bash
cd /home/zjx/code/mine/ai-reader-flow
go build ./...
```

- [ ] **Step 2: Build frontend**

```bash
cd frontend && npm run build 2>&1 | tail -5
```

- [ ] **Step 3: Deploy**

```bash
docker compose build app && docker compose up -d app
```

- [ ] **Step 4: Test API**

```bash
curl http://localhost:5561/api/cron-times
# Expected: ["09:00"]

curl -X PUT http://localhost:5561/api/cron-times \
  -H "Content-Type: application/json" \
  -d '{"times":["09:00","18:00","21:00"]}'
# Expected: {"success":true}

curl http://localhost:5561/api/cron-times
# Expected: ["09:00","18:00","21:00"]
```

---

## Verification Checklist

- [ ] `GET /api/cron-times` 返回当前时间列表
- [ ] `PUT /api/cron-times` 正确更新配置文件
- [ ] 格式校验拒绝非 `HH:MM` 格式
- [ ] Settings UI 显示时间列表
- [ ] 添加新时间后列表更新
- [ ] 删除时间后列表更新
- [ ] 重复时间被拒绝（modal 提示）
