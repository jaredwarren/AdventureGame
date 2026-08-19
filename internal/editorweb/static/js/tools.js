// The tool registry and the seven editing tools.
//
// The viewport owns pointer capture and event normalization; tools only ever see
// a plain {sx, sy, wx, wy, tx, ty, inBounds, button, modifiers} record. Adding a
// tool is one registry entry plus a toolbar button.

import { TILE } from './tileart.js';
import {
  state, emit, activeLayer, inBounds, tileIndex, gidAt, topGidAt,
  mapW, mapH, allocObjectID, ensureMarkersLayer, reindex,
} from './store.js';
import { TileEditBatch, cmd } from './commands.js';
import * as history from './history.js';
import * as render from './render.js';
import { markerAt, hitRect } from './markers.js';
import { api } from './api.js';
import { toast } from './ui/dialogs.js';

export const tools = new Map();
export const toolState = { active: null, lastByMode: { tile: 'brush', marker: 'select' } };

function register(tool) { tools.set(tool.id, tool); }

export function setTool(id) {
  const next = tools.get(id);
  if (!next || next === toolState.active) return;
  toolState.active?.onDeactivate?.();
  toolState.active = next;
  toolState.lastByMode[next.mode] = id;
  state.ui.mode = next.mode;
  state.ui.tool = id;
  render.setOverlayHook(next.drawOverlay ?? null);
  next.onActivate?.();
  emit('tools', 'ui');
  render.invalidateAll();
}

/** Switches editing mode, returning to the tool last used in that mode. */
export function setMode(mode) {
  setTool(toolState.lastByMode[mode] ?? (mode === 'tile' ? 'brush' : 'select'));
}

// ---- tile tools ----

let batch = null;
let lastPaint = { tx: -1, ty: -1 };

function beginBatch(label) {
  const layer = activeLayer();
  if (!layer) return false;
  batch = new TileEditBatch(layer, label);
  lastPaint = { tx: -1, ty: -1 };
  return true;
}

function commitBatch() {
  if (!batch) return;
  const command = batch.commit();
  batch = null;
  if (command) {
    history.push(command);
    state.validation.stale = true;
    render.resetAnimationScan();
    emit('doc', 'validation');
    render.invalidateAll();
  }
}

function paintTile(tx, ty, gid) {
  if (!batch || !inBounds(tx, ty)) return;
  batch.set(tileIndex(tx, ty), gid);
}

/**
 * Paints a line of tiles between two positions.
 *
 * The in-game editor only dedupes against the previous tile, so a fast drag
 * skips everything between two pointer samples. Interpolating fixes that.
 */
function paintLine(x0, y0, x1, y1, gid) {
  let dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0);
  const sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1;
  let err = dx - dy;
  for (;;) {
    paintTile(x0, y0, gid);
    if (x0 === x1 && y0 === y1) break;
    const e2 = 2 * err;
    if (e2 > -dy) { err -= dy; x0 += sx; }
    if (e2 < dx) { err += dx; y0 += sy; }
  }
}

function brushTool(id, label, gidFor) {
  return {
    id, label, mode: 'tile', cursor: 'crosshair',
    onPointerDown(e) {
      if (e.button !== 0 || !e.inBounds) return;
      if (!beginBatch(label)) return;
      paintTile(e.tx, e.ty, gidFor());
      lastPaint = { tx: e.tx, ty: e.ty };
      render.invalidateAll();
    },
    onPointerMove(e) {
      if (!batch) return;
      if (e.tx === lastPaint.tx && e.ty === lastPaint.ty) return;
      if (lastPaint.tx >= 0) paintLine(lastPaint.tx, lastPaint.ty, e.tx, e.ty, gidFor());
      else paintTile(e.tx, e.ty, gidFor());
      lastPaint = { tx: e.tx, ty: e.ty };
      render.invalidateAll();
    },
    onPointerUp() { commitBatch(); },
    onDeactivate() { commitBatch(); },
    drawOverlay(ctx) {
      const c = state.ui.cursor;
      if (c && inBounds(c.tx, c.ty)) render.ghostTile(ctx, gidFor(), c.tx, c.ty, 0.55);
    },
  };
}

register(brushTool('brush', 'paint', () => state.ui.brushGID));
register(brushTool('eraser', 'erase', () => 0));

let rectAnchor = null;

register({
  id: 'rect', label: 'rectangle', mode: 'tile', cursor: 'crosshair',
  onPointerDown(e) {
    if (e.button !== 0 || !e.inBounds) return;
    rectAnchor = { tx: e.tx, ty: e.ty };
    render.invalidateOverlay();
  },
  onPointerMove() { if (rectAnchor) render.invalidateOverlay(); },
  onPointerUp(e) {
    if (!rectAnchor) return;
    const r = normalizeRect(rectAnchor, { tx: e.tx, ty: e.ty });
    rectAnchor = null;

    const gid = e.shift ? 0 : state.ui.brushGID;
    if (!beginBatch(e.alt ? 'outline' : 'fill rectangle')) return;
    for (let ty = r.ty0; ty <= r.ty1; ty++) {
      for (let tx = r.tx0; tx <= r.tx1; tx++) {
        const edge = tx === r.tx0 || tx === r.tx1 || ty === r.ty0 || ty === r.ty1;
        if (e.alt && !edge) continue;
        paintTile(tx, ty, gid);
      }
    }
    commitBatch();
  },
  onDeactivate() { rectAnchor = null; commitBatch(); },
  drawOverlay(ctx) {
    const c = state.ui.cursor;
    if (!c) return;
    if (rectAnchor) render.outlineTileRect(ctx, normalizeRect(rectAnchor, c));
    else render.ghostTile(ctx, state.ui.brushGID, c.tx, c.ty, 0.4);
  },
});

function normalizeRect(a, b) {
  const clamp = (v, hi) => Math.max(0, Math.min(v, hi - 1));
  return {
    tx0: clamp(Math.min(a.tx, b.tx), mapW()),
    ty0: clamp(Math.min(a.ty, b.ty), mapH()),
    tx1: clamp(Math.max(a.tx, b.tx), mapW()),
    ty1: clamp(Math.max(a.ty, b.ty), mapH()),
  };
}

register({
  id: 'flood', label: 'flood fill', mode: 'tile', cursor: 'cell',
  onPointerDown(e) {
    if (e.button !== 0 || !e.inBounds) return;
    const layer = activeLayer();
    if (!layer) return;

    const target = gidAt(layer, e.tx, e.ty);
    const gid = state.ui.brushGID;
    if (target === gid) return;

    if (!beginBatch(e.shift ? 'replace all' : 'flood fill')) return;
    if (e.shift) {
      // Global replace: every matching tile in the layer, not just the region.
      for (let i = 0; i < layer.data.length; i++) {
        if (layer.data[i] === target) batch.set(i, gid);
      }
    } else {
      floodFill(layer, e.tx, e.ty, target, gid);
    }
    commitBatch();
  },
  drawOverlay(ctx) {
    const c = state.ui.cursor;
    if (c && inBounds(c.tx, c.ty)) render.ghostTile(ctx, state.ui.brushGID, c.tx, c.ty, 0.55);
  },
});

/** Four-connected flood fill, iterative so a large region cannot blow the stack. */
function floodFill(layer, tx, ty, target, gid) {
  const W = mapW(), H = mapH();
  const stack = [[tx, ty]];
  const seen = new Set();
  while (stack.length) {
    const [x, y] = stack.pop();
    if (x < 0 || y < 0 || x >= W || y >= H) continue;
    const i = y * W + x;
    if (seen.has(i)) continue;
    seen.add(i);
    if (layer.data[i] !== target) continue;
    batch.set(i, gid);
    stack.push([x + 1, y], [x - 1, y], [x, y + 1], [x, y - 1]);
  }
}

register({
  id: 'eyedropper', label: 'pick tile', mode: 'tile', cursor: 'copy',
  onPointerDown(e) {
    if (!e.inBounds) return;
    pickTile(e.tx, e.ty, e.alt);
  },
  drawOverlay(ctx) {
    const c = state.ui.cursor;
    if (c && inBounds(c.tx, c.ty)) render.outlineTileRect(ctx, { tx0: c.tx, ty0: c.ty, tx1: c.tx, ty1: c.ty }, '#ffe14d');
  },
});

/**
 * Picks the GID under the cursor into the brush.
 *
 * With acrossLayers, picks the topmost visible tile and switches the active
 * layer to wherever it came from, which is usually what you meant.
 */
export function pickTile(tx, ty, acrossLayers = false) {
  if (!inBounds(tx, ty)) return;
  if (acrossLayers) {
    const { gid, layerIndex } = topGidAt(tx, ty);
    if (layerIndex >= 0) state.ui.activeLayer = layerIndex;
    state.ui.brushGID = gid;
  } else {
    state.ui.brushGID = gidAt(activeLayer(), tx, ty);
  }
  emit('ui', 'palette');
  render.invalidateAll();
}

// ---- marker tools ----

let drag = null;

register({
  id: 'select', label: 'select', mode: 'marker', cursor: 'default',
  onPointerDown(e) {
    if (e.button !== 0) return;
    const obj = markerAt(e.wx, e.wy);
    state.ui.selection = obj?.id ?? null;
    if (obj) {
      drag = {
        id: obj.id,
        token: history.newDragToken(),
        grabDX: e.wx - obj.x,
        grabDY: e.wy - obj.y,
        before: { x: obj.x, y: obj.y },
        moved: false,
      };
    }
    emit('selection');
    render.invalidateOverlay();
  },
  onPointerMove(e) {
    if (!drag) {
      const obj = markerAt(e.wx, e.wy);
      const id = obj?.id ?? null;
      if (id !== state.ui.hover) {
        state.ui.hover = id;
        render.invalidateOverlay();
      }
      return;
    }
    const obj = state.doc.objById.get(drag.id);
    if (!obj) return;

    let nx = e.wx - drag.grabDX;
    let ny = e.wy - drag.grabDY;
    if (snapFor(obj.type, e)) {
      nx = Math.round(nx / TILE) * TILE;
      ny = Math.round(ny / TILE) * TILE;
    }
    if (nx !== obj.x || ny !== obj.y) {
      obj.x = nx;
      obj.y = ny;
      drag.moved = true;
      render.invalidateOverlay();
      emit('selection');
    }
  },
  onPointerUp() {
    if (!drag) return;
    const obj = state.doc.objById.get(drag.id);
    if (obj && drag.moved) {
      // The object was already moved live; record the change without reapplying.
      history.push(cmd.markerMove(drag.id, drag.before, { x: obj.x, y: obj.y }), drag.token);
      state.validation.stale = true;
      emit('doc', 'validation');
    }
    drag = null;
  },
  onDeactivate() { drag = null; },
});

/** Door and sign markers snap to the tile grid; Alt inverts that. */
function snapFor(type, e) {
  const schema = state.schema?.markers.find((m) => m.type === type);
  return (schema?.snapsToGrid ?? false) !== e.alt;
}

register({
  id: 'marker-add', label: 'add marker', mode: 'marker', cursor: 'copy',
  async onPointerDown(e) {
    if (e.button !== 0) return;
    await addMarker(e.wx, e.wy);
    if (e.alt) setTool('select');
  },
  drawOverlay(ctx) {
    const c = state.ui.cursor;
    if (c) render.outlineTileRect(ctx, { tx0: c.tx, ty0: c.ty, tx1: c.tx, ty1: c.ty }, '#46d17a');
  },
});

/**
 * Adds a marker at a world point.
 *
 * The object is built by the SERVER, which runs world.InitMarkerObject. That is
 * what makes door/sign grid snapping and every default property identical to
 * what the game's own editor would produce, instead of a JS reimplementation
 * that can drift.
 */
export async function addMarker(wx, wy) {
  const type = state.ui.markerType;
  ensureMarkersLayer();
  const id = allocObjectID();

  let obj;
  try {
    const res = await api.newMarker({
      type, x: wx, y: wy, nextObjectId: id, pickupKind: state.ui.pickupKind,
    });
    obj = res.object;
  } catch (err) {
    toast(`could not create marker: ${err.message}`, 'error');
    return null;
  }

  const objects = state.doc.markersLayer.objects;
  history.run(cmd.markerAdd(obj, objects.length));
  state.ui.selection = obj.id;
  state.validation.stale = true;
  emit('doc', 'selection', 'validation');
  render.invalidateOverlay();
  return obj;
}

export function deleteSelected() {
  const id = state.ui.selection;
  if (id === null) return;
  const objects = state.doc.markersLayer?.objects ?? [];
  const index = objects.findIndex((o) => o.id === id);
  if (index < 0) return;

  history.run(cmd.markerDelete(objects[index], index));
  state.ui.selection = null;
  state.validation.stale = true;
  emit('doc', 'selection', 'validation');
  render.invalidateOverlay();
}

export async function duplicateSelected() {
  const id = state.ui.selection;
  const src = state.doc.objById.get(id);
  if (!src) return;

  const copy = structuredClone(src);
  copy.id = allocObjectID();
  copy.x += TILE;
  copy.y += TILE;

  const objects = state.doc.markersLayer.objects;
  history.run(cmd.markerAdd(copy, objects.length));
  state.ui.selection = copy.id;
  state.validation.stale = true;
  emit('doc', 'selection', 'validation');
  render.invalidateOverlay();
}

export { hitRect, reindex };
