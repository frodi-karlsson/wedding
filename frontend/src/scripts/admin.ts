import { api } from './api';
import { translate, type Lang } from './i18n';
import type { Guest, Invite } from './types';

export interface InviteForm {
  id?: number;
  name: string;
  min_plus: number;
  max_plus: number;
  guest_names: string[];
  link_lang: Lang;
}

type AdminView = 'login' | 'dashboard' | 'form';

export interface AdminState {
  view: AdminView;
  lang: Lang;
  invites: Invite[];
  error?: string;
  form?: InviteForm;
}

export function buildShareLink(id: number, lang: Lang): string {
  const prefix = lang === 'en' ? '' : lang;
  return `${window.location.origin}/${prefix}?id=${id}`;
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const ALL_LANGS: Lang[] = ['en', 'is', 'de', 'sv'];

function langOptions(selected: Lang): string {
  return ALL_LANGS.map((l) => `<option value="${l}" ${l === selected ? 'selected' : ''}>${l}</option>`).join('');
}

function createEmptyForm(lang: Lang): InviteForm {
  return {
    name: '',
    min_plus: 0,
    max_plus: 1,
    guest_names: [''],
    link_lang: lang,
  };
}

function formFromInvite(invite: Invite, guests: Guest[], lang: Lang): InviteForm {
  const names =
    guests.length > 0
      ? guests
          .slice()
          .sort((a, b) => Number(b.is_primary) - Number(a.is_primary))
          .map((g) => g.name)
      : [invite.name];
  return {
    id: invite.id,
    name: invite.name,
    min_plus: invite.min_plus,
    max_plus: invite.max_plus,
    guest_names: names,
    link_lang: lang,
  };
}

function renderGuestNames(container: HTMLElement, form: InviteForm, lang: Lang): void {
  const additional = form.guest_names.slice(1);
  if (additional.length === 0) {
    container.innerHTML = '';
    return;
  }
  container.innerHTML = additional
    .map(
      (name, i) => `
        <div class="guest-name-row">
          <input type="text" class="guest-name" data-index="${i + 1}" value="${escapeHtml(name)}">
          <button type="button" data-action="remove-name" data-index="${i + 1}">${escapeHtml(translate('admin_remove_name', lang))}</button>
        </div>
      `,
    )
    .join('');
}

function renderLogin(root: HTMLElement, state: AdminState): void {
  const error = state.error ? `<p class="error">${escapeHtml(state.error)}</p>` : '';
  root.innerHTML = `
    <form class="admin-login" data-action="login">
      <h2>${escapeHtml(translate('admin_title', state.lang))}</h2>
      <label>
        <span>${escapeHtml(translate('admin_password_label', state.lang))}</span>
        <input type="password" name="password" required>
      </label>
      <button type="submit">${escapeHtml(translate('admin_login', state.lang))}</button>
      ${error}
    </form>
  `;
}

function renderDashboard(root: HTMLElement, state: AdminState): void {
  const rows = state.invites
    .map((invite) => {
      const link = buildShareLink(invite.id, state.lang);
      return `
        <tr>
          <td>${invite.id}</td>
          <td>${escapeHtml(invite.name)}</td>
          <td>${invite.min_plus}</td>
          <td>${invite.max_plus}</td>
          <td>${invite.submitted ? '✓' : '—'}</td>
          <td><a href="${escapeHtml(link)}">${escapeHtml(link)}</a></td>
          <td class="actions">
            <select class="link-lang" data-id="${invite.id}">${langOptions(state.lang)}</select>
            <button type="button" data-action="copy-link" data-id="${invite.id}">${escapeHtml(translate('admin_copy_link', state.lang))}</button>
            <button type="button" data-action="edit" data-id="${invite.id}">${escapeHtml(translate('admin_edit', state.lang))}</button>
            <button type="button" data-action="delete" data-id="${invite.id}">${escapeHtml(translate('admin_delete', state.lang))}</button>
          </td>
        </tr>
      `;
    })
    .join('');

  root.innerHTML = `
    <div class="admin-dashboard">
      <div class="toolbar">
        <button type="button" data-action="new-invite">${escapeHtml(translate('admin_new_invite', state.lang))}</button>
        <button type="button" data-action="logout">${escapeHtml(translate('admin_logout', state.lang))}</button>
      </div>
      <table>
        <thead>
          <tr>
            <th>${escapeHtml(translate('admin_id', state.lang))}</th>
            <th>${escapeHtml(translate('admin_name_label', state.lang))}</th>
            <th>${escapeHtml(translate('admin_min_label', state.lang))}</th>
            <th>${escapeHtml(translate('admin_max_label', state.lang))}</th>
            <th>${escapeHtml(translate('admin_submitted', state.lang))}</th>
            <th>${escapeHtml(translate('admin_link', state.lang))}</th>
            <th>${escapeHtml(translate('admin_actions', state.lang))}</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
  `;
}

function renderForm(root: HTMLElement, state: AdminState): void {
  const form = state.form ?? createEmptyForm(state.lang);
  const isCreate = form.id === undefined;
  const submitLabel = translate(isCreate ? 'admin_create' : 'admin_save', state.lang);
  const heading = translate(isCreate ? 'admin_new_invite' : 'admin_edit', state.lang);

  const linkLangField = isCreate
    ? `
        <label>
          <span>${escapeHtml(translate('admin_link_lang_label', state.lang))}</span>
          <select name="link_lang">${langOptions(form.link_lang)}</select>
        </label>
      `
    : '';

  root.innerHTML = `
    <form class="admin-form" data-action="${isCreate ? 'create' : 'update'}">
      <h2>${escapeHtml(heading)}</h2>
      <label>
        <span>${escapeHtml(translate('admin_name_label', state.lang))}</span>
        <input type="text" name="name" value="${escapeHtml(form.guest_names[0])}" required>
      </label>
      <label>
        <span>${escapeHtml(translate('admin_min_label', state.lang))}</span>
        <input type="number" name="min_plus" min="0" value="${form.min_plus}" required>
      </label>
      <label>
        <span>${escapeHtml(translate('admin_max_label', state.lang))}</span>
        <input type="number" name="max_plus" min="0" value="${form.max_plus}" required>
      </label>
      <div class="guest-names">
        <span>${escapeHtml(translate('admin_guest_names_label', state.lang))}</span>
        <div class="guest-names-list"></div>
        <button type="button" data-action="add-name">${escapeHtml(translate('admin_add_name', state.lang))}</button>
      </div>
      ${linkLangField}
      <div class="form-actions">
        <button type="submit">${escapeHtml(submitLabel)}</button>
        <button type="button" data-action="cancel">${escapeHtml(translate('admin_cancel', state.lang))}</button>
      </div>
    </form>
  `;

  const list = root.querySelector('.guest-names-list');
  if (list instanceof HTMLElement) {
    renderGuestNames(list, form, state.lang);
  }
}

function render(root: HTMLElement, state: AdminState): void {
  if (state.view === 'login') {
    renderLogin(root, state);
  } else if (state.view === 'dashboard') {
    renderDashboard(root, state);
  } else {
    renderForm(root, state);
  }
}

export async function mountAdmin(root: HTMLElement, lang: Lang): Promise<void> {
  let state: AdminState = { view: 'login', lang, invites: [] };

  const update = (next: AdminState): void => {
    state = next;
    render(root, state);
  };

  const refreshDashboard = async (): Promise<void> => {
    try {
      const response = await api.listInvites().run();
      update({ ...state, view: 'dashboard', invites: response.invites, error: undefined });
    } catch {
      update({ ...state, view: 'login', error: undefined });
    }
  };

  root.addEventListener('submit', async (event) => {
    event.preventDefault();
    const formEl = event.target as HTMLElement;
    const action = formEl.dataset.action;

    if (action === 'login') {
      const input = root.querySelector('input[name="password"]') as HTMLInputElement | null;
      const password = input?.value ?? '';
      try {
        await api.adminLogin(password).run();
        await refreshDashboard();
      } catch {
        update({ ...state, error: translate('admin_login_error', state.lang) });
      }
      return;
    }

    if (!state.form) return;

    if (action === 'create' || action === 'update') {
      const nameInput = root.querySelector('input[name="name"]') as HTMLInputElement | null;
      const minInput = root.querySelector('input[name="min_plus"]') as HTMLInputElement | null;
      const maxInput = root.querySelector('input[name="max_plus"]') as HTMLInputElement | null;
      const linkLangSelect = root.querySelector('select[name="link_lang"]') as HTMLSelectElement | null;

      const name = nameInput?.value.trim() ?? '';
      const min_plus = Number(minInput?.value);
      const max_plus = Number(maxInput?.value);
      const link_lang = (linkLangSelect?.value as Lang) ?? state.lang;

      if (!name || Number.isNaN(min_plus) || Number.isNaN(max_plus) || min_plus > max_plus) {
        return;
      }

      const additionalInputs = root.querySelectorAll('.guest-name') as NodeListOf<HTMLInputElement>;
      const additional = Array.from(additionalInputs).map((input) => input.value.trim());
      const guest_names = [name, ...additional.filter((n) => n.length > 0)];

      const body = { name, min_plus, max_plus, guest_names };

      try {
        const response =
          action === 'create'
            ? await api.createInvite(body).run()
            : await api.updateInvite(state.form.id!, body).run();

        if (action === 'create') {
          const shareLink = buildShareLink(response.invite.id, link_lang);
          window.alert(shareLink);
        }

        await refreshDashboard();
      } catch {
        // Error swallowed; UI could be extended with a form-level error.
      }
    }
  });

  root.addEventListener('click', async (event) => {
    const target = event.target as HTMLElement;
    const action = target.dataset.action;
    const id = Number(target.dataset.id);

    switch (action) {
      case 'new-invite': {
        update({ ...state, view: 'form', form: createEmptyForm(state.lang) });
        break;
      }
      case 'logout': {
        try {
          await api.adminLogout().run();
        } catch {
          // ignore logout errors
        }
        update({ ...state, view: 'login', invites: [], error: undefined });
        break;
      }
      case 'cancel': {
        await refreshDashboard();
        break;
      }
      case 'add-name': {
        if (state.form) {
          state.form.guest_names.push('');
          const list = root.querySelector('.guest-names-list');
          if (list instanceof HTMLElement) {
            renderGuestNames(list, state.form, state.lang);
          }
        }
        break;
      }
      case 'remove-name': {
        if (state.form) {
          const index = Number(target.dataset.index);
          state.form.guest_names.splice(index, 1);
          const list = root.querySelector('.guest-names-list');
          if (list instanceof HTMLElement) {
            renderGuestNames(list, state.form, state.lang);
          }
        }
        break;
      }
      case 'edit': {
        try {
          const response = await api.getAdminInvite(id).run();
          update({
            ...state,
            view: 'form',
            form: formFromInvite(response.invite, response.guests, state.lang),
          });
        } catch {
          // ignore load error
        }
        break;
      }
      case 'delete': {
        if (window.confirm(translate('admin_delete_confirm', state.lang))) {
          try {
            await api.deleteInvite(id).run();
          } catch {
            // ignore delete error
          }
          await refreshDashboard();
        }
        break;
      }
      case 'copy-link': {
        const select = root.querySelector(`select.link-lang[data-id="${id}"]`) as HTMLSelectElement | null;
        const linkLang = (select?.value as Lang) ?? state.lang;
        const link = buildShareLink(id, linkLang);
        try {
          await navigator.clipboard.writeText(link);
          const original = target.textContent ?? '';
          target.textContent = translate('admin_copied', state.lang);
          setTimeout(() => {
            target.textContent = original;
          }, 1500);
        } catch {
          // ignore clipboard errors
        }
        break;
      }
    }
  });

  root.addEventListener('input', (event) => {
    const target = event.target as HTMLInputElement;
    if (!state.form) return;

    if (target.name === 'name') {
      state.form.guest_names[0] = target.value;
      return;
    }

    if (target.classList.contains('guest-name')) {
      const index = Number(target.dataset.index);
      if (state.form.guest_names[index] !== undefined) {
        state.form.guest_names[index] = target.value;
      }
    }
  });

  render(root, state);

  try {
    const response = await api.listInvites().run();
    update({ ...state, view: 'dashboard', invites: response.invites });
  } catch {
    update({ ...state, view: 'login' });
  }
}
