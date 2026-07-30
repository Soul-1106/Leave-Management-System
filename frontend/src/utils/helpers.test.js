import { describe, expect, it } from 'vitest';
import { getInitials } from './helpers';

describe('getInitials', () => {
  it('returns at most two uppercase initials', () => {
    expect(getInitials('Sarah Jane Employee')).toBe('SJ');
  });

  it('handles extra whitespace and empty values', () => {
    expect(getInitials('  haneef   ali  ')).toBe('HA');
    expect(getInitials()).toBe('');
  });
});
