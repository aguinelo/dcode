import { describe, expect, it } from 'vitest';
import type { GitResponse } from '../../src/protocol/generated';
import { decodeGit, decodeGitDiff } from '../../src/protocol/git';
import {
  branchNow,
  diffLines,
  movedKey,
  rootIfElsewhere,
  shortId,
  startedElsewhere,
  statusOf,
  summarize,
  summaryText,
  worktreeWord,
} from '../../src/state/git';
import { recordedGit } from '../../src/fixtures/git';
import { fixtureActions } from '../../src/renderer/fixture';
import { done, log } from './helpers';

// Where a conversation works, worded from what the daemon says of its
// folder's git (D41). Every state is a sentence — never a blank that reads as
// "no branch" or "nothing changed".

const repo: GitResponse = {
  repository: true,
  root: '/w/dcode',
  branch: 'feat/x',
  detached: false,
  head: 'abc1234',
  worktree: false,
  files: [],
  truncated: false,
};

describe('the folder’s git, worded', () => {
  it('says the branch now, or that HEAD is detached and where', () => {
    expect(branchNow(repo)).toBe('feat/x');
    expect(branchNow({ ...repo, branch: undefined, detached: true })).toBe('HEAD destacado em abc1234');
    expect(branchNow({ ...repo, branch: undefined, detached: true, head: undefined })).toBe('HEAD destacado');
  });

  it('says the branch the session started on only when the folder moved since', () => {
    expect(startedElsewhere(repo, 'feat/x')).toBeNull();
    expect(startedElsewhere(repo, 'main')).toBe('main');
    expect(startedElsewhere(repo, undefined)).toBeNull();
    expect(startedElsewhere({ ...repo, detached: true, branch: undefined }, 'main')).toBeNull();
  });

  it('names the checkout a linked worktree belongs to', () => {
    expect(worktreeWord(repo)).toBeNull();
    expect(worktreeWord({ ...repo, worktree: true, main_checkout: '/src/dcode' })).toBe('worktree de dcode');
    expect(worktreeWord({ ...repo, worktree: true })).toBe('worktree');
  });

  it('says the top of the repository only when it is not the folder itself', () => {
    expect(rootIfElsewhere(repo, '/w/dcode/')).toBeNull();
    expect(rootIfElsewhere(repo, '/w/dcode/desktop')).toBe('/w/dcode');
  });

  it('sums the lines, and says binary files apart rather than as zero', () => {
    const s = summarize([
      { path: 'a', status: 'modified', added: 3, removed: 1, binary: false },
      { path: 'b.png', status: 'untracked', added: 0, removed: 0, binary: true },
    ]);
    expect(s).toEqual({ files: 2, added: 3, removed: 1, binary: 1 });
    expect(summaryText(s)).toBe('2 arquivos · +3 −1 · 1 binário');
    expect(summaryText(summarize([]))).toBe('nada mudou desde o último commit');
  });

  it('gives each status a letter and a word, and an unknown one says it is unknown', () => {
    expect(statusOf('untracked').letter).toBe('U');
    expect(statusOf('conflicted').word).toBe('em conflito');
    expect(statusOf('renamed').word).toContain('renamed');
  });

  it('shortens an id to eight', () => {
    expect(shortId('0123456789abcdef')).toBe('01234567');
    expect(shortId('s-1')).toBe('s-1');
  });
});

describe('a unified diff, line by line', () => {
  it('keeps the headers apart, so +++ is never an added line', () => {
    const lines = diffLines(['diff --git a/a b/a', '--- a/a', '+++ b/a', '@@ -1,2 +1,2 @@', ' same', '-old', '+new', '\\ No newline at end of file', ''].join('\n'));
    expect(lines.map((l) => l.kind)).toEqual(['meta', 'meta', 'meta', 'hunk', 'ctx', 'del', 'add', 'note']);
  });
});

describe('when the folder is worth reading again', () => {
  it('moves with a finished tool call or a new state, not with each fragment of text', () => {
    const before = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'a' } })
      .fold();
    const row = { state: 'running', lastAt: null };
    const k1 = movedKey(row, before);
    const after = log()
      .add('tool.requested', { turn_id: 't', tool_call_id: 'c1', name: 'write', input: { path: 'a' } })
      .add('tool.completed', done('c1', { added: 1 }))
      .fold();
    expect(movedKey(row, after)).not.toBe(k1);
    expect(movedKey({ state: 'recorded', lastAt: '2026-10-13T00:00:00Z' }, undefined)).toBe('recorded:2026-10-13T00:00:00Z');
  });
});

describe('the recording’s git', () => {
  it('answers each recorded folder as the daemon would, and every recorded answer reads at the boundary', async () => {
    const tree = await fixtureActions.readGit('s-catalogar');
    expect(tree.ok && tree.value.worktree && tree.value.branch).toBe('feat/catalogo');
    const eva = await fixtureActions.readGit('s-ci');
    expect(eva.ok && eva.value.repository).toBe(false);
    for (const git of Object.values(recordedGit)) expect(decodeGit(git).ok).toBe(true);
  });

  it('gives a diff only of a file the folder lists, and says so otherwise', async () => {
    const d = await fixtureActions.readGitDiff('s-catalogar', 'internal/alpha/store.go');
    expect(d.ok && d.value.diff).toContain('+\t// The one write path');
    const none = await fixtureActions.readGitDiff('s-checkout', 'internal/alpha/store.go');
    expect(!none.ok && none.why).toContain('nada foi enviado');
    const unknown = await fixtureActions.readGit('s-nowhere');
    expect(!unknown.ok && unknown.why).toContain('nada foi enviado');
  });
});

describe('reading the git routes at the boundary', () => {
  it('reads a tree, a plain folder, and a null list as empty', () => {
    expect(decodeGit(repo)).toEqual({ ok: true, value: repo });
    expect(decodeGit({ repository: false, detached: false, worktree: false, truncated: false, files: null })).toEqual({
      ok: true,
      value: { repository: false, detached: false, worktree: false, truncated: false, files: [] },
    });
  });

  it('says what is wrong in an answer that does not match', () => {
    expect(decodeGit('x')).toEqual({ ok: false, reason: 'git da pasta: a resposta não é um objeto' });
    expect(decodeGit({ ...repo, branch: 3 })).toEqual({ ok: false, reason: 'git da pasta: branch deveria ser texto' });
    expect(decodeGit({ ...repo, files: {} })).toEqual({ ok: false, reason: 'git da pasta: files deveria ser uma lista' });
    const bad = decodeGit({ ...repo, files: [{ path: 'a', status: 'modified', added: '1', removed: 0, binary: false }] });
    expect(!bad.ok && bad.reason).toContain('files[0].added');
  });

  it('reads a diff, and says a malformed one', () => {
    const d = { path: 'a', status: 'modified', diff: '+x\n', binary: false, truncated: true, limit: 262144 };
    expect(decodeGitDiff(d)).toEqual({ ok: true, value: d });
    expect(decodeGitDiff(null)).toEqual({ ok: false, reason: 'diff: a resposta não é um objeto' });
    const bad = decodeGitDiff({ ...d, truncated: 'yes' });
    expect(!bad.ok && bad.reason).toContain('truncated');
  });
});
