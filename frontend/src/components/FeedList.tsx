import {useState, useEffect} from 'react'
import {Link} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {Plus, RefreshCw, Trash2, ExternalLink, Rss} from 'lucide-react'
import {GetFeeds, AddFeed, DeleteFeed, RefreshAllFeeds} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function FeedList() {
  const {t} = useTranslation()
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
        <h1 className="page-title">{t('feeds.title')}</h1>
        <button
          onClick={handleRefreshAll}
          disabled={refreshing}
          className="btn btn-primary"
        >
          <RefreshCw size={16} className={refreshing ? 'spinning' : ''} />
          {refreshing ? 'Refreshing...' : t('feeds.refreshAll')}
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
            placeholder={t('feeds.placeholder')}
            className="form-input"
            required
          />
          <button type="submit" disabled={loading} className="btn btn-primary">
            <Plus size={16} />
            {t('feeds.addFeed')}
          </button>
        </form>

        {feeds.length === 0 ? (
          <div className="empty-state">
            <Rss />
            <p>{t('feeds.empty')}</p>
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
                    {t('feeds.viewArticles')}
                  </Link>
                  <button
                    onClick={(e) => handleDeleteFeed(feed.id, e)}
                    className="btn btn-danger btn-sm btn-icon"
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
    </>
  )
}
