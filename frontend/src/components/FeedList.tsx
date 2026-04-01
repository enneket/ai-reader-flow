import {useState, useEffect} from 'react'
import {Link, useLocation} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {Plus, RefreshCw, Trash2, Rss, FileText, Settings, LayoutGrid} from 'lucide-react'
import {Modal} from 'antd'
import {api, Feed, Article} from '../api'
import {ArticleCard} from './ArticleCard'
import {ArticleReader} from './ArticleReader'

type RefreshProgress = {
  message: string
  current?: number
  total?: number
}

export function FeedList() {
  const {t} = useTranslation()
  const location = useLocation()
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [selectedFeed, setSelectedFeed] = useState<Feed | null>(null)
  const [articles, setArticles] = useState<Article[]>([])
  const [selectedArticle, setSelectedArticle] = useState<Article | null>(null)
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [articlesLoading, setArticlesLoading] = useState(false)
  const [error, setError] = useState('')
  const [refreshing, setRefreshing] = useState(false)
  const [isSummarizing, setIsSummarizing] = useState<number | null>(null)
  const [refreshProgress, setRefreshProgress] = useState<RefreshProgress | null>(null)

  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  const loadFeeds = async () => {
    try {
      const data = await api.getFeeds()
      setFeeds(data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load feeds')
    }
  }

  useEffect(() => {
    loadFeeds()
  }, [])

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(''), 5000)
      return () => clearTimeout(timer)
    }
  }, [error])

  // SSE listener for refresh progress events
  useEffect(() => {
    const es = new EventSource('/api/events')

    es.addEventListener('refresh:start', (e) => {
      const data = JSON.parse(e.data)
      setRefreshProgress({message: `开始刷新 ${data.total || 0} 个订阅源...`, total: data.total})
      setRefreshing(true)
    })

    es.addEventListener('refresh:progress', (e) => {
      const data = JSON.parse(e.data)
      setRefreshProgress({
        message: `正在刷新 ${data.current}/${data.total} 个订阅源: ${data.feedTitle || ''}`,
        current: data.current,
        total: data.total,
      })
    })

    es.addEventListener('refresh:complete', () => {
      setRefreshProgress(null)
      setRefreshing(false)
      loadFeeds()
      if (selectedFeed) loadArticles(selectedFeed.id)
    })

    es.addEventListener('refresh:error', (e) => {
      const data = JSON.parse(e.data)
      setRefreshProgress(null)
      setRefreshing(false)
      Modal.error({title: '刷新失败', content: data.message || '刷新订阅源失败'})
    })

    es.addEventListener('briefing:start', () => {
      // Briefing started from Briefing page - this feedlist doesn't track it
    })

    return () => es.close()
  }, [selectedFeed])

  const loadArticles = async (feedId: number) => {
    setArticlesLoading(true)
    try {
      const data = await api.getArticles(feedId, 'all')
      const sorted = (data || []).sort((a, b) => (b.quality_score || 0) - (a.quality_score || 0))
      setArticles(sorted)
    } catch (err: any) {
      setError(err.message || 'Failed to load articles')
    } finally {
      setArticlesLoading(false)
    }
  }

  const handleSelectFeed = (feed: Feed) => {
    setSelectedFeed(feed)
    setSelectedArticle(null)
    loadArticles(feed.id)
  }

  const handleAddFeed = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFeedUrl.trim()) return

    setLoading(true)
    setError('')
    try {
      const newFeed = await api.addFeed(newFeedUrl)
      setNewFeedUrl('')
      await loadFeeds()
      if (newFeed) {
        handleSelectFeed(newFeed)
      }
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
      await api.deleteFeed(id)
      if (selectedFeed?.id === id) {
        setSelectedFeed(null)
        setArticles([])
        setSelectedArticle(null)
      }
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to delete feed')
    }
  }

  const handleRefreshAll = async () => {
    setError('')
    try {
      await api.refreshAllFeeds()
      // SSE will handle setting refreshing=true and progress updates
      // On complete/error it will set refreshing=false
    } catch (err: any) {
      if (err.message.includes('409')) {
        Modal.warning({title: '操作冲突', content: '正在刷新或生成中，请稍候'})
      } else {
        setError(err.message || 'Failed to refresh feeds')
      }
    }
  }

  const handleArticleClick = (article: Article) => {
    setSelectedArticle(article)
  }

  const handleBack = () => {
    setSelectedArticle(null)
  }

  const handleAccept = async (id: number) => {
    try {
      await api.acceptArticle(id)
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to accept article:', err)
    }
  }

  const handleReject = async (id: number) => {
    try {
      await api.rejectArticle(id)
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to reject article:', err)
    }
  }

  const handleSnooze = async (id: number) => {
    try {
      await api.snoozeArticle(id)
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to snooze article:', err)
    }
  }

  const handleSave = async (id: number) => {
    try {
      await api.createNote(id, '')
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to save note:', err)
    }
  }

  const handleGenerateSummary = async (id: number) => {
    setIsSummarizing(id)
    try {
      await api.generateSummary(id)
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to generate summary:', err)
    } finally {
      setIsSummarizing(null)
    }
  }

  const handleFetchFullArticle = async (id: number) => {
    try {
      await api.refreshArticle(id)
      if (selectedFeed) await loadArticles(selectedFeed.id)
    } catch (err: any) {
      console.error('Failed to fetch full article:', err)
    }
  }

  const handleOpenExternal = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="app">
      {/* Unified top masthead */}
      <header className="masthead">
        <div className="masthead-left">
          <a href="/" className="masthead-logo">
            AI RSS Reader
          </a>
        </div>
        <div className="masthead-center">{dateStr}</div>
        <div className="masthead-right">
          <Link to="/settings" className="masthead-btn" title="Settings">
            <Settings size={18} />
          </Link>
        </div>
      </header>

      <div className="app-body">
        {/* Column 1: Sidebar Navigation */}
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <Rss size={24} />
              <span>{t('nav.aiRss')}</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <Link
              to="/feeds"
              className={`nav-item ${isActive('/feeds') ? 'active' : ''}`}
            >
              <LayoutGrid />
              <span>{t('nav.feeds')}</span>
            </Link>
            <Link
              to="/briefing"
              className={`nav-item ${isActive('/briefing') ? 'active' : ''}`}
            >
              <FileText />
              <span>简报</span>
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

        {/* Column 2: Feed List */}
        <div className="feeds-list-col">
          <div className="feeds-list-header">
            <span style={{fontSize: '0.9rem', fontWeight: 600}}>{t('feeds.title')}</span>
            <button
              onClick={handleRefreshAll}
              disabled={refreshing}
              className="btn btn-ghost btn-sm"
              title="Refresh all"
            >
              <RefreshCw size={14} className={refreshing ? 'spinning' : ''} />
            </button>
          </div>

          <form onSubmit={handleAddFeed} className="feeds-add-form">
            <input
              type="url"
              value={newFeedUrl}
              onChange={(e) => setNewFeedUrl(e.target.value)}
              placeholder={t('feeds.placeholder')}
              className="form-input"
              required
            />
            <button type="submit" disabled={loading} className="btn btn-primary btn-sm">
              <Plus size={14} />
            </button>
          </form>

          {error && (
            <div className="alert alert-error" style={{margin: 'var(--space-2)'}}>
              <span>{error}</span>
            </div>
          )}

          {refreshProgress && (
            <div style={{
              padding: 'var(--space-2)',
              background: 'var(--bg-secondary)',
              borderRadius: 'var(--radius)',
              margin: 'var(--space-2)',
              fontSize: '0.8rem',
            }}>
              <div style={{marginBottom: 'var(--space-1)'}}>🔄 {refreshProgress.message}</div>
              {refreshProgress.total && refreshProgress.current && (
                <div style={{
                  height: '3px',
                  background: 'var(--bg-primary)',
                  borderRadius: '2px',
                  overflow: 'hidden',
                }}>
                  <div style={{
                    height: '100%',
                    width: `${(refreshProgress.current / refreshProgress.total) * 100}%`,
                    background: 'var(--accent)',
                    transition: 'width 0.3s ease',
                  }} />
                </div>
              )}
            </div>
          )}

          <div className="feeds-list">
            {feeds.length === 0 ? (
              <div className="empty-state" style={{padding: 'var(--space-4)', textAlign: 'center', color: 'var(--text-secondary)'}}>
                <Rss size={24} />
                <p style={{fontSize: '0.8rem', marginTop: 'var(--space-2)'}}>{t('feeds.empty')}</p>
              </div>
            ) : (
              feeds.map((feed) => (
                <div
                  key={feed.id}
                  className={`feed-item ${selectedFeed?.id === feed.id ? 'selected' : ''}`}
                  onClick={() => handleSelectFeed(feed)}
                >
                  <div className="feed-item-info">
                    <span className="feed-item-title">{feed.title || 'Untitled Feed'}</span>
                    <span className="feed-item-url">{feed.url}</span>
                  </div>
                  <button
                    onClick={(e) => handleDeleteFeed(feed.id, e)}
                    className="btn btn-ghost btn-sm btn-icon"
                    aria-label="Delete feed"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 3: Articles List */}
        <div className="articles-list-col">
          <div className="articles-list-header">
            <span style={{fontSize: '0.9rem', fontWeight: 600}}>
              {selectedFeed?.title || ''}
            </span>
            <span style={{fontSize: '0.75rem', color: 'var(--text-secondary)'}}>
              {articles.length} article{articles.length !== 1 ? 's' : ''}
            </span>
          </div>

          <div className="articles-list">
            {!selectedFeed ? (
              <div className="empty-state" style={{padding: 'var(--space-8)', textAlign: 'center', color: 'var(--text-secondary)'}}>
                <FileText size={32} />
                <p style={{fontSize: '0.85rem', marginTop: 'var(--space-2)'}}>Select a feed to view articles</p>
              </div>
            ) : articlesLoading ? (
              <div className="loading" style={{padding: 'var(--space-4)'}}>Loading...</div>
            ) : articles.length === 0 ? (
              <div className="empty-state" style={{padding: 'var(--space-8)', textAlign: 'center', color: 'var(--text-secondary)'}}>
                <FileText size={32} />
                <p style={{fontSize: '0.85rem', marginTop: 'var(--space-2)'}}>No articles yet</p>
              </div>
            ) : (
              articles.map((article, index) => (
                <ArticleCard
                  key={article.id}
                  article={article}
                  feedName={selectedFeed?.title || ''}
                  isSelected={selectedArticle?.id === article.id}
                  isLead={index === 0}
                  isSummarizing={isSummarizing === article.id}
                  onClick={() => handleArticleClick(article)}
                />
              ))
            )}
          </div>
        </div>

        {/* Column 4: Article Reader */}
        <div className="articles-reader-col">
          {selectedArticle ? (
            <ArticleReader
              article={selectedArticle}
              feedName={selectedFeed?.title || ''}
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
          ) : (
            <div className="articles-empty-reader">
              <FileText size={48} />
              <p>Select an article to read</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}