export const roleNavigation = {
  employee: [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'my-leaves', label: 'My Leaves', icon: 'leaves' },
    { id: 'apply', label: 'Apply Leave', icon: 'apply' },
  ],
  manager: [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'approvals', label: 'Leave Approvals', icon: 'approvals' },
    { id: 'employees', label: 'My Team', icon: 'employees' },
  ],
  admin: [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'admin-users', label: 'People & Access', icon: 'employees' },
  ],
};

export function formatBalance(value, total) {
  return `${value}/${total} days`;
}
