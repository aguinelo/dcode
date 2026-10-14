import { describe, expect, it } from 'vitest';
import { bundleRoot } from '../../scripts/dev-name.mjs';

describe('the development bundle that is renamed', () => {
  it('is the .app around an executable in Contents/MacOS', () => {
    expect(bundleRoot('/x/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron')).toBe('/x/node_modules/electron/dist/Electron.app');
  });

  it('is none for an executable that is not in a bundle', () => {
    expect(bundleRoot('/x/node_modules/electron/dist/electron')).toBeNull();
    expect(bundleRoot('/x/Electron/Contents/MacOS/Electron')).toBeNull();
  });
});
