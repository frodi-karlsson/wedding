import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { RsvpForm } from '../src/islands/RsvpForm';
import { api } from '../src/scripts/api';
import type { InviteWithGuestsResponse } from '../src/scripts/types.gen';
import type { Task } from 'anabranch';
import type { HttpError } from '@anabranch/web-client';

function mockTask<T>(value: T): Task<T, HttpError> {
  return { run: vi.fn().mockResolvedValue(value) } as unknown as Task<T, HttpError>;
}

function mockFailingTask<T>(error: Error): Task<T, HttpError> {
  return { run: vi.fn().mockRejectedValue(error) } as unknown as Task<T, HttpError>;
}

const mockInviteResponse: InviteWithGuestsResponse = {
  invite: { id: 'abc123', name: 'Ada & Guest', min_plus: 0, max_plus: 2, submitted: false, message: '' },
  guests: [
    { id: 1, name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true, co_primary: false },
  ],
};

beforeEach(() => {
  vi.stubGlobal('location', {
    origin: 'https://example.com',
    pathname: '/',
    search: '?id=abc123',
    hash: '',
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

test('should render a loading state initially', () => {
  vi.spyOn(api, 'getInvite').mockReturnValue(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);

  expect(screen.getByText('Loading…')).toBeInTheDocument();
});

test('should render the form with the invite name after loading', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValue(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada & Guest')).toBeInTheDocument();
  });
});

test('should add a guest row when clicking add guest', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValue(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada & Guest')).toBeInTheDocument();
  });

  const addButton = screen.getByText('+ Add guest');
  fireEvent.click(addButton);

  expect(screen.getAllByRole('group')).toHaveLength(2);
});

test('should disable submit when a guest name is empty', async () => {
  const emptyNameResponse: InviteWithGuestsResponse = {
    invite: { id: 'abc123', name: 'Ada & Guest', min_plus: 0, max_plus: 2, submitted: false, message: '' },
    guests: [
      { id: 1, name: '', dietary_preference: '', alcohol_free: false, is_primary: true, co_primary: false },
    ],
  };
  vi.spyOn(api, 'getInvite').mockReturnValue(mockTask(emptyNameResponse));

  render(() => <RsvpForm lang="en" />);

  await waitFor(() => {
    expect(screen.getByText('Ada & Guest')).toBeInTheDocument();
  });

  const submitButton = screen.getByRole('button', { name: /Submit RSVP/i });
  expect(submitButton).toBeDisabled();
});

test('should show an error message when the invite fails to load', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValue(mockFailingTask(new Error('Network error')));

  render(() => <RsvpForm lang="en" />);

  await waitFor(() => {
    expect(screen.getByText(/Could not load your invitation/i)).toBeInTheDocument();
  });
});
