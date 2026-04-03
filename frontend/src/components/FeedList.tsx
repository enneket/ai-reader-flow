import {useState, useEffect} from 'react'
import {Link, useLocation} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import {Plus, RefreshCw, Trash2, Rss, FileText, Settings, LayoutGrid} from 'lucide-react'
import {Modal} from 'antd'
import {api, Feed, Article} from '../api'
import {ArticleCard} from './ArticleCard'
import {ArticleReader} from './ArticleReader'

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
  const [refreshingFeedIds, setRefreshingFeedIds] = useState<Set<number>>(new Set())
  const [refreshingMessage, setRefreshingMessage] = useState('')
  const [refreshingPercent, setRefreshingPercent] = useState(0)
  const [isSummarizing, setIsSummarizing] = useState<number | null>(null)
  const [progressModal, setProgressModal] = useState<{open: boolean; title: string; content: string; percent: number}>({open: false, title: '', content: '', percent: 0})
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editFeed, setEditFeed] = useState<{id: number; title: string; url: string} | null>(null)

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

  // SSE for refresh progress
  useEffect(() => {
    const es = new EventSource('/api/events')

    es.addEventListener('refresh:start', (e) => {
      const data = JSON.parse(e.data)
      setRefreshingMessage(`开始刷新 ${data.total || 0} 个订阅源...`)
      setRefreshingPercent(0)
      setRefreshing(true)
    })

    es.addEventListener('refresh:progress', (e) => {
      const data = JSON.parse(e.data)
      const completed = data.current
      const total = data.total
      const percent = total > 0 ? Math.round((completed / total) * 100) : 0
      setRefreshingMessage(`正在刷新 ${data.feedTitle || ''} (${completed}/${total})`)
      setRefreshingPercent(percent)
    })

    es.addEventListener('refresh:complete', () => {
      setRefreshingMessage('刷新完成')
      setRefreshingPercent(100)
      setTimeout(() => {
        setRefreshingFeedIds(new Set())
        setRefreshing(false)
        setRefreshingPercent(0)
        setRefreshingMessage('')
      }, 800)
    })

    es.addEventListener('refresh:error', (e) => {
      const data = JSON.parse(e.data)
      setRefreshingMessage(data.message || '刷新失败')
      setRefreshingPercent(0)
      setTimeout(() => {
        setRefreshingFeedIds(new Set())
        setRefreshing(false)
        setRefreshingPercent(0)
        setRefreshingMessage('')
      }, 800)
    })

    return () => es.close()
  }, [])

  // Polling for refresh progress - only active when refreshing
  useEffect(() => {
    if (!refreshing) return

    const pollInterval = setInterval(async () => {
      try {
        const res = await fetch('/api/refresh/status')
        const data = await res.json()
        if (!data.inProgress) {
          // Refresh completed
          setRefreshing(false)
          if (data.error) {
            setProgressModal({open: false, title: '', content: '', percent: 0})
            Modal.error({title: '刷新失败', content: data.error})
          } else {
            setProgressModal({open: true, title: '刷新完成', content: `成功刷新 ${data.success || 0} 个订阅源`, percent: 100})
            setTimeout(() => setProgressModal({open: false, title: '', content: '', percent: 0}), 1500)
          }
          // Apply refresh results to local feed state
          if (data.results && Array.isArray(data.results)) {
            setFeeds(prevFeeds => prevFeeds.map(feed => {
              const result = data.results.find((r: { feedId: number }) => r.feedId === feed.id)
              if (result) {
                return {
                  ...feed,
                  last_refresh_success: result.success ? result.newCount : -1,
                  last_refresh_error: result.error || '',
                  last_refreshed: new Date().toISOString(),
                }
              }
              return feed
            }))
          }
          if (data.success !== undefined) {
            loadFeeds()
            if (selectedFeed) loadArticles(selectedFeed.id)
          }
          return
        }
        // In progress
        const completed = (data.success || 0) + (data.failed || 0)
        const total = data.total || 0
        const percent = total > 0 ? Math.round((completed / total) * 100) : 0
        setProgressModal({
          open: true,
          title: '正在刷新订阅源',
          content: `正在刷新 ${data.feedTitle || ''} (${completed}/${total})`,
          percent,
        })
      } catch (e) {
        console.error('Failed to poll refresh status:', e)
      }
    }, 1000)

    return () => clearInterval(pollInterval)
  }, [refreshing, selectedFeed])

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

  const handleRefreshOneFeed = async (feedId: number, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setRefreshingFeedIds(prev => new Set([...prev, feedId]))
    try {
      await api.refreshFeed(feedId)
      await loadFeeds()
      if (selectedFeed?.id === feedId) {
        await loadArticles(feedId)
      }
    } catch (err: any) {
      setError(err.message || '刷新失败')
    } finally {
      setRefreshingFeedIds(prev => {
        const next = new Set(prev)
        next.delete(feedId)
        return next
      })
    }
  }

  const handleRefreshAll = async () => {
    setError('')
    setRefreshing(true)
    setProgressModal({open: true, title: '正在刷新订阅源', content: '准备中...', percent: 0})
    try {
      await api.refreshAllFeeds()
      // Polling will update the modal
    } catch (err: any) {
      setRefreshing(false)
      setProgressModal({open: false, title: '', content: '', percent: 0})
      if (err.message.includes('409')) {
        Modal.warning({title: '操作冲突', content: '正在刷新或生成中，请稍候'})
      } else {
        setError(err.message || 'Failed to refresh feeds')
      }
    }
  }

  const handleArticleClick = async (article: Article) => {
    setSelectedArticle(article)
    // If article is unread, mark as read and update badge
    if (article.status === 'unread') {
      try {
        await api.acceptArticle(article.id)
        setFeeds(prev => prev.map(f =>
          f.id === article.feed_id
            ? {...f, unread_count: Math.max(0, f.unread_count - 1)}
            : f
        ))
      } catch (err) {
        console.error('Failed to accept article:', err)
      }
    }
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

  const handleEditFeed = (feed: Feed, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setEditFeed({id: feed.id, title: feed.title || '', url: feed.url})
    setEditModalOpen(true)
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
      <Modal
        open={progressModal.open}
        title={progressModal.title}
        footer={null}
        closable={false}
        maskClosable={false}
      >
        <div style={{padding: '8px 0'}}>
          <p>{progressModal.content}</p>
          <div style={{marginTop: 16}}>
            <div style={{height: 8, background: 'var(--bg-secondary)', borderRadius: 4, overflow: 'hidden'}}>
              <div style={{
                height: '100%',
                width: `${progressModal.percent}%`,
                background: 'var(--accent)',
                transition: 'width 0.3s ease',
              }} />
            </div>
          </div>
        </div>
      </Modal>
      <Modal
        open={editModalOpen}
        title="订阅源设置"
        onOk={async () => {
          if (!editFeed) return
          try {
            await api.updateFeed(editFeed.id, {title: editFeed.title, url: editFeed.url, group: ''})
            setEditModalOpen(false)
            await loadFeeds()
          } catch (err: any) {
            setError(err.message || '更新失败')
          }
        }}
        onCancel={() => setEditModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <div style={{display: 'flex', flexDirection: 'column', gap: 12}}>
          <div>
            <label style={{fontSize: '0.85rem', marginBottom: 4, display: 'block'}}>标题</label>
            <input
              className="form-input"
              value={editFeed?.title || ''}
              onChange={e => setEditFeed(prev => prev ? {...prev, title: e.target.value} : null)}
            />
          </div>
          <div>
            <label style={{fontSize: '0.85rem', marginBottom: 4, display: 'block'}}>订阅源链接</label>
            <input
              className="form-input"
              value={editFeed?.url || ''}
              onChange={e => setEditFeed(prev => prev ? {...prev, url: e.target.value} : null)}
            />
          </div>
        </div>
      </Modal>
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

      {(refreshingFeedIds.size > 0 || refreshing) && (
        <div className="refresh-progress-bar">
          <div className="refresh-progress-info">{progressModal.content}</div>
          <div className="refresh-progress-track">
            <div className="refresh-progress-fill" style={{width: `${progressModal.percent}%`}} />
          </div>
        </div>
      )}

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
                  <div className="feed-item-status">
                    {feed.last_refresh_success === -1 && (
                      <span className="status-failed" title={feed.last_refresh_error}>❌</span>
                    )}
                    {feed.unread_count > 0 && (
                      <span className="status-new">+{feed.unread_count}</span>
                    )}
                  </div>
                  <div style={{display: 'flex', gap: 2, alignItems: 'center'}}>
                    <button
                      onClick={(e) => handleRefreshOneFeed(feed.id, e)}
                      className="btn btn-ghost btn-sm btn-icon"
                      aria-label="Refresh feed"
                      title="刷新"
                      style={{padding: '0 4px'}}
                      disabled={refreshingFeedIds.has(feed.id)}
                    >
                      <RefreshCw size={11} className={refreshingFeedIds.has(feed.id) ? 'spinning' : ''} />
                    </button>
                    <button
                      onClick={(e) => handleDeleteFeed(feed.id, e)}
                      className="btn btn-ghost btn-sm btn-icon"
                      aria-label="Delete feed"
                      style={{padding: '0 4px'}}
                    >
                      <Trash2 size={11} />
                    </button>
                    <button
                      onClick={(e) => handleEditFeed(feed, e)}
                      className="btn btn-ghost btn-sm btn-icon"
                      aria-label="Edit feed"
                      style={{padding: '0 4px'}}
                    >
                      <Settings size={11} />
                    </button>
                  </div>
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