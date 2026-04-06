import {useState, useEffect} from 'react'
import DOMPurify from 'dompurify'
import {ExternalLink, Check, X, Clock, Save, Sparkles, FileText, RefreshCw} from 'lucide-react'
import {useTranslation} from 'react-i18next'
import {api, Article} from '../api'

interface ArticleReaderProps {
  article: Article | null
  feedName: string
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
  onAccept,
  onReject,
  onSnooze,
  onSave,
  onGenerateSummary,
  onRefresh,
  onOpenExternal,
  onBack,
}: ArticleReaderProps) {
  const {t} = useTranslation()
  const [showOriginal, setShowOriginal] = useState(false)

  useEffect(() => {
    api.getShowOriginalLanguage().then(data => {
      setShowOriginal(data.show_original_language)
    }).catch(console.error)
  }, [])

  if (!article) {
    return (
      <div className="article-reader-col">
        <div className="article-reader-empty">
          <FileText />
          <p>{t('articles.selectToView')}</p>
        </div>
      </div>
    )
  }

  const hasSummary = article.summary && article.summary.length > 0
  const cleanSummary = hasSummary
    ? DOMPurify.sanitize(article.summary)
    : ''
  const isTranslated = article.is_translated && !!article.translated_content
  const displayContent = (showOriginal || !isTranslated) ? article.content : article.translated_content
  const hasDisplayContent = !!displayContent && displayContent.length > 0

  return (
    <div className="article-reader-col">
      <div className="article-reader">
        {/* Meta */}
        <div className="article-reader-meta">
          <span>{feedName}</span>
          {article.author && <span>{article.author}</span>}
          <span>{formatDate(article.published)}</span>
        </div>

        {/* Title */}
        <div style={{display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '16px'}}>
          <h1 className="article-reader-title">{article.title}</h1>
          {isTranslated && (
            <button
              onClick={() => setShowOriginal(!showOriginal)}
              className="lang-toggle-btn"
              title={showOriginal ? '显示中文翻译' : '显示英文原文'}
            >
              {showOriginal ? '中' : 'EN'}
            </button>
          )}
        </div>

        {/* Summary lead — only shown when no full content */}
        {hasSummary && !hasDisplayContent && (
          <div
            className="article-reader-summary"
            dangerouslySetInnerHTML={{__html: cleanSummary}}
          />
        )}

        {/* Content */}
        {hasDisplayContent ? (
          <div
            className="article-reader-content"
            dangerouslySetInnerHTML={{__html: DOMPurify.sanitize(displayContent)}}
          />
        ) : (
          <div style={{color: 'var(--text-secondary)', fontSize: '14px'}}>
            <p>{t('articles.noContent')}</p>
            {article.link && (
              <a
                href={article.link}
                target="_blank"
                rel="noopener noreferrer"
                style={{color: 'var(--accent)', display: 'inline-flex', alignItems: 'center', gap: '4px', marginTop: '8px'}}
              >
                <ExternalLink size={14} />
                {t('articles.openOriginal')}
              </a>
            )}
          </div>
        )}

      </div>

    </div>
  )
}
