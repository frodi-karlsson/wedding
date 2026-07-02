import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { AdminPanel } from '../src/islands/AdminPanel';
import { api } from '../src/scripts/api';
import type { ListInvitesResponse, StatusResponse } from '../src/scripts/types.gen';
import type { Task } from 'anabranch';
import type { HttpError } from '@anabranch/web-client';

function mockTask<T>(value: T): Task<T, HttpError> {
  return { run: vi.fn().mockResolvedValue(value) } as unknown as Task<T, HttpError>;
}

function mockFailingTask<T>(error: Error): Task<T, HttpError> {
  return { run: vi.fn().mockRejectedValue(error) } as unknown as Task<T, HttpError>;
}

const mockListResponse: ListInvitesResponse = {
  invites: [
    { id: '1', name: 'Ada', min_plus: 0, max_plus: 2, submitted: false },
    { id: '2', name: 'Bob', min_plus: 1, max_plus: 3, submitted: true },
  ],
};

const mockLoginResponse: StatusResponse = { status: 'ok' };

beforeEach(() => {
  vi.stubGlobal('location', {
    origin: 'https://example.com',
    pathname: '/admin',
    search: '',
    hash: '',
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

test('should render the dashboard when listInvites succeeds', async () => {
  vi.spyOn(api, 'listInvites').mockReturnValue(mockTask(mockListResponse));

  render(() => <AdminPanel lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada')).toBeInTheDocument();
    expect(screen.getByText('Bob')).toBeInTheDocument();
  });
});

test('should render the login view when listInvites fails', async () => {
  vi.spyOn(api, 'listInvites').mockReturnValue(mockFailingTask(new Error('Unauthorized')));

  render(() => <AdminPanel lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Log in')).toBeInTheDocument();
  });
});

test('should show the create form when clicking new invite', async () => {
  vi.spyOn(api, 'listInvites').mockReturnValue(mockTask(mockListResponse));

  render(() => <AdminPanel lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada')).toBeInTheDocument();
  });

  fireEvent.click(screen.getByText('+ New invite'));

  expect(screen.getByText('Create invite')).toBeInTheDocument();
});

test('should log out and return to login view', async () => {
  vi.spyOn(api, 'listInvites').mockReturnValue(mockTask(mockListResponse));
  vi.spyOn(api, 'adminLogout').mockReturnValue(mockTask(mockLoginResponse));

  render(() => <AdminPanel lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada')).toBeInTheDocument();
  });

  fireEvent.click(screen.getByText('Log out'));

  await waitFor(() => {
    expect(screen.getByText('Log in')).toBeInTheDocument();
  });
});
