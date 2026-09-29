import js from '@eslint/js';
import { defineConfig } from 'eslint/config';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default defineConfig(
  {
    // generated.ts comes from tygo and is compared byte for byte in CI, so it
    // is never reformatted or linted here.
    ignores: ['.vite/', 'out/', 'dist/', 'node_modules/', 'test-results/', 'src/protocol/generated.ts'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['src/renderer/**/*.{ts,tsx}'],
    extends: [reactHooks.configs.flat.recommended],
    languageOptions: { globals: globals.browser },
  },
  {
    // The renderer never sees Node: what it needs from outside comes through
    // the preload's API. Held here rather than trusted to review.
    files: ['src/renderer/**', 'src/state/**', 'src/protocol/**', 'src/fixtures/**', 'src/shared/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            { group: ['electron', 'electron/*'], message: 'The renderer reaches Electron only through the preload API.' },
            { group: ['node:*', 'fs', 'fs/*', 'path', 'os', 'child_process', 'net'], message: 'The renderer never sees Node.' },
          ],
        },
      ],
    },
  },
  {
    files: ['src/main/**', 'src/preload/**', 'scripts/**', '*.config.*'],
    languageOptions: { globals: globals.node },
  },
  {
    // Node, with callbacks that Playwright runs inside the page.
    files: ['tests/visual/**'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },
);
