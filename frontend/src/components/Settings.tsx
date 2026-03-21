import {useState, useEffect} from 'react'
import {Save, Plus, Trash2} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {changeLanguage} from '../i18n'
import i18n from '../i18n'
import {GetAIConfig, SaveAIConfig, GetFilterRules, AddFilterRule, DeleteFilterRule} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function Settings() {
  const {t} = useTranslation()
  const [aiConfig, setAIConfig] = useState<models.AIProviderConfig>({
    provider: 'openai',
    api_key: '',
    base_url: 'https://api.openai.com/v1',
    model: 'gpt-3.5-turbo',
    max_tokens: 500
  })
  const [filterRules, setFilterRules] = useState<models.FilterRule[]>([])
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
      const config = await GetAIConfig()
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
      const rules = await GetFilterRules()
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
      await SaveAIConfig(provider, apiKey, baseURL, model, maxTokens)
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
      await AddFilterRule(ruleType, ruleValue, ruleAction)
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
      await DeleteFilterRule(id)
      await loadFilterRules()
    } catch (err: any) {
      setError(err.message || 'Failed to delete filter rule')
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">{t('settings.title')}</h1>
      </header>

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
            <select
              value={i18n.language}
              onChange={(e) => changeLanguage(e.target.value as 'en' | 'zh')}
              className="form-select"
            >
              <option value="en">{t('settings.english')}</option>
              <option value="zh">{t('settings.chinese')}</option>
            </select>
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
                  <span className="badge" style={{background: 'var(--bg-surface)'}}>
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
      </div>
    </>
  )
}
