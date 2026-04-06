import {useState} from 'react'
import {RefreshCw, Settings, Search, X} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {api, Article} from '../api'

interface MastheadProps {
  isRefreshing: boolean
  onRefresh: () => void
  onSettings: () => void
  onSearchResults?: (articles: Article[]) => void
  onClearSearch?: () => void
}

export function Masthead({isRefreshing, onRefresh, onSettings, onSearchResults, onClearSearch}: MastheadProps) {
  const {t} = useTranslation()
  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
  const [searchQuery, setSearchQuery] = useState('')
  const [searching, setSearching] = useState(false)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!searchQuery.trim()) {
      onClearSearch?.()
      return
    }
    setSearching(true)
    try {
      const results = await api.searchArticles(searchQuery.trim())
      onSearchResults?.(results)
    } catch {
      // search error — silently ignore
    } finally {
      setSearching(false)
    }
  }

  const handleClearSearch = () => {
    setSearchQuery('')
    onClearSearch?.()
  }

  return (
    <header className="masthead">
      <div className="masthead-left">
        <a href="/" className="masthead-logo">
          AI RSS Reader
        </a>
      </div>
      <div className="masthead-center">{dateStr}</div>
      <div className="masthead-right">
        <form onSubmit={handleSearch} className="masthead-search">
          <Search size={14} className="masthead-search-icon" />
          <input
            type="text"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder={t('common.searchPlaceholder')}
            className="masthead-search-input"
          />
          {searchQuery && (
            <button type="button" onClick={handleClearSearch} className="masthead-search-clear">
              <X size={12} />
            </button>
          )}
        </form>
        <button
          className={`masthead-btn ${isRefreshing ? 'updating' : ''}`}
          onClick={onRefresh}
          disabled={isRefreshing || searching}
          title={t('feeds.refreshAll')}
        >
          <RefreshCw size={18} className={isRefreshing ? 'spinning' : ''} />
        </button>
        <button
          className="masthead-btn"
          onClick={onSettings}
          title={t('common.settings')}
        >
          <Settings size={18} />
        </button>
      </div>
    </header>
  )
}
