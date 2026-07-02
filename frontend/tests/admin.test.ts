import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { buildShareLink, createEmptyForm, formFromInvite } from '../src/scripts/admin';
import type { GuestResponse, InviteResponse } from '../src/scripts/types.gen';

beforeEach(() => {
  vi.stubGlobal('location', { origin: 'https://example.com' });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('should build share link for en without a language prefix', () => {
  const result = buildShareLink('abc123', 'en');

  expect(result).toBe('https://example.com/?id=abc123');
});

test('should build share link for is with the language prefix', () => {
  const result = buildShareLink('abc123', 'is');

  expect(result).toBe('https://example.com/is?id=abc123');
});

test('should build share link for de and sv', () => {
  const de = buildShareLink('1', 'de');
  const sv = buildShareLink('2', 'sv');

  expect(de).toBe('https://example.com/de?id=1');
  expect(sv).toBe('https://example.com/sv?id=2');
});

test('should create an empty form with the given language', () => {
  const result = createEmptyForm('de');

  expect(result).toEqual({
    name: '',
    min_plus: 0,
    max_plus: 1,
    guest_names: [''],
    link_lang: 'de',
  });
});

test('should derive the primary guest name from the invite when no guests exist', () => {
  const invite: InviteResponse = { id: '5', name: 'Ada', min_plus: 0, max_plus: 2, submitted: false };

  const result = formFromInvite(invite, [], 'en');

  expect(result.guest_names).toEqual(['Ada']);
  expect(result.link_lang).toBe('en');
});

test('should sort guest names with the primary guest first', () => {
  const invite: InviteResponse = { id: '6', name: 'Ada', min_plus: 1, max_plus: 3, submitted: false };
  const guests: GuestResponse[] = [
    { id: 2, name: 'Bob', dietary_preference: '', alcohol_free: false, is_primary: false },
    { id: 1, name: 'Ada', dietary_preference: '', alcohol_free: false, is_primary: true },
    { id: 3, name: 'Cid', dietary_preference: '', alcohol_free: false, is_primary: false },
  ];

  const result = formFromInvite(invite, guests, 'is');

  expect(result.guest_names).toEqual(['Ada', 'Bob', 'Cid']);
  expect(result.link_lang).toBe('is');
});
