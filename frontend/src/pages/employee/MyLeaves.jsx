import { useState } from 'react';
import { LeaveCard } from '../../components/leave/LeaveCard';
import { LEAVE_FILTERS } from '../../utils/constants';

export function MyLeaves({ leaves = [], onDelete }) {
  const [filter, setFilter] = useState('All');
  const [deletingId, setDeletingId] = useState('');

  async function deleteRequest(leave) {
    if (!window.confirm('Delete this pending leave request? This cannot be undone.')) return;
    setDeletingId(leave.id);
    try {
      await onDelete?.(leave);
    } catch (error) {
      window.alert(error.message || 'Unable to delete the leave request.');
    } finally {
      setDeletingId('');
    }
  }

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
          filtered.map((leave) => (
            <LeaveCard
              key={leave.id}
              leave={leave}
              onDelete={onDelete ? deleteRequest : undefined}
              deleting={deletingId === leave.id}
            />
          ))
        ) : (
          <p className="empty-state">No leave requests found for this filter.</p>
        )}
      </div>
    </section>
  );
}
