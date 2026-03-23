interface StatusBadgeProps {
  status: string
}

const LABELS: Record<string, string> = {
  accepted: 'Accepted',
  rejected: 'Rejected',
  snoozed: 'Snoozed',
  saved: 'Saved',
  filtered: 'Filtered',
}

export function StatusBadge({status}: StatusBadgeProps) {
  const label = LABELS[status] || status
  return (
    <span className={`badge badge-${status}`}>
      {label}
    </span>
  )
}
