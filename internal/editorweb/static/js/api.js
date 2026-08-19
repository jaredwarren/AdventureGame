// Every fetch in the app goes through here, so the editor token and error
// shaping live in one place.

const token = document.documentElement.dataset.token || '';

export class ApiError extends Error {
  constructor(status, body) {
    super(body?.error || `HTTP ${status}`);
    this.status = status;
    this.kind = body?.kind || 'http';
    this.detail = body?.detail;
  }
}

async function request(method, path, body, extraHeaders) {
  const headers = { 'X-Editor-Token': token, ...extraHeaders };
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  const resp = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (resp.status === 204) return null;

  let payload = null;
  const text = await resp.text();
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      if (!resp.ok) throw new ApiError(resp.status, { error: text.slice(0, 400) });
      throw new ApiError(resp.status, { error: 'response was not JSON' });
    }
  }
  if (!resp.ok) throw new ApiError(resp.status, payload);
  return payload;
}

export const api = {
  health: () => request('GET', '/api/health'),
  schema: () => request('GET', '/api/schema'),
  listMaps: () => request('GET', '/api/maps'),
  loadMap: (id) => request('GET', `/api/maps/${id}`),

  /**
   * Saves a map. A 409 ApiError carries {currentEtag, serverMap} so the caller
   * can offer reload-vs-overwrite; force:true takes the overwrite branch.
   */
  saveMap: (id, map, baseEtag, force = false) =>
    request('PUT', `/api/maps/${id}`, { baseEtag, force: force === true, map }),

  /** Creates a marker via internal/world, so snapping and defaults cannot drift. */
  newMarker: (req) => request('POST', '/api/marker', req),

  /** Validates the in-memory document, before it ever reaches disk. */
  validate: (id, map) => request('POST', '/api/validate', { id, map }),
  validateAll: () => request('GET', '/api/validate'),

  thumb: (id) => request('GET', `/api/thumbs/${id}`),
};
