import { StatCard } from '../../components/dashboard/StatCard';
import { LeaveStatusBadge } from '../../components/leave/LeaveStatusBadge';

export function EmployeeDashboard({ stats, upcomingItems, renderUpcomingItem }) {
  return (
    <>
      <section className="section-block">
        <div className="cards-grid">
          {stats.map((stat) => (
            <StatCard key={stat.label} {...stat} />
          ))}
        </div>
      </section>

      <section className="section-block">
        <h3 className="section-title">Upcoming leaves</h3>
        <div className="approval-list">
          {upcomingItems.length > 0 ? (
            upcomingItems.map(renderUpcomingItem)
          ) : (
            <p className="empty-state">No upcoming leaves scheduled.</p>
          )}
        </div>
      </section>
    </>
  );
}

export function UpcomingLeaveCard({ item }) {
  return (
    <article className="today-card">
      <div className="today-card__header">
        <h3>{item.type}</h3>
        <LeaveStatusBadge status={item.status} />
      </div>
      <p className="today-card__subtitle">{item.dates}</p>
      {item.approver && <p className="record-card__body">Approver: {item.approver}</p>}
      {item.days != null && <p className="record-card__body">Days: {item.days}</p>}
    </article>
  );
}
