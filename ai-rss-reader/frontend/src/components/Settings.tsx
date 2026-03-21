import {useState, useEffect} from 'react'
import {GetAIConfig, SaveAIConfig, GetFilterRules, AddFilterRule, DeleteFilterRule} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function Settings() {
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
        <div className="settings">
            <h2>Settings</h2>

            {error && <div className="error">{error}</div>}
            {success && <div className="success">{success}</div>}

            <section className="settings-section">
                <h3>AI Provider Configuration</h3>
                <form onSubmit={handleSaveAIConfig} className="ai-config-form">
                    <div className="form-group">
                        <label>Provider</label>
                        <select value={provider} onChange={(e) => setProvider(e.target.value)}>
                            <option value="openai">OpenAI</option>
                            <option value="claude">Claude</option>
                            <option value="ollama">Ollama (Local)</option>
                        </select>
                    </div>

                    <div className="form-group">
                        <label>API Key</label>
                        <input
                            type="password"
                            value={apiKey}
                            onChange={(e) => setApiKey(e.target.value)}
                            placeholder="Enter API key"
                        />
                    </div>

                    <div className="form-group">
                        <label>Base URL</label>
                        <input
                            type="url"
                            value={baseURL}
                            onChange={(e) => setBaseURL(e.target.value)}
                            placeholder="https://api.openai.com/v1"
                        />
                    </div>

                    <div className="form-group">
                        <label>Model</label>
                        <input
                            type="text"
                            value={model}
                            onChange={(e) => setModel(e.target.value)}
                            placeholder="gpt-3.5-turbo"
                        />
                    </div>

                    <div className="form-group">
                        <label>Max Tokens</label>
                        <input
                            type="number"
                            value={maxTokens}
                            onChange={(e) => setMaxTokens(parseInt(e.target.value))}
                            min={100}
                            max={4000}
                        />
                    </div>

                    <button type="submit" disabled={loading} className="btn-save">
                        {loading ? 'Saving...' : 'Save AI Config'}
                    </button>
                </form>
            </section>

            <section className="settings-section">
                <h3>Filter Rules</h3>
                <form onSubmit={handleAddFilterRule} className="filter-rule-form">
                    <div className="form-row">
                        <select value={ruleType} onChange={(e) => setRuleType(e.target.value)}>
                            <option value="keyword">Keyword</option>
                            <option value="source">Source/Author</option>
                            <option value="ai_preference">AI Preference</option>
                        </select>
                        <input
                            type="text"
                            value={ruleValue}
                            onChange={(e) => setRuleValue(e.target.value)}
                            placeholder="Enter keyword or value"
                            required
                        />
                        <select value={ruleAction} onChange={(e) => setRuleAction(e.target.value)}>
                            <option value="exclude">Exclude</option>
                            <option value="include">Include</option>
                        </select>
                        <button type="submit" disabled={loading}>Add Rule</button>
                    </div>
                </form>

                {filterRules.length === 0 ? (
                    <p className="empty-state">No filter rules defined.</p>
                ) : (
                    <ul className="filter-rules">
                        {filterRules.map((rule) => (
                            <li key={rule.id} className="filter-rule-item">
                                <span className={`badge ${rule.action}-badge`}>
                                    {rule.action}
                                </span>
                                <span className="badge type-badge">{rule.type}</span>
                                <span className="rule-value">{rule.value}</span>
                                <button
                                    onClick={() => handleDeleteFilterRule(rule.id)}
                                    className="btn-delete-small"
                                >
                                    Delete
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </section>
        </div>
    )
}
