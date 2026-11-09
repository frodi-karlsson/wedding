import { localePrefix, type Lang } from './i18n';
import { sortPrimaryFirst } from './guests';
import type { GuestResponse, InviteResponse, PresetGuestInput } from './types.gen';

export interface InviteForm {
  id?: string;
  name: string;
  min_plus: number;
  max_plus: number;
  guests: PresetGuestInput[];
  link_lang: Lang;
  // Group invite: guests are co-primaries (no single primary, no pluses).
  group: boolean;
}

export type AdminAuthenticatedView = 'dashboard' | 'form' | 'submission' | 'invite';

export interface AdminState {
  view: AdminAuthenticatedView;
  lang: Lang;
  invites: InviteResponse[];
  error?: string;
  formError?: string;
  form?: InviteForm;
  // Populated when viewing a single invite's submission (read-only).
  viewInvite?: InviteResponse;
  viewGuests?: GuestResponse[];
  // Populated when showing an invite's shareable link + QR ("Get invite").
  linkInvite?: InviteResponse;
  linkLang?: Lang;
}

export function buildShareLink(origin: string, id: string, lang: Lang): string {
  return `${origin}${localePrefix(lang) || '/'}?id=${id}`;
}

export function createEmptyForm(lang: Lang): InviteForm {
  return {
    name: '',
    min_plus: 0,
    max_plus: 1,
    guests: [{ name: '', alcohol_free: false }],
    link_lang: lang,
    group: false,
  };
}

export function formFromInvite(
  invite: InviteResponse,
  guests: GuestResponse[],
  lang: Lang,
): InviteForm {
  const group = guests.length > 0 && guests.every((g) => g.co_primary);
  const ordered = group ? guests : sortPrimaryFirst(guests);
  const presets =
    guests.length > 0
      ? ordered.map((g) => ({ id: g.id, name: g.name, alcohol_free: g.alcohol_free }))
      : [{ name: invite.name, alcohol_free: false }];
  return {
    id: invite.id,
    name: invite.name,
    min_plus: invite.min_plus,
    max_plus: invite.max_plus,
    guests: presets,
    link_lang: lang,
    group,
  };
}
