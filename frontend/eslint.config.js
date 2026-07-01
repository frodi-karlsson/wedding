import tseslint from 'typescript-eslint';
import astro from 'eslint-plugin-astro';
import vitest from 'eslint-plugin-vitest';

const windowBan = [
  'error',
  {
    name: 'window',
    message: 'Use `globalThis` instead of `window` for portability.',
  },
];

const windowMemberBan = [
  'error',
  {
    selector: 'MemberExpression[object.name="window"]',
    message: 'Use `globalThis` instead of `window` for portability.',
  },
];

export default [
  {
    ignores: ['dist/**', 'node_modules/**', '.astro/**'],
  },
  ...tseslint.configs.recommended,
  ...astro.configs.recommended,
  {
    files: ['**/*.{js,ts,astro}'],
    rules: {
      'no-restricted-globals': windowBan,
      'no-restricted-syntax': windowMemberBan,
    },
  },
  {
    files: ['src/env.d.ts'],
    rules: {
      '@typescript-eslint/triple-slash-reference': 'off',
    },
  },
  {
    files: ['**/*.test.ts', 'tests/**'],
    ...vitest.configs.recommended,
    rules: {
      ...vitest.configs.recommended.rules,
      'vitest/consistent-test-it': ['error', { fn: 'test', withinDescribe: 'test' }],
    },
  },
];
