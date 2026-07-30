import { useState } from 'react';
import { LeaveCard } from '../../components/leave/LeaveCard';
import { LEAVE_FILTERS } from '../../utils/constants';

export function MyLeaves({ leaves = [] }) {
  const [filter, setFilter] = useState('All');

  const filtered = filter === 'All'
    ? leaves
    : leaves.filter((leave) => String(leave.status).toLowerCase() === filter.toLowerCase());

  return (
    <section className="section-block">
      <h2 className="section-title">My Leaves</h2>
      <p className="record-card__subtext" style={{ marginBottom: 16 }}>
        Track every request with its current approval state.
      </p>

      <div className="toolbar" style={{ marginBottom: 16 }}>
        <div className="toolbar__chips">
          {LEAVE_FILTERS.map((item) => (
            <button
              key={item}
              type="button"
              className={`toolbar__chip ${filter === item ? 'active' : ''}`}
              onClick={() => setFilter(item)}
            >
              {item}
            </button>
          ))}
        </div>
      </div>

      <div className="card-list">
        {filtered.length > 0 ? (
          filtered.map((leave) => <LeaveCard key={`${leave.type}-${leave.dates}`} leave={leave} />)
        ) : (
          <p className="empty-state">No leave requests found for this filter.</p>
        )}
      </div>
    </section>
  );
}
