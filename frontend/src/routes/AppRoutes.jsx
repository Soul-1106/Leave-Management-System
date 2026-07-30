import { useCallback, useEffect, useState } from 'react';
import { Login } from '../pages/auth/Login';
import { Sidebar } from '../components/layout/Sidebar';
import { Navbar } from '../components/layout/Navbar';
import { Layout } from '../components/layout/Layout';
import { EmployeeDashboard, UpcomingLeaveCard } from '../pages/employee/Dashboard';
import { MyLeaves } from '../pages/employee/MyLeaves';
import { ApplyLeave } from '../pages/employee/ApplyLeave';
import { ManagerDashboard } from '../pages/manager/Dashboard';
import { Employees } from '../pages/manager/Employees';
import { LeaveApprovals } from '../pages/manager/LeaveApprovals';
import { UserManagement } from '../pages/admin/UserManagement';
import { NotFound } from '../pages/NotFound';
import { getInitials } from '../utils/helpers';
import { useAuth } from '../hooks/useAuth';
import { apiDelete, apiGet, apiGetStrict, apiPatch, apiPostStrict, apiUploadStrict } from '../services/api';
import { roleNavigation } from '../config';

const initialApplication = {
  leaveType: 'Casual Leave',
  fromDate: '',
  toDate: '',
  reason: '',
  attachment: null,
  confirmation: false,
};

export function AppRoutes() {
  const { user, signIn, signOut } = useAuth();
  const [activeView, setActiveView] = useState('dashboard');
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [application, setApplication] = useState(initialApplication);
  const [toast, setToast] = useState(null);

  const [appData, setAppData] = useState({
    stats: [],
    balances: [],
    myLeaves: [],
    approvals: [],
    approvalHistory: [],
    employees: [],
    adminUsers: [],
    adminBalances: [],
    departments: [],
    loading: true,
  });

  const refreshData = useCallback(async () => {
    const manager = user.role === 'manager';
    const employee = user.role === 'employee';
    const admin = user.role === 'admin';
    const [stats, balances] = await Promise.all([
      apiGet('/dashboard/stats'),
      employee ? apiGet('/leaves/balances') : Promise.resolve([]),
    ]);
    const [myLeaves, approvals, approvalHistory, employees] = manager
      ? await Promise.all([
          Promise.resolve([]),
          apiGet('/leaves/approvals'),
          apiGet('/leaves/history'),
          apiGet('/employees'),
        ])
      : await Promise.all([
          apiGet('/leaves/my'),
          Promise.resolve([]),
          Promise.resolve([]),
          Promise.resolve([]),
        ]);
    const [adminUsers, adminBalances, departments] = admin
      ? await Promise.all([apiGet('/admin/users'), apiGet('/admin/balances'), apiGet('/admin/departments')])
      : [[], [], []];
    setAppData({ stats: stats || [], balances: balances || [], myLeaves: myLeaves || [], approvals: approvals || [], approvalHistory: approvalHistory || [], employees: employees || [], adminUsers: adminUsers || [], adminBalances: adminBalances || [], departments: departments || [], loading: false });
  }, [user.role]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), 4000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    document.body.classList.toggle('drawer-open', drawerOpen);
    const onKeyDown = (event) => {
      if (event.key === 'Escape') setDrawerOpen(false);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.classList.remove('drawer-open');
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [drawerOpen]);

  useEffect(() => {
    if (!user.authenticated) return undefined;

    const initialRefresh = window.setTimeout(refreshData, 0);

    const refreshOnFocus = () => refreshData();
    window.addEventListener('focus', refreshOnFocus);
    return () => {
      window.clearTimeout(initialRefresh);
      window.removeEventListener('focus', refreshOnFocus);
    };
  }, [user.authenticated, activeView, refreshData]);

  if (user.loading) return <div className="loading-state">Checking session...</div>;
  if (!user.authenticated) return <Login onLogin={signIn} />;

  const title = getTitle(activeView, user.role);

  return (
    <Layout
      drawerOpen={drawerOpen}
      onCloseDrawer={() => setDrawerOpen(false)}
      sidebar={
        <Sidebar
          role={user.role}
          activeView={activeView}
          open={drawerOpen}
          onNavigate={(view) => {
            setActiveView(view);
            setDrawerOpen(false);
          }}
          onLogout={() => {
            signOut();
            setActiveView('dashboard');
          }}
          userName={user.name}
          userEmail={user.email}
        />
      }
      navbar={
        <Navbar
          title={title}
          profileInitials={getInitials(user.name)}
          menuOpen={drawerOpen}
          onMenuClick={() => setDrawerOpen((open) => !open)}
        />
      }
    >
      {renderPage(activeView, user.role, {
        setApplication,
        setToast,
        application,
        appData,
        refreshData,
      })}

      {toast && (
        <div className="toast toast--success" role="status" aria-live="polite">
          <div>
            <strong>{toast.title}</strong>
            <p>{toast.message}</p>
          </div>
        </div>
      )}
    </Layout>
  );
}

function renderPage(view, role, { setApplication, setToast, application, appData, refreshData }) {
  if (appData.loading) {
    return <div className="loading-state">Loading data...</div>;
  }

  const knownViews = new Set([
    ...(roleNavigation[role] || []).map((item) => item.id),
  ]);

  if (!knownViews.has(view)) {
    return <NotFound />;
  }

  if (view === 'my-leaves') {
    return <MyLeaves leaves={appData.myLeaves} onDelete={async (leave) => {
      await apiDelete(`/leaves/my/${leave.id}`);
      await refreshData();
      setToast({
        title: 'Leave request deleted',
        message: 'The pending request has been removed.',
      });
    }} />;
  }

  if (view === 'apply') {
    return (
      <ApplyLeave
        application={application}
        balances={appData.balances}
        onChange={setApplication}
        onSubmit={async () => {
          if (application.attachment) {
            const allowedTypes = new Set(['application/pdf', 'image/jpeg', 'image/png']);
            if (!allowedTypes.has(application.attachment.type)) {
              throw new Error('Attachment must be a PDF, JPEG, or PNG file.');
            }
            if (application.attachment.size > 5 * 1024 * 1024) {
              throw new Error('Attachment must be 5 MB or smaller.');
            }
          }
          const result = await apiPostStrict('/leaves/my', {
            leaveType: application.leaveType,
            startDate: application.fromDate,
            endDate: application.toDate,
            reason: application.reason,
          });
          if (result) {
            if (application.attachment) {
              await apiUploadStrict(`/attachments/${result.id}`, application.attachment);
            }
            await refreshData();
            setApplication(initialApplication);
            setToast({
            title: 'Leave application submitted',
            message: application.attachment
              ? 'Your request and supporting document are ready for review.'
              : 'Your request has been queued for review.',
            });
          }
        }}
      />
    );
  }

  if (view === 'employees' && role !== 'employee') {
    return <Employees employees={appData.employees} />;
  }

  if (view === 'approvals' && role !== 'employee') {
    return <LeaveApprovals approvals={appData.approvals} actionable onDecision={async (item, status) => {
      await apiPatch(`/leaves/${item.leaveId}/decision`, { status });
      await refreshData();
    }} onViewAttachment={async (item) => {
      const popup = window.open('', '_blank');
      try {
        const attachment = await apiGetStrict(`/attachments/${item.leaveId}`);
        if (popup) {
          popup.opener = null;
          popup.location = attachment.url;
        } else {
          window.location.assign(attachment.url);
        }
      } catch (error) {
        popup?.close();
        window.alert(error.message || 'Unable to open attachment');
      }
    }} />;
  }

  if (view === 'history' && role === 'manager') {
    return <LeaveApprovals
      title="Decision History"
      approvals={appData.approvalHistory}
      onViewAttachment={async (item) => {
        const popup = window.open('', '_blank');
        try {
          const attachment = await apiGetStrict(`/attachments/${item.leaveId}`);
          if (popup) {
            popup.opener = null;
            popup.location = attachment.url;
          } else {
            window.location.assign(attachment.url);
          }
        } catch (error) {
          popup?.close();
          window.alert(error.message || 'Unable to open attachment');
        }
      }}
    />;
  }

  if (view === 'admin-users' && role === 'admin') {
    return <UserManagement users={appData.adminUsers} balances={appData.adminBalances} departments={appData.departments} onSave={async (form) => {
      const payload = {
        name: form.name,
        email: form.email,
        password: form.password,
        role: form.role,
        employeeId: form.role === 'employee' ? form.employeeId : '',
        designation: form.role === 'employee' ? form.designation : '',
        departmentId: form.role === 'employee' ? form.departmentId : '',
        managerId: form.role === 'employee' ? form.managerId : '',
      };
      if (form.userId) {
        await apiPatch(`/admin/users/${form.userId}`, payload);
      } else {
        await apiPostStrict('/admin/users', payload);
      }
      await refreshData();
      setToast({ title: 'Account saved', message: 'People and access settings are up to date.' });
    }} onSaveBalance={async (balance) => {
      await apiPatch('/admin/balances', balance);
      await refreshData();
      setToast({ title: 'Balance updated', message: `${balance.leaveType} balance has been saved.` });
    }} />;
  }

  if (role === 'employee') {
    return (
      <EmployeeDashboard
        stats={appData.stats}
        upcomingItems={appData.myLeaves.slice(0, 2)}
        renderUpcomingItem={(item) => <UpcomingLeaveCard key={`${item.type}-${item.dates}`} item={item} />}
      />
    );
  }

  return (
    <ManagerDashboard
      stats={appData.stats}
      recentActivity={appData.approvals.slice(0, 5).map((item) => ({
        date: item.requested,
        title: `${item.name} — ${item.leave}`,
        summary: `${item.dates}: ${item.reason}`,
        tag: item.status,
      }))}
    />
  );
}

function getTitle(view, role) {
  if (view === 'admin-users') return 'People & Access';
  if (view === 'employees') return 'Employee Management';
  if (view === 'approvals') return 'Leave Approvals';
  if (view === 'history') return 'Decision History';
  if (view === 'apply') return 'Apply for Leave';
  if (view === 'my-leaves') return 'My Leaves';
  if (role === 'admin') return 'Administration Dashboard';
  return role === 'employee' ? 'Leave Dashboard' : 'Manager Dashboard';
}
