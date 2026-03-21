# AI RSS Reader UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete UI overhaul of AI RSS Reader with modern minimal design - new sidebar navigation, clean color system, consistent components.

**Architecture:** React + Vite frontend with Wails/Go backend. New Layout component wraps all pages. CSS variables for theming. Lucide icons. Google Fonts (Inter + JetBrains Mono).

**Tech Stack:** React 18, React Router 6, Vite, Lucide React, CSS Modules/Variables

---

## File Structure

```
frontend/src/
├── App.tsx              # Routes + Layout wrapper (MODIFY)
├── App.css              # Global CSS variables + styles (REWRITE)
├── style.css            # Base resets (MINOR)
├── main.tsx             # Entry point (MINOR - add font link)
├── components/
│   ├── Layout.tsx       # Sidebar + main shell (CREATE)
│   ├── FeedList.tsx     # Feeds page (REDESIGN)
│   ├── ArticleList.tsx  # Articles page (REDESIGN)
│   ├── NoteList.tsx     # Notes page (REDESIGN)
│   └── Settings.tsx     # Settings page (REDESIGN)
```

---

## Task 1: Install Dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Add lucide-react to dependencies**

Modify `frontend/package.json` to add:
```json
"lucide-react": "^0.294.0"
```

Run: `cd /home/dabao/code/ai-flow/frontend && npm install lucide-react`

---

## Task 2: Global Styles + CSS Variables

**Files:**
- Rewrite: `frontend/src/App.css`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write base CSS variables and resets**

Replace `frontend/src/App.css` with:

```css
/* ============================================
   CSS Variables - Design System Tokens
   ============================================ */
:root {
  /* Colors */
  --bg-primary: #FAFAFA;
  --bg-surface: #FFFFFF;
  --bg-sidebar: #F5F5F5;
  --text-primary: #1A1A1A;
  --text-secondary: #6B7280;
  --text-muted: #9CA3AF;
  --border: #E5E7EB;
  --accent: #3B82F6;
  --accent-hover: #2563EB;
  --accent-subtle: #EFF6FF;
  --error: #EF4444;
  --success: #10B981;
  --warning: #F59E0B;

  /* Typography */
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

  /* Spacing */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  /* Radii */
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 1px 3px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 4px 6px rgba(0, 0, 0, 0.1);

  /* Transitions */
  --transition-fast: 100ms ease-out;
  --transition-base: 150ms ease-out;
  --transition-slow: 200ms ease-out;
}

/* ============================================
   Base Styles
   ============================================ */
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body, #root {
  height: 100%;
}

body {
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  color: var(--text-primary);
  background: var(--bg-primary);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Typography */
h1, h2, h3, h4, h5, h6 {
  font-weight: 600;
  line-height: 1.3;
  color: var(--text-primary);
}

h1 { font-size: 32px; }
h2 { font-size: 24px; }
h3 { font-size: 18px; }
h4 { font-size: 16px; }

a {
  color: var(--accent);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

/* Focus states */
:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

/* ============================================
   Layout
   ============================================ */
.app {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.app-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ============================================
   Sidebar
   ============================================ */
.sidebar {
  width: 240px;
  height: 100vh;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: var(--space-6);
  border-bottom: 1px solid var(--border);
}

.sidebar-logo {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.sidebar-logo svg {
  color: var(--accent);
}

.sidebar-nav {
  flex: 1;
  padding: var(--space-4) var(--space-3);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-weight: 500;
  text-decoration: none;
  transition: all var(--transition-base);
  position: relative;
  height: 44px;
}

.nav-item:hover {
  background: var(--bg-surface);
  color: var(--text-primary);
  text-decoration: none;
}

.nav-item.active {
  background: var(--accent-subtle);
  color: var(--accent);
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 24px;
  background: var(--accent);
  border-radius: 0 2px 2px 0;
}

.nav-item svg {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.sidebar-footer {
  padding: var(--space-4);
  border-top: 1px solid var(--border);
}

/* ============================================
   Page Header
   ============================================ */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-6);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
}

.page-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6);
}

/* ============================================
   Cards
   ============================================ */
.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  transition: all var(--transition-base);
}

.card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

/* ============================================
   Buttons
   ============================================ */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  height: 40px;
  padding: 0 var(--space-4);
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all var(--transition-fast);
  border: none;
  text-decoration: none;
}

.btn:active {
  transform: scale(0.98);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--accent);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.btn-secondary {
  background: transparent;
  color: var(--accent);
  border: 1px solid var(--accent);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--accent-subtle);
}

.btn-danger {
  background: transparent;
  color: var(--error);
  border: 1px solid var(--error);
}

.btn-danger:hover:not(:disabled) {
  background: #FEF2F2;
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
}

.btn-ghost:hover:not(:disabled) {
  background: var(--bg-sidebar);
  color: var(--text-primary);
}

.btn-icon {
  width: 40px;
  padding: 0;
}

.btn-sm {
  height: 32px;
  padding: 0 var(--space-3);
  font-size: 13px;
}

/* ============================================
   Forms
   ============================================ */
.form-group {
  margin-bottom: var(--space-4);
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.form-input {
  width: 100%;
  height: 44px;
  padding: 0 var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: 14px;
  font-family: inherit;
  color: var(--text-primary);
  background: var(--bg-surface);
  transition: all var(--transition-fast);
}

.form-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder {
  color: var(--text-muted);
}

.form-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%236B7280' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 40px;
}

/* ============================================
   Badges
   ============================================ */
.badge {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.badge-filtered {
  background: var(--accent-subtle);
  color: var(--accent);
}

.badge-saved {
  background: #D1FAE5;
  color: #059669;
}

.badge-exclude {
  background: #FEF3C7;
  color: #D97706;
}

.badge-include {
  background: #D1FAE5;
  color: #059669;
}

/* ============================================
   Lists
   ============================================ */
.list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.list-item {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  transition: all var(--transition-base);
}

.list-item:hover {
  background: #F9FAFB;
}

/* ============================================
   Alerts
   ============================================ */
.alert {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-4);
}

.alert-error {
  background: #FEF2F2;
  color: var(--error);
}

.alert-success {
  background: #D1FAE5;
  color: #059669;
}

.alert-close {
  margin-left: auto;
  background: none;
  border: none;
  cursor: pointer;
  color: inherit;
  opacity: 0.7;
}

.alert-close:hover {
  opacity: 1;
}

/* ============================================
   Empty States
   ============================================ */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-16);
  text-align: center;
  color: var(--text-secondary);
}

.empty-state svg {
  width: 48px;
  height: 48px;
  margin-bottom: var(--space-4);
  opacity: 0.5;
}

.empty-state p {
  margin-bottom: var(--space-4);
}

/* ============================================
   Loading
   ============================================ */
.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
  color: var(--text-secondary);
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ============================================
   Page-specific: Feeds
   ============================================ */
.feed-form {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
}

.feed-form .form-input {
  flex: 1;
}

.feed-card {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--space-4);
}

.feed-info h3 {
  margin-bottom: var(--space-1);
}

.feed-url {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
  word-break: break-all;
}

.feed-desc {
  margin-top: var(--space-2);
  color: var(--text-secondary);
  font-size: 13px;
}

.feed-actions {
  display: flex;
  gap: var(--space-2);
  flex-shrink: 0;
}

/* ============================================
   Page-specific: Articles
   ============================================ */
.filter-bar {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
}

.filter-bar .form-select {
  width: auto;
  min-width: 150px;
}

.article-meta {
  display: flex;
  gap: var(--space-3);
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: var(--space-2);
}

.article-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-1);
}

.article-title a {
  color: var(--text-primary);
}

.article-title a:hover {
  color: var(--accent);
}

.article-author {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: var(--space-2);
}

.article-summary {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: var(--space-3);
}

.article-badges {
  display: flex;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.article-actions {
  display: flex;
  gap: var(--space-2);
}

/* ============================================
   Page-specific: Notes
   ============================================ */
.notes-layout {
  display: flex;
  height: 100%;
  margin: calc(-1 * var(--space-6));
}

.notes-sidebar {
  width: 280px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.notes-sidebar-header {
  padding: var(--space-4);
  border-bottom: 1px solid var(--border);
  font-weight: 600;
}

.notes-list {
  flex: 1;
  overflow-y: auto;
}

.note-item {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
}

.note-item:hover {
  background: var(--bg-sidebar);
}

.note-item.selected {
  background: var(--accent-subtle);
}

.note-item.selected::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--accent);
}

.note-item h4 {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: var(--space-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-date {
  font-size: 12px;
  color: var(--text-muted);
}

.note-delete-btn {
  position: absolute;
  right: var(--space-3);
  top: 50%;
  transform: translateY(-50%);
  opacity: 0;
  transition: opacity var(--transition-fast);
}

.note-item:hover .note-delete-btn {
  opacity: 1;
}

.notes-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6);
}

.notes-content .empty-state {
  height: 100%;
}

.markdown-content {
  max-width: 700px;
  line-height: 1.7;
}

.markdown-content h1 { font-size: 28px; margin: var(--space-6) 0 var(--space-4); }
.markdown-content h2 { font-size: 24px; margin: var(--space-5) 0 var(--space-3); }
.markdown-content h3 { font-size: 18px; margin: var(--space-4) 0 var(--space-2); }
.markdown-content p { margin-bottom: var(--space-4); }
.markdown-content strong { font-weight: 600; }
.markdown-content em { font-style: italic; }
.markdown-content a { color: var(--accent); }
.markdown-content blockquote {
  border-left: 3px solid var(--border);
  padding-left: var(--space-4);
  color: var(--text-secondary);
  margin: var(--space-4) 0;
}
.markdown-content code {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--bg-sidebar);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.markdown-content pre {
  background: var(--bg-sidebar);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  overflow-x: auto;
  margin: var(--space-4) 0;
}
.markdown-content pre code {
  background: none;
  padding: 0;
}
.markdown-content hr {
  border: none;
  border-top: 1px solid var(--border);
  margin: var(--space-6) 0;
}

/* ============================================
   Page-specific: Settings
   ============================================ */
.settings-section {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  margin-bottom: var(--space-6);
}

.settings-section h3 {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--border);
}

.ai-config-form {
  max-width: 500px;
}

.form-row {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.form-row .form-select,
.form-row .form-input {
  flex: 1;
  min-width: 150px;
}

.filter-rules {
  list-style: none;
  margin-top: var(--space-4);
}

.filter-rule-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--bg-sidebar);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-2);
}

.rule-value {
  flex: 1;
  font-family: var(--font-mono);
  font-size: 13px;
}

/* ============================================
   Responsive
   ============================================ */
@media (max-width: 768px) {
  .sidebar {
    width: 64px;
  }

  .sidebar-header,
  .sidebar-footer {
    display: none;
  }

  .nav-item span {
    display: none;
  }

  .nav-item {
    justify-content: center;
    padding: var(--space-3);
  }

  .page-header {
    padding: var(--space-4);
  }

  .page-content {
    padding: var(--space-4);
  }

  .feed-form {
    flex-direction: column;
  }

  .filter-bar {
    flex-direction: column;
  }

  .filter-bar .form-select {
    width: 100%;
  }

  .notes-layout {
    flex-direction: column;
  }

  .notes-sidebar {
    width: 100%;
    max-height: 40vh;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd /home/dabao/code/ai-flow
git add frontend/src/App.css
git commit -m "style: add CSS variables and global design system"
```

---

## Task 3: Create Layout Component

**Files:**
- Create: `frontend/src/components/Layout.tsx`

- [ ] **Step 1: Write Layout component**

Create `frontend/src/components/Layout.tsx`:

```tsx
import {Link, useLocation} from 'react-router-dom'
import {Rss, FileText, Settings, LayoutGrid} from 'lucide-react'

interface LayoutProps {
  children: React.ReactNode
}

export function Layout({children}: LayoutProps) {
  const location = useLocation()

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="sidebar-logo">
            <Rss size={24} />
            <span>AI RSS</span>
          </div>
        </div>

        <nav className="sidebar-nav">
          <Link
            to="/"
            className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}
          >
            <LayoutGrid />
            <span>Feeds</span>
          </Link>
          <Link
            to="/articles"
            className={`nav-item ${isActive('/articles') ? 'active' : ''}`}
          >
            <FileText />
            <span>Articles</span>
          </Link>
          <Link
            to="/notes"
            className={`nav-item ${isActive('/notes') ? 'active' : ''}`}
          >
            <FileText />
            <span>Notes</span>
          </Link>
          <Link
            to="/settings"
            className={`nav-item ${isActive('/settings') ? 'active' : ''}`}
          >
            <Settings />
            <span>Settings</span>
          </Link>
        </nav>

        <div className="sidebar-footer">
          <div style={{fontSize: '12px', color: 'var(--text-muted)'}}>
            AI RSS Reader v1.0
          </div>
        </div>
      </aside>

      <main className="app-main">
        {children}
      </main>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/Layout.tsx
git commit -m "feat: add Layout component with sidebar navigation"
```

---

## Task 4: Update App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Wrap routes with Layout**

Replace `frontend/src/App.tsx` with:

```tsx
import {Routes, Route} from 'react-router-dom'
import {Layout} from './components/Layout'
import {FeedList} from './components/FeedList'
import {ArticleList} from './components/ArticleList'
import {NoteList} from './components/NoteList'
import {Settings} from './components/Settings'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<FeedList />} />
        <Route path="/articles" element={<ArticleList />} />
        <Route path="/articles/:feedId" element={<ArticleList />} />
        <Route path="/notes" element={<NoteList />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  )
}

export default App
```

- [ ] **Step 2: Update main.tsx to add Google Fonts**

Check `frontend/src/main.tsx`:
```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import {BrowserRouter} from 'react-router-dom'
import App from './App'
import './style.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

Add Google Fonts link to `frontend/index.html` (create if not exists or check existing):
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.tsx frontend/src/main.tsx frontend/index.html
git commit -m "feat: integrate Layout with routing"
```

---

## Task 5: Redesign FeedList

**Files:**
- Rewrite: `frontend/src/components/FeedList.tsx`

- [ ] **Step 1: Write redesigned FeedList**

Replace `frontend/src/components/FeedList.tsx` with:

```tsx
import {useState, useEffect} from 'react'
import {Link} from 'react-router-dom'
import {Plus, RefreshCw, Trash2, ExternalLink, Rss} from 'lucide-react'
import {GetFeeds, AddFeed, DeleteFeed, RefreshAllFeeds} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function FeedList() {
  const [feeds, setFeeds] = useState<models.Feed[]>([])
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [refreshing, setRefreshing] = useState(false)

  const loadFeeds = async () => {
    try {
      const data = await GetFeeds()
      setFeeds(data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load feeds')
    }
  }

  useEffect(() => {
    loadFeeds()
  }, [])

  const handleAddFeed = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFeedUrl.trim()) return

    setLoading(true)
    setError('')
    try {
      await AddFeed(newFeedUrl)
      setNewFeedUrl('')
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to add feed')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteFeed = async (id: number, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    try {
      await DeleteFeed(id)
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to delete feed')
    }
  }

  const handleRefreshAll = async () => {
    setRefreshing(true)
    setError('')
    try {
      await RefreshAllFeeds()
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to refresh feeds')
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">RSS Feeds</h1>
        <button
          onClick={handleRefreshAll}
          disabled={refreshing}
          className="btn btn-primary"
        >
          <RefreshCw size={16} className={refreshing ? 'spinning' : ''} />
          {refreshing ? 'Refreshing...' : 'Refresh All'}
        </button>
      </header>

      <div className="page-content">
        {error && (
          <div className="alert alert-error">
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        <form onSubmit={handleAddFeed} className="feed-form">
          <input
            type="url"
            value={newFeedUrl}
            onChange={(e) => setNewFeedUrl(e.target.value)}
            placeholder="Enter RSS feed URL (e.g., https://news.ycombinator.com/rss)"
            className="form-input"
            required
          />
          <button type="submit" disabled={loading} className="btn btn-primary">
            <Plus size={16} />
            Add Feed
          </button>
        </form>

        {feeds.length === 0 ? (
          <div className="empty-state">
            <Rss />
            <p>No feeds yet. Add your first RSS feed above.</p>
          </div>
        ) : (
          <div className="list">
            {feeds.map((feed) => (
              <div key={feed.id} className="card feed-card">
                <div className="feed-info">
                  <h3>{feed.title || 'Untitled Feed'}</h3>
                  <p className="feed-url">{feed.url}</p>
                  {feed.description && (
                    <p className="feed-desc">{feed.description}</p>
                  )}
                </div>
                <div className="feed-actions">
                  <Link to={`/articles/${feed.id}`} className="btn btn-secondary btn-sm">
                    <ExternalLink size={14} />
                    View Articles
                  </Link>
                  <button
                    onClick={(e) => handleDeleteFeed(feed.id, e)}
                    className="btn btn-danger btn-sm btn-icon"
                    aria-label="Delete feed"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
```

- [ ] **Step 2: Add spinning animation for refresh icon**

Add to `App.css`:
```css
.spinning {
  animation: spin 1s linear infinite;
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/FeedList.tsx
git commit -m "refactor: redesign FeedList with new UI"
```

---

## Task 6: Redesign ArticleList

**Files:**
- Rewrite: `frontend/src/components/ArticleList.tsx`

- [ ] **Step 1: Write redesigned ArticleList**

Replace `frontend/src/components/ArticleList.tsx` with:

```tsx
import {useState, useEffect} from 'react'
import {useParams} from 'react-router-dom'
import {FileText, Sparkles, Save} from 'lucide-react'
import {GetArticles, GetFeeds, GenerateSummary, CreateNote, FilterAllArticles} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function ArticleList() {
  const [articles, setArticles] = useState<models.Article[]>([])
  const [feeds, setFeeds] = useState<models.Feed[]>([])
  const [selectedFeedId, setSelectedFeedId] = useState<number>(0)
  const [filterMode, setFilterMode] = useState('all')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [generatingSummary, setGeneratingSummary] = useState<number | null>(null)
  const params = useParams()

  useEffect(() => {
    const fid = params.feedId ? parseInt(params.feedId) : 0
    setSelectedFeedId(fid)
  }, [params.feedId])

  useEffect(() => {
    loadFeeds()
  }, [])

  useEffect(() => {
    loadArticles()
  }, [selectedFeedId, filterMode])

  const loadFeeds = async () => {
    try {
      const data = await GetFeeds()
      setFeeds(data || [])
    } catch (err: any) {
      console.error('Failed to load feeds:', err)
    }
  }

  const loadArticles = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await GetArticles(selectedFeedId, filterMode)
      setArticles(data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load articles')
    } finally {
      setLoading(false)
    }
  }

  const handleGenerateSummary = async (articleId: number) => {
    setGeneratingSummary(articleId)
    try {
      await GenerateSummary(articleId)
      await loadArticles()
    } catch (err: any) {
      setError(err.message || 'Failed to generate summary')
    } finally {
      setGeneratingSummary(null)
    }
  }

  const handleCreateNote = async (articleId: number) => {
    const article = articles.find(a => a.id === articleId)
    if (!article) return

    try {
      await CreateNote(articleId, article.summary || article.content)
      await loadArticles()
    } catch (err: any) {
      setError(err.message || 'Failed to create note')
    }
  }

  const handleFilterAll = async () => {
    setLoading(true)
    try {
      await FilterAllArticles()
      await loadArticles()
    } catch (err: any) {
      setError(err.message || 'Failed to filter articles')
    } finally {
      setLoading(false)
    }
  }

  const getFeedTitle = (feedId: number) => {
    const feed = feeds.find(f => f.id === feedId)
    return feed ? feed.title : 'Unknown Feed'
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">Articles</h1>
      </header>

      <div className="page-content">
        {error && (
          <div className="alert alert-error">
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        <div className="filter-bar">
          <select
            value={selectedFeedId}
            onChange={(e) => setSelectedFeedId(parseInt(e.target.value))}
            className="form-input form-select"
          >
            <option value={0}>All Feeds</option>
            {feeds.map((feed) => (
              <option key={feed.id} value={feed.id}>{feed.title}</option>
            ))}
          </select>

          <select
            value={filterMode}
            onChange={(e) => setFilterMode(e.target.value)}
            className="form-input form-select"
          >
            <option value="all">All Articles</option>
            <option value="filtered">Filtered (AI)</option>
            <option value="saved">Saved</option>
          </select>

          <button
            onClick={handleFilterAll}
            disabled={loading}
            className="btn btn-primary"
          >
            <Sparkles size={16} />
            Filter with AI
          </button>
        </div>

        {loading && articles.length === 0 ? (
          <div className="loading">
            <div className="spinner" />
            <span style={{marginLeft: '8px'}}>Loading...</span>
          </div>
        ) : articles.length === 0 ? (
          <div className="empty-state">
            <FileText />
            <p>
              {filterMode === 'all'
                ? 'No articles yet. Add some RSS feeds first.'
                : `No ${filterMode} articles.`}
            </p>
          </div>
        ) : (
          <div className="list">
            {articles.map((article) => (
              <div key={article.id} className="card">
                <div className="article-meta">
                  <span>{getFeedTitle(article.feed_id)}</span>
                  <span>{formatDate(article.published)}</span>
                </div>

                <h3 className="article-title">
                  <a href={article.link} target="_blank" rel="noopener noreferrer">
                    {article.title}
                  </a>
                </h3>

                {article.author && (
                  <p className="article-author">By {article.author}</p>
                )}

                {article.summary && (
                  <p className="article-summary">
                    {article.summary.substring(0, 200)}
                    {article.summary.length > 200 ? '...' : ''}
                  </p>
                )}

                <div className="article-badges">
                  {article.is_filtered && (
                    <span className="badge badge-filtered">Filtered</span>
                  )}
                  {article.is_saved && (
                    <span className="badge badge-saved">Saved</span>
                  )}
                </div>

                <div className="article-actions">
                  <button
                    onClick={() => handleGenerateSummary(article.id)}
                    disabled={generatingSummary === article.id}
                    className="btn btn-secondary btn-sm"
                  >
                    <Sparkles size={14} />
                    {generatingSummary === article.id ? 'Generating...' : 'AI Summary'}
                  </button>
                  {!article.is_saved && (
                    <button
                      onClick={() => handleCreateNote(article.id)}
                      className="btn btn-secondary btn-sm"
                    >
                      <Save size={14} />
                      Save as Note
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ArticleList.tsx
git commit -m "refactor: redesign ArticleList with new UI"
```

---

## Task 7: Redesign NoteList

**Files:**
- Rewrite: `frontend/src/components/NoteList.tsx`

- [ ] **Step 1: Write redesigned NoteList**

Replace `frontend/src/components/NoteList.tsx` with:

```tsx
import {useState, useEffect} from 'react'
import {FileText, Trash2} from 'lucide-react'
import {GetNotes, ReadNote, DeleteNote} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function NoteList() {
  const [notes, setNotes] = useState<models.Note[]>([])
  const [selectedNote, setSelectedNote] = useState<models.Note | null>(null)
  const [noteContent, setNoteContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    loadNotes()
  }, [])

  const loadNotes = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await GetNotes()
      setNotes(data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load notes')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectNote = async (note: models.Note) => {
    setSelectedNote(note)
    try {
      const content = await ReadNote(note.id)
      setNoteContent(content || '')
    } catch (err: any) {
      setError(err.message || 'Failed to read note')
      setNoteContent('')
    }
  }

  const handleDeleteNote = async (noteId: number, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await DeleteNote(noteId)
      if (selectedNote?.id === noteId) {
        setSelectedNote(null)
        setNoteContent('')
      }
      await loadNotes()
    } catch (err: any) {
      setError(err.message || 'Failed to delete note')
    }
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // Simple markdown formatting
  const formatMarkdown = (text: string): string => {
    if (!text) return ''

    let html = text
      .replace(/^### (.+)$/gm, '<h3>$1</h3>')
      .replace(/^## (.+)$/gm, '<h2>$1</h2>')
      .replace(/^# (.+)$/gm, '<h1>$1</h1>')
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      .replace(/\[(.+?)\]\((.+?)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
      .replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>')
      .replace(/^---$/gm, '<hr>')
      .replace(/\n\n/g, '</p><p>')
      .replace(/\n/g, '<br>')

    html = '<p>' + html + '</p>'
    html = html.replace(/<p><\/p>/g, '')
    html = html.replace(/<p>(<h[1-3]>)/g, '$1')
    html = html.replace(/(<\/h[1-3]>)<\/p>/g, '$1')
    html = html.replace(/<p>(<blockquote>)/g, '$1')
    html = html.replace(/(<\/blockquote>)<\/p>/g, '$1')
    html = html.replace(/<p>(<hr>)<\/p>/g, '$1')

    return html
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">Notes</h1>
      </header>

      <div className="page-content" style={{padding: 0, height: 'calc(100vh - 73px)'}}>
        {error && (
          <div className="alert alert-error" style={{margin: 'var(--space-4)'}}>
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        <div className="notes-layout">
          <aside className="notes-sidebar">
            <div className="notes-sidebar-header">
              {notes.length} {notes.length === 1 ? 'Note' : 'Notes'}
            </div>
            <div className="notes-list">
              {notes.length === 0 ? (
                <div className="empty-state" style={{padding: 'var(--space-8)'}}>
                  <FileText />
                  <p>No notes yet. Save articles to create notes.</p>
                </div>
              ) : (
                notes.map((note) => (
                  <div
                    key={note.id}
                    className={`note-item ${selectedNote?.id === note.id ? 'selected' : ''}`}
                    onClick={() => handleSelectNote(note)}
                  >
                    <h4>{note.title || 'Untitled Note'}</h4>
                    <p className="note-date">{formatDate(note.created_at)}</p>
                    <button
                      onClick={(e) => handleDeleteNote(note.id, e)}
                      className="btn btn-ghost btn-sm btn-icon note-delete-btn"
                      aria-label="Delete note"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                ))
              )}
            </div>
          </aside>

          <div className="notes-content">
            {selectedNote ? (
              <div className="markdown-content" dangerouslySetInnerHTML={{__html: formatMarkdown(noteContent)}} />
            ) : (
              <div className="empty-state">
                <FileText />
                <p>Select a note to view its content</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/NoteList.tsx
git commit -m "refactor: redesign NoteList with two-column layout"
```

---

## Task 8: Redesign Settings

**Files:**
- Rewrite: `frontend/src/components/Settings.tsx`

- [ ] **Step 1: Write redesigned Settings**

Replace `frontend/src/components/Settings.tsx` with:

```tsx
import {useState, useEffect} from 'react'
import {Save, Plus, Trash2} from 'lucide-react'
import {GetAIConfig, SaveAIConfig, GetFilterRules, AddFilterRule, DeleteFilterRule} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function Settings() {
  const [aiConfig, setAIConfig] = useState<models.AIProviderConfig>({
    provider: 'openai',
    api_key: '',
    base_url: 'https://api.openai.com/v1',
    model: 'gpt-3.5-turbo',
    max_tokens: 500
  })
  const [filterRules, setFilterRules] = useState<models.FilterRule[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // AI Config form state
  const [provider, setProvider] = useState('openai')
  const [apiKey, setApiKey] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [model, setModel] = useState('')
  const [maxTokens, setMaxTokens] = useState(500)

  // Filter rule form state
  const [ruleType, setRuleType] = useState('keyword')
  const [ruleValue, setRuleValue] = useState('')
  const [ruleAction, setRuleAction] = useState('exclude')

  useEffect(() => {
    loadAIConfig()
    loadFilterRules()
  }, [])

  const loadAIConfig = async () => {
    try {
      const config = await GetAIConfig()
      setAIConfig(config)
      setProvider(config.provider)
      setApiKey(config.api_key)
      setBaseURL(config.base_url)
      setModel(config.model)
      setMaxTokens(config.max_tokens)
    } catch (err: any) {
      setError(err.message || 'Failed to load AI config')
    }
  }

  const loadFilterRules = async () => {
    try {
      const rules = await GetFilterRules()
      setFilterRules(rules || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load filter rules')
    }
  }

  const handleSaveAIConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      await SaveAIConfig(provider, apiKey, baseURL, model, maxTokens)
      setSuccess('AI configuration saved successfully!')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Failed to save AI config')
    } finally {
      setLoading(false)
    }
  }

  const handleAddFilterRule = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!ruleValue.trim()) return

    setLoading(true)
    setError('')
    try {
      await AddFilterRule(ruleType, ruleValue, ruleAction)
      setRuleValue('')
      await loadFilterRules()
    } catch (err: any) {
      setError(err.message || 'Failed to add filter rule')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteFilterRule = async (id: number) => {
    try {
      await DeleteFilterRule(id)
      await loadFilterRules()
    } catch (err: any) {
      setError(err.message || 'Failed to delete filter rule')
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">Settings</h1>
      </header>

      <div className="page-content">
        {error && (
          <div className="alert alert-error">
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        {success && (
          <div className="alert alert-success">
            <span>{success}</span>
          </div>
        )}

        <section className="settings-section">
          <h3>AI Provider Configuration</h3>
          <form onSubmit={handleSaveAIConfig} className="ai-config-form">
            <div className="form-group">
              <label className="form-label">Provider</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="form-input form-select"
              >
                <option value="openai">OpenAI</option>
                <option value="claude">Claude</option>
                <option value="ollama">Ollama (Local)</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">API Key</label>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter API key"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Base URL</label>
              <input
                type="url"
                value={baseURL}
                onChange={(e) => setBaseURL(e.target.value)}
                placeholder="https://api.openai.com/v1"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Model</label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="gpt-3.5-turbo"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Max Tokens</label>
              <input
                type="number"
                value={maxTokens}
                onChange={(e) => setMaxTokens(parseInt(e.target.value))}
                min={100}
                max={4000}
                className="form-input"
              />
            </div>

            <button type="submit" disabled={loading} className="btn btn-primary">
              <Save size={16} />
              {loading ? 'Saving...' : 'Save AI Config'}
            </button>
          </form>
        </section>

        <section className="settings-section">
          <h3>Filter Rules</h3>
          <form onSubmit={handleAddFilterRule}>
            <div className="form-row">
              <select
                value={ruleType}
                onChange={(e) => setRuleType(e.target.value)}
                className="form-input form-select"
              >
                <option value="keyword">Keyword</option>
                <option value="source">Source/Author</option>
                <option value="ai_preference">AI Preference</option>
              </select>
              <input
                type="text"
                value={ruleValue}
                onChange={(e) => setRuleValue(e.target.value)}
                placeholder="Enter keyword or value"
                className="form-input"
                required
              />
              <select
                value={ruleAction}
                onChange={(e) => setRuleAction(e.target.value)}
                className="form-input form-select"
              >
                <option value="exclude">Exclude</option>
                <option value="include">Include</option>
              </select>
              <button type="submit" disabled={loading} className="btn btn-secondary">
                <Plus size={16} />
                Add Rule
              </button>
            </div>
          </form>

          {filterRules.length === 0 ? (
            <p style={{color: 'var(--text-secondary)', marginTop: 'var(--space-4)'}}>
              No filter rules defined.
            </p>
          ) : (
            <ul className="filter-rules">
              {filterRules.map((rule) => (
                <li key={rule.id} className="filter-rule-item">
                  <span className={`badge badge-${rule.action}`}>
                    {rule.action}
                  </span>
                  <span className="badge" style={{background: 'var(--bg-surface)'}}>
                    {rule.type}
                  </span>
                  <span className="rule-value">{rule.value}</span>
                  <button
                    onClick={() => handleDeleteFilterRule(rule.id)}
                    className="btn btn-ghost btn-sm btn-icon"
                    aria-label="Delete rule"
                  >
                    <Trash2 size={14} />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/Settings.tsx
git commit -m "refactor: redesign Settings with card sections"
```

---

## Task 9: Polish - Error Dismiss

**Files:**
- Modify: All component files

- [ ] **Step 1: Add auto-dismiss for error alerts**

Update each component to auto-dismiss errors after 5 seconds. Add this helper at the top of each component:

```tsx
// Add after useState declarations
useEffect(() => {
  if (error) {
    const timer = setTimeout(() => setError(''), 5000)
    return () => clearTimeout(timer)
  }
}, [error])
```

Apply to: FeedList, ArticleList, NoteList, Settings

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/FeedList.tsx frontend/src/components/ArticleList.tsx frontend/src/components/NoteList.tsx frontend/src/components/Settings.tsx
git commit -m "feat: add auto-dismiss for error alerts"
```

---

## Task 10: Verify and Test

**Files:**
- All modified files

- [ ] **Step 1: Run build to verify no errors**

```bash
cd /home/dabao/code/ai-flow/frontend && npm run build
```

Expected: Successful build with no TypeScript errors

- [ ] **Step 2: Test in browser (if dev server available)**

```bash
cd /home/dabao/code/ai-flow/frontend && npm run dev
```

- [ ] **Step 3: Final commit for UI redesign**

```bash
git add -A
git commit -m "feat: complete UI redesign with modern minimal style

- New sidebar navigation layout
- CSS variables design system
- Redesigned all pages (Feeds, Articles, Notes, Settings)
- Lucide icons throughout
- Responsive mobile support
- Error auto-dismiss
- Loading and empty states"
```

---

## Summary

Total: **10 tasks**

1. Install Dependencies (lucide-react)
2. Global Styles + CSS Variables
3. Create Layout Component
4. Update App.tsx with routing
5. Redesign FeedList
6. Redesign ArticleList
7. Redesign NoteList
8. Redesign Settings
9. Polish - Error Dismiss
10. Verify and Test
