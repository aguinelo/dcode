// Where a conversation's folder is in git now (`GET /v1/git`) and one changed
// file's diff (`GET /v1/git/diff`), read at the boundary like the memory and
// the skills: an answer that does not match the generated type is a fact to
// say, never a list to guess from.

import type * as P from './generated';
import { check, isRecord, opt, req, type Shape } from './validate';

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

const CHECKS: Shape<P.GitChecks> = {
  state: req('string'),
  passed: req('number'),
  failed: req('number'),
  pending: req('number'),
  skipped: req('number'),
};

const PR: Shape<P.GitPR> = {
  number: req('number'),
  url: req('string'),
  title: req('string'),
  state: req('string'),
  draft: req('boolean'),
  base: req('string'),
  head: req('string'),
  checks: req('object', CHECKS),
};

const PR_RESPONSE: Shape<P.GitPRResponse> = {
  branch: req('string'),
  base: req('string'),
  pr: opt('object', PR),
};

/**
 * Reads the answer of `GET /v1/git/pr` and of `POST /v1/git/pr` (D60). A PR
 * whose address is not https is refused here: the line opens it in the
 * browser, and only an https link ever leaves the window.
 */
export function decodeGitPR(raw: unknown): Decoded<P.GitPRResponse> {
  const why = check(PR_RESPONSE, raw, 'pr');
  if (why) return { ok: false, reason: `PR da branch: ${why}` };
  const got = raw as unknown as P.GitPRResponse;
  if (!got.pr) return { ok: true, value: { branch: got.branch, base: got.base } };
  if (!got.pr.url.startsWith('https://')) return { ok: false, reason: `PR da branch: o endereço não é https: ${got.pr.url}` };
  return { ok: true, value: got };
}
