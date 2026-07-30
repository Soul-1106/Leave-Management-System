import { useMemo, useState } from 'react';

const emptyForm = {
  userId: '',
  name: '',
  email: '',
  password: '',
  role: 'employee',
  employeeId: '',
  designation: '',
  departmentId: '',
  managerId: '',
};

export function UserManagement({ users = [], balances = [], departments = [], onSave, onSaveBalance }) {
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const managers = useMemo(() => users.filter((user) => user.role === 'manager'), [users]);

  function update(field, value) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  function edit(user) {
    setError('');
    setForm({ ...emptyForm, ...user, password: '' });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function submit(event) {
    event.preventDefault();
    setError('');
    setSaving(true);
    try {
      await onSave(form);
      setForm(emptyForm);
    } catch (saveError) {
      setError(saveError.message || 'Unable to save user');
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="section-block">
      <div className="section-header">
        <div>
          <h2 className="section-title">People & Access</h2>
          <p className="record-card__subtext">Create accounts, update employee details, and assign employees to managers.</p>
        </div>
      </div>

      <div className="admin-grid">
        <form className="panel-card admin-form" onSubmit={submit}>
          <div className="section-header">
            <h3>{form.userId ? 'Edit account' : 'Add account'}</h3>
            {form.userId && <button type="button" className="ghost-button" onClick={() => setForm(emptyForm)}>Cancel</button>}
          </div>
          <div className="form-row">
            <Field label="Full name"><input className="input" value={form.name} onChange={(event) => update('name', event.target.value)} required /></Field>
            <Field label="Email"><input className="input" type="email" value={form.email} onChange={(event) => update('email', event.target.value)} required /></Field>
          </div>
          <div className="form-row">
            <Field label="Role">
              <select className="input" value={form.role} onChange={(event) => update('role', event.target.value)} disabled={Boolean(form.userId)}>
                <option value="employee">Employee</option>
                <option value="manager">Manager</option>
              </select>
            </Field>
            <Field label={form.userId ? 'New password (optional)' : 'Temporary password'}>
              <input className="input" type="password" minLength={8} value={form.password} onChange={(event) => update('password', event.target.value)} required={!form.userId} />
            </Field>
          </div>
          {form.role === 'employee' && (
            <>
              <div className="form-row">
                <Field label="Employee ID"><input className="input" value={form.employeeId} onChange={(event) => update('employeeId', event.target.value)} required /></Field>
                <Field label="Designation"><input className="input" value={form.designation} onChange={(event) => update('designation', event.target.value)} required /></Field>
              </div>
              <div className="form-row">
                <Field label="Department">
                  <select className="input" value={form.departmentId} onChange={(event) => update('departmentId', event.target.value)}>
                    <option value="">Not assigned</option>
                    {departments.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                  </select>
                </Field>
                <Field label="Manager">
                  <select className="input" value={form.managerId} onChange={(event) => update('managerId', event.target.value)}>
                    <option value="">Not assigned</option>
                    {managers.map((item) => <option key={item.userId} value={item.userId}>{item.name}</option>)}
                  </select>
                </Field>
              </div>
            </>
          )}
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="primary-button" disabled={saving}>{saving ? 'Saving...' : form.userId ? 'Save changes' : 'Create account'}</button>

          {form.userId && form.role === 'employee' && (
            <div className="balance-editor">
              <div>
                <h3>Leave balances</h3>
                <p className="record-card__subtext">Set yearly allocation and used days.</p>
              </div>
              {balances.filter((item) => item.userId === form.userId).map((balance) => (
                <BalanceEditor key={`${balance.leaveTypeId}-${balance.year}`} balance={balance} onSave={onSaveBalance} />
              ))}
            </div>
          )}
        </form>

        <div className="panel-card admin-list">
          <h3>Accounts</h3>
          {users.length === 0 ? <p className="empty-state">No accounts found.</p> : users.map((user) => (
            <article className="admin-user-row" key={user.userId}>
              <div>
                <strong>{user.name}</strong>
                <span>{user.email}</span>
                <small>{user.role === 'employee' ? `${user.employeeId} · ${user.designation || 'Employee'}` : 'Manager'}{user.managerName ? ` · Manager: ${user.managerName}` : ''}</small>
              </div>
              <button type="button" className="secondary-button" onClick={() => edit(user)}>Edit</button>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function Field({ label, children }) {
  return <label className="field-group"><span className="field-label">{label}</span>{children}</label>;
}

function BalanceEditor({ balance, onSave }) {
  const [allocated, setAllocated] = useState(balance.totalAllocated);
  const [used, setUsed] = useState(balance.used);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  async function save() {
    setError('');
    setSaving(true);
    try {
      await onSave({
        userId: balance.userId,
        leaveTypeId: balance.leaveTypeId,
        leaveType: balance.leaveType,
        year: balance.year,
        totalAllocated: Number(allocated),
        used: Number(used),
      });
    } catch (saveError) {
      setError(saveError.message || 'Unable to update balance');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="balance-editor__row">
      <strong>{balance.leaveType} <small>{balance.year}</small></strong>
      <label>Allocated<input className="input" type="number" min="0" value={allocated} onChange={(event) => setAllocated(event.target.value)} /></label>
      <label>Used<input className="input" type="number" min="0" max={allocated} value={used} onChange={(event) => setUsed(event.target.value)} /></label>
      <button type="button" className="secondary-button" onClick={save} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
      {error && <p className="form-error" role="alert">{error}</p>}
    </div>
  );
}
