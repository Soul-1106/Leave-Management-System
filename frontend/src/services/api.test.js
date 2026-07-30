import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { apiGetStrict, apiPostStrict } from './api';

describe('API client', () => {
  beforeEach(() => {
    vi.stubGlobal('document', { cookie: 'lms_csrf=test-csrf-token' });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('sends cookies and the CSRF header for writes', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ status: 'created' }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ));
    vi.stubGlobal('fetch', fetchMock);

    await apiPostStrict('/leaves/my', { reason: 'Holiday' });

    expect(fetchMock).toHaveBeenCalledWith('/api/leaves/my', expect.objectContaining({
      method: 'POST',
      credentials: 'include',
      headers: expect.objectContaining({ 'X-CSRF-Token': 'test-csrf-token' }),
    }));
  });

  it('rejects non-JSON success responses from strict reads', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      '<!doctype html>',
      { status: 200, headers: { 'Content-Type': 'text/html' } },
    )));

    await expect(apiGetStrict('/me')).rejects.toThrow('non-JSON response');
  });

  it('uses the API error message returned by the backend', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: 'insufficient leave balance' }),
      { status: 409, headers: { 'Content-Type': 'application/json' } },
    )));

    await expect(apiPostStrict('/leaves/id/decision', {}))
      .rejects.toThrow('insufficient leave balance');
  });
});
