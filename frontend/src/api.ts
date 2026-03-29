// API client for the Go REST backend

const BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(`${res.status} ${text}`)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }
  return res.json()
}

// ─── Feeds ────────────────────────────────────────────────────────────────

export interface Feed {
  id: number
  title: string
  url: string
  description: string
  icon_url: string
  last_fetched: string | null
  is_dead: boolean
  created_at: string
  group: string
}

export const api = {
  getFeeds: () => request<Feed[]>('/feeds'),

  addFeed: (url: string) =>
    request<Feed>('/feeds', { method: 'POST', body: JSON.stringify({ url }) }),

  updateFeed: (id: number, group: string) =>
    request<Feed>(`/feeds/${id}`, { method: 'PATCH', body: JSON.stringify({ group }) }),

  deleteFeed: (id: number) =>
    request<void>(`/feeds/${id}`, { method: 'DELETE' }),

  getDeadFeeds: () => request<Feed[]>('/feeds/dead'),

  deleteDeadFeed: (id: number) =>
    request<void>(`/feeds/dead/${id}`, { method: 'DELETE' }),

  refreshFeed: (id: number) =>
    request<void>(`/feeds/${id}/refresh`, { method: 'POST' }),

  refreshAllFeeds: () =>
    request<void>('/refresh', { method: 'POST' }),

  // ─── Articles ────────────────────────────────────────────────────────────

  getArticles: (feedId: number = 0, filterMode: string = 'all') => {
    const params = new URLSearchParams()
    if (feedId) params.set('feedId', String(feedId))
    if (filterMode) params.set('filterMode', filterMode)
    const qs = params.toString()
    return request<Article[]>(`/articles${qs ? `?${qs}` : ''}`)
  },

  searchArticles: (q: string) => {
    return request<Article[]>(`/articles/search?q=${encodeURIComponent(q)}`)
  },

  getArticle: (id: number) => request<Article>(`/articles/${id}`),

  refreshArticle: (id: number) => request<Article>(`/articles/${id}/refresh`, { method: 'POST' }),

  acceptArticle: (id: number) =>
    request<void>(`/articles/${id}/accept`, { method: 'POST' }),

  rejectArticle: (id: number) =>
    request<void>(`/articles/${id}/reject`, { method: 'POST' }),

  snoozeArticle: (id: number) =>
    request<void>(`/articles/${id}/snooze`, { method: 'POST' }),

  generateSummary: (id: number) =>
    request<{ summary: string }>(`/articles/${id}/summary`, { method: 'POST' }),

  createNote: (id: number, summary: string) =>
    request<Note>(`/articles/${id}/note`, {
      method: 'POST',
      body: JSON.stringify({ summary }),
    }),

  filterArticle: (id: number) =>
    request<{ passed: boolean }>(`/articles/${id}/filter`, { method: 'POST' }),

  // ─── Filter Rules ────────────────────────────────────────────────────────

  getFilterRules: () => request<FilterRule[]>('/filter-rules'),

  addFilterRule: (type: string, value: string, action: string) =>
    request<void>('/filter-rules', {
      method: 'POST',
      body: JSON.stringify({ type, value, action }),
    }),

  deleteFilterRule: (id: number) =>
    request<void>(`/filter-rules/${id}`, { method: 'DELETE' }),

  // ─── Notes ───────────────────────────────────────────────────────────────

  getNotes: () => request<Note[]>('/notes'),

  readNote: (id: number) => request<{ content: string }>(`/notes/${id}`),

  deleteNote: (id: number) => request<void>(`/notes/${id}`, { method: 'DELETE' }),

  // ─── AI Config ───────────────────────────────────────────────────────────

  getAIConfig: () => request<AIProviderConfig>('/ai-config'),

  saveAIConfig: (config: AIProviderConfig) =>
    request<void>('/ai-config', {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  testAIConfig: () =>
    request<{success: boolean; message?: string; error?: string}>('/ai-config/test', {
      method: 'POST',
    }),

  // ─── OPML ────────────────────────────────────────────────────────────────

  exportOPML: () => {
    const base = ''
    return fetch(`${base}/opml`).then(res => {
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
      return res.blob()
    })
  },

  importOPML: (blob: Blob) => {
    const base = ''
    return fetch(`${base}/opml`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/xml' },
      body: blob,
    }).then(res => res.json())
  },

  // ─── Export ────────────────────────────────────────────────────────────────

  exportSavedArticles: (format: 'json' | 'markdown' = 'json') => {
    return fetch(`/api/export?format=${format}`).then(res => {
      if (!res.ok) throw new Error('Export failed')
      return res.blob()
    })
  },

  // ─── Briefings ────────────────────────────────────────────────────────────

  getBriefings: () => request<Briefing[]>('/briefings'),

  getBriefing: (id: number) => request<Briefing>(`/briefings/${id}`),

  generateBriefing: () => request<void>('/briefings/generate', { method: 'POST' }),

  deleteBriefing: (id: number) =>
    request<void>(`/briefings/${id}`, { method: 'DELETE' }),
}

// ─── Models (plain interfaces, match Go backend) ─────────────────────────

export interface Article {
  id: number
  feed_id: number
  title: string
  link: string
  content: string
  summary: string
  author: string
  published: string | null
  is_filtered: boolean
  is_saved: boolean
  status: string
  created_at: string
  quality_score?: number
}

export interface Note {
  id: number
  article_id: number
  file_path: string
  title: string
  created_at: string
}

export interface FilterRule {
  id: number
  type: string
  value: string
  action: string
  enabled: boolean
  created_at: string
}

export interface AIProviderConfig {
  provider: string
  api_key: string
  base_url: string
  model: string
  max_tokens: number
}

export interface BriefingArticle {
  id: number
  briefing_item_id: number
  article_id: number
  title: string
}

export interface BriefingItem {
  id: number
  briefing_id: number
  topic: string
  summary: string
  sort_order: number
  articles: BriefingArticle[]
}

export interface Briefing {
  id: number
  status: string
  error?: string
  created_at: string
  completed_at?: string
  items?: BriefingItem[]
}
