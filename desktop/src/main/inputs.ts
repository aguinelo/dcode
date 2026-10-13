import path from 'node:path';
import {
  ApprovalAllow,
  ApprovalAllowAlways,
  ApprovalAllowProject,
  ApprovalAllowSession,
  ApprovalDeny,
  CodeInvalidInput,
  ModeAssist,
  ModeAuto,
  ModePlan,
} from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { NewSessionOptions, Refusal } from '../shared/api';

// What the renderer hands the main process is checked here, at the boundary,
// before any of it reaches the daemon. A bad argument is a refusal the window
// can show — never a throw across IPC, and never a request sent with a hole in
// it for the daemon to guess about.

export type Checked<T> = { ok: true; value: T } | { ok: false; refusal: Refusal };

/** The answers the protocol has to an approval. */
export const DECISIONS: readonly string[] = [ApprovalAllow, ApprovalAllowSession, ApprovalAllowProject, ApprovalAllowAlways, ApprovalDeny];

function invalid(message: string): { ok: false; refusal: Refusal } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

/** How a value reads in a refusal: short, and never the whole of a long one. */
function shown(v: unknown): string {
  if (typeof v === 'string') return v.length > 80 ? `“${v.slice(0, 80)}…”` : `“${v}”`;
  if (v === undefined) return 'nada';
  if (v === null) return 'null';
  return Array.isArray(v) ? 'uma lista' : typeof v === 'object' ? 'um objeto' : `${typeof v} ${String(v)}`;
}

function filled(v: unknown): v is string {
  return typeof v === 'string' && v.trim() !== '';
}

/** An id the daemon gave: a session's, an approval's, a conversation's. */
export function id(v: unknown, what: string): Checked<string> {
  return filled(v) ? { ok: true, value: v } : invalid(`${what} precisa ser um texto não vazio, e veio ${shown(v)}.`);
}

/** Text a person wrote, for a turn or a correction. */
export function text(v: unknown): Checked<string> {
  return filled(v) ? { ok: true, value: v } : invalid(`A mensagem precisa ter texto, e veio ${shown(v)}.`);
}

/** A project's folder: the daemon takes only absolute ones. */
export function workspace(v: unknown): Checked<string> {
  if (!filled(v)) return invalid(`A pasta do projeto precisa ser um caminho, e veio ${shown(v)}.`);
  if (!path.isAbsolute(v)) return invalid(`A pasta do projeto precisa ser um caminho absoluto, e é ${shown(v)}.`);
  return { ok: true, value: v };
}

export function decision(v: unknown): Checked<string> {
  if (typeof v === 'string' && DECISIONS.includes(v)) return { ok: true, value: v };
  return invalid(`A resposta à aprovação precisa ser uma de ${DECISIONS.join(', ')}, e veio ${shown(v)}.`);
}

/** The behavioural modes the daemon switches between (`POST …/mode`). */
export const MODES: readonly string[] = [ModePlan, ModeAssist, ModeAuto];

export function mode(v: unknown): Checked<string> {
  if (typeof v === 'string' && MODES.includes(v)) return { ok: true, value: v };
  return invalid(`O modo precisa ser um de ${MODES.join(', ')}, e veio ${shown(v)}.`);
}

/** The sandboxes a session opens in (`CreateSessionRequest.sandbox_mode`), as the daemon parses them. */
export const SANDBOXES: readonly string[] = ['read-only', 'workspace-write', 'full-access'];

/** What a new session opens on: each option only when given. */
export function sessionOptions(v: unknown): Checked<NewSessionOptions> {
  if (v === undefined) return { ok: true, value: {} };
  if (!isRecord(v)) return invalid(`As opções da sessão nova precisam ser um objeto, e veio ${shown(v)}.`);
  const out: NewSessionOptions = {};
  if (v.model !== undefined) {
    const named = model(v.model);
    if (!named.ok) return named;
    out.model = named.value;
  }
  if (v.sandboxMode !== undefined) {
    if (typeof v.sandboxMode !== 'string' || !SANDBOXES.includes(v.sandboxMode)) {
      return invalid(`O sandbox precisa ser um de ${SANDBOXES.join(', ')}, e veio ${shown(v.sandboxMode)}.`);
    }
    out.sandboxMode = v.sandboxMode;
  }
  return { ok: true, value: out };
}

/** A profile or a model, as the daemon resolves it (`CreateSessionRequest.model`). */
export function model(v: unknown): Checked<string> {
  if (!filled(v) || /\s/.test(v)) return invalid(`O modelo precisa ser um nome sem espaços, e veio ${shown(v)}.`);
  return { ok: true, value: v };
}

/** The conversation to continue: its id and its folder, and nothing else is read. */
export function conversation(v: unknown): Checked<{ id: string; workspace: string }> {
  if (!isRecord(v)) return invalid(`A conversa a continuar precisa vir com id e pasta, e veio ${shown(v)}.`);
  const which = id(v.id, 'O id da conversa');
  if (!which.ok) return which;
  const where = workspace(v.workspace);
  if (!where.ok) return where;
  return { ok: true, value: { id: which.value, workspace: where.value } };
}

/**
 * The id of a recorded conversation to read (D42): the alphabet the daemon
 * names records by (`session.ValidID`) and nothing else. The id becomes part
 * of a path on the daemon, so one carrying a `/`, a `.` or an escape is
 * refused here, before it is sent, rather than left to the daemon to refuse.
 */
export function recordedId(v: unknown): Checked<string> {
  if (typeof v === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(v)) return { ok: true, value: v };
  return invalid(`O id da conversa precisa ter de 1 a 128 letras, dígitos, - ou _, e veio ${shown(v)}.`);
}

/** A position in a recorded conversation's events: a whole number from 1. */
export function position(v: unknown): Checked<number> {
  if (typeof v === 'number' && Number.isSafeInteger(v) && v >= 1) return { ok: true, value: v };
  return invalid(`A posição no histórico precisa ser um número inteiro a partir de 1, e veio ${shown(v)}.`);
}

/** A command to type into a terminal (D37), or to run in the session after `!` (D40). */
export function command(v: unknown): Checked<string> {
  if (!filled(v)) return invalid(`O comando precisa ser um texto não vazio, e veio ${shown(v)}.`);
  if (v.length > 16_384) return invalid(`O comando tem ${v.length} caracteres, e o terminal recebe até 16384.`);
  return { ok: true, value: v };
}

/** Keys the person pressed in a terminal, as xterm.js gave them: escapes and Enter included. */
export function keys(v: unknown): Checked<string> {
  if (typeof v !== 'string' || v === '') return invalid(`O que vai ao terminal precisa ser um texto, e veio ${shown(v)}.`);
  if (v.length > 65_536) return invalid(`O terminal recebe até 65536 caracteres de uma vez, e vieram ${v.length}.`);
  return { ok: true, value: v };
}

/** A terminal's size in cells. */
export function size(cols: unknown, rows: unknown): Checked<{ cols: number; rows: number }> {
  const cell = (n: unknown): n is number => typeof n === 'number' && Number.isInteger(n) && n >= 2 && n <= 1000;
  if (!cell(cols) || !cell(rows)) {
    return invalid(`O tamanho do terminal precisa ser em células, de 2 a 1000, e veio ${shown(cols)} × ${shown(rows)}.`);
  }
  return { ok: true, value: { cols, rows } };
}

/** Text to put on the clipboard. */
export function clip(v: unknown): Checked<string> {
  if (typeof v !== 'string' || v === '') return invalid(`Não há o que copiar: veio ${shown(v)}.`);
  if (v.length > 1_048_576) return invalid(`O texto a copiar tem ${v.length} caracteres, e o app copia até 1048576.`);
  return { ok: true, value: v };
}

/**
 * The folder a session's terminal opens in: the workspace the daemon listed
 * for it. The window names the session, never the folder — a terminal opens
 * only where that session works.
 */
export function workspaceOf(list: ReadonlyArray<{ id: string; workspace: string }> | null, sessionId: string): Checked<string> {
  const found = list?.find((c) => c.id === sessionId);
  if (!found) return invalid(`O daemon não lista a sessão ${shown(sessionId)}: o terminal abre só na pasta de uma sessão que ele conhece.`);
  return workspace(found.workspace);
}
