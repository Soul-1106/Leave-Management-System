export function Icon({ name, className = '' }) {
  return (
    <svg className={`icon ${className}`.trim()} viewBox="0 0 24 24" aria-hidden="true">
      {renderPath(name)}
    </svg>
  );
}

function renderPath(name) {
  switch (name) {
    case 'menu':
      return <path d="M4 6h16M4 12h16M4 18h16" />;
    case 'close':
      return <path d="M6 6l12 12M18 6L6 18" />;
    case 'dashboard':
      return <path d="M4 4h7v7H4zM13 4h7v4h-7zM13 10h7v10h-7zM4 13h7v7H4z" />;
    case 'leaves':
      return <path d="M5 4h14v16H5zM8 8h8M8 12h8M8 16h5" />;
    case 'apply':
      return <path d="M12 5v14M5 12h14" />;
    case 'approvals':
      return <path d="M6 12l4 4 8-8M4 5h16v14H4z" />;
    case 'employees':
      return <path d="M16 11a4 4 0 10-8 0 4 4 0 008 0zM4 20a8 8 0 0116 0" />;
    case 'logout':
      return <path d="M10 17l5-5-5-5M15 12H4M15 4h5v16h-5" />;
    case 'calendar':
      return <path d="M7 3v3M17 3v3M4 8h16M5 5h14v15H5zM8 12h3M13 12h3M8 16h3" />;
    case 'health':
      return <path d="M12 21s-7-4.5-7-10A4 4 0 0112 6a4 4 0 017 5c0 5.5-7 10-7 10z" />;
    case 'briefcase':
      return <path d="M10 6h4a2 2 0 012 2v1H8V8a2 2 0 012-2zM4 9h16v10H4zM9 9v-1M15 9v-1" />;
    case 'check':
      return <path d="M20 6L9 17l-5-5" />;
    default:
      return <circle cx="12" cy="12" r="8" />;
  }
}
