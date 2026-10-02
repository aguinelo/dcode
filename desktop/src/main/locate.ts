import { execFile, type ExecFileException } from 'node:child_process';
import { accessSync, constants, statSync } from 'node:fs';
import path from 'node:path';

// Where the daemon is (D19): which binary would start one, and which socket
// one listens on. The socket comes from DCODE_SOCKET or from `dcode socket`
// itself — never from a second copy of the core's rule, which is the copy that
// drifts.
//
// The environment, the filesystem and the running of commands are handed in,
// so each rule here is tested without a dcode installed anywhere.

export type Found = { ok: true; path: string } | { ok: false; reason: string };

/** What a path holds, as far as running it goes. */
export type FileKind = 'executable' | 'missing' | 'not-file' | 'not-executable' | 'unreadable';

/** How a short command ended. */
export type RunResult =
  | { kind: 'exited'; code: number; stdout: string; stderr: string }
  | { kind: 'killed'; signal: string; timedOut: boolean; stderr: string }
  | { kind: 'error'; message: string };

export interface Lookup {
  env: Readonly<Record<string, string | undefined>>;
  home: string;
  /** Where relative paths in the environment are read from: the app's working directory. */
  cwd: string;
  probe(file: string): FileKind;
  run(file: string, args: string[], timeoutMs: number): Promise<RunResult>;
}

/** How long `dcode socket` has to answer. */
export const SOCKET_TIMEOUT_MS = 5_000;

const WHY: Record<Exclude<FileKind, 'executable'>, string> = {
  missing: 'que não existe',
  'not-file': 'que não é um arquivo',
  'not-executable': 'que não é executável',
  unreadable: 'que o app não consegue ler',
};

/** The places a dcode is looked for when DCODE_BIN names none, in order, each once. */
export function candidates(l: Lookup): string[] {
  const dirs = (l.env.PATH ?? '').split(path.delimiter).filter((d) => d !== '');
  const all = [path.join(l.home, '.local', 'bin', 'dcode'), ...dirs.map((d) => path.join(path.resolve(l.cwd, d), 'dcode'))];
  return [...new Set(all)];
}

/**
 * The binary that starts a daemon: DCODE_BIN, else `~/.local/bin/dcode` —
 * where install.sh and `make install` put it — else the first `dcode` on the
 * PATH.
 */
export function findBinary(l: Lookup): Found {
  const chosen = l.env.DCODE_BIN;
  if (chosen) {
    const file = path.resolve(l.cwd, chosen);
    const kind = l.probe(file);
    if (kind === 'executable') return { ok: true, path: file };
    // No fallback: a binary named on purpose and missing is a mistake to show,
    // not one to cover with whichever dcode happens to be installed.
    return { ok: false, reason: `DCODE_BIN aponta para ${file}, ${WHY[kind]}.` };
  }
  const tried = candidates(l);
  for (const file of tried) {
    if (l.probe(file) === 'executable') return { ok: true, path: file };
  }
  return {
    ok: false,
    reason:
      `Não achei o dcode. DCODE_BIN não está definido, e procurei em ${tried.join(', ')}. ` +
      'Instale o dcode ou aponte DCODE_BIN para ele.',
  };
}

/** DCODE_SOCKET, when it names one: attaching there needs no binary at all. */
export function chosenSocket(l: Pick<Lookup, 'env' | 'cwd'>): string | null {
  const chosen = l.env.DCODE_SOCKET;
  return chosen ? path.resolve(l.cwd, chosen) : null;
}

/** The socket `<bin> socket` prints: the core's own answer to where its daemon listens (N1). */
export async function askSocket(l: Lookup, bin: string): Promise<Found> {
  const cmd = `“${bin} socket”`;
  const r = await l.run(bin, ['socket'], SOCKET_TIMEOUT_MS);
  switch (r.kind) {
    case 'error':
      return { ok: false, reason: `Não deu para rodar ${cmd}: ${r.message}.` };
    case 'killed': {
      if (r.timedOut) return { ok: false, reason: `${cmd} não respondeu em ${SOCKET_TIMEOUT_MS / 1000} s.` };
      const said = r.stderr.trim();
      return { ok: false, reason: `${cmd} morreu pelo sinal ${r.signal}${said ? `: ${said}` : '.'}` };
    }
    case 'exited': {
      if (r.code !== 0) {
        const said = r.stderr.trim();
        return { ok: false, reason: said ? `${cmd} falhou: ${said}` : `${cmd} saiu com código ${r.code} sem dizer por quê.` };
      }
      const printed = r.stdout.trim();
      if (printed === '') return { ok: false, reason: `${cmd} não imprimiu caminho nenhum.` };
      if (printed.includes('\n') || !path.isAbsolute(printed)) {
        return { ok: false, reason: `${cmd} imprimiu “${printed}”, que não é um caminho absoluto.` };
      }
      return { ok: true, path: printed };
    }
  }
}

export type SocketFound = { ok: true; path: string; bin: string | null } | { ok: false; reason: string };

/**
 * The socket: DCODE_SOCKET, else what the binary says. Returns the binary it
 * asked, when it asked one, so starting a daemon does not look for it twice.
 */
export async function findSocket(l: Lookup): Promise<SocketFound> {
  const chosen = chosenSocket(l);
  if (chosen !== null) return { ok: true, path: chosen, bin: null };
  const bin = findBinary(l);
  if (!bin.ok) return bin;
  const socket = await askSocket(l, bin.path);
  return socket.ok ? { ok: true, path: socket.path, bin: bin.path } : socket;
}

function probeFile(file: string): FileKind {
  try {
    if (!statSync(file).isFile()) return 'not-file';
  } catch (err) {
    const code = (err as NodeJS.ErrnoException).code;
    return code === 'ENOENT' || code === 'ENOTDIR' ? 'missing' : 'unreadable';
  }
  try {
    accessSync(file, constants.X_OK);
    return 'executable';
  } catch {
    return 'not-executable';
  }
}

function runShort(file: string, args: string[], timeoutMs: number): Promise<RunResult> {
  return new Promise((resolve) => {
    execFile(file, args, { timeout: timeoutMs, encoding: 'utf8' }, (err: ExecFileException | null, stdout, stderr) => {
      if (err === null) return resolve({ kind: 'exited', code: 0, stdout, stderr });
      if (typeof err.code === 'number') return resolve({ kind: 'exited', code: err.code, stdout, stderr });
      if (err.signal) return resolve({ kind: 'killed', signal: err.signal, timedOut: err.killed === true, stderr });
      resolve({ kind: 'error', message: err.message });
    });
  });
}

/** The lookup of the running app: its environment, its disk, its processes. */
export function systemLookup(env: Readonly<Record<string, string | undefined>>, home: string, cwd: string): Lookup {
  return { env, home, cwd, probe: probeFile, run: runShort };
}
