import DOMPurify from 'dompurify'
import {ExternalLink, Check, X, Clock, Save, Sparkles, FileText, RefreshCw} from 'lucide-react'
import {Article} from '../api'
import {StatusBadge} from './StatusBadge'

interface ArticleReaderProps {
  article: Article | null
  feedName: string
  isSummarizing: boolean
  onAccept: (id: number) => void
  onReject: (id: number) => void
  onSnooze: (id: number) => void
  onSave: (id: number) => void
  onGenerateSummary: (id: number) => void
  onRefresh: (id: number) => void
  onOpenExternal: (url: string) => void
  onBack?: () => void
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

export function ArticleReader({
  article,
  feedName,
  isSummarizing,
  onAccept,
  onReject,
  onSnooze,
  onSave,
  onGenerateSummary,
  onRefresh,
  onOpenExternal,
  onBack,
}: ArticleReaderProps) {
  if (!article) {
    return (
      <div className="article-reader-col">
        <div className="article-reader-empty">
          <FileText />
          <p>Select an article to read</p>
        </div>
      </div>
    )
  }

  const hasSummary = article.summary && article.summary.length > 0
  const hasContent = article.content && article.content.length > 0
  const cleanContent = hasContent
    ? DOMPurify.sanitize(article.content)
    : ''

  return (
    <div className="article-reader-col">
      <div className="article-reader">
        {/* Meta */}
        <div className="article-reader-meta">
          <span>{feedName}</span>
          {article.author && <span>{article.author}</span>}
          <span>{formatDate(article.published)}</span>
          {isSummarizing && <span className="summarizing-dot" title="Generating summary…" />}
        </div>

        {/* Title */}
        <h1 className="article-reader-title">{article.title}</h1>

        {/* Summary lead — shown prominently */}
        {hasSummary && (
          <div className="article-reader-summary">
            {article.summary}
          </div>
        )}

        {/* Content */}
        {hasContent ? (
          <div
            className="article-reader-content"
            dangerouslySetInnerHTML={{__html: cleanContent}}
          />
        ) : (
          <div style={{color: 'var(--text-secondary)', fontSize: '14px'}}>
            <p>This article has no content.</p>
            {article.link && (
              <a
                href={article.link}
                target="_blank"
                rel="noopener noreferrer"
                style={{color: 'var(--accent)', display: 'inline-flex', alignItems: 'center', gap: '4px', marginTop: '8px'}}
              >
                <ExternalLink size={14} />
                Open original article
              </a>
            )}
          </div>
        )}

        {/* Status indicator */}
        {article.status && article.status !== 'unread' && (
          <div style={{marginTop: '24px', display: 'flex', alignItems: 'center', gap: '8px'}}>
            <StatusBadge status={article.status} />
          </div>
        )}
      </div>

      {/* Sticky Action Bar */}
      <div className="action-bar" style={{maxWidth: '680px', margin: '0 auto'}}>
        <button
          className="btn btn-primary"
          disabled={article.status === 'accepted'}
          onClick={() => onAccept(article.id)}
        >
          <Check size={14} />
          Accept
        </button>
        <button
          className="btn btn-danger"
          disabled={article.status === 'rejected'}
          onClick={() => onReject(article.id)}
        >
          <X size={14} />
          Reject
        </button>
        <button
          className="btn btn-secondary"
          disabled={article.status === 'snoozed'}
          onClick={() => onSnooze(article.id)}
        >
          <Clock size={14} />
          Snooze
        </button>
        {!article.is_saved && (
          <button
            className="btn btn-secondary"
            onClick={() => onSave(article.id)}
          >
            <Save size={14} />
            Save as Note
          </button>
        )}
        {!hasSummary && (
          <button
            className="btn btn-secondary"
            disabled={isSummarizing}
            onClick={() => onGenerateSummary(article.id)}
          >
            <Sparkles size={14} />
            {isSummarizing ? 'Summarizing…' : 'Generate Summary'}
          </button>
        )}
        {hasContent && article.content.length < 300 && (
          <button
            className="btn btn-secondary"
            onClick={() => onRefresh(article.id)}
          >
            <RefreshCw size={14} />
            Fetch Full Article
          </button>
        )}
        <a
          href={article.link}
          target="_blank"
          rel="noopener noreferrer"
          className="btn btn-ghost"
          style={{marginLeft: 'auto'}}
        >
          <ExternalLink size={14} />
          Open Original
        </a>
      </div>
    </div>
  )
}
