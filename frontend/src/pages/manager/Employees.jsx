import { EmployeeTable } from '../../components/employee/EmployeeTable';

export function Employees({ employees = [] }) {
  return (
    <section className="section-block">
      <div className="section-header">
        <div>
          <h2 className="section-title">Employee Management</h2>
          <p className="record-card__subtext">View people, roles, and current availability at a glance.</p>
        </div>
      </div>

      {employees.length > 0 ? (
        <EmployeeTable employees={employees} />
      ) : (
        <p className="empty-state">No employees found.</p>
      )}
    </section>
  );
}
