// The validation panel.
//
// Validation runs against the IN-MEMORY document, so problems surface while you
// work rather than after you commit them to disk. Clicking an issue jumps to
// whatever it is about.

import { state, emit, subscribe, inBounds } from '../store.js';
import { api } from '../api.js';
import * as render from '../render.js';
import * as view from '../view.js';
import { TILE } from '../tileart.js';

let host, badgeEl;
let debounce = 0;

export function initValidate() {
  host = document.getElementById('validate-body');
  badgeEl = document.getElementById('validate-badge');

  document.getElementById('btn-validate').addEventListener('click', () => runValidation(true));

  subscribe('validation', renderValidation);
  subscribe('doc', scheduleValidation);
}

/** Re-validates shortly after edits stop, so typing does not spam the server. */
function scheduleValidation() {
  if (!state.validation.stale) return;
  clearTimeout(debounce);
  debounce = setTimeout(() => runValidation(false), 900);
}

export async function runValidation(explicit) {
  if (!state.doc || state.validation.running) return;
  state.validation.running = true;
  emit('validation');

  try {
    const res = await api.validate(state.doc.id, state.doc.tmj);
    state.validation = { issues: res.issues ?? [], stale: false, running: false };
  } catch (err) {
    state.validation = {
      issues: [{ severity: 'error', code: 'validate_failed', message: err.message }],
      stale: false, running: false,
    };
  }
  emit('validation');
  if (explicit) {
    const { errorCount, warnCount } = counts();
    const { toast } = await import('./dialogs.js');
    toast(errorCount || warnCount
      ? `${errorCount} error(s), ${warnCount} warning(s)`
      : 'No problems found', errorCount ? 'error' : 'info');
  }
}

function counts() {
  let errorCount = 0, warnCount = 0;
  for (const i of state.validation.issues) {
    if (i.severity === 'error') errorCount++;
    else if (i.severity === 'warn') warnCount++;
  }
  return { errorCount, warnCount };
}

export function errorIssues() {
  return state.validation.issues.filter((i) => i.severity === 'error');
}

function renderValidation() {
  if (!host) return;
  const { errorCount, warnCount } = counts();

  badgeEl.textContent = state.validation.running ? '…'
    : errorCount ? String(errorCount)
    : warnCount ? String(warnCount) : '';
  badgeEl.className = 'badge' + (errorCount ? ' badge-error' : warnCount ? ' badge-warn' : '');
  badgeEl.hidden = !badgeEl.textContent;

  host.replaceChildren();
  if (!state.validation.issues.length) {
    const p = document.createElement('p');
    p.className = 'empty';
    p.textContent = state.validation.running ? 'Checking…' : 'No problems found.';
    host.append(p);
    return;
  }

  const ul = document.createElement('ul');
  ul.className = 'issue-list';
  for (const issue of state.validation.issues) {
    const li = document.createElement('li');
    li.className = `issue issue-${issue.severity}`;

    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'issue-button';
    b.innerHTML =
      `<span class="issue-code">${issue.code}</span>` +
      `<span class="issue-message"></span>`;
    b.querySelector('.issue-message').textContent = issue.message;
    b.addEventListener('click', () => locate(issue));

    li.append(b);
    ul.append(li);
  }
  host.append(ul);
}

/** Jumps to whatever an issue is about: a marker, a tile, or a map property. */
function locate(issue) {
  if (issue.objectId && state.doc.objById.has(issue.objectId)) {
    const obj = state.doc.objById.get(issue.objectId);
    state.ui.selection = issue.objectId;
    state.ui.mode = 'marker';
    centreOn(obj.x, obj.y);
    emit('selection', 'ui');
    render.invalidateAll();
    return;
  }
  if (issue.tileX !== undefined && issue.tileY !== undefined && inBounds(issue.tileX, issue.tileY)) {
    centreOn(issue.tileX * TILE + TILE / 2, issue.tileY * TILE + TILE / 2);
    render.invalidateAll();
  }
}

function centreOn(wx, wy) {
  view.view.ox = wx - view.worldViewW() / 2;
  view.view.oy = wy - view.worldViewH() / 2;
  view.clampPan(state.doc.tmj.width, state.doc.tmj.height);
}
