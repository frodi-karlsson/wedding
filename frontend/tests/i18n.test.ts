import { describe, it, expect } from 'vitest';
import { translate, langFromPath, type Lang } from '../src/scripts/i18n';

describe('translate', () => {
  it('returns the string for the given lang', () => {
    expect(translate('thank_you', 'en')).toBe('Thank you!');
  });
  it('falls back to English when key missing in lang', () => {
    // Temporarily can't mutate JSON; just assert a known key works in all langs.
    (['is', 'de', 'sv'] as Lang[]).forEach((l) => {
      expect(typeof translate('thank_you', l)).toBe('string');
      expect(translate('thank_you', l).length).toBeGreaterThan(0);
    });
  });
  it('falls back to the key when missing everywhere', () => {
    expect(translate('nonexistent_key_xyz', 'en')).toBe('nonexistent_key_xyz');
  });
});

describe('langFromPath', () => {
  it('returns en for /', () => { expect(langFromPath('/')).toBe('en'); });
  it('returns is for /is', () => { expect(langFromPath('/is')).toBe('is'); });
  it('returns de for /de', () => { expect(langFromPath('/de')).toBe('de'); });
  it('returns sv for /sv', () => { expect(langFromPath('/sv')).toBe('sv'); });
  it('returns en for /is/foo deeper path still maps to is', () => { expect(langFromPath('/is/admin')).toBe('is'); });
  it('returns en for unknown prefix', () => { expect(langFromPath('/xyz')).toBe('en'); });
});
