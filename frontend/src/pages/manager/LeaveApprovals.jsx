import { ApprovalCard } from '../../components/leave/ApprovalCard';

export function LeaveApprovals({ title = 'Leave Approvals', approvals = [], onDecision, onViewAttachment, actionable = false }) {
  const pendingCount = approvals.filter((a) => String(a.status).toLowerCase() === 'pending').length;
  const approvedCount = approvals.filter((a) => String(a.status).toLowerCase() === 'approved').length;
  const rejectedCount = approvals.filter((a) => String(a.status).toLowerCase() === 'rejected').length;

  return (
    <section className="section-block">
      <div className="section-header">
        <div>
          <h2 className="section-title">{title}</h2>
          <p className="record-card__subtext">
            {actionable ? 'Review pending requests and record a decision.' : 'View the complete leave-request history for your team.'}
          </p>
        </div>
        <div style={{ display: 'flex', gap: 16, fontSize: 14 }}>
          {actionable && <span><strong>Pending:</strong> {pendingCount}</span>}
          <span>
            <strong>Approved:</strong> {approvedCount}
          </span>
          {!actionable && <span><strong>Rejected:</strong> {rejectedCount}</span>}
        </div>
      </div>

      <div className="approval-list">
        {approvals.length > 0 ? (
          approvals.map((item) => <ApprovalCard key={item.leaveId ?? item.id ?? item.name} item={item} onDecision={onDecision} onViewAttachment={onViewAttachment} actionable={actionable} />)
        ) : (
          <p className="empty-state">{actionable ? 'No pending leave requests to review.' : 'No approval history yet.'}</p>
        )}
      </div>
    </section>
  );
}
