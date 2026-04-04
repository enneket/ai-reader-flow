# 订阅源失效红色边条实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 订阅源刷新失败时列表项左侧显示 3px 红色边条，刷新成功后自动消失。

**Architecture:** 在 `FeedList.tsx` 的 `feed-item` className 条件追加 `is-dead`，CSS 添加对应样式。

**Tech Stack:** React (FeedList.tsx), CSS (style.css)

---

### Task 1: 添加 CSS 样式

**Files:**
- Modify: `frontend/src/style.css`

- [ ] **Step 1: 找到 `.feed-item` 样式位置，在其后添加 dead 样式**

在 `style.css` 第 1296 行 `.feed-item {` 后添加：

```css
/* Dead feed indicator */
.feed-item.is-dead {
  border-left: 3px solid var(--danger);
}
```

---

### Task 2: 条件追加 is-dead class

**Files:**
- Modify: `frontend/src/components/FeedList.tsx:455-459`

- [ ] **Step 1: 修改 className 条件**

当前代码（第 457 行）：
```tsx
className={`feed-item ${selectedFeed?.id === feed.id ? 'selected' : ''}`}
```

改为：
```tsx
className={`feed-item ${selectedFeed?.id === feed.id ? 'selected' : ''} ${feed.last_refresh_success === -1 ? 'is-dead' : ''}`}
```

---

### Task 3: 验证构建

- [ ] **Step 1: 运行前端构建**

Run: `cd frontend && npm run build 2>&1 | tail -5`
Expected: `✓ N modules transformed.` 无错误

---

### Task 4: 提交

- [ ] **Step 1: 提交改动**

```bash
git add frontend/src/style.css frontend/src/components/FeedList.tsx
git commit -m "feat(frontend): add red left border for dead feeds

Red 3px left border on feed items when last_refresh_success === -1,
automatically clears when refresh succeeds.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```
