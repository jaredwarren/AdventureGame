// The map browser: a filterable list plus the 10x10 world grid.
//
// Thumbnails are painted one pixel per TILE from Tile.SwatchColor, not from the
// tile atlas. Drawing 300 atlas tiles per map across 100 maps and downscaling
// 16x would be 30,000 draw calls producing noise; a 20x15 ImageData is 300
// writes and reads correctly at any size. That is exactly what SwatchColor is
// for.

import { state, subscribe, emit } from '../store.js';
import { api } from '../api.js';
import { hexToRGBA } from '../tileart.js';

let gridHost, listHost, searchEl;
const thumbCache = new Map(); // id -> {canvas, etag}
const pending = new Set();
let queue = [];
let active = 0;
const MAX_PARALLEL = 6;

export function initBrowser(onOpen) {
  gridHost = document.getElementById('world-grid');
  listHost = document.getElementById('map-lists');
  searchEl = document.getElementById('map-search');

  searchEl.addEventListener('input', () => {
    state.ui.mapQuery = searchEl.value.trim().toLowerCase();
    renderLists(onOpen);
  });

  subscribe('maps', () => { renderGrid(onOpen); renderLists(onOpen); });
  subscribe('doc', markCurrent);
  subscribe('history', markCurrent);
}

// ---- the world grid ----

function renderGrid(onOpen) {
  if (!gridHost || !state.schema) return;
  const { gridCols, gridRows } = state.schema.constants;
  gridHost.replaceChildren();
  gridHost.style.setProperty('--cols', String(gridCols));

  // The LETTER is the row (y, north to south, A..J) and the NUMBER is the column
  // (x, west to east, 1..10), matching the game's overworld map (scenes/map.go).
  gridHost.append(el('span', 'grid-corner', ''));
  for (let col = 0; col < gridCols; col++) {
    gridHost.append(el('span', 'grid-head', String(col + 1)));
  }

  for (let row = 0; row < gridRows; row++) {
    const letter = String.fromCharCode(65 + row);
    gridHost.append(el('span', 'grid-head', letter));
    for (let col = 0; col < gridCols; col++) {
      gridHost.append(gridCell(`${letter}-${col + 1}`, onOpen));
    }
  }
}

function gridCell(id, onOpen) {
  const info = state.mapIndex.get(id);
  const b = document.createElement('button');
  b.type = 'button';
  b.className = 'grid-cell';
  b.dataset.mapId = id;
  b.title = info
    ? `${id}\n${info.width}x${info.height}\n${describeMarkers(info)}`
    : `${id} (missing)`;

  if (!info) {
    b.classList.add('missing');
    b.disabled = true;
    return b;
  }
  b.addEventListener('click', () => onOpen(id));
  observeThumb(b, id);
  return b;
}

function describeMarkers(info) {
  const entries = Object.entries(info.markerCounts ?? {});
  return entries.length ? entries.map(([k, v]) => `${k} ${v}`).join(', ') : 'no markers';
}

// ---- the flat lists ----

function renderLists(onOpen) {
  if (!listHost) return;
  listHost.replaceChildren();

  const q = state.ui.mapQuery;
  const groups = [
    ['standalone', 'Standalone'],
    ['room', 'Dungeon rooms'],
    ['grid', 'World grid'],
  ];

  for (const [group, label] of groups) {
    const maps = state.maps.filter((m) => m.group === group && (!q || m.id.toLowerCase().includes(q)));
    if (!maps.length) continue;

    const details = document.createElement('details');
    details.className = 'map-group';
    details.open = group !== 'grid' || !!q;

    const summary = document.createElement('summary');
    summary.innerHTML = `<span>${label}</span><span class="count">${maps.length}</span>`;
    details.append(summary);

    const ul = document.createElement('ul');
    ul.className = 'map-list';
    for (const m of maps) {
      const li = document.createElement('li');
      const b = document.createElement('button');
      b.type = 'button';
      b.className = 'map-item';
      b.dataset.mapId = m.id;
      b.textContent = m.id;
      if (m.parseError) {
        b.classList.add('broken');
        b.title = m.parseError;
      }
      b.addEventListener('click', () => onOpen(m.id));
      li.append(b);
      ul.append(li);
    }
    details.append(ul);
    listHost.append(details);
  }
  markCurrent();
}

function markCurrent() {
  const id = state.doc?.id;
  for (const el of document.querySelectorAll('[data-map-id]')) {
    const isCurrent = el.dataset.mapId === id;
    el.classList.toggle('current', isCurrent);
    if (isCurrent) el.setAttribute('aria-current', 'true');
    else el.removeAttribute('aria-current');
  }
}

// ---- thumbnails ----

/** Loads a thumbnail only once its cell scrolls into view. */
function observeThumb(cell, id) {
  const cached = thumbCache.get(id);
  if (cached) {
    attachThumb(cell, cached.canvas);
    return;
  }
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      io.disconnect();
      enqueue(id, (canvas) => attachThumb(cell, canvas));
    }
  }, { rootMargin: '200px' });
  io.observe(cell);
}

function attachThumb(cell, canvas) {
  cell.replaceChildren();
  const img = canvas.cloneNode(true);
  img.getContext('2d').drawImage(canvas, 0, 0);
  img.className = 'thumb';
  cell.append(img);
}

function enqueue(id, cb) {
  if (pending.has(id)) return;
  pending.add(id);
  queue.push({ id, cb });
  pump();
}

function pump() {
  while (active < MAX_PARALLEL && queue.length) {
    const job = queue.shift();
    active++;
    api.thumb(job.id)
      .then((t) => {
        const canvas = paintThumb(t);
        thumbCache.set(job.id, { canvas, etag: t.etag });
        job.cb(canvas);
      })
      .catch(() => { /* a broken map simply has no thumbnail */ })
      .finally(() => { active--; pending.delete(job.id); pump(); });
  }
}

/** Expands the RLE payload into one pixel per tile. */
function paintThumb(t) {
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(1, t.w);
  canvas.height = Math.max(1, t.h);
  const ctx = canvas.getContext('2d');
  const img = ctx.createImageData(canvas.width, canvas.height);

  let p = 0;
  for (let i = 0; i < t.rle.length; i += 2) {
    const [r, g, b, a] = swatchRGBA(t.rle[i]);
    for (let n = 0; n < t.rle[i + 1]; n++, p++) {
      const o = p * 4;
      img.data[o] = r; img.data[o + 1] = g; img.data[o + 2] = b; img.data[o + 3] = a;
    }
  }
  ctx.putImageData(img, 0, 0);

  // Marker dots on top answer "which cells have a spawn / a door / nothing" at a
  // glance, which is the main thing the grid is for.
  const types = state.schema?.markers ?? [];
  for (const [tx, ty, typeIdx] of t.markers ?? []) {
    ctx.fillStyle = markerDotColor(types[typeIdx]?.type);
    ctx.fillRect(tx, ty, 1, 1);
  }
  return canvas;
}

function swatchRGBA(gid) {
  if (gid === 0) return [0, 0, 0, 0];
  return state.atlas?.swatchRGBA.get(gid) ?? hexToRGBA('#303038');
}

function markerDotColor(type) {
  return {
    spawn: '#7cff9a', enemy: '#ff6b60', pickup: '#ffe14d',
    door: '#ffab3d', shrine: '#c39bff', sign: '#e0c39a',
  }[type] ?? '#ffffff';
}

/** Repaints one cell straight from the in-memory document, after a save. */
export function refreshThumb(id) {
  thumbCache.delete(id);
  const cells = document.querySelectorAll(`[data-map-id="${CSS.escape(id)}"].grid-cell`);
  if (!cells.length) return;
  enqueue(id, (canvas) => cells.forEach((c) => attachThumb(c, canvas)));
}

export function focusSearch() {
  searchEl?.focus();
  searchEl?.select();
}

/** The neighbouring grid cell in a direction, or null at the edge. */
export function neighbourID(dir) {
  const grid = state.doc?.grid;
  if (!grid) return null;
  const { gridCols, gridRows } = state.schema.constants;
  let { col, row } = grid;
  if (dir === 'w') col--;
  if (dir === 'e') col++;
  if (dir === 'n') row--;
  if (dir === 's') row++;
  if (col < 0 || row < 0 || col >= gridCols || row >= gridRows) return null;
  const id = `${String.fromCharCode(65 + row)}-${col + 1}`;
  return state.mapIndex.has(id) ? id : null;
}

/** Updates the four adjacency chips floating over the canvas edges. */
export function renderAdjacency(onOpen) {
  for (const dir of ['n', 's', 'e', 'w']) {
    const chip = document.getElementById(`adj-${dir}`);
    if (!chip) continue;
    const id = neighbourID(dir);
    chip.hidden = !id;
    if (!id) continue;
    chip.textContent = { n: '↑', s: '↓', e: '→', w: '←' }[dir] + ' ' + id;
    chip.onclick = () => onOpen(id);
  }
}

function el(tag, cls, text) {
  const e = document.createElement(tag);
  e.className = cls;
  e.textContent = text;
  return e;
}

export { emit };
