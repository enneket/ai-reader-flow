import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api, type Feed, type Article, type Note, type FilterRule, type AIProviderConfig } from './api'

// Mock fetch
const mockFetch = vi.fn()
global.fetch = mockFetch

// Helper to create mock response with proper headers
function createMockResponse(data: unknown, options: { ok?: boolean; status?: number; statusText?: string; isJson?: boolean } = {}) {
  const { ok = true, status = 200, statusText = 'OK', isJson = true } = options
  return {
    ok,
    status,
    statusText,
    headers: new Headers(),
    json: isJson ? () => Promise.resolve(data) : undefined,
    blob: !isJson ? () => Promise.resolve(data as Blob) : undefined,
    text: () => Promise.resolve(statusText),
  }
}

describe('api', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getFeeds', () => {
    it('returns feeds on success', async () => {
      const feeds: Feed[] = [
        { id: 1, title: 'Test Feed', url: 'http://test.com/feed.xml', description: '', icon_url: '', last_fetched: null, is_dead: false, created_at: '', group: '' }
      ]
      mockFetch.mockResolvedValueOnce(createMockResponse(feeds))

      const result = await api.getFeeds()
      expect(result).toEqual(feeds)
      expect(mockFetch).toHaveBeenCalledWith('/api/feeds', expect.any(Object))
    })

    it('throws error on failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        headers: new Headers(),
        text: () => Promise.resolve('server error'),
      })

      await expect(api.getFeeds()).rejects.toThrow('500 server error')
    })
  })

  describe('addFeed', () => {
    it('posts feed URL and returns created feed', async () => {
      const feed: Feed = { id: 1, title: 'New Feed', url: 'http://new.com/feed.xml', description: '', icon_url: '', last_fetched: null, is_dead: false, created_at: '', group: '' }
      mockFetch.mockResolvedValueOnce(createMockResponse(feed))

      const result = await api.addFeed('http://new.com/feed.xml')
      expect(result).toEqual(feed)
      expect(mockFetch).toHaveBeenCalledWith('/api/feeds', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ url: 'http://new.com/feed.xml' }),
      }))
    })
  })

  describe('getArticles', () => {
    it('returns articles with default params', async () => {
      const articles: Article[] = [
        { id: 1, feed_id: 1, title: 'Test Article', link: 'http://test.com/article', content: '', summary: '', author: '', published: null, is_filtered: false, is_saved: false, status: 'unread', created_at: '' }
      ]
      mockFetch.mockResolvedValueOnce(createMockResponse(articles))

      const result = await api.getArticles()
      expect(result).toEqual(articles)
      expect(mockFetch).toHaveBeenCalledWith('/api/articles?filterMode=all', expect.any(Object))
    })

    it('includes feedId in params when provided', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse([]))

      await api.getArticles(5)
      expect(mockFetch).toHaveBeenCalledWith('/api/articles?feedId=5&filterMode=all', expect.any(Object))
    })

    it('includes filterMode in params when provided', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse([]))

      await api.getArticles(0, 'saved')
      expect(mockFetch).toHaveBeenCalledWith('/api/articles?filterMode=saved', expect.any(Object))
    })
  })

  describe('searchArticles', () => {
    it('encodes search query', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse([]))

      await api.searchArticles('go lang')
      expect(mockFetch).toHaveBeenCalledWith('/api/articles/search?q=go%20lang', expect.any(Object))
    })
  })

  describe('acceptArticle', () => {
    it('posts to accept endpoint', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(undefined, { status: 204 }))

      await api.acceptArticle(123)
      expect(mockFetch).toHaveBeenCalledWith('/api/articles/123/accept', expect.objectContaining({ method: 'POST' }))
    })
  })

  describe('rejectArticle', () => {
    it('posts to reject endpoint', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(undefined, { status: 204 }))

      await api.rejectArticle(456)
      expect(mockFetch).toHaveBeenCalledWith('/api/articles/456/reject', expect.objectContaining({ method: 'POST' }))
    })
  })

  describe('generateSummary', () => {
    it('returns summary', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse({ summary: 'AI summary text' }))

      const result = await api.generateSummary(1)
      expect(result).toEqual({ summary: 'AI summary text' })
    })
  })

  describe('getFilterRules', () => {
    it('returns filter rules', async () => {
      const rules: FilterRule[] = [
        { id: 1, type: 'keyword', value: 'golang', action: 'include', enabled: true, created_at: '' }
      ]
      mockFetch.mockResolvedValueOnce(createMockResponse(rules))

      const result = await api.getFilterRules()
      expect(result).toEqual(rules)
    })
  })

  describe('getNotes', () => {
    it('returns notes', async () => {
      const notes: Note[] = [
        { id: 1, article_id: 1, file_path: '/notes/test.md', title: 'Test Note', created_at: '' }
      ]
      mockFetch.mockResolvedValueOnce(createMockResponse(notes))

      const result = await api.getNotes()
      expect(result).toEqual(notes)
    })
  })

  describe('getAIConfig', () => {
    it('returns AI config', async () => {
      const config: AIProviderConfig = {
        provider: 'openai',
        api_key: 'sk-test',
        base_url: 'https://api.openai.com/v1',
        model: 'gpt-4',
        max_tokens: 500,
      }
      mockFetch.mockResolvedValueOnce(createMockResponse(config))

      const result = await api.getAIConfig()
      expect(result).toEqual(config)
    })
  })

  describe('saveAIConfig', () => {
    it('puts AI config', async () => {
      const config: AIProviderConfig = {
        provider: 'claude',
        api_key: 'sk-ant',
        base_url: 'https://api.anthropic.com',
        model: 'claude-3',
        max_tokens: 1000,
      }
      mockFetch.mockResolvedValueOnce(createMockResponse(undefined, { status: 204 }))

      await api.saveAIConfig(config)
      expect(mockFetch).toHaveBeenCalledWith('/api/ai-config', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(config),
      }))
    })
  })

  describe('exportOPML', () => {
    it('returns blob', async () => {
      const blob = new Blob(['<opml></opml>'], { type: 'application/xml' })
      mockFetch.mockResolvedValueOnce(createMockResponse(blob, { isJson: false }))

      const result = await api.exportOPML()
      expect(result).toEqual(blob)
    })
  })

  describe('exportSavedArticles', () => {
    it('returns blob', async () => {
      const blob = new Blob(['[]'], { type: 'application/json' })
      mockFetch.mockResolvedValueOnce(createMockResponse(blob, { isJson: false }))

      const result = await api.exportSavedArticles('json')
      expect(result).toEqual(blob)
    })
  })
})
