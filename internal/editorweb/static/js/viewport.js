// Viewport input: pointer capture, event normalization, pan and zoom.
//
// Tools never touch DOM events. Everything arrives as a normalized record, and
// panning is handled here so it can never collide with a tool gesture — which is
// the bug the in-game editor has, where "A" both pans left and adds a marker.

import { TILE } from './tileart.js';
import { state, emit, mapW, mapH, inBounds, topGidAt } from './store.js';
import * as view from './view.js';
import * as render from './render.js';
import { toolState, pickTile } from './tools.js';

let el = null;
let panning = null;
let spaceHeld = false;

export function initViewport(element) {
  el = element;

  const ro = new ResizeObserver(() => {
    view.resize(el.clientWidth, el.clientHeight);
    view.clampPan(mapW(), mapH());
    render.invalidateAll();
  });
  ro.observe(el);

  el.addEventListener('pointerdown', onPointerDown);
  el.addEventListener('pointermove', onPointerMove);
  el.addEventListener('pointerup', onPointerUp);
  el.addEventListener('pointercancel', onPointerUp);
  el.addEventListener('pointerleave', onPointerLeave);
  el.addEventListener('contextmenu', (e) => e.preventDefault());
  // passive:false is required: a trackpad pinch arrives as wheel+ctrlKey and
  // would zoom the whole page instead of the map.
  el.addEventListener('wheel', onWheel, { passive: false });

  window.addEventListener('keydown', (e) => {
    if (e.code === 'Space' && !isTyping(e)) { spaceHeld = true; updateCursor(); }
  });
  window.addEventListener('keyup', (e) => {
    if (e.code === 'Space') { spaceHeld = false; updateCursor(); }
  });
  window.addEventListener('blur', () => { spaceHeld = false; updateCursor(); });
}

export function isTyping(e) {
  const t = e.target;
  return t instanceof HTMLElement &&
    (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable);
}

/** Builds the normalized event record tools consume. */
function mk(ev) {
  const rect = el.getBoundingClientRect();
  const sx = ev.clientX - rect.left;
  const sy = ev.clientY - rect.top;
  const wx = view.cssToWorldX(sx);
  const wy = view.cssToWorldY(sy);
  const tx = view.worldToTile(wx);
  const ty = view.worldToTile(wy);
  return {
    sx, sy, wx, wy, tx, ty,
    inBounds: inBounds(tx, ty),
    button: ev.button,
    shift: ev.shiftKey, alt: ev.altKey, ctrl: ev.ctrlKey, meta: ev.metaKey,
    raw: ev,
  };
}

function onPointerDown(ev) {
  // Adjacency chips live inside #viewport but own their own clicks; capturing
  // here would swallow the button's click and prevent openMap navigation.
  if (ev.target instanceof Element && ev.target.closest('.adj')) return;

  el.setPointerCapture(ev.pointerId);
  const e = mk(ev);

  // Pan always wins, so it can never be swallowed by a tool.
  if (ev.button === 1 || spaceHeld) {
    panning = { sx: e.sx, sy: e.sy, ox: view.view.ox, oy: view.view.oy };
    updateCursor();
    return;
  }
  // Right-click stays the eyedropper, as in the in-game editor, but Alt+click
  // now does the same so right-click remains available.
  if (ev.button === 2 && state.ui.mode === 'tile') {
    pickTile(e.tx, e.ty, e.shift);
    return;
  }
  if (ev.button !== 0) return;

  if (e.alt && state.ui.mode === 'tile') {
    pickTile(e.tx, e.ty, true);
    return;
  }
  toolState.active?.onPointerDown?.(e);
}

function onPointerMove(ev) {
  const e = mk(ev);
  updateStatusCursor(e);

  if (panning) {
    view.view.ox = panning.ox - ((e.sx - panning.sx) * view.view.dpr) / view.view.scale;
    view.view.oy = panning.oy - ((e.sy - panning.sy) * view.view.dpr) / view.view.scale;
    view.clampPan(mapW(), mapH());
    render.invalidateAll();
    return;
  }
  toolState.active?.onPointerMove?.(e);
}

function onPointerUp(ev) {
  if (el.hasPointerCapture?.(ev.pointerId)) el.releasePointerCapture(ev.pointerId);
  if (panning) {
    panning = null;
    updateCursor();
    return;
  }
  toolState.active?.onPointerUp?.(mk(ev));
}

function onPointerLeave() {
  state.ui.cursor = null;
  state.ui.hover = null;
  emit('cursor');
  render.invalidateOverlay();
}

function onWheel(ev) {
  ev.preventDefault();
  const rect = el.getBoundingClientRect();
  const sx = ev.clientX - rect.left;
  const sy = ev.clientY - rect.top;

  if (ev.shiftKey && !ev.ctrlKey) {
    view.panBy((ev.deltaY * view.view.dpr) / view.view.scale, 0, mapW(), mapH());
    render.invalidateAll();
    return;
  }
  if (view.zoomAt(sx, sy, ev.deltaY < 0 ? 1 : -1, mapW(), mapH())) {
    emit('ui');
    render.invalidateAll();
  }
}

function updateStatusCursor(e) {
  const prev = state.ui.cursor;
  if (prev && prev.tx === e.tx && prev.ty === e.ty) return;
  state.ui.cursor = { tx: e.tx, ty: e.ty, wx: e.wx, wy: e.wy };
  emit('cursor');
  render.invalidateOverlay();
}

function updateCursor() {
  if (!el) return;
  el.style.cursor = panning ? 'grabbing' : spaceHeld ? 'grab' : (toolState.active?.cursor ?? 'default');
}

export function refreshCursor() { updateCursor(); }

/** Describes the tile under the cursor for the status bar. */
export function cursorInfo() {
  const c = state.ui.cursor;
  if (!c || !state.atlas) return '';
  if (!inBounds(c.tx, c.ty)) return `tile ${c.tx},${c.ty}  (outside)`;

  const { gid } = topGidAt(c.tx, c.ty);
  const def = state.atlas.def(gid);
  const tags = def?.tags?.length ? ` [${def.tags.join(',')}]` : '';
  return `tile ${c.tx},${c.ty}  px ${Math.floor(c.wx)},${Math.floor(c.wy)}  gid ${gid} ${def?.name ?? '?'}${tags}`;
}

export { TILE };
