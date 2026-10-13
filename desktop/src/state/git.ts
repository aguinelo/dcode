// What the window says of a conversation's folder in git (D41): which session
// it is, where the folder is, the branch now, whether it is a worktree and of
// which checkout, and what changed. The facts come from the daemon
// (`GET /v1/git`); this only words them. Every failure is a sentence the panel
// shows — never an empty list that reads as "nothing changed".

import type { GitFile, GitResponse } from '../protocol/generated';
import type { SessionView } from './session';
import { basename, type Row } from './sidebar';

/** Where a read of the folder's git stands. */
export type GitState = { kind: 'idle' } | { kind: 'loading' } | { kind: 'failed'; why: string } | { kind: 'ready'; git: GitResponse };

/** How a change reads, one letter in the list and a word on hover — the letters git status uses. */
export const STATUS: Record<string, { letter: string; word: string; tone: string }> = {
  modified: { letter: 'M', word: 'modificado', tone: 'warn' },
  added: { letter: 'A', word: 'novo, já preparado (git add)', tone: 'ok' },
  deleted: { letter: 'D', word: 'apagado', tone: 'err' },
  untracked: { letter: 'U', word: 'novo, fora do git ainda', tone: 'ok' },
  conflicted: { letter: '!', word: 'em conflito', tone: 'err' },
};

export function statusOf(status: string): { letter: string; word: string; tone: string } {
  return STATUS[status] ?? { letter: '·', word: `estado que o app não conhece: ${status}`, tone: 'faint' };
}

/** A session's id as the window shows it: the first eight, the whole on hover and copy. */
export function shortId(id: string): string {
  return id.length > 8 ? id.slice(0, 8) : id;
}

export interface GitSummary {
  files: number;
  added: number;
  removed: number;
  /** Files git counts no lines for: binary. Said, not added up as zero. */
  binary: number;
}

export function summarize(files: readonly GitFile[]): GitSummary {
  let added = 0;
  let removed = 0;
  let binary = 0;
  for (const f of files) {
    added += f.added;
    removed += f.removed;
    if (f.binary) binary++;
  }
  return { files: files.length, added, removed, binary };
}

/** The branch as it is now, or what HEAD is when it is on none. */
export function branchNow(git: GitResponse): string {
  if (git.detached) return git.head ? `HEAD destacado em ${git.head}` : 'HEAD destacado';
  return git.branch ?? '—';
}

/**
 * The branch the session started on, when it is not the one checked out now:
 * `Session.branch` is read once, at the session's start, and the folder may
 * have moved since. Null when they agree, or when either is not known.
 */
export function startedElsewhere(git: GitResponse, startedOn: string | undefined): string | null {
  if (!startedOn || git.detached || !git.branch) return null;
  return startedOn === git.branch ? null : startedOn;
}

/** "worktree de dcode" — the checkout a linked worktree belongs to, by its folder's name. */
export function worktreeWord(git: GitResponse): string | null {
  if (!git.worktree) return null;
  return git.main_checkout ? `worktree de ${basename(git.main_checkout)}` : 'worktree';
}

/**
 * The top of the repository, when it is not the folder itself — a session
 * opened in a subfolder writes paths the list shows from the top.
 */
export function rootIfElsewhere(git: GitResponse, workspace: string): string | null {
  if (!git.root) return null;
  const trim = (p: string) => p.replace(/\/+$/, '');
  return trim(git.root) === trim(workspace) ? null : git.root;
}

/** "3 arquivos · +12 −4", with binary files said apart. */
export function summaryText(s: GitSummary): string {
  if (s.files === 0) return 'nada mudou desde o último commit';
  const files = `${s.files} ${s.files === 1 ? 'arquivo' : 'arquivos'}`;
  const bin = s.binary > 0 ? ` · ${s.binary} ${s.binary === 1 ? 'binário' : 'binários'}` : '';
  return `${files} · +${s.added} −${s.removed}${bin}`;
}


export type DiffLineKind = 'meta' | 'hunk' | 'add' | 'del' | 'ctx' | 'note';

export interface DiffLine {
  kind: DiffLineKind;
  text: string;
}

/**
 * A unified diff, line by line, for drawing. The file headers (`diff --git`,
 * `index`, `---`, `+++`) are meta, kept apart from the lines that changed so a
 * `+++` header is never counted as an added line.
 */
export function diffLines(text: string): DiffLine[] {
  const out: DiffLine[] = [];
  let inHunk = false;
  for (const line of text.replace(/\n$/, '').split('\n')) {
    if (line.startsWith('@@')) {
      inHunk = true;
      out.push({ kind: 'hunk', text: line });
    } else if (!inHunk) {
      out.push({ kind: 'meta', text: line });
    } else if (line.startsWith('+')) {
      out.push({ kind: 'add', text: line });
    } else if (line.startsWith('-')) {
      out.push({ kind: 'del', text: line });
    } else if (line.startsWith('\\')) {
      out.push({ kind: 'note', text: line });
    } else if (line.startsWith('diff --git')) {
      inHunk = false;
      out.push({ kind: 'meta', text: line });
    } else {
      out.push({ kind: 'ctx', text: line });
    }
  }
  return out;
}

/**
 * What, changing, makes the folder worth reading again: a tool call that
 * finished, or the conversation changing state — not each fragment of text,
 * which changes nothing on disk.
 */
export function movedKey(row: Pick<Row, 'state' | 'lastAt'>, session: SessionView | undefined): string {
  if (!session) return `${row.state}:${row.lastAt ?? ''}`;
  let done = 0;
  for (const e of session.entries) if (e.kind === 'tool' && e.call.result) done++;
  return `${session.state}:${done}`;
}

/** What the panel says when the folder's git could not be read: the daemon's reason, whole. */
export function failedText(why: string): string {
  return `Não foi possível ler o git da pasta: ${why}`;
}
