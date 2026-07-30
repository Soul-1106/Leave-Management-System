import { Table } from '../common/Table';
import { EmployeeCard } from './EmployeeCard';

export function EmployeeTable({ employees }) {
  return <Table headers={['Name', 'ID', 'Designation', 'Department']} rows={employees.map((employee) => <EmployeeCard key={employee.id} employee={employee} />)} />;
}
