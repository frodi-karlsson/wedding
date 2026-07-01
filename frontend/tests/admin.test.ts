import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { buildShareLink } from '../src/scripts/admin';

beforeEach(() => {
  vi.stubGlobal('location', { origin: 'https://example.com' });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('should build share link for en without a language prefix', () => {
  const result = buildShareLink(42, 'en');

  expect(result).toBe('https://example.com/?id=42');
});

test('should build share link for is with the language prefix', () => {
  const result = buildShareLink(42, 'is');

  expect(result).toBe('https://example.com/is?id=42');
});

test('should build share link for de and sv', () => {
  const de = buildShareLink(1, 'de');
  const sv = buildShareLink(2, 'sv');

  expect(de).toBe('https://example.com/de?id=1');
  expect(sv).toBe('https://example.com/sv?id=2');
});
