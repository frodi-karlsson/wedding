import { test, expect } from 'vitest';
import { translate, type Lang } from '../src/scripts/i18n';

const langs: Lang[] = ['en', 'is', 'de', 'sv'];
const keys = ['hero_eyebrow', 'hero_date', 'hero_location', 'hero_cta'];

test('every locale resolves the hero keys to a non-empty, non-fallback string', () => {
  for (const lang of langs) {
    for (const key of keys) {
      const value = translate(key, lang);
      expect(value, `${lang}.${key}`).not.toBe(key); // not the raw-key fallback
      expect(value.length, `${lang}.${key}`).toBeGreaterThan(0);
    }
  }
});
