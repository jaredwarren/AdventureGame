// Shared API helpers for the tile art editor (mirrors api.js token handling).

const token = document.documentElement.dataset.token || '';

export class ApiError extends Error {
  constructor(status, body) {
    super(body?.error || `HTTP ${status}`);
    this.status = status;
    this.kind = body?.kind || 'http';
    this.detail = body?.detail;
  }
}

async function request(method, path, body) {
  const headers = { 'X-Editor-Token': token };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const resp = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await resp.text();
  let payload = null;
  if (text) {
    try { payload = JSON.parse(text); }
    catch {
      if (!resp.ok) throw new ApiError(resp.status, { error: text.slice(0, 400) });
      throw new ApiError(resp.status, { error: 'response was not JSON' });
    }
  }
  if (!resp.ok) throw new ApiError(resp.status, payload);
  return payload;
}

export const api = {
  listTiles: () => request('GET', '/api/tiles'),
  loadTile: (gid) => request('GET', `/api/tiles/${gid}`),
  saveTile: (gid, art) => request('PUT', `/api/tiles/${gid}`, { art }),
};
