// What the window says of a conversation's branch on GitHub (D60): its PR, the
// checks as a dot and a word, and what a new one would be — prefilled for the
// person to read and change, and never sent without them. The facts come from
// the daemon (`GET /v1/git/pr`); this only words them.

import type { GitChecks, GitPR, GitPRResponse, GitResponse } from '../protocol/generated';
import type { PrDraft } from '../shared/api';
import { filesTouched } from './context';
import type { Entry } from './session';

/** Where a read of the branch's PR stands. A failure keeps the tone of whose it is. */
export type PrState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'failed'; why: string; tone: 'warn' | 'dim' | 'err' }
  | { kind: 'ready'; pr: GitPRResponse };

/** GitHub's caps, which the daemon and the main process hold too. */
export const PR_TITLE_LIMIT = 256;
export const PR_BODY_LIMIT = 65536;

/** The fewest milliseconds between two reads the window starts on its own — on focus, or while CI runs. */
export const PR_MIN_INTERVAL_MS = 30_000;
/** While a PR's checks are still running and the panel shows it, how often it is read again. */
export const PR_PENDING_POLL_MS = 60_000;

/** The tone of a failure, by whose it is: gh not ready, a branch with no PR to have, or a failure. */
export function failureTone(kind: 'gh' | 'refused' | 'failed'): 'warn' | 'dim' | 'err' {
  return kind === 'gh' ? 'warn' : kind === 'refused' ? 'dim' : 'err';
}

/** A PR's checks as a dot's tone, a word and the counts behind it. */
export function checksView(c: GitChecks): { tone: string; word: string; detail: string } {
  const total = c.passed + c.failed + c.pending + c.skipped;
  const skipped = c.skipped > 0 ? ` · ${c.skipped} ${c.skipped === 1 ? 'pulada' : 'puladas'}` : '';
  switch (c.state) {
    case 'pass':
      return { tone: 'ok', word: 'CI verde', detail: `${c.passed} de ${total} passaram${skipped}` };
    case 'fail':
      return { tone: 'err', word: 'CI vermelho', detail: `${c.failed} de ${total} ${c.failed === 1 ? 'falhou' : 'falharam'}` };
    case 'pending':
      return { tone: 'warn', word: 'CI rodando', detail: `${c.pending} de ${total} ainda ${c.pending === 1 ? 'roda' : 'rodam'}` };
    case 'none':
      return { tone: 'faint', word: 'sem CI', detail: total === 0 ? 'nenhuma checagem neste PR' : `nenhuma passou nem falhou${skipped}` };
    default:
      return { tone: 'err', word: 'CI em estado desconhecido', detail: `o daemon disse "${c.state}", que o app não conhece` };
  }
}

/** How the PR stands, in a word. */
export function prStateView(pr: GitPR): { tone: string; word: string } {
  switch (pr.state) {
    case 'open':
      return pr.draft ? { tone: 'dim', word: 'rascunho' } : { tone: 'accent', word: 'aberto' };
    case 'merged':
      return { tone: 'ok', word: 'merged' };
    case 'closed':
      return { tone: 'faint', word: 'fechado' };
    default:
      return { tone: 'err', word: `estado desconhecido: ${pr.state}` };
  }
}

/** The branch has a PR that is still open: there is nothing to create, only to follow. */
export function hasOpenPR(got: GitPRResponse): boolean {
  return got.pr?.state === 'open';
}

/**
 * Why "Criar PR" cannot be pressed now, or null. The daemon refuses the same
 * and more (nothing to propose, no origin); this only spares asking for what
 * the window already knows it would refuse.
 */
export function cannotOpen(git: GitResponse | null, got: GitPRResponse): string | null {
  if (git && git.detached) return 'HEAD está destacado: um PR é de uma branch, e nenhuma está em uso.';
  if (got.branch === got.base) return `A pasta está na própria base (${got.base}): um PR precisa de uma branch sua.`;
  return null;
}

/** What stays out of the PR: the folder's changes that are in no commit. Null when there are none. */
export function uncommittedNote(git: GitResponse | null): string | null {
  if (!git || !git.repository || git.files.length === 0) return null;
  const n = git.files.length;
  return `${n} ${n === 1 ? 'arquivo mudado na pasta fica' : 'arquivos mudados na pasta ficam'} fora do PR: ele leva só o que já está em commits.`;
}

/** The conversation's last answer: what the model said in its last turn, whole. */
function lastAnswer(entries: readonly Entry[]): string {
  let turn: string | null = null;
  const parts: string[] = [];
  for (let i = entries.length - 1; i >= 0; i--) {
    const e = entries[i];
    if (e?.kind !== 'model') continue;
    if (turn === null) turn = e.turnId;
    if (e.turnId !== turn) break;
    parts.unshift(e.text);
  }
  return parts.join('').trim();
}

function cutBytes(text: string, max: number): string {
  const enc = new TextEncoder();
  if (enc.encode(text).length <= max) return text;
  let s = text;
  while (enc.encode(s).length > max - 4) s = s.slice(0, Math.floor(s.length * 0.95));
  return `${s}\n…`;
}

/**
 * What the sheet opens with (D60): the conversation's title on one line, and
 * a body of what the conversation said last and the files it wrote. A start
 * for the person to read and change; the sheet sends what they confirm.
 */
export function draftFor(title: string, entries: readonly Entry[] | undefined, got: GitPRResponse): PrDraft {
  const line = (title.split('\n')[0] ?? '').trim();
  const said = entries ? lastAnswer(entries) : '';
  const files = entries ? filesTouched(entries).filter((f) => !f.failed) : [];
  const list = files.length > 0 ? ['Arquivos que esta conversa mudou:', '', ...files.map((f) => `- \`${f.path}\` (+${f.added} −${f.removed})`)].join('\n') : '';
  const body = [said, list].filter((s) => s !== '').join('\n\n');
  return { title: [...line].slice(0, PR_TITLE_LIMIT).join(''), body: cutBytes(body, PR_BODY_LIMIT), base: got.base, draft: false };
}

/** What is wrong with a draft as typed, or null: the same rules the main process and the daemon hold. */
export function draftProblem(d: PrDraft): string | null {
  if (d.title.trim() === '') return 'O PR precisa de um título.';
  if (/[\r\n]/.test(d.title)) return 'O título é uma linha só.';
  if ([...d.title].length > PR_TITLE_LIMIT) return `O título passa de ${PR_TITLE_LIMIT} caracteres.`;
  if (new TextEncoder().encode(d.body).length > PR_BODY_LIMIT) return `O corpo passa de ${PR_BODY_LIMIT} bytes.`;
  if (d.base.trim() === '') return 'O PR precisa de uma base.';
  if (d.base.startsWith('-') || /[\s~^:?*[\\]/.test(d.base)) return `"${d.base}" não é nome de branch.`;
  return null;
}
