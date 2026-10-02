import { chmodSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { KEEP_LINES, exitWords, lastWords, startServe } from '../../src/main/serve';
import { until } from './fake-daemon';

// The child the app starts, with a shell script standing in for dcode: what it
// is run with, what it says, how it ends, and a stop that never waits forever.

const dirs: string[] = [];
afterEach(() => {
  for (const d of dirs.splice(0)) rmSync(d, { recursive: true, force: true });
});

/** A fake dcode: a shell script with the given body. */
function script(body: string): { bin: string; dir: string } {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'dcm-'));
  dirs.push(dir);
  const bin = path.join(dir, 'dcode');
  writeFileSync(bin, `#!/bin/sh\n${body}\n`);
  chmodSync(bin, 0o755);
  return { bin, dir };
}

function start(bin: string, cwd: string) {
  const echoed: string[] = [];
  const child = startServe(bin, '/tmp/dcm-x/s.sock', {
    cwd,
    env: process.env,
    echo: (stream, line) => echoed.push(`${stream}: ${line}`),
  });
  return { child, echoed };
}

describe('the daemon the app starts', () => {
  it('runs `serve --socket <socket>` where it is told, echoing each line and keeping them', async () => {
    const { bin, dir } = script('echo "args: $*"\necho "cwd: $(pwd -P)"\necho "warn: careful" >&2\nexit 3');
    const { child, echoed } = start(bin, dir);
    expect(await child.exited).toEqual({ code: 3, signal: null });
    expect(child.ended()).toEqual({ code: 3, signal: null });
    expect(echoed).toContain('stdout: args: serve --socket /tmp/dcm-x/s.sock');
    expect(echoed).toContain(`stdout: cwd: ${realpathSync(dir)}`);
    expect(echoed).toContain('stderr: warn: careful');
    expect(child.lines()).toHaveLength(3);
  });

  it('keeps only its last lines', async () => {
    const { bin, dir } = script('i=1\nwhile [ $i -le 50 ]; do echo "line $i"; i=$((i+1)); done');
    const { child } = start(bin, dir);
    await child.exited;
    expect(child.lines()).toHaveLength(KEEP_LINES);
    expect(child.lines()[0]).toBe(`line ${50 - KEEP_LINES + 1}`);
    expect(child.lines().at(-1)).toBe('line 50');
  });

  it('is stopped with SIGTERM when it listens to it', async () => {
    const { bin, dir } = script('echo ready\nwhile true; do sleep 0.05; done');
    const { child } = start(bin, dir);
    await until(() => child.lines().includes('ready'), 'the script to start');
    expect(child.ended()).toBeNull();
    expect(await child.stop(2_000)).toEqual({ code: null, signal: 'SIGTERM' });
  });

  it('is killed when it ignores SIGTERM past the grace, and the stop does not wait longer', async () => {
    const { bin, dir } = script("trap '' TERM\necho ready\nwhile true; do sleep 0.05; done");
    const { child } = start(bin, dir);
    await until(() => child.lines().includes('ready'), 'the script to start');
    const began = Date.now();
    const exit = await child.stop(100);
    expect(exit.signal).toBe('SIGKILL');
    expect(Date.now() - began).toBeLessThan(1_500);
  });

  it('says why when it cannot be run at all', async () => {
    const { child } = start('/nonexistent/dcode', os.tmpdir());
    const exit = await child.exited;
    expect(exit.error).toContain('ENOENT');
    expect(exitWords(exit)).toContain('sem chegar a rodar');
  });
});

describe('how an ended daemon is told', () => {
  it('says the code or the signal', () => {
    expect(exitWords({ code: 1, signal: null })).toBe('com código 1');
    expect(exitWords({ code: null, signal: 'SIGKILL' })).toBe('pelo sinal SIGKILL');
  });

  it('ends with its last lines, or says it said nothing', () => {
    expect(lastWords(['dcode 0.23.0 · listening', '', 'panic: x'])).toBe(' Últimas linhas:\ndcode 0.23.0 · listening\npanic: x');
    expect(lastWords(['', ' '])).toBe(' Não disse nada antes.');
  });
});
