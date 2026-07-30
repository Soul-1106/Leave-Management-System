import { Avatar } from '../common/Avatar';
import { LeaveStatusBadge } from './LeaveStatusBadge';

export function ApprovalCard({ item, onDecision, onViewAttachment, actionable = false }) {
  const isPending = String(item.status ?? 'pending').toLowerCase() === 'pending';

  return (
    <article className={`approval-card ${isPending ? 'approval-card--pending' : ''}`}>
      <div className="approval-card__header">
        <Avatar name={item.name} size="md" />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="approval-card__headline">
            <h3>{item.name}</h3>
            <span className="mono">{item.id}</span>
          </div>
          <p className="record-card__subtext">
            {item.dept} | {item.role}
          </p>
        </div>
        {item.status && <LeaveStatusBadge status={item.status} />}
      </div>

      <div className="approval-card__meta">
        <div>
          <span className="meta-label">Leave type</span>
          <strong>{item.leave ?? item.type}</strong>
        </div>
        <div>
          <span className="meta-label">Dates</span>
          <strong>{item.dates}</strong>
        </div>
        <div>
          <span className="meta-label">Requested</span>
          <strong>{item.requested ?? '—'}</strong>
        </div>
        <div>
          <span className="meta-label">Days</span>
          <strong>{item.days ?? '—'}</strong>
        </div>
      </div>

      {item.reason && <p className="record-card__body">Reason: {item.reason}</p>}

      {item.hasAttachment && (
        <button type="button" className="secondary-button attachment-button" onClick={() => onViewAttachment?.(item)}>
          View document{item.attachmentName ? `: ${item.attachmentName}` : ''}
        </button>
      )}

      {actionable && isPending && (
        <div className="approval-card__actions">
          <button type="button" className="success-button" onClick={() => onDecision?.(item, 'approved')}>Approve</button>
          <button type="button" className="danger-button" onClick={() => onDecision?.(item, 'rejected')}>Reject</button>
        </div>
      )}
    </article>
  );
}
