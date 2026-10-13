import { VitePlugin } from '@electron-forge/plugin-vite';
import { FusesPlugin } from '@electron-forge/plugin-fuses';
import type { ForgeConfig } from '@electron-forge/shared-types';
import { cp } from 'node:fs/promises';
import path from 'node:path';
import { FuseV1Options, FuseVersion } from '@electron/fuses';

// No makers yet: this version builds and runs, it does not package or sign.
// scripts/build.mjs reads the Vite plugin from here, so `npm run build` and
// `electron-forge start` bundle the same entries with the same configuration.
//
// node-pty (D37) is the one native module. It is N-API, so one binary serves
// Node and every Electron alike: macOS and Windows take the prebuilt one it
// ships, Linux compiles it once at `npm ci`. Nothing is rebuilt against
// Electron's headers, so the rebuild step leaves it alone. The Vite plugin
// packages only .vite/, and node-pty is external to the bundle (it finds its
// binary next to its own files), so it is copied into the package after the
// build, and kept out of the asar archive: its spawn-helper is an executable,
// and the .node a library, neither of which can run from inside one.
const NATIVE = 'node-pty';

// The app icon (D49), without its extension: the packager takes icon.icns on
// macOS and icon.ico on Windows. Linux reads no icon at package time; its
// icon.png waits beside them for the maker that will. All three are made from
// icon.svg by `npm run gen:icons`.
const ICON = path.join(__dirname, 'assets', 'icon', 'icon');

const config: ForgeConfig = {
  packagerConfig: {
    icon: ICON,
    asar: { unpack: `**/node_modules/${NATIVE}/**` },
  },
  rebuildConfig: { ignoreModules: [NATIVE] },
  hooks: {
    packageAfterCopy: async (_config, buildPath) => {
      await cp(path.join(__dirname, 'node_modules', NATIVE), path.join(buildPath, 'node_modules', NATIVE), { recursive: true });
    },
  },
  makers: [],
  plugins: [
    new VitePlugin({
      build: [
        { entry: 'src/main/main.ts', config: 'vite.main.config.ts', target: 'main' },
        { entry: 'src/preload/preload.ts', config: 'vite.preload.config.ts', target: 'preload' },
      ],
      renderer: [{ name: 'main_window', config: 'vite.renderer.config.ts' }],
    }),
    // Switched off at package time, before signing: a desktop client of a
    // daemon has no use for running as Node, for NODE_OPTIONS or for an
    // inspector flag, and each is a way in.
    new FusesPlugin({
      version: FuseVersion.V1,
      [FuseV1Options.RunAsNode]: false,
      [FuseV1Options.EnableCookieEncryption]: true,
      [FuseV1Options.EnableNodeOptionsEnvironmentVariable]: false,
      [FuseV1Options.EnableNodeCliInspectArguments]: false,
      [FuseV1Options.EnableEmbeddedAsarIntegrityValidation]: true,
      [FuseV1Options.OnlyLoadAppFromAsar]: true,
    }),
  ],
};

export default config;
