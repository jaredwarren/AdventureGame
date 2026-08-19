// The toolbar: tools, mode, undo/redo, save, and zoom.

import { state, emit, subscribe } from '../store.js';
import { tools, toolState, setTool, setMode } from '../tools.js';
import * as history from '../history.js';
import * as view from '../view.js';
import * as render from '../render.js';
import { refreshCursor } from '../viewport.js';

const TOOL_ORDER = ['brush', 'eraser', 'rect', 'flood', 'eyedropper', 'select', 'marker-add'];

let toolHost, markerTypeEl;

export function initToolbar({ onSave, onReload, onPlay }) {
  toolHost = document.getElementById('tool-buttons');
  markerTypeEl = document.getElementById('marker-type');

  for (const id of TOOL_ORDER) {
    const tool = tools.get(id);
    if (!tool) continue;
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'btn tool-btn';
    b.dataset.tool = id;
    b.dataset.mode = tool.mode;
    b.textContent = tool.label;
    b.title = tool.label;
    b.addEventListener('click', () => setTool(id));
    toolHost.append(b);
  }

  for (const m of state.schema.markers) {
    const opt = document.createElement('option');
    opt.value = m.type;
    opt.textContent = m.label;
    markerTypeEl.append(opt);
  }
  markerTypeEl.value = state.ui.markerType;
  markerTypeEl.addEventListener('change', () => {
    state.ui.markerType = markerTypeEl.value;
    emit('ui');
  });

  document.getElementById('btn-save').addEventListener('click', () => onSave());
  document.getElementById('btn-reload').addEventListener('click', () => onReload());
  document.getElementById('btn-play').addEventListener('click', () => onPlay());
  document.getElementById('btn-undo').addEventListener('click', () => { history.undo(); render.invalidateAll(); });
  document.getElementById('btn-redo').addEventListener('click', () => { history.redo(); render.invalidateAll(); });

  for (const [id, mode] of [['mode-tile', 'tile'], ['mode-marker', 'marker']]) {
    document.getElementById(id).addEventListener('click', () => setMode(mode));
  }

  document.getElementById('btn-zoom-in').addEventListener('click', () => zoom(1));
  document.getElementById('btn-zoom-out').addEventListener('click', () => zoom(-1));
  document.getElementById('btn-zoom-fit').addEventListener('click', () => {
    view.zoomToFit(state.doc.tmj.width, state.doc.tmj.height);
    emit('ui');
    render.invalidateAll();
  });

  document.getElementById('toggle-grid').addEventListener('change', (e) => {
    state.ui.showGrid = e.target.checked;
    render.invalidateOverlay();
  });
  document.getElementById('toggle-markers').addEventListener('change', (e) => {
    state.ui.showMarkersInTileMode = e.target.checked;
    render.invalidateOverlay();
  });

  subscribe('tools', renderToolbar);
  subscribe('ui', renderToolbar);
  subscribe('history', renderToolbar);
  subscribe('doc', renderToolbar);
}

function zoom(dir) {
  view.zoomAt(view.view.cssW / 2, view.view.cssH / 2, dir, state.doc.tmj.width, state.doc.tmj.height);
  emit('ui');
  render.invalidateAll();
}

export function renderToolbar() {
  if (!toolHost) return;

  for (const b of toolHost.querySelectorAll('.tool-btn')) {
    b.setAttribute('aria-pressed', String(b.dataset.tool === toolState.active?.id));
    // Tools are filtered by mode so the bar only offers what applies.
    b.hidden = b.dataset.mode !== state.ui.mode;
  }
  document.getElementById('mode-tile').setAttribute('aria-pressed', String(state.ui.mode === 'tile'));
  document.getElementById('mode-marker').setAttribute('aria-pressed', String(state.ui.mode === 'marker'));
  document.getElementById('marker-controls').hidden = state.ui.mode !== 'marker';
  document.getElementById('tile-controls').hidden = state.ui.mode !== 'tile';
  if (markerTypeEl.value !== state.ui.markerType) markerTypeEl.value = state.ui.markerType;

  const undoBtn = document.getElementById('btn-undo');
  const redoBtn = document.getElementById('btn-redo');
  undoBtn.disabled = !history.canUndo();
  redoBtn.disabled = !history.canRedo();
  undoBtn.title = history.canUndo() ? `Undo ${history.undoLabel()}` : 'Nothing to undo';
  redoBtn.title = history.canRedo() ? `Redo ${history.redoLabel()}` : 'Nothing to redo';

  document.getElementById('zoom-label').textContent = `${Math.round(view.view.zoom * 100)}%`;
  document.getElementById('map-title').textContent = state.doc?.id ?? '—';
  document.getElementById('dirty-dot').hidden = !history.isDirty();
  document.getElementById('btn-save').disabled = !history.isDirty();

  refreshCursor();
}
