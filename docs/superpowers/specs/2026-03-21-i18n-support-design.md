# Internationalization (i18n) Support Design

## Overview

Add i18n support to AI RSS Reader with English and Simplified Chinese languages. Language selection is in Settings page, with auto-detection on first load and persistence via localStorage.

## Design

### Language Selection
- Dropdown selector in Settings page with options: English, 中文
- Default: auto-detect from browser language (`navigator.language`)
- If browser is zh-* → default to Chinese, otherwise English
- Selection persisted in `localStorage.setItem('language', 'en' | 'zh')`

### Supported Languages
| Code | Language |
|------|----------|
| `en` | English |
| `zh` | Simplified Chinese |

### Translation File Structure
```
frontend/src/
├── i18n/
│   ├── index.ts          # i18n configuration
│   ├── en.json           # English translations
│   └── zh.json           # Chinese translations
```

### Translated Strings

**Navigation (Layout.tsx)**
- "AI RSS" → "AI RSS"
- "Feeds" → "订阅源"
- "Articles" → "文章"
- "Notes" → "笔记"
- "Settings" → "设置"

**Feeds Page (FeedList.tsx)**
- "RSS Feeds" → "RSS 订阅源"
- "Refresh All" → "刷新全部"
- "Add Feed" → "添加订阅源"
- "Enter RSS feed URL" → "输入 RSS 订阅源地址"
- "View Articles" → "查看文章"
- "Delete" → "删除"
- "No feeds yet. Add your first RSS feed to get started." → "暂无订阅源。添加第一个 RSS 订阅源开始使用。"

**Articles Page (ArticleList.tsx)**
- "Articles" → "文章"
- "All Feeds" → "全部订阅源"
- "All" → "全部"
- "Filtered" → "已过滤"
- "Saved" → "已收藏"
- "Filter with AI" → "AI 筛选"
- "AI Summary" → "AI 摘要"
- "Save as Note" → "保存为笔记"
- "No articles yet. Add a feed first." → "暂无文章。请先添加订阅源。"

**Notes Page (NoteList.tsx)**
- "Notes" → "笔记"
- "Note" / "Notes" → "笔记"
- "No notes yet. Save articles to create notes." → "暂无笔记。收藏文章来创建笔记。"
- "Select a note to view its content" → "选择一个笔记查看内容"
- "Untitled Note" → "无标题笔记"

**Settings Page (Settings.tsx)**
- "Settings" → "设置"
- "AI Provider Configuration" → "AI 服务商配置"
- "Provider" → "服务商"
- "API Key" → "API 密钥"
- "Base URL" → "接口地址"
- "Model" → "模型"
- "Max Tokens" → "最大 Token 数"
- "Save AI Config" → "保存 AI 配置"
- "Filter Rules" → "筛选规则"
- "Type" → "类型"
- "Value" → "值"
- "Action" → "动作"
- "Add Rule" → "添加规则"
- "Language" → "语言"
- "English" → "English"
- "中文" → "中文"

**Common**
- "Loading..." → "加载中..."
- "Error" → "错误"
- "Success" → "成功"
- "Cancel" → "取消"
- "Confirm" → "确认"
- "Save" → "保存"
- "Delete" → "删除"

### Auto-Detection Flow
```
1. App loads
2. Check localStorage for 'language'
3. If exists → use stored value
4. If not → check navigator.language
   - If starts with 'zh' → default 'zh'
   - Otherwise → default 'en'
5. Apply language immediately
```

## Technical Approach

- **Library**: react-i18next
- **Configuration**: Frontend only (no backend changes needed)
- **Persistence**: localStorage key `'language'`
- **Bundle impact**: ~10KB gzipped (i18next + translations)

## Files to Create/Modify

### Create
- `frontend/src/i18n/index.ts`
- `frontend/src/i18n/en.json`
- `frontend/src/i18n/zh.json`

### Modify
- `frontend/src/main.tsx` - wrap with I18nextProvider
- `frontend/src/components/Layout.tsx` - use translations
- `frontend/src/components/FeedList.tsx` - use translations
- `frontend/src/components/ArticleList.tsx` - use translations
- `frontend/src/components/NoteList.tsx` - use translations
- `frontend/src/components/Settings.tsx` - add language selector
