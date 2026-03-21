import {useState, useEffect} from 'react'
import {useParams} from 'react-router-dom'
import {GetArticles, GetFeeds, GenerateSummary, CreateNote, FilterAllArticles} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function ArticleList() {
    const [articles, setArticles] = useState<models.Article[]>([])
    const [feeds, setFeeds] = useState<models.Feed[]>([])
    const [selectedFeedId, setSelectedFeedId] = useState<number>(0)
    const [filterMode, setFilterMode] = useState('all')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')
    const [generatingSummary, setGeneratingSummary] = useState<number | null>(null)
    const params = useParams()

    useEffect(() => {
        const fid = params.feedId ? parseInt(params.feedId) : 0
        setSelectedFeedId(fid)
    }, [params.feedId])

    useEffect(() => {
        loadFeeds()
    }, [])

    useEffect(() => {
        loadArticles()
    }, [selectedFeedId, filterMode])

    const loadFeeds = async () => {
        try {
            const data = await GetFeeds()
            setFeeds(data || [])
        } catch (err: any) {
            console.error('Failed to load feeds:', err)
        }
    }

    const loadArticles = async () => {
        setLoading(true)
        setError('')
        try {
            const data = await GetArticles(selectedFeedId, filterMode)
            setArticles(data || [])
        } catch (err: any) {
            setError(err.message || 'Failed to load articles')
        } finally {
            setLoading(false)
        }
    }

    const handleGenerateSummary = async (articleId: number) => {
        setGeneratingSummary(articleId)
        try {
            await GenerateSummary(articleId)
            await loadArticles()
        } catch (err: any) {
            setError(err.message || 'Failed to generate summary')
        } finally {
            setGeneratingSummary(null)
        }
    }

    const handleCreateNote = async (articleId: number) => {
        const article = articles.find(a => a.id === articleId)
        if (!article) return

        try {
            await CreateNote(articleId, article.summary || article.content)
            await loadArticles()
        } catch (err: any) {
            setError(err.message || 'Failed to create note')
        }
    }

    const handleFilterAll = async () => {
        setLoading(true)
        try {
            await FilterAllArticles()
            await loadArticles()
        } catch (err: any) {
            setError(err.message || 'Failed to filter articles')
        } finally {
            setLoading(false)
        }
    }

    const getFeedTitle = (feedId: number) => {
        const feed = feeds.find(f => f.id === feedId)
        return feed ? feed.title : 'Unknown Feed'
    }

    const formatDate = (dateStr: string) => {
        if (!dateStr) return ''
        const date = new Date(dateStr)
        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        })
    }

    return (
        <div className="article-list">
            <div className="article-header">
                <h2>Articles</h2>
                <div className="article-filters">
                    <select
                        value={selectedFeedId}
                        onChange={(e) => setSelectedFeedId(parseInt(e.target.value))}
                    >
                        <option value={0}>All Feeds</option>
                        {feeds.map((feed) => (
                            <option key={feed.id} value={feed.id}>{feed.title}</option>
                        ))}
                    </select>
                    <select
                        value={filterMode}
                        onChange={(e) => setFilterMode(e.target.value)}
                    >
                        <option value="all">All Articles</option>
                        <option value="filtered">Filtered (AI)</option>
                        <option value="saved">Saved</option>
                    </select>
                    <button onClick={handleFilterAll} disabled={loading} className="btn-filter">
                        Filter with AI
                    </button>
                </div>
            </div>

            {error && <div className="error">{error}</div>}

            {loading && <div className="loading">Loading...</div>}

            {articles.length === 0 && !loading ? (
                <p className="empty-state">
                    {filterMode === 'all'
                        ? 'No articles yet. Add some RSS feeds first.'
                        : `No ${filterMode} articles.`}
                </p>
            ) : (
                <ul className="articles">
                    {articles.map((article) => (
                        <li key={article.id} className={`article-item ${article.is_filtered ? 'filtered' : ''} ${article.is_saved ? 'saved' : ''}`}>
                            <div className="article-meta">
                                <span className="article-feed">{getFeedTitle(article.feed_id)}</span>
                                <span className="article-date">{formatDate(article.published)}</span>
                            </div>
                            <h3>
                                <a href={article.link} target="_blank" rel="noopener noreferrer">
                                    {article.title}
                                </a>
                            </h3>
                            {article.author && (
                                <p className="article-author">By {article.author}</p>
                            )}
                            {article.summary && (
                                <p className="article-summary">{article.summary.substring(0, 200)}...</p>
                            )}
                            {article.is_filtered && (
                                <span className="badge filtered-badge">Filtered</span>
                            )}
                            {article.is_saved && (
                                <span className="badge saved-badge">Saved</span>
                            )}
                            <div className="article-actions">
                                <button
                                    onClick={() => handleGenerateSummary(article.id)}
                                    disabled={generatingSummary === article.id}
                                    className="btn-summary"
                                >
                                    {generatingSummary === article.id ? 'Generating...' : 'AI Summary'}
                                </button>
                                {!article.is_saved && (
                                    <button
                                        onClick={() => handleCreateNote(article.id)}
                                        className="btn-save"
                                    >
                                        Save as Note
                                    </button>
                                )}
                            </div>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
