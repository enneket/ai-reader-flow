import {useState, useEffect} from 'react'
import {useParams, Link, useNavigate} from 'react-router-dom'
import {FileText, RefreshCw, Settings, LayoutGrid, ArrowLeft} from 'lucide-react'
import {api, Briefing as BriefingType} from '../api'

export function BriefingDetail() {
  const {id} = useParams<{id: string}>()
  const navigate = useNavigate()
  const [briefing, setBriefing] = useState<BriefingType | null>(null)
  const [loading, setLoading] = useState(true)

  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  useEffect(() => {
    loadBriefing()
  }, [id])

  // SSE listener for briefing status updates + polling fallback
  useEffect(() => {
    // Poll every 3s while briefing is generating
    const pollInterval = setInterval(() => {
      if (briefing?.status === 'generating') {
        loadBriefing()
      }
    }, 3000)

    // SSE for real-time completion/error events
    const es = new EventSource('/api/events')

    es.addEventListener('briefing:complete', () => {
      loadBriefing()
    })

    es.addEventListener('briefing:error', () => {
      loadBriefing()
    })

    return () => {
      clearInterval(pollInterval)
      es.close()
    }
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
            <Link to="/settings" className="masthead-btn" title="Settings">
              <Settings size={18} />
            </Link>
          </div>
        </header>
        <div className="app-body">
          <aside className="sidebar">
            <div className="sidebar-header">
              <div className="sidebar-logo">
                <FileText size={24} />
                <span>AI RSS</span>
              </div>
            </div>
            <nav className="sidebar-nav">
              <Link to="/feeds" className="nav-item">
                <LayoutGrid />
                <span>订阅源</span>
              </Link>
              <Link to="/briefing" className="nav-item active">
                <FileText />
                <span>简报</span>
              </Link>
              <Link to="/settings" className="nav-item">
                <Settings />
                <span>设置</span>
              </Link>
            </nav>
          </aside>
          <main className="app-main">
            <div className="page-content">
              <div className="loading">加载中...</div>
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
            <Link to="/settings" className="masthead-btn" title="Settings">
              <Settings size={18} />
            </Link>
          </div>
        </header>
        <div className="app-body">
          <aside className="sidebar">
            <div className="sidebar-header">
              <div className="sidebar-logo">
                <FileText size={24} />
                <span>AI RSS</span>
              </div>
            </div>
            <nav className="sidebar-nav">
              <Link to="/feeds" className="nav-item">
                <LayoutGrid />
                <span>订阅源</span>
              </Link>
              <Link to="/briefing" className="nav-item active">
                <FileText />
                <span>简报</span>
              </Link>
              <Link to="/settings" className="nav-item">
                <Settings />
                <span>设置</span>
              </Link>
            </nav>
          </aside>
          <main className="app-main">
            <div className="page-content">
              <div className="empty-state">
                <FileText size={48} />
                <p>简报不存在</p>
                <Link to="/briefing" className="btn btn-primary">返回简报列表</Link>
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
          <Link to="/settings" className="masthead-btn" title="Settings">
            <Settings size={18} />
          </Link>
        </div>
      </header>

      <div className="app-body">
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <FileText size={24} />
              <span>AI RSS</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <Link to="/feeds" className="nav-item">
              <LayoutGrid />
              <span>订阅源</span>
            </Link>
            <Link to="/briefing" className="nav-item active">
              <FileText />
              <span>简报</span>
            </Link>
            <Link to="/settings" className="nav-item">
              <Settings />
              <span>设置</span>
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
                <h1>{briefing?.created_at ? formatBriefingTitle(briefing.created_at) : '简报详情'}</h1>
              </div>
            </div>

            <div className="briefing-detail">
              <div className="briefing-meta">
                <span className="briefing-date">{formatDate(briefing.created_at)}</span>
                <span className="briefing-time">{formatTime(briefing.created_at)}</span>
                <span className={`status-badge status-${briefing.status}`}>
                  {briefing.status === 'completed' ? '已完成' : briefing.status === 'failed' ? '失败' : '生成中'}
                </span>
              </div>

              {briefing.status === 'completed' && briefing.items && (
                <div className="briefing-items">
                  {briefing.items.map((item) => (
                    <div key={item.id} className="briefing-item">
                      <h3>{item.topic} ({item.articles.length}篇)</h3>
                      <p className="briefing-summary">{item.summary}</p>
                      <ul className="briefing-articles">
                        {item.articles.map((article) => (
                          <li key={article.id}>{article.title}</li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              )}

              {briefing.status === 'failed' && briefing.error && (
                <div className="alert alert-error">
                  <p>错误: {briefing.error}</p>
                </div>
              )}
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
