// Draw tile.Art shapes onto a Canvas2D context (tile-local coords → pixels).

export function drawArt(ctx, art, ox, oy, scale, opts = {}) {
  drawShapes(ctx, art.layers || [], ox, oy, scale);

  if (art.spatial?.variants?.length) {
    const tx = opts.tx ?? 0;
    const ty = opts.ty ?? 0;
    const idx = opts.variantIndex ?? pickVariant(art.spatial.variants, spatialHash(tx, ty));
    if (idx >= 0) {
      drawShapes(ctx, art.spatial.variants[idx].layers || [], ox, oy, scale);
    }
  }
}

export function spatialHash(tx, ty) {
  let h = Math.imul(tx, 374761393) + Math.imul(ty, 668265263);
  h = (h ^ 0x5bf03635) >>> 0;
  return h;
}

export function pickVariant(variants, roll) {
  let r = roll % 100;
  let cum = 0;
  for (let i = 0; i < variants.length; i++) {
    cum += variants[i].weight | 0;
    if (r < cum) return i;
  }
  return Math.max(0, variants.length - 1);
}

function strokeWidthPx(sw, scale, size) {
  const s = size || 16;
  return Math.max(0.5, (sw || 1) * (scale / s));
}

function drawShapes(ctx, shapes, ox, oy, scale) {
  for (const s of shapes) drawShape(ctx, s, ox, oy, scale);
}

function num(v, fallback = 0) {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

function drawShape(ctx, s, ox, oy, scale) {
  const size = 16;
  const sw = strokeWidthPx(s.strokeWidth, scale, size);
  ctx.lineCap = 'butt';
  ctx.lineJoin = 'miter';
  switch (s.type) {
    case 'rect': {
      const x = ox + num(s.x) * scale, y = oy + num(s.y) * scale;
      const w = num(s.w) * scale, h = num(s.h) * scale;
      if (s.fill) { ctx.fillStyle = s.fill; ctx.fillRect(x, y, w, h); }
      if (s.stroke) {
        ctx.strokeStyle = s.stroke;
        ctx.lineWidth = sw;
        ctx.strokeRect(x, y, w, h);
      }
      break;
    }
    case 'line': {
      if (!s.stroke) break;
      ctx.strokeStyle = s.stroke;
      ctx.lineWidth = sw;
      ctx.beginPath();
      ctx.moveTo(ox + num(s.x1) * scale, oy + num(s.y1) * scale);
      ctx.lineTo(ox + num(s.x2) * scale, oy + num(s.y2) * scale);
      ctx.stroke();
      break;
    }
    case 'circle': {
      const cx = ox + num(s.cx) * scale, cy = oy + num(s.cy) * scale, r = num(s.r) * scale;
      if (s.fill) {
        ctx.fillStyle = s.fill;
        ctx.beginPath(); ctx.arc(cx, cy, r, 0, Math.PI * 2); ctx.fill();
      }
      if (s.stroke) {
        ctx.strokeStyle = s.stroke;
        ctx.lineWidth = sw;
        ctx.beginPath(); ctx.arc(cx, cy, r, 0, Math.PI * 2); ctx.stroke();
      }
      break;
    }
    case 'path': {
      ctx.beginPath();
      for (const seg of (s.segs || [])) {
        switch (seg.op) {
          case 'M': ctx.moveTo(ox + num(seg.x) * scale, oy + num(seg.y) * scale); break;
          case 'L': ctx.lineTo(ox + num(seg.x) * scale, oy + num(seg.y) * scale); break;
          case 'Q': ctx.quadraticCurveTo(
            ox + num(seg.cx) * scale, oy + num(seg.cy) * scale,
            ox + num(seg.x) * scale, oy + num(seg.y) * scale); break;
          case 'Z': ctx.closePath(); break;
        }
      }
      const filled = s.filled !== undefined ? s.filled : !!s.fill;
      if (filled && s.fill) { ctx.fillStyle = s.fill; ctx.fill(); }
      else if (s.stroke) {
        ctx.strokeStyle = s.stroke;
        ctx.lineWidth = sw;
        ctx.stroke();
      }
      break;
    }
  }
}

export function hitTest(shapes, tx, ty, pad = 1.5) {
  const hits = hitTestAll(shapes, tx, ty, pad);
  return hits.length ? hits[0] : -1;
}

/** All shape indices under the point, front-most first. */
export function hitTestAll(shapes, tx, ty, pad = 1.5) {
  const hits = [];
  for (let i = shapes.length - 1; i >= 0; i--) {
    if (shapeContains(shapes[i], tx, ty, pad)) hits.push(i);
  }
  // Prefer details (lines/circles/small rects) over full-tile fill plates.
  hits.sort((a, b) => shapePickScore(shapes[a]) - shapePickScore(shapes[b]));
  return hits;
}

function shapePickScore(s) {
  if (!s) return 1000;
  if (s.type === 'line') return 0;
  if (s.type === 'circle') return 1;
  if (s.type === 'path') return 2;
  if (s.type === 'rect') {
    const area = Math.max(0, num(s.w)) * Math.max(0, num(s.h));
    // Full-ish tile fills are last so grass blades stay clickable.
    if (area >= 14 * 14) return 50;
    return 3 + area / 64;
  }
  return 10;
}

/**
 * Handle under the cursor for a selected shape, in tile coords.
 * pad is in tile units (caller scales from screen pixels).
 */
export function hitHandle(s, tx, ty, pad = 1.0) {
  if (!s) return null;
  const near = (x, y) => Math.hypot(tx - num(x), ty - num(y)) <= pad;

  switch (s.type) {
    case 'line':
      if (near(s.x1, s.y1)) return { kind: 'p1' };
      if (near(s.x2, s.y2)) return { kind: 'p2' };
      return null;
    case 'rect': {
      const x = num(s.x), y = num(s.y), w = num(s.w), h = num(s.h);
      if (near(x, y)) return { kind: 'nw' };
      if (near(x + w, y)) return { kind: 'ne' };
      if (near(x, y + h)) return { kind: 'sw' };
      if (near(x + w, y + h)) return { kind: 'se' };
      return null;
    }
    case 'circle': {
      const cx = num(s.cx), cy = num(s.cy), r = num(s.r);
      // Cardinal radius handles
      if (near(cx + r, cy)) return { kind: 'radius' };
      if (near(cx - r, cy)) return { kind: 'radius' };
      if (near(cx, cy + r)) return { kind: 'radius' };
      if (near(cx, cy - r)) return { kind: 'radius' };
      return null;
    }
    default:
      return null;
  }
}

function shapeContains(s, x, y, pad) {
  switch (s.type) {
    case 'rect': {
      const rx = num(s.x), ry = num(s.y), rw = num(s.w), rh = num(s.h);
      // Large fills: only the border is a hit target so overlays stay selectable.
      if (rw * rh >= 14 * 14) {
        const inset = Math.max(pad, 1.2);
        const inOuter = x >= rx - pad && x <= rx + rw + pad && y >= ry - pad && y <= ry + rh + pad;
        const inInner = x >= rx + inset && x <= rx + rw - inset && y >= ry + inset && y <= ry + rh - inset;
        return inOuter && !inInner;
      }
      return x >= rx - pad && x <= rx + rw + pad && y >= ry - pad && y <= ry + rh + pad;
    }
    case 'line':
      return distToSeg(x, y, num(s.x1), num(s.y1), num(s.x2), num(s.y2)) <= pad + (s.strokeWidth || 1);
    case 'circle':
      return Math.hypot(x - num(s.cx), y - num(s.cy)) <= num(s.r) + pad;
    default:
      return false;
  }
}

function distToSeg(px, py, x1, y1, x2, y2) {
  const dx = x2 - x1, dy = y2 - y1;
  const len2 = dx * dx + dy * dy;
  if (len2 === 0) return Math.hypot(px - x1, py - y1);
  let t = ((px - x1) * dx + (py - y1) * dy) / len2;
  t = Math.max(0, Math.min(1, t));
  return Math.hypot(px - (x1 + t * dx), py - (y1 + t * dy));
}

export function newShapeId(prefix = 's') {
  return prefix + '_' + Math.random().toString(36).slice(2, 9);
}

/** Draw selection handles for the active shape. */
export function drawSelection(ctx, s, ox, oy, scale) {
  if (!s) return;
  ctx.save();
  ctx.strokeStyle = '#5b8dee';
  ctx.lineWidth = 1;
  ctx.setLineDash([4, 3]);
  let box;
  switch (s.type) {
    case 'rect':
      box = { x: num(s.x), y: num(s.y), w: num(s.w), h: num(s.h) };
      break;
    case 'circle': {
      const cx = num(s.cx), cy = num(s.cy), r = num(s.r);
      box = { x: cx - r, y: cy - r, w: r * 2, h: r * 2 };
      ctx.beginPath();
      ctx.moveTo(ox + (cx - r) * scale, oy + cy * scale);
      ctx.lineTo(ox + (cx + r) * scale, oy + cy * scale);
      ctx.moveTo(ox + cx * scale, oy + (cy - r) * scale);
      ctx.lineTo(ox + cx * scale, oy + (cy + r) * scale);
      ctx.setLineDash([3, 3]);
      ctx.stroke();
      ctx.setLineDash([]);
      drawHandle(ctx, ox + (cx + r) * scale, oy + cy * scale);
      drawHandle(ctx, ox + (cx - r) * scale, oy + cy * scale);
      drawHandle(ctx, ox + cx * scale, oy + (cy + r) * scale);
      drawHandle(ctx, ox + cx * scale, oy + (cy - r) * scale);
      ctx.restore();
      return;
    }
    case 'line':
      ctx.beginPath();
      ctx.moveTo(ox + num(s.x1) * scale, oy + num(s.y1) * scale);
      ctx.lineTo(ox + num(s.x2) * scale, oy + num(s.y2) * scale);
      ctx.stroke();
      ctx.setLineDash([]);
      drawHandle(ctx, ox + num(s.x1) * scale, oy + num(s.y1) * scale);
      drawHandle(ctx, ox + num(s.x2) * scale, oy + num(s.y2) * scale);
      ctx.restore();
      return;
    default:
      ctx.restore();
      return;
  }
  ctx.strokeRect(ox + box.x * scale, oy + box.y * scale, box.w * scale, box.h * scale);
  ctx.setLineDash([]);
  const corners = [
    [box.x, box.y], [box.x + box.w, box.y],
    [box.x, box.y + box.h], [box.x + box.w, box.y + box.h],
  ];
  for (const [hx, hy] of corners) drawHandle(ctx, ox + hx * scale, oy + hy * scale);
  ctx.restore();
}

function drawHandle(ctx, x, y) {
  ctx.fillStyle = '#fff';
  ctx.strokeStyle = '#5b8dee';
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.rect(x - 4, y - 4, 8, 8);
  ctx.fill();
  ctx.stroke();
}
