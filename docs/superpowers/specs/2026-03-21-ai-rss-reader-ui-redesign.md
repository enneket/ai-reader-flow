# AI RSS Reader UI Redesign Specification

## Overview

Complete UI overhaul of AI RSS Reader application (Wails/Go + React) following Modern Minimal design principles.

## Design System

### Style
- **Pattern**: Modern Minimal - generous whitespace, clear hierarchy, content-first
- **Icon Library**: Lucide React (consistent stroke, modern feel)
- **No emojis** as icons

### Color Palette

| Token | Hex | Usage |
|-------|-----|-------|
| `bg-primary` | `#FAFAFA` | Main background |
| `bg-surface` | `#FFFFFF` | Cards, panels |
| `bg-sidebar` | `#F5F5F5` | Sidebar background |
| `text-primary` | `#1A1A1A` | Headings, primary text |
| `text-secondary` | `#6B7280` | Descriptions, meta text |
| `text-muted` | `#9CA3AF` | Placeholders, disabled |
| `border` | `#E5E7EB` | Dividers, input borders |
| `accent` | `#3B82F6` | Primary actions, highlights |
| `accent-hover` | `#2563EB` | Hover state |
| `accent-subtle` | `#EFF6FF` | Active backgrounds |
| `error` | `#EF4444` | Error states |
| `success` | `#10B981` | Success states |
| `warning` | `#F59E0B` | Warning states |

### Typography

- **Primary Font**: Inter (Google Fonts)
  - Headings: 600-700 weight
  - Body: 400 weight
  - Labels: 500 weight
- **Monospace**: JetBrains Mono (URLs, code)
- **Scale**: 12 / 14 / 16 / 18 / 24 / 32px

### Spacing System
- Base unit: 4px
- Spacing scale: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64px
- Card padding: 16-24px
- Section gaps: 24-32px

### Component Specifications

#### Cards
- Border radius: 12px
- Shadow: `0 1px 3px rgba(0,0,0,0.1)`
- Hover: shadow deepens, translateY(-2px)
- Transition: 150ms ease-out

#### Buttons
- Height: 40-44px (touch-friendly)
- Border radius: 8px
- Primary: blue bg, white text
- Secondary: transparent bg, blue text/border
- Danger: red border/text
- Press feedback: scale(0.98)

#### Inputs
- Height: 44px
- Border radius: 8px
- Border: 1px `#E5E7EB`
- Focus: blue border + subtle blue shadow
- Label: above input, 500 weight

#### Navigation Items
- Height: 44px
- Active: left 3px blue indicator, accent-subtle bg
- Hover: subtle bg change
- Icon + text layout

## Layout

### Desktop (>768px)
```
┌──────────────────────────────────────────────────────┐
│ ┌──────────┬─────────────────────────────────────┐  │
│ │ Sidebar  │ Header: Title + Actions              │  │
│ │ 240px    ├─────────────────────────────────────┤  │
│ │          │                                     │  │
│ │ - Logo   │ Content Area                         │  │
│ │ - Nav    │ (scrollable)                         │  │
│ │ - Footer │                                     │  │
│ └──────────┴─────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

### Mobile (<768px)
- Sidebar collapses to icon-only mode (64px)
- Or: hamburger menu → slide-out drawer

## Page Designs

### Feeds Page (`/`)

**Header**
- Title: "RSS Feeds"
- Action: "Refresh All" button (primary)

**Add Feed Section**
- Horizontal form: input (flex-grow) + "Add Feed" button
- Placeholder: "Enter RSS feed URL"

**Feed List**
- Vertical stack of cards
- Card content: title, URL (truncated), description
- Card actions: "View Articles" (secondary), "Delete" (danger, icon)

### Articles Page (`/articles`)

**Header**
- Title: "Articles"
- (No persistent action button)

**Filter Bar**
- Horizontal layout
- Select: Feed filter (All Feeds / specific feed)
- Select: Status filter (All / Filtered / Saved)
- Button: "Filter with AI" (primary)

**Article Cards**
- Meta row: feed name + date
- Title: link to original article (new tab)
- Author (if available)
- Summary: 200 char truncation + "..."
- Badges: "Filtered" (blue), "Saved" (green)
- Actions: "AI Summary" (secondary), "Save as Note" (secondary)

### Notes Page (`/notes`)

**Layout**: Two-column (sidebar list + content preview)

**Notes Sidebar** (240px)
- Scrollable list
- Note item: title, date
- Selected: blue left indicator
- Delete button on hover

**Content Area**
- Empty: "Select a note to view"
- With selection: rendered Markdown
- Styled headings, bold, italic, links, blockquotes, code

### Settings Page (`/settings`)

**Sections** (stacked, card-wrapped)

**AI Provider Configuration**
- Form fields: Provider (select), API Key (password), Base URL, Model, Max Tokens
- Submit: "Save AI Config" button

**Filter Rules**
- Form: Type (select) + Value (input) + Action (select) + Add button
- Rule list: badges for type/action + value + delete button

## Interaction Specifications

### Transitions
- Page transitions: fade 200ms
- Card hover: shadow + translateY 150ms
- Button press: scale 100ms
- Sidebar active: instant indicator, 150ms bg

### Loading States
- Skeleton screens for lists (>300ms load)
- Button spinner during submit
- "Loading..." text fallback

### Error Handling
- Error banner above content (red bg, white text)
- Auto-dismiss after 5s or manual close

### Empty States
- Centered icon + message + action
- Examples:
  - Feeds: "No feeds yet" + add feed prompt
  - Articles: "No articles yet" + add feed prompt
  - Notes: "Select a note to view its content"

## Accessibility

- All interactive elements: keyboard navigable
- Focus rings visible (blue outline)
- Touch targets: minimum 44x44px
- Color contrast: 4.5:1 minimum
- Screen reader labels for icon-only buttons
- Reduced motion: respect `prefers-reduced-motion`

## File Changes

### New/Modified Files
```
frontend/src/
├── App.tsx           # Layout shell + routing
├── App.css           # Global styles + CSS variables
├── style.css        # Base resets + typography
├── components/
│   ├── Layout.tsx   # Sidebar + main layout wrapper
│   ├── FeedList.tsx # Simplified, uses new card style
│   ├── ArticleList.tsx
│   ├── NoteList.tsx
│   └── Settings.tsx
```

### Dependencies
- `lucide-react` for icons
- Google Fonts: Inter, JetBrains Mono

## Implementation Order

1. Global styles + CSS variables
2. Layout component (sidebar + shell)
3. App.tsx routing update
4. FeedList redesign
5. ArticleList redesign
6. NoteList redesign
7. Settings redesign
8. Polish: animations, loading states, empty states
