import { test, expect, vi, afterEach, type Mock } from 'vitest';
import { api } from '../src/scripts/api';

const baseUrl = 'http://localhost:8080';

class FakeHeaders {
  private readonly map = new Map<string, string>();

  constructor(init: Record<string, string> = {}) {
    for (const [key, value] of Object.entries(init)) {
      this.map.set(key.toLowerCase(), value);
    }
  }

  get(name: string): string | null {
    return this.map.get(name.toLowerCase()) ?? null;
  }
}

function fakeResponse(body: unknown) {
  const text = JSON.stringify(body);
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new FakeHeaders({ 'Content-Type': 'application/json' }),
    url: '',
    json: async () => body,
    text: async () => text,
    blob: async () => ({ size: 0, type: '' }) as unknown as Blob,
  };
}

function setupFetch(responseBody: unknown) {
  const fetchMock = vi.fn(() => Promise.resolve(fakeResponse(responseBody)));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock as Mock<(input: string, init: RequestInit) => Promise<ReturnType<typeof fakeResponse>>>;
}

function lastCall(fetchMock: Mock) {
  expect(fetchMock).toHaveBeenCalledOnce();
  return fetchMock.mock.calls[0] as [string, RequestInit];
}

afterEach(() => {
  vi.unstubAllGlobals();
});

test('should GET /invites/{id} and return the invite with guests', async () => {
  const response = {
    invite: { id: 'test-id', name: 'Ada', min_plus: 0, max_plus: 1, submitted: false, message: '' },
    guests: [{ id: 1, name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.getInvite('test-id').run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/invites/test-id`);
  expect(init.method).toBe('GET');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(result).toEqual(response);
});

test('should POST /invites/{id}/rsvp with the guest list', async () => {
  const guests = [{ name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true }];
  const response = {
    invite: { id: 'rsvp-id', name: 'Ada', min_plus: 0, max_plus: 1, submitted: true, message: '' },
    guests: [{ id: 1, name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.rsvp('rsvp-id', guests, '').run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/invites/rsvp-id/rsvp`);
  expect(init.method).toBe('POST');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(JSON.parse(init.body as string)).toEqual({ guests, message: '' });
  expect(result).toEqual(response);
});

test('should POST /admin/login with the password', async () => {
  const response = { status: 'ok' };
  const fetchMock = setupFetch(response);

  const result = await api.adminLogin('secret').run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/login`);
  expect(init.method).toBe('POST');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(JSON.parse(init.body as string)).toEqual({ password: 'secret' });
  expect(result).toEqual(response);
});

test('should POST /admin/logout', async () => {
  const response = { status: 'ok' };
  const fetchMock = setupFetch(response);

  await api.adminLogout().run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/logout`);
  expect(init.method).toBe('POST');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
});

test('should GET /admin/invites', async () => {
  const response = {
    invites: [{ id: 'list-id', name: 'Ada', min_plus: 0, max_plus: 1, submitted: false, message: '' }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.listInvites().run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/invites`);
  expect(init.method).toBe('GET');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(result).toEqual(response);
});

test('should POST /admin/invites with the full CreateInviteRequest body including guest_names', async () => {
  const body = {
    name: 'Ada',
    min_plus: 0,
    max_plus: 1,
    guest_names: ['Ada'],
  };
  const response = {
    invite: { id: 'created-id', name: 'Ada', min_plus: 0, max_plus: 1, submitted: false, message: '' },
    guests: [{ id: 1, name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.createInvite(body).run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/invites`);
  expect(init.method).toBe('POST');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(JSON.parse(init.body as string)).toEqual(body);
  expect(result).toEqual(response);
});

test('should GET /admin/invites/{id} for an admin invite', async () => {
  const response = {
    invite: { id: 'admin-id', name: 'Bob', min_plus: 1, max_plus: 2, submitted: false, message: '' },
    guests: [{ id: 2, name: 'Bob', dietary_preference: 'vegan', alcohol_free: true, is_primary: true }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.getAdminInvite('admin-id').run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/invites/admin-id`);
  expect(init.method).toBe('GET');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(result).toEqual(response);
});

test('should PUT /admin/invites/{id} with the full update body including guest_names', async () => {
  const body = {
    name: 'Ada + Guest',
    min_plus: 0,
    max_plus: 1,
    guest_names: ['Ada + Guest'],
  };
  const response = {
    invite: { id: 'update-id', name: 'Ada + Guest', min_plus: 0, max_plus: 1, submitted: false, message: '' },
    guests: [{ id: 1, name: 'Ada + Guest', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true }],
  };
  const fetchMock = setupFetch(response);

  const result = await api.updateInvite('update-id', body).run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/invites/update-id`);
  expect(init.method).toBe('PUT');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(JSON.parse(init.body as string)).toEqual(body);
  expect(result).toEqual(response);
});

test('should DELETE /admin/invites/{id}', async () => {
  const fetchMock = setupFetch({ status: 'ok' });

  const result = await api.deleteInvite('delete-id').run();

  const [url, init] = lastCall(fetchMock);
  expect(url).toBe(`${baseUrl}/admin/invites/delete-id`);
  expect(init.method).toBe('DELETE');
  expect(init.credentials).toBe('include');
  expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
  expect(result).toBeUndefined();
});
