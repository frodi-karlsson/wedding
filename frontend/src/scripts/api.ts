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

const client = WebClient.create()
  .withBaseUrl(resolvedBaseUrl)
  .withTimeout(10_000)
  .withRetry({ attempts: 3 })
  .withHeaders({ 'Content-Type': 'application/json' })
  .withFetch(fetchWithCredentials);

export const api = {
  getInvite(id: string): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`invites/${id}`).map((r) => r.data as InviteWithGuestsResponse);
  },

  rsvp(id: string, guests: GuestInput[]): Task<InviteWithGuestsResponse, HttpError> {
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

  getAdminInvite(id: string): Task<InviteWithGuestsResponse, HttpError> {
    return client.get(`admin/invites/${id}`).map((r) => r.data as InviteWithGuestsResponse);
  },

  updateInvite(id: string, body: UpdateInviteRequest): Task<InviteWithGuestsResponse, HttpError> {
    return client.put(`admin/invites/${id}`, body).map((r) => r.data as InviteWithGuestsResponse);
  },

  deleteInvite(id: string): Task<void, HttpError> {
    return client.delete(`admin/invites/${id}`).map(() => undefined);
  },
};
