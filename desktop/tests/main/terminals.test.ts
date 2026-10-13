import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Terminals, type Pty, type TerminalPush } from '../../src/main/terminals';

// The PTYs the main process holds for the windows: each belongs to the window
// that opened it, opens only in the session's workspace, and has the proposed
// command typed — never run — once the shell shows its first prompt.

class FakePty implements Pty {
  written: string[] = [];
  size: [number, number] = [0, 0];
  killed = false;
  private data: (d: string) => void = () => {};
  private exit: (e: { exitCode: number; signal?: number }) => void = () => {};
  onData(cb: (d: string) => void) {
    this.data = cb;
  }
  onExit(cb: (e: { exitCode: number; signal?: number }) => void) {
    this.exit = cb;
  }
  write(d: string) {
    this.written.push(d);
  }
  resize(c: number, r: number) {
    this.size = [c, r];
  }
  kill() {
    this.killed = true;
  }
  emit(d: string) {
    this.data(d);
  }
  end(code: number) {
    this.exit({ exitCode: code });
  }
}

let ptys: FakePty[];
let pushed: Array<{ owner: number; push: TerminalPush }>;
let spawned: Array<{ file: string; args: string[]; cwd: string }>;

function terminals(over: Partial<ConstructorParameters<typeof Terminals>[0]> = {}) {
  return new Terminals({
    spawn: (file, args, o) => {
      spawned.push({ file, args, cwd: o.cwd });
      const p = new FakePty();
      ptys.push(p);
      return p;
    },
    prepare: async () => '/data/shell',
    env: { SHELL: '/bin/zsh', HOME: '/Users/ana' },
    platform: 'darwin',
    isDir: async (dir) => dir === '/work/app',
    push: (owner, push) => pushed.push({ owner, push }),
    typeAfterMs: 3000,
    max: 2,
    ...over,
  });
}

beforeEach(() => {
  ptys = [];
  pushed = [];
  spawned = [];
  vi.useFakeTimers();
});
afterEach(() => vi.useRealTimers());

describe('a terminal for a window', () => {
  it('opens in the workspace and types the command after the first prompt, without Enter', async () => {
    const t = terminals();
    const opened = await t.open(7, '/work/app', 'npm test', 80, 24);
    expect(opened.ok).toBe(true);
    expect(spawned[0]).toEqual({ file: '/bin/zsh', args: ['-l'], cwd: '/work/app' });
    const pty = ptys[0]!;
    pty.emit('loading rc');
    expect(pty.written).toEqual([]);
    pty.emit('\x1b]133;D\x07% ');
    // The prompt is marked, and the line editor has not started reading: typed now, the tty would echo it.
    expect(pty.written).toEqual([]);
    pty.emit('\x1b[?2004h');
    expect(pty.written).toEqual(['\x1b[200~npm test\x1b[201~']);
    expect(pushed.map((p) => p.owner)).toEqual([7, 7, 7]);
    expect(pushed[0]?.push).toEqual({ kind: 'data', id: opened.ok ? opened.value.id : '', data: 'loading rc' });
    vi.advanceTimersByTime(5000);
    expect(pty.written).toHaveLength(1);
  });

  it('types it a moment after the prompt when no line editor announces itself', async () => {
    const t = terminals();
    await t.open(7, '/work/app', 'ls', 80, 24);
    ptys[0]?.emit('\x1b]133;D\x07$ ');
    vi.advanceTimersByTime(249);
    expect(ptys[0]?.written).toEqual([]);
    vi.advanceTimersByTime(1);
    expect(ptys[0]?.written).toEqual(['\x1b[200~ls\x1b[201~']);
  });

  it('types it anyway when the shell never marks a prompt', async () => {
    const t = terminals();
    await t.open(7, '/work/app', 'ls', 80, 24);
    vi.advanceTimersByTime(3000);
    expect(ptys[0]?.written).toEqual(['\x1b[200~ls\x1b[201~']);
  });

  it('refuses a folder that is not there, and a command it cannot type', async () => {
    const t = terminals();
    const gone = await t.open(7, '/work/gone', 'ls', 80, 24);
    expect(gone.ok).toBe(false);
    if (!gone.ok) expect(gone.refusal.message).toContain('/work/gone');
    expect((await t.open(7, '/work/app', 'ls\r', 80, 24)).ok).toBe(false);
    expect(spawned).toHaveLength(0);
  });

  it('holds no more than its limit, and says so', async () => {
    const t = terminals();
    await t.open(1, '/work/app', 'a', 80, 24);
    await t.open(1, '/work/app', 'b', 80, 24);
    const third = await t.open(1, '/work/app', 'c', 80, 24);
    expect(third.ok).toBe(false);
    if (!third.ok) expect(third.refusal.message).toContain('2');
  });

  it('answers only the window that opened it', async () => {
    const t = terminals();
    const o = await t.open(7, '/work/app', 'ls', 80, 24);
    const id = o.ok ? o.value.id : '';
    expect(t.write(8, id, 'x').ok).toBe(false);
    expect(t.resize(8, id, 100, 30).ok).toBe(false);
    expect(t.kill(8, id).ok).toBe(false);
    expect(t.write(7, 'nope', 'x').ok).toBe(false);
    expect(t.write(7, id, 'sudo !!\r').ok).toBe(true);
    expect(t.resize(7, id, 100, 30).ok).toBe(true);
    expect(ptys[0]?.size).toEqual([100, 30]);
    expect(ptys[0]?.written).toContain('sudo !!\r');
  });

  it('says when the shell ends, and forgets it', async () => {
    const t = terminals();
    const o = await t.open(7, '/work/app', 'ls', 80, 24);
    const id = o.ok ? o.value.id : '';
    ptys[0]?.end(130);
    expect(pushed.at(-1)).toEqual({ owner: 7, push: { kind: 'exit', id, code: 130 } });
    expect(t.write(7, id, 'x').ok).toBe(false);
  });

  it('kills what a closed window left, and everything on quit', async () => {
    const t = terminals({ max: 8 });
    await t.open(1, '/work/app', 'a', 80, 24);
    await t.open(2, '/work/app', 'b', 80, 24);
    t.closeOwner(1);
    expect(ptys.map((p) => p.killed)).toEqual([true, false]);
    t.closeAll();
    expect(ptys.map((p) => p.killed)).toEqual([true, true]);
  });
});
