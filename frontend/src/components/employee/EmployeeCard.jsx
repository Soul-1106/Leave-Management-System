import { Avatar } from '../common/Avatar';
import { getInitials } from '../../utils/helpers';

export function EmployeeCard({ employee }) {
  return (
    <div className="table-row">
      <div className="table-row__name">
        <Avatar initials={getInitials(employee.name)} size="sm" />
        <div>
          <strong>{employee.name}</strong>
          <span className="record-card__subtext">{employee.email ?? employee.dept}</span>
        </div>
      </div>
      <span className="table-row__field mono" data-label="Employee ID">{employee.id}</span>
      <span className="table-row__field" data-label="Role">{employee.role}</span>
      <span className="table-row__field" data-label="Department">{employee.dept}</span>
    </div>
  );
}
