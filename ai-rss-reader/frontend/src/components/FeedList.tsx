import {useState, useEffect} from 'react'
import {Link} from 'react-router-dom'
import {GetFeeds, AddFeed, DeleteFeed, RefreshAllFeeds} from '../../wailsjs/go/main/App'
import {models} from '../../wailsjs/go/models'

export function FeedList() {
    const [feeds, setFeeds] = useState<models.Feed[]>([])
    const [newFeedUrl, setNewFeedUrl] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    const loadFeeds = async () => {
        try {
            const data = await GetFeeds()
            setFeeds(data || [])
        } catch (err: any) {
            setError(err.message || 'Failed to load feeds')
        }
    }

    useEffect(() => {
        loadFeeds()
    }, [])

    const handleAddFeed = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!newFeedUrl.trim()) return

        setLoading(true)
        setError('')
        try {
            await AddFeed(newFeedUrl)
            setNewFeedUrl('')
            await loadFeeds()
        } catch (err: any) {
            setError(err.message || 'Failed to add feed')
        } finally {
            setLoading(false)
        }
    }

    const handleDeleteFeed = async (id: number) => {
        try {
            await DeleteFeed(id)
            await loadFeeds()
        } catch (err: any) {
            setError(err.message || 'Failed to delete feed')
        }
    }

    const handleRefreshAll = async () => {
        setLoading(true)
        setError('')
        try {
            await RefreshAllFeeds()
            await loadFeeds()
        } catch (err: any) {
            setError(err.message || 'Failed to refresh feeds')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="feed-list">
            <div className="feed-header">
                <h2>RSS Feeds</h2>
                <button onClick={handleRefreshAll} disabled={loading} className="btn-refresh">
                    {loading ? 'Refreshing...' : 'Refresh All'}
                </button>
            </div>

            {error && <div className="error">{error}</div>}

            <form onSubmit={handleAddFeed} className="add-feed-form">
                <input
                    type="url"
                    value={newFeedUrl}
                    onChange={(e) => setNewFeedUrl(e.target.value)}
                    placeholder="Enter RSS feed URL (e.g., https://news.ycombinator.com/rss)"
                    required
                />
                <button type="submit" disabled={loading}>Add Feed</button>
            </form>

            {feeds.length === 0 ? (
                <p className="empty-state">No feeds yet. Add your first RSS feed above.</p>
            ) : (
                <ul className="feeds">
                    {feeds.map((feed) => (
                        <li key={feed.id} className="feed-item">
                            <div className="feed-info">
                                <h3>{feed.title}</h3>
                                <p className="feed-url">{feed.url}</p>
                                {feed.description && (
                                    <p className="feed-desc">{feed.description}</p>
                                )}
                            </div>
                            <div className="feed-actions">
                                <Link to={`/articles/${feed.id}`} className="btn-view">
                                    View Articles
                                </Link>
                                <button
                                    onClick={() => handleDeleteFeed(feed.id)}
                                    className="btn-delete"
                                >
                                    Delete
                                </button>
                            </div>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    )
}
