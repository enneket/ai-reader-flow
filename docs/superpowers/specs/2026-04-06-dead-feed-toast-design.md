# Dead Feed Toast Notification Design

## Goal

When a feed refresh fails with 404/410, show an elegant toast notification instead of just logging to console. The notification informs the user the feed is dead without interrupting workflow.

## Design

### Notification Behavior

- Triggered when `handleRefreshOneFeed` detects a 404/410 error
- Shows `AppModal` warning type with auto-close (5 seconds)
- Content: "订阅源 [名称] 已失效，不再自动刷新"
- User can click OK or wait for auto-close
- Dead feed in list shows `is-dead` CSS class (gray + strikethrough)

### State

```typescript
const [deadFeedAlert, setDeadFeedAlert] = useState<{
  open: boolean
  feedName: string
  feedId: number
} | null>(null)
```

### i18n

zh: `"订阅源 "{{name}}" 无法访问（404/410），已标记为失效，不再自动刷新。"`
en: `Feed "{{name}}" is no longer accessible (404/410) and has been marked as dead.`

### Implementation

In `FeedList.tsx`:
1. Add `deadFeedAlert` state
2. In `handleRefreshOneFeed` catch block: detect 404/410, set state
3. Render `AppModal` when `deadFeedAlert?.open`

No backend changes needed.
