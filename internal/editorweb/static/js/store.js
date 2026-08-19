// The document, the UI state, and a tiny pub/sub.
//
// Two rules govern the document and both matter:
//
//  1. NEVER rebuild doc.tmj. Mutate layer.data[i], obj.x, prop.value in place.
//     The server re-encodes through tiled.Map on save, so key order does not
//     matter, but rebuilding in JS would drop any field this app does not know
//     about instead of passing it through.
//  2. Derived indexes (tileLayers, objById) are rebuilt from tmj, never
//     authoritative. If they disagree with tmj, tmj wins.

import { TILE } from './tileart.js';

const channels = new Map();

/** Subscribes to a channel. Returns an unsubscribe function. */
export function subscribe(channel, fn) {
  if (!channels.has(channel)) channels.set(channel, new Set());
  channels.get(channel).add(fn);
  return () => channels.get(channel).delete(fn);
}

export function emit(...names) {
  for (const name of names) {
    for (const fn of channels.get(name) ?? []) {
      try {
        fn();
      } catch (err) {
        console.error(`subscriber for "${name}" failed`, err);
      }
    }
  }
}

export const state = {
  schema: null,
  atlas: null,
  maps: [],
  mapIndex: new Map(),
  doc: null,
  health: null,

  ui: {
    mode: 'tile',            // 'tile' | 'marker'
    tool: 'brush',
    activeLayer: 0,          // index into doc.tileLayers
    layerView: 'dim',        // 'all' | 'dim' | 'isolate'
    showGrid: true,
    showMarkersInTileMode: true,
    brushGID: 1,
    favorites: [],
    markerType: 'enemy',
    pickupKind: 'coin',
    filterPaletteByLayer: true,
    paletteQuery: '',
    mapQuery: '',
    selection: null,         // object id
    hover: null,             // object id
    cursor: null,            // {tx, ty, wx, wy}
    status: '',
  },

  validation: { issues: [], stale: true, running: false },
};

/** Wraps a loaded map payload into the editable document. */
export function openDoc(payload) {
  const doc = {
    id: payload.id,
    file: payload.file,
    group: payload.group,
    grid: payload.grid ?? null,
    etag: payload.etag,
    reformats: !!payload.reformats,
    unmodeledFields: payload.unmodeledFields ?? [],
    tmj: payload.map,
    tileLayers: [],
    markersLayer: null,
    objById: new Map(),
  };
  reindex(doc);
  state.doc = doc;
  state.validation = { issues: payload.issues ?? [], stale: false, running: false };

  state.ui.activeLayer = Math.max(0, doc.tileLayers.length - 1);
  state.ui.selection = null;
  state.ui.hover = null;
  return doc;
}

/** Rebuilds the derived indexes. Call after any structural change. */
export function reindex(doc) {
  doc.tileLayers = [];
  doc.markersLayer = null;
  doc.objById = new Map();

  const layers = doc.tmj.layers ?? [];
  for (let i = 0; i < layers.length; i++) {
    const l = layers[i];
    if (l.type === 'tilelayer') {
      doc.tileLayers.push({ layer: l, index: i });
    } else if (l.type === 'objectgroup' && l.name === 'markers') {
      doc.markersLayer = l;
    }
  }
  for (const o of doc.markersLayer?.objects ?? []) {
    doc.objById.set(o.id, o);
  }
}

export const mapW = () => state.doc?.tmj.width ?? 0;
export const mapH = () => state.doc?.tmj.height ?? 0;
export const worldW = () => mapW() * TILE;
export const worldH = () => mapH() * TILE;

export function activeLayer() {
  const doc = state.doc;
  if (!doc) return null;
  return doc.tileLayers[state.ui.activeLayer]?.layer ?? null;
}

export function inBounds(tx, ty) {
  return tx >= 0 && ty >= 0 && tx < mapW() && ty < mapH();
}

export function tileIndex(tx, ty) {
  return ty * mapW() + tx;
}

/** GID at a tile in a specific layer. */
export function gidAt(layer, tx, ty) {
  if (!layer || !inBounds(tx, ty)) return 0;
  return layer.data?.[tileIndex(tx, ty)] ?? 0;
}

/**
 * The GID actually visible at a tile: the topmost layer with a non-zero value.
 *
 * Mirrors the renderer, where gid 0 is a transparency hole rather than black.
 * Returns {gid, layerIndex}, with layerIndex -1 when every layer is empty.
 */
export function topGidAt(tx, ty) {
  const doc = state.doc;
  if (!doc) return { gid: 0, layerIndex: -1 };
  for (let i = doc.tileLayers.length - 1; i >= 0; i--) {
    const gid = gidAt(doc.tileLayers[i].layer, tx, ty);
    if (gid !== 0) return { gid, layerIndex: i };
  }
  return { gid: 0, layerIndex: -1 };
}

/** Allocates the next Tiled object id, keeping nextobjectid consistent. */
export function allocObjectID() {
  const tmj = state.doc.tmj;
  let next = tmj.nextobjectid ?? 1;
  let max = 0;
  for (const l of tmj.layers ?? []) {
    for (const o of l.objects ?? []) max = Math.max(max, o.id ?? 0);
  }
  if (next <= max) next = max + 1;
  tmj.nextobjectid = next + 1;
  return next;
}

/**
 * Ensures a "markers" object group exists, creating it lazily.
 *
 * The in-game editor injects missing layers on open, which silently modifies
 * every map you look at. Here the layer appears only when you actually add a
 * marker, as part of that undoable action.
 */
export function ensureMarkersLayer() {
  const doc = state.doc;
  if (doc.markersLayer) return doc.markersLayer;

  const layer = {
    id: nextLayerID(doc.tmj),
    type: 'objectgroup',
    name: 'markers',
    visible: true,
    opacity: 1,
    x: 0,
    y: 0,
    objects: [],
  };
  doc.tmj.layers.push(layer);
  reindex(doc);
  return doc.markersLayer;
}

export function nextLayerID(tmj) {
  let next = tmj.nextlayerid ?? 1;
  let max = 0;
  for (const l of tmj.layers ?? []) max = Math.max(max, l.id ?? 0);
  if (next <= max) next = max + 1;
  tmj.nextlayerid = next + 1;
  return next;
}

/** Map-level Tiled property value, or undefined. */
export function mapProp(name) {
  return (state.doc?.tmj.properties ?? []).find((p) => p.name === name)?.value;
}

export function setMapProp(name, type, value) {
  const tmj = state.doc.tmj;
  if (!tmj.properties) tmj.properties = [];
  const existing = tmj.properties.find((p) => p.name === name);
  if (value === undefined || value === null || value === '') {
    tmj.properties = tmj.properties.filter((p) => p.name !== name);
    if (tmj.properties.length === 0) delete tmj.properties;
    return;
  }
  if (existing) {
    existing.value = value;
    existing.type = type;
  } else {
    tmj.properties.push({ name, type, value });
  }
}

/** Object-level Tiled property value, or undefined. */
export function objProp(obj, name) {
  return (obj.properties ?? []).find((p) => p.name === name)?.value;
}

export function setObjProp(obj, name, type, value) {
  if (!obj.properties) obj.properties = [];
  const existing = obj.properties.find((p) => p.name === name);
  if (existing) {
    existing.value = value;
    existing.type = type;
  } else {
    obj.properties.push({ name, type, value });
  }
}

export function removeObjProp(obj, name) {
  if (!obj.properties) return;
  obj.properties = obj.properties.filter((p) => p.name !== name);
  if (obj.properties.length === 0) delete obj.properties;
}

export function markerSchema(type) {
  return state.schema?.markers.find((m) => m.type === type) ?? null;
}
