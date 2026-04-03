# 刷新反馈交互优化设计方案

**目标：** 解决"点刷新不知道有没有动"的问题，提供清晰的操作反馈。

## 问题根因

- 单个订阅源刷新：点击后无任何 UI 变化，用户不知道操作是否生效
- 全部刷新：modal 在 SSE 完成后 1500ms 即消失，进度条来不及看清

## 设计方案

### 反馈机制分层

**Level 1 — 单个刷新按钮**
- 点击后 button 进入 `disabled` 状态
- `RefreshCw` 图标添加 `.spinning` 类，图标旋转
- tooltip 变为"刷新中..."
- 刷新完成后恢复

**Level 2 — 全部刷新**
- 触发时 button 进入 `disabled` + `.spinning` 状态
- masthead 正下方渲染 slim 进度条（fixed 定位，高度 3px）
- 文字实时显示当前 feed 名和进度：`正在刷新 量子位 (3/8)`
- SSE 推送 `refresh:progress` 时更新宽度和文字
- 刷新完成后：进度条保持 100% 约 800ms，然后 opacity 1→0 淡出（300ms）
- 失败时：进度条变红（danger 色），800ms 后消失

**Level 3 — 状态统一**
- 任意一个 feed 正在刷新时 → 顶部进度条显示
- 不区分"单个刷新"和"全部刷新"，统一用进度条反馈

## 文件改动

### `frontend/src/components/FeedList.tsx`

**State 变更：**
- 新增 `refreshingFeedIds: Set<number>` 追踪哪些 feed 正在刷新
- 复用现有 `progressModal` 改为 slim inline bar（不再用 Ant Modal popup）

**顶部进度条渲染（mashhead 正下方）：**
```tsx
{isAnyRefreshing && (
  <div className="refresh-progress-bar">
    <div className="refresh-progress-info">
      {refreshingMessage}
    </div>
    <div className="refresh-progress-track">
      <div className="refresh-progress-fill" style={{width: `${refreshingPercent}%`}} />
    </div>
  </div>
)}
```

**单个刷新 handler 变更（handleRefreshOneFeed）：**
- 调用前：`setRefreshingFeedIds(prev => new Set([...prev, feedId]))`
- 完成后/失败：移除对应 id

**全部刷新 handler 变更（handleRefreshAll）：**
- 调用前：`setRefreshing(true)`
- SSE `refresh:progress`：更新 `refreshingMessage` 和 `refreshingPercent`
- SSE `refresh:complete/error`：延迟 800ms 后 setRefreshing(false)

**CSS（内联或 style.css）：**
```css
.refresh-progress-bar {
  position: fixed;
  top: var(--masthead-height); /* mashhead 正下方 */
  left: 0; right: 0;
  z-index: 100;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  padding: 6px 16px;
  transition: opacity 0.3s ease;
}
.refresh-progress-info {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 4px;
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
```

**按钮状态：**
- 单个刷新：`disabled={refreshingFeedIds.has(feed.id)}`
- 全部刷新：`disabled={refreshing}`

### SSE 事件监听（不变）

`refresh:start` → `refresh:progress` → `refresh:complete/error` 推送链路保持不变，只需在对应 handler 中更新前端状态。

## 实现顺序

1. State 和进度条 UI 骨架（无 SSE 连接）
2. 单个刷新按钮的 disabled + spinning
3. 全部刷新的 SSE 状态联动
4. 进度条淡出动画
5. 失败状态（红色进度条）
