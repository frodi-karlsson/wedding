import { localePrefix, type Lang } from './i18n';
import { sortPrimaryFirst } from './guests';
import type { GuestResponse, InviteResponse } from './types.gen';

export interface InviteForm {
  id?: string;
  name: string;
  min_plus: number;
  max_plus: number;
  guest_names: string[];
  link_lang: Lang;
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
    guest_names: [''],
    link_lang: lang,
  };
}

export function formFromInvite(
  invite: InviteResponse,
  guests: GuestResponse[],
  lang: Lang,
): InviteForm {
  const names =
    guests.length > 0
      ? sortPrimaryFirst(guests).map((g) => g.name)
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
