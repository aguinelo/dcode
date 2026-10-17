// What the recording's daemon answers about each project's git (D41): the
// dcode folder is a linked worktree with work in it, bizpack a clean checkout,
// and eva no repository at all — the three things the panel has to tell apart.
// One answer per folder, as the daemon gives it: the conversations in a folder
// share its branch, whatever branch each started on.

import type { GitDiffResponse, GitPRResponse, GitResponse } from '../protocol/generated';

const DCODE = '/Users/aguinelo/work/dcode';
const BIZPACK = '/Users/aguinelo/work/bizpack';
const EVA = '/Users/aguinelo/work/eva';
const LOJA = '/Users/aguinelo/work/loja';

const LIMIT = 262144;

export const recordedGit: Readonly<Record<string, GitResponse>> = {
  [DCODE]: {
    repository: true,
    root: DCODE,
    branch: 'feat/catalogo',
    detached: false,
    head: '9656919',
    worktree: true,
    main_checkout: '/Users/aguinelo/src/dcode',
    files: [
      { path: 'internal/alpha/ARCH.md', status: 'untracked', added: 24, removed: 0, binary: false },
      { path: 'internal/alpha/store.go', status: 'modified', added: 3, removed: 1, binary: false },
      { path: 'internal/bravo/ARCH.md', status: 'untracked', added: 0, removed: 0, binary: false },
      { path: 'internal/store/legacy.go', status: 'deleted', added: 0, removed: 18, binary: false },
    ],
    truncated: false,
  },
  [BIZPACK]: {
    repository: true,
    root: BIZPACK,
    branch: 'main',
    detached: false,
    head: '4be1c02',
    worktree: false,
    files: [],
    truncated: false,
  },
  [EVA]: { repository: false, detached: false, worktree: false, files: [], truncated: false },
  // The second recording's shop (loop-recording.ts): the frete work in the folder, not yet committed.
  [LOJA]: {
    repository: true,
    root: LOJA,
    branch: 'feat/frete',
    detached: false,
    head: '1c0d2e7',
    worktree: false,
    files: [
      { path: 'internal/frete/tabela.go', status: 'untracked', added: 88, removed: 0, binary: false },
      { path: 'internal/checkout/total.go', status: 'modified', added: 12, removed: 0, binary: false },
    ],
    truncated: false,
  },
};

const diff = (path: string, status: string, text: string): GitDiffResponse => ({
  path,
  status,
  diff: text,
  binary: false,
  truncated: false,
  limit: LIMIT,
});

export const recordedDiffs: Readonly<Record<string, GitDiffResponse>> = {
  'internal/alpha/store.go': diff(
    'internal/alpha/store.go',
    'modified',
    [
      'diff --git a/internal/alpha/store.go b/internal/alpha/store.go',
      'index 3f1c2aa..8d0e4b1 100644',
      '--- a/internal/alpha/store.go',
      '+++ b/internal/alpha/store.go',
      '@@ -38,7 +38,9 @@ func (s *Store) Put(ctx context.Context, rec Record) error {',
      ' \tif err := rec.Validate(); err != nil {',
      ' \t\treturn err',
      ' \t}',
      '-\treturn s.Save(ctx, rec)',
      '+\t// The one write path: see ARCH.md.',
      '+\tif err := s.Save(ctx, rec); err != nil {',
      '+\t\treturn fmt.Errorf("alpha: put %s: %w", rec.ID, err)',
      ' }',
      '',
    ].join('\n'),
  ),
  'internal/alpha/ARCH.md': diff(
    'internal/alpha/ARCH.md',
    'untracked',
    [
      'diff --git a/internal/alpha/ARCH.md b/internal/alpha/ARCH.md',
      'new file mode 100644',
      '--- /dev/null',
      '+++ b/internal/alpha/ARCH.md',
      '@@ -0,0 +1,4 @@',
      '+# alpha',
      '+',
      '+Três chamadas de `.Save`, todas no caminho de escrita:',
      '+`Store.Put`, `Store.Replace` e `Batch.Flush`.',
      '',
    ].join('\n'),
  ),
};

/**
 * What the recording's daemon answers about each folder's PR (D60): the dcode
 * worktree's branch has one open with CI still running, bizpack is on its base
 * and has none, and eva, in no repository, has no branch to ask about.
 */
export const recordedPRs: Readonly<Record<string, GitPRResponse | string>> = {
  [DCODE]: {
    branch: 'feat/catalogo',
    base: 'main',
    pr: {
      number: 212,
      url: 'https://github.com/aguinelo/dcode/pull/212',
      title: 'Catalogar chamadas de .Save',
      state: 'open',
      draft: false,
      base: 'main',
      head: 'feat/catalogo',
      checks: { state: 'pending', passed: 3, failed: 0, pending: 1, skipped: 0 },
    },
  },
  [BIZPACK]: { branch: 'main', base: 'main' },
  [EVA]: 'the folder is in no git repository',
  // The shop's branch has a PR whose CI passed: the green D61 keeps for what was measured.
  [LOJA]: {
    branch: 'feat/frete',
    base: 'main',
    pr: {
      number: 48,
      url: 'https://github.com/aguinelo/loja/pull/48',
      title: 'Tabela de frete do contrato',
      state: 'open',
      draft: false,
      base: 'main',
      head: 'feat/frete',
      checks: { state: 'pass', passed: 6, failed: 0, pending: 0, skipped: 1 },
    },
  },
};
