# AppModal — 自定义主题化模态框

## Status

Approved for implementation.

---

## 背景

项目使用 Ant Design v6 组件库，但自定义了暖棕色杂志风格主题（CSS 变量：背景 `#2C2416`、强调色 `#D97706` 等）。`antd Modal.warning/error` 使用白色背景和 Ant Design 默认样式，与整体 UI 风格不符。需要用自定义 `AppModal` 组件统一替换所有 Modal。

---

## 设计决策

### 范围

统一替换项目中所有 Modal：
- `FeedList.tsx`：操作冲突（409）
- `Briefing.tsx`：操作冲突、刷新失败、生成失败

### 实现方式

完全自定义 React 组件，不使用 `antd Modal`，通过 `ReactDOM.createPortal` 渲染到 `body`。

### 组件设计

**文件名**：`src/components/AppModal.tsx`

**Props 接口**：
```ts
interface AppModalProps {
  type: 'warning' | 'error'
  title: string
  content: string
  onOk: () => void
}
```

**样式映射**：

| type     | 图标 | 背景色                | 边框色     |
|----------|------|----------------------|------------|
| `warning`| ⏳   | `rgba(217,119,6,0.12)` | `#D97706` |
| `error`  | 🔴   | `rgba(220,38,38,0.12)` | `#DC2626` |

**通用样式**（继承 CSS 变量）：
- 弹框背景：`--surface (#3D3226)`
- 标题文字：`--text-primary (#F5EFE6)`，16px，font-weight 600
- 内容文字：`--text-secondary (#C4B89A)`，14px，line-height 1.5
- 弹框圆角：`--radius (6px)`，8px 宽
- 遮罩背景：`rgba(0,0,0,0.6)`
- 弹框阴影：`0 20px 60px rgba(0,0,0,0.5)`

**确定按钮**：
- 背景：`--accent (#D97706)`
- hover 背景：`--accent-hover (#F59E0B)`
- 文字：白色，14px，font-weight 500
- 圆角：`--radius (6px)`
- padding：8px 20px

### 交互行为

1. 点击遮罩层 → 关闭
2. 点击确定按钮 → 关闭（调用 `onOk`）
3. 按 ESC 键 → 关闭
4. 键盘 Tab 聚焦陷阱（Tab 键在弹框内循环，不跳到外部）

### 文件改动

| 文件 | 改动 |
|------|------|
| `src/components/AppModal.tsx` | **新建** — 模态框组件 |
| `src/components/FeedList.tsx` | `Modal.warning` → `AppModal` |
| `src/components/Briefing.tsx` | `Modal.warning` / `Modal.error` → `AppModal` |

---

## 验收标准

1. `FeedList` 点击刷新后立即再点，弹出「操作冲突」主题化弹框（暖棕色）
2. `Briefing` 页面各错误场景均弹出主题化弹框（error 类型红色边框）
3. ESC 键和遮罩点击均能关闭弹框
4. antd `Modal` 相关 import 可从 `FeedList.tsx` 和 `Briefing.tsx` 中移除（如果仅用于 Modal）
