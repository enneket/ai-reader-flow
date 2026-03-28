import {useState, useEffect, useRef} from 'react'
import {Link, useLocation} from 'react-router-dom'
import {Save, Plus, Trash2, Upload, Download, Sun, Moon, Rss, FileText, LayoutGrid, Settings as SettingsIcon} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {changeLanguage} from '../i18n'
import i18n from '../i18n'
import {CustomSelect} from './CustomSelect'
import {api, AIProviderConfig, FilterRule} from '../api'

export function Settings() {
  const {t} = useTranslation()
  const location = useLocation()
  const [aiConfig, setAIConfig] = useState<AIProviderConfig>({
    provider: 'openai',
    api_key: '',
    base_url: 'https://api.openai.com/v1',
    model: 'gpt-3.5-turbo',
    max_tokens: 500
  })
  const [filterRules, setFilterRules] = useState<FilterRule[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // AI Config form state
  const [provider, setProvider] = useState('openai')
  const [apiKey, setApiKey] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [model, setModel] = useState('')
  const [maxTokens, setMaxTokens] = useState(500)

  // Filter rule form state
  const [ruleType, setRuleType] = useState('keyword')
  const [ruleValue, setRuleValue] = useState('')
  const [ruleAction, setRuleAction] = useState('exclude')

  // OPML import
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [importing, setImporting] = useState(false)
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    return (localStorage.getItem('theme') as 'dark' | 'light') || 'dark'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  useEffect(() => {
    loadAIConfig()
    loadFilterRules()
  }, [])

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(''), 5000)
      return () => clearTimeout(timer)
    }
  }, [error])

  const loadAIConfig = async () => {
    try {
      const config = await api.getAIConfig()
      setAIConfig(config)
      setProvider(config.provider)
      setApiKey(config.api_key)
      setBaseURL(config.base_url)
      setModel(config.model)
      setMaxTokens(config.max_tokens)
    } catch (err: any) {
      setError(err.message || 'Failed to load AI config')
    }
  }

  const loadFilterRules = async () => {
    try {
      const rules = await api.getFilterRules()
      setFilterRules(rules || [])
    } catch (err: any) {
      setError(err.message || 'Failed to load filter rules')
    }
  }

  const handleSaveAIConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      await api.saveAIConfig({
        provider,
        api_key: apiKey,
        base_url: baseURL,
        model,
        max_tokens: maxTokens,
      })
      setSuccess('AI configuration saved successfully!')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Failed to save AI config')
    } finally {
      setLoading(false)
    }
  }

  const handleAddFilterRule = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!ruleValue.trim()) return

    setLoading(true)
    setError('')
    try {
      await api.addFilterRule(ruleType, ruleValue, ruleAction)
      setRuleValue('')
      await loadFilterRules()
    } catch (err: any) {
      setError(err.message || 'Failed to add filter rule')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteFilterRule = async (id: number) => {
    try {
      await api.deleteFilterRule(id)
      await loadFilterRules()
    } catch (err: any) {
      setError(err.message || 'Failed to delete filter rule')
    }
  }

  const handleExportOPML = async () => {
    try {
      const blob = await api.exportOPML()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'feeds.opml'
      a.click()
      URL.revokeObjectURL(url)
    } catch (err: any) {
      setError(err.message || 'Failed to export OPML')
    }
  }

  const handleImportOPML = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setImporting(true)
    setError('')
    setSuccess('')
    try {
      const result = await api.importOPML(file) as { imported: number; total: number; message?: string }
      if (result.imported === 0 && result.message) {
        setSuccess(result.message)
      } else {
        setSuccess(`Imported ${result.imported} of ${result.total} feeds`)
      }
      setTimeout(() => setSuccess(''), 5000)
    } catch (err: any) {
      setError(err.message || 'Failed to import OPML')
    } finally {
      setImporting(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleExportJSON = async () => {
    try {
      const blob = await api.exportSavedArticles('json')
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'saved-articles.json'
      a.click()
      URL.revokeObjectURL(url)
    } catch (err: any) {
      setError(err.message || 'Failed to export articles')
    }
  }

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  const handleExportMarkdown = async () => {
    try {
      const blob = await api.exportSavedArticles('markdown')
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'saved-articles.md'
      a.click()
      URL.revokeObjectURL(url)
    } catch (err: any) {
      setError(err.message || 'Failed to export articles')
    }
  }

  const today = new Date()
  const dateStr = today.toLocaleDateString('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  return (
    <div className="app">
      {/* Unified top masthead - consistent across all pages */}
      <header className="masthead">
        <div className="masthead-left">
          <a href="/" className="masthead-logo">
            AI RSS Reader
          </a>
        </div>
        <div className="masthead-center">{dateStr}</div>
        <div className="masthead-right">
          <Link to="/settings" className="masthead-btn" title="Settings">
            <SettingsIcon size={18} />
          </Link>
        </div>
      </header>

      <div className="app-body">
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
              to="/articles"
              className={`nav-item ${isActive('/articles') ? 'active' : ''}`}
            >
              <FileText />
              <span>{t('nav.articles')}</span>
            </Link>
            <Link
              to="/notes"
              className={`nav-item ${isActive('/notes') ? 'active' : ''}`}
            >
              <FileText />
              <span>{t('nav.notes')}</span>
            </Link>
            <Link
              to="/settings"
              className={`nav-item ${isActive('/settings') ? 'active' : ''}`}
            >
              <SettingsIcon />
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
        {error && (
          <div className="alert alert-error">
            <span>{error}</span>
            <button className="alert-close" onClick={() => setError('')}>×</button>
          </div>
        )}

        {success && (
          <div className="alert alert-success">
            <span>{success}</span>
          </div>
        )}

        <section className="settings-section">
          <h3>{t('settings.language')}</h3>
          <div className="form-group">
            <CustomSelect
              value={i18n.language}
              onChange={(val) => changeLanguage(val as 'en' | 'zh')}
              options={[
                {value: 'en', label: t('settings.english')},
                {value: 'zh', label: t('settings.chinese')},
              ]}
              className="language-select"
            />
          </div>
        </section>

        <section className="settings-section">
          <h3>Appearance</h3>
          <div className="form-group" style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
            <button
              type="button"
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
              className="btn btn-secondary"
              style={{display: 'flex', alignItems: 'center', gap: '8px'}}
            >
              {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
              {theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
            </button>
          </div>
        </section>

        <section className="settings-section">
          <h3>{t('settings.aiConfig')}</h3>
          <form onSubmit={handleSaveAIConfig} className="ai-config-form">
            <div className="form-group">
              <label className="form-label">{t('settings.provider')}</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="form-input form-select"
              >
                <option value="openai">OpenAI</option>
                <option value="claude">Claude</option>
                <option value="ollama">Ollama (Local)</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.apiKey')}</label>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter API key"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.baseURL')}</label>
              <input
                type="url"
                value={baseURL}
                onChange={(e) => setBaseURL(e.target.value)}
                placeholder="https://api.openai.com/v1"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.model')}</label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="gpt-3.5-turbo"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.maxTokens')}</label>
              <input
                type="number"
                value={maxTokens}
                onChange={(e) => setMaxTokens(parseInt(e.target.value))}
                min={100}
                max={4000}
                className="form-input"
              />
            </div>

            <button type="submit" disabled={loading} className="btn btn-primary">
              <Save size={16} />
              {loading ? t('common.loading') : t('settings.saveAIConfig')}
            </button>
          </form>
        </section>

        <section className="settings-section">
          <h3>{t('settings.filterRules')}</h3>
          <form onSubmit={handleAddFilterRule}>
            <div className="form-row">
              <select
                value={ruleType}
                onChange={(e) => setRuleType(e.target.value)}
                className="form-input form-select"
              >
                <option value="keyword">Keyword</option>
                <option value="source">Source/Author</option>
                <option value="ai_preference">AI Preference</option>
              </select>
              <input
                type="text"
                value={ruleValue}
                onChange={(e) => setRuleValue(e.target.value)}
                placeholder="Enter keyword or value"
                className="form-input"
                required
              />
              <select
                value={ruleAction}
                onChange={(e) => setRuleAction(e.target.value)}
                className="form-input form-select"
              >
                <option value="exclude">Exclude</option>
                <option value="include">Include</option>
              </select>
              <button type="submit" disabled={loading} className="btn btn-secondary">
                <Plus size={16} />
                {t('settings.addRule')}
              </button>
            </div>
          </form>

          {filterRules.length === 0 ? (
            <p style={{color: 'var(--text-secondary)', marginTop: 'var(--space-4)'}}>
              {t('settings.noFilterRules')}
            </p>
          ) : (
            <ul className="filter-rules">
              {filterRules.map((rule) => (
                <li key={rule.id} className="filter-rule-item">
                  <span className={`badge badge-${rule.action}`}>
                    {rule.action}
                  </span>
                  <span className="badge" style={{background: 'var(--surface)'}}>
                    {rule.type}
                  </span>
                  <span className="rule-value">{rule.value}</span>
                  <button
                    onClick={() => handleDeleteFilterRule(rule.id)}
                    className="btn btn-ghost btn-sm btn-icon"
                    aria-label="Delete rule"
                  >
                    <Trash2 size={14} />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="settings-section">
          <h3>OPML</h3>
          <p style={{color: 'var(--text-secondary)', marginBottom: 'var(--space-4)', fontSize: '0.875rem'}}>
            Export or import your RSS feed subscriptions via OPML file.
          </p>
          <div className="form-row">
            <button onClick={handleExportOPML} className="btn btn-secondary">
              <Download size={16} />
              Export OPML
            </button>
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={importing}
              className="btn btn-secondary"
            >
              <Upload size={16} />
              {importing ? 'Importing...' : 'Import OPML'}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".opml,.xml"
              style={{display: 'none'}}
              onChange={handleImportOPML}
            />
          </div>
        </section>

        <section className="settings-section">
          <h3>Export Saved Articles</h3>
          <p style={{color: 'var(--text-secondary)', marginBottom: 'var(--space-4)', fontSize: '0.875rem'}}>
            Download your saved articles as JSON or Markdown.
          </p>
          <div className="form-row">
            <button onClick={handleExportJSON} className="btn btn-secondary">
              <Download size={16} />
              Export JSON
            </button>
            <button onClick={handleExportMarkdown} className="btn btn-secondary">
              <Download size={16} />
              Export Markdown
            </button>
          </div>
        </section>
        </div>
      </main>
    </div>
    </div>
  )
}
