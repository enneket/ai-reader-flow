import {useState, useEffect, useRef} from 'react'
import {Link, useLocation} from 'react-router-dom'
import {Save, Upload, Download, Sun, Moon, Rss, FileText, LayoutGrid, Settings as SettingsIcon} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {changeLanguage} from '../i18n'
import i18n from '../i18n'
import {CustomSelect} from './CustomSelect'
import {api, AIProviderConfig} from '../api'

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
  const [loading, setLoading] = useState(false)
  const [testingConnection, setTestingConnection] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // AI Config form state
  const [provider, setProvider] = useState('openai')
  const [apiKey, setApiKey] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [model, setModel] = useState('')
  const [maxTokens, setMaxTokens] = useState(500)

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

  const handleTestConnection = async () => {
    // Save current config first
    setTestingConnection(true)
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
      const result = await api.testAIConfig()
      if (result.success) {
        setSuccess('Connection successful! AI is reachable.')
        setTimeout(() => setSuccess(''), 5000)
      } else {
        setError(result.error || 'Connection failed')
      }
    } catch (err: any) {
      setError(err.message || 'Failed to test AI connection')
    } finally {
      setTestingConnection(false)
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
          <h3>{t('settings.appearance')}</h3>
          <div className="form-group" style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
            <button
              type="button"
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
              className="btn btn-secondary"
              style={{display: 'flex', alignItems: 'center', gap: '8px'}}
            >
              {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
              {theme === 'dark' ? t('settings.switchLight') : t('settings.switchDark')}
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
                <option value="openai">OpenAI 兼容（通用）</option>
                <option value="claude">Claude</option>
                <option value="ollama">Ollama (本地)</option>
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

            <button type="button" onClick={handleTestConnection} disabled={testingConnection || loading} className="btn btn-outline-accent">
              <Save size={16} />
              {testingConnection ? t('settings.testing') : t('settings.testConnection')}
            </button>
            <button type="submit" disabled={loading} className="btn btn-primary">
              <Save size={16} />
              {loading ? t('common.loading') : t('settings.saveAIConfig')}
            </button>
          </form>
        </section>

        <section className="settings-section">
          <h3>{t('settings.opml')}</h3>
          <p style={{color: 'var(--text-secondary)', marginBottom: 'var(--space-4)', fontSize: '0.875rem'}}>
            {t('settings.opmlDesc')}
          </p>
          <div className="form-row">
            <button onClick={handleExportOPML} className="btn btn-secondary">
              <Download size={16} />
              {t('settings.exportOPML')}
            </button>
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={importing}
              className="btn btn-secondary"
            >
              <Upload size={16} />
              {importing ? t('settings.importing') : t('settings.importOPML')}
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
          <h3>{t('settings.exportSaved')}</h3>
          <p style={{color: 'var(--text-secondary)', marginBottom: 'var(--space-4)', fontSize: '0.875rem'}}>
            {t('settings.exportSavedDesc')}
          </p>
          <div className="form-row">
            <button onClick={handleExportJSON} className="btn btn-secondary">
              <Download size={16} />
              {t('settings.exportJSON')}
            </button>
            <button onClick={handleExportMarkdown} className="btn btn-secondary">
              <Download size={16} />
              {t('settings.exportMarkdown')}
            </button>
          </div>
        </section>
        </div>
      </main>
    </div>
    </div>
  )
}
