import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';

export default defineConfig({
  output: 'static',
  integrations: [solidJs()],
  i18n: {
    // defaultLocale must stay in sync with DEFAULT_LOCALE in src/scripts/i18n.ts.
    locales: ['en', 'is', 'de', 'sv'],
    defaultLocale: 'sv',
    prefixDefaultLocale: false,
  },
  vite: {
    css: { preprocessorOptions: { scss: {} } },
  },
});
