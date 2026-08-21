import { api, ApiError } from './api.js';
import {
  drawArt, drawSelection, hitHandle, hitTest, hitTestAll, newShapeId,
} from './draw.js';

const SIZE = 16;

const els = {
  boot: document.getElementById('boot'),
  app: document.getElementById('app'),
  list: document.getElementById('tile-list'),
  search: document.getElementById('tile-search'),
  canvas: document.getElementById('edit-canvas'),
  preview: document.getElementById('preview-canvas'),
  title: document.getElementById('tile-title'),
  shapeList: document.getElementById('shape-list'),
  variantTabs: document.getElementById('variant-tabs'),
  dirty: document.getElementById('dirty-dot'),
  status: document.getElementById('status-msg'),
  spatialToggle: document.getElementById('toggle-spatial'),
  weightRow: document.getElementById('weight-row'),
  weight: document.getElementById('prop-weight'),
  fill: document.getElementById('prop-fill'),
  stroke: document.getElementById('prop-stroke'),
  sw: document.getElementById('prop-sw'),
  zoom: document.getElementById('zoom-select'),
};

const state = {
  tiles: [],
  art: null,
  dirty: false,
  tool: 'select',
  selected: -1,
  /** 'base' | number (variant index) */
  editTarget: 'base',
  scale: 32,
  undo: [],
  redo: [],
  drag: null,
};

const ctx = els.canvas.getContext('2d');
const pctx = els.preview.getContext('2d');

function setStatus(msg) { els.status.textContent = msg; }

function clone(v) { return JSON.parse(JSON.stringify(v)); }

function pushUndo() {
  if (!state.art) return;
  state.undo.push(clone(state.art));
  if (state.undo.length > 80) state.undo.shift();
  state.redo = [];
}

function markDirty(v = true) {
  state.dirty = v;
  els.dirty.hidden = !v;
}

function currentLayers() {
  if (!state.art) return [];
  if (state.editTarget === 'base') return state.art.layers;
  return state.art.spatial.variants[state.editTarget].layers;
}

function setCurrentLayers(layers) {
  if (state.editTarget === 'base') state.art.layers = layers;
  else state.art.spatial.variants[state.editTarget].layers = layers;
}

async function boot() {
  try {
    const data = await api.listTiles();
    state.tiles = data.tiles || [];
    renderList();
    els.boot.hidden = true;
    els.app.hidden = false;
    setStatus(`${state.tiles.length} tiles — pick one to edit`);
    const q = new URLSearchParams(location.search).get('gid');
    if (q) await loadGid(Number(q));
  } catch (e) {
    els.boot.innerHTML = `<h1>Tile art editor</h1><p class="fatal">${e.message}</p>`;
  }
}

function renderList(filter = '') {
  const f = filter.trim().toLowerCase();
  els.list.innerHTML = '';
  for (const t of state.tiles) {
    const label = `${t.gid} ${t.name}`;
    if (f && !label.toLowerCase().includes(f)) continue;
    const btn = document.createElement('button');
    btn.type = 'button';
    if (state.art?.gid === t.gid) btn.classList.add('active');
    btn.innerHTML = `<span class="tile-swatch" style="background:${t.swatch}"></span>
      <span><div>${t.name}</div><div class="tile-meta">gid ${t.gid}${t.hasArt ? ' · art' : ''}${t.hasSpatial ? ' · spatial' : ''}</div></span>`;
    btn.addEventListener('click', () => loadGid(t.gid));
    els.list.appendChild(btn);
  }
}

els.search.addEventListener('input', () => renderList(els.search.value));

async function loadGid(gid) {
  if (state.dirty && !confirm('Discard unsaved changes?')) return;
  try {
    const data = await api.loadTile(gid);
    state.art = data.art;
    state.selected = -1;
    state.editTarget = 'base';
    state.undo = [];
    state.redo = [];
    markDirty(false);
    els.title.textContent = `${state.art.name} (${state.art.gid})`;
    els.spatialToggle.checked = !!(state.art.spatial && state.art.spatial.variants?.length);
    updateSpatialUI();
    renderList(els.search.value);
    redraw();
    setStatus(data.synthesized
      ? `Synthesized from Go drawer — save to create ${gid}_*.tile.json`
      : (data.path || 'Loaded'));
  } catch (e) {
    setStatus(e.message);
  }
}

function updateSpatialUI() {
  const on = els.spatialToggle.checked;
  els.weightRow.hidden = !on;
  els.variantTabs.hidden = !on;
  if (on && !state.art.spatial) {
    state.art.spatial = {
      mode: 'gridHash',
      variants: [
        { id: 'variant_a', weight: 50, layers: [] },
        { id: 'variant_b', weight: 50, layers: [] },
      ],
    };
  }
  if (!on) {
    state.art.spatial = null;
    state.editTarget = 'base';
  }
  renderVariantTabs();
}

function renderVariantTabs() {
  els.variantTabs.innerHTML = '';
  if (!state.art?.spatial) return;
  const mk = (label, target) => {
    const b = document.createElement('button');
    b.type = 'button';
    b.textContent = label;
    if (state.editTarget === target) b.classList.add('active');
    b.addEventListener('click', () => {
      state.editTarget = target;
      state.selected = -1;
      renderVariantTabs();
      redraw();
    });
    els.variantTabs.appendChild(b);
  };
  mk('Base', 'base');
  state.art.spatial.variants.forEach((v, i) => {
    mk(`${v.id} (${v.weight})`, i);
  });
  if (typeof state.editTarget === 'number') {
    els.weight.value = state.art.spatial.variants[state.editTarget].weight;
  }
}

els.spatialToggle.addEventListener('change', () => {
  pushUndo();
  updateSpatialUI();
  markDirty();
  redraw();
});

document.getElementById('btn-add-variant').addEventListener('click', () => {
  if (!state.art.spatial) return;
  pushUndo();
  const n = state.art.spatial.variants.length + 1;
  // Shrink others proportionally later; for now set weight 1 and fix on save prompt
  state.art.spatial.variants.push({
    id: `variant_${n}`,
    weight: 1,
    layers: [],
  });
  normalizeWeights();
  state.editTarget = state.art.spatial.variants.length - 1;
  updateSpatialUI();
  markDirty();
  redraw();
});

els.weight.addEventListener('change', () => {
  if (typeof state.editTarget !== 'number' || !state.art.spatial) return;
  pushUndo();
  state.art.spatial.variants[state.editTarget].weight = Math.max(1, Number(els.weight.value) || 1);
  normalizeWeights();
  updateSpatialUI();
  markDirty();
  redraw();
});

function normalizeWeights() {
  const vars = state.art.spatial.variants;
  const sum = vars.reduce((a, v) => a + (v.weight | 0), 0);
  if (sum === 100) return;
  // Scale to 100, fix remainder on last.
  let allocated = 0;
  for (let i = 0; i < vars.length; i++) {
    if (i === vars.length - 1) {
      vars[i].weight = Math.max(1, 100 - allocated);
    } else {
      vars[i].weight = Math.max(1, Math.round((vars[i].weight / sum) * 100));
      allocated += vars[i].weight;
    }
  }
}

document.querySelectorAll('#shape-tools [data-tool]').forEach((btn) => {
  btn.addEventListener('click', () => {
    state.tool = btn.dataset.tool;
    document.querySelectorAll('#shape-tools [data-tool]').forEach((b) => {
      b.setAttribute('aria-pressed', b === btn ? 'true' : 'false');
    });
  });
});

els.zoom.addEventListener('change', () => {
  state.scale = Number(els.zoom.value) || 32;
  resizeCanvas();
  redraw();
});

function resizeCanvas() {
  const px = SIZE * state.scale;
  els.canvas.width = px;
  els.canvas.height = px;
}

function redraw() {
  if (!state.art) return;
  const scale = state.scale;
  ctx.clearRect(0, 0, els.canvas.width, els.canvas.height);
  // Checker underlay for transparency
  const cell = Math.max(4, scale / 2);
  for (let y = 0; y < els.canvas.height; y += cell) {
    for (let x = 0; x < els.canvas.width; x += cell) {
      ctx.fillStyle = ((x / cell + y / cell) & 1) ? '#1a1d26' : '#12141a';
      ctx.fillRect(x, y, cell, cell);
    }
  }

  const previewArt = clone(state.art);
  // While editing a spatial variant, show base + that variant (not hash pick).
  if (typeof state.editTarget === 'number' && previewArt.spatial) {
    const only = previewArt.spatial.variants[state.editTarget];
    drawArt(ctx, { ...previewArt, spatial: null }, 0, 0, scale);
    drawArt(ctx, { layers: only.layers, size: SIZE }, 0, 0, scale);
  } else if (state.editTarget === 'base') {
    drawArt(ctx, { ...previewArt, spatial: null }, 0, 0, scale);
  } else {
    drawArt(ctx, previewArt, 0, 0, scale);
  }

  const layers = currentLayers();
  if (state.selected >= 0 && layers[state.selected]) {
    drawSelection(ctx, layers[state.selected], 0, 0, scale);
  }

  // Pixel grid
  ctx.strokeStyle = 'rgba(255,255,255,0.08)';
  ctx.lineWidth = 1;
  for (let i = 0; i <= SIZE; i++) {
    const p = i * scale + 0.5;
    ctx.beginPath(); ctx.moveTo(p, 0); ctx.lineTo(p, SIZE * scale); ctx.stroke();
    ctx.beginPath(); ctx.moveTo(0, p); ctx.lineTo(SIZE * scale, p); ctx.stroke();
  }

  renderShapeList();
  drawPreviewStrip();
}

function renderShapeList() {
  const layers = currentLayers();
  els.shapeList.innerHTML = '';
  layers.forEach((s, i) => {
    const li = document.createElement('li');
    li.textContent = `${i}: ${s.type} ${s.id}`;
    if (i === state.selected) li.classList.add('selected');
    li.addEventListener('click', () => { state.selected = i; redraw(); syncProps(); });
    els.shapeList.appendChild(li);
  });
}

function syncProps() {
  const s = currentLayers()[state.selected];
  if (!s) return;
  if (s.fill) els.fill.value = toColorInput(s.fill);
  if (s.stroke) els.stroke.value = toColorInput(s.stroke);
  if (s.strokeWidth) els.sw.value = s.strokeWidth;
}

function toColorInput(hex) {
  if (!hex) return '#000000';
  if (hex.length === 9) return hex.slice(0, 7);
  if (hex.length === 4) {
    return '#' + hex[1] + hex[1] + hex[2] + hex[2] + hex[3] + hex[3];
  }
  return hex;
}

function drawPreviewStrip() {
  const cols = 10, rows = 2;
  const cell = 16;
  els.preview.width = cols * cell;
  els.preview.height = rows * cell;
  pctx.clearRect(0, 0, els.preview.width, els.preview.height);
  if (!state.art) return;
  for (let ty = 0; ty < rows; ty++) {
    for (let tx = 0; tx < cols; tx++) {
      drawArt(pctx, state.art, tx * cell, ty * cell, 1, { tx, ty });
    }
  }
}

function canvasTilePos(ev) {
  const rect = els.canvas.getBoundingClientRect();
  const x = (ev.clientX - rect.left) * (els.canvas.width / rect.width);
  const y = (ev.clientY - rect.top) * (els.canvas.height / rect.height);
  return {
    x: Math.max(0, Math.min(SIZE, x / state.scale)),
    y: Math.max(0, Math.min(SIZE, y / state.scale)),
  };
}

function snap(v) {
  return Math.round(v * 2) / 2; // half-pixel snap
}

/** Handle hit pad in tile units — grows when zoomed out so endpoints stay grabable. */
function handlePad() {
  return Math.max(1.0, 10 / state.scale);
}

function applyResize(s, handle, p, orig) {
  if (s.type === 'line') {
    if (handle === 'p1') { s.x1 = snap(p.x); s.y1 = snap(p.y); }
    else if (handle === 'p2') { s.x2 = snap(p.x); s.y2 = snap(p.y); }
    return;
  }
  if (s.type === 'circle' && handle === 'radius') {
    s.r = snap(Math.max(0.5, Math.hypot(p.x - num(s.cx), p.y - num(s.cy))));
    return;
  }
  if (s.type !== 'rect') return;
  const o = orig;
  let x1 = num(o.x), y1 = num(o.y), x2 = num(o.x) + num(o.w), y2 = num(o.y) + num(o.h);
  if (handle.includes('n')) y1 = p.y;
  if (handle.includes('s')) y2 = p.y;
  if (handle.includes('w')) x1 = p.x;
  if (handle.includes('e')) x2 = p.x;
  s.x = snap(Math.min(x1, x2));
  s.y = snap(Math.min(y1, y2));
  s.w = snap(Math.abs(x2 - x1));
  s.h = snap(Math.abs(y2 - y1));
}

function num(v, fallback = 0) {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

els.canvas.addEventListener('pointerdown', (ev) => {
  if (!state.art) return;
  els.canvas.setPointerCapture(ev.pointerId);
  const p = canvasTilePos(ev);
  const layers = currentLayers();

  if (state.tool === 'select') {
    // 1) Prefer handles on the already-selected shape (endpoint / corner edit).
    if (state.selected >= 0 && layers[state.selected]) {
      const h = hitHandle(layers[state.selected], p.x, p.y, handlePad());
      if (h) {
        state.drag = {
          mode: 'handle',
          handle: h.kind,
          start: p,
          orig: clone(layers[state.selected]),
        };
        pushUndo();
        redraw();
        syncProps();
        return;
      }
    }

    const hits = hitTestAll(layers, p.x, p.y);
    let idx = -1;
    if (ev.altKey && hits.length) {
      // Alt+click cycles through stacked shapes under the cursor.
      const cur = hits.indexOf(state.selected);
      idx = hits[(cur + 1) % hits.length];
    } else if (hits.length) {
      // Keep current selection if still under the pointer (so endpoints stay editable).
      if (state.selected >= 0 && hits.includes(state.selected)) idx = state.selected;
      else idx = hits[0];
    } else {
      idx = -1;
    }

    state.selected = idx;
    if (idx >= 0) {
      // Second chance: handle on newly selected shape.
      const h = hitHandle(layers[idx], p.x, p.y, handlePad());
      if (h) {
        state.drag = {
          mode: 'handle',
          handle: h.kind,
          start: p,
          orig: clone(layers[idx]),
        };
      } else {
        state.drag = {
          mode: 'move',
          start: p,
          orig: clone(layers[idx]),
        };
      }
      pushUndo();
    }
    redraw();
    syncProps();
    setStatus(idx >= 0
      ? `Selected ${layers[idx].type} — drag handles to edit; Alt+click to cycle stack`
      : 'Select a shape (Alt+click cycles overlapping shapes)');
    return;
  }

  pushUndo();
  const id = newShapeId(state.tool);
  let shape;
  if (state.tool === 'rect') {
    shape = { id, type: 'rect', x: snap(p.x), y: snap(p.y), w: 0, h: 0, fill: els.fill.value };
  } else if (state.tool === 'line') {
    shape = {
      id, type: 'line',
      x1: snap(p.x), y1: snap(p.y), x2: snap(p.x), y2: snap(p.y),
      stroke: els.stroke.value, strokeWidth: Number(els.sw.value) || 1,
    };
  } else if (state.tool === 'circle') {
    shape = {
      id, type: 'circle',
      cx: snap(p.x), cy: snap(p.y), r: 0,
      fill: els.fill.value,
    };
  }
  layers.push(shape);
  state.selected = layers.length - 1;
  state.drag = { mode: 'create', start: p, tool: state.tool };
  markDirty();
  redraw();
});

els.canvas.addEventListener('pointermove', (ev) => {
  if (!state.drag || !state.art) return;
  const p = canvasTilePos(ev);
  const layers = currentLayers();
  const s = layers[state.selected];
  if (!s) return;

  if (state.drag.mode === 'handle') {
    applyResize(s, state.drag.handle, p, state.drag.orig);
    markDirty();
    redraw();
    return;
  }

  if (state.drag.mode === 'move') {
    const dx = p.x - state.drag.start.x;
    const dy = p.y - state.drag.start.y;
    const o = state.drag.orig;
    if (s.type === 'rect') {
      s.x = snap(num(o.x) + dx); s.y = snap(num(o.y) + dy);
    } else if (s.type === 'circle') {
      s.cx = snap(num(o.cx) + dx); s.cy = snap(num(o.cy) + dy);
    } else if (s.type === 'line') {
      s.x1 = snap(num(o.x1) + dx); s.y1 = snap(num(o.y1) + dy);
      s.x2 = snap(num(o.x2) + dx); s.y2 = snap(num(o.y2) + dy);
    }
    markDirty();
    redraw();
    return;
  }

  if (state.drag.mode === 'create') {
    const a = state.drag.start;
    if (s.type === 'rect') {
      s.x = snap(Math.min(a.x, p.x));
      s.y = snap(Math.min(a.y, p.y));
      s.w = snap(Math.abs(p.x - a.x));
      s.h = snap(Math.abs(p.y - a.y));
    } else if (s.type === 'line') {
      s.x2 = snap(p.x); s.y2 = snap(p.y);
    } else if (s.type === 'circle') {
      s.r = snap(Math.hypot(p.x - a.x, p.y - a.y));
    }
    redraw();
  }
});

els.canvas.addEventListener('pointerup', () => {
  state.drag = null;
});

// Hover cursor: show resize affordance on handles.
els.canvas.addEventListener('pointermove', (ev) => {
  if (state.drag || state.tool !== 'select' || !state.art) return;
  const p = canvasTilePos(ev);
  const layers = currentLayers();
  const s = layers[state.selected];
  if (s && hitHandle(s, p.x, p.y, handlePad())) {
    els.canvas.style.cursor = 'pointer';
  } else if (hitTest(layers, p.x, p.y) >= 0) {
    els.canvas.style.cursor = 'move';
  } else {
    els.canvas.style.cursor = 'default';
  }
}, true);
document.getElementById('btn-delete').addEventListener('click', () => {
  const layers = currentLayers();
  if (state.selected < 0 || !layers[state.selected]) return;
  pushUndo();
  layers.splice(state.selected, 1);
  state.selected = -1;
  markDirty();
  redraw();
});

document.getElementById('btn-shape-up').addEventListener('click', () => {
  const layers = currentLayers();
  const i = state.selected;
  if (i <= 0) return;
  pushUndo();
  [layers[i - 1], layers[i]] = [layers[i], layers[i - 1]];
  state.selected = i - 1;
  markDirty();
  redraw();
});

document.getElementById('btn-shape-down').addEventListener('click', () => {
  const layers = currentLayers();
  const i = state.selected;
  if (i < 0 || i >= layers.length - 1) return;
  pushUndo();
  [layers[i + 1], layers[i]] = [layers[i], layers[i + 1]];
  state.selected = i + 1;
  markDirty();
  redraw();
});

function applyColorProps() {
  const s = currentLayers()[state.selected];
  if (!s) return;
  pushUndo();
  if (s.type === 'line' || s.stroke) s.stroke = els.stroke.value;
  if (s.type !== 'line' && (s.fill || s.type === 'rect' || s.type === 'circle')) {
    s.fill = els.fill.value;
  }
  if (s.strokeWidth !== undefined || s.type === 'line') {
    s.strokeWidth = Number(els.sw.value) || 1;
  }
  markDirty();
  redraw();
}

els.fill.addEventListener('change', applyColorProps);
els.stroke.addEventListener('change', applyColorProps);
els.sw.addEventListener('change', applyColorProps);

document.getElementById('btn-undo').addEventListener('click', () => {
  if (!state.undo.length) return;
  state.redo.push(clone(state.art));
  state.art = state.undo.pop();
  markDirty();
  updateSpatialUI();
  redraw();
});

document.getElementById('btn-redo').addEventListener('click', () => {
  if (!state.redo.length) return;
  state.undo.push(clone(state.art));
  state.art = state.redo.pop();
  markDirty();
  updateSpatialUI();
  redraw();
});

document.getElementById('btn-save').addEventListener('click', async () => {
  if (!state.art) return;
  if (state.art.spatial) normalizeWeights();
  try {
    const data = await api.saveTile(state.art.gid, state.art);
    state.art = data.art;
    markDirty(false);
    const t = state.tiles.find((x) => x.gid === state.art.gid);
    if (t) { t.hasArt = true; t.hasSpatial = !!state.art.spatial; }
    renderList(els.search.value);
    setStatus(`Saved ${data.path || ''}`);
  } catch (e) {
    setStatus(e instanceof ApiError ? e.message : String(e));
  }
});

window.addEventListener('keydown', (ev) => {
  if ((ev.metaKey || ev.ctrlKey) && ev.key === 's') {
    ev.preventDefault();
    document.getElementById('btn-save').click();
  }
  if ((ev.metaKey || ev.ctrlKey) && ev.key === 'z') {
    ev.preventDefault();
    document.getElementById(ev.shiftKey ? 'btn-redo' : 'btn-undo').click();
  }
  if (ev.key === 'Delete' || ev.key === 'Backspace') {
    if (document.activeElement?.tagName === 'INPUT') return;
    document.getElementById('btn-delete').click();
  }
});

resizeCanvas();
boot();
