// A production build of main, preload and renderer into .vite/, without
// packaging or signing.
//
// Forge has no build-only command: `package` builds and then packages. So this
// loads forge.config.ts the way Forge itself does (jiti) and hands its Vite
// plugin configuration to the plugin's own config generator — the build here is
// the one `electron-forge package` would make, by construction, and there is no
// second copy of the entries or the externals to drift.

import { rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createJiti } from 'jiti';
import { build } from 'vite';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

async function main() {
  const jiti = createJiti(import.meta.url);
  const forge = await jiti.import(path.join(root, 'forge.config.ts'), { default: true });
  const vitePlugin = forge.plugins?.find((p) => p?.name === 'vite');
  if (!vitePlugin?.config) throw new Error('forge.config.ts has no Vite plugin to build from');

  // Internal to the plugin (it has no build-only API); pinned by the lockfile,
  // and a change there fails this script loudly rather than building wrong.
  // A CommonJS module with `exports.default`: from ESM its class sits one level down.
  const mod = await import('@electron-forge/plugin-vite/dist/ViteConfig.js');
  const ViteConfigGenerator = mod.default?.default ?? mod.default;
  if (typeof ViteConfigGenerator !== 'function') throw new Error('the Forge Vite plugin no longer exports its config generator');
  const generator = new ViteConfigGenerator(vitePlugin.config, root, true);

  await rm(path.join(root, '.vite'), { recursive: true, force: true });
  const configs = [...(await generator.getBuildConfigs()), ...(await generator.getRendererConfig())];
  for (const config of configs) {
    await build({ configFile: false, logLevel: 'warn', ...config });
  }
  console.log(`built ${configs.length} targets into .vite/`);
}

main().catch((err) => {
  console.error(`build failed: ${err?.stack ?? err}`);
  process.exit(1);
});
