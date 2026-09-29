import { defineConfig } from 'vitest/config';

// The unit suite: the pure state the window draws, fed protocol events. No
// DOM and no Electron — what is tested here is what the events mean.
export default defineConfig({
  test: {
    include: ['tests/unit/**/*.test.{ts,tsx}'],
    environment: 'node',
  },
});
