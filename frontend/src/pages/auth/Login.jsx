import { useState } from 'react';
import { Icon } from '../../components/common/Icon';
import { LOGIN_HIGHLIGHTS } from '../../utils/constants';

export function Login({ onLogin }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  return (
    <main className="login-page">
      <section className="login-card">
        <div className="login-card__brand">
          <div className="brand-mark brand-mark--large"><Icon name="calendar" /></div>
          <p className="page-kicker">Leave Management System</p>
          <h1 className="login-title">Login to Leave Management</h1>
          <p className="login-subtitle">Sign in with your Supabase account.</p>
        </div>
        <form className="login-form" onSubmit={async (event) => {
          event.preventDefault();
          setError('');
          setSubmitting(true);
          try {
            await onLogin(email, password);
          } catch (loginError) {
            setError(loginError.message || 'Login failed');
          } finally {
            setSubmitting(false);
          }
        }}>
          <div className="field-group">
            <label className="field-label">Email <span className="required-mark">*</span></label>
            <input className="input" type="email" value={email} onChange={(event) => setEmail(event.target.value)} required autoComplete="email" />
          </div>
          <div className="field-group">
            <label className="field-label">Password <span className="required-mark">*</span></label>
            <input className="input" type="password" value={password} onChange={(event) => setPassword(event.target.value)} required autoComplete="current-password" />
          </div>
          {error && <p className="record-card__body" role="alert">{error}</p>}
          <button type="submit" className="primary-button primary-button--full" disabled={submitting}>
            {submitting ? 'Signing in…' : 'Login'}
          </button>
        </form>
        <div className="login-card__notes">
          {LOGIN_HIGHLIGHTS.map((item) => <div key={item} className="note-chip"><Icon name="check" /><span>{item}</span></div>)}
        </div>
      </section>
    </main>
  );
}
