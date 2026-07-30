export function StatCard({ label, value, delta, tone }) {
  return (
    <article className={`stat-card stat-card--${tone}`}>
      <span className="meta-label">{label}</span>
      <div className="stat-card__value-row">
        <strong>{value}</strong>
        <span>{delta}</span>
      </div>
      <div className="stat-card__bar" />
    </article>
  );
}