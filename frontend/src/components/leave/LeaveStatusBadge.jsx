export function LeaveStatusBadge({ status }) {
  return <span className={`status-pill status-pill--${String(status).toLowerCase()}`}>{status}</span>;
}