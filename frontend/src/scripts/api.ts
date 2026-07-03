import { WebClient } from '@anabranch/web-client';
import type { Task } from 'anabranch';
import type { HttpError } from '@anabranch/web-client';
import type {
  CreateInviteRequest,
  GuestInput,
  InviteWithGuestsResponse,
  ListInvitesResponse,
  StatusResponse,
  UpdateInviteRequest,
} from './types.gen';

const baseUrl = import.meta.env.PUBLIC_API_URL;
if (!baseUrl && import.meta.env.PROD) {
  throw new Error('PUBLIC_API_URL is not set');
}
const resolvedBaseUrl = baseUrl ?? 'http://localhost:8080';

const fetchWithCredentials: typeof globalThis.fetch = (input, init) =>
  globalThis.fetch(input, { ...init, credentials: 'include' });

interface ResLike {
  data: unknown;
}

/**
 * Validate and narrow a response payload at the network boundary.
 *
 * The web client types `res.data` as `unknown`; casting it straight to a
 * response type lets backend drift surface later as `undefined`/`NaN`. This
 * checks that `data` is a non-null object and, when a `guard` is supplied, that
 * the expected fields are present, throwing a handled Error otherwise so the
 * failure is caught by the Task's error channel rather than propagating silently.
 */
function unwrap<T>(res: ResLike, guard?: (data: object) => boolean): T {
  const { data } = res;
  if (data === null || typeof data !== 'object') {
    throw new Error('Unexpected API response: missing body');
  }
  if (guard && !guard(data)) {
    throw new Error('Unexpected API response: unexpected shape');
  }
  return data as T;
}

const hasKeys =
  (...keys: string[]) =>
  (data: object): boolean =>
    keys.every((k) => k in data);

const isInviteWithGuests = hasKeys('invite', 'guests');
const isListInvites = hasKeys('invites');
const isStatus = hasKeys('status');

const client = WebClient.create()
  .withBaseUrl(resolvedBaseUrl)
  .withTimeout(10_000)
  .withRetry({ attempts: 3 })
  .withHeaders({ 'Content-Type': 'application/json' })
  .withFetch(fetchWithCredentials);

export const api = {
  getInvite(id: string): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`invites/${id}`).map((r) => unwrap<InviteWithGuestsResponse>(r, isInviteWithGuests));
  },

  rsvp(id: string, guests: GuestInput[], message: string): Task<InviteWithGuestsResponse, HttpError> {
    return client
      .post(`invites/${id}/rsvp`, { guests, message })
      .map((r) => unwrap<InviteWithGuestsResponse>(r, isInviteWithGuests));
  },

  adminLogin(password: string): Task<StatusResponse, HttpError> {
    return client.post('admin/login', { password }).map((r) => unwrap<StatusResponse>(r, isStatus));
  },

  adminLogout(): Task<StatusResponse, HttpError> {
    return client.post('admin/logout', {}).map((r) => unwrap<StatusResponse>(r, isStatus));
  },

  listInvites(): Task<ListInvitesResponse, HttpError> {
    return client.get('admin/invites').map((r) => unwrap<ListInvitesResponse>(r, isListInvites));
  },

  createInvite(body: CreateInviteRequest): Task<InviteWithGuestsResponse, HttpError> {
    return client.post('admin/invites', body).map((r) => unwrap<InviteWithGuestsResponse>(r, isInviteWithGuests));
  },

  getAdminInvite(id: string): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`admin/invites/${id}`).map((r) => unwrap<InviteWithGuestsResponse>(r, isInviteWithGuests));
  },

  updateInvite(id: string, body: UpdateInviteRequest): Task<InviteWithGuestsResponse, HttpError> {
    return client.put(`admin/invites/${id}`, body).map((r) => unwrap<InviteWithGuestsResponse>(r, isInviteWithGuests));
  },

  deleteInvite(id: string): Task<void, HttpError> {
    return client.delete(`admin/invites/${id}`).map(() => undefined);
  },
};
