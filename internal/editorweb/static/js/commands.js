// Undo/redo commands.
//
// These are patches, not snapshots: each records only what changed plus what it
// replaced. A full-map flood fill on a 25x21 map costs about 6KB rather than two
// whole-map copies, and a property edit costs a few dozen bytes.
//
// apply() and invert() are pure functions of (doc, cmd). Redo is just apply()
// again, so there is only one code path to keep correct.

import { reindex, setObjProp, removeObjProp, setMapProp } from './store.js';

/** A batch of tile edits, accumulated during one gesture. */
export class TileEditBatch {
  constructor(layer, label) {
    this.layer = layer;
    this.label = label;
    this.edits = new Map(); // index -> {before, after}
  }

  /** Applies one tile edit immediately so the stroke is visible as you draw. */
  set(index, gid) {
    const current = this.layer.data[index];
    if (current === gid) return false;
    const existing = this.edits.get(index);
    if (existing) {
      existing.after = gid;
    } else {
      this.edits.set(index, { before: current, after: gid });
    }
    this.layer.data[index] = gid;
    return true;
  }

  /**
   * Converts the batch into one undoable command, or null if nothing actually
   * changed (so a click that repaints the same tile adds no history entry).
   */
  commit() {
    const live = [...this.edits.entries()].filter(([, e]) => e.before !== e.after);
    if (live.length === 0) return null;

    const idx = new Int32Array(live.length);
    const before = new Int32Array(live.length);
    const after = new Int32Array(live.length);
    live.forEach(([index, e], i) => {
      idx[i] = index;
      before[i] = e.before;
      after[i] = e.after;
    });

    return {
      type: 'tiles',
      label: this.label,
      layerId: this.layer.id,
      idx, before, after,
      bytes: live.length * 12 + 64,
    };
  }
}

export const cmd = {
  markerAdd: (obj, index) => ({
    type: 'marker.add', label: `add ${obj.type}`,
    obj: structuredClone(obj), index, bytes: 256,
  }),
  markerDelete: (obj, index) => ({
    type: 'marker.del', label: `delete ${obj.type}`,
    obj: structuredClone(obj), index, bytes: 256,
  }),
  markerMove: (id, before, after) => ({
    type: 'marker.move', label: 'move marker',
    id, before: { ...before }, after: { ...after }, bytes: 96,
  }),
  markerResize: (id, before, after) => ({
    type: 'marker.resize', label: 'resize marker',
    id, before: { ...before }, after: { ...after }, bytes: 128,
  }),
  markerReplace: (id, before, after) => ({
    type: 'marker.replace', label: 'change marker',
    id, before: structuredClone(before), after: structuredClone(after), bytes: 512,
  }),
  objProp: (id, name, ptype, before, after) => ({
    type: 'prop.obj', label: `set ${name}`,
    id, name, ptype, before, after, bytes: 160,
  }),
  mapProp: (name, ptype, before, after) => ({
    type: 'prop.map', label: `set ${name}`,
    name, ptype, before, after, bytes: 160,
  }),
  layerAdd: (layer, index) => ({
    type: 'layer.add', label: `add layer ${layer.name}`,
    layer: structuredClone(layer), index,
    bytes: (layer.data?.length ?? 0) * 4 + 256,
  }),
  layerRemove: (layer, index) => ({
    type: 'layer.del', label: `remove layer ${layer.name}`,
    layer: structuredClone(layer), index,
    bytes: (layer.data?.length ?? 0) * 4 + 256,
  }),
};

/** Applies a command to the document. */
export function apply(doc, c) {
  switch (c.type) {
    case 'tiles': {
      const layer = findLayerByID(doc, c.layerId);
      if (!layer) return;
      for (let i = 0; i < c.idx.length; i++) layer.data[c.idx[i]] = c.after[i];
      break;
    }
    case 'marker.add': {
      const objects = markersObjects(doc);
      objects.splice(c.index, 0, structuredClone(c.obj));
      reindex(doc);
      break;
    }
    case 'marker.del': {
      const objects = markersObjects(doc);
      objects.splice(c.index, 1);
      reindex(doc);
      break;
    }
    case 'marker.move':
    case 'marker.resize': {
      const o = doc.objById.get(c.id);
      if (o) Object.assign(o, c.after);
      break;
    }
    case 'marker.replace': {
      const objects = markersObjects(doc);
      const i = objects.findIndex((o) => o.id === c.id);
      if (i >= 0) objects[i] = structuredClone(c.after);
      reindex(doc);
      break;
    }
    case 'prop.obj': {
      const o = doc.objById.get(c.id);
      if (!o) break;
      if (c.after === undefined) removeObjProp(o, c.name);
      else setObjProp(o, c.name, c.ptype, c.after);
      break;
    }
    case 'prop.map':
      setMapProp(c.name, c.ptype, c.after);
      break;
    case 'layer.add':
      doc.tmj.layers.splice(c.index, 0, structuredClone(c.layer));
      reindex(doc);
      break;
    case 'layer.del':
      doc.tmj.layers.splice(c.index, 1);
      reindex(doc);
      break;
  }
}

/** Returns the inverse of a command. Applying it undoes the original. */
export function invert(c) {
  switch (c.type) {
    case 'tiles':
      return { ...c, before: c.after, after: c.before };
    case 'marker.add':
      return { ...c, type: 'marker.del' };
    case 'marker.del':
      return { ...c, type: 'marker.add' };
    case 'marker.move':
    case 'marker.resize':
    case 'marker.replace':
    case 'prop.obj':
    case 'prop.map':
      return { ...c, before: c.after, after: c.before };
    case 'layer.add':
      return { ...c, type: 'layer.del' };
    case 'layer.del':
      return { ...c, type: 'layer.add' };
    default:
      return c;
  }
}

function findLayerByID(doc, id) {
  return (doc.tmj.layers ?? []).find((l) => l.id === id) ?? null;
}

function markersObjects(doc) {
  const layer = (doc.tmj.layers ?? []).find((l) => l.type === 'objectgroup' && l.name === 'markers');
  if (!layer) return [];
  if (!layer.objects) layer.objects = [];
  return layer.objects;
}
