interface QualityPillProps {
  score: number
}

export function QualityPill({score}: QualityPillProps) {
  const label = score < 20 ? 'low' : score < 30 ? 'medium' : 'high'

  return (
    <span
      className={`quality-pill ${label}`}
      title={`Quality score: ${score}/40`}
    >
      {score}
    </span>
  )
}
