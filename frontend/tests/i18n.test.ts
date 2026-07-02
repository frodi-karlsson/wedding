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
  const dict = is as Record<string, string | undefined>;
  const original = dict[key];

  delete dict[key];

  try {
    const result = translate(key, 'is');

    expect(result).toBe('Thank you!');
  } finally {
    dict[key] = original;
  }
});

test('should fall back to the key when it is missing in all locales', () => {
  const result = translate('nonexistent_key_xyz', 'en');

  expect(result).toBe('nonexistent_key_xyz');
});

test('should select the singular variant for count 1', () => {
  const result = translate('rsvp_intro', 'en', 1);

  expect(result).toContain('{max} guest{min_clause}');
});

test('should select the plural variant for count 2', () => {
  const result = translate('rsvp_intro', 'en', 2);

  expect(result).toContain('{max} guests{min_clause}');
});

test('should apply language-specific plural rules (Icelandic dative sg vs pl)', () => {
  // Icelandic plural category "one" applies to 1, 21, 31… (singular agreement),
  // "other" to everything else — Intl.PluralRules encodes this.
  expect(translate('rsvp_intro', 'is', 1)).toContain('gesti{min_clause}');
  expect(translate('rsvp_intro', 'is', 21)).toContain('gesti{min_clause}');
  expect(translate('rsvp_intro', 'is', 2)).toContain('gestum{min_clause}');
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
