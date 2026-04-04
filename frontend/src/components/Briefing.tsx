import {useState, useEffect} from 'react'
import {Link, useLocation, useNavigate} from 'react-router-dom'
import {FileText, RefreshCw, Settings, LayoutGrid, ChevronLeft, ChevronRight} from 'lucide-react'
import {api, Briefing as BriefingType} from '../api'
import {AppModal, injectAppModalStyles} from './AppModal'

const PAGE_SIZE = 20

type ProgressState = {
  type: 'idle' | 'refreshing' | 'briefing'
  message: string
  current?: number
  total?: number
}

export function Briefing() {
  const location = useLocation()
  const navigate = useNavigate()
  const [briefings, setBriefings] = useState<BriefingType[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [page, setPage] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [progress, setProgress] = useState<ProgressState>({type: 'idle', message: ''})
  const [modal, setModal] = useState<{type: 'warning'|'error'; title: string; content: string} | null>(null)

  injectAppModalStyles()

  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  const isActive = (path: string) => {
    return location.pathname === path
  }

  useEffect(() => {
    loadBriefings(0)
  }, [])

  // SSE listener for refresh and briefing progress events
  useEffect(() => {
    const es = new EventSource('/api/events')

    es.addEventListener('refresh:start', (e) => {
      const data = JSON.parse(e.data)
      setProgress({type: 'refreshing', message: `开始刷新 ${data.total || 0} 个订阅源...`, total: data.total})
      setGenerating(true)
    })

    es.addEventListener('refresh:progress', (e) => {
      const data = JSON.parse(e.data)
      setProgress({
        type: 'refreshing',
        message: `正在刷新 ${data.current}/${data.total} 个订阅源: ${data.feedTitle || ''}`,
        current: data.current,
        total: data.total,
      })
    })

    es.addEventListener('refresh:complete', () => {
      setProgress({type: 'idle', message: ''})
    })

    es.addEventListener('refresh:error', (e) => {
      const data = JSON.parse(e.data)
      setProgress({type: 'idle', message: ''})
      setGenerating(false)
      setModal({type: 'error', title: '刷新失败', content: data.message || '刷新订阅源失败'})
    })

    es.addEventListener('briefing:start', () => {
      setProgress({type: 'briefing', message: '开始生成简报...'})
      setGenerating(true)
    })

    es.addEventListener('briefing:progress', (e) => {
      const data = JSON.parse(e.data)
      const stageMessages: Record<string, string> = {
        checking: '检查生成状态...',
        fetching: '正在获取文章...',
        analyzing: '正在分析文章主题...',
        generating: '正在生成简报...',
      }
      setProgress({type: 'briefing', message: stageMessages[data.stage] || data.detail || '生成中...'})
    })

    es.addEventListener('briefing:complete', () => {
      setProgress({type: 'idle', message: ''})
      setGenerating(false)
      loadBriefings(0)
    })

    es.addEventListener('briefing:error', (e) => {
      const data = JSON.parse(e.data)
      setProgress({type: 'idle', message: ''})
      setGenerating(false)
      setModal({type: 'error', title: '生成失败', content: data.message || '生成简报失败'})
    })

    return () => es.close()
  }, [])

  const loadBriefings = async (pageNum: number) => {
    setLoading(true)
    try {
      const data = await api.getBriefings(PAGE_SIZE, pageNum * PAGE_SIZE)
      if (pageNum === 0) {
        setBriefings(data || [])
      } else {
        setBriefings(prev => [...prev, ...(data || [])])
      }
      setHasMore((data || []).length === PAGE_SIZE)
      setPage(pageNum)
    } catch (err) {
      console.error('Failed to load briefings:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleGenerate = async () => {
    try {
      const result = await api.generateBriefing()
      if (!result.success) {
        if (result.code === 'OPERATION_IN_PROGRESS') {
          setModal({type: 'warning', title: '操作冲突', content: result.error || '正在执行其他操作，请稍候'})
        } else {
          setModal({type: 'error', title: '错误', content: result.error || '生成失败'})
        }
        return
      }
      // SSE will handle setting generating=false on completion/error
      // But we should set it true immediately for UX
      setGenerating(true)
    } catch (err: any) {
      if (err.message.includes('409')) {
        setModal({type: 'warning', title: '操作冲突', content: '正在刷新或生成中，请稍候'})
      } else {
        console.error('Failed to generate briefing:', err)
      }
    }
  }

  const handleBriefingClick = (briefing: BriefingType) => {
    navigate(`/briefing/${briefing.id}`)
  }

  const handleLoadMore = () => {
    loadBriefings(page + 1)
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

  return (
    <div className="app">
      {/* Unified top masthead */}
      <header className="masthead">
        <div className="masthead-left">
          <a href="/feeds" className="masthead-logo">
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
              <button
                onClick={handleGenerate}
                disabled={generating}
                className="btn btn-primary"
              >
                <RefreshCw size={16} className={generating ? 'spinning' : ''} />
                {generating ? '生成中...' : '立即生成简报'}
              </button>
            </div>

            {progress.type !== 'idle' && (
              <div style={{
                padding: 'var(--space-3)',
                background: 'var(--bg-secondary)',
                borderRadius: 'var(--radius)',
                marginBottom: 'var(--space-3)',
                fontSize: '0.9rem',
              }}>
                <div style={{marginBottom: 'var(--space-2)', color: 'var(--text-secondary)'}}>
                  {progress.type === 'refreshing' ? '🔄 刷新订阅源' : '📝 生成简报'}
                </div>
                <div style={{marginBottom: progress.total ? 'var(--space-2)' : 0}}>
                  {progress.message}
                </div>
                {progress.total && progress.current && (
                  <div style={{
                    height: '4px',
                    background: 'var(--bg-primary)',
                    borderRadius: '2px',
                    overflow: 'hidden',
                  }}>
                    <div style={{
                      height: '100%',
                      width: `${(progress.current / progress.total) * 100}%`,
                      background: 'var(--accent)',
                      transition: 'width 0.3s ease',
                    }} />
                  </div>
                )}
              </div>
            )}

            {loading && briefings.length === 0 && (
              <div className="loading">加载中...</div>
            )}

            {!loading && briefings.length === 0 && (
              <div className="empty-state">
                <FileText size={48} />
                <p>暂无简报</p>
                <p style={{fontSize: '0.9rem', color: 'var(--text-secondary)'}}>
                  点击上方按钮立即生成
                </p>
              </div>
            )}

            {briefings.length > 0 && (
              <div className="briefing-list">
                {briefings.map((briefing) => (
                  <div
                    key={briefing.id}
                    className="briefing-card"
                    onClick={() => handleBriefingClick(briefing)}
                    style={{cursor: 'pointer'}}
                  >
                    <h3 className="briefing-card-title">{formatBriefingTitle(briefing.created_at)}</h3>
                    <div className="briefing-card-header">
                      <span className="briefing-date">{formatDate(briefing.created_at)}</span>
                      <span className="briefing-time">{formatTime(briefing.created_at)}</span>
                      <span className={`status-badge status-${briefing.status}`}>
                        {briefing.status === 'generating' ? '生成中' :
                         briefing.status === 'completed' ? '已完成' :
                         briefing.status === 'failed' ? '失败' : briefing.status}
                      </span>
                    </div>
                    {briefing.status === 'failed' && briefing.error && (
                      <p className="briefing-error">错误: {briefing.error}</p>
                    )}
                  </div>
                ))}
              </div>
            )}

            {hasMore && !loading && briefings.length > 0 && (
              <div style={{display: 'flex', justifyContent: 'center', marginTop: 'var(--space-4)'}}>
                <button
                  onClick={handleLoadMore}
                  className="btn btn-secondary"
                >
                  <ChevronRight size={16} />
                  加载更多
                </button>
              </div>
            )}
          </div>
        </main>
      </div>

      {modal && (
        <AppModal
          type={modal.type}
          title={modal.title}
          content={modal.content}
          onOk={() => setModal(null)}
        />
      )}
    </div>
  )
}
