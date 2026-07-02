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
  },
});
