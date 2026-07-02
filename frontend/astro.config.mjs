import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';

export default defineConfig({
  output: 'static',
  integrations: [solidJs()],
  i18n: {
    locales: ['en', 'is', 'de', 'sv'],
    defaultLocale: 'en',
    prefixDefaultLocale: false,
  },
  vite: {
    css: { preprocessorOptions: { scss: {} } },
  },
});
