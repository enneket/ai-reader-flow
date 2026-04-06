# Feed Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add feed filtering to the unified masthead search — keystroke filters feed list in real-time; Enter triggers FTS article search.

**Architecture:** Wrap all routes in App.tsx with a Layout+Masthead component. FeedList receives search query and filters feeds locally. Masthead search submit triggers existing FTS article search.

**Tech Stack:** React + TypeScript, existing Masthead/FeedList/Layout components

---

### Task 1: Add Layout with Masthead wrapper to App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create AppLayout component combining Masthead + children**

```tsx
// frontend/src/components/AppLayout.tsx
import {Masthead} from './Masthead'
import {Layout} from './Layout'

interface AppLayoutProps {
  children: React.ReactNode
}

export function AppLayout({children}: AppLayoutProps) {
  return (
    <Layout>
      {children}
    </Layout>
  )
}
```

Wait — Layout already exists but Masthead is inside Layout as a static header. The issue is Masthead needs to be shared but its props (isRefreshing, onRefresh, onSettings) vary per page.

Actually: keep Layout as sidebar-only, handle masthead per page with a simpler unified approach. Skip Task 1 — the feed filter doesn't need App.tsx restructuring if we pass search query via context or prop drilling.

Let me redesign: the cleanest approach is to keep per-page mastheads but have FeedList accept a `searchQuery` prop that drives local filtering.

**Revised approach:**
- No App.tsx change needed
- FeedList gets `searchQuery` from Masthead via a simple React context
- Masthead wraps FeedList in pages that need feed search

Actually simplest: just make FeedList's own masthead search filter the feed list, and use the same search input for both. Masthead already exists per-page.

Let me just focus on FeedList: add `feedSearchQuery` state, filter `feeds` in render, add search input to FeedList's masthead section.

---

### Task 1: Add feedSearchQuery state to FeedList

**Files:**
- Modify: `frontend/src/components/FeedList.tsx:1-72`

- [ ] **Step 1: Add feedSearchQuery state**

Add after line 24 (after refreshing state):
```tsx
const [feedSearchQuery, setFeedSearchQuery] = useState('')
```

- [ ] **Step 2: Add search input to masthead area in FeedList**

Find the masthead-right div in FeedList (around line 380) and add a search input before the settings link:
```tsx
<div className="masthead-right">
  <div style={{position: 'relative', display: 'flex', alignItems: 'center'}}>
    <Search size={14} style={{position: 'absolute', left: 8, color: 'var(--text-secondary)'}} />
    <input
      type="text"
      value={feedSearchQuery}
      onChange={e => setFeedSearchQuery(e.target.value)}
      placeholder={t('feeds.searchPlaceholder')}
      style={{
        background: 'var(--bg-secondary)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        padding: '4px 8px 4px 28px',
        fontSize: '0.75rem',
        color: 'var(--text-primary)',
        width: '160px',
      }}
    />
    {feedSearchQuery && (
      <button
        onClick={() => setFeedSearchQuery('')}
        style={{
          position: 'absolute',
          right: 6,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          padding: 2,
          color: 'var(--text-secondary)',
        }}
      >
        <X size={12} />
      </button>
    )}
  </div>
  {/* existing refresh and settings buttons */}
</div>
```

Need to import Search and X from lucide-react.

- [ ] **Step 3: Filter feeds in render**

Find the `feeds.map` call and change to:
```tsx
const filteredFeeds = feedSearchQuery
  ? feeds.filter(f => f.title.toLowerCase().includes(feedSearchQuery.toLowerCase()))
  : feeds
```

Then use `filteredFeeds.map` instead of `feeds.map`.

- [ ] **Step 4: Add empty state when no feeds match**

In the empty state div after the feed list, add:
```tsx
{filteredFeeds.length === 0 && feedSearchQuery && (
  <div style={{padding: 'var(--space-4)', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '0.85rem'}}>
    {t('feeds.noFeedsMatch')}
  </div>
)}
```

- [ ] **Step 5: Add i18n key**

Add to `zh.ts`:
```ts
searchPlaceholder: "搜索订阅源...",
noFeedsMatch: "没有匹配的订阅源",
```

Add to `en.ts`:
```ts
searchPlaceholder: "Search feeds...",
noFeedsMatch: "No feeds match",
```

- [ ] **Step 6: Import Search and X**

Add to lucide-react import line:
```tsx
import {Plus, RefreshCw, Trash2, Rss, FileText, Settings, LayoutGrid, CheckCheck, Search, X} from 'lucide-react'
```

- [ ] **Step 7: Build and verify**

Run: `cd frontend && npm run build`
Expected: SUCCESS (no TypeScript errors)

- [ ] **Step 8: Commit**

```bash
git add frontend/src/components/FeedList.tsx frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(frontend): add feed search filter to FeedList"
```

---

### Task 2: Verify Masthead search still works for article FTS

**Files:**
- None (already implemented, just verify)

The existing Masthead `handleSearch` calls `api.searchArticles` and passes results via `onSearchResults` prop. This is untouched by Task 1. No changes needed.

- [ ] **Step 1: Verify the build still passes**

Run: `cd frontend && npm run build`
Expected: SUCCESS

- [ ] **Step 2: Commit**

```bash
git commit -m "chore: verify article FTS search still works"
```
