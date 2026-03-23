import {useState, useEffect} from 'react'
import {useParams, Link, useNavigate} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {FileText, Sparkles, Save, ExternalLink, LayoutGrid, Settings, Check, X, Clock} from 'lucide-react'
import {GetArticles, GetFeeds, GenerateSummary, CreateNote, FilterAllArticles, AcceptArticle, RejectArticle, SnoozeArticle, GetDeadFeeds} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function ArticleList() {
  const {t} = useTranslation()
  const navigate = useNavigate()
  const [articles, setArticles] = useState<models.Article[]>([])
  const [feeds, setFeeds] = useState<models.Feed[]>([])
  const [selectedFeedId, setSelectedFeedId] = useState<number>(0)
  const [selectedArticle, setSelectedArticle] = useState<models.Article | null>(null)
  const [filterMode, setFilterMode] = useState('all')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [generatingSummary, setGeneratingSummary] = useState<number | null>(null)
  const [deadFeeds, setDeadFeeds] = useState<models.Feed[]>([])
  const params = useParams()

  // Load from URL params on mount
  useEffect(() => {
    if (params.feedId) {
      setSelectedFeedId(parseInt(params.feedId))
    }
  }, [])

  useEffect(() => {
    loadFeeds()
    loadDeadFeeds()
  }, [])

  const loadDeadFeeds = async () => {
    try {
      const data = await GetDeadFeeds()
      setDeadFeeds(data || [])
    } catch (err) {
      console.error('Failed to load dead feeds:', err)
    }
  }

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
      const updated = articles.find(a => a.id === articleId)
      if (updated) {
        setSelectedArticle(updated)
      }
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

  const handleAccept = async (id: number) => {
    try {
      await AcceptArticle(id)
      await loadArticles()
      const updated = articles.find(a => a.id === id)
      if (updated) {
        setSelectedArticle({...updated, status: 'accepted'})
      }
    } catch (err: any) {
      setError(err.message || 'Failed to accept article')
    }
  }

  const handleReject = async (id: number) => {
    try {
      await RejectArticle(id)
      await loadArticles()
      const updated = articles.find(a => a.id === id)
      if (updated) {
        setSelectedArticle({...updated, status: 'rejected'})
      }
    } catch (err: any) {
      setError(err.message || 'Failed to reject article')
    }
  }

  const handleSnooze = async (id: number) => {
    try {
      await SnoozeArticle(id)
      await loadArticles()
      const updated = articles.find(a => a.id === id)
      if (updated) {
        setSelectedArticle({...updated, status: 'snoozed'})
      }
    } catch (err: any) {
      setError(err.message || 'Failed to snooze article')
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

  const handleFeedClick = (feedId: number) => {
    setSelectedFeedId(feedId)
    setSelectedArticle(null)
    if (feedId > 0) {
      navigate(`/articles/${feedId}`)
    } else {
      navigate('/articles')
    }
  }

  const handleArticleClick = (article: models.Article) => {
    setSelectedArticle(article)
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
    <div className="articles-page">
      {/* Top Navigation */}
      <header className="articles-top-nav">
        <nav className="articles-nav">
          <Link to="/" className="articles-nav-item">
            <LayoutGrid size={16} />
            <span>{t('nav.feeds')}</span>
          </Link>
          <Link to="/articles" className="articles-nav-item active">
            <FileText size={16} />
            <span>{t('nav.articles')}</span>
          </Link>
          <Link to="/notes" className="articles-nav-item">
            <FileText size={16} />
            <span>{t('nav.notes')}</span>
          </Link>
          <Link to="/settings" className="articles-nav-item">
            <Settings size={16} />
            <span>{t('nav.settings')}</span>
          </Link>
        </nav>
      </header>

      {/* Dead Feeds Banner */}
      {deadFeeds.length > 0 && (
        <div className="dead-feeds-banner">
          <span>{t('feeds.deadWarning', { count: deadFeeds.length })}</span>
        </div>
      )}

      {/* 3 Column Content */}
      <div className="articles-3col">
        {/* Column 1: Feed List */}
        <div className="articles-col-feed">
          <div className="articles-col-header">{t('articles.feeds')}</div>
          <div className="feed-list">
            <button
              className={`feed-btn ${selectedFeedId === 0 ? 'active' : ''}`}
              onClick={() => handleFeedClick(0)}
            >
              {t('articles.allFeeds')}
            </button>
            {feeds.map((feed) => (
              <button
                key={feed.id}
                className={`feed-btn ${selectedFeedId === feed.id ? 'active' : ''}`}
                onClick={() => handleFeedClick(feed.id)}
              >
                {feed.title || 'Untitled'}
              </button>
            ))}
          </div>
        </div>

        {/* Column 2: Article List */}
        <div className="articles-col-list">
          <div className="articles-col-header">
            <span>{t('articles.title')}</span>
            <select
              value={filterMode}
              onChange={(e) => setFilterMode(e.target.value)}
              className="form-select-sm"
            >
              <option value="all">{t('articles.all')}</option>
              <option value="unread">{t('articles.status.unread')}</option>
              <option value="accepted">{t('articles.status.accepted')}</option>
              <option value="rejected">{t('articles.status.rejected')}</option>
              <option value="snoozed">{t('articles.status.snoozed')}</option>
              <option value="filtered">{t('articles.filtered')}</option>
              <option value="saved">{t('articles.saved')}</option>
            </select>
          </div>
          <div className="article-list">
            {loading && articles.length === 0 ? (
              <div className="loading">{t('common.loading')}</div>
            ) : articles.length === 0 ? (
              <div className="empty-state">{t('articles.empty')}</div>
            ) : (
              articles.map((article) => (
                <div
                  key={article.id}
                  className={`article-card ${selectedArticle?.id === article.id ? 'selected' : ''}`}
                  onClick={() => handleArticleClick(article)}
                >
                  <div className="article-card-meta">
                    <span>{getFeedTitle(article.feed_id)}</span>
                    <span>{formatDate(article.published)}</span>
                  </div>
                  <div className="article-card-title">{article.title}</div>
                  {article.summary && (
                    <div className="article-card-summary">
                      {article.summary.substring(0, 80)}...
                    </div>
                  )}
                  <div className="article-card-badges">
                    {article.status && article.status !== 'unread' && (
                      <span className={`badge badge-${article.status}`}>
                        {t(`articles.status.${article.status}`)}
                      </span>
                    )}
                    {article.is_filtered && <span className="badge badge-filtered">{t('articles.filtered')}</span>}
                    {article.is_saved && <span className="badge badge-saved">{t('articles.saved')}</span>}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 3: Article Content */}
        <div className="articles-col-content">
          {selectedArticle ? (
            <div className="article-content">
              <h2 className="article-content-title">{selectedArticle.title}</h2>
              <div className="article-content-meta">
                <span>{getFeedTitle(selectedArticle.feed_id)}</span>
                <span>{formatDate(selectedArticle.published)}</span>
                {selectedArticle.author && <span>{selectedArticle.author}</span>}
              </div>
              <a
                href={selectedArticle.link}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-secondary btn-sm"
              >
                <ExternalLink size={14} />
                {t('articles.viewOriginal')}
              </a>

              <div className="article-content-section">
                <h4>{t('articles.summary')}</h4>
                <p>{selectedArticle.summary || t('articles.noSummary')}</p>
              </div>

              <div className="article-content-section">
                <h4>{t('articles.content')}</h4>
                <div dangerouslySetInnerHTML={{__html: selectedArticle.content || ''}} />
              </div>

              <div className="article-content-actions">
                <button
                  onClick={() => handleAccept(selectedArticle.id)}
                  disabled={selectedArticle.status === 'accepted'}
                  className="btn btn-primary"
                >
                  <Check size={16} />
                  {t('articles.accept')}
                </button>
                <button
                  onClick={() => handleReject(selectedArticle.id)}
                  disabled={selectedArticle.status === 'rejected'}
                  className="btn btn-danger"
                >
                  <X size={16} />
                  {t('articles.reject')}
                </button>
                <button
                  onClick={() => handleSnooze(selectedArticle.id)}
                  disabled={selectedArticle.status === 'snoozed'}
                  className="btn btn-secondary"
                >
                  <Clock size={16} />
                  {t('articles.snooze')}
                </button>
              </div>

              <div className="article-content-actions">
                <button
                  onClick={() => handleGenerateSummary(selectedArticle.id)}
                  disabled={generatingSummary === selectedArticle.id}
                  className="btn btn-secondary"
                >
                  <Sparkles size={16} />
                  {generatingSummary === selectedArticle.id ? t('common.loading') : t('articles.aiSummary')}
                </button>
                {!selectedArticle.is_saved && (
                  <button
                    onClick={() => handleCreateNote(selectedArticle.id)}
                    className="btn btn-primary"
                  >
                    <Save size={16} />
                    {t('articles.saveAsNote')}
                  </button>
                )}
              </div>
            </div>
          ) : (
            <div className="empty-state">
              <FileText size={48} />
              <p>{t('articles.selectToView')}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
