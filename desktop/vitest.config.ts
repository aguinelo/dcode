import { defineConfig } from 'vitest/config';

// Two suites, no DOM and no Electron. tests/unit: the pure state the window
// draws, fed protocol events — what the events mean. tests/main: the main
// process's own decisions, under Node — typed by tsconfig.node.json, because
// the renderer's project has no Node in it.
export default defineConfig({
  test: {
    include: ['tests/unit/**/*.test.{ts,tsx}', 'tests/main/**/*.test.ts'],
    environment: 'node',
  },
});
