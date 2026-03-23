import {RefreshCw, Settings} from 'lucide-react'

interface MastheadProps {
  isRefreshing: boolean
  onRefresh: () => void
  onSettings: () => void
}

export function Masthead({isRefreshing, onRefresh, onSettings}: MastheadProps) {
  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  return (
    <header className="masthead">
      <div className="masthead-left">
        <a href="/" className="masthead-logo">
          AI RSS Reader
        </a>
      </div>
      <div className="masthead-center">{dateStr}</div>
      <div className="masthead-right">
        <button
          className={`masthead-btn ${isRefreshing ? 'updating' : ''}`}
          onClick={onRefresh}
          disabled={isRefreshing}
          title="Refresh all feeds"
        >
          <RefreshCw size={18} className={isRefreshing ? 'spinning' : ''} />
        </button>
        <button
          className="masthead-btn"
          onClick={onSettings}
          title="Settings"
        >
          <Settings size={18} />
        </button>
      </div>
    </header>
  )
}
