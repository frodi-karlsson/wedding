import { test, expect } from 'vitest';
import type { InviteResponse, GuestResponse } from '../src/scripts/types.gen';
import {
  createRsvpState,
  addGuest,
  removeGuest,
  readdCoPrimary,
  updateGuest,
  canSubmit,
  isGroupInvite,
} from '../src/scripts/rsvp.service';

function mockInvite(overrides: Partial<InviteResponse> = {}): InviteResponse {
  return { id: '1', name: 'Ada', min_plus: 0, max_plus: 2, submitted: false, message: '', ...overrides };
}

function mockGuest(overrides: Partial<GuestResponse> = {}): GuestResponse {
  return { id: 1, name: 'Ada', dietary_preference: '', alcohol_free: false, is_primary: true, co_primary: false, ...overrides };
}

test('should create state with the primary guest first and status ready', () => {
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
    co_primary: false,
  });
  expect(state.guests[1]).toEqual({
    name: 'Guest',
    dietary_preference: '',
    alcohol_free: false,
    is_primary: false,
    co_primary: false,
  });
});

test('should create state mapping existing guest data into GuestInput', () => {
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
    co_primary: false,
  });
});

test('should add a non-primary empty guest row when under max_plus', () => {
  const state = createRsvpState(mockInvite({ max_plus: 1 }), [mockGuest()], 'en');

  const next = addGuest(state);

  expect(next.guests).toHaveLength(2);
  expect(next.guests[1]).toEqual({
    name: '',
    dietary_preference: '',
    alcohol_free: false,
    is_primary: false,
    co_primary: false,
  });
});

test('should not add a guest when the non-primary count is already at max_plus', () => {
  const state = createRsvpState(mockInvite({ max_plus: 0 }), [mockGuest()], 'en');

  const next = addGuest(state);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

test('should remove a non-primary guest row', () => {
  const state = createRsvpState(
    mockInvite({ max_plus: 1 }),
    [mockGuest(), mockGuest({ id: 2, name: 'Plus', is_primary: false })],
    'en',
  );

  const next = removeGuest(state, 1);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

test('should not remove the primary guest', () => {
  const state = createRsvpState(mockInvite(), [mockGuest()], 'en');

  const next = removeGuest(state, 0);

  expect(next.guests).toHaveLength(1);
  expect(next.guests[0].is_primary).toBe(true);
});

test('should update a guest field immutably', () => {
  const state = createRsvpState(mockInvite(), [mockGuest()], 'en');

  const next = updateGuest(state, 0, { name: 'Ada Updated', dietary_preference: 'vegetarian' });

  expect(next.guests[0].name).toBe('Ada Updated');
  expect(next.guests[0].dietary_preference).toBe('vegetarian');
  expect(state.guests[0].name).toBe('Ada');
});

test('should not allow submit when the plus count is below min_plus', () => {
  const state = createRsvpState(mockInvite({ min_plus: 1, max_plus: 2 }), [mockGuest()], 'en');

  const result = canSubmit(state);

  expect(result).toBe(false);
});

test('should allow submit when the plus count is at min_plus and all names are non-empty', () => {
  const state = createRsvpState(
    mockInvite({ min_plus: 1, max_plus: 2 }),
    [mockGuest({ name: 'Ada' }), mockGuest({ id: 2, name: 'Bob', is_primary: false })],
    'en',
  );

  const result = canSubmit(state);

  expect(result).toBe(true);
});

test('should not allow submit when any name is empty or only whitespace', () => {
  const state = createRsvpState(
    mockInvite({ min_plus: 1, max_plus: 2 }),
    [mockGuest({ name: 'Ada' }), mockGuest({ id: 2, name: '   ', is_primary: false })],
    'en',
  );

  const result = canSubmit(state);

  expect(result).toBe(false);
});

// --- Group (co-primary) invites ---

function groupState() {
  return createRsvpState(
    mockInvite({ name: 'Alice & Bob', min_plus: 0, max_plus: 0 }),
    [
      mockGuest({ id: 1, name: 'Alice', is_primary: false, co_primary: true }),
      mockGuest({ id: 2, name: 'Bob', is_primary: false, co_primary: true }),
    ],
    'en',
  );
}

test('isGroupInvite is true when any guest is a co-primary', () => {
  expect(isGroupInvite(groupState())).toBe(true);
  expect(isGroupInvite(createRsvpState(mockInvite(), [mockGuest()], 'en'))).toBe(false);
});

test('group invites do not add extra guests', () => {
  const next = addGuest(groupState());
  expect(next.guests).toHaveLength(2);
});

test('group invites allow removing a co-primary but keep at least one', () => {
  const afterOne = removeGuest(groupState(), 1);
  expect(afterOne.guests).toHaveLength(1);

  const afterTwo = removeGuest(afterOne, 0);
  expect(afterTwo.guests).toHaveLength(1); // last co-primary is retained
});

test('group invites can submit with co-primaries and no min/max constraint', () => {
  expect(canSubmit(groupState())).toBe(true);
});

test('removed co-primaries are remembered and can be re-added with their name', () => {
  const removed = removeGuest(groupState(), 1); // remove Bob
  expect(removed.guests).toHaveLength(1);
  expect(removed.removedCoPrimaries).toHaveLength(1);
  expect(removed.removedCoPrimaries?.[0].name).toBe('Bob');

  const readded = readdCoPrimary(removed, 0);
  expect(readded.guests).toHaveLength(2);
  expect(readded.guests.some((g) => g.name === 'Bob' && g.co_primary)).toBe(true);
  expect(readded.removedCoPrimaries).toHaveLength(0);
});
