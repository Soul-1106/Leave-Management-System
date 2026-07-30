import { getInitials } from '../../utils/helpers';

export function Avatar({ name, initials, size = 'md', className = '' }) {
  const label = initials ?? getInitials(name ?? '');
  return (
    <div className={`avatar avatar--${size} ${className}`.trim()} aria-hidden="true">
      {label}
    </div>
  );
}
