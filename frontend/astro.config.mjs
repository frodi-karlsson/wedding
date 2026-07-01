import { defineConfig } from 'astro/config';

export default defineConfig({
  output: 'static',
  i18n: {
    locales: ['en', 'is', 'de', 'sv'],
    defaultLocale: 'en',
    prefixDefaultLocale: false,
  },
  vite: {
    css: { preprocessorOptions: { scss: {} } },
  },
});
