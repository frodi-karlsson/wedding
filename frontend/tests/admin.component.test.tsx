import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { AdminPanel } from '../src/islands/AdminPanel';
import { AdminForm } from '../src/islands/components/AdminForm';
import { api } from '../src/scripts/api';
import { formFromInvite } from '../src/scripts/admin.service';
import type {
  ListInvitesResponse,
  StatusResponse,
  InviteWithGuestsResponse,
} from '../src/scripts/types.gen';
import { mockTask, mockFailingTask, settleTasks } from './tasks';

const mockListResponse: ListInvitesResponse = {
  invites: [
    { id: 'H7K2Q', name: 'Ada', min_plus: 0, max_plus: 2, submitted: false, message: '' },
    { id: 'X4M9B', name: 'Bob', min_plus: 1, max_plus: 3, submitted: true, message: '' },
  ],
};

const mockLogoutResponse: StatusResponse = { status: 'ok' };

const familyInvite: InviteWithGuestsResponse = {
  invite: { id: 'X4M9B', name: 'Bob', min_plus: 1, max_plus: 3, submitted: false, message: '' },
  guests: [
    { id: 1, name: 'Bob', dietary_preference: '', alcohol_free: false, is_primary: true, co_primary: false },
    { id: 2, name: 'Kid', dietary_preference: '', alcohol_free: true, is_primary: false, co_primary: false },
    { id: 3, name: 'Cara', dietary_preference: 'vegan', alcohol_free: false, is_primary: false, co_primary: false },
  ],
};

beforeEach(() => {
  vi.stubGlobal('location', {
    origin: 'https://example.com',
    pathname: '/admin',
    search: '',
    hash: '',
    assign: vi.fn(),
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

async function renderDashboard(): Promise<void> {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockTask({ role: 'admin' }));
  vi.spyOn(api, 'listInvites').mockReturnValue(mockTask(mockListResponse));
  render(() => <AdminPanel lang="en" />);
  await settleTasks();
}

function renderFamilyEditForm(onSubmit = vi.fn()): typeof onSubmit {
  render(() => (
    <AdminForm
      lang="en"
      form={formFromInvite(familyInvite.invite, familyInvite.guests, 'en')}
      isCreate={false}
      onSubmit={onSubmit}
      onCancel={vi.fn()}
    />
  ));
  return onSubmit;
}

function alcoholFreeBoxes(): HTMLInputElement[] {
  return screen.getAllByRole('checkbox', { name: 'Alcohol-free' }) as HTMLInputElement[];
}

function guestNameInputs(): string[] {
  return [...document.querySelectorAll<HTMLInputElement>('.guest-name')].map((i) => i.value);
}

function submitAdminForm(): void {
  fireEvent.submit(document.querySelector('form.admin-form') as HTMLFormElement);
}

test('should render the dashboard when listInvites succeeds', async () => {
  await renderDashboard();

  expect(screen.getByText('Ada')).toBeInTheDocument();
  expect(screen.getByText('Bob')).toBeInTheDocument();
});

test('should render invite ids in full so they can be read off the screen', async () => {
  await renderDashboard();

  expect(screen.getByText('H7K2Q')).toBeInTheDocument();
  expect(screen.getByText('X4M9B')).toBeInTheDocument();
});

test('should render the login view when there is no session', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockFailingTask(new Error('Unauthorized')));
  render(() => <AdminPanel lang="en" />);

  await settleTasks();

  expect(screen.getByText('Log in')).toBeInTheDocument();
});

test('should send a translator session to the speeches page', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockTask({ role: 'translator' }));
  render(() => <AdminPanel lang="en" />);

  await settleTasks();

  expect(globalThis.location.assign).toHaveBeenCalledWith('/en/speeches');
});

test('should show the dashboard after logging in as admin', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockFailingTask(new Error('Unauthorized')));
  vi.spyOn(api, 'login').mockReturnValueOnce(mockTask({ role: 'admin' }));
  vi.spyOn(api, 'listInvites').mockReturnValueOnce(mockTask(mockListResponse));
  render(() => <AdminPanel lang="en" />);
  await settleTasks();
  fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'secret' } });

  fireEvent.submit(document.querySelector('form.admin-login') as HTMLFormElement);
  await settleTasks();

  expect(api.login).toHaveBeenCalledWith('secret');
  expect(screen.getByText('Ada')).toBeInTheDocument();
});

test('should send a translator to the speeches page after logging in', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockFailingTask(new Error('Unauthorized')));
  vi.spyOn(api, 'login').mockReturnValueOnce(mockTask({ role: 'translator' }));
  render(() => <AdminPanel lang="en" />);
  await settleTasks();
  fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'secret' } });

  fireEvent.submit(document.querySelector('form.admin-login') as HTMLFormElement);
  await settleTasks();

  expect(globalThis.location.assign).toHaveBeenCalledWith('/en/speeches');
});

test('should show the create form when clicking new invite', async () => {
  await renderDashboard();

  fireEvent.click(screen.getByText('+ New invite'));

  expect(screen.getByText('Create invite')).toBeInTheDocument();
});

test('should show the submission view when clicking view', async () => {
  vi.spyOn(api, 'getAdminInvite').mockReturnValueOnce(
    mockTask<InviteWithGuestsResponse>({
      invite: { id: 'X4M9B', name: 'Bob', min_plus: 1, max_plus: 3, submitted: true, message: 'Cannot wait!' },
      guests: [
        { id: 1, name: 'Bob', dietary_preference: 'Vegetarian', alcohol_free: true, is_primary: true, co_primary: false },
        { id: 2, name: 'Cara', dietary_preference: '', alcohol_free: false, is_primary: false, co_primary: false },
      ],
    }),
  );
  await renderDashboard();

  // one "View" button per invite row; click Bob's (second row)
  fireEvent.click(screen.getAllByText('View')[1]);
  await settleTasks();

  expect(screen.getByText('Cannot wait!')).toBeInTheDocument();
  expect(screen.getByText('Cara')).toBeInTheDocument();
});

test('should log out and return to login view', async () => {
  vi.spyOn(api, 'logout').mockReturnValueOnce(mockTask(mockLogoutResponse));
  await renderDashboard();

  fireEvent.click(screen.getByText('Log out'));
  await settleTasks();

  expect(screen.getByText('Log in')).toBeInTheDocument();
});

test('should show the get-invite screen with the shareable link when clicking Get invite', async () => {
  await renderDashboard();

  // Ada is the first row.
  fireEvent.click(screen.getAllByText('Get invite')[0]);

  expect(screen.getByDisplayValue('https://example.com/en?id=H7K2Q')).toBeInTheDocument();
});

test('should send preset alcohol-free flags when creating an invite', async () => {
  vi.spyOn(api, 'createInvite').mockReturnValueOnce(
    mockTask<InviteWithGuestsResponse>({
      invite: { id: 'K3M7X', name: 'Ada', min_plus: 0, max_plus: 1, submitted: false, message: '' },
      guests: [],
    }),
  );
  await renderDashboard();
  fireEvent.click(screen.getByText('+ New invite'));
  fireEvent.input(screen.getByRole('textbox', { name: 'Name (primary invitee)' }), { target: { value: 'Ada' } });
  fireEvent.click(screen.getByText('+ Add name'));
  fireEvent.input(document.querySelector('.guest-name') as HTMLInputElement, { target: { value: 'Kid' } });

  fireEvent.click(alcoholFreeBoxes()[1]);
  submitAdminForm();
  await settleTasks();

  expect(api.createInvite).toHaveBeenCalledWith({
    name: 'Ada',
    min_plus: 0,
    max_plus: 1,
    guests: [
      { name: 'Ada', alcohol_free: false },
      { name: 'Kid', alcohol_free: true },
    ],
    group: false,
  });
});

test('should show preset alcohol-free flags ticked when editing an invite', () => {
  renderFamilyEditForm();

  const checked = alcoholFreeBoxes().map((b) => b.checked);

  expect(checked).toEqual([false, true, false]);
});

test('should save an edited invite with each guest\'s id and alcohol-free flag', () => {
  const onSubmit = renderFamilyEditForm();

  submitAdminForm();

  expect(onSubmit).toHaveBeenCalledWith({
    id: 'X4M9B',
    name: 'Bob',
    min_plus: 1,
    max_plus: 3,
    guests: [
      { id: 1, name: 'Bob', alcohol_free: false },
      { id: 2, name: 'Kid', alcohol_free: true },
      { id: 3, name: 'Cara', alcohol_free: false },
    ],
    link_lang: 'en',
    group: false,
  });
});

test('should save the primary invitee as alcohol-free when its box is ticked', () => {
  const onSubmit = renderFamilyEditForm();

  fireEvent.click(alcoholFreeBoxes()[0]);
  submitAdminForm();

  expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    guests: [
      { id: 1, name: 'Bob', alcohol_free: true },
      { id: 2, name: 'Kid', alcohol_free: true },
      { id: 3, name: 'Cara', alcohol_free: false },
    ],
  }));
});

test('should keep each remaining guest\'s id and flag when removing a middle guest', () => {
  const onSubmit = renderFamilyEditForm();

  fireEvent.click(screen.getAllByText('Remove')[0]);
  submitAdminForm();

  expect(guestNameInputs()).toEqual(['Cara']);
  expect(alcoholFreeBoxes().map((b) => b.checked)).toEqual([false, false]);
  expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    guests: [
      { id: 1, name: 'Bob', alcohol_free: false },
      { id: 3, name: 'Cara', alcohol_free: false },
    ],
  }));
});

test('should send the edited guests to updateInvite when saving from the panel', async () => {
  vi.spyOn(api, 'getAdminInvite').mockReturnValueOnce(mockTask(familyInvite));
  vi.spyOn(api, 'updateInvite').mockReturnValueOnce(mockTask(familyInvite));
  await renderDashboard();
  // Bob is the second row.
  fireEvent.click(screen.getAllByText('Edit')[1]);
  await settleTasks();

  submitAdminForm();
  await settleTasks();

  expect(api.updateInvite).toHaveBeenCalledWith('X4M9B', {
    name: 'Bob',
    min_plus: 1,
    max_plus: 3,
    guests: [
      { id: 1, name: 'Bob', alcohol_free: false },
      { id: 2, name: 'Kid', alcohol_free: true },
      { id: 3, name: 'Cara', alcohol_free: false },
    ],
    group: false,
  });
});
