import {useState, useEffect} from 'react'
import {useNavigate, Link, useLocation} from 'react-router-dom'
import {Rss, FileText, ChevronLeft, ChevronRight, Settings, Plus, X, LayoutGrid} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {api, Article, Feed} from '../api'
import {Masthead} from './Masthead'
import {ArticleCard} from './ArticleCard'
import {ArticleReader} from './ArticleReader'

const SIDEBAR_COLLAPSED_KEY = 'sidebar_collapsed'

export function ArticleList() {
  const navigate = useNavigate()
  const location = useLocation()
  const {t} = useTranslation()
  const [articles, setArticles] = useState<Article[]>([])
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [selectedFeedId, setSelectedFeedId] = useState<number>(0)
  const [selectedArticle, setSelectedArticle] = useState<Article | null>(null)
  const [filterMode, setFilterMode] = useState('all')
  const [loading, setLoading] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [isSummarizing, setIsSummarizing] = useState<number | null>(null)
  const [searchResults, setSearchResults] = useState<Article[] | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
  })
  const [mobileReaderVisible, setMobileReaderVisible] = useState(false)
  const [showAddFeed, setShowAddFeed] = useState(false)
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [addFeedLoading, setAddFeedLoading] = useState(false)
  const [addFeedError, setAddFeedError] = useState('')
  const [showShortcuts, setShowShortcuts] = useState(false)

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  // Keyboard shortcuts
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      if (showAddFeed) return

      switch (e.key) {
        case 'j':
          navigateNext()
          break
        case 'k':
          navigatePrev()
          break
        case 'o':
        case 'Enter':
          if (selectedArticle) setMobileReaderVisible(true)
          break
        case 'Escape':
          setMobileReaderVisible(false)
          setSelectedArticle(null)
          break
        case 's':
          if (selectedArticle) handleSave(selectedArticle.id)
          break
        case 'a':
          if (selectedArticle) handleAccept(selectedArticle.id)
          break
        case 'x':
          if (selectedArticle) handleReject(selectedArticle.id)
          break
        case 'r':
          if (!isRefreshing) handleRefresh()
          break
        case '?':
          setShowShortcuts(v => !v)
          break
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [selectedArticle, isRefreshing, showAddFeed, articles])
  useEffect(() => {
    loadFeeds()
    loadArticles()
  }, [])

  // SSE: listen for new article events and auto-refresh
  useEffect(() => {
    let es: EventSource | null = null
    let reconnectTimer: ReturnType<typeof setTimeout>

    const connect = () => {
      es = new EventSource('/api/events')
      es.addEventListener('new_articles', () => {
        loadArticles()
        loadFeeds()
      })
      es.addEventListener('error', () => {
        es?.close()
        reconnectTimer = setTimeout(connect, 5000)
      })
    }

    connect()
    return () => {
      es?.close()
      clearTimeout(reconnectTimer)
    }
  }, [])

  const loadFeeds = async () => {
    try {
      const data = await api.getFeeds()
      setFeeds(data || [])
    } catch (err) {
      console.error('Failed to load feeds:', err)
    }
  }

  const loadArticles = async () => {
    setLoading(true)
    try {
      const data = await api.getArticles(selectedFeedId, filterMode)
      // Sort by quality_score descending (highest quality first)
      const sorted = (data || []).sort((a, b) => (b.quality_score || 0) - (a.quality_score || 0))
      setArticles(sorted)
    } catch (err) {
      console.error('Failed to load articles:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadArticles()
  }, [selectedFeedId, filterMode])

  // Poll for new summaries after refresh
  useEffect(() => {
    if (!isRefreshing) return
    let polls = 0
    const interval = setInterval(() => {
      loadArticles()
      polls++
      if (polls >= 10) clearInterval(interval) // 5 min max
    }, 30000)
    return () => clearInterval(interval)
  }, [isRefreshing])

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      await api.refreshAllFeeds()
      await loadArticles()
    } catch (err) {
      console.error('Failed to refresh feeds:', err)
    } finally {
      setIsRefreshing(false)
    }
  }

  const handleSettings = () => {
    navigate('/settings')
  }

  const handleAddFeed = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFeedUrl.trim()) return
    setAddFeedLoading(true)
    setAddFeedError('')
    try {
      await api.addFeed(newFeedUrl)
      setNewFeedUrl('')
      setShowAddFeed(false)
      await loadFeeds()
    } catch (err: any) {
      setAddFeedError(err.message || 'Failed to add feed')
    } finally {
      setAddFeedLoading(false)
    }
  }

  const handleFeedClick = (feedId: number) => {
    setSelectedFeedId(feedId)
    setSelectedArticle(null)
    setMobileReaderVisible(false)
  }

  const handleArticleClick = (article: Article) => {
    setSelectedArticle(article)
    setMobileReaderVisible(true)
  }

  const handleAccept = async (id: number) => {
    try {
      await api.acceptArticle(id)
      await loadArticles()
      if (selectedArticle?.id === id) {
        setSelectedArticle({...selectedArticle, status: 'accepted'})
      }
    } catch (err) {
      console.error('Failed to accept article:', err)
    }
  }

  const handleReject = async (id: number) => {
    try {
      await api.rejectArticle(id)
      await loadArticles()
      if (selectedArticle?.id === id) {
        setSelectedArticle({...selectedArticle, status: 'rejected'})
      }
    } catch (err) {
      console.error('Failed to reject article:', err)
    }
  }

  const handleSnooze = async (id: number) => {
    try {
      await api.snoozeArticle(id)
      await loadArticles()
      if (selectedArticle?.id === id) {
        setSelectedArticle({...selectedArticle, status: 'snoozed'})
      }
    } catch (err) {
      console.error('Failed to snooze article:', err)
    }
  }

  const handleSave = async (id: number) => {
    const article = articles.find(a => a.id === id)
    if (!article) return
    try {
      await api.createNote(id, article.summary || article.content)
      await loadArticles()
    } catch (err) {
      console.error('Failed to save note:', err)
    }
  }

  const handleGenerateSummary = async (id: number) => {
    setIsSummarizing(id)
    try {
      await api.generateSummary(id)
      await loadArticles()
      if (selectedArticle?.id === id) {
        const updated = articles.find(a => a.id === id)
        if (updated) setSelectedArticle(updated)
      }
    } catch (err) {
      console.error('Failed to generate summary:', err)
    } finally {
      setIsSummarizing(null)
    }
  }

  const handleFetchFullArticle = async (id: number) => {
    try {
      const updated = await api.refreshArticle(id)
      await loadArticles()
      if (selectedArticle?.id === id) {
        setSelectedArticle(updated as any)
      }
    } catch (err) {
      console.error('Failed to fetch full article:', err)
    }
  }

  const handleOpenExternal = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const handleBack = () => {
    setMobileReaderVisible(false)
    setSelectedArticle(null)
  }

  const navigateNext = () => {
    if (articles.length === 0) return
    const idx = selectedArticle ? articles.findIndex(a => a.id === selectedArticle.id) : -1
    const nextIdx = Math.min(idx + 1, articles.length - 1)
    handleArticleClick(articles[nextIdx])
  }

  const navigatePrev = () => {
    if (articles.length === 0) return
    const idx = selectedArticle ? articles.findIndex(a => a.id === selectedArticle.id) : 0
    const prevIdx = Math.max(idx - 1, 0)
    handleArticleClick(articles[prevIdx])
  }

  const toggleSidebar = () => {
    const next = !sidebarCollapsed
    setSidebarCollapsed(next)
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next))
  }

  const getFeedName = (feedId: number): string => {
    const feed = feeds.find(f => f.id === feedId)
    return feed ? feed.title : 'Unknown Feed'
  }

  const sortedFeeds = [...feeds].sort((a, b) => (a.title || '').localeCompare(b.title || ''))

  // Group feeds by group name
  const groupedFeeds = sortedFeeds.reduce<Record<string, Feed[]>>((acc, feed) => {
    const g = feed.group || ''
    if (!acc[g]) acc[g] = []
    acc[g].push(feed)
    return acc
  }, {})
  const sortedGroups = Object.keys(groupedFeeds).sort((a, b) => {
    if (a === '') return -1  // ungrouped at top
    return a.localeCompare(b)
  })

  return (
    <div className="app">
      <Masthead
        isRefreshing={isRefreshing}
        onRefresh={handleRefresh}
        onSettings={handleSettings}
        onSearchResults={setSearchResults}
        onClearSearch={() => setSearchResults(null)}
      />

      <div className="app-body">
        {/* Navigation Sidebar - unified across all pages */}
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <Rss size={24} />
              <span>{t('nav.aiRss')}</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <Link
              to="/"
              className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}
            >
              <LayoutGrid />
              <span>{t('nav.feeds')}</span>
            </Link>
            <Link
              to="/articles"
              className={`nav-item ${isActive('/articles') ? 'active' : ''}`}
            >
              <FileText />
              <span>{t('nav.articles')}</span>
            </Link>
            <Link
              to="/notes"
              className={`nav-item ${isActive('/notes') ? 'active' : ''}`}
            >
              <FileText />
              <span>{t('nav.notes')}</span>
            </Link>
            <Link
              to="/settings"
              className={`nav-item ${isActive('/settings') ? 'active' : ''}`}
            >
              <Settings />
              <span>{t('nav.settings')}</span>
            </Link>
          </nav>

          <div className="sidebar-footer">
            <div style={{fontSize: '12px', color: 'var(--text-secondary)'}}>
              AI RSS Reader v1.0
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="app-main">
          {/* Header with feed filter dropdown */}
          <header className="page-header">
            <div style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px'}}>
              <h1 className="page-title">Today's Briefing</h1>
              <div style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
                <select
                  className="form-select"
                  style={{width: 'auto', minWidth: '150px'}}
                  value={selectedFeedId}
                  onChange={e => setSelectedFeedId(Number(e.target.value))}
                >
                  <option value={0}>All Feeds</option>
                  {sortedFeeds.map(feed => (
                    <option key={feed.id} value={feed.id}>
                      {feed.title || 'Untitled'}
                    </option>
                  ))}
                </select>
                <select
                  className="form-select"
                  style={{width: 'auto', minWidth: '120px'}}
                  value={filterMode}
                  onChange={e => setFilterMode(e.target.value)}
                >
                  <option value="all">All</option>
                  <option value="unread">Unread</option>
                  <option value="accepted">Accepted</option>
                  <option value="rejected">Rejected</option>
                  <option value="snoozed">Snoozed</option>
                  <option value="saved">Saved</option>
                  <option value="filtered">Filtered</option>
                </select>
              </div>
            </div>
          </header>

          <div className="page-content">

          <div className="article-list">
            {searchResults !== null ? (
              searchResults.length === 0 ? (
                <div className="empty-state">
                  <FileText />
                  <p>No results found</p>
                </div>
              ) : (
                <>
                  <div className="article-list-header" style={{padding: '8px 16px', fontSize: '0.8rem', color: 'var(--text-secondary)'}}>
                    {searchResults.length} result{searchResults.length !== 1 ? 's' : ''}
                  </div>
                  {searchResults.map((article, index) => (
                    <ArticleCard
                      key={article.id}
                      article={article}
                      feedName={getFeedName(article.feed_id)}
                      isSelected={selectedArticle?.id === article.id}
                      isLead={index === 0}
                      isSummarizing={isSummarizing === article.id}
                      onClick={() => handleArticleClick(article)}
                    />
                  ))}
                </>
              )
            ) : loading && articles.length === 0 ? (
              <div className="loading">Loading…</div>
            ) : articles.length === 0 ? (
              <div className="empty-state">
                <FileText />
                <p>Your briefing is clear</p>
              </div>
            ) : (
              articles.map((article, index) => (
                <ArticleCard
                  key={article.id}
                  article={article}
                  feedName={getFeedName(article.feed_id)}
                  isSelected={selectedArticle?.id === article.id}
                  isLead={index === 0}
                  isSummarizing={isSummarizing === article.id}
                  onClick={() => handleArticleClick(article)}
                />
              ))
            )}
          </div>
        </div>
      </main>
    </div>

    {/* Article Reader Modal */}
    {selectedArticle && (
      <div className="modal-overlay" onClick={handleBack}>
        <div className="modal" onClick={e => e.stopPropagation()} style={{maxWidth: '700px', width: '90vw', maxHeight: '90vh', overflow: 'auto'}}>
          <div className="modal-header">
            <h2>{selectedArticle.title || 'Article'}</h2>
            <button className="btn btn-ghost btn-icon" onClick={handleBack}>
              <X size={18} />
            </button>
          </div>
          <div className="modal-body">
            <ArticleReader
              article={selectedArticle}
              feedName={selectedArticle ? getFeedName(selectedArticle.feed_id) : ''}
              isSummarizing={isSummarizing === selectedArticle?.id}
              onAccept={handleAccept}
              onReject={handleReject}
              onSnooze={handleSnooze}
              onSave={handleSave}
              onGenerateSummary={handleGenerateSummary}
              onRefresh={handleFetchFullArticle}
              onOpenExternal={handleOpenExternal}
              onBack={handleBack}
            />
          </div>
        </div>
      </div>
    )}

    {/* Shortcuts Help */}
    {showShortcuts && (
        <div className="modal-overlay" onClick={() => setShowShortcuts(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{minWidth: '280px'}}>
            <div className="modal-header">
              <h2>Keyboard Shortcuts</h2>
              <button className="btn btn-ghost btn-icon" onClick={() => setShowShortcuts(false)}>
                <X size={18} />
              </button>
            </div>
            <div className="modal-body" style={{display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '0.875rem'}}>
              {[
                ['j / k', 'Next / Previous article'],
                ['o / Enter', 'Open article reader'],
                ['Esc', 'Close reader'],
                ['a', 'Accept article'],
                ['x', 'Reject article'],
                ['s', 'Save as note'],
                ['r', 'Refresh all feeds'],
                ['?', 'Toggle this help'],
              ].map(([key, desc]) => (
                <div key={key} style={{display: 'flex', justifyContent: 'space-between', gap: '16px'}}>
                  <kbd style={{background: 'var(--surface)', padding: '2px 8px', borderRadius: '4px', fontFamily: 'monospace', fontSize: '0.8rem'}}>{key}</kbd>
                  <span style={{color: 'var(--text-secondary)'}}>{desc}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Add Feed Modal */}
      {showAddFeed && (
        <div className="modal-overlay" onClick={() => setShowAddFeed(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>添加订阅源</h2>
              <button className="btn btn-ghost btn-icon" onClick={() => setShowAddFeed(false)}>
                <X size={18} />
              </button>
            </div>
            <form onSubmit={handleAddFeed} className="modal-body">
              {addFeedError && (
                <div className="alert alert-error" style={{marginBottom: '12px'}}>
                  <span>{addFeedError}</span>
                </div>
              )}
              <input
                type="url"
                value={newFeedUrl}
                onChange={e => setNewFeedUrl(e.target.value)}
                placeholder="https://example.com/feed.xml"
                className="form-input"
                required
                autoFocus
              />
              <button type="submit" disabled={addFeedLoading} className="btn btn-primary" style={{marginTop: '12px', width: '100%'}}>
                {addFeedLoading ? '添加中...' : '添加订阅源'}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
