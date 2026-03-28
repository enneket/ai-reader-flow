import {useState, useEffect} from 'react'
import {Link, useLocation, useNavigate} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {Plus, RefreshCw, Trash2, Rss, FileText, Settings, LayoutGrid} from 'lucide-react'
import {api, Feed} from '../api'

export function FeedList() {
  const {t} = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [refreshing, setRefreshing] = useState(false)

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

  const handleAddFeed = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFeedUrl.trim()) return

    setLoading(true)
    setError('')
    try {
      await api.addFeed(newFeedUrl)
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
      await api.deleteFeed(id)
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to delete feed')
    }
  }

  const handleRefreshAll = async () => {
    setRefreshing(true)
    setError('')
    try {
      await api.refreshAllFeeds()
      await loadFeeds()
    } catch (err: any) {
      setError(err.message || 'Failed to refresh feeds')
    } finally {
      setRefreshing(false)
    }
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

        <main className="app-main">
          <div className="page-content">
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)'}}>
              <h1 style={{fontSize: '1.5rem', fontWeight: 600}}>{t('feeds.title')}</h1>
              <button
                onClick={handleRefreshAll}
                disabled={refreshing}
                className="btn btn-secondary"
              >
                <RefreshCw size={14} className={refreshing ? 'spinning' : ''} />
                {refreshing ? 'Refreshing...' : t('feeds.refreshAll')}
              </button>
            </div>

            {error && (
              <div className="alert alert-error" style={{marginBottom: 'var(--space-4)'}}>
                <span>{error}</span>
                <button className="alert-close" onClick={() => setError('')}>×</button>
              </div>
            )}

            <form onSubmit={handleAddFeed} style={{display: 'flex', gap: 'var(--space-2)', marginBottom: 'var(--space-4)'}}>
              <input
                type="url"
                value={newFeedUrl}
                onChange={(e) => setNewFeedUrl(e.target.value)}
                placeholder={t('feeds.placeholder')}
                className="form-input"
                required
              />
              <button type="submit" disabled={loading} className="btn btn-primary">
                <Plus size={14} />
                {t('feeds.addFeed')}
              </button>
            </form>

            {feeds.length === 0 ? (
              <div className="empty-state">
                <Rss size={48} />
                <p>{t('feeds.empty')}</p>
              </div>
            ) : (
              <div className="list">
                {feeds.map((feed) => (
                  <div
                    key={feed.id}
                    className="card feed-card clickable"
                    onClick={() => navigate(`/articles/${feed.id}`)}
                  >
                    <div className="feed-info">
                      <h3>{feed.title || 'Untitled Feed'}</h3>
                      <p className="feed-url">{feed.url}</p>
                      {feed.description && (
                        <p className="feed-desc">{feed.description}</p>
                      )}
                    </div>
                    <div className="feed-actions">
                      <button
                        onClick={(e) => handleDeleteFeed(feed.id, e)}
                        className="btn btn-ghost btn-sm btn-icon"
                        aria-label={t('feeds.delete')}
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
