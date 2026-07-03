import type { Lang } from './i18n';
import { sortPrimaryFirst } from './guests';
import type { GuestInput, GuestResponse, InviteResponse } from './types.gen';

export type RsvpStatus = 'loading' | 'ready' | 'submitting' | 'confirmed' | 'error';

export interface RsvpState {
  invite: InviteResponse;
  guests: GuestInput[];
  message: string;
  status: RsvpStatus;
  lang: Lang;
  errorMessage?: string;
  // Co-primaries removed in the form, kept (with their names) so they can be
  // added back. Only populated for group invites.
  removedCoPrimaries?: GuestInput[];
}

export function createRsvpState(
  invite: InviteResponse,
  guests: GuestResponse[],
  lang: Lang,
): RsvpState {
  const mapped = sortPrimaryFirst(
    guests.map((guest) => ({
      name: guest.name,
      dietary_preference: guest.dietary_preference,
      alcohol_free: guest.alcohol_free,
      is_primary: guest.is_primary,
      co_primary: guest.co_primary,
    })),
  );
  return { invite, guests: mapped, message: '', status: 'ready', lang };
}

/**
 * A group invite is a unit of co-primaries — no single "primary" and no
 * addable pluses. Detected by the presence of any co-primary guest.
 */
export function isGroupInvite(state: RsvpState): boolean {
  return state.guests.some((guest) => guest.co_primary);
}

export function addGuest(state: RsvpState): RsvpState {
  if (isGroupInvite(state)) {
    return state; // group invites are a fixed list, nothing to add
  }
  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  if (plusCount >= state.invite.max_plus) {
    return state;
  }
  return {
    ...state,
    guests: [
      ...state.guests,
      { name: '', dietary_preference: '', alcohol_free: false, is_primary: false, co_primary: false },
    ],
  };
}

export function removeGuest(state: RsvpState, index: number): RsvpState {
  const guest = state.guests[index];
  if (!guest || guest.is_primary) {
    return state;
  }
  // A group invite must keep at least one co-primary.
  if (isGroupInvite(state) && state.guests.length <= 1) {
    return state;
  }
  // Remember removed co-primaries so they can be added back with their name.
  const removedCoPrimaries = guest.co_primary
    ? [...(state.removedCoPrimaries ?? []), guest]
    : state.removedCoPrimaries;
  return {
    ...state,
    guests: state.guests.filter((_, i) => i !== index),
    removedCoPrimaries,
  };
}

/** Move a previously-removed co-primary back into the guest list. */
export function readdCoPrimary(state: RsvpState, removedIndex: number): RsvpState {
  const removed = state.removedCoPrimaries ?? [];
  const guest = removed[removedIndex];
  if (!guest) {
    return state;
  }
  return {
    ...state,
    guests: [...state.guests, guest],
    removedCoPrimaries: removed.filter((_, i) => i !== removedIndex),
  };
}

export function updateGuest(
  state: RsvpState,
  index: number,
  patch: Partial<GuestInput>,
): RsvpState {
  return {
    ...state,
    guests: state.guests.map((guest, i) => (i === index ? { ...guest, ...patch } : guest)),
  };
}

export function updateMessage(state: RsvpState, message: string): RsvpState {
  return { ...state, message };
}

export function canSubmit(state: RsvpState): boolean {
  const allNamed = state.guests.every((guest) => guest.name.trim().length > 0);
  if (isGroupInvite(state)) {
    return state.guests.length >= 1 && allNamed;
  }
  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  if (plusCount < state.invite.min_plus || plusCount > state.invite.max_plus) {
    return false;
  }
  return allNamed;
}
