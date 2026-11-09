import { test, expect } from 'vitest';
import * as backendLangs from '../src/scripts/langs.gen';
import { LOCALE_CODES } from '../src/scripts/i18n';

test('should serve exactly the languages the backend accepts', () => {
  const backend = Object.values(backendLangs).toSorted();

  expect(backend).toEqual(LOCALE_CODES.toSorted());
});
