import { defineConfig } from 'vite';

// The entry, the externals and the build defines come from the Forge Vite
// plugin (forge.config.ts). One more external: node-pty is native and finds
// its binary relative to its own folder, so it is required at run time from
// node_modules, never bundled (D37). Vite merges this list with Forge's.
export default defineConfig({
  build: {
    rollupOptions: {
      external: ['node-pty'],
    },
  },
});
