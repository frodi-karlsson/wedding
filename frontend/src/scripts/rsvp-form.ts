import { api } from './api';
import { translate, type Lang } from './i18n';
import { escapeHtml } from './html';
import type { GuestResponse, GuestInput, InviteResponse } from './types.gen';

export type RsvpStatus = 'loading' | 'ready' | 'submitting' | 'confirmed' | 'error';

export interface RsvpState {
  invite: InviteResponse;
  guests: GuestInput[];
  status: RsvpStatus;
  lang: Lang;
  errorMessage?: string;
}

export function createRsvpState(invite: InviteResponse, guests: GuestResponse[], lang: Lang): RsvpState {
  const mapped = guests.map((guest) => ({
    name: guest.name,
    dietary_preference: guest.dietary_preference,
    alcohol_free: guest.alcohol_free,
    is_primary: guest.is_primary,
  }));
  mapped.sort((a, b) => Number(b.is_primary) - Number(a.is_primary));
  return { invite, guests: mapped, status: 'ready', lang };
}

export function addGuest(state: RsvpState): RsvpState {
  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  if (plusCount >= state.invite.max_plus) {
    return state;
  }
  return {
    ...state,
    guests: [
      ...state.guests,
      { name: '', dietary_preference: '', alcohol_free: false, is_primary: false },
    ],
  };
}

export function removeGuest(state: RsvpState, index: number): RsvpState {
  if (state.guests[index]?.is_primary) {
    return state;
  }
  return {
    ...state,
    guests: state.guests.filter((_, i) => i !== index),
  };
}

export function updateGuest(state: RsvpState, index: number, patch: Partial<GuestInput>): RsvpState {
  return {
    ...state,
    guests: state.guests.map((guest, i) => (i === index ? { ...guest, ...patch } : guest)),
  };
}

export function canSubmit(state: RsvpState): boolean {
  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  if (plusCount < state.invite.min_plus || plusCount > state.invite.max_plus) {
    return false;
  }
  return state.guests.every((guest) => guest.name.trim().length > 0);
}

export function guestsToInput(state: RsvpState): GuestInput[] {
  return state.guests;
}

function updateSubmitDisabled(root: HTMLElement, state: RsvpState): void {
  const submitBtn = root.querySelector('.rsvp-form .submit') as HTMLButtonElement | null;
  if (submitBtn) {
    submitBtn.disabled = !canSubmit(state) || state.status === 'submitting';
  }
}

function clearInlineError(root: HTMLElement): void {
  const errorEl = root.querySelector('.rsvp-form > .error') as HTMLElement | null;
  if (errorEl) {
    errorEl.remove();
  }
}

export function render(root: HTMLElement, state: RsvpState): void {
  const { lang } = state;

  if (state.status === 'loading') {
    root.innerHTML = `<p class="loading">${escapeHtml(translate('loading', lang))}</p>`;
    return;
  }

  if (state.status === 'error') {
    root.innerHTML = `<p class="error">${escapeHtml(state.errorMessage ?? translate('load_error', lang))}</p>`;
    return;
  }

  if (state.status === 'confirmed') {
    root.innerHTML = `
      <div class="confirmed card">
        <h2 class="heading heading--md">${escapeHtml(translate('thank_you', lang))}</h2>
        <p>${escapeHtml(translate('thank_you_body', lang))}</p>
      </div>
    `;
    return;
  }

  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  const canAdd = plusCount < state.invite.max_plus;
  const submitDisabled = state.status === 'submitting' || !canSubmit(state);
  const minClause =
    state.invite.min_plus > 0
      ? escapeHtml(
          translate('min_clause', lang).replace('{min}', String(state.invite.min_plus)),
        )
      : '';
  const intro = escapeHtml(translate('rsvp_intro', lang, state.invite.max_plus))
    .replace('{max}', String(state.invite.max_plus))
    .replace('{min_clause}', minClause);

  const rows = state.guests
    .map((guest, index) => {
      const nameLabel = escapeHtml(translate(guest.is_primary ? 'name_you_label' : 'name_label', lang));
      const legend = guest.is_primary
        ? escapeHtml(translate('name_you_label', lang))
        : `${escapeHtml(translate('guest_label', lang))} ${index}`;
      const removeButton = guest.is_primary
        ? ''
        : `<button type="button" class="btn btn--ghost btn--sm remove" data-action="remove" data-index="${index}">${escapeHtml(translate('remove_guest', lang))}</button>`;
      return `
        <fieldset class="guest-row" data-index="${index}">
          <legend>${legend}</legend>
          <label>
            <span>${nameLabel}</span>
            <input type="text" value="${escapeHtml(guest.name)}" data-action="update" data-index="${index}" data-field="name" required>
          </label>
          <label>
            <span>${escapeHtml(translate('dietary_label', lang))}</span>
            <input type="text" value="${escapeHtml(guest.dietary_preference)}" data-action="update" data-index="${index}" data-field="dietary">
          </label>
          <label class="checkbox">
            <input type="checkbox" ${guest.alcohol_free ? 'checked' : ''} data-action="update" data-index="${index}" data-field="alcohol_free">
            <span>${escapeHtml(translate('alcohol_free_label', lang))}</span>
          </label>
          ${removeButton}
        </fieldset>
      `;
    })
    .join('');

  const addButton = canAdd
    ? `<button type="button" class="btn btn--secondary btn--md add" data-action="add">${escapeHtml(translate('add_guest', lang))}</button>`
    : '';
  const submitLabel =
    state.status === 'submitting'
      ? escapeHtml(translate('submitting', lang))
      : escapeHtml(translate('submit', lang));

  const submitError =
    state.status === 'ready' && state.errorMessage
      ? `<p class="error">${escapeHtml(state.errorMessage)}</p>`
      : '';

  root.innerHTML = `
    <form class="rsvp-form card" data-action="submit">
      <h2 class="heading heading--md">${escapeHtml(state.invite.name)}</h2>
      <p class="intro">${intro}</p>
      <div class="guests">${rows}</div>
      <div class="actions">
        ${addButton}
        <button type="submit" class="btn btn--primary btn--md submit" ${submitDisabled ? 'disabled' : ''}>${submitLabel}</button>
      </div>
      ${submitError}
    </form>
  `;
}

export async function mountRsvpForm(root: HTMLElement, inviteId: string, lang: Lang): Promise<void> {
  let state: RsvpState = {
    invite: { id: '', name: '', min_plus: 0, max_plus: 0, submitted: false },
    guests: [],
    status: 'loading',
    lang,
  };

  const update = (next: RsvpState) => {
    state = next;
    render(root, state);
  };

  root.addEventListener('click', (event) => {
    const target = event.target as HTMLElement;
    const action = target.dataset.action;
    if (action === 'add') {
      event.preventDefault();
      state = { ...addGuest(state), errorMessage: undefined };
      render(root, state);
    } else if (action === 'remove') {
      event.preventDefault();
      const index = Number(target.dataset.index);
      state = { ...removeGuest(state, index), errorMessage: undefined };
      render(root, state);
    }
  });

  root.addEventListener('input', (event) => {
    const target = event.target as HTMLInputElement;
    if (target.dataset.action !== 'update') return;
    const index = Number(target.dataset.index);
    const field = target.dataset.field as 'name' | 'dietary' | 'alcohol_free';
    const patch: Partial<GuestInput> = {};
    if (field === 'alcohol_free') {
      patch.alcohol_free = target.checked;
    } else if (field === 'dietary') {
      patch.dietary_preference = target.value;
    } else if (field === 'name') {
      patch.name = target.value;
    }
    state = updateGuest(state, index, patch);
    if (state.errorMessage) {
      state = { ...state, errorMessage: undefined };
      clearInlineError(root);
    }
    if (field === 'name') {
      updateSubmitDisabled(root, state);
    }
  });

  root.addEventListener('submit', (event) => {
    event.preventDefault();
    if (!canSubmit(state)) return;
    const guests = guestsToInput(state);
    state = { ...state, status: 'submitting', errorMessage: undefined };
    render(root, state);
    api
      .rsvp(inviteId, guests)
      .run()
      .then(() => {
        state = { ...state, status: 'confirmed' };
        render(root, state);
      })
      .catch(() => {
        state = { ...state, status: 'ready', errorMessage: translate('submit_error', state.lang) };
        render(root, state);
      });
  });

  render(root, state);
  root.classList.add('fade-in');

  try {
    const response = await api.getInvite(inviteId).run();
    update(createRsvpState(response.invite, response.guests, lang));
  } catch {
    update({ ...state, status: 'error', errorMessage: translate('load_error', lang) });
  }
}
