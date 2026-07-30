import { StatCard } from '../../components/dashboard/StatCard';

export function ManagerDashboard({ stats = [], recentActivity = [] }) {
  return (
    <>
      <section className="section-block section-block--tight">
        <div className="section-heading">
          <h3>Team summary</h3>
        </div>
        <div className="quick-stats-grid">
          {stats.map((stat) => (
            <StatCard key={stat.label} {...stat} />
          ))}
        </div>
      </section>

      <section className="dashboard-card">
        <div className="dashboard-card__head">
          <h3>Pending requests</h3>
        </div>
        <div className="request-list">
          {recentActivity.length > 0 ? recentActivity.map((item) => (
            <article key={item.title} className="request-item">
              <div className="request-item__meta">
                <span>{item.date}</span>
                <span className="request-item__status">{item.tag}</span>
              </div>
              <h4>{item.title}</h4>
              <p>{item.summary}</p>
            </article>
          )) : <p className="empty-state">No pending requests.</p>}
        </div>
      </section>
    </>
  );
}
