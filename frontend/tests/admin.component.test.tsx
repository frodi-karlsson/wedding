import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { AdminPanel } from '../src/islands/AdminPanel';
import { api } from '../src/scripts/api';
import type {
  ListInvitesResponse,
  StatusResponse,
  InviteWithGuestsResponse,
} from '../src/scripts/types.gen';
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
    { id: '1', name: 'Ada', min_plus: 0, max_plus: 2, submitted: false, message: '' },
    { id: '2', name: 'Bob', min_plus: 1, max_plus: 3, submitted: true, message: '' },
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

test('should show the submission view when clicking view', async () => {
  vi.spyOn(api, 'listInvites').mockReturnValue(mockTask(mockListResponse));
  vi.spyOn(api, 'getAdminInvite').mockReturnValue(
    mockTask<InviteWithGuestsResponse>({
      invite: { id: '2', name: 'Bob', min_plus: 1, max_plus: 3, submitted: true, message: 'Cannot wait!' },
      guests: [
        { id: 1, name: 'Bob', dietary_preference: 'Vegetarian', alcohol_free: true, is_primary: true },
        { id: 2, name: 'Cara', dietary_preference: '', alcohol_free: false, is_primary: false },
      ],
    }),
  );

  render(() => <AdminPanel lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Bob')).toBeInTheDocument();
  });

  // one "View" button per invite row; click Bob's (second row)
  fireEvent.click(screen.getAllByText('View')[1]);

  await waitFor(() => {
    expect(screen.getByText('Cannot wait!')).toBeInTheDocument();
    expect(screen.getByText('Cara')).toBeInTheDocument();
  });
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
