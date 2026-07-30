import { roleNavigation } from '../../config';
import { Icon } from '../common/Icon';
import { getInitials } from '../../utils/helpers';

export function Sidebar({ role = 'employee', activeView, onNavigate, onLogout, userName, userEmail, open }) {
  const items = roleNavigation[role] || roleNavigation.employee;
  const sectionLabel = 'Main';
  const initials = getInitials(userName ?? 'User');

  return (
    <aside id="primary-sidebar" className={`sidebar ${open ? 'open' : ''}`} aria-label="Application navigation">
      <div className="sidebar__brand">
        <div className="brand-mark">
          <Icon name="calendar" />
        </div>
        <div>
          <p className="sidebar__eyebrow">Leave Management</p>
          <h1 className="sidebar__title">{role === 'employee' ? 'Employee Portal' : role === 'admin' ? 'Admin Portal' : 'Manager Portal'}</h1>
        </div>
      </div>

      <div className="sidebar__section-label">{sectionLabel}</div>

      <nav className="sidebar__nav" aria-label="Primary">
        {items.map((item) => (
          <button
            key={item.id}
            type="button"
            className={`nav-item ${activeView === item.id ? 'active' : ''}`}
            onClick={() => onNavigate(item.id)}
          >
            <span className="nav-item__icon-wrap">
              <Icon name={item.icon} />
            </span>
            <span className="nav-item__label">{item.label}</span>
          </button>
        ))}
      </nav>

      <nav className="sidebar__nav sidebar__nav--compact sidebar__section-label--spaced" aria-label="Session">
        <button type="button" className="nav-item nav-item--compact" onClick={onLogout}>
          <span className="nav-item__icon-wrap">
            <Icon name="logout" />
          </span>
          <span className="nav-item__label nav-item__label--danger">Sign Out</span>
        </button>
      </nav>

      <div className="sidebar__footer">
        <div className="sidebar__profile-card">
          <div className="sidebar__avatar">{initials}</div>
          <div>
            <strong>{userName ?? 'User'}</strong>
            <span>{userEmail ?? 'user@company.com'}</span>
          </div>
        </div>
      </div>
    </aside>
  );
}
