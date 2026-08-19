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

/** Resolves the pickup kind for a pickup marker object. */
export function getPickupKind(obj) {
  if (obj.type !== 'pickup') return null;
  const props = obj.properties ?? [];
  const p = props.find((p) => p.name === 'kind');
  if (p && p.value) return String(p.value).toLowerCase();
  if (obj.name) {
    const n = obj.name.toLowerCase().replace(/[0-9_]/g, '');
    if (['coin', 'heart', 'bomb', 'key', 'torch', 'pegasusboots', 'boots', 'shield'].includes(n)) {
      if (n === 'pegasusboots' || n === 'boots') return 'pegasus_boots';
      return n;
    }
  }
  return 'coin';
}

/**
 * Draws a clean, stylized vector graphic for the marker inside its bounding box.
 * Returns true if an icon was drawn, or false to fall back to the text glyph.
 */
export function drawMarkerGraphic(ctx, obj, cx, cy, size) {
  if (obj.type === 'pickup') {
    const kind = getPickupKind(obj);
    switch (kind) {
      case 'heart': drawHeartIcon(ctx, cx, cy, size); break;
      case 'bomb': drawBombIcon(ctx, cx, cy, size); break;
      case 'coin': drawCoinIcon(ctx, cx, cy, size); break;
      case 'key':
      case 'small_key': drawKeyIcon(ctx, cx, cy, size); break;
      case 'torch': drawTorchIcon(ctx, cx, cy, size); break;
      case 'pegasus_boots':
      case 'boots': drawBootsIcon(ctx, cx, cy, size); break;
      case 'shield': drawShieldIcon(ctx, cx, cy, size); break;
      default: drawCoinIcon(ctx, cx, cy, size); break;
    }
    return true;
  }
  if (obj.type === 'enemy') {
    drawEnemyIcon(ctx, obj, cx, cy, size);
    return true;
  }
  if (obj.type === 'shrine') {
    drawShrineIcon(ctx, cx, cy, size);
    return true;
  }
  if (obj.type === 'sign') {
    drawSignIcon(ctx, cx, cy, size);
    return true;
  }
  if (obj.type === 'door') {
    drawDoorIcon(ctx, cx, cy, size);
    return true;
  }
  if (obj.type === 'spawn') {
    drawSpawnIcon(ctx, cx, cy, size);
    return true;
  }
  return false;
}

function drawHeartIcon(ctx, cx, cy, size) {
  const s = size * 0.52;
  const top = cy - s * 0.7;
  ctx.save();
  ctx.translate(cx, top);
  ctx.beginPath();
  const w = s * 0.9;
  const h = s * 1.5;
  ctx.moveTo(0, h * 0.3);
  ctx.bezierCurveTo(-w * 0.8, -h * 0.3, -w * 1.1, h * 0.4, 0, h * 0.95);
  ctx.bezierCurveTo(w * 1.1, h * 0.4, w * 0.8, -h * 0.3, 0, h * 0.3);
  ctx.closePath();
  ctx.fillStyle = 'rgba(235, 55, 65, 0.85)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(160, 20, 30, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.08);
  ctx.stroke();
  // Inner shine highlight
  ctx.beginPath();
  ctx.arc(-w * 0.35, h * 0.15, Math.max(1, size * 0.08), 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
  ctx.fill();
  ctx.restore();
}

function drawBombIcon(ctx, cx, cy, size) {
  const r = size * 0.35;
  const bodyY = cy + size * 0.1;
  ctx.save();
  // Fuse line
  ctx.beginPath();
  ctx.moveTo(cx, bodyY - r);
  ctx.quadraticCurveTo(cx + size * 0.2, bodyY - r - size * 0.25, cx + size * 0.28, bodyY - r - size * 0.15);
  ctx.strokeStyle = 'rgba(210, 180, 130, 0.85)';
  ctx.lineWidth = Math.max(1, size * 0.08);
  ctx.stroke();
  // Spark
  ctx.beginPath();
  ctx.arc(cx + size * 0.28, bodyY - r - size * 0.15, Math.max(1.5, size * 0.1), 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(255, 120, 0, 0.95)';
  ctx.fill();
  ctx.beginPath();
  ctx.arc(cx + size * 0.28, bodyY - r - size * 0.15, Math.max(1, size * 0.05), 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(255, 230, 80, 0.98)';
  ctx.fill();
  // Fuse cap
  ctx.fillStyle = 'rgba(130, 130, 140, 0.9)';
  ctx.fillRect(cx - size * 0.12, bodyY - r - size * 0.08, size * 0.24, size * 0.12);
  // Bomb body
  ctx.beginPath();
  ctx.arc(cx, bodyY, r, 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(35, 35, 42, 0.88)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(70, 70, 85, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.07);
  ctx.stroke();
  // Highlight
  ctx.beginPath();
  ctx.arc(cx - r * 0.35, bodyY - r * 0.35, r * 0.25, 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
  ctx.fill();
  ctx.restore();
}

function drawCoinIcon(ctx, cx, cy, size) {
  const r = size * 0.38;
  ctx.save();
  ctx.beginPath();
  ctx.arc(cx, cy, r, 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(255, 205, 30, 0.85)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(200, 120, 10, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.08);
  ctx.stroke();
  // Inner ring
  ctx.beginPath();
  ctx.arc(cx, cy, r * 0.65, 0, Math.PI * 2);
  ctx.strokeStyle = 'rgba(220, 150, 15, 0.8)';
  ctx.lineWidth = Math.max(0.75, size * 0.05);
  ctx.stroke();
  // Center bar
  ctx.beginPath();
  ctx.moveTo(cx, cy - r * 0.4);
  ctx.lineTo(cx, cy + r * 0.4);
  ctx.stroke();
  ctx.restore();
}

function drawKeyIcon(ctx, cx, cy, size) {
  ctx.save();
  const topY = cy - size * 0.22;
  const botY = cy + size * 0.35;
  const headR = size * 0.22;
  // Key ring / head
  ctx.beginPath();
  ctx.arc(cx, topY, headR, 0, Math.PI * 2);
  ctx.strokeStyle = 'rgba(255, 205, 30, 0.95)';
  ctx.lineWidth = Math.max(1.2, size * 0.1);
  ctx.stroke();
  // Shaft
  ctx.beginPath();
  ctx.moveTo(cx, topY + headR - 1);
  ctx.lineTo(cx, botY);
  ctx.stroke();
  // Teeth
  ctx.beginPath();
  ctx.moveTo(cx, botY - size * 0.15);
  ctx.lineTo(cx + size * 0.18, botY - size * 0.15);
  ctx.moveTo(cx, botY);
  ctx.lineTo(cx + size * 0.22, botY);
  ctx.stroke();
  ctx.restore();
}

function drawTorchIcon(ctx, cx, cy, size) {
  ctx.save();
  const w = size * 0.2;
  const h = size * 0.5;
  // Handle
  ctx.fillStyle = 'rgba(139, 90, 43, 0.88)';
  ctx.fillRect(cx - w * 0.5, cy - h * 0.1, w, h);
  // Metal collar
  ctx.fillStyle = 'rgba(80, 80, 85, 0.9)';
  ctx.fillRect(cx - w * 0.8, cy - h * 0.2, w * 1.6, h * 0.18);
  // Outer flame
  ctx.beginPath();
  ctx.moveTo(cx, cy - h * 0.9);
  ctx.quadraticCurveTo(cx - w * 1.5, cy - h * 0.4, cx - w * 0.7, cy - h * 0.2);
  ctx.lineTo(cx + w * 0.7, cy - h * 0.2);
  ctx.quadraticCurveTo(cx + w * 1.5, cy - h * 0.4, cx, cy - h * 0.9);
  ctx.fillStyle = 'rgba(255, 60, 10, 0.9)';
  ctx.fill();
  // Inner flame
  ctx.beginPath();
  ctx.moveTo(cx, cy - h * 0.75);
  ctx.quadraticCurveTo(cx - w * 0.9, cy - h * 0.35, cx - w * 0.4, cy - h * 0.2);
  ctx.lineTo(cx + w * 0.4, cy - h * 0.2);
  ctx.quadraticCurveTo(cx + w * 0.9, cy - h * 0.35, cx, cy - h * 0.75);
  ctx.fillStyle = 'rgba(255, 185, 20, 0.95)';
  ctx.fill();
  ctx.restore();
}

function drawBootsIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.45;
  // Boot shaft & foot
  ctx.beginPath();
  ctx.moveTo(-s * 0.4, -s * 0.7);
  ctx.lineTo(s * 0.3, -s * 0.7);
  ctx.lineTo(s * 0.3, s * 0.1);
  ctx.lineTo(s * 0.75, s * 0.3);
  ctx.quadraticCurveTo(s * 0.85, s * 0.7, s * 0.5, s * 0.7);
  ctx.lineTo(-s * 0.4, s * 0.7);
  ctx.closePath();
  ctx.fillStyle = 'rgba(185, 45, 45, 0.88)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(120, 20, 20, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.06);
  ctx.stroke();
  // Gold cuff
  ctx.fillStyle = 'rgba(255, 215, 0, 0.9)';
  ctx.fillRect(-s * 0.45, -s * 0.75, s * 0.8, s * 0.2);
  // Wing
  ctx.beginPath();
  ctx.moveTo(-s * 0.5, -s * 0.5);
  ctx.lineTo(-s * 0.9, -s * 0.1);
  ctx.lineTo(-s * 0.3, s * 0.15);
  ctx.closePath();
  ctx.fillStyle = 'rgba(255, 245, 180, 0.95)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(200, 170, 80, 0.85)';
  ctx.lineWidth = Math.max(0.75, size * 0.05);
  ctx.stroke();
  ctx.restore();
}

function drawShieldIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.42;
  ctx.beginPath();
  ctx.moveTo(0, -s * 0.8);
  ctx.lineTo(s * 0.75, -s * 0.6);
  ctx.quadraticCurveTo(s * 0.75, s * 0.3, 0, s * 0.9);
  ctx.quadraticCurveTo(-s * 0.75, s * 0.3, -s * 0.75, -s * 0.6);
  ctx.closePath();
  ctx.fillStyle = 'rgba(60, 110, 175, 0.85)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(220, 190, 60, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.08);
  ctx.stroke();
  // Inner crest / cross
  ctx.fillStyle = 'rgba(255, 215, 0, 0.9)';
  ctx.fillRect(-s * 0.1, -s * 0.4, s * 0.2, s * 0.8);
  ctx.fillRect(-s * 0.35, -s * 0.2, s * 0.7, s * 0.2);
  ctx.restore();
}

function drawShrineIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.42;
  ctx.beginPath();
  ctx.moveTo(0, -s * 0.85);
  ctx.lineTo(s * 0.7, 0);
  ctx.lineTo(0, s * 0.85);
  ctx.lineTo(-s * 0.7, 0);
  ctx.closePath();
  ctx.fillStyle = 'rgba(155, 110, 222, 0.85)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(215, 180, 255, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.08);
  ctx.stroke();
  // Inner facet
  ctx.beginPath();
  ctx.moveTo(0, -s * 0.85);
  ctx.lineTo(0, s * 0.85);
  ctx.moveTo(-s * 0.7, 0);
  ctx.lineTo(s * 0.7, 0);
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.5)';
  ctx.lineWidth = Math.max(0.75, size * 0.05);
  ctx.stroke();
  ctx.restore();
}

function drawSignIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.42;
  // Post
  ctx.fillStyle = 'rgba(139, 90, 43, 0.88)';
  ctx.fillRect(-s * 0.12, -s * 0.1, s * 0.24, s * 0.9);
  // Board
  ctx.fillStyle = 'rgba(200, 160, 106, 0.88)';
  ctx.fillRect(-s * 0.75, -s * 0.75, s * 1.5, s * 0.7);
  ctx.strokeStyle = 'rgba(139, 90, 43, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.06);
  ctx.strokeRect(-s * 0.75, -s * 0.75, s * 1.5, s * 0.7);
  // Text lines
  ctx.fillStyle = 'rgba(90, 50, 20, 0.7)';
  ctx.fillRect(-s * 0.55, -s * 0.55, s * 1.1, s * 0.1);
  ctx.fillRect(-s * 0.55, -s * 0.35, s * 0.8, s * 0.1);
  ctx.restore();
}

function drawDoorIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.42;
  // Archway
  ctx.beginPath();
  ctx.moveTo(-s * 0.6, s * 0.8);
  ctx.lineTo(-s * 0.6, -s * 0.2);
  ctx.quadraticCurveTo(-s * 0.6, -s * 0.8, 0, -s * 0.8);
  ctx.quadraticCurveTo(s * 0.6, -s * 0.8, s * 0.6, -s * 0.2);
  ctx.lineTo(s * 0.6, s * 0.8);
  ctx.closePath();
  ctx.fillStyle = 'rgba(224, 144, 24, 0.8)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(255, 200, 80, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.07);
  ctx.stroke();
  // Inner doorway opening
  ctx.beginPath();
  ctx.moveTo(-s * 0.35, s * 0.8);
  ctx.lineTo(-s * 0.35, -s * 0.1);
  ctx.quadraticCurveTo(-s * 0.35, -s * 0.55, 0, -s * 0.55);
  ctx.quadraticCurveTo(s * 0.35, -s * 0.55, s * 0.35, -s * 0.1);
  ctx.lineTo(s * 0.35, s * 0.8);
  ctx.fillStyle = 'rgba(30, 20, 10, 0.9)';
  ctx.fill();
  ctx.restore();
}

function drawSpawnIcon(ctx, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.38;
  // Player figure / hero silhouette
  ctx.beginPath();
  ctx.arc(0, -s * 0.45, s * 0.35, 0, Math.PI * 2);
  ctx.fillStyle = 'rgba(70, 209, 122, 0.88)';
  ctx.fill();
  ctx.strokeStyle = 'rgba(30, 130, 60, 0.95)';
  ctx.lineWidth = Math.max(1, size * 0.06);
  ctx.stroke();
  // Body
  ctx.beginPath();
  ctx.moveTo(-s * 0.55, s * 0.75);
  ctx.lineTo(-s * 0.45, -s * 0.05);
  ctx.lineTo(s * 0.45, -s * 0.05);
  ctx.lineTo(s * 0.55, s * 0.75);
  ctx.closePath();
  ctx.fillStyle = 'rgba(70, 209, 122, 0.88)';
  ctx.fill();
  ctx.stroke();
  ctx.restore();
}

function drawEnemyIcon(ctx, obj, cx, cy, size) {
  ctx.save();
  ctx.translate(cx, cy);
  const s = size * 0.42;
  const name = (obj.name ?? '').toLowerCase();
  if (name.includes('bat')) {
    // Bat: wings + body
    ctx.beginPath();
    ctx.moveTo(0, s * 0.2);
    ctx.quadraticCurveTo(-s * 0.6, -s * 0.7, -s * 0.9, -s * 0.2);
    ctx.quadraticCurveTo(-s * 0.5, s * 0.4, 0, s * 0.5);
    ctx.quadraticCurveTo(s * 0.5, s * 0.4, s * 0.9, -s * 0.2);
    ctx.quadraticCurveTo(s * 0.6, -s * 0.7, 0, s * 0.2);
    ctx.fillStyle = 'rgba(224, 87, 79, 0.88)';
    ctx.fill();
    ctx.strokeStyle = 'rgba(140, 25, 20, 0.95)';
    ctx.lineWidth = Math.max(1, size * 0.06);
    ctx.stroke();
  } else if (name.includes('boss') || name.includes('knight')) {
    // Skull / Helmet
    ctx.beginPath();
    ctx.arc(0, -s * 0.15, s * 0.65, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(224, 87, 79, 0.9)';
    ctx.fill();
    ctx.strokeStyle = 'rgba(140, 25, 20, 0.95)';
    ctx.lineWidth = Math.max(1, size * 0.07);
    ctx.stroke();
    // Eyes
    ctx.fillStyle = 'rgba(30, 10, 10, 0.9)';
    ctx.beginPath();
    ctx.arc(-s * 0.22, -s * 0.15, s * 0.15, 0, Math.PI * 2);
    ctx.arc(s * 0.22, -s * 0.15, s * 0.15, 0, Math.PI * 2);
    ctx.fill();
  } else {
    // Slime: round mound with eyes
    ctx.beginPath();
    ctx.moveTo(-s * 0.75, s * 0.6);
    ctx.quadraticCurveTo(-s * 0.85, -s * 0.4, 0, -s * 0.65);
    ctx.quadraticCurveTo(s * 0.85, -s * 0.4, s * 0.75, s * 0.6);
    ctx.closePath();
    ctx.fillStyle = 'rgba(224, 87, 79, 0.88)';
    ctx.fill();
    ctx.strokeStyle = 'rgba(140, 25, 20, 0.95)';
    ctx.lineWidth = Math.max(1, size * 0.06);
    ctx.stroke();
    // Slime eyes
    ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
    ctx.beginPath();
    ctx.arc(-s * 0.22, -s * 0.1, s * 0.15, 0, Math.PI * 2);
    ctx.arc(s * 0.22, -s * 0.1, s * 0.15, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillStyle = 'rgba(30, 10, 10, 0.9)';
    ctx.beginPath();
    ctx.arc(-s * 0.22, -s * 0.1, s * 0.08, 0, Math.PI * 2);
    ctx.arc(s * 0.22, -s * 0.1, s * 0.08, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.restore();
}
