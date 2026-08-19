// Bootstrap and the document lifecycle (open, save, reload).

import { api, ApiError } from './api.js';
import { TileAtlas } from './tileart.js';
import { state, emit, subscribe, openDoc } from './store.js';
import * as view from './view.js';
import * as render from './render.js';
import * as history from './history.js';
import { setTool, setMode } from './tools.js';
import { initViewport, cursorInfo } from './viewport.js';
import { initDialogs, toast, confirmDiscard, confirmOverwrite, confirmInvalid } from './ui/dialogs.js';
import { initToolbar } from './ui/toolbar.js';
import { initPalette, renderPalette } from './ui/palette.js';
import { initLayers } from './ui/layers.js';
import { initProps } from './ui/props.js';
import { initBrowser, refreshThumb, renderAdjacency } from './ui/browser.js';
import { initValidate, errorIssues, runValidation } from './ui/validate.js';

const DRAFT_PREFIX = 'agedit.draft.';

async function boot() {
  initDialogs();

  try {
    const [health, schema, maps] = await Promise.all([api.health(), api.schema(), api.listMaps()]);
    state.health = health;
    state.schema = schema;

    if (schema.schemaVersion !== 1) {
      fatal(`This editor build expects schema version 1 but the server sent ${schema.schemaVersion}. Restart the server.`);
      return;
    }
    state.atlas = new TileAtlas(schema);
    state.ui.favorites = schema.favorites ?? [];
    state.ui.brushGID = schema.favorites?.[0] ?? 1;
    state.ui.markerType = schema.markers[0]?.type ?? 'enemy';
    state.ui.pickupKind = schema.pickups[0]?.value ?? 'coin';
    setMaps(maps);
  } catch (err) {
    fatal(`Could not reach the editor server: ${err.message}`);
    return;
  }

  document.getElementById('maps-root').textContent = state.health.root;

  render.init(document.getElementById('c-map'), document.getElementById('c-overlay'));
  initViewport(document.getElementById('viewport'));
  initToolbar({ onSave: save, onReload: reload });
  initPalette();
  initLayers();
  initProps();
  initValidate();
  initBrowser(openMap);
  emitInitialState();

  subscribe('cursor', updateStatus);
  subscribe('doc', updateStatus);
  subscribe('ui', updateStatus);
  subscribe('selection', updateStatus);

  window.addEventListener('beforeunload', (e) => {
    if (!history.isDirty()) return;
    e.preventDefault();
    e.returnValue = '';
  });

  setTool('brush');

  const first = new URLSearchParams(location.search).get('map')
    || (state.mapIndex.has('F-5') ? 'F-5' : state.maps[0]?.id);
  if (first) await openMap(first);

  document.getElementById('app').hidden = false;
  document.getElementById('boot').hidden = true;
}

function setMaps(payload) {
  state.maps = payload.maps ?? [];
  state.mapIndex = new Map(state.maps.map((m) => [m.id, m]));
  emit('maps');
}

/**
 * Broadcasts the initial state once every panel has subscribed.
 *
 * boot() has to load the schema and map list BEFORE it can build the UI (the
 * toolbar needs the marker list, the palette needs the atlas), so the emits
 * those loads trigger land before anyone is listening. Replaying them here is
 * what makes the map browser and grid appear on first paint.
 */
function emitInitialState() {
  emit('maps', 'ui', 'tools', 'history', 'palette', 'validation');
}

/** Opens a map, guarding unsaved changes first. */
export async function openMap(id) {
  if (state.doc && state.doc.id !== id && history.isDirty()) {
    const choice = await confirmDiscard(state.doc.id);
    if (choice === null) return;
    if (choice === 'save' && !(await save())) return;
  }

  let payload;
  try {
    payload = await api.loadMap(id);
  } catch (err) {
    toast(`Could not open ${id}: ${err.message}`, 'error');
    return;
  }

  openDoc(payload);
  history.useMap(id);
  render.resetAnimationScan();

  view.resize(
    document.getElementById('viewport').clientWidth,
    document.getElementById('viewport').clientHeight,
  );
  view.zoomToFit(payload.map.width, payload.map.height);

  await offerDraft(id);

  history.markSaved();
  emit('doc', 'ui', 'selection', 'validation', 'palette');
  renderPalette();
  renderAdjacency(openMap);
  render.startAnimation();
  render.invalidateAll();

  const url = new URL(location.href);
  url.searchParams.set('map', id);
  history_replace(url);

  if (payload.reformats) {
    toast(`${id} will be reformatted when saved (whitespace only)`, 'warn', 5000);
  }
  if (payload.unmodeledFields?.length) {
    toast(`${id} has fields this editor cannot represent: ${payload.unmodeledFields.join(', ')}`, 'error', 8000);
  }
}

function history_replace(url) {
  window.history.replaceState(null, '', url);
}

// ---- drafts ----
//
// A debounced local snapshot means a crashed tab or a stray reload does not lose
// work. Drafts are cleared on a successful save.

let draftTimer = 0;
subscribe('doc', () => {
  if (!state.doc) return;
  clearTimeout(draftTimer);
  draftTimer = setTimeout(saveDraft, 2000);
});

function draftKey(id) { return DRAFT_PREFIX + id; }

function saveDraft() {
  if (!state.doc || !history.isDirty()) return;
  try {
    localStorage.setItem(draftKey(state.doc.id), JSON.stringify({
      etag: state.doc.etag, at: Date.now(), tmj: state.doc.tmj,
    }));
  } catch {
    // Quota exceeded on a very large map: drafts are a convenience, not a
    // guarantee, so degrade silently rather than interrupting an edit.
  }
}

async function offerDraft(id) {
  let draft;
  try {
    draft = JSON.parse(localStorage.getItem(draftKey(id)) ?? 'null');
  } catch {
    return;
  }
  if (!draft?.tmj) return;

  // A draft taken against a different on-disk version is not safely mergeable.
  if (draft.etag !== state.doc.etag) {
    localStorage.removeItem(draftKey(id));
    return;
  }
  const when = new Date(draft.at).toLocaleTimeString();
  const { choose } = await import('./ui/dialogs.js');
  const choice = await choose({
    title: 'Unsaved draft found',
    body: `${id} has unsaved changes from ${when}.`,
    buttons: [
      { label: 'Restore', value: 'restore', kind: 'primary' },
      { label: 'Discard', value: 'discard', kind: 'danger' },
    ],
  });
  if (choice === 'restore') {
    state.doc.tmj = draft.tmj;
    const { reindex } = await import('./store.js');
    reindex(state.doc);
    toast('Draft restored', 'info');
  } else {
    localStorage.removeItem(draftKey(id));
  }
}

// ---- save and reload ----

export async function save(force = false) {
  force = force === true;
  const doc = state.doc;
  if (!doc) return false;

  try {
    const res = await api.saveMap(doc.id, doc.tmj, doc.etag, force);
    doc.etag = res.etag;
    state.validation = { issues: res.issues ?? [], stale: false, running: false };
    history.markSaved();
    localStorage.removeItem(draftKey(doc.id));

    emit('doc', 'validation', 'history');
    refreshThumb(doc.id);
    toast(`Saved ${doc.id}`, 'success', 1800);
    return true;
  } catch (err) {
    return handleSaveError(err);
  }
}

async function handleSaveError(err) {
  if (!(err instanceof ApiError)) {
    toast(`Save failed: ${err.message}`, 'error');
    return false;
  }

  if (err.kind === 'stale_etag') {
    // Most often the in-game editor saved the same file.
    const choice = await confirmOverwrite(state.doc.id);
    if (choice === 'overwrite') return save(true);
    if (choice === 'reload') { await reload(true); return false; }
    return false;
  }

  if (err.kind === 'invalid_map') {
    state.validation = { issues: err.detail?.issues ?? [], stale: false, running: false };
    emit('validation');
    const choice = await confirmInvalid(errorIssues());
    if (choice === 'force') return save(true);
    return false;
  }

  toast(`Save failed: ${err.message}`, 'error');
  return false;
}

export async function reload(discard = false) {
  discard = discard === true;
  const doc = state.doc;
  if (!doc) return;

  if (!discard && history.isDirty()) {
    const choice = await confirmDiscard(doc.id);
    if (choice === null) return;
    if (choice === 'save' && !(await save())) return;
  }
  localStorage.removeItem(draftKey(doc.id));
  history.reset(doc.id);

  const id = doc.id;
  state.doc = null;
  await openMap(id);
  toast(`Reloaded ${id}`, 'info', 1500);
}

// ---- status bar ----

function updateStatus() {
  const el = document.getElementById('status-left');
  const right = document.getElementById('status-right');
  if (!el || !state.doc) return;

  const doc = state.doc;
  const bits = [cursorInfo()].filter(Boolean);
  bits.push(`${doc.tmj.width}x${doc.tmj.height}`, `${Math.round(view.view.zoom * 100)}%`);

  const layer = doc.tileLayers[state.ui.activeLayer]?.layer;
  if (layer) bits.push(`layer ${layer.name}`);
  if (state.ui.mode === 'tile' && state.atlas) {
    bits.push(`brush ${state.atlas.name(state.ui.brushGID)}`);
  } else if (state.ui.selection !== null) {
    const obj = doc.objById.get(state.ui.selection);
    if (obj) bits.push(`sel ${obj.type}#${obj.id}`);
  }
  el.textContent = bits.join('  •  ');

  right.textContent = history.isDirty() ? 'unsaved changes' : 'saved';
  right.className = history.isDirty() ? 'dirty' : '';
}

function fatal(message) {
  const boot = document.getElementById('boot');
  boot.innerHTML = '';
  const h = document.createElement('h1');
  h.textContent = 'Map editor';
  const p = document.createElement('p');
  p.className = 'fatal';
  p.textContent = message;
  boot.append(h, p);
}

boot().catch((err) => {
  console.error(err);
  fatal(err.message);
});

export { runValidation };
