// Pan, zoom, and the screen/world/tile coordinate conversions.
//
// Crispness rule: do every drawing calculation in DEVICE pixels and keep the
// world-to-device scale an integer. Using ctx.scale(dpr) together with a
// fractional zoom is what produces shimmering half-pixel seams between tiles.
// With an integer scale, a 16px source drawn into a 16*scale destination is an
// exact nearest-neighbour upscale, which is the same chunky look the game gets
// from upscaling its 320x240 framebuffer.

import { TILE } from './tileart.js';

export const ZOOM_STEPS = [1, 2, 3, 4, 6, 8, 12, 16];

export const view = {
  ox: 0,        // world px at the viewport's left edge
  oy: 0,        // world px at the viewport's top edge
  zoom: 3,      // CSS px per world px
  dpr: 1,
  scale: 3,     // DEVICE px per world px, always an integer
  cssW: 0,
  cssH: 0,
};

/** Recomputes the device-pixel geometry. Call on resize, zoom, and DPR change. */
export function resize(cssW, cssH) {
  view.cssW = cssW;
  view.cssH = cssH;
  view.dpr = window.devicePixelRatio || 1;
  view.scale = Math.max(1, Math.round(view.zoom * view.dpr));
}

/** Sizes a canvas's backing store to the viewport in device pixels. */
export function fitCanvas(canvas) {
  const w = Math.max(1, Math.round(view.cssW * view.dpr));
  const h = Math.max(1, Math.round(view.cssH * view.dpr));
  if (canvas.width !== w) canvas.width = w;
  if (canvas.height !== h) canvas.height = h;
  canvas.style.width = view.cssW + 'px';
  canvas.style.height = view.cssH + 'px';

  const ctx = canvas.getContext('2d');
  ctx.setTransform(1, 0, 0, 1, 0, 0); // we work directly in device px
  ctx.imageSmoothingEnabled = false;  // must be re-set after every resize
  return ctx;
}

export const deviceW = () => Math.round(view.cssW * view.dpr);
export const deviceH = () => Math.round(view.cssH * view.dpr);

/** Width of the visible area in world pixels. */
export const worldViewW = () => deviceW() / view.scale;
export const worldViewH = () => deviceH() / view.scale;

// CSS px (what pointer events report, relative to the viewport) -> world px.
export const cssToWorldX = (cx) => view.ox + (cx * view.dpr) / view.scale;
export const cssToWorldY = (cy) => view.oy + (cy * view.dpr) / view.scale;

// World px -> device px, snapped so tiles land on whole pixels.
export const worldToDevX = (wx) => Math.round((wx - view.ox) * view.scale);
export const worldToDevY = (wy) => Math.round((wy - view.oy) * view.scale);

/**
 * World px -> tile index.
 *
 * Math.floor rather than truncation so negative coordinates (which happen while
 * panning past the map edge) map to negative tiles instead of folding onto 0.
 */
export const worldToTile = (w) => Math.floor(w / TILE);

/**
 * Clamps the pan.
 *
 * Deliberately looser than the in-game editor's camera clamp: half a viewport of
 * overscroll on each side keeps edge tiles away from the panels, and a map
 * smaller than the viewport is centred rather than jammed into the corner.
 */
export function clampPan(mapW, mapH) {
  const worldW = mapW * TILE;
  const worldH = mapH * TILE;
  const vw = worldViewW();
  const vh = worldViewH();

  if (worldW <= vw) {
    view.ox = (worldW - vw) / 2;
  } else {
    view.ox = Math.max(-vw / 2, Math.min(view.ox, worldW - vw / 2));
  }
  if (worldH <= vh) {
    view.oy = (worldH - vh) / 2;
  } else {
    view.oy = Math.max(-vh / 2, Math.min(view.oy, worldH - vh / 2));
  }
}

/** Zooms one step, keeping the world point under the cursor fixed. */
export function zoomAt(cssX, cssY, direction, mapW, mapH) {
  const wx = cssToWorldX(cssX);
  const wy = cssToWorldY(cssY);

  const i = ZOOM_STEPS.indexOf(view.zoom);
  const next = ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, (i < 0 ? 2 : i) + direction))];
  if (next === view.zoom) return false;

  view.zoom = next;
  resize(view.cssW, view.cssH);
  view.ox = wx - (cssX * view.dpr) / view.scale;
  view.oy = wy - (cssY * view.dpr) / view.scale;
  clampPan(mapW, mapH);
  return true;
}

export function setZoom(z, mapW, mapH) {
  const cx = view.cssW / 2;
  const cy = view.cssH / 2;
  const wx = cssToWorldX(cx);
  const wy = cssToWorldY(cy);
  view.zoom = z;
  resize(view.cssW, view.cssH);
  view.ox = wx - (cx * view.dpr) / view.scale;
  view.oy = wy - (cy * view.dpr) / view.scale;
  clampPan(mapW, mapH);
}

/** Picks the largest zoom step that shows the whole map, and centres it. */
export function zoomToFit(mapW, mapH) {
  const worldW = mapW * TILE;
  const worldH = mapH * TILE;
  if (!worldW || !worldH || !view.cssW) return;

  let best = ZOOM_STEPS[0];
  for (const z of ZOOM_STEPS) {
    if (worldW * z <= view.cssW && worldH * z <= view.cssH) best = z;
  }
  view.zoom = best;
  resize(view.cssW, view.cssH);
  view.ox = (worldW - worldViewW()) / 2;
  view.oy = (worldH - worldViewH()) / 2;
}

export function panBy(worldDX, worldDY, mapW, mapH) {
  view.ox += worldDX;
  view.oy += worldDY;
  clampPan(mapW, mapH);
}
