import { WebClient } from '@anabranch/web-client';
import type { Task } from 'anabranch';
import type { HttpError } from '@anabranch/web-client';
import type {
  CreateInviteRequest,
  GuestInput,
  InviteWithGuestsResponse,
  ListInvitesResponse,
  StatusResponse,
} from './types.gen';

const env = import.meta.env as unknown as ImportMetaEnv | undefined;
const baseUrl = env?.PUBLIC_API_URL ?? 'http://localhost:8080';

const fetchWithCredentials: typeof globalThis.fetch = (input, init) =>
  globalThis.fetch(input, { ...init, credentials: 'include' });

const client = WebClient.create()
  .withBaseUrl(baseUrl)
  .withTimeout(10_000)
  .withRetry({ attempts: 3 })
  .withHeaders({ 'Content-Type': 'application/json' })
  .withFetch(fetchWithCredentials);

export const api = {
  getInvite(id: number): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`invites/${id}`).map((r) => r.data as InviteWithGuestsResponse);
  },

  rsvp(id: number, guests: GuestInput[]): Task<InviteWithGuestsResponse, HttpError> {
    return client
      .post(`invites/${id}/rsvp`, { guests })
      .map((r) => r.data as InviteWithGuestsResponse);
  },

  adminLogin(password: string): Task<StatusResponse, HttpError> {
    return client.post('admin/login', { password }).map((r) => r.data as StatusResponse);
  },

  adminLogout(): Task<StatusResponse, HttpError> {
    return client.post('admin/logout', {}).map((r) => r.data as StatusResponse);
  },

  listInvites(): Task<ListInvitesResponse, HttpError> {
    return client.get('admin/invites').map((r) => r.data as ListInvitesResponse);
  },

  createInvite(body: CreateInviteRequest): Task<InviteWithGuestsResponse, HttpError> {
    return client.post('admin/invites', body).map((r) => r.data as InviteWithGuestsResponse);
  },

  getAdminInvite(id: number): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`admin/invites/${id}`).map((r) => r.data as InviteWithGuestsResponse);
  },

  updateInvite(id: number, body: CreateInviteRequest): Task<InviteWithGuestsResponse, HttpError> {
    return client.put(`admin/invites/${id}`, body).map((r) => r.data as InviteWithGuestsResponse);
  },

  deleteInvite(id: number): Task<void, HttpError> {
    return client.delete(`admin/invites/${id}`).map(() => undefined);
  },
};
