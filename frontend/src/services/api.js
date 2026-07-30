const API_URL = '/api';

function csrfToken() {
  const prefix = 'lms_csrf=';
  const cookie = document.cookie.split('; ').find((item) => item.startsWith(prefix));
  return cookie ? decodeURIComponent(cookie.slice(prefix.length)) : '';
}

function getHeaders(customHeaders = {}, includeCsrf = false) {
  const headers = {
    'Content-Type': 'application/json',
    ...customHeaders,
  };

  if (includeCsrf && csrfToken()) {
    headers['X-CSRF-Token'] = csrfToken();
  }

  return headers;
}

export async function establishBackendSession(accessToken) {
  const res = await fetch(`${API_URL}/session`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
  });
  if (!res.ok) throw new Error((await res.text()).trim() || 'Unable to establish backend session');
  return res.json();
}

export async function endBackendSession() {
  await fetch(`${API_URL}/session/logout`, {
    method: 'POST',
    credentials: 'include',
    headers: getHeaders({}, true),
  });
}

export async function apiGet(path) {
  try {
    const headers = getHeaders();
    const res = await fetch(`${API_URL}${path}`, { headers, credentials: 'include' });
    if (!res.ok) {
      const errorText = await res.text();
      console.warn(`apiGet ${path} failed: ${res.status} ${errorText}`);
      return null;
    }
    return await res.json();
  } catch (error) {
    console.error('apiGet error:', error);
    return null;
  }
}

export async function apiPostStrict(path, payload) {
  return apiWriteStrict('POST', path, payload);
}

export async function apiUploadStrict(path, file) {
  const form = new FormData();
  form.append('attachment', file);
  const headers = {};
  const token = csrfToken();
  if (token) headers['X-CSRF-Token'] = token;
  const res = await fetch(`${API_URL}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: form,
  });
  if (!res.ok) {
    const text = await res.text();
    try {
      const parsed = JSON.parse(text);
      throw new Error(parsed.error || `Upload failed with ${res.status}`);
    } catch (error) {
      if (error instanceof SyntaxError) throw new Error(text || `Upload failed with ${res.status}`, { cause: error });
      throw error;
    }
  }
  return res.json();
}

export async function apiGetStrict(path) {
  const headers = getHeaders();
  const res = await fetch(`${API_URL}${path}`, { headers, credentials: 'include' });
  if (!res.ok) {
    const message = (await res.text()).trim();
    throw new Error(message || `Backend request failed with ${res.status}`);
  }
  const contentType = res.headers.get('content-type') || '';
  if (!contentType.includes('application/json')) {
    throw new Error('Backend returned a non-JSON response. Check that the Go API is running and the development proxy is configured.');
  }
  return await res.json();
}

export async function apiPatch(path, payload) {
  return apiWriteStrict('PATCH', path, payload);
}

export async function apiDelete(path) {
  return apiWriteStrict('DELETE', path);
}

async function apiWriteStrict(method, path, payload) {
  const headers = getHeaders({}, true);
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    credentials: 'include',
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });
  if (!res.ok) {
    const text = await res.text();
    try {
      const parsed = JSON.parse(text);
      throw new Error(parsed.error || `Request failed with ${res.status}`);
    } catch (error) {
      if (error instanceof SyntaxError) throw new Error(text || `Request failed with ${res.status}`, { cause: error });
      throw error;
    }
  }
  return await res.json();
}
