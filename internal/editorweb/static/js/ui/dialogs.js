// Toasts and modal dialogs, built on <dialog> so focus trapping and Escape are
// the browser's problem rather than ours.

let toastHost = null;

export function initDialogs() {
  toastHost = document.getElementById('toasts');
}

export function toast(message, kind = 'info', ms = 3200) {
  if (!toastHost) return;
  const el = document.createElement('div');
  el.className = `toast toast-${kind}`;
  el.textContent = message;
  toastHost.append(el);
  setTimeout(() => {
    el.classList.add('leaving');
    setTimeout(() => el.remove(), 200);
  }, ms);
}

/**
 * Shows a modal with arbitrary buttons and resolves with the chosen value.
 * Dismissing (Escape, backdrop) resolves with null.
 */
export function choose({ title, body, buttons }) {
  return new Promise((resolve) => {
    const dlg = document.createElement('dialog');
    dlg.className = 'modal';

    const h = document.createElement('h2');
    h.textContent = title;
    dlg.append(h);

    if (body) {
      const p = document.createElement('div');
      p.className = 'modal-body';
      if (typeof body === 'string') p.textContent = body;
      else p.append(body);
      dlg.append(p);
    }

    const row = document.createElement('div');
    row.className = 'modal-buttons';
    for (const b of buttons) {
      const btn = document.createElement('button');
      btn.textContent = b.label;
      btn.className = b.kind ? `btn btn-${b.kind}` : 'btn';
      btn.addEventListener('click', () => {
        dlg.close();
        resolve(b.value);
      });
      row.append(btn);
    }
    dlg.append(row);

    dlg.addEventListener('close', () => {
      dlg.remove();
      resolve(null);
    }, { once: true });

    document.body.append(dlg);
    dlg.showModal();
    row.querySelector('button')?.focus();
  });
}

export function confirmDiscard(mapId) {
  return choose({
    title: 'Unsaved changes',
    body: `${mapId} has unsaved edits.`,
    buttons: [
      { label: 'Save and continue', value: 'save', kind: 'primary' },
      { label: 'Discard', value: 'discard', kind: 'danger' },
      { label: 'Cancel', value: null },
    ],
  });
}

/**
 * The conflict dialog. It fires when the file changed under us — most often
 * because the in-game editor saved the same map.
 */
export function confirmOverwrite(mapId) {
  return choose({
    title: 'Changed on disk',
    body: `${mapId} was modified outside this editor since you opened it. `
        + 'Overwriting replaces those changes; reloading discards yours.',
    buttons: [
      { label: 'Overwrite disk', value: 'overwrite', kind: 'danger' },
      { label: 'Reload from disk', value: 'reload' },
      { label: 'Cancel', value: null },
    ],
  });
}

/** Offers to save despite validation errors. */
export function confirmInvalid(issues) {
  const list = document.createElement('ul');
  list.className = 'issue-list';
  for (const i of issues.slice(0, 8)) {
    const li = document.createElement('li');
    li.textContent = i.message;
    list.append(li);
  }
  return choose({
    title: `${issues.length} problem${issues.length === 1 ? '' : 's'} would break this map`,
    body: list,
    buttons: [
      { label: 'Save anyway', value: 'force', kind: 'danger' },
      { label: 'Cancel', value: null, kind: 'primary' },
    ],
  });
}
