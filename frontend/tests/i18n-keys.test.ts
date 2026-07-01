import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from 'vitest';

const localesDir = resolve(dirname(fileURLToPath(import.meta.url)), '../src/locales');

const locales = ['en', 'is', 'de', 'sv'] as const;

const keysByLocale: Record<string, string[]> = Object.fromEntries(
  locales.map((lang) => {
    const data = JSON.parse(readFileSync(resolve(localesDir, `${lang}.json`), 'utf8'));
    return [lang, Object.keys(data).sort()];
  }),
);

test('should have identical key sets across all locale files', () => {
  const enKeys = keysByLocale.en;

  for (const lang of locales) {
    expect(keysByLocale[lang], `locale '${lang}' key set must match 'en'`).toEqual(enKeys);
  }
});

test('should have at least one key in every locale file', () => {
  for (const lang of locales) {
    expect(keysByLocale[lang].length > 0, `locale '${lang}' has no keys`).toBe(true);
  }
});
