// The tile palette.
//
// Swatches are plain elements backed by the atlas as a CSS background, so all 53
// tiles cost zero extra canvases: the background-position is just -gid*size.
//
// The in-game editor only applies its floor/non-floor filter when layer
// isolation happens to be on, which is almost certainly unintended. Here the
// filter is its own checkbox.

import { TILE } from '../tileart.js';
import { state, emit, subscribe } from '../store.js';
import * as render from '../render.js';

const SWATCH = 30;
let host, searchEl, filterEl;

export function initPalette() {
  host = document.getElementById('palette-groups');
  searchEl = document.getElementById('palette-search');
  filterEl = document.getElementById('palette-filter-layer');

  searchEl.addEventListener('input', () => {
    state.ui.paletteQuery = searchEl.value.trim().toLowerCase();
    renderPalette();
  });
  filterEl.addEventListener('change', () => {
    state.ui.filterPaletteByLayer = filterEl.checked;
    renderPalette();
  });

  subscribe('palette', renderPalette);
  subscribe('ui', updateSelection);
  subscribe('doc', renderPalette);
}

/**
 * Layer-role filter: the bottom layer is the ground plane and should only offer
 * floor tiles, while upper layers hold everything that sits on top of it.
 */
function allowedOnActiveLayer(def) {
  if (!state.ui.filterPaletteByLayer) return true;
  const isBase = state.ui.activeLayer === 0;
  return isBase ? def.floor : !def.floor || def.gid === 0;
}

function matchesQuery(def) {
  const q = state.ui.paletteQuery;
  if (!q) return true;
  return def.name.includes(q) || String(def.gid) === q || (def.tags ?? []).some((t) => t.includes(q));
}

export function renderPalette() {
  if (!host || !state.schema) return;
  const { schema, atlas } = state;
  host.replaceChildren();
  host.style.setProperty('--sheet', `url(${atlas.cssURL()})`);
  host.style.setProperty('--sheet-size', atlas.swatchSheetSize(SWATCH));

  for (const group of schema.palette) {
    const defs = group.gids
      .map((gid) => atlas.def(gid))
      .filter((d) => d && allowedOnActiveLayer(d) && matchesQuery(d));
    if (defs.length === 0) continue;

    const details = document.createElement('details');
    details.className = 'palette-group';
    // Collapse the big autotile blocks unless a search is narrowing them.
    details.open = !group.collapsed || !!state.ui.paletteQuery;

    const summary = document.createElement('summary');
    summary.innerHTML = `<span>${group.label}</span><span class="count">${defs.length}</span>`;
    details.append(summary);

    const grid = document.createElement('div');
    grid.className = 'swatch-grid';
    for (const def of defs) grid.append(makeSwatch(def));
    details.append(grid);
    host.append(details);
  }

  if (!host.children.length) {
    const empty = document.createElement('p');
    empty.className = 'empty';
    empty.textContent = state.ui.paletteQuery
      ? 'No tiles match.'
      : 'No tiles available for this layer. Turn off the layer filter to see all tiles.';
    host.append(empty);
  }
  updateSelection();
}

function makeSwatch(def) {
  const b = document.createElement('button');
  b.type = 'button';
  b.className = 'swatch';
  b.dataset.gid = String(def.gid);
  b.title = `${def.name} (gid ${def.gid})${def.tags?.length ? '\n' + def.tags.join(', ') : ''}`;
  b.setAttribute('aria-label', def.name);

  if (def.gid === 0) {
    b.classList.add('swatch-empty');
    b.textContent = '∅';
  } else {
    b.style.backgroundPosition = state.atlas.swatchOffset(def.gid, SWATCH);
  }
  if (state.atlas.isAnimated(def.gid)) b.classList.add('swatch-anim');

  b.addEventListener('click', () => {
    state.ui.brushGID = def.gid;
    emit('ui');
    render.invalidateOverlay();
  });
  return b;
}

function updateSelection() {
  if (!host) return;
  for (const el of host.querySelectorAll('.swatch')) {
    el.classList.toggle('selected', Number(el.dataset.gid) === state.ui.brushGID);
  }
  const label = document.getElementById('brush-label');
  if (label && state.atlas) {
    const def = state.atlas.def(state.ui.brushGID);
    label.textContent = def ? `${def.name} (${def.gid})` : `gid ${state.ui.brushGID}`;
  }
}

/** Cycles through the currently visible palette entries (the , and . keys). */
export function stepBrush(delta) {
  const visible = [...host.querySelectorAll('.swatch')].map((el) => Number(el.dataset.gid));
  if (!visible.length) return;
  const i = visible.indexOf(state.ui.brushGID);
  const next = visible[(i < 0 ? 0 : i + delta + visible.length) % visible.length];
  state.ui.brushGID = next;
  emit('ui');
  render.invalidateOverlay();
}

export function selectFavorite(slot) {
  const gid = state.schema?.favorites?.[slot];
  if (gid === undefined) return;
  state.ui.brushGID = gid;
  emit('ui');
  render.invalidateOverlay();
}
