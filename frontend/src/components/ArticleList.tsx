import {useState, useEffect} from 'react'
import {useParams, Link} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {FileText, Sparkles, Save, ExternalLink, LayoutGrid, Settings} from 'lucide-react'
import {GetArticles, GetFeeds, GenerateSummary, CreateNote, FilterAllArticles} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function ArticleList() {
  const {t} = useTranslation()
  const [articles, setArticles] = useState<models.Article[]>([])
  const [feeds, setFeeds] = useState<models.Feed[]>([])
  const [selectedFeedId, setSelectedFeedId] = useState<number>(0)
  const [selectedArticle, setSelectedArticle] = useState<models.Article | null>(null)
  const [filterMode, setFilterMode] = useState('all')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [generatingSummary, setGeneratingSummary] = useState<number | null>(null)
  const params = useParams()

  useEffect(() => {
    const fid = params.feedId ? parseInt(params.feedId) : 0
    setSelectedFeedId(fid)
  }, [params.feedId])

  useEffect(() => {
    loadFeeds()
  }, [])

  useEffect(() => {
    loadArticles()
  }, [selectedFeedId, filterMode])

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(''), 5000)
      return () => clearTimeout(timer)
    }
  }, [error])

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
      // Refresh selected article from updated list
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
    <div className="articles-layout">
      {/* Top Navigation Bar */}
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

      {/* 3-column content area */}
      <div className="articles-content">
        {/* Feed List Sidebar */}
        <aside className="articles-sidebar">
        <div className="articles-sidebar-header">
          {t('articles.feeds')}
        </div>
        <div className="articles-feed-list">
          <button
            className={`feed-item ${selectedFeedId === 0 ? 'active' : ''}`}
            onClick={() => {
              setSelectedFeedId(0)
              setSelectedArticle(null)
            }}
          >
            <span className="feed-item-icon">📰</span>
            <span className="feed-item-name">{t('articles.allFeeds')}</span>
          </button>
          {feeds.map((feed) => (
            <button
              key={feed.id}
              className={`feed-item ${selectedFeedId === feed.id ? 'active' : ''}`}
              onClick={() => {
                setSelectedFeedId(feed.id)
                setSelectedArticle(null)
              }}
            >
              <span className="feed-item-icon">📋</span>
              <span className="feed-item-name">{feed.title || 'Untitled'}</span>
            </button>
          ))}
        </div>
      </aside>

      {/* Article List */}
      <div className="articles-list-panel">
        <header className="articles-list-header">
          <h1 className="page-title">{t('articles.title')}</h1>
          <div className="filter-controls">
            <select
              value={filterMode}
              onChange={(e) => setFilterMode(e.target.value)}
              className="form-input form-select"
            >
              <option value="all">{t('articles.all')}</option>
              <option value="filtered">{t('articles.filtered')}</option>
              <option value="saved">{t('articles.saved')}</option>
            </select>

            <button
              onClick={handleFilterAll}
              disabled={loading}
              className="btn btn-primary btn-sm"
            >
              <Sparkles size={14} />
              {t('articles.filterWithAI')}
            </button>
          </div>
        </header>

        <div className="articles-list">
          {loading && articles.length === 0 ? (
            <div className="loading">
              <div className="spinner" />
              <span style={{marginLeft: '8px'}}>{t('common.loading')}</span>
            </div>
          ) : articles.length === 0 ? (
            <div className="empty-state">
              <FileText />
              <p>{t('articles.empty')}</p>
            </div>
          ) : (
            articles.map((article) => (
              <div
                key={article.id}
                className={`article-item ${selectedArticle?.id === article.id ? 'selected' : ''}`}
                onClick={() => setSelectedArticle(article)}
              >
                <div className="article-meta">
                  <span>{getFeedTitle(article.feed_id)}</span>
                  <span>{formatDate(article.published)}</span>
                </div>
                <h3 className="article-title">{article.title}</h3>
                {article.author && (
                  <p className="article-author">By {article.author}</p>
                )}
                {article.summary && (
                  <p className="article-summary">
                    {article.summary.substring(0, 100)}
                    {article.summary.length > 100 ? '...' : ''}
                  </p>
                )}
                <div className="article-badges">
                  {article.is_filtered && (
                    <span className="badge badge-filtered">{t('articles.filtered')}</span>
                  )}
                  {article.is_saved && (
                    <span className="badge badge-saved">{t('articles.saved')}</span>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Article Content Preview */}
      <div className="articles-content-panel">
        {selectedArticle ? (
          <div className="article-content">
            <div className="article-content-header">
              <h2>{selectedArticle.title}</h2>
              <div className="article-content-meta">
                <span>{getFeedTitle(selectedArticle.feed_id)}</span>
                <span>{formatDate(selectedArticle.published)}</span>
                {selectedArticle.author && <span>By {selectedArticle.author}</span>}
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
            </div>

            <div className="article-content-body">
              <h4>{t('articles.summary')}</h4>
              <p>{selectedArticle.summary || t('articles.noSummary')}</p>

              <h4>{t('articles.content')}</h4>
              <div
                className="article-full-content"
                dangerouslySetInnerHTML={{__html: selectedArticle.content || selectedArticle.summary || ''}}
              />
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
            <FileText />
            <p>{t('articles.selectToView')}</p>
          </div>
        )}
      </div>
      </div>
    </div>
  )
}
