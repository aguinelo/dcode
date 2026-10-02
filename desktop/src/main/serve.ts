import { spawn } from 'node:child_process';

// The daemon the app starts when none answers (D19): `dcode serve` as a child
// process, its output echoed into the app's own as it comes and its last lines
// kept, so a daemon that dies can be said with what it said before dying.

/** How the child ended. */
export interface Exit {
  code: number | null;
  signal: string | null;
  /** Why it never ran, when starting it failed. */
  error?: string;
}

export interface Serve {
  readonly pid: number | undefined;
  /** How it ended, or null while it runs. */
  ended(): Exit | null;
  /** Resolves once it has ended and what it printed has been read. */
  readonly exited: Promise<Exit>;
  /** Its last lines, stdout and stderr in the order they came. */
  lines(): string[];
  /** SIGTERM, up to `graceMs` for it to end, then SIGKILL. Bounded: it never waits on a child that will not die. */
  stop(graceMs?: number): Promise<Exit>;
}

/** Lines kept for the reason shown when it dies. */
export const KEEP_LINES = 40;
/** How long a SIGTERM is given before the SIGKILL. */
export const STOP_GRACE_MS = 5_000;
/** How long the pipes may stay open after the child ended: a grandchild may hold them. */
const PIPES_GRACE_MS = 250;

export interface ServeOptions {
  cwd: string;
  env: NodeJS.ProcessEnv;
  /** Where each line goes as it comes: the app's own stdout or stderr. */
  echo(stream: 'stdout' | 'stderr', line: string): void;
}

/** Resolves with what `p` gives, or with null after `ms`; the timer never outlives the race. */
function within<T>(p: Promise<T>, ms: number): Promise<T | null> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const late = new Promise<null>((resolve) => {
    timer = setTimeout(() => resolve(null), ms);
  });
  return Promise.race([p, late]).finally(() => clearTimeout(timer));
}

/** Splits text into lines as it arrives, keeping the unfinished one for later. */
function lineSplitter(onLine: (line: string) => void): { push(chunk: string): void; flush(): void } {
  let open = '';
  return {
    push(chunk) {
      const parts = (open + chunk).split(/\r?\n/);
      open = parts.pop() ?? '';
      for (const p of parts) onLine(p);
    },
    flush() {
      if (open !== '') onLine(open);
      open = '';
    },
  };
}

/** Starts `<bin> serve --socket <socket>`. */
export function startServe(bin: string, socket: string, o: ServeOptions): Serve {
  const kept: string[] = [];
  let exit: Exit | null = null;
  let announce: (e: Exit) => void = () => {};
  const exited = new Promise<Exit>((resolve) => {
    announce = resolve;
  });

  const child = spawn(bin, ['serve', '--socket', socket], { cwd: o.cwd, env: o.env, stdio: ['ignore', 'pipe', 'pipe'] });
  const keep = (stream: 'stdout' | 'stderr', line: string): void => {
    o.echo(stream, line);
    kept.push(line);
    if (kept.length > KEEP_LINES) kept.shift();
  };
  const out = lineSplitter((l) => keep('stdout', l));
  const err = lineSplitter((l) => keep('stderr', l));
  child.stdout.setEncoding('utf8');
  child.stderr.setEncoding('utf8');
  child.stdout.on('data', (c: string) => out.push(c));
  child.stderr.on('data', (c: string) => err.push(c));

  const finish = (e: Exit): void => {
    if (exit) return;
    out.flush();
    err.flush();
    exit = e;
    announce(e);
  };
  child.on('error', (e) => {
    // Also emitted when a signal cannot be sent; only a child that never got a pid failed to start.
    if (child.pid === undefined) finish({ code: null, signal: null, error: e.message });
  });
  child.on('exit', (code, signal) => {
    const late = setTimeout(() => finish({ code, signal }), PIPES_GRACE_MS);
    child.once('close', () => {
      clearTimeout(late);
      finish({ code, signal });
    });
  });

  return {
    pid: child.pid,
    ended: () => exit,
    exited,
    lines: () => [...kept],
    async stop(graceMs = STOP_GRACE_MS) {
      if (exit) return exit;
      child.kill('SIGTERM');
      const graceful = await within(exited, graceMs);
      if (graceful) return graceful;
      child.kill('SIGKILL');
      return (await within(exited, 1_000)) ?? { code: null, signal: 'SIGKILL' };
    },
  };
}

/** How a child ended, in the words the status bar uses. */
export function exitWords(e: Exit): string {
  if (e.error !== undefined) return `sem chegar a rodar (${e.error})`;
  if (e.signal) return `pelo sinal ${e.signal}`;
  return `com código ${e.code ?? '?'}`;
}

/** The last lines of a child, as the end of a reason; nothing when it said nothing. */
export function lastWords(lines: string[]): string {
  const said = lines.filter((l) => l.trim() !== '');
  return said.length === 0 ? ' Não disse nada antes.' : ` Últimas linhas:\n${said.join('\n')}`;
}
