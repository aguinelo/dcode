// Where a conversation's folder is in git now (`GET /v1/git`) and one changed
// file's diff (`GET /v1/git/diff`), read at the boundary like the memory and
// the skills: an answer that does not match the generated type is a fact to
// say, never a list to guess from.

import type * as P from './generated';
import { check, isRecord, req, type Shape } from './validate';

const FILE: Shape<P.GitFile> = {
  path: req('string'),
  status: req('string'),
  added: req('number'),
  removed: req('number'),
  binary: req('boolean'),
};

type Decoded<T> = { ok: true; value: T } | { ok: false; reason: string };

function optString(raw: Record<string, unknown>, key: string): string | null | undefined {
  const v = raw[key];
  if (v === undefined) return undefined;
  return typeof v === 'string' ? v : null;
}

/** Reads the answer of `GET /v1/git`, as it came off the wire. */
export function decodeGit(raw: unknown): Decoded<P.GitResponse> {
  const fail = (reason: string): Decoded<P.GitResponse> => ({ ok: false, reason: `git da pasta: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  for (const key of ['repository', 'detached', 'worktree', 'truncated'] as const) {
    if (typeof raw[key] !== 'boolean') return fail(`${key} deveria ser booleano`);
  }
  const out: P.GitResponse = {
    repository: raw.repository as boolean,
    detached: raw.detached as boolean,
    worktree: raw.worktree as boolean,
    truncated: raw.truncated as boolean,
    files: [],
  };
  for (const key of ['root', 'branch', 'head', 'main_checkout'] as const) {
    const v = optString(raw, key);
    if (v === null) return fail(`${key} deveria ser texto`);
    if (v) out[key] = v;
  }
  const list = raw.files ?? [];
  if (!Array.isArray(list)) return fail('files deveria ser uma lista');
  for (let i = 0; i < list.length; i++) {
    const why = check(FILE, list[i], `files[${i}]`);
    if (why) return fail(why);
  }
  out.files = list as P.GitFile[];
  return { ok: true, value: out };
}

/** Reads the answer of `GET /v1/git/diff`, as it came off the wire. */
export function decodeGitDiff(raw: unknown): Decoded<P.GitDiffResponse> {
  const fail = (reason: string): Decoded<P.GitDiffResponse> => ({ ok: false, reason: `diff: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  const shape: Shape<P.GitDiffResponse> = {
    path: req('string'),
    status: req('string'),
    diff: req('string'),
    binary: req('boolean'),
    truncated: req('boolean'),
    limit: req('number'),
  };
  const why = check(shape, raw, 'diff');
  if (why) return fail(why);
  return { ok: true, value: raw as unknown as P.GitDiffResponse };
}
