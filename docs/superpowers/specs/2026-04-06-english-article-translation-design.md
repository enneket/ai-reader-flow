# English Article Translation Design

## Goal

For English articles: auto-translate to Chinese on refresh, then generate summary from Chinese content. Add language toggle button on article reader.

## Architecture

1. **DB**: Add `is_translated` and `translated_content` columns to `articles` table
2. **Translation Pipeline**: On article refresh, detect English → translate → store → summarize
3. **Display Toggle**: Global setting to show original English or translated Chinese
4. **UI**: Language toggle button on ArticleReader

## Database Changes

```sql
ALTER TABLE articles ADD COLUMN is_translated INTEGER DEFAULT 0;
ALTER TABLE articles ADD COLUMN translated_content TEXT;
```

## API Changes

### Translation Trigger (on article refresh)

```
fetch article content
  → isEnglish(content)?
    → YES: translate via AI, store in translated_content, set is_translated=1
    → NO: skip translation
  → generate summary from displayed content (translated if available)
```

### Language Detection

```go
func isEnglish(content string) bool {
    if len(content) < 100 {
        return false
    }
    asciiCount := 0
    for _, r := range content {
        if r < 128 {
            asciiCount++
        }
    }
    return float64(asciiCount)/float64(len(content)) > 0.5
}
```

### New Endpoints

- `GET /api/settings/show_original` — returns boolean
- `PUT /api/settings/show_original` — updates boolean

### Article Response

Include `is_translated` and `translated_content` in article JSON response.

## Frontend Changes

### ArticleReader

Add language toggle button (top-right corner):
- Default view: show translated Chinese
- Button shows "EN" → click to switch to English
- When showing English: button shows "中" → click to switch back

### Settings Page

Add toggle:
- "英文文章显示原文" (show_original_language) — default: false (show Chinese)

### State Management

- Global setting `show_original_language` stored in backend
- Per-article `is_translated`, `translated_content` stored in DB
- ArticleReader reads `translated_content` if available and `show_original_language=false`

## Flow

```
Article Refresh:
1. Fetch from RSS
2. Detect language (isEnglish)
3. If English:
   a. Call AI translate (translation prompt)
   b. Store translated_content
   c. Set is_translated=1
4. Generate summary from displayed content (Chinese if translated)

Article View:
1. Check is_translated
2. Check show_original_language setting
3. Display translated Chinese OR original English
```

## Fallback

If translation fails → show original English, is_translated stays 0.
