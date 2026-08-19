// Tile art: replays the server's draw-op programs into a GPU-friendly atlas.
//
// The game draws tiles as procedural vector art (internal/world/tile/draw.go),
// not sprites. Rather than reimplement ~54 drawers in JavaScript, the Go side
// records each drawer's calls against the headless tile.Canvas interface and
// ships them as JSON ops. The six primitives map 1:1 onto Canvas2D, so replaying
// them here is exact, and the art stays defined in exactly one place.
//
// See internal/editorweb/canvasrec.go for the wire format.

export const TILE = 16;

/**
 * Replay one tile's op program into a 2D context, offset by (dx, dy).
 *
 * Unknown opcodes are skipped rather than throwing: a primitive added to
 * tile.Canvas in Go should degrade to a missing detail, never a blank editor.
 *
 * @param {CanvasRenderingContext2D} ctx
 * @param {Array<Array>} ops
 * @param {string[]} colors  shared color table; ops carry an index into it
 */
export function replayOps(ctx, ops, colors, dx = 0, dy = 0) {
  ctx.save();
  ctx.translate(dx, dy);
  // Match Ebiten's vector defaults so strokes land the same way.
  ctx.lineCap = 'butt';
  ctx.lineJoin = 'miter';
  ctx.miterLimit = 10;

  for (const op of ops) {
    switch (op[0]) {
      case 'fr': // ["fr", x, y, w, h, color]
        ctx.fillStyle = colors[op[5]];
        ctx.fillRect(op[1], op[2], op[3], op[4]);
        break;

      case 'sr': // ["sr", x, y, w, h, strokeWidth, color]
        ctx.strokeStyle = colors[op[6]];
        ctx.lineWidth = op[5];
        ctx.strokeRect(op[1], op[2], op[3], op[4]);
        break;

      case 'sl': // ["sl", x1, y1, x2, y2, strokeWidth, color]
        ctx.strokeStyle = colors[op[6]];
        ctx.lineWidth = op[5];
        ctx.beginPath();
        ctx.moveTo(op[1], op[2]);
        ctx.lineTo(op[3], op[4]);
        ctx.stroke();
        break;

      case 'fc': // ["fc", cx, cy, r, color]
        ctx.fillStyle = colors[op[4]];
        ctx.beginPath();
        ctx.arc(op[1], op[2], op[3], 0, Math.PI * 2);
        ctx.fill();
        break;

      case 'sc': // ["sc", cx, cy, r, strokeWidth, color]
        ctx.strokeStyle = colors[op[5]];
        ctx.lineWidth = op[4];
        ctx.beginPath();
        ctx.arc(op[1], op[2], op[3], 0, Math.PI * 2);
        ctx.stroke();
        break;

      case 'p': { // ["p", segs, fill01, strokeWidth, color]
        const segs = op[1];
        ctx.beginPath();
        for (let i = 0; i < segs.length; ) {
          switch (segs[i]) {
            case 0: ctx.moveTo(segs[i + 1], segs[i + 2]); i += 3; break;
            case 1: ctx.lineTo(segs[i + 1], segs[i + 2]); i += 3; break;
            // Go stores the control point separately; the wire order already
            // matches quadraticCurveTo's argument order.
            case 2: ctx.quadraticCurveTo(segs[i + 1], segs[i + 2], segs[i + 3], segs[i + 4]); i += 5; break;
            case 3: ctx.closePath(); i += 1; break;
            default: i = segs.length; break; // unrecognized: abandon this path
          }
        }
        if (op[2]) {
          ctx.fillStyle = colors[op[4]];
          ctx.fill('nonzero'); // matches Go's FillRuleNonZero
        } else {
          ctx.strokeStyle = colors[op[4]];
          ctx.lineWidth = op[3];
          ctx.stroke();
        }
        break;
      }
      // default: unknown opcode, ignore
    }
  }
  ctx.restore();
}

/**
 * Number of tile slots per atlas row.
 *
 * The atlas holds one slot per registered GID plus one per animation frame,
 * which is a few hundred. Laid out as a single strip that would be ~6600px wide
 * — past the 4096px texture limit on older GPUs, forcing Chrome to tile the
 * texture or fall back to software raster in the hot tile loop. A grid keeps it
 * comfortably small (64x7 slots is 1024x112).
 */
const ATLAS_COLS = 64;

/**
 * TileAtlas packs every tile (and every animation frame) into one texture.
 *
 * A single source image means the tile loop never switches textures. Each slot's
 * source rect is derived from its index, so lookups stay arithmetic.
 */
export class TileAtlas {
  constructor(schema) {
    this.schema = schema;
    this.byGID = new Map();
    this.swatch = new Map();
    this.swatchRGBA = new Map();
    this.animated = [];
    this._build();
  }

  _build() {
    const { colors, tiles } = this.schema;

    // Lay out slots: every tile gets slot 0 (its static/first frame), then each
    // animated tile's remaining frames follow.
    let slot = 0;
    const plan = [];
    for (const t of tiles) {
      this.byGID.set(t.gid, t);
      this.swatch.set(t.gid, t.swatch);
      this.swatchRGBA.set(t.gid, hexToRGBA(t.swatch));
      t.slot = slot++;
      plan.push({ gid: t.gid, ops: t.ops, slot: t.slot });
    }
    for (const t of tiles) {
      if (!t.frames || t.frames.length < 2) continue;
      t.frameSlots = [t.slot];
      for (let i = 1; i < t.frames.length; i++) {
        t.frameSlots.push(slot);
        plan.push({ gid: t.gid, ops: t.frames[i], slot });
        slot++;
      }
      this.animated.push(t.gid);
    }

    const rows = Math.ceil(slot / ATLAS_COLS);
    const cvs = makeCanvas(ATLAS_COLS * TILE, rows * TILE);
    const ctx = cvs.getContext('2d', { alpha: true });
    ctx.imageSmoothingEnabled = false;
    for (const p of plan) {
      // GID 0 is a transparency hole, not black: the renderer skips it. Leave
      // its slot empty so a hole reads as a hole in the editor too.
      if (p.gid === 0) continue;
      replayOps(ctx, p.ops, colors, (p.slot % ATLAS_COLS) * TILE, Math.floor(p.slot / ATLAS_COLS) * TILE);
    }

    this.canvas = cvs;
    this.width = cvs.width;
    this.height = cvs.height;
    this.slotCount = slot;
  }

  /** The atlas slot index for a GID at an animation frame index, or -1. */
  slotOf(gid, frame = 0) {
    const t = this.byGID.get(gid);
    if (!t) return -1;
    if (t.frameSlots && t.frameSlots.length > 1) {
      return t.frameSlots[frame % t.frameSlots.length];
    }
    return t.slot;
  }

  /** Source rect in the atlas for a GID at an animation frame index. */
  srcX(gid, frame = 0) {
    const s = this.slotOf(gid, frame);
    return s < 0 ? -1 : (s % ATLAS_COLS) * TILE;
  }

  srcY(gid, frame = 0) {
    const s = this.slotOf(gid, frame);
    return s < 0 ? -1 : Math.floor(s / ATLAS_COLS) * TILE;
  }

  def(gid) { return this.byGID.get(gid); }
  name(gid) { return this.byGID.get(gid)?.name ?? `gid ${gid}`; }
  isAnimated(gid) { return (this.byGID.get(gid)?.frames?.length ?? 0) > 1; }

  /** Frame count for the animation loop, or 1 for static tiles. */
  frameCount(gid) { return this.byGID.get(gid)?.frameSlots?.length ?? 1; }

  /** Longest animation loop across all tiles, for the shared frame counter. */
  get maxFrames() {
    let n = 1;
    for (const gid of this.animated) n = Math.max(n, this.frameCount(gid));
    return n;
  }

  /**
   * A data URL of the atlas, used as a CSS background so palette swatches cost
   * no extra canvases.
   */
  cssURL() {
    if (!this._url) this._url = this.canvas.toDataURL();
    return this._url;
  }

  /** Draws one tile at 1:1 into an arbitrary context (palette, previews). */
  drawTile(ctx, gid, dx, dy, size = TILE, frame = 0) {
    const sx = this.srcX(gid, frame);
    if (sx < 0) return;
    ctx.drawImage(this.canvas, sx, this.srcY(gid, frame), TILE, TILE, dx, dy, size, size);
  }

  /** CSS background-position for a palette swatch, in swatch-sized units. */
  swatchOffset(gid, size) {
    const s = this.slotOf(gid, 0);
    if (s < 0) return '0 0';
    return `${-(s % ATLAS_COLS) * size}px ${-Math.floor(s / ATLAS_COLS) * size}px`;
  }

  /** CSS background-size for a palette swatch sheet. */
  swatchSheetSize(size) {
    return `${ATLAS_COLS * size}px ${(this.canvas.height / TILE) * size}px`;
  }
}

function makeCanvas(w, h) {
  const c = document.createElement('canvas');
  c.width = Math.max(1, w);
  c.height = Math.max(1, h);
  return c;
}

/** Parses "#rrggbb" / "#rrggbbaa" into [r,g,b,a] for the minimap's ImageData. */
export function hexToRGBA(hex) {
  const s = hex.replace('#', '');
  const v = parseInt(s.slice(0, 6), 16);
  const a = s.length >= 8 ? parseInt(s.slice(6, 8), 16) : 255;
  return [(v >> 16) & 255, (v >> 8) & 255, v & 255, a];
}
