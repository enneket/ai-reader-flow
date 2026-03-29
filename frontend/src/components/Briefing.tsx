import {useState, useEffect} from 'react'
import {Link, useLocation} from 'react-router-dom'
import {FileText, RefreshCw, Settings, LayoutGrid} from 'lucide-react'
import {api, Briefing as BriefingType} from '../api'

export function Briefing() {
  const location = useLocation()
  const [briefings, setBriefings] = useState<BriefingType[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  useEffect(() => {
    loadBriefings()
  }, [])

  const loadBriefings = async () => {
    setLoading(true)
    try {
      const data = await api.getBriefings()
      setBriefings(data || [])
    } catch (err) {
      console.error('Failed to load briefings:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await api.generateBriefing()
      await loadBriefings()
    } catch (err) {
      console.error('Failed to generate briefing:', err)
    } finally {
      setGenerating(false)
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

  return (
    <div className="app">
      <div className="app-body">
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <FileText size={24} />
              <span>AI RSS</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <Link
              to="/feeds"
              className={`nav-item ${isActive('/feeds') ? 'active' : ''}`}
            >
              <LayoutGrid />
              <span>订阅源</span>
            </Link>
            <Link
              to="/"
              className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}
            >
              <FileText />
              <span>简报</span>
            </Link>
            <Link
              to="/settings"
              className={`nav-item ${isActive('/settings') ? 'active' : ''}`}
            >
              <Settings />
              <span>设置</span>
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
            <div className="briefing-header">
              <h1>简报</h1>
              <button
                onClick={handleGenerate}
                disabled={generating}
                className="btn btn-primary"
              >
                <RefreshCw size={16} className={generating ? 'spinning' : ''} />
                {generating ? '生成中...' : '立即生成简报'}
              </button>
            </div>

            {loading ? (
              <div className="loading">加载中...</div>
            ) : briefings.length === 0 ? (
              <div className="empty-state">
                <FileText size={48} />
                <p>暂无简报</p>
                <p style={{fontSize: '0.9rem', color: 'var(--text-secondary)'}}>
                  点击上方按钮立即生成
                </p>
              </div>
            ) : (
              <div className="briefing-list">
                {briefings.map((briefing) => (
                  <div key={briefing.id} className="briefing-card">
                    <div className="briefing-card-header">
                      <span className="briefing-date">{formatDate(briefing.created_at)}</span>
                      <span className="briefing-time">{formatTime(briefing.created_at)}</span>
                      <span className={`status-badge status-${briefing.status}`}>
                        {briefing.status === 'generating' ? '生成中' :
                         briefing.status === 'completed' ? '已完成' :
                         briefing.status === 'failed' ? '失败' : briefing.status}
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
                      <p className="briefing-error">错误: {briefing.error}</p>
                    )}
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
