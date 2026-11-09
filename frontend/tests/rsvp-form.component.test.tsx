import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { RsvpForm } from '../src/islands/RsvpForm';
import { api } from '../src/scripts/api';
import type { InviteWithGuestsResponse } from '../src/scripts/types.gen';
import { mockTask, mockFailingTask, settleTasks } from './tasks';

const mockInviteResponse: InviteWithGuestsResponse = {
  invite: { id: 'K3M7X', name: 'Ada & Guest', min_plus: 0, max_plus: 2, submitted: false, message: '' },
  guests: [
    { id: 1, name: 'Ada', dietary_preference: 'vegetarian', alcohol_free: false, is_primary: true, co_primary: false },
  ],
};

beforeEach(() => {
  vi.stubGlobal('location', {
    origin: 'https://example.com',
    pathname: '/',
    search: '?id=K3M7X',
    hash: '',
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

test('should render a loading state initially', () => {
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);

  expect(screen.getByText('Loading…')).toBeInTheDocument();
});

test('should render the form with the invite name after loading', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);
  await settleTasks();

  expect(screen.getByText('Ada & Guest')).toBeInTheDocument();
});

test('should add a guest row when clicking add guest', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(mockTask(mockInviteResponse));

  render(() => <RsvpForm lang="en" />);
  await settleTasks();

  fireEvent.click(screen.getByText('+ Add guest'));

  expect(screen.getAllByRole('group')).toHaveLength(2);
});

test('should disable submit when a guest name is empty', async () => {
  const emptyNameResponse: InviteWithGuestsResponse = {
    invite: { id: 'K3M7X', name: 'Ada & Guest', min_plus: 0, max_plus: 2, submitted: false, message: '' },
    guests: [
      { id: 1, name: '', dietary_preference: '', alcohol_free: false, is_primary: true, co_primary: false },
    ],
  };
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(mockTask(emptyNameResponse));

  render(() => <RsvpForm lang="en" />);
  await settleTasks();

  expect(screen.getByRole('button', { name: /Submit RSVP/i })).toBeDisabled();
});

test('should show an error message when the invite fails to load', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(mockFailingTask(new Error('Network error')));

  render(() => <RsvpForm lang="en" />);
  await settleTasks();

  expect(screen.getByText(/Could not load your invitation/i)).toBeInTheDocument();
});

test('should render the alcohol-free box ticked for a guest the admin preset', async () => {
  vi.spyOn(api, 'getInvite').mockReturnValueOnce(
    mockTask<InviteWithGuestsResponse>({
      invite: { id: 'K3M7X', name: 'Ada', min_plus: 0, max_plus: 1, submitted: false, message: '' },
      guests: [
        { id: 1, name: 'Ada', dietary_preference: '', alcohol_free: false, is_primary: true, co_primary: false },
        { id: 2, name: 'Kid', dietary_preference: '', alcohol_free: true, is_primary: false, co_primary: false },
      ],
    }),
  );

  render(() => <RsvpForm lang="en" />);
  await settleTasks();

  const boxes = screen.getAllByRole('checkbox', { name: 'Alcohol-free' }) as HTMLInputElement[];
  expect(boxes.map((b) => b.checked)).toEqual([false, true]);
});
