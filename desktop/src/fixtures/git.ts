// What the recording's daemon answers about each project's git (D41): the
// dcode folder is a linked worktree with work in it, bizpack a clean checkout,
// and eva no repository at all — the three things the panel has to tell apart.
// One answer per folder, as the daemon gives it: the conversations in a folder
// share its branch, whatever branch each started on.

import type { GitDiffResponse, GitResponse } from '../protocol/generated';

const DCODE = '/Users/aguinelo/work/dcode';
const BIZPACK = '/Users/aguinelo/work/bizpack';
const EVA = '/Users/aguinelo/work/eva';

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
