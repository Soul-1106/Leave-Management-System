import { Icon } from '../common/Icon';

export function Navbar({ title, profileInitials, onMenuClick, menuOpen }) {
  return (
    <header className="topbar">
      <div className="topbar__left">
        <button
          type="button"
          className="icon-pill mobile-menu-button"
          aria-label={menuOpen ? 'Close navigation' : 'Open navigation'}
          aria-expanded={menuOpen}
          aria-controls="primary-sidebar"
          onClick={onMenuClick}
        >
          <Icon name={menuOpen ? 'close' : 'menu'} />
        </button>
        <div>
          <h1 className="page-title">{title}</h1>
        </div>
      </div>

      <div className="topbar__actions">
        <span className="avatar avatar--sm" aria-label="Signed-in user">
          {profileInitials}
        </span>
      </div>
    </header>
  );
}
