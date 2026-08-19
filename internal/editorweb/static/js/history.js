// Undo/redo stacks, with dirty tracking riding the same counter.
//
// The "dirty" flag is seq !== savedSeq rather than a boolean, so undoing back
// past the save point correctly reports the document as clean again. A boolean
// (or a change counter) gets that wrong and leaves you with a permanent unsaved
// marker after you undo an accidental edit.
//
// History is kept per map for the few most recently opened maps, so hopping
// between adjacent world-grid cells does not throw away your undo stack.

import { state, emit, reindex } from './store.js';
import { apply, invert } from './commands.js';

const LIMIT_ENTRIES = 300;
const LIMIT_BYTES = 24 * 1024 * 1024;
const KEEP_MAPS = 3;

function newStack() {
  return { undo: [], redo: [], bytes: 0, seq: 0, savedSeq: 0, dragToken: 0, editSession: 0 };
}

const stacks = new Map(); // mapId -> stack
const recent = [];        // mapId, most recent first
let current = newStack();
let currentID = null;

/** Switches to a map's history, keeping the last few maps' stacks alive. */
export function useMap(mapId) {
  if (currentID === mapId) return;
  if (currentID) stacks.set(currentID, current);

  currentID = mapId;
  current = stacks.get(mapId) ?? newStack();
  stacks.set(mapId, current);

  const at = recent.indexOf(mapId);
  if (at >= 0) recent.splice(at, 1);
  recent.unshift(mapId);
  while (recent.length > KEEP_MAPS) stacks.delete(recent.pop());

  emit('history');
}

/** Marks the current state as saved. */
export function markSaved() {
  current.savedSeq = current.seq;
  emit('history');
}

export const isDirty = () => current.seq !== current.savedSeq;
export const canUndo = () => current.undo.length > 0;
export const canRedo = () => current.redo.length > 0;
export const undoLabel = () => current.undo.at(-1)?.label ?? '';
export const redoLabel = () => current.redo.at(-1)?.label ?? '';

/**
 * Pushes an already-applied command.
 *
 * coalesceKey merges consecutive edits into one entry: a marker drag is one
 * undo, and typing a sign's text is one undo rather than one per keystroke.
 */
export function push(command, coalesceKey = null) {
  if (!command) return;

  const prev = current.undo.at(-1);
  if (coalesceKey && prev && prev.coalesceKey === coalesceKey) {
    prev.after = command.after; // keep prev.before, take the newest after
    current.seq++;
    emit('history', 'doc');
    return;
  }

  command.coalesceKey = coalesceKey;
  current.undo.push(command);
  current.redo.length = 0;
  current.bytes += command.bytes ?? 128;
  current.seq++;

  while (current.undo.length > LIMIT_ENTRIES || current.bytes > LIMIT_BYTES) {
    const dropped = current.undo.shift();
    current.bytes -= dropped.bytes ?? 128;
    // The save point just fell off the end of the stack, so we can no longer
    // prove the document is clean by undoing. Never claim it is.
    if (current.savedSeq >= 0) current.savedSeq = -1;
  }
  emit('history', 'doc');
}

/** Applies a command and records it in one step. */
export function run(command, coalesceKey = null) {
  if (!command) return;
  apply(state.doc, command);
  push(command, coalesceKey);
}

export function undo() {
  const c = current.undo.pop();
  if (!c) return false;
  apply(state.doc, invert(c));
  current.redo.push(c);
  current.seq--;
  afterTimeTravel();
  return true;
}

export function redo() {
  const c = current.redo.pop();
  if (!c) return false;
  apply(state.doc, c);
  current.undo.push(c);
  current.seq++;
  afterTimeTravel();
  return true;
}

function afterTimeTravel() {
  reindex(state.doc);
  // A selected marker may have just been undone out of existence.
  if (state.ui.selection !== null && !state.doc.objById.has(state.ui.selection)) {
    state.ui.selection = null;
  }
  state.validation.stale = true;
  emit('history', 'doc', 'selection', 'validation');
}

/** Starts a new drag; intra-drag moves coalesce into a single undo entry. */
export function newDragToken() {
  return `drag:${++current.dragToken}`;
}

/** Ends the current text/number edit session so the next edit is separate. */
export function endEditSession() {
  current.editSession++;
}

export function editSessionKey(target, name) {
  return `prop:${target}:${name}:${current.editSession}`;
}

/** Discards history for a map, used when reloading it from disk. */
export function reset(mapId) {
  stacks.delete(mapId);
  if (currentID === mapId) {
    current = newStack();
    stacks.set(mapId, current);
  }
  emit('history');
}
