import path from 'node:path';
import {
  ApprovalAllow,
  ApprovalAllowAlways,
  ApprovalAllowProject,
  ApprovalAllowSession,
  ApprovalDeny,
  CodeInvalidInput,
} from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { Refusal } from '../shared/api';

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

/** The conversation to continue: its id and its folder, and nothing else is read. */
export function conversation(v: unknown): Checked<{ id: string; workspace: string }> {
  if (!isRecord(v)) return invalid(`A conversa a continuar precisa vir com id e pasta, e veio ${shown(v)}.`);
  const which = id(v.id, 'O id da conversa');
  if (!which.ok) return which;
  const where = workspace(v.workspace);
  if (!where.ok) return where;
  return { ok: true, value: { id: which.value, workspace: where.value } };
}
