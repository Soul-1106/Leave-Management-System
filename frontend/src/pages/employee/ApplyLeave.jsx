import { useState } from 'react';
import { formatBalance } from '../../config';

export function ApplyLeave({ application, onChange, onSubmit, balances = [] }) {
  const currentBalance = balances.find((item) => item.label === application.leaveType) ?? balances[0];
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  return (
    <section className="section-block section-block--form">
      <h2 className="section-title">Apply for Leave</h2>
      <p className="record-card__subtext" style={{ marginBottom: 24 }}>
        Submit a new request with an optional supporting document.
      </p>

      <form
        className="apply-form"
        onSubmit={async (event) => {
          event.preventDefault();
          setError('');
          setSubmitting(true);
          try {
            await onSubmit?.();
          } catch (submitError) {
            setError(submitError.message || 'Unable to submit leave request');
          } finally {
            setSubmitting(false);
          }
        }}
      >
        <div className="field-group">
          <label className="field-label">
            Leave type <span className="required-mark">*</span>
          </label>
          <select
            className="input"
            value={application.leaveType}
            onChange={(event) => onChange({ ...application, leaveType: event.target.value })}
          >
            <option>Casual Leave</option>
            <option>Sick Leave</option>
            <option>Annual Leave</option>
          </select>
          {currentBalance && (
            <p className="field-help">
              Available balance: {formatBalance(currentBalance.used, currentBalance.total)}
            </p>
          )}
        </div>

        <div className="form-row">
          <div className="field-group">
            <label className="field-label">
              From date <span className="required-mark">*</span>
            </label>
            <input
              type="date"
              className="input"
              value={application.fromDate}
              onChange={(event) => onChange({ ...application, fromDate: event.target.value })}
            />
          </div>
          <div className="field-group">
            <label className="field-label">
              To date <span className="required-mark">*</span>
            </label>
            <input
              type="date"
              className="input"
              value={application.toDate}
              onChange={(event) => onChange({ ...application, toDate: event.target.value })}
            />
          </div>
        </div>

        <div className="field-group">
          <label className="field-label">
            Reason for leave <span className="required-mark">*</span>
          </label>
          <textarea
            className="textarea"
            placeholder="Enter reason for leave"
            value={application.reason}
            onChange={(event) => onChange({ ...application, reason: event.target.value })}
          />
        </div>

        <div className="field-group">
          <label className="field-label">Attachment (optional)</label>
          <label className="upload-zone">
            <span>Drag or click to attach a supporting document</span>
            <input
              type="file"
              accept=".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png"
              className="visually-hidden"
              onChange={(event) =>
                onChange({ ...application, attachment: event.target.files?.[0] ?? null })
              }
            />
            {application.attachment && <strong>{application.attachment.name}</strong>}
            <small>PDF, JPEG, or PNG. Maximum 5 MB.</small>
          </label>
        </div>

        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={application.confirmation}
            onChange={(event) => onChange({ ...application, confirmation: event.target.checked })}
          />
          <span>I confirm this information is correct.</span>
        </label>

        {error && <p className="form-error" role="alert">{error}</p>}

        <div className="form-actions">
          <button type="button" className="secondary-button" onClick={() => onChange({ ...application, reason: '', attachment: null, confirmation: false })}>
            Cancel
          </button>
          <button
            type="submit"
            className="primary-button"
            disabled={submitting || !application.confirmation || !application.reason || !application.fromDate || !application.toDate}
          >
            {submitting ? 'Submitting...' : 'Submit application'}
          </button>
        </div>
      </form>
    </section>
  );
}
