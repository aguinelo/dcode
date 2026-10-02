import { afterEach, describe, expect, it } from 'vitest';
import { Connection } from '../../src/main/connection';
import type { FileKind, Lookup } from '../../src/main/locate';
import type { Exit, Serve } from '../../src/main/serve';
import { request } from '../../src/main/wire';
import type { DaemonStatus } from '../../src/shared/api';
import { fakeDaemon, tempSocket, until, type FakeDaemon } from './fake-daemon';

// Finding a daemon or starting one (D19): which binary, which socket, a child
// that comes up, one that dies before it answers or never answers, and
// quitting — which stops the daemon the app started and never another.

const TIMING = { healthEveryMs: 40, lostEveryMs: 40, startWithinMs: 1_000, startPollMs: 10, retryFromMs: 10, retryUpToMs: 50 };
const BIN = '/opt/dcode/bin/dcode';

const cleanups: (() => void | Promise<void>)[] = [];
afterEach(async () => {
  for (const c of cleanups.splice(0).reverse()) await c();
});

interface FakeChild extends Serve {
  stops: number;
  die(exit: Exit): void;
}

/** A child process as the connection sees one: it ends when told to, or when stopped. */
function fakeChild(lines: string[], onStop: () => Promise<void> = () => Promise.resolve()): FakeChild {
  let exit: Exit | null = null;
  let announce: (e: Exit) => void = () => {};
  const exited = new Promise<Exit>((resolve) => {
    announce = resolve;
  });
  const child: FakeChild = {
    pid: 4242,
    ended: () => exit,
    exited,
    lines: () => lines,
    stops: 0,
    die(e) {
      if (exit) return;
      exit = e;
      announce(e);
    },
    async stop() {
      child.stops++;
      await onStop();
      child.die({ code: null, signal: 'SIGTERM' });
      return exited;
    },
  };
  return child;
}

function lookup(env: Record<string, string>, files: Record<string, FileKind> = { [BIN]: 'executable' }, printed = ''): Lookup & { ran: string[] } {
  const ran: string[] = [];
  return {
    env,
    home: '/Users/ana',
    cwd: '/',
    probe: (f) => files[f] ?? 'missing',
    run: (file, args) => {
      ran.push([file, ...args].join(' '));
      return Promise.resolve({ kind: 'exited', code: 0, stdout: `${printed}\n`, stderr: '' });
    },
    ran,
  };
}

/** A connection whose `serve` is the test's; every child it started is kept, in order. */
function connect(l: Lookup, serve: (bin: string, socket: string) => FakeChild, timing = TIMING) {
  const statuses: DaemonStatus[] = [];
  const children: FakeChild[] = [];
  const conn = new Connection(
    {
      lookup: l,
      serve: (bin, socket) => {
        const child = serve(bin, socket);
        children.push(child);
        return child;
      },
      timing,
    },
    { status: (s) => statuses.push(s), conversations: () => {}, sessionEvents: () => {}, streamEnd: () => {} },
  );
  cleanups.push(() => conn.close());
  const last = () => statuses[statuses.length - 1];
  const reason = () => {
    const s = last();
    return s && (s.state === 'failed' || s.state === 'lost') ? s.reason : '';
  };
  return { conn, statuses, children, last, reason };
}

function socketDir() {
  const t = tempSocket();
  cleanups.push(t.remove);
  return t.socket;
}

/** Starts the fake daemon on the socket a little later, as a real process takes a moment to listen. */
function comesUp(socket: string): { daemon: () => FakeDaemon | null; ready: Promise<FakeDaemon> } {
  let d: FakeDaemon | null = null;
  const ready = new Promise<FakeDaemon>((resolve) =>
    setTimeout(() => {
      void fakeDaemon(socket).then((up) => {
        d = up;
        cleanups.push(() => up.kill());
        resolve(up);
      });
    }, 30),
  );
  return { daemon: () => d, ready };
}

describe('starting a daemon', () => {
  it('runs dcode serve on the socket when nothing answers, and is connected as started by the app', async () => {
    const socket = socketDir();
    const asked: string[][] = [];
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), (bin, sock) => {
      asked.push([bin, sock]);
      const up = comesUp(sock);
      return fakeChild(['dcode 0.0.0-fake · listening'], async () => void (await up.ready).kill());
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    expect(asked).toEqual([[BIN, socket]]);
    expect(c.last()).toEqual({ state: 'connected', version: '0.0.0-fake', socket, started: true });
    expect(c.conn.startedByApp()).toBe(true);
  });

  it('asks dcode socket where to listen when DCODE_SOCKET is not set', async () => {
    const socket = socketDir();
    const l = lookup({ DCODE_BIN: BIN }, { [BIN]: 'executable' }, socket);
    const c = connect(l, (_bin, sock) => {
      comesUp(sock);
      return fakeChild([]);
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    expect(l.ran).toEqual([`${BIN} socket`]);
  });

  it('is lost when the daemon it started exits, saying how and what it said last', async () => {
    const socket = socketDir();
    const ups: ReturnType<typeof comesUp>[] = [];
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), (_bin, sock) => {
      ups.push(comesUp(sock));
      return fakeChild(['dcode 0.0.0-fake · listening', 'panic: something broke']);
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    await (await ups[0]!.ready).kill();
    c.children[0]!.die({ code: 2, signal: null });
    await until(() => c.reason().includes('saiu com código 2'), 'lost with the exit');
    expect(c.last()?.state).toBe('lost');
    expect(c.reason()).toContain('panic: something broke');
    expect(c.conn.startedByApp()).toBe(false);
  });

  it('fails naming DCODE_BIN when nothing answers and it is missing, and starts nothing', async () => {
    const socket = socketDir();
    const missing = '/tmp/dck-x/missing/dcode';
    let started = false;
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: missing }, {}), () => {
      started = true;
      return fakeChild([]);
    });
    c.conn.start();
    await until(() => c.last()?.state === 'failed', 'failed');
    expect(c.reason()).toContain(missing);
    expect(c.reason()).toContain(`Nenhum daemon responde em ${socket}`);
    expect(started).toBe(false);
    expect(c.statuses.map((s) => s.state)).toEqual(['failed']);
  });

  it('fails listing where it looked when there is no dcode and no DCODE_SOCKET', async () => {
    const c = connect(lookup({ PATH: '/usr/bin' }, {}), () => fakeChild([]));
    c.conn.start();
    await until(() => c.last()?.state === 'failed', 'failed');
    expect(c.reason()).toContain('/Users/ana/.local/bin/dcode');
    expect(c.reason()).toContain('/usr/bin/dcode');
  });

  it('fails with the exit and the last lines of a daemon that ends before answering', async () => {
    const socket = socketDir();
    const said = `dcode: server: another dcode daemon is already listening on ${socket}`;
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), () => {
      const child = fakeChild([said]);
      setTimeout(() => child.die({ code: 1, signal: null }), 20);
      return child;
    });
    c.conn.start();
    await until(() => c.last()?.state === 'failed', 'failed');
    expect(c.reason()).toContain('saiu com código 1 antes de responder');
    expect(c.reason()).toContain(said);
  });

  it('attaches instead when its child lost the socket to another daemon that answers', async () => {
    const socket = socketDir();
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), (_bin, sock) => {
      const child = fakeChild([]);
      void comesUp(sock).ready.then(() => child.die({ code: 1, signal: null }));
      return child;
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    expect(c.last()).toMatchObject({ state: 'connected', started: false });
  });

  it('stops a daemon that does not answer in time, and says so', async () => {
    const socket = socketDir();
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), () => fakeChild(['starting…']), {
      ...TIMING,
      startWithinMs: 150,
    });
    c.conn.start();
    await until(() => c.last()?.state === 'failed', 'failed');
    expect(c.reason()).toContain('não respondeu');
    expect(c.reason()).toContain('o app o encerrou');
    expect(c.children[0]?.stops).toBe(1);
  });
});

describe('closing', () => {
  it('stops the daemon the app started', async () => {
    const socket = socketDir();
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), (_bin, sock) => {
      const up = comesUp(sock);
      return fakeChild([], async () => void (await up.ready).kill());
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    await c.conn.close();
    expect(c.children[0]?.stops).toBe(1);
    expect((await request(socket, 'GET', '/health')).ok).toBe(false);
    // Stopped on purpose, so nothing is said about it to a window that is closing.
    expect(c.statuses.map((s) => s.state)).toEqual(['connected']);
  });

  it('never stops a daemon the app attached to', async () => {
    const socket = socketDir();
    const daemon = await fakeDaemon(socket);
    cleanups.push(() => daemon.kill());
    const c = connect(lookup({ DCODE_SOCKET: socket }), () => {
      throw new Error('started a daemon while one answers');
    });
    c.conn.start();
    await until(() => c.last()?.state === 'connected', 'connected');
    await c.conn.close();
    expect((await request(socket, 'GET', '/health')).ok).toBe(true);
  });

  it('stops a daemon still starting, and goes no further', async () => {
    const socket = socketDir();
    const c = connect(lookup({ DCODE_SOCKET: socket, DCODE_BIN: BIN }), () => fakeChild([]));
    c.conn.start();
    await until(() => c.children.length === 1, 'the child started');
    await c.conn.close();
    expect(c.children[0]?.stops).toBe(1);
    await new Promise((resolve) => setTimeout(resolve, 60));
    expect(c.statuses).toEqual([]);
  });
});
