// The property panel.
//
// Every widget here is generated from the server's schema, which is itself
// derived by probing internal/world. Adding a property to a marker handler in Go
// therefore makes it editable here with no JavaScript change at all — and the
// unknown-type fallback means even a field type nobody anticipated round-trips
// untouched rather than being eaten.
//
// One subtlety worth preserving: a door's spawn_x/spawn_y are Tiled STRINGS that
// the game parses with ParseDoorSpawnCoord (a number or "*"). The schema says so,
// and the widget is a text field so "*" can be typed. Advertising them as numbers
// would quietly change the on-disk type.

import { state, emit, subscribe, objProp, mapProp, markerSchema } from '../store.js';
import * as history from '../history.js';
import { cmd } from '../commands.js';
import * as render from '../render.js';
import { markerSummary } from '../markers.js';
import { toast } from './dialogs.js';

let host, titleEl, tabsEl;
let tab = 'marker';
let lastSelection = undefined;

export function initProps() {
  host = document.getElementById('props-body');
  titleEl = document.getElementById('props-title');
  tabsEl = document.getElementById('props-tabs');

  tabsEl.addEventListener('click', (e) => {
    const b = e.target.closest('button[data-tab]');
    if (!b) return;
    tab = b.dataset.tab;
    renderProps(true);
  });

  subscribe('selection', () => {
    const sel = state.ui.selection;
    if (sel !== lastSelection) {
      lastSelection = sel;
      if (sel !== null) tab = 'marker';
      renderProps(true);
    }
  });
  subscribe('doc', () => renderProps(false));
  subscribe('maps', () => renderProps(false));
}

export function renderProps(force = false) {
  if (!host || !state.doc) return;

  if (!force && host.contains(document.activeElement)) {
    return;
  }

  for (const b of tabsEl.querySelectorAll('button[data-tab]')) {
    b.setAttribute('aria-pressed', String(b.dataset.tab === tab));
  }
  host.replaceChildren();

  if (tab === 'map') {
    titleEl.textContent = state.doc.id;
    renderMapProps();
  } else {
    renderMarkerProps();
  }
}

// ---- marker properties ----

function renderMarkerProps() {
  const obj = state.doc.objById.get(state.ui.selection);
  if (!obj) {
    titleEl.textContent = 'No selection';
    const p = document.createElement('p');
    p.className = 'empty';
    p.textContent = 'Select a marker to edit its properties.';
    host.append(p);
    return;
  }
  titleEl.textContent = markerSummary(obj);
  const schema = markerSchema(obj.type);

  host.append(fieldRow('id', readOnly(String(obj.id))));
  host.append(fieldRow('name', textInput(obj.name ?? '', (v) => {
    setObjectField(obj, 'name', v || undefined, `name:${obj.id}`);
  })));

  host.append(fieldRow('x', numberInput(obj.x, 1, (v) => setObjectField(obj, 'x', v, `x:${obj.id}`))));
  host.append(fieldRow('y', numberInput(obj.y, 1, (v) => setObjectField(obj, 'y', v, `y:${obj.id}`))));

  if (schema?.sizable) {
    host.append(fieldRow('width', numberInput(obj.width ?? 16, 1, (v) => setObjectField(obj, 'width', v, `w:${obj.id}`))));
    host.append(fieldRow('height', numberInput(obj.height ?? 16, 1, (v) => setObjectField(obj, 'height', v, `h:${obj.id}`))));
  }

  if (schema?.fields?.length) {
    host.append(divider('Properties'));
    renderFields(host, schema.fields,
      (name) => objProp(obj, name),
      (field, value) => setMarkerProp(obj, field, value));
  }

  host.append(divider(''));
  const actions = document.createElement('div');
  actions.className = 'prop-actions';

  const snap = button('Snap to grid', () => {
    setObjectField(obj, 'x', Math.round(obj.x / 16) * 16, null);
    setObjectField(obj, 'y', Math.round(obj.y / 16) * 16, null);
  });
  const del = button('Delete', async () => {
    const { deleteSelected } = await import('../tools.js');
    deleteSelected();
  }, 'danger');
  const dup = button('Duplicate', async () => {
    const { duplicateSelected } = await import('../tools.js');
    duplicateSelected();
  });
  actions.append(snap, dup, del);
  host.append(actions);
}

function setObjectField(obj, key, value, coalesceKey) {
  const before = structuredClone(obj);
  obj[key] = value;
  const after = structuredClone(obj);
  history.push(cmd.markerReplace(obj.id, before, after), coalesceKey);
  state.validation.stale = true;
  emit('doc', 'validation');
  render.invalidateOverlay();
}

function setMarkerProp(obj, field, value) {
  const before = objProp(obj, field.name);
  if (before === value) return;
  history.run(
    cmd.objProp(obj.id, field.name, field.tiledType, before, value),
    history.editSessionKey(`obj${obj.id}`, field.name),
  );
  state.validation.stale = true;
  emit('doc', 'validation');
  render.invalidateOverlay();
}

// ---- map properties ----

function renderMapProps() {
  const tmj = state.doc.tmj;

  host.append(fieldRow('size', readOnly(`${tmj.width} x ${tmj.height} tiles`)));
  host.append(fieldRow('tile size', readOnly(`${tmj.tilewidth} x ${tmj.tileheight}`)));
  host.append(fieldRow('file', readOnly(state.doc.file ?? state.doc.id)));

  const counts = {};
  for (const o of state.doc.markersLayer?.objects ?? []) counts[o.type] = (counts[o.type] ?? 0) + 1;
  const summary = Object.entries(counts).map(([k, v]) => `${k} ${v}`).join(', ') || 'none';
  host.append(fieldRow('markers', readOnly(summary)));

  if (state.doc.reformats) {
    host.append(note(
      'Saving this map will reformat it (whitespace and omitted defaults). '
      + 'No data changes, but expect a large diff.'));
  }
  if (state.doc.unmodeledFields?.length) {
    host.append(note(
      `This file has fields this editor cannot represent and a save would drop: `
      + state.doc.unmodeledFields.join(', '), 'warn'));
  }

  host.append(divider('Map properties'));
  renderFields(host, state.schema.mapProperties,
    (name) => mapProp(name),
    (field, value) => {
      const before = mapProp(field.name);
      if (before === value) return;
      history.run(
        cmd.mapProp(field.name, field.tiledType, before, value),
        history.editSessionKey('map', field.name),
      );
      state.validation.stale = true;
      emit('doc', 'validation');
      render.invalidateAll();
    });
}

// ---- the generic schema-driven form ----

/**
 * Renders a list of schema fields into host.
 *
 * Re-renders itself on 'change' so conditional fields (showIf) appear and
 * disappear as their controlling checkbox is toggled.
 */
export function renderFields(container, fields, getVal, setVal) {
  const wrap = document.createElement('div');
  wrap.className = 'field-list';

  for (const f of fields) {
    if (f.showIf && getVal(f.showIf.field) !== f.showIf.eq) continue;

    const current = getVal(f.name);
    const value = current === undefined ? f.default : current;
    const widget = makeWidget(f, value, (v, live) => {
      setVal(f, v);
      if (!live) {
        history.endEditSession();
        // Re-render so showIf reacts to the change that was just committed.
        if (f.type === 'bool' || f.type === 'enum' || f.widget === 'select') {
          renderProps(true);
        }
      }
    });
    wrap.append(fieldRow(f.label ?? f.name, widget, f.note, f.unit));
  }
  container.append(wrap);
}

function makeWidget(f, value, onChange) {
  switch (f.type) {
    case 'int':
    case 'float': {
      const step = f.step ?? (f.type === 'int' ? 1 : 0.1);
      if (f.widget === 'slider' && f.min !== undefined && f.max !== undefined) {
        return sliderInput(value ?? f.default ?? 0, f, (v) => onChange(v, false));
      }
      return numberInput(value ?? f.default ?? 0, step, (v, live) => onChange(f.type === 'int' ? Math.round(v) : v, live), f);
    }
    case 'bool': {
      const el = document.createElement('input');
      el.type = 'checkbox';
      el.checked = !!value;
      el.addEventListener('change', () => onChange(el.checked, false));
      return el;
    }
    case 'multiline': {
      const el = document.createElement('textarea');
      el.rows = f.rows ?? 3;
      el.value = value ?? '';
      el.addEventListener('input', () => onChange(el.value, true));
      el.addEventListener('change', () => onChange(el.value, false));
      el.addEventListener('blur', () => history.endEditSession());
      return el;
    }
    case 'enum':
      return selectInput(f.options ?? [], value ?? f.default, (v) => onChange(v, false));
    case 'mapref':
      return mapRefInput(value ?? f.default, (v) => onChange(v, false));
    default:
      if (f.numeric) {
        return numberInput(value ?? f.default ?? 0, f.step ?? 1, onChange, f);
      }
      return textInput(value ?? '', (v, live) => onChange(v, live));
  }
}

function numberInput(value, step, onChange, f) {
  const el = document.createElement('input');
  el.type = 'number';
  el.step = String(step);
  if (f?.min !== undefined) el.min = String(f.min);
  if (f?.max !== undefined) el.max = String(f.max);
  el.value = value === undefined || value === null ? '' : String(value);
  const read = () => {
    if (el.value === '') return f?.numeric ? '' : 0;
    const n = Number(el.value);
    return Number.isFinite(n) ? n : 0;
  };
  el.addEventListener('input', () => onChange(f?.numeric ? (el.value === '' ? '' : String(read())) : read(), true));
  el.addEventListener('change', () => onChange(f?.numeric ? (el.value === '' ? '' : String(read())) : read(), false));
  el.addEventListener('blur', () => history.endEditSession());
  return el;
}

function sliderInput(value, f, onChange) {
  const wrap = document.createElement('div');
  wrap.className = 'slider';

  const range = document.createElement('input');
  range.type = 'range';
  range.min = String(f.min);
  range.max = String(f.max);
  range.step = String(f.step ?? 0.05);
  range.value = String(value);

  const out = document.createElement('span');
  out.className = 'slider-value';
  out.textContent = String(value);

  range.addEventListener('input', () => { out.textContent = range.value; });
  range.addEventListener('change', () => onChange(Number(range.value)));
  wrap.append(range, out);
  return wrap;
}

function textInput(value, onChange) {
  const el = document.createElement('input');
  el.type = 'text';
  el.value = value ?? '';
  el.addEventListener('input', () => onChange(el.value, true));
  el.addEventListener('change', () => onChange(el.value, false));
  el.addEventListener('blur', () => history.endEditSession());
  return el;
}

function selectInput(options, value, onChange) {
  const el = document.createElement('select');
  for (const o of options) {
    const opt = document.createElement('option');
    opt.value = o.value;
    opt.textContent = o.label ?? o.value;
    el.append(opt);
  }
  // Preserve a value the schema does not list rather than silently retargeting.
  if (value !== undefined && !options.some((o) => o.value === value)) {
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = `${value} (not registered)`;
    el.append(opt);
  }
  el.value = value ?? '';
  el.addEventListener('change', () => onChange(el.value));
  return el;
}

/** A dropdown of existing maps, plus a jump button and a missing-target warning. */
function mapRefInput(value, onChange) {
  const wrap = document.createElement('div');
  wrap.className = 'mapref';

  const el = document.createElement('select');
  for (const m of state.maps) {
    const opt = document.createElement('option');
    opt.value = m.id;
    opt.textContent = m.id;
    el.append(opt);
  }
  const known = state.maps.some((m) => m.id === value);
  if (!known && value) {
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = `${value} (missing)`;
    el.append(opt);
    wrap.classList.add('missing');
  }
  el.value = value ?? '';
  el.addEventListener('change', () => onChange(el.value));

  const go = document.createElement('button');
  go.type = 'button';
  go.className = 'icon-btn';
  go.textContent = '↗';
  go.title = 'Open this map';
  go.addEventListener('click', async () => {
    if (!state.maps.some((m) => m.id === el.value)) {
      toast(`${el.value} does not exist`, 'error');
      return;
    }
    const { openMap } = await import('../main.js');
    openMap(el.value);
  });

  wrap.append(el, go);
  return wrap;
}

// ---- small builders ----

function fieldRow(label, control, note, unit) {
  const row = document.createElement('div');
  row.className = 'field-row';

  const l = document.createElement('label');
  l.className = 'field-label';
  l.textContent = label;
  if (note) l.title = note;

  const c = document.createElement('div');
  c.className = 'field-control';
  c.append(control);
  if (unit) {
    const u = document.createElement('span');
    u.className = 'field-unit';
    u.textContent = unit;
    c.append(u);
  }
  row.append(l, c);
  if (note) {
    const n = document.createElement('p');
    n.className = 'field-note';
    n.textContent = note;
    row.append(n);
  }
  return row;
}

function readOnly(text) {
  const el = document.createElement('span');
  el.className = 'readonly';
  el.textContent = text;
  return el;
}

function divider(label) {
  const el = document.createElement('div');
  el.className = 'divider';
  if (label) el.textContent = label;
  return el;
}

function note(text, kind = 'info') {
  const el = document.createElement('p');
  el.className = `panel-note panel-note-${kind}`;
  el.textContent = text;
  return el;
}

function button(label, onClick, kind) {
  const b = document.createElement('button');
  b.type = 'button';
  b.className = 'btn btn-small' + (kind ? ` btn-${kind}` : '');
  b.textContent = label;
  b.addEventListener('click', onClick);
  return b;
}
