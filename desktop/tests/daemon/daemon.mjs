// The daemon the check talks to: the core's own `dcode`, built from this
// checkout, configured only by environment so nothing of the person's own
// configuration, credential or history leaks in or out.

import { spawn, spawnSync } from 'node:child_process';
import { mkdir, realpath, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { healthy } from './wire.mjs';

export const CHECK_VERSION = '0.0.0-check';

/**
 * Builds `dcode` from the repository unless DCODE_BIN names one. From the
 * checkout rather than the installed release: the window has to agree with
 * the protocol it was generated from, and that is this tree's.
 */
export function dcodeBinary(repo, out) {
  if (process.env.DCODE_BIN) return path.resolve(process.env.DCODE_BIN);
  const bin = path.join(out, 'dcode');
  // A version no release carries, so the status bar can be checked for the
  // daemon's own words rather than for something that happens to say "dev".
  const ldflags = `-X github.com/aguinelo/dcode/internal/version.Version=${CHECK_VERSION}`;
  const r = spawnSync('go', ['build', '-ldflags', ldflags, '-o', bin, './cmd/dcode'], { cwd: repo, encoding: 'utf8' });
  if (r.error) throw new Error(`could not run go to build dcode (${r.error.message}); install Go or set DCODE_BIN`);
  if (r.status !== 0) throw new Error(`go build ./cmd/dcode failed:\n${r.stderr}`);
  return bin;
}

/**
 * The environment a daemon of the check runs with: its own home, the scripted
 * model, a fake key, no update check, and the network withheld — so every
 * shell command crosses a boundary and asks, which is what the approval
 * scenarios need.
 */
export function daemonEnv(home, modelURL, socket) {
  return {
    DCODE_HOME: home,
    DCODE_SOCKET: socket,
    DCODE_MODEL: 'scripted',
    DCODE_TRANSPORT: 'openai',
    DCODE_FAMILY: 'generic',
    DCODE_BASE_URL: modelURL,
    DCODE_API_KEY: 'sk-check-not-a-real-key',
    DCODE_ALLOW_NETWORK: 'false',
    DCODE_APPROVAL_POLICY: 'on-request',
    DCODE_SANDBOX_MODE: 'workspace-write',
    DCODE_UPDATE_CHECK: 'false',
  };
}

/**
 * A git repository with one committed file, for sessions to open in. Returned
 * as its real path: the temp directory is a symlink on macOS, and the daemon
 * names a workspace by where it really is.
 */
export async function workspace(dir) {
  await mkdir(dir, { recursive: true });
  await writeFile(path.join(dir, 'README.md'), '# scratch\n\nA workspace the desktop check opens sessions in.\n');
  for (const args of [
    ['init', '-q', '-b', 'main'],
    ['add', 'README.md'],
    ['-c', 'user.name=check', '-c', 'user.email=check@localhost', 'commit', '-q', '-m', 'init'],
  ]) {
    const r = spawnSync('git', args, { cwd: dir, encoding: 'utf8' });
    if (r.status !== 0) throw new Error(`git ${args.join(' ')} failed in ${dir}: ${r.stderr || r.error?.message}`);
  }
  return realpath(dir);
}

/** Starts `dcode serve` on the socket and waits until it answers. */
export async function startDaemon(bin, env, cwd) {
  const log = [];
  const child = spawn(bin, ['serve', '--socket', env.DCODE_SOCKET], { cwd, env: { ...process.env, ...env }, stdio: ['ignore', 'pipe', 'pipe'] });
  child.stdout.on('data', (c) => log.push(String(c)));
  child.stderr.on('data', (c) => log.push(String(c)));
  const exited = new Promise((resolve) => child.once('exit', (code, signal) => resolve({ code, signal })));
  const deadline = Date.now() + 10_000;
  while (!(await healthy(env.DCODE_SOCKET))) {
    const done = await Promise.race([exited, new Promise((r) => setTimeout(() => r(null), 100))]);
    if (done) throw new Error(`dcode serve exited (${done.code ?? done.signal}) before answering:\n${log.join('')}`);
    if (Date.now() > deadline) {
      child.kill('SIGKILL');
      throw new Error(`dcode serve did not answer on ${env.DCODE_SOCKET} within 10 s:\n${log.join('')}`);
    }
  }
  return {
    log,
    pid: child.pid,
    exited,
    /** SIGKILL: the daemon dies the way a crash kills it, with nothing said. */
    kill() {
      child.kill('SIGKILL');
      return exited;
    },
    async stop() {
      if (child.exitCode !== null || child.signalCode !== null) return;
      child.kill('SIGTERM');
      const t = setTimeout(() => child.kill('SIGKILL'), 5000);
      await exited;
      clearTimeout(t);
    },
  };
}
