import {useTranslation} from 'react-i18next'

interface QualityPillProps {
  score: number
}

export function QualityPill({score}: QualityPillProps) {
  const {t} = useTranslation()
  const label = score < 20 ? 'low' : score < 30 ? 'medium' : 'high'

  return (
    <span
      className={`quality-pill ${label}`}
      title={t('articles.qualityScore', { score })}
    >
      {score}
    </span>
  )
}
