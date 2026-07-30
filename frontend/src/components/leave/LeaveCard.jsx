import { LEAVE_TYPE_TOKENS } from '../../utils/constants';
import { Icon } from '../common/Icon';
import { LeaveStatusBadge } from './LeaveStatusBadge';

export function LeaveCard({ leave }) {
  const type = leave.type ?? leave.leaveType ?? 'Casual Leave';
  const token = LEAVE_TYPE_TOKENS[type] ?? LEAVE_TYPE_TOKENS['Casual Leave'];

  return (
    <article className="record-card record-card--leave">
      <div className="record-card__top">
        <div className={`leave-mark leave-mark--${token.tone}`}>
          <Icon name={token.icon} />
        </div>
        <LeaveStatusBadge status={leave.status} />
      </div>
      <h3>{type}</h3>
      <p className="record-card__subtext">{leave.dates}</p>
      {leave.reason && <p className="record-card__body">Reason: {leave.reason}</p>}
      <div className="record-card__footer">
        {leave.approver && <span>Approver: {leave.approver}</span>}
        {leave.days != null && <span>Days: {leave.days}</span>}
      </div>
    </article>
  );
}
