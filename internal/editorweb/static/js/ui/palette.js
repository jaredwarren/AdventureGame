// The tile palette.
//
// Swatches are plain elements backed by the atlas as a CSS background, so all 53
// tiles cost zero extra canvases: the background-position is just -gid*size.
//
// Family rows show only the root/base tile until expanded, keeping Terrain and
// Hazards short while still exposing every autotile variant on click.
//
// The in-game editor only applies its floor/non-floor filter when layer
// isolation happens to be on, which is almost certainly unintended. Here the
// filter is its own checkbox.

import { TILE } from '../tileart.js';
import { state, emit, subscribe } from '../store.js';
import * as render from '../render.js';

const SWATCH = 30;
let host, searchEl, filterEl;

// User-toggled open/closed state, keyed by palette group/family id. Absent means
// use the schema default (!group.collapsed). Families default closed. Search does
// not write here, so clearing the query restores whatever the user last chose.
const openById = new Map();

export function initPalette() {
  host = document.getElementById('palette-groups');
  searchEl = document.getElementById('palette-search');

  searchEl.addEventListener('input', () => {
    state.ui.paletteQuery = searchEl.value.trim().toLowerCase();
    renderPalette();
  });

  subscribe('palette', renderPalette);
  subscribe('ui', updateSelection);
}

function matchesQuery(def) {
  const q = state.ui.paletteQuery;
  if (!q) return true;
  return def.name.includes(q) || String(def.gid) === q || (def.tags ?? []).some((t) => t.includes(q));
}

function groupIsOpen(group) {
  if (state.ui.paletteQuery) return true;
  if (openById.has(group.id)) return openById.get(group.id);
  return !group.collapsed;
}

function familyIsOpen(family, matchedDefs) {
  if (state.ui.paletteQuery) return matchedDefs.length > 0;
  if (openById.has(family.id)) return openById.get(family.id);
  return false;
}

function bindToggle(details, id) {
  details.addEventListener('toggle', () => {
    if (state.ui.paletteQuery) return;
    openById.set(id, details.open);
  });
}

export function renderPalette() {
  if (!host || !state.schema) return;
  const { schema, atlas } = state;
  host.replaceChildren();
  host.style.setProperty('--sheet', `url(${atlas.cssURL()})`);
  host.style.setProperty('--sheet-size', atlas.swatchSheetSize(SWATCH));

  for (const group of schema.palette) {
    const standalone = (group.gids ?? [])
      .map((gid) => atlas.def(gid))
      .filter((d) => d && matchesQuery(d));

    const families = [];
    for (const family of group.families ?? []) {
      const defs = family.gids
        .map((gid) => atlas.def(gid))
        .filter((d) => d && matchesQuery(d));
      if (defs.length > 0) families.push({ family, defs });
    }

    if (standalone.length === 0 && families.length === 0) continue;

    const details = document.createElement('details');
    details.className = 'palette-group';
    details.dataset.groupId = group.id;
    details.open = groupIsOpen(group);

    const tileCount = standalone.length + families.reduce((n, f) => n + f.defs.length, 0);
    const summary = document.createElement('summary');
    summary.innerHTML = `<span>${group.label}</span><span class="count">${tileCount}</span>`;
    details.append(summary);

    if (standalone.length) {
      const grid = document.createElement('div');
      grid.className = 'swatch-grid';
      for (const def of standalone) grid.append(makeSwatch(def));
      details.append(grid);
    }

    for (const { family, defs } of families) {
      details.append(makeFamilyRow(family, defs));
    }

    host.append(details);
    bindToggle(details, group.id);
  }

  if (!host.children.length) {
    const empty = document.createElement('p');
    empty.className = 'empty';
    empty.textContent = state.ui.paletteQuery
      ? 'No tiles match.'
      : 'No tiles available.';
    host.append(empty);
  }
  updateSelection();
}

function makeFamilyRow(family, defs) {
  const details = document.createElement('details');
  details.className = 'palette-family';
  details.dataset.familyId = family.id;
  details.open = familyIsOpen(family, defs);

  const summary = document.createElement('summary');
  summary.className = 'palette-family-summary';

  const baseDef = state.atlas.def(family.baseGid)
    ?? defs.find((d) => d.gid === family.baseGid)
    ?? defs[0];
  if (baseDef) {
    const root = makeSwatch(baseDef);
    root.classList.add('palette-family-root');
    root.addEventListener('click', (e) => {
      // Selecting the root brush should not toggle the family open/closed.
      e.preventDefault();
      e.stopPropagation();
    });
    summary.append(root);
  }

  const label = document.createElement('span');
  label.className = 'palette-family-label';
  label.textContent = family.label;
  summary.append(label);

  const count = document.createElement('span');
  count.className = 'count';
  count.textContent = String(defs.length);
  summary.append(count);

  details.append(summary);

  const grid = document.createElement('div');
  grid.className = 'swatch-grid';
  for (const def of defs) grid.append(makeSwatch(def));
  details.append(grid);

  bindToggle(details, family.id);
  return details;
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
  // Deduplicate: family root swatches also appear in the expanded grid.
  const unique = [...new Set(visible)];
  if (!unique.length) return;
  const i = unique.indexOf(state.ui.brushGID);
  const next = unique[(i < 0 ? 0 : i + delta + unique.length) % unique.length];
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
