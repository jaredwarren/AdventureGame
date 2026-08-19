// The layers panel: active layer, per-layer visibility and opacity, and the
// three-state view mode.

import { state, emit, subscribe, mapW, mapH, nextLayerID, reindex } from '../store.js';
import * as render from '../render.js';
import * as history from '../history.js';
import { cmd } from '../commands.js';

let host, viewModeEl;

export function initLayers() {
  host = document.getElementById('layer-list');
  viewModeEl = document.getElementById('layer-view-mode');

  viewModeEl.addEventListener('change', () => {
    state.ui.layerView = viewModeEl.value;
    render.invalidateAll();
  });

  subscribe('doc', renderLayers);
  subscribe('ui', renderLayers);
}

export function renderLayers() {
  if (!host || !state.doc) return;
  host.replaceChildren();
  viewModeEl.value = state.ui.layerView;

  state.doc.tileLayers.forEach(({ layer }, i) => {
    const row = document.createElement('div');
    row.className = 'layer-row' + (i === state.ui.activeLayer ? ' active' : '');

    const pick = document.createElement('input');
    pick.type = 'radio';
    pick.name = 'active-layer';
    pick.checked = i === state.ui.activeLayer;
    pick.title = 'Edit this layer';
    pick.addEventListener('change', () => {
      state.ui.activeLayer = i;
      emit('ui', 'palette');
      render.invalidateAll();
    });

    const name = document.createElement('span');
    name.className = 'layer-name';
    name.textContent = layer.name || `layer ${layer.id}`;

    const eye = document.createElement('button');
    eye.type = 'button';
    eye.className = 'icon-btn';
    eye.textContent = layer.visible === false ? '🚫' : '👁';
    eye.title = 'Toggle visibility';
    eye.addEventListener('click', () => {
      layer.visible = layer.visible === false;
      emit('doc');
      render.invalidateAll();
    });

    const opacity = document.createElement('input');
    opacity.type = 'range';
    opacity.min = '0';
    opacity.max = '1';
    opacity.step = '0.05';
    opacity.value = String(layer.opacity ?? 1);
    opacity.title = 'Layer opacity';
    opacity.addEventListener('input', () => {
      layer.opacity = Number(opacity.value);
      render.invalidateAll();
    });

    row.append(pick, name, eye, opacity);
    host.append(row);
  });

  renderBaseLayerBanner();
}

/**
 * Offers to add the missing "base" layer instead of injecting it silently.
 *
 * The in-game editor mutates the file on open, which makes every map you merely
 * look at come up dirty. Here it is an explicit, undoable action.
 */
function renderBaseLayerBanner() {
  const banner = document.getElementById('layer-banner');
  if (!banner) return;
  banner.replaceChildren();

  if (state.doc.tileLayers.length >= 2) {
    banner.hidden = true;
    return;
  }
  banner.hidden = false;

  const text = document.createElement('span');
  text.textContent = 'This map has a single tile layer.';

  const add = document.createElement('button');
  add.type = 'button';
  add.className = 'btn btn-small';
  add.textContent = 'Add grass base layer';
  add.addEventListener('click', addBaseLayer);

  banner.append(text, add);
}

function addBaseLayer() {
  const doc = state.doc;
  const W = mapW(), H = mapH();
  const data = new Array(W * H).fill(state.schema.layers.defaultBaseGid);

  const layer = {
    id: nextLayerID(doc.tmj),
    type: 'tilelayer',
    name: state.schema.layers.tileLayerNames[0],
    visible: true,
    opacity: 1,
    width: W,
    height: H,
    x: 0,
    y: 0,
    data,
  };
  // Prepend so it sits beneath the existing layer, matching the game's ordering.
  const index = doc.tmj.layers.findIndex((l) => l.type === 'tilelayer');
  history.run(cmd.layerAdd(layer, index < 0 ? 0 : index));

  state.ui.activeLayer = doc.tileLayers.length - 1;
  state.validation.stale = true;
  emit('doc', 'ui', 'validation');
  render.invalidateAll();
}

export function cycleLayer(delta) {
  const n = state.doc?.tileLayers.length ?? 0;
  if (n === 0) return;
  state.ui.activeLayer = (state.ui.activeLayer + delta + n) % n;
  emit('ui', 'palette');
  render.invalidateAll();
}

export function cycleViewMode() {
  const modes = ['all', 'dim', 'isolate'];
  const i = modes.indexOf(state.ui.layerView);
  state.ui.layerView = modes[(i + 1) % modes.length];
  emit('ui');
  render.invalidateAll();
}

export { reindex };
