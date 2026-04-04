# 订阅源失效视觉标记

## Status

Approved for implementation.

---

## 背景

订阅源刷新失败（404/410）时，目前只显示红色 ❌ emoji，视觉提示不够醒目。用户难以快速识别哪些订阅源已失效。

---

## 设计决策

### 方案

订阅源列表项（`.feed-item`）增加左侧红色边条，持续显示直到刷新成功：

```css
.feed-item.is-dead {
  border-left: 3px solid var(--danger);
}
```

### 触发条件

当 `feed.last_refresh_success === -1` 时，该订阅源列表项添加 `is-dead` class。

### 交互

- 刷新按钮保留，用户可尝试重新刷新
- 刷新成功后自动恢复正常（移除 `is-dead` class）
- 不改变其他交互行为

### 文件改动

| 文件 | 改动 |
|------|------|
| `src/style.css` | 添加 `.feed-item.is-dead` 样式 |
| `src/components/FeedList.tsx` | 列表项添加 `is-dead` class 条件 |

---

## 验收标准

1. 失效订阅源列表项左侧有 3px 红色边条
2. 刷新成功后红色边条消失
3. 刷新按钮功能不受影响
