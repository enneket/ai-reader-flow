import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import i18n from './i18n/index'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
}
vi.stubGlobal('localStorage', localStorageMock)

// Mock navigator.language
Object.defineProperty(navigator, 'language', {
  value: 'en',
  writable: true,
})

describe('i18n', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorageMock.getItem.mockReturnValue(null)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('changeLanguage', async () => {
    // Need to import dynamically since i18n initializes on load
    const { changeLanguage } = await import('./i18n/index')

    it('saves language to localStorage', async () => {
      localStorageMock.getItem.mockReturnValue('en')
      await changeLanguage('zh')
      expect(localStorageMock.setItem).toHaveBeenCalledWith('language', 'zh')
    })

    it('changes i18n language', async () => {
      localStorageMock.getItem.mockReturnValue('en')
      await changeLanguage('zh')
      expect(i18n.language).toBe('zh')
    })
  })

  describe('en translations', async () => {
    const { en } = await import('./i18n/en')

    it('has nav translations', () => {
      expect(en.nav).toBeDefined()
      expect(en.nav.aiRss).toBe('AI RSS')
      expect(en.nav.feeds).toBe('Feeds')
      expect(en.nav.articles).toBe('Articles')
      expect(en.nav.notes).toBe('Notes')
      expect(en.nav.settings).toBe('Settings')
    })

    it('has feeds translations', () => {
      expect(en.feeds).toBeDefined()
      expect(en.feeds.title).toBe('RSS Feeds')
      expect(en.feeds.refreshAll).toBe('Refresh All')
      expect(en.feeds.addFeed).toBe('Add Feed')
    })

    it('has articles translations', () => {
      expect(en.articles).toBeDefined()
      expect(en.articles.empty).toBe('No articles yet. Add a feed first.')
      expect(en.articles.status.unread).toBe('Unread')
      expect(en.articles.status.accepted).toBe('Accepted')
    })

    it('has settings translations', () => {
      expect(en.settings).toBeDefined()
      expect(en.settings.aiConfig).toBe('AI Provider Configuration')
      expect(en.settings.provider).toBe('Provider')
    })
  })

  describe('zh translations', async () => {
    const { zh } = await import('./i18n/zh')

    it('has nav translations', () => {
      expect(zh.nav).toBeDefined()
      expect(zh.nav.aiRss).toBe('AI RSS')
      expect(zh.nav.feeds).toBe('订阅源')
      expect(zh.nav.articles).toBe('文章')
    })

    it('has feeds translations', () => {
      expect(zh.feeds).toBeDefined()
      expect(zh.feeds.title).toBe('RSS 订阅源')
      expect(zh.feeds.addFeed).toBe('添加订阅源')
    })

    it('has articles translations', () => {
      expect(zh.articles).toBeDefined()
      expect(zh.articles.empty).toBe('暂无文章。请先添加订阅源。')
      expect(zh.articles.status.unread).toBe('未读')
    })
  })
})
