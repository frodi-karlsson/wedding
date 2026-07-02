/// <reference types="vitest/config" />
import { getViteConfig } from 'astro/config';

export default getViteConfig({
  resolve: { conditions: ['browser', 'development'] },
  environments: {
    ssr: {
      resolve: { conditions: ['browser', 'development'] },
    },
  },
  test: {
    environment: 'happy-dom',
    globals: true,
    isolate: false,
    setupFiles: ['./tests/setup.ts'],
    // Playwright specs live in e2e/ and are run by `pnpm test:e2e`, not Vitest.
    include: ['tests/**/*.{test,spec}.{ts,tsx}'],
  },
});
