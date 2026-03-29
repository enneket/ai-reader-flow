import {Article} from '../api'
import {QualityPill} from './QualityPill'
import {StatusBadge} from './StatusBadge'

interface ArticleCardProps {
  article: Article
  feedName: string
  isSelected: boolean
  isLead: boolean
  isSummarizing: boolean
  onClick: () => void
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}

// Strip HTML tags from text
function stripHtml(html: string): string {
  if (!html) return ''
  // Replace BR tags with newlines
  let text = html.replace(/<br\s*\/?>/gi, '\n')
  // Strip remaining HTML tags
  text = text.replace(/<[^>]*>/g, '')
  // Decode common HTML entities
  text = text.replace(/&nbsp;/g, ' ')
  text = text.replace(/&amp;/g, '&')
  text = text.replace(/&lt;/g, '<')
  text = text.replace(/&gt;/g, '>')
  text = text.replace(/&quot;/g, '"')
  text = text.replace(/&#39;/g, "'")
  return text.trim()
}

function truncate(text: string, maxChars: number): string {
  if (!text) return ''
  // Strip HTML first
  const clean = stripHtml(text)
  if (clean.length <= maxChars) return clean
  // Don't cut mid-word
  const truncated = clean.substring(0, maxChars)
  const lastSpace = truncated.lastIndexOf(' ')
  return (lastSpace > maxChars * 0.8 ? truncated.substring(0, lastSpace) : truncated) + '…'
}

export function ArticleCard({
  article,
  feedName,
  isSelected,
  isLead,
  isSummarizing,
  onClick,
}: ArticleCardProps) {
  const hasSummary = article.summary && article.summary.length > 0
  const deck = hasSummary
    ? truncate(article.summary, 120)
    : truncate(article.content || '', 120)

  return (
    <div
      className={`article-card ${isSelected ? 'selected' : ''} ${isLead ? 'lead' : ''}`}
      onClick={onClick}
    >
      <div className="article-card-meta">
        <span>{feedName}</span>
        <span>{formatDate(article.published)}</span>
      </div>

      <div className="article-card-title">{article.title}</div>

      {deck && <div className="article-card-deck">{deck}</div>}

      <div className="article-card-footer">
        {isSummarizing ? (
          <span className="summarizing-dot" title="AI is generating a summary…" />
        ) : hasSummary ? (
          <span className="badge badge-ai">AI</span>
        ) : null}

        {article.status && article.status !== 'unread' && (
          <StatusBadge status={article.status} />
        )}

        {article.is_filtered && (
          <span className="badge badge-filtered">Filtered</span>
        )}

        {article.is_saved && (
          <span className="badge badge-saved">Saved</span>
        )}

        <QualityPill score={article.quality_score || 0} />
      </div>
    </div>
  )
}
