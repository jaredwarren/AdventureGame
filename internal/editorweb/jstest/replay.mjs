// Replays tile draw-op programs through a recording mock of Canvas2D and prints
// a normalized trace, so a Go test can assert the browser's interpretation of
// the wire format matches what the recorder produced.
//
// Run: node replay.mjs <schema.json>

import { readFileSync } from 'node:fs';
import { replayOps } from '../static/js/tileart.js';

/** Records the Canvas2D calls replayOps makes, in order. */
class MockContext {
  constructor() {
    this.trace = [];
    this.fillStyle = '';
    this.strokeStyle = '';
    this.lineWidth = 0;
    this.lineCap = '';
    this.lineJoin = '';
    this.miterLimit = 0;
  }
  #n(v) { return Math.round(v * 1000) / 1000; }
  #push(op, ...args) { this.trace.push([op, ...args.map((a) => (typeof a === 'number' ? this.#n(a) : a))]); }

  save() {}
  restore() {}
  translate() {}
  beginPath() { this.#push('beginPath'); }
  moveTo(x, y) { this.#push('moveTo', x, y); }
  lineTo(x, y) { this.#push('lineTo', x, y); }
  quadraticCurveTo(cx, cy, x, y) { this.#push('quadTo', cx, cy, x, y); }
  closePath() { this.#push('closePath'); }
  arc(cx, cy, r) { this.#push('arc', cx, cy, r); }
  fillRect(x, y, w, h) { this.#push('fillRect', x, y, w, h, this.fillStyle); }
  strokeRect(x, y, w, h) { this.#push('strokeRect', x, y, w, h, this.lineWidth, this.strokeStyle); }
  fill(rule) { this.#push('fill', this.fillStyle, rule ?? 'nonzero'); }
  stroke() { this.#push('stroke', this.lineWidth, this.strokeStyle); }
}

const schema = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const out = {};
for (const tile of schema.tiles) {
  const ctx = new MockContext();
  replayOps(ctx, tile.ops, schema.colors);
  out[tile.gid] = ctx.trace;
}
process.stdout.write(JSON.stringify(out));
