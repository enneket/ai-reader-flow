# 刷新反馈交互优化实现计划

**目标：** 提供清晰可见的刷新进度反馈——顶部 slim 进度条 + 单个按钮 spinner 状态。

**架构：** SSE 推送 `refresh:start/progress/complete/error` 事件，前端监听并更新 slim progress bar UI；单个刷新按钮加 disabled + spinning 状态。

**Tech Stack:** React + TypeScript, CSS (inline + style.css)

---

## 文件映射

**Modify:** `frontend/src/components/FeedList.tsx`
- 主要改动文件，state 和 UI 都在这里

**Modify:** `frontend/src/style.css`
- 添加 `.refresh-progress-*` 样式

---

## Task 1: 单个刷新按钮的 disabled + spinning 状态

**Files:** `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: 新增 `refreshingFeedIds` state（找到现有 state 声明区域，在 `refreshing` 附近添加）**

在 `useState` 区域添加：
```tsx
const [refreshingFeedIds, setRefreshingFeedIds] = useState<Set<number>>(new Set())
const [refreshingMessage, setRefreshingMessage] = useState('')
const [refreshingPercent, setRefreshingPercent] = useState(0)
```

- [ ] **Step 2: 修改 `handleRefreshOneFeed`（约 line 172）**

在函数开头添加：
```tsx
setRefreshingFeedIds(prev => new Set([...prev, feedId]))
```

在 try 块末尾（刷新完成后）添加：
```tsx
} finally {
    setRefreshingFeedIds(prev => {
        const next = new Set(prev)
        next.delete(feedId)
        return next
    })
}
```

在 catch 块也添加同样的 finally 逻辑。

- [ ] **Step 3: 单个刷新按钮加 disabled 属性（约 line 469-476）**

在 `<button onClick={(e) => handleRefreshOneFeed(feed.id, e)}` 上加：
```tsx
disabled={refreshingFeedIds.has(feed.id)}
```

- [ ] **Step 4: RefreshCw 图标加 spinner class（约 line 475）**

```tsx
<RefreshCw size={11} className={refreshingFeedIds.has(feed.id) ? 'spinning' : ''} />
```

- [ ] **Step 5: 验证 build**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): add per-feed refresh disabled+spinner state"
```

---

## Task 2: Slim 进度条 UI 组件

**Files:** `frontend/src/style.css`, `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: 添加 CSS 样式到 style.css 末尾**

```css
.refresh-progress-bar {
  position: fixed;
  top: var(--masthead-height);
  left: 0;
  right: 0;
  z-index: 100;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  padding: 8px 16px 6px;
  transition: opacity 0.3s ease;
}
.refresh-progress-info {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 5px;
}
.refresh-progress-track {
  height: 3px;
  background: var(--bg-primary);
  border-radius: 2px;
  overflow: hidden;
}
.refresh-progress-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s ease;
}
.refresh-progress-bar.error .refresh-progress-fill {
  background: var(--danger);
}
.refresh-progress-bar.complete {
  opacity: 0;
  pointer-events: none;
}
```

- [ ] **Step 2: 在 FeedList 顶部渲染进度条（在 `<header className="masthead">` 后面）**

在 `app-body` div 之前添加：
```tsx
{(refreshingFeedIds.size > 0 || refreshing) && (
  <div className={`refresh-progress-bar ${refreshingFeedIds.size === 0 && !refreshing ? 'complete' : ''} ${/* error state added in task 4 */''}`}>
    <div className="refresh-progress-info">{refreshingMessage}</div>
    <div className="refresh-progress-track">
      <div className="refresh-progress-fill" style={{width: `${refreshingPercent}%`}} />
    </div>
  </div>
)}
```

- [ ] **Step 3: 验证 build**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/style.css frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): add slim refresh progress bar UI"
```

---

## Task 3: SSE 事件监听驱动进度条

**Files:** `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: 在 FeedList 中找到现有的 SSE EventSource 逻辑（应该在文件后半部分，或参考 Briefing.tsx 的实现）**

如果 FeedList 里没有 SSE 监听，则添加（参考 Briefing.tsx 的 useEffect）：
```tsx
useEffect(() => {
  const es = new EventSource('/api/events')

  es.addEventListener('refresh:start', (e) => {
    const data = JSON.parse(e.data)
    setRefreshingMessage(`开始刷新 ${data.total || 0} 个订阅源...`)
    setRefreshingPercent(0)
  })

  es.addEventListener('refresh:progress', (e) => {
    const data = JSON.parse(e.data)
    const completed = data.current
    const total = data.total
    const percent = total > 0 ? Math.round((completed / total) * 100) : 0
    setRefreshingMessage(`正在刷新 ${data.feedTitle || ''} (${completed}/${total})`)
    setRefreshingPercent(percent)
  })

  es.addEventListener('refresh:complete', () => {
    setRefreshingMessage('刷新完成')
    setRefreshingPercent(100)
    setTimeout(() => {
      setRefreshingFeedIds(new Set())
      setRefreshing(false)
      setRefreshingPercent(0)
      setRefreshingMessage('')
    }, 800)
  })

  es.addEventListener('refresh:error', (e) => {
    const data = JSON.parse(e.data)
    setRefreshingMessage(data.message || '刷新失败')
    setRefreshingPercent(0)
    setTimeout(() => {
      setRefreshingFeedIds(new Set())
      setRefreshing(false)
      setRefreshingPercent(0)
      setRefreshingMessage('')
    }, 800)
  })

  return () => es.close()
}, [])
```

- [ ] **Step 2: 验证 build**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): wire SSE events to refresh progress bar"
```

---

## Task 4: 移除旧的 polling Modal 逻辑

**Files:** `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: 找到并删除旧的 polling useEffect（lines 60-115）和 progressModal 相关 Modal**

删除整个 `useEffect` polling 块（lines 60-115）：
```tsx
// 删除这段 ↓
// Polling for refresh progress - only active when refreshing
useEffect(() => {
  if (!refreshing) return
  const pollInterval = setInterval(async () => {
    ...
  }, 1000)
  return () => clearInterval(pollInterval)
}, [refreshing, selectedFeed])
```

删除 Modal 组件（大约在 lines 295-315）：
```tsx
<Modal
  open={progressModal.open}
  ...
/>
```

删除相关 state：`progressModal`（如果只用于 polling modal）

注意：保留 `setRefreshing(false)` 等基础状态管理逻辑，只删掉 Modal 渲染和 polling interval。

- [ ] **Step 2: 验证 build**

```bash
cd frontend && npm run build 2>&1 | tail -15
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): remove old polling modal, use SSE-driven progress bar"
```

---

## Task 5: E2E 验证

- [ ] **Step 1: Rebuild and restart web**

```bash
docker compose build --no-cache web && docker compose up -d web
```

- [ ] **Step 2: 用 browse 验证**

```bash
~/.claude/skills/gstack/browse/dist/browse goto http://localhost:5561/feeds
# 点击单个刷新按钮 → 验证按钮 disabled + spinner
# 点击全部刷新 → 验证顶部进度条出现 + 实时更新
```

- [ ] **Step 3: 截图留存**

```bash
~/.claire/skills/gstack/browse/dist/browse screenshot /tmp/refresh-feedback.png
```
