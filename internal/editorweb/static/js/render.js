// Canvas rendering: two stacked canvases and one rAF loop.
//
// The map canvas repaints only on document, layer, pan or zoom changes; the
// overlay repaints on every pointer move. Separating them means a mouse move
// costs a few dozen overlay ops instead of a full tile repaint.

import { TILE } from './tileart.js';
import { state, mapW, mapH, topGidAt } from './store.js';
import * as view from './view.js';
import { hitRect, markerColor, markerGlyph, drawMarkerGraphic } from './markers.js';

const DIRTY_MAP = 1;
const DIRTY_OVERLAY = 2;

let mapCanvas, overlayCanvas, mapCtx, overlayCtx;
let dirty = 0;
let raf = 0;
let animFrame = 0;
let animTimer = 0;

/** Extra painting contributed by the active tool (brush ghost, rubber band). */
let overlayHook = null;
export function setOverlayHook(fn) { overlayHook = fn; }

export function init(mapEl, overlayEl) {
  mapCanvas = mapEl;
  overlayCanvas = overlayEl;
  invalidate(DIRTY_MAP | DIRTY_OVERLAY);
}

export function invalidate(bits = DIRTY_MAP | DIRTY_OVERLAY) {
  dirty |= bits;
  if (!raf) raf = requestAnimationFrame(frame);
}

export const invalidateAll = () => invalidate(DIRTY_MAP | DIRTY_OVERLAY);
export const invalidateOverlay = () => invalidate(DIRTY_OVERLAY);

function frame() {
  raf = 0;
  if (!state.doc || !state.atlas) return;
  if (dirty & DIRTY_MAP) paintMap();
  if (dirty & DIRTY_OVERLAY) paintOverlay();
  dirty = 0;
}

/**
 * Water and lava are the only animated tiles. Ticking a shared frame counter
 * keeps the preview honest without re-rendering anything else.
 */
export function startAnimation() {
  stopAnimation();
  const frames = state.atlas?.maxFrames ?? 1;
  // Skip the timer entirely on maps with no water or lava, rather than waking up
  // several times a second to decide there is nothing to redraw.
  if (frames <= 1 || !mapHasAnimatedTiles()) return;
  animTimer = setInterval(() => {
    animFrame = (animFrame + 1) % frames;
    invalidate(DIRTY_MAP);
  }, 130);
}

export function stopAnimation() {
  if (animTimer) clearInterval(animTimer);
  animTimer = 0;
}

let animatedPresent = null;
export function resetAnimationScan() { animatedPresent = null; }

function mapHasAnimatedTiles() {
  if (animatedPresent !== null) return animatedPresent;
  const doc = state.doc;
  if (!doc) return false;
  const animated = new Set(state.atlas.animated);
  animatedPresent = false;
  for (const { layer } of doc.tileLayers) {
    for (const gid of layer.data ?? []) {
      if (animated.has(gid)) { animatedPresent = true; break; }
    }
    if (animatedPresent) break;
  }
  return animatedPresent;
}

// ---- map layer ----

function paintMap() {
  const ctx = view.fitCanvas(mapCanvas);
  const { atlas, doc, ui } = state;
  const W = mapW(), H = mapH();

  ctx.clearRect(0, 0, mapCanvas.width, mapCanvas.height);
  paintCheckerboard(ctx, W, H);

  // Viewport culling keeps the tile loop proportional to what is visible, not
  // to map size.
  const tx0 = Math.max(0, Math.floor(view.view.ox / TILE));
  const ty0 = Math.max(0, Math.floor(view.view.oy / TILE));
  const tx1 = Math.min(W, Math.ceil((view.view.ox + view.worldViewW()) / TILE) + 1);
  const ty1 = Math.min(H, Math.ceil((view.view.oy + view.worldViewH()) / TILE) + 1);

  const ts = TILE * view.view.scale;

  for (let k = 0; k < doc.tileLayers.length; k++) {
    const layer = doc.tileLayers[k].layer;
    if (layer.visible === false) continue;
    if (ui.layerView === 'isolate' && k !== ui.activeLayer) continue;

    let alpha = layer.opacity ?? 1;
    if (ui.layerView === 'dim' && k !== ui.activeLayer) alpha *= 0.35;
    ctx.globalAlpha = alpha;

    const data = layer.data ?? [];
    for (let ty = ty0; ty < ty1; ty++) {
      const row = ty * W;
      const dy = view.worldToDevY(ty * TILE);
      for (let tx = tx0; tx < tx1; tx++) {
        const gid = data[row + tx];
        if (!gid) continue; // gid 0 is a hole, not a black tile
        const sx = atlas.srcX(gid, animFrame);
        if (sx < 0) continue;
        ctx.drawImage(atlas.canvas, sx, atlas.srcY(gid, animFrame), TILE, TILE,
          view.worldToDevX(tx * TILE), dy, ts, ts);
      }
    }
  }
  ctx.globalAlpha = 1;
}

/**
 * A checkerboard behind the map makes gid-0 holes unmistakable.
 *
 * This matters because the runtime SKIPS gid 0 rather than drawing it: a hole in
 * `ground` shows `base` beneath, but a hole in both renders nothing at all in
 * game. The in-game editor cannot show that difference.
 */
function paintCheckerboard(ctx, W, H) {
  const x0 = view.worldToDevX(0);
  const y0 = view.worldToDevY(0);
  const w = W * TILE * view.view.scale;
  const h = H * TILE * view.view.scale;
  if (w <= 0 || h <= 0) return;

  ctx.save();
  ctx.beginPath();
  ctx.rect(x0, y0, w, h);
  ctx.clip();

  ctx.fillStyle = '#171a21';
  ctx.fillRect(x0, y0, w, h);
  ctx.fillStyle = '#1e222b';
  const c = 8 * view.view.dpr;
  const cols = Math.ceil(w / c), rows = Math.ceil(h / c);
  for (let r = 0; r < rows; r++) {
    for (let q = (r % 2); q < cols; q += 2) {
      ctx.fillRect(x0 + q * c, y0 + r * c, c, c);
    }
  }
  ctx.restore();
}

// ---- overlay layer ----

function paintOverlay() {
  const ctx = view.fitCanvas(overlayCanvas);
  const { ui, doc } = state;
  const W = mapW(), H = mapH();

  ctx.clearRect(0, 0, overlayCanvas.width, overlayCanvas.height);

  if (ui.showGrid) paintGrid(ctx, W, H);
  paintBounds(ctx, W, H);

  const showMarkers = ui.mode === 'marker' || ui.showMarkersInTileMode;
  if (showMarkers) {
    ctx.globalAlpha = ui.mode === 'marker' ? 1 : 0.45;
    for (const o of doc.markersLayer?.objects ?? []) paintMarker(ctx, o);
    ctx.globalAlpha = 1;
  }

  if (overlayHook) overlayHook(ctx);
  paintCursor(ctx);
}

function paintGrid(ctx, W, H) {
  const ts = TILE * view.view.scale;
  if (ts < 8) return; // too dense to read

  const x0 = view.worldToDevX(0), y0 = view.worldToDevY(0);
  const w = W * ts, h = H * ts;

  ctx.save();
  ctx.lineWidth = 1;
  for (const [step, alpha] of [[1, 0.09], [8, 0.18]]) {
    if (step === 8 && ts < 6) continue;
    ctx.strokeStyle = `rgba(255,255,255,${alpha})`;
    ctx.beginPath();
    for (let tx = 0; tx <= W; tx += step) {
      const x = Math.round(x0 + tx * ts) + 0.5;
      ctx.moveTo(x, y0); ctx.lineTo(x, y0 + h);
    }
    for (let ty = 0; ty <= H; ty += step) {
      const y = Math.round(y0 + ty * ts) + 0.5;
      ctx.moveTo(x0, y); ctx.lineTo(x0 + w, y);
    }
    ctx.stroke();
  }
  ctx.restore();
}

function paintBounds(ctx, W, H) {
  const ts = TILE * view.view.scale;
  ctx.save();
  ctx.strokeStyle = 'rgba(255,255,255,0.35)';
  ctx.lineWidth = 2;
  ctx.strokeRect(view.worldToDevX(0) + 0.5, view.worldToDevY(0) + 0.5, W * ts, H * ts);
  ctx.restore();
}

function paintMarker(ctx, obj) {
  const { ui } = state;
  const r = hitRect(obj);
  const x = view.worldToDevX(r.x);
  const y = view.worldToDevY(r.y);
  const w = Math.max(2, Math.round(r.w * view.view.scale));
  const h = Math.max(2, Math.round(r.h * view.view.scale));

  const selected = ui.selection === obj.id;
  const hovered = ui.hover === obj.id;
  const color = markerColor(obj.type);

  ctx.save();
  ctx.fillStyle = color + '28';
  ctx.fillRect(x, y, w, h);

  ctx.strokeStyle = selected ? '#ffe14d' : hovered ? '#5ce1e6' : color;
  ctx.lineWidth = selected || hovered ? 2 : 1;
  ctx.strokeRect(x + 0.5, y + 0.5, w - 1, h - 1);

  // A cross at the raw (x, y) origin. For feet-anchored markers the JSON y is
  // the BOTTOM of the box, which is a persistent source of authoring confusion;
  // showing the actual stored point removes the guesswork.
  const ox = view.worldToDevX(obj.x);
  const oy = view.worldToDevY(obj.y);
  const arm = 3 * view.view.dpr;
  ctx.strokeStyle = selected ? '#ffe14d' : color;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(ox - arm, oy); ctx.lineTo(ox + arm, oy);
  ctx.moveTo(ox, oy - arm); ctx.lineTo(ox, oy + arm);
  ctx.stroke();

  if (w >= 10 * view.view.dpr && h >= 10 * view.view.dpr) {
    const cx = x + w / 2;
    const cy = y + h / 2;
    const iconSize = Math.min(w, h) * 0.75;
    ctx.save();
    ctx.globalAlpha = (ctx.globalAlpha || 1) * 0.65;
    const drawn = drawMarkerGraphic(ctx, obj, cx, cy, iconSize);
    ctx.restore();

    if (!drawn) {
      ctx.fillStyle = color;
      ctx.font = `${Math.round(9 * view.view.dpr)}px ui-monospace, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(markerGlyph(obj.type), cx, cy);
    }
  }

  if (selected) {
    const schema = state.schema?.markers.find((m) => m.type === obj.type);
    if (schema?.sizable) paintHandles(ctx, x, y, w, h);
  }
  ctx.restore();
}

function paintHandles(ctx, x, y, w, h) {
  const s = 4 * view.view.dpr;
  ctx.fillStyle = '#ffe14d';
  for (const [hx, hy] of [[x, y], [x + w, y], [x, y + h], [x + w, y + h]]) {
    ctx.fillRect(hx - s / 2, hy - s / 2, s, s);
  }
}

function paintCursor(ctx) {
  const c = state.ui.cursor;
  if (!c || state.ui.mode !== 'tile') return;
  const ts = TILE * view.view.scale;
  ctx.save();
  ctx.strokeStyle = 'rgba(255,255,255,0.75)';
  ctx.lineWidth = 2;
  ctx.strokeRect(view.worldToDevX(c.tx * TILE) + 1, view.worldToDevY(c.ty * TILE) + 1, ts - 2, ts - 2);
  ctx.restore();
}

/** Draws a translucent tile preview, used by the brush and rect tools. */
export function ghostTile(ctx, gid, tx, ty, alpha = 0.6) {
  const ts = TILE * view.view.scale;
  ctx.save();
  ctx.globalAlpha = alpha;
  const sx = state.atlas.srcX(gid, 0);
  if (gid && sx >= 0) {
    ctx.drawImage(state.atlas.canvas, sx, state.atlas.srcY(gid, 0), TILE, TILE,
      view.worldToDevX(tx * TILE), view.worldToDevY(ty * TILE), ts, ts);
  } else {
    ctx.fillStyle = 'rgba(255,80,80,0.35)';
    ctx.fillRect(view.worldToDevX(tx * TILE), view.worldToDevY(ty * TILE), ts, ts);
  }
  ctx.restore();
}

/** Outlines a tile rectangle, used by the rect tool's rubber band. */
export function outlineTileRect(ctx, r, color = '#5ce1e6') {
  const ts = TILE * view.view.scale;
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.setLineDash([6, 4]);
  ctx.strokeRect(
    view.worldToDevX(r.tx0 * TILE) + 1,
    view.worldToDevY(r.ty0 * TILE) + 1,
    (r.tx1 - r.tx0 + 1) * ts - 2,
    (r.ty1 - r.ty0 + 1) * ts - 2,
  );
  ctx.restore();
}

export { topGidAt };
