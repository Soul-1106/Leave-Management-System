export function Layout({ sidebar, navbar, children, drawerOpen, onCloseDrawer }) {
  return (
    <div className="app-shell">
      {sidebar}
      <button
        type="button"
        className={`drawer-backdrop ${drawerOpen ? 'open' : ''}`}
        aria-label="Close navigation"
        aria-hidden={!drawerOpen}
        tabIndex={drawerOpen ? 0 : -1}
        onClick={onCloseDrawer}
      />
      <div className="content-shell">
        {navbar}
        <main className="main-grid main-grid--single"><div className="main-column">{children}</div></main>
      </div>
    </div>
  );
}
