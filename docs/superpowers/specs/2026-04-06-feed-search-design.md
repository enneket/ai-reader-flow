# Feed Search Feature Design

## Goal

Add feed filtering capability to the unified masthead search box, enabling users to quickly filter their subscribed feeds by name in the sidebar feed list. Also activate the unused Masthead component across all pages for consistent navigation.

## Architecture

### Current State
- `Masthead.tsx` exists with search functionality but is **not used** — FeedList, Briefing, BriefingDetail, and Settings each render inline masthead HTML
- `FeedList.tsx` has no search/filter capability for feeds
- Existing `api.searchArticles` provides FTS article search (already wired to Masthead)

### Approach
- **Option A** (chosen): Unified masthead search box that:
  1. On keystroke: frontend filter of feed list (real-time, no server round-trip)
  2. On Enter: triggers existing FTS article search (`api.searchArticles`)

## Changes

### 1. Activate Unified Masthead Across All Pages
Refactor `App.tsx` to wrap routes with a layout component containing the Masthead, so all pages share the same top navigation.

**Files:**
- `frontend/src/App.tsx` — Add `Layout` wrapper with Masthead

### 2. Add Feed Filtering to FeedList
Add `feedSearchQuery` state in `FeedList.tsx`. Filter `feeds` array in render by `feed.title.toLowerCase().includes(query.toLowerCase())`.

**Files:**
- `frontend/src/components/FeedList.tsx` — Add `feedSearchQuery` state, filtered feeds rendering

### 3. Masthead Search UX Refinement
The existing search box already calls `onSearchResults` on submit. We keep this behavior for article FTS search. Feed filtering happens in FeedList via `feedSearchQuery` state — no backend change needed.

**Files:**
- `frontend/src/components/Masthead.tsx` — Already implemented, no changes

## Data Flow

```
User types in masthead search
  └─→ FeedList receives via props or context
        └─→ feedSearchQuery state updated
              └─→ feeds.filter(f => f.title.includes(query)) rendered
              └─→ (Enter) api.searchArticles() called, results shown in article panel
```

## Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `Masthead` | Renders search input, calls `onSearchSubmit(articleQuery)` on Enter |
| `FeedList` | Maintains `feedSearchQuery` state, filters feeds list, handles article search results display |
| `Layout` | Wraps page content with consistent masthead + sidebar |

## i18n

Add `feeds.searchPlaceholder` key for feed filter input placeholder.

## Testing

1. Type in search box → feed list filters in real-time (no Enter needed)
2. Type in search box and press Enter → article FTS search results appear
3. Clear search → all feeds restored
4. Navigate to Briefing/Settings → masthead with search visible
