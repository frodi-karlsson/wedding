/// <reference types="vitest/config" />
import { getViteConfig } from 'astro/config';

export default getViteConfig({
  test: { environment: 'jsdom', globals: true, isolate: false },
});
