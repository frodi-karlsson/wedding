import { it, expect, vi, beforeEach, afterEach } from 'vitest';
import { buildShareLink } from '../src/scripts/admin';

beforeEach(() => {
  vi.stubGlobal('window', { location: { origin: 'https://example.com' } });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

it('should build share link for en without a language prefix', () => {
  expect(buildShareLink(42, 'en')).toBe('https://example.com/?id=42');
});

it('should build share link for is with the language prefix', () => {
  expect(buildShareLink(42, 'is')).toBe('https://example.com/is?id=42');
});

it('should build share link for de and sv', () => {
  expect(buildShareLink(1, 'de')).toBe('https://example.com/de?id=1');
  expect(buildShareLink(2, 'sv')).toBe('https://example.com/sv?id=2');
});
