import {useState, useEffect, useRef} from 'react'
import {Link, useLocation, useNavigate} from 'react-router-dom'
import {FileText, RefreshCw, Settings, LayoutGrid, ChevronLeft, ChevronRight} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import i18n from '../i18n'
import {api, Briefing as BriefingType} from '../api'
import {AppModal, injectAppModalStyles} from './AppModal'

const PAGE_SIZE = 20

export function Briefing() {
  const {t} = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const [briefings, setBriefings] = useState<BriefingType[]>([])
  const [loading, setLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [page, setPage] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [modal, setModal] = useState<{type: 'warning'|'error'; title: string; content: string} | null>(null)
  // Use ref for true guard (avoids async setState race with double-click)
  const generatingRef = useRef(false)

  injectAppModalStyles()

  const today = new Date()
  const dateStr = today.toLocaleDateString(i18n.language === 'zh' ? 'zh-CN' : 'en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  const isActive = (path: string) => {
    return location.pathname === path
  }

  // On mount: check if a briefing generation is in progress
  useEffect(() => {
    const checkProgress = async () => {
      try {
        const data = await api.getProgress()
        if (data.operation === 'generating') {
          generatingRef.current = true
          setGenerating(true)
        }
      } catch {
        // Non-critical — ignore
      }
    }
    checkProgress()
    loadBriefings(0)
  }, [])

  // Progress polling — polls /api/progress every 1s while operation is in progress.
  // Poll /api/progress while generating, reload briefings on completion.
  useEffect(() => {
    if (!generating) return

    const poll = async () => {
      try {
        const data = await api.getProgress()
        if (data.operation === 'idle') {
          generatingRef.current = false
          setGenerating(false)
          loadBriefings(0)
        }
      } catch {
        // Non-critical — keep polling
      }
    }

    poll()
    const timer = setInterval(poll, 3000)
    return () => clearInterval(timer)
  }, [generating])

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
    // Guard against double-click using ref (sync, no async setState race)
    if (generatingRef.current) return

    // Immediately disable the button before the async call
    generatingRef.current = true
    setGenerating(true)

    try {
      const result = await api.generateBriefing()
      if (!result.success) {
        if (result.code === 'OPERATION_IN_PROGRESS') {
          setModal({type: 'warning', title: '操作冲突', content: result.error || '正在执行其他操作，请稍候'})
        } else {
          setModal({type: 'error', title: '错误', content: result.error || '生成失败'})
        }
        // Reset generating since the operation didn't actually start
        generatingRef.current = false
        setGenerating(false)
        return
      }
      // Polling effect handles completion detection via /api/progress
    } catch (err: any) {
      generatingRef.current = false
      setGenerating(false)
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
          <Link to="/settings" className="masthead-btn" title={t('common.settings')}>
            <Settings size={18} />
          </Link>
        </div>
      </header>

      <div className="app-body">
        <aside className="sidebar">
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
              <span>{t('nav.briefing')}</span>
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
            <div className="briefing-header">
              <button
                onClick={handleGenerate}
                disabled={generating}
                className="btn btn-primary"
              >
                <RefreshCw size={16} className={generating ? 'spinning' : ''} />
                {generating ? t('common.loading') : t('briefing.generateBriefing')}
              </button>
            </div>

            {loading && briefings.length === 0 && (
              <div className="loading">加载中...</div>
            )}

            {!loading && briefings.length === 0 && (
              <div className="empty-state">
                <FileText size={48} />
                <p>{t('briefing.noBriefings')}</p>
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
                  {t('briefing.loadMore')}
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
