# Dead Feed Toast Notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show an elegant toast notification when a feed refresh fails with 404/410, informing the user the feed is dead without interrupting workflow.

**Architecture:** Use existing AppModal component as toast notification in FeedList.tsx. Detect 404/410 errors in handleRefreshOneFeed and trigger the modal. No backend changes needed.

**Tech Stack:** React, TypeScript, existing AppModal component, i18n

---

## Task 1: Add dead feed alert state and detection

**Files:**
- Modify: `frontend/src/components/FeedList.tsx` — add state and error detection
- Modify: `frontend/src/i18n/zh.ts` — add i18n key
- Modify: `frontend/src/i18n/en.ts` — add i18n key

- [ ] **Step 1: Add state for dead feed alert**

In `FeedList.tsx`, add after the existing state declarations (around line 29):

```typescript
const [deadFeedAlert, setDeadFeedAlert] = useState<{
  open: boolean
  feedName: string
  feedId: number
} | null>(null)
```

- [ ] **Step 2: Update handleRefreshOneFeed to detect dead feeds**

In `handleRefreshOneFeed` catch block (around line 191-202), add detection for 404/410:

```typescript
} catch (err: any) {
  const isDead = err.message?.includes('404') ||
                 err.message?.includes('410') ||
                 err.message?.includes('not found') ||
                 err.message?.includes('dead')
  if (isDead) {
    setDeadFeedAlert({
      open: true,
      feedName: feeds.find(f => f.id === feedId)?.title || 'Unknown',
      feedId
    })
    // Reload feeds to update is-dead styling
    loadFeeds()
  } else {
    setError(err.message || '刷新失败')
  }
  // ... rest of existing finally block
}
```

- [ ] **Step 3: Add AppModal for dead feed alert**

Add after the existing `conflictModalOpen` AppModal (around line 572):

```tsx
{deadFeedAlert?.open && (
  <AppModal
    type="warning"
    title={t('feeds.deadFeedTitle')}
    content={t('feeds.deadFeedMessage', { name: deadFeedAlert.feedName })}
    autoClose={5000}
    onOk={() => setDeadFeedAlert(null)}
  />
)}
```

- [ ] **Step 4: Add i18n keys**

In `zh.ts`:
```typescript
deadFeedTitle: "订阅源已失效",
deadFeedMessage: "订阅源 "{{name}}" 无法访问（404/410），已标记为失效，不再自动刷新。",
```

In `en.ts`:
```typescript
deadFeedTitle: "Feed Unavailable",
deadFeedMessage: "Feed "{{name}}" is no longer accessible (404/410) and has been marked as dead.",
```

- [ ] **Step 5: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/FeedList.tsx frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(frontend): add dead feed toast notification on 404/410"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `frontend/src/components/FeedList.tsx` | Add deadFeedAlert state, detect 404/410 in catch block, render AppModal |
| `frontend/src/i18n/zh.ts` | Add deadFeedTitle and deadFeedMessage |
| `frontend/src/i18n/en.ts` | Add deadFeedTitle and deadFeedMessage |
