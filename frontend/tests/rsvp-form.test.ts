import { it, expect } from 'vitest';
import type { Invite, Guest } from '../src/scripts/types';
import {
  createRsvpState,
  addGuest,
  removeGuest,
  updateGuest,
  canSubmit,
  guestsToInput,
} from '../src/scripts/rsvp-form';

function mockInvite(overrides: Partial<Invite> = {}): Invite {
  return { id: 1, name: 'Ada', min_plus: 0, max_plus: 2, submitted: false, ...overrides };
}

function mockGuest(overrides: Partial<Guest> = {}): Guest {
  return { id: 1, name: 'Ada', dietary_preference: '', alcohol_free: false, is_primary: true, ...overrides };
}

it('should create state with the primary guest first and status ready', () => {
  const invite = mockInvite({ max_plus: 2 });
  const primary = mockGuest({ name: 'Ada', is_primary: true });
  const plus = mockGuest({ id: 2, name: 'Guest', is_primary: false });

  const state = createRsvpState(invite, [plus, primary], 'en');

  expect(state.status).toBe('ready');
  expect(state.lang).toBe('en');
  expect(state.invite).toBe(invite);
  expect(state.guests).toHaveLength(2);
  expect(state.guests[0]).toEqual({
    name: 'Ada',
    dietary_preference: '',
    alcohol_free: false,
    is_primary: true,
  });
  expect(state.guests[1]).toEqual({
    name: 'Guest',
    dietary_preference: '',
    alcohol_free: false,
    is_primary: false,
  });
});

it('should create state mapping existing guest data into GuestInput', () => {
  const invite = mockInvite({ max_plus: 1 });
  const guest = mockGuest({
    name: 'Bob',
    dietary_preference: 'vegan',
    alcohol_free: true,
    is_primary: true,
  });

  const state = createRsvpState(invite, [guest], 'is');

  expect(state.guests[0]).toEqual({
    name: 'Bob',
    dietary_preference: 'vegan',
    alcohol_free: true,
    is_primary: true,
  });
});

it('should add a non-primary empty guest row when under max_plus', () => {
  const state = createRsvpState(mockInvite({ max_plus: 1 }), [mockGuest()], 'en');

  const next = addGuest(state);

  expect(next.guests).toHaveLength(2);
  expect(next.guests[1]).toEqual({
    name: '',
    dietary_preference: '',
    alcohol_free: false,
    is_primary: false,
  });
});

it('should not add a guest when the non-primary count is already at max_plus', () => {
  const state = createRsvpState(mockInvite({ max_plus: 0 }), [mockGuest()], 'en');

  const next = addGuest(state);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

it('should remove a non-primary guest row', () => {
  const state = createRsvpState(
    mockInvite({ max_plus: 1 }),
    [mockGuest(), mockGuest({ id: 2, name: 'Plus', is_primary: false })],
    'en',
  );

  const next = removeGuest(state, 1);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

it('should not remove the primary guest', () => {
  const state = createRsvpState(mockInvite(), [mockGuest()], 'en');

  const next = removeGuest(state, 0);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

it('should update a guest field immutably', () => {
  const state = createRsvpState(mockInvite(), [mockGuest()], 'en');

  const next = updateGuest(state, 0, { name: 'Ada Updated', dietary_preference: 'vegetarian' });

  expect(next.guests[0].name).toBe('Ada Updated');
  expect(next.guests[0].dietary_preference).toBe('vegetarian');
  expect(state.guests[0].name).toBe('Ada');
});

it('should not allow submit when the plus count is below min_plus', () => {
  const state = createRsvpState(mockInvite({ min_plus: 1, max_plus: 2 }), [mockGuest()], 'en');

  const result = canSubmit(state);

  expect(result).toBe(false);
});

it('should allow submit when the plus count is at min_plus and all names are non-empty', () => {
  const state = createRsvpState(
    mockInvite({ min_plus: 1, max_plus: 2 }),
    [mockGuest({ name: 'Ada' }), mockGuest({ id: 2, name: 'Bob', is_primary: false })],
    'en',
  );

  const result = canSubmit(state);

  expect(result).toBe(true);
});

it('should not allow submit when any name is empty or only whitespace', () => {
  const state = createRsvpState(
    mockInvite({ min_plus: 1, max_plus: 2 }),
    [mockGuest({ name: 'Ada' }), mockGuest({ id: 2, name: '   ', is_primary: false })],
    'en',
  );

  const result = canSubmit(state);

  expect(result).toBe(false);
});

it('should return the guests array from guestsToInput', () => {
  const state = createRsvpState(mockInvite(), [mockGuest()], 'en');

  const result = guestsToInput(state);

  expect(result).toBe(state.guests);
});
