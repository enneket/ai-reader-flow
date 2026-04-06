import {useTranslation} from 'react-i18next'

interface StatusBadgeProps {
  status: string
}

export function StatusBadge({status}: StatusBadgeProps) {
  const {t} = useTranslation()
  const label = t(`articles.status.${status}`, {defaultValue: status})
  return (
    <span className={`badge badge-${status}`}>
      {label}
    </span>
  )
}
