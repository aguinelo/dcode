import { describe, expect, it } from 'vitest';
import { userDataDir } from '../../src/main/paths';

describe('where the app keeps its data', () => {
  const appData = '/Users/ana/Library/Application Support';

  it('is a folder of its own, never the one dcode keeps its configuration in', () => {
    const got = userDataDir({}, appData);
    expect(got).toEqual({ ok: true, dir: `${appData}/dcode-desktop` });
    // macOS compares names without case, so Electron's default — the product
    // name, "DCode" — is dcode's own folder there.
    expect(got.ok && got.dir.toLowerCase()).not.toBe(`${appData}/dcode`.toLowerCase());
  });

  it('is DCODE_DESKTOP_USER_DATA when that names one', () => {
    expect(userDataDir({ DCODE_DESKTOP_USER_DATA: '/tmp/dck-1/user-data' }, appData)).toEqual({
      ok: true,
      dir: '/tmp/dck-1/user-data',
    });
  });

  it('refuses a relative DCODE_DESKTOP_USER_DATA, naming it', () => {
    const got = userDataDir({ DCODE_DESKTOP_USER_DATA: 'user-data' }, appData);
    expect(got.ok).toBe(false);
    expect(!got.ok && got.reason).toContain('“user-data”');
  });
});
