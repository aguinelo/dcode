import { describe, expect, it } from 'vitest';
import { findBinary, findSocket, type FileKind, type Lookup, type RunResult } from '../../src/main/locate';

const HOME = '/Users/ana';
const LOCAL = '/Users/ana/.local/bin/dcode';

/**
 * A machine with the given files, and `dcode socket` answering `socket`. Every
 * command run is recorded, so a test can say nothing was asked.
 */
function machine(env: Record<string, string>, files: Record<string, FileKind>, socket: RunResult = ok('/tmp/dcode-501/dcode.sock\n')) {
  const ran: string[] = [];
  const lookup: Lookup = {
    env,
    home: HOME,
    cwd: '/Users/ana/work',
    probe: (file) => files[file] ?? 'missing',
    run: (file, args) => {
      ran.push([file, ...args].join(' '));
      return Promise.resolve(socket);
    },
  };
  return { lookup, ran };
}

function ok(stdout: string): RunResult {
  return { kind: 'exited', code: 0, stdout, stderr: '' };
}

describe('the binary that starts a daemon', () => {
  it('is DCODE_BIN when that names one', () => {
    const { lookup } = machine({ DCODE_BIN: '/opt/dcode/bin/dcode' }, { '/opt/dcode/bin/dcode': 'executable', [LOCAL]: 'executable' });
    expect(findBinary(lookup)).toEqual({ ok: true, path: '/opt/dcode/bin/dcode' });
  });

  it('fails naming a missing DCODE_BIN, and never falls back to another dcode', () => {
    const { lookup } = machine(
      { DCODE_BIN: '/tmp/dck-1/missing/dcode', PATH: '/usr/local/bin' },
      { [LOCAL]: 'executable', '/usr/local/bin/dcode': 'executable' },
    );
    const got = findBinary(lookup);
    expect(got.ok).toBe(false);
    expect(!got.ok && got.reason).toContain('/tmp/dck-1/missing/dcode');
    expect(!got.ok && got.reason).toContain('não existe');
  });

  it('fails naming a DCODE_BIN that is not executable, or not a file', () => {
    const notExec = findBinary(machine({ DCODE_BIN: '/opt/dcode' }, { '/opt/dcode': 'not-executable' }).lookup);
    expect(!notExec.ok && notExec.reason).toBe('DCODE_BIN aponta para /opt/dcode, que não é executável.');
    const dir = findBinary(machine({ DCODE_BIN: '/opt' }, { '/opt': 'not-file' }).lookup);
    expect(!dir.ok && dir.reason).toBe('DCODE_BIN aponta para /opt, que não é um arquivo.');
  });

  it('reads a relative DCODE_BIN from where the app was opened', () => {
    const { lookup } = machine({ DCODE_BIN: 'bin/dcode' }, { '/Users/ana/work/bin/dcode': 'executable' });
    expect(findBinary(lookup)).toEqual({ ok: true, path: '/Users/ana/work/bin/dcode' });
  });

  it('is ~/.local/bin/dcode before anything on the PATH', () => {
    const { lookup } = machine({ PATH: '/opt/homebrew/bin:/usr/local/bin' }, { [LOCAL]: 'executable', '/opt/homebrew/bin/dcode': 'executable' });
    expect(findBinary(lookup)).toEqual({ ok: true, path: LOCAL });
  });

  it('is the first executable dcode on the PATH when ~/.local/bin has none', () => {
    const { lookup } = machine(
      { PATH: '/opt/homebrew/bin:/usr/local/bin:/usr/bin' },
      { [LOCAL]: 'not-executable', '/opt/homebrew/bin/dcode': 'not-executable', '/usr/local/bin/dcode': 'executable', '/usr/bin/dcode': 'executable' },
    );
    expect(findBinary(lookup)).toEqual({ ok: true, path: '/usr/local/bin/dcode' });
  });

  it('fails listing every place it looked when there is none', () => {
    const { lookup } = machine({ PATH: '/opt/homebrew/bin::/usr/bin' }, {});
    const got = findBinary(lookup);
    expect(got.ok).toBe(false);
    if (got.ok) return;
    expect(got.reason).toContain('DCODE_BIN não está definido');
    for (const place of [LOCAL, '/opt/homebrew/bin/dcode', '/usr/bin/dcode']) expect(got.reason).toContain(place);
  });
});

describe('the socket', () => {
  it('is DCODE_SOCKET, without asking dcode socket or looking for a binary', async () => {
    const { lookup, ran } = machine({ DCODE_SOCKET: '/tmp/dck-1/d.sock', DCODE_BIN: '/nowhere/dcode' }, {});
    expect(await findSocket(lookup)).toEqual({ ok: true, path: '/tmp/dck-1/d.sock', bin: null });
    expect(ran).toEqual([]);
  });

  it('is what dcode socket prints, trimmed, when DCODE_SOCKET is not set', async () => {
    const { lookup, ran } = machine({}, { [LOCAL]: 'executable' }, ok('  /tmp/dcode-501/dcode.sock\n'));
    expect(await findSocket(lookup)).toEqual({ ok: true, path: '/tmp/dcode-501/dcode.sock', bin: LOCAL });
    expect(ran).toEqual([`${LOCAL} socket`]);
  });

  it('fails with what dcode socket said on stderr when it fails', async () => {
    const said = 'dcode: /tmp/dcode-501 is not owned by you\n';
    const { lookup } = machine({}, { [LOCAL]: 'executable' }, { kind: 'exited', code: 1, stdout: '', stderr: said });
    const got = await findSocket(lookup);
    expect(got.ok).toBe(false);
    expect(!got.ok && got.reason).toContain('dcode: /tmp/dcode-501 is not owned by you');
  });

  it('fails with the exit code when dcode socket fails without a word', async () => {
    const { lookup } = machine({}, { [LOCAL]: 'executable' }, { kind: 'exited', code: 2, stdout: '', stderr: '' });
    const got = await findSocket(lookup);
    expect(!got.ok && got.reason).toBe(`“${LOCAL} socket” saiu com código 2 sem dizer por quê.`);
  });

  it('fails saying so when dcode socket does not answer in time, or prints no path', async () => {
    const late = await findSocket(machine({}, { [LOCAL]: 'executable' }, { kind: 'killed', signal: 'SIGTERM', timedOut: true, stderr: '' }).lookup);
    expect(!late.ok && late.reason).toBe(`“${LOCAL} socket” não respondeu em 5 s.`);
    const blank = await findSocket(machine({}, { [LOCAL]: 'executable' }, ok('\n')).lookup);
    expect(!blank.ok && blank.reason).toContain('não imprimiu caminho nenhum');
    const relative = await findSocket(machine({}, { [LOCAL]: 'executable' }, ok('dcode.sock\n')).lookup);
    expect(!relative.ok && relative.reason).toContain('não é um caminho absoluto');
  });

  it('fails with the binary’s reason when there is no DCODE_SOCKET and no dcode to ask', async () => {
    const got = await findSocket(machine({ DCODE_BIN: '/tmp/x/dcode' }, {}).lookup);
    expect(!got.ok && got.reason).toBe('DCODE_BIN aponta para /tmp/x/dcode, que não existe.');
  });
});
