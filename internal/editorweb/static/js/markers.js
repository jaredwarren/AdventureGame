// Marker geometry and drawing.
//
// The selection box for a marker is NOT simply its x/y/width/height. Spawn,
// enemy and pickup markers are FEET-anchored: the object's y is the BOTTOM of
// the box, so the rect starts at y - hitboxHeight. Door, shrine and sign use
// their own size, clamping a non-positive width or height to 16.
//
// None of that is written out here. The server probes
// world.MarkerObjectHitRect and ships affine coefficients, verified against the
// real function over a dense input grid before it will even start. So this file
// only evaluates a formula it was given.

import { state, markerSchema } from './store.js';

/** Evaluates one axis of a hit-rect model. */
function axis(m, o) {
  return (m.k ?? 0) + (m.x ?? 0) * o.x + (m.y ?? 0) * o.y +
         (m.w ?? 0) * (o.width ?? 0) + (m.h ?? 0) * (o.height ?? 0);
}

/** The selection box for a marker object, in world pixels. */
export function hitRect(obj) {
  const schema = markerSchema(obj.type);
  const model = schema?.hitRect ?? state.schema?.unknownMarkerHit;
  if (!model) return { x: obj.x, y: obj.y, w: 16, h: 16 };

  const width = obj.width ?? 0;
  const height = obj.height ?? 0;
  return {
    x: axis(model.x, obj),
    y: axis(model.y, obj),
    w: width <= 0 && model.wWhenNonPositive ? model.wWhenNonPositive : axis(model.w, obj),
    h: height <= 0 && model.hWhenNonPositive ? model.hWhenNonPositive : axis(model.h, obj),
  };
}

/**
 * Topmost marker at a world point, or null.
 *
 * Iterates backwards so the object drawn last (on top) wins, matching the
 * in-game editor's picking order.
 */
export function markerAt(wx, wy) {
  const objects = state.doc?.markersLayer?.objects ?? [];
  for (let i = objects.length - 1; i >= 0; i--) {
    const o = objects[i];
    const r = hitRect(o);
    if (wx >= r.x && wx <= r.x + r.w && wy >= r.y && wy <= r.y + r.h) return o;
  }
  return null;
}

// Per-type colors, used for both the canvas overlay and the minimap dots.
export const MARKER_COLORS = {
  spawn: '#46d17a',
  enemy: '#e0574f',
  pickup: '#e8c33f',
  door: '#e09018',
  shrine: '#9b6ede',
  sign: '#c8a06a',
};

export function markerColor(type) {
  return MARKER_COLORS[type] ?? '#8fa6c8';
}

/** Short glyph drawn inside a marker box when there is room. */
export function markerGlyph(type) {
  return { spawn: 'P', enemy: 'E', pickup: 'i', door: 'D', shrine: 'S', sign: 'T' }[type] ?? '?';
}

/**
 * One-line summary for the status bar and the property panel header.
 */
export function markerSummary(obj) {
  const bits = [`${obj.type} #${obj.id}`];
  if (obj.name) bits.push(`"${obj.name}"`);
  const props = obj.properties ?? [];
  const show = { door: 'target_map', pickup: 'kind', sign: 'text' }[obj.type];
  if (show) {
    const v = props.find((p) => p.name === show)?.value;
    if (v !== undefined) {
      const s = String(v);
      bits.push(`${show}=${s.length > 24 ? s.slice(0, 24) + '…' : s}`);
    }
  }
  if (obj.type === 'enemy') {
    const get = (n) => props.find((p) => p.name === n)?.value;
    bits.push(`hp=${get('hp') ?? '?'}`, `spd=${get('speed') ?? '?'}`);
    if (get('is_boss')) bits.push('BOSS');
  }
  return bits.join('  ');
}
