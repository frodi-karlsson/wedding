import { test, expect, vi } from 'vitest';
import { localeFromPath, buildLocaleHref, hasInviteId, scrollToFragment } from '../src/scripts/nav';

test('should return en for the root path', () => {
  expect(localeFromPath('/')).toBe('en');
});

test('should return en for a path with no locale prefix', () => {
  expect(localeFromPath('/admin')).toBe('en');
});

test('should return is for the Icelandic language prefix', () => {
  expect(localeFromPath('/is')).toBe('is');
});

test('should return de for the German language prefix', () => {
  expect(localeFromPath('/de/admin')).toBe('de');
});

test('should return sv for the Swedish language prefix', () => {
  expect(localeFromPath('/sv')).toBe('sv');
});

test('should return en for an unknown language prefix', () => {
  expect(localeFromPath('/xyz')).toBe('en');
});

test('should preserve query params and fragment when switching from en to is', () => {
  const url = { pathname: '/', search: '?id=abc123', hash: '#rsvp' };

  expect(buildLocaleHref(url, 'is')).toBe('/is?id=abc123#rsvp');
});

test('should preserve query params and fragment when switching from is to en', () => {
  const url = { pathname: '/is', search: '?id=abc123', hash: '#rsvp' };

  expect(buildLocaleHref(url, 'en')).toBe('/?id=abc123#rsvp');
});

test('should preserve query params and fragment when switching from is to de', () => {
  const url = { pathname: '/is', search: '?id=abc123', hash: '#rsvp' };

  expect(buildLocaleHref(url, 'de')).toBe('/de?id=abc123#rsvp');
});

test('should handle switching from a deeper path', () => {
  const url = { pathname: '/is/admin', search: '?id=abc123', hash: '' };

  expect(buildLocaleHref(url, 'en')).toBe('/admin?id=abc123');
});

test('should handle switching to the same locale', () => {
  const url = { pathname: '/is', search: '?id=abc123', hash: '#rsvp' };

  expect(buildLocaleHref(url, 'is')).toBe('/is?id=abc123#rsvp');
});

test('should handle root path with no params or fragment', () => {
  const url = { pathname: '/', search: '', hash: '' };

  expect(buildLocaleHref(url, 'en')).toBe('/');
  expect(buildLocaleHref(url, 'is')).toBe('/is');
});

test('should return true when id param is present', () => {
  expect(hasInviteId({ search: '?id=abc123' })).toBe(true);
});

test('should return true when id param is present with other params', () => {
  expect(hasInviteId({ search: '?foo=bar&id=abc123&baz=qux' })).toBe(true);
});

test('should return false when id param is absent', () => {
  expect(hasInviteId({ search: '?foo=bar' })).toBe(false);
});

test('should return false when search is empty', () => {
  expect(hasInviteId({ search: '' })).toBe(false);
});

test('should scroll to the element matching the hash', () => {
  const el = document.createElement('div');
  el.id = 'location';
  document.body.appendChild(el);
  el.scrollIntoView = vi.fn();

  scrollToFragment('#location');

  expect(el.scrollIntoView).toHaveBeenCalledTimes(1);
  expect(el.scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth' });

  document.body.removeChild(el);
});

test('should do nothing when the hash has no matching element', () => {
  expect(() => scrollToFragment('#nonexistent')).not.toThrow();
});

test('should do nothing when the hash is empty', () => {
  expect(() => scrollToFragment('')).not.toThrow();
});
