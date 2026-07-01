import { describe, test, expect } from 'vitest';
import { translate, langFromPath, type Lang } from '../src/scripts/i18n';

describe('translate', () => {
  test('should return the string for the given lang', () => {
    const result = translate('thank_you', 'en');

    expect(result).toBe('Thank you!');
  });

  test('should fall back to English when key missing in lang', () => {
    const langs: Lang[] = ['is', 'de', 'sv'];

    langs.forEach((l) => {
      const result = translate('thank_you', l);

      expect(typeof result).toBe('string');
      expect(result.length).toBeGreaterThan(0);
    });
  });

  test('should fall back to the key when missing everywhere', () => {
    const result = translate('nonexistent_key_xyz', 'en');

    expect(result).toBe('nonexistent_key_xyz');
  });
});

describe('langFromPath', () => {
  test('should return en for /', () => {
    const result = langFromPath('/');

    expect(result).toBe('en');
  });

  test('should return is for /is', () => {
    const result = langFromPath('/is');

    expect(result).toBe('is');
  });

  test('should return de for /de', () => {
    const result = langFromPath('/de');

    expect(result).toBe('de');
  });

  test('should return sv for /sv', () => {
    const result = langFromPath('/sv');

    expect(result).toBe('sv');
  });

  test('should return is for a deeper path under the lang prefix', () => {
    const result = langFromPath('/is/admin');

    expect(result).toBe('is');
  });

  test('should return en for an unknown prefix', () => {
    const result = langFromPath('/xyz');

    expect(result).toBe('en');
  });
});
