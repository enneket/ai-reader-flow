# 简报功能重构设计

## 概述

将现有的 RSS 阅读器重构为简报生成器。用户每天定时或手动触发，从订阅源抓取文章后，AI 自动识别主题并生成聚合简报。

## 页面结构

| 页面 | 路由 | 说明 |
|------|------|------|
| 订阅源 | `/feeds` | 管理 RSS 订阅 |
| 简报 | `/` | AI 聚合的简报列表 |
| 设置 | `/settings` | AI 配置、每日时间 |

### 导航

- 订阅源: LayoutGrid
- 简报: FileText
- 设置: Settings

---

## 功能流程

### 简报生成

```
1. Cron 定时触发（用户设置时间，如早上 9:00）
   OR
2. 用户点击"立即生成简报"

→ 抓取所有订阅源新文章（自上次刷新后未读的文章）
→ AI 阅读所有文章内容（标题 + 摘要）
→ AI 识别主题（自动聚类）
→ 每个主题提取核心观点
→ 生成简报，存入数据库（带时间戳）
→ 用户看到新的简报卡片
```

**注："新文章"定义**：自上次生成简报后新抓取的文章（通过 `articles.created_at` 判断）

### Cron 定时

- 用户在设置页面配置每日触发时间（如 "09:00"）
- 系统使用 `robfig/cron` 或类似库实现定时调度
- 时区：使用服务器本地时区
- 旧的 `FilterAllArticlesNew` cron 在简报功能上线后停用

---

## 数据模型

### Briefing (简报)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | int64 | 主键 |
| Status | string | 生成状态: `pending`, `generating`, `completed`, `failed` |
| Error | string | 失败原因（如果失败） |
| CreatedAt | time.Time | 创建时间 |
| CompletedAt | time.Time | 完成时间（可选） |

### BriefingItem (简报条目)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | int64 | 主键 |
| BriefingID | int64 | 关联的简报 ID |
| Topic | string | 主题名称，如"AI"、"创业" |
| Summary | string | 该主题的核心观点摘要 |
| SortOrder | int | 排序顺序（按文章数量） |

### BriefingArticle (简报文章引用)

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | int64 | 主键 |
| BriefingItemID | int64 | 关联的简报条目 ID |
| ArticleID | int64 | 原始文章 ID（外键） |
| Title | string | 文章标题（冗余存储，方便显示） |

---

## 简报页面 UI

### 正常状态

```
┌─────────────────────────────────────┐
│ 简报                        09:00 ⟳ │
├─────────────────────────────────────┤
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ AI (3篇)            09:00 AM  │ │
│ │ • GPT-5 发布：核心更新点一览    │ │
│ │ • Claude 3.5 上线：性能突破    │ │
│ │ • AI 编程工具趋势分析           │ │
│ └─────────────────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ 创业 (2篇)          09:15 AM   │ │
│ │ • YC 最新创业方向洞察           │ │
│ │ • 2026 年值得关注的 10 个赛道  │ │
│ └─────────────────────────────────┘ │
│                                     │
└─────────────────────────────────────┘
```

### 空状态

```
┌─────────────────────────────────────┐
│ 简报                        09:00 ⟳ │
├─────────────────────────────────────┤
│                                     │
│   暂无简报                             │
│   点击下方按钮立即生成                  │
│                                     │
│   [立即生成简报]                       │
│                                     │
└─────────────────────────────────────┘
```

---

## API 设计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/briefings` | 获取简报列表（支持 `?limit=20&offset=0`） |
| GET | `/api/briefings/{id}` | 获取简报详情（含条目和文章） |
| POST | `/api/briefings/generate` | 手动触发生成简报 |
| GET | `/api/briefings/{id}/status` | 获取生成状态（SSE 或轮询） |
| DELETE | `/api/briefings/{id}` | 删除某个简报 |
| GET | `/api/settings/schedule` | 获取每日计划时间 |
| PUT | `/api/settings/schedule` | 设置每日计划时间（cron 表达式或时间） |

### 简报生成流程

```
POST /api/briefings/generate
  → 创建 Briefing(status=pending)
  → 后台 goroutine 执生成
  → SSE 广播进度 或 前端轮询 /briefings/{id}/status
  → 完成时 status=completed
```

### GET /api/briefings/{id} 响应

```json
{
  "id": 1,
  "status": "completed",
  "created_at": "2026-03-29T09:00:00Z",
  "items": [
    {
      "id": 1,
      "topic": "AI",
      "summary": "核心观点1\n核心观点2",
      "articles": [
        {"id": 101, "title": "GPT-5 发布"},
        {"id": 102, "title": "Claude 3.5 上线"}
      ]
    }
  ]
}
```

---

## AI 简报生成 Prompt

### 输入格式

用户输入给 AI 的文章列表格式：

```
文章 ID: 101
标题: GPT-5 发布：核心更新点一览
摘要: GPT-5 是 OpenAI 最新一代大语言模型...

---
文章 ID: 102
标题: Claude 3.5 上线：性能突破
摘要: Anthropic 发布 Claude 3.5...
---
```

### Prompt

```
System: 你是一个内容策划助手。给定一组文章，你需要：
1. 将文章按主题分组（相似内容的文章分到同一组）
2. 为每个主题起一个简短的名字（如"AI"、"创业"、"科技"）
3. 为每个主题提取核心观点（用简洁的 bullets，每条不超过 20 字）

输出格式（严格按 JSON 格式，不要有其他内容）：
{
  "topics": [
    {
      "name": "主题名称",
      "article_ids": [101, 102],
      "summary": "• 核心观点1\n• 核心观点2\n• 核心观点3"
    }
  ]
}

规则：
- 每个简报最多 5 个主题
- 每个主题最多 5 篇核心文章
- 只包含真正有价值的文章，无关内容请忽略
- 主题按文章数量排序（多的在前）
- 如果文章太少或无价值，返回空的 topics 数组

User: 以下是今天的文章：
{articles}
```

---

## 数据库迁移

需要新增表：

```sql
CREATE TABLE briefings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE briefing_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    briefing_id INTEGER NOT NULL,
    topic TEXT NOT NULL,
    summary TEXT,
    FOREIGN KEY (briefing_id) REFERENCES briefings(id) ON DELETE CASCADE
);

CREATE TABLE briefing_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    briefing_item_id INTEGER NOT NULL,
    article_id INTEGER NOT NULL,
    title TEXT,
    FOREIGN KEY (briefing_item_id) REFERENCES briefing_items(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);
```

---

## 前端路由

```tsx
<Routes>
  <Route path="/" element={<Briefing />} />
  <Route path="/feeds" element={<FeedList />} />
  <Route path="/settings" element={<Settings />} />
</Routes>
```

---

## 移除的功能

- ArticleList 页面（被简报替代）
- NoteList 页面
- filterMode 过滤逻辑（所有文章都进入简报生成）
- embedding 语义去重
- 质量评分公式

---

## 实施顺序

1. 创建数据库迁移（briefings, briefing_items, briefing_articles 表）
2. 实现 Briefing 数据模型和 Repository
3. 实现简报生成 Service（GenerateBriefing）
4. 实现 API endpoints
5. 创建 Briefing 前端组件
6. 手动测试简报生成功能
7. 添加 Cron 调度（替换旧的 FilterAllArticlesNew cron）
8. 更新路由，移除 ArticleList/NoteList 相关路由
9. 清理旧代码和数据（可选，先保留 articles 表用于历史）
