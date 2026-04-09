import {useState, useEffect} from 'react'
import {useParams, Link, useNavigate} from 'react-router-dom'
import {FileText, RefreshCw, Settings, LayoutGrid, ArrowLeft} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import i18n from '../i18n'
import {api, Briefing as BriefingType} from '../api'

export function BriefingDetail() {
  const {t} = useTranslation()
  const {id} = useParams<{id: string}>()
  const navigate = useNavigate()
  const [briefing, setBriefing] = useState<BriefingType | null>(null)
  const [loading, setLoading] = useState(true)

  const today = new Date()
  const dateStr = today.toLocaleDateString(i18n.language === 'zh' ? 'zh-CN' : 'en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  useEffect(() => {
    loadBriefing()
  }, [id])

  // Poll every 3s while briefing is generating
  useEffect(() => {
    if (briefing?.status !== 'generating') return
    const pollInterval = setInterval(() => {
      loadBriefing()
    }, 3000)
    return () => clearInterval(pollInterval)
  }, [briefing?.status])

  const loadBriefing = async () => {
    if (!id) return
    setLoading(true)
    try {
      const data = await api.getBriefing(parseInt(id))
      setBriefing(data)
    } catch (err) {
      console.error('Failed to load briefing:', err)
    } finally {
      setLoading(false)
    }
  }

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    })
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    })
  }

  const formatBriefingTitle = (dateStr: string) => {
    const date = new Date(dateStr)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    const seconds = String(date.getSeconds()).padStart(2, '0')
    return `${year}年${month}月${day}日${hours}时${minutes}分${seconds}秒 简报`
  }

  const isActive = (path: string) => {
    return false
  }

  if (loading) {
    return (
      <div className="app">
        <header className="masthead">
          <div className="masthead-left">
            <a href="/feeds" className="masthead-logo">AI RSS Reader</a>
          </div>
          <div className="masthead-center">{dateStr}</div>
          <div className="masthead-right">
            <Link to="/settings" className="masthead-btn" title={t('common.settings')}>
              <Settings size={18} />
            </Link>
          </div>
        </header>
        <div className="app-body">
          <aside className="sidebar">
            <div className="sidebar-header">
              <div className="sidebar-logo">
                <FileText size={24} />
                <span>{t('nav.aiRss')}</span>
              </div>
            </div>
            <nav className="sidebar-nav">
              <Link to="/feeds" className="nav-item">
                <LayoutGrid />
                <span>{t('nav.feeds')}</span>
              </Link>
              <Link to="/briefing" className="nav-item active">
                <FileText />
                <span>{t('nav.briefing')}</span>
              </Link>
              <Link to="/settings" className="nav-item">
                <Settings />
                <span>{t('nav.settings')}</span>
              </Link>
            </nav>
          </aside>
          <main className="app-main">
            <div className="page-content">
              <div className="loading">{t('common.loading')}</div>
            </div>
          </main>
        </div>
      </div>
    )
  }

  if (!briefing) {
    return (
      <div className="app">
        <header className="masthead">
          <div className="masthead-left">
            <a href="/feeds" className="masthead-logo">AI RSS Reader</a>
          </div>
          <div className="masthead-center">{dateStr}</div>
          <div className="masthead-right">
            <Link to="/settings" className="masthead-btn" title={t('common.settings')}>
              <Settings size={18} />
            </Link>
          </div>
        </header>
        <div className="app-body">
          <aside className="sidebar">
            <div className="sidebar-header">
              <div className="sidebar-logo">
                <FileText size={24} />
                <span>{t('nav.aiRss')}</span>
              </div>
            </div>
            <nav className="sidebar-nav">
              <Link to="/feeds" className="nav-item">
                <LayoutGrid />
                <span>{t('nav.feeds')}</span>
              </Link>
              <Link to="/briefing" className="nav-item active">
                <FileText />
                <span>{t('nav.briefing')}</span>
              </Link>
              <Link to="/settings" className="nav-item">
                <Settings />
                <span>{t('nav.settings')}</span>
              </Link>
            </nav>
          </aside>
          <main className="app-main">
            <div className="page-content">
              <div className="empty-state">
                <FileText size={48} />
                <p>{t('briefing.notFound')}</p>
                <Link to="/briefing" className="btn btn-primary">{t('briefing.backToList')}</Link>
              </div>
            </div>
          </main>
        </div>
      </div>
    )
  }

  return (
    <div className="app">
      <header className="masthead">
        <div className="masthead-left">
          <a href="/feeds" className="masthead-logo">AI RSS Reader</a>
        </div>
        <div className="masthead-center">{dateStr}</div>
        <div className="masthead-right">
          <Link to="/settings" className="masthead-btn" title={t('common.settings')}>
            <Settings size={18} />
          </Link>
        </div>
      </header>

      <div className="app-body">
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <FileText size={24} />
              <span>{t('nav.aiRss')}</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <Link to="/feeds" className="nav-item">
              <LayoutGrid />
              <span>{t('nav.feeds')}</span>
            </Link>
            <Link to="/briefing" className="nav-item active">
              <FileText />
              <span>{t('nav.briefing')}</span>
            </Link>
            <Link to="/settings" className="nav-item">
              <Settings />
              <span>{t('nav.settings')}</span>
            </Link>
          </nav>

          <div className="sidebar-footer">
            <div style={{fontSize: '12px', color: 'var(--text-secondary)'}}>AI RSS Reader v1.0</div>
          </div>
        </aside>

        <main className="app-main">
          <div className="page-content">
            <div className="briefing-header">
              <div style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
                <Link to="/briefing" className="btn btn-ghost" style={{padding: '8px'}}>
                  <ArrowLeft size={18} />
                </Link>
                <h1>{briefing?.created_at ? formatBriefingTitle(briefing.created_at) : t('briefing.title')}</h1>
              </div>
            </div>

            <div className="briefing-detail">
              <div className="briefing-meta">
                <span className="briefing-date">{formatDate(briefing.created_at)}</span>
                <span className="briefing-time">{formatTime(briefing.created_at)}</span>
                <span className={`status-badge status-${briefing.status}`}>
                  {briefing.status === 'completed' ? t('briefing.completed') : briefing.status === 'failed' ? t('briefing.failed') : t('briefing.generating')}
                </span>
              </div>

              {briefing.status === 'completed' && briefing.items && (
                <div className="briefing-items">
                  {briefing.items.map((item) => (
                    <div key={item.id} className="briefing-item">
                      <h3>{item.topic} ({item.articles.length}{t('briefing.articles')})</h3>
                      <p className="briefing-summary">{item.summary}</p>
                      <ul className="briefing-articles">
                        {item.articles.map((article) => (
                          <li key={article.id}>
                            {article.stance && (
                              <span className={`stance-badge stance-${article.stance}`}>
                                {article.stance}
                              </span>
                            )}
                            {article.insight ? (
                              <span className="article-insight">{article.insight}</span>
                            ) : (
                              article.title
                            )}
                            {article.key_argument && (
                              <p className="article-key-argument">{article.key_argument}</p>
                            )}
                            {article.source_url && (
                              <a href={article.source_url} target="_blank" rel="noopener" className="article-source">
                                阅读原文 ↗
                              </a>
                            )}
                          </li>
                        ))}
                      </ul>
                      {item.consensus && (
                        <div className="topic-consensus">
                          <strong>共识：</strong>{item.consensus}
                        </div>
                      )}
                      {item.disputes && (
                        <div className="topic-disputes">
                          <strong>分歧：</strong>{item.disputes}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {briefing.status === 'failed' && briefing.error && (
                <div className="alert alert-error">
                  <p>{t('briefing.errorPrefix')} {briefing.error}</p>
                </div>
              )}
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
