import type { Lang } from './i18n';
import type { GuestInput, GuestResponse, InviteResponse } from './types.gen';

export type RsvpStatus = 'loading' | 'ready' | 'submitting' | 'confirmed' | 'error';

export interface RsvpState {
  invite: InviteResponse;
  guests: GuestInput[];
  message: string;
  status: RsvpStatus;
  lang: Lang;
  errorMessage?: string;
}

export function createRsvpState(
  invite: InviteResponse,
  guests: GuestResponse[],
  lang: Lang,
): RsvpState {
  const mapped = guests.map((guest) => ({
    name: guest.name,
    dietary_preference: guest.dietary_preference,
    alcohol_free: guest.alcohol_free,
    is_primary: guest.is_primary,
  }));
  mapped.sort((a, b) => Number(b.is_primary) - Number(a.is_primary));
  return { invite, guests: mapped, message: '', status: 'ready', lang };
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
  const plusCount = state.guests.filter((guest) => !guest.is_primary).length;
  if (plusCount < state.invite.min_plus || plusCount > state.invite.max_plus) {
    return false;
  }
  return state.guests.every((guest) => guest.name.trim().length > 0);
}
