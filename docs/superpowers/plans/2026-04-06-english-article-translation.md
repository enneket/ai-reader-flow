# English Article Translation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-translate English articles to Chinese on refresh, store translated content in DB, display with language toggle button in ArticleReader, and add a global setting to control default view.

**Architecture:**
- DB: Add `is_translated` (INTEGER) and `translated_content` (TEXT) columns to `articles` via migration
- Backend: Add `isEnglish()` detection and `translateArticle()` method in RSSService; inject translation step into `RefreshArticle()` pipeline after fetching full content
- API: Add `show_original_language` setting via existing key-value settings pattern
- Frontend: Add language toggle button to ArticleReader; add global toggle in Settings page

**Tech Stack:** Go (Chi router), React+TypeScript, SQLite, OpenAI/Claude/Ollama AI providers

---

## Task 1: DB Migration — Add translation columns to articles table

**Files:**
- Modify: `internal/repository/sqlite/db.go` — add migration calls

- [ ] **Step 1: Add migration calls in createTables()**

In `db.go` after the existing migration calls (~line 163), add:

```go
// Migration: add translation columns for English article translation feature
_ = migrateAddColumn("articles", "is_translated", "INTEGER DEFAULT 0")
_ = migrateAddColumn("articles", "translated_content", "TEXT")
```

Run: `go build ./...`
Expected: Compiles without errors

- [ ] **Step 2: Commit**

```bash
git add internal/repository/sqlite/db.go
git commit -m "feat(db): add is_translated and translated_content columns to articles"
```

---

## Task 2: Update Article model with translation fields

**Files:**
- Modify: `internal/models/models.go` — add fields to Article struct

- [ ] **Step 1: Add translation fields to Article struct**

In `models.go`, add to the Article struct (after `CreatedAt time.Time`):

```go
IsTranslated      bool   `json:"is_translated"`
TranslatedContent string `json:"translated_content"`
```

- [ ] **Step 2: Commit**

```bash
git add internal/models/models.go
git commit -m "feat(models): add IsTranslated and TranslatedContent to Article"
```

---

## Task 3: Update ArticleRepository to handle translation fields

**Files:**
- Modify: `internal/repository/sqlite/article_repository.go` — handle new fields in Create, Update, GetByID, scanArticles

- [ ] **Step 1: Update INSERT in Create()**

In `Create()` at line 22-26, change INSERT to include the new columns:

```go
result, err := DB.Exec(
    `INSERT INTO articles (feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    article.FeedID, article.Title, article.Link, article.Content, article.Summary,
    article.Author, article.Published.Format(time.RFC3339), article.IsFiltered, article.IsSaved, status, article.CreatedAt.Format(time.RFC3339), article.IsTranslated, article.TranslatedContent,
)
```

- [ ] **Step 2: Update SELECT in GetByFeedID(), GetAll(), GetByID(), GetByIDs(), GetRecentForBriefing(), GetArticlesAfter(), Search()**

In each SELECT statement, add `is_translated, translated_content` before the FROM or after `created_at`. For example, in `GetByFeedID()` at line 47:

```go
`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles WHERE feed_id = ? ORDER BY published DESC LIMIT ? OFFSET ?`
```

Update all 7 SELECT statements:
- `GetByFeedID()` line 47
- `GetAll()` line 62
- `GetByID()` line 98
- `GetByIDs()` line 132
- `GetRecentForBriefing()` line 199
- `GetArticlesAfter()` line 214
- `Search()` line 239

- [ ] **Step 3: Update scanArticles() to scan new columns**

In `scanArticles()` at line 258, change Scan to:

```go
err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.Link, &a.Content, &a.Summary, &a.Author, &published, &a.IsFiltered, &a.IsSaved, &a.Status, &createdAt, &a.IsTranslated, &a.TranslatedContent)
```

- [ ] **Step 4: Update Update() method to include translation fields**

At line 142-145, change Update() to:

```go
_, err := DB.Exec(
    `UPDATE articles SET title = ?, content = ?, summary = ?, is_filtered = ?, is_saved = ?, status = ?, is_translated = ?, translated_content = ? WHERE id = ?`,
    article.Title, article.Content, article.Summary, article.IsFiltered, article.IsSaved, article.Status, article.IsTranslated, article.TranslatedContent, article.ID,
)
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/sqlite/article_repository.go
git commit -m "feat(repo): handle is_translated and translated_content in ArticleRepository"
```

---

## Task 4: Add translation logic to RSSService

**Files:**
- Modify: `internal/service/rss_service.go` — add isEnglish(), translateArticle(), inject into RefreshArticle()

- [ ] **Step 1: Add imports**

Add `"ai-rss-reader/internal/ai"` to imports if not present.

- [ ] **Step 2: Add isEnglish() language detection function**

Add after the `stripHTML` function (after line 307):

```go
// isEnglish returns true if the content appears to be English based on ASCII ratio.
// Content with >50% ASCII characters is considered English.
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

- [ ] **Step 3: Add TranslateArticle() method**

Add after `RefreshArticle()` (after line 276):

```go
// TranslateArticle translates article content to Chinese if it's English.
// Returns true if translation was performed, false if skipped or failed.
func (s *RSSService) TranslateArticle(article *models.Article) error {
    // Skip if already translated
    if article.IsTranslated && article.TranslatedContent != "" {
        return nil
    }

    // Detect language
    if !isEnglish(article.Content) {
        return nil // Not English, skip translation
    }

    // Get translation prompt from DB
    promptRepo := sqlite.NewPromptRepository()
    promptConfig, err := promptRepo.GetByType("translation")
    if err != nil || promptConfig == nil || promptConfig.Prompt == "" {
        // Fallback: use default prompt
        provider := ai.GetProvider()
        translated, err := provider.GenerateSummaryWithPrompt(
            article.Content,
            "你是一位精通中英文互译的专业翻译官。必须仅输出中文译文，禁止任何额外话语。",
            "将以下英文文章翻译成中文。严格保留原始Markdown格式，专业术语使用业界通用中文表达，语言风格地道通顺。\n\n"+article.Content,
        )
        if err != nil {
            log.Printf("translation failed for article %d: %v", article.ID, err)
            return err
        }
        article.TranslatedContent = translated
        article.IsTranslated = true
    } else {
        // Use configured prompt
        provider := ai.GetProvider()
        translated, err := provider.GenerateSummaryWithPrompt(article.Content, promptConfig.System, promptConfig.Prompt)
        if err != nil {
            log.Printf("translation failed for article %d: %v", article.ID, err)
            return err
        }
        article.TranslatedContent = translated
        article.IsTranslated = true
    }

    // Update article in DB
    if err := s.articleRepo.Update(article); err != nil {
        log.Printf("failed to save translation for article %d: %v", article.ID, err)
        return err
    }
    return nil
}
```

- [ ] **Step 4: Update RefreshArticle() to call TranslateArticle**

After fetching full content and updating article (line 270-275), before returning:

```go
// Translate if English (after fetching full content)
if article.Link != "" {
    if err := s.TranslateArticle(article); err != nil {
        log.Printf("warning: translation failed for article %d: %v", article.ID, err)
        // Don't fail the refresh if translation fails
    }
}
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./...`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add internal/service/rss_service.go
git commit -m "feat(service): add English article translation to RSSService"
```

---

## Task 5: Add show_original_language API endpoints

**Files:**
- Modify: `cmd/server/main.go` — add GET/PUT /api/settings/show_original endpoints
- Modify: `frontend/src/api.ts` — add getShowOriginalLanguage(), setShowOriginalLanguage()
- Modify: `frontend/src/components/Settings.tsx` — add show_original_language toggle

- [ ] **Step 1: Add backend handler functions in main.go**

Add handler functions before the existing handlers. Find an appropriate location (after existing settings handlers) and add:

```go
func handleGetShowOriginalLanguage(w http.ResponseWriter, r *http.Request) {
    settingsRepo := sqlite.NewSettingsRepository()
    value, _ := settingsRepo.Get("show_original_language")
    if value == "" {
        value = "false"
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]bool{"show_original_language": value == "true"})
}

func handleSetShowOriginalLanguage(w http.ResponseWriter, r *http.Request) {
    var req map[string]bool
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", 400)
        return
    }
    settingsRepo := sqlite.NewSettingsRepository()
    val := "false"
    if req["show_original_language"] {
        val = "true"
    }
    settingsRepo.Set("show_original_language", val)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]bool{"show_original_language": val == "true"})
}
```

- [ ] **Step 2: Register routes in main.go**

Add these routes with the other routes (around line with `Route{"GET", "/api/settings/show_original"...}` or similar):

```go
Route{"GET", "/api/settings/show_original", handleGetShowOriginalLanguage},
Route{"PUT", "/api/settings/show_original", handleSetShowOriginalLanguage},
```

- [ ] **Step 3: Add frontend API methods in api.ts**

Add to the `api` object in `frontend/src/api.ts`:

```typescript
getShowOriginalLanguage: () =>
  request<{show_original_language: boolean}>('/settings/show_original'),

setShowOriginalLanguage: (showOriginal: boolean) =>
  request<{show_original_language: boolean}>('/settings/show_original', {
    method: 'PUT',
    body: JSON.stringify({show_original_language: showOriginal}),
  }),
```

- [ ] **Step 4: Add Settings toggle in Settings.tsx**

In the Settings component, add state for `showOriginalLanguage` (after the existing state declarations):

```typescript
const [showOriginalLanguage, setShowOriginalLanguage] = useState(false)
```

Load it in a useEffect (add after `loadAIConfig` call):

```typescript
useEffect(() => {
  loadAIConfig()
  loadPrompts()
  api.getShowOriginalLanguage().then(data => {
    setShowOriginalLanguage(data.show_original_language)
  }).catch(console.error)
}, [])
```

Add a new settings section after the appearance section:

```tsx
<section className="settings-section">
  <h3>文章显示</h3>
  <div className="form-group" style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
    <label style={{display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer'}}>
      <input
        type="checkbox"
        checked={showOriginalLanguage}
        onChange={(e) => {
          const newVal = e.target.checked
          setShowOriginalLanguage(newVal)
          api.setShowOriginalLanguage(newVal).catch(err => {
            setError(err.message || 'Failed to save')
            setShowOriginalLanguage(!newVal)
          })
        }}
      />
      英文文章显示原文（默认显示中文翻译）
    </label>
  </div>
</section>
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./...` and `cd frontend && npm run build` (or check TypeScript): `npx tsc --noEmit`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go frontend/src/api.ts frontend/src/components/Settings.tsx
git commit -m "feat(api:settings): add show_original_language setting endpoints"
```

---

## Task 6: Add language toggle button to ArticleReader

**Files:**
- Modify: `frontend/src/components/ArticleReader.tsx` — add language toggle button and logic
- Modify: `frontend/src/style.css` — add styles for language toggle button

- [ ] **Step 1: Read ArticleReader.tsx to understand current structure**

Run: `cat frontend/src/components/ArticleReader.tsx | head -100`

Note the structure: where the header/toolbar is, how content is displayed, how article state is managed.

- [ ] **Step 2: Add state for language toggle and translate button visibility**

Add state:
```typescript
const [showOriginal, setShowOriginal] = useState(false)
const [isTranslated, setIsTranslated] = useState(false)
```

After article loads (in useEffect that sets article), set:
```typescript
setIsTranslated(article.is_translated && !!article.translated_content)
```

- [ ] **Step 3: Add language toggle button in the header area**

Find the header/toolbar section of ArticleReader and add a button. The button should show "EN" when showing Chinese translation and "中" when showing original English. Position: top-right corner.

```tsx
{isTranslated && (
  <button
    onClick={() => setShowOriginal(!showOriginal)}
    style={{
      padding: '4px 8px',
      borderRadius: '4px',
      border: '1px solid var(--border-color)',
      background: showOriginal ? 'var(--bg-secondary)' : 'var(--accent)',
      color: showOriginal ? 'var(--text-primary)' : 'white',
      cursor: 'pointer',
      fontSize: '12px',
      fontWeight: 500,
    }}
    title={showOriginal ? '显示中文翻译' : '显示英文原文'}
  >
    {showOriginal ? '中' : 'EN'}
  </button>
)}
```

- [ ] **Step 4: Update content display to use translated content**

In the content display section, determine which content to show:
```typescript
const displayContent = showOriginal || !isTranslated
  ? article.content
  : article.translated_content
```

Use `displayContent` instead of `article.content` when rendering the article body.

- [ ] **Step 5: Add CSS for the toggle button**

Add to `style.css`:
```css
.lang-toggle-btn {
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s;
}
.lang-toggle-btn:hover {
  background: var(--hover-bg);
}
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/ArticleReader.tsx frontend/src/style.css
git commit -m "feat(frontend): add language toggle button to ArticleReader"
```

---

## Task 7: Integration test and verification

**Files:** None (testing existing code)

- [ ] **Step 1: Test the full flow**

1. Start the backend: `make dev:go`
2. Start the frontend: `make dev:frontend`
3. Find an English article in a feed
4. Click refresh on the article
5. Verify:
   - Article gets `is_translated=1` in DB
   - `translated_content` column has Chinese text
   - Summary is generated from Chinese content
6. Open the article in ArticleReader
7. Verify:
   - Toggle button shows "EN" (Chinese is displayed)
   - Clicking "EN" switches to original English content, button shows "中"
   - Clicking "中" switches back to Chinese

- [ ] **Step 2: Test settings toggle**

1. Go to Settings
2. Find the new "文章显示" section
3. Toggle "英文文章显示原文"
4. Verify: The default view for translated articles changes

- [ ] **Step 8: Commit final**

```bash
git add -A
git commit -m "feat: add English article translation with language toggle"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `internal/repository/sqlite/db.go` | Add migration for `is_translated` and `translated_content` columns |
| `internal/models/models.go` | Add `IsTranslated` and `TranslatedContent` fields to Article |
| `internal/repository/sqlite/article_repository.go` | Handle new fields in all CRUD operations and scans |
| `internal/service/rss_service.go` | Add `isEnglish()`, `TranslateArticle()`, call from `RefreshArticle()` |
| `cmd/server/main.go` | Add `GET/PUT /api/settings/show_original` handlers and routes |
| `frontend/src/api.ts` | Add `getShowOriginalLanguage()` and `setShowOriginalLanguage()` |
| `frontend/src/components/Settings.tsx` | Add show_original_language toggle UI |
| `frontend/src/components/ArticleReader.tsx` | Add language toggle button, display translated/original based on state |
| `frontend/src/style.css` | Add styles for language toggle button |

---

## Dependencies

- Task 3 depends on Task 1 and Task 2 (DB columns and model must exist first)
- Task 4 depends on Task 3 (repository must handle new fields)
- Task 5 depends on Task 1 (migrations run at startup)
- Task 6 depends on Task 5 (API must exist for frontend integration)
