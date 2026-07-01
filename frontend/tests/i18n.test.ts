import { test, expect } from 'vitest';
import { translate, langFromPath, type Lang } from '../src/scripts/i18n';
import is from '../src/locales/is.json';

test('should return the translated string for a known key in the requested language', () => {
  const result = translate('thank_you', 'en');

  expect(result).toBe('Thank you!');
});

test('should fall back to English when a key is missing in a non-default locale', () => {
  const langs: Lang[] = ['is', 'de', 'sv'];

  langs.forEach((l) => {
    const result = translate('thank_you', l);

    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });
});

test('should fall back to the English value when a key is missing from a specific locale', () => {
  const key = 'thank_you';
  const original = is[key];

  delete is[key];

  try {
    const result = translate(key, 'is');

    expect(result).toBe('Thank you!');
  } finally {
    is[key] = original;
  }
});

test('should fall back to the key when it is missing in all locales', () => {
  const result = translate('nonexistent_key_xyz', 'en');

  expect(result).toBe('nonexistent_key_xyz');
});

test('should return en for the root path', () => {
  const result = langFromPath('/');

  expect(result).toBe('en');
});

test('should return is for the Icelandic language prefix', () => {
  const result = langFromPath('/is');

  expect(result).toBe('is');
});

test('should return de for the German language prefix', () => {
  const result = langFromPath('/de');

  expect(result).toBe('de');
});

test('should return sv for the Swedish language prefix', () => {
  const result = langFromPath('/sv');

  expect(result).toBe('sv');
});

test('should return the language prefix for a deeper path', () => {
  const result = langFromPath('/is/admin');

  expect(result).toBe('is');
});

test('should return en for an unknown language prefix', () => {
  const result = langFromPath('/xyz');

  expect(result).toBe('en');
});
