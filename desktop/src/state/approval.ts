// The answers an approval offers, as the TUI offers them (D39, after
// `approvalKeys` and `standingScope` in internal/tui/render.go).
//
// Only a network crossing can be answered for longer than the session: a
// standing grant to write outside the workspace would be a grant over the whole
// filesystem given once, for a reason nobody records. And a network crossing
// does not offer "this session" — the TUI gives it the project and for good
// instead. Allowing is always an explicit key; ↵ and esc deny (D22).

import type { ApprovalRequest } from '../protocol/generated';
import type { SessionView } from './session';
import { ApprovalAllow, ApprovalAllowAlways, ApprovalAllowProject, ApprovalAllowSession, ApprovalDeny } from '../protocol/generated';

export interface Choice {
  key: string;
  label: string;
  decision: string;
  /** What else gives it: ↵ and esc on deny. */
  hint: string;
  /** Outlives the session: recorded in the user's configuration. */
  standing: boolean;
}

/** The crossings whose answer can outlive the session. */
export function standingScope(request: { boundary_crossed: string }): boolean {
  return request.boundary_crossed === 'network';
}

const DENY = { label: 'Negar', decision: ApprovalDeny, hint: '↵ esc', standing: false };

export function choicesFor(request: { boundary_crossed: string }): Choice[] {
  const list = standingScope(request)
    ? [
        { label: 'Permitir uma vez', decision: ApprovalAllow, hint: '', standing: false },
        { label: 'Permitir neste projeto', decision: ApprovalAllowProject, hint: '', standing: true },
        { label: 'Permitir sempre', decision: ApprovalAllowAlways, hint: '', standing: true },
        DENY,
      ]
    : [{ label: 'Permitir uma vez', decision: ApprovalAllow, hint: '', standing: false }, { label: 'Permitir nesta sessão', decision: ApprovalAllowSession, hint: '', standing: false }, DENY];
  return list.map((c, i) => ({ ...c, key: String(i + 1) }));
}

/** The decision a key gives while this approval waits, or null for a key that gives none. */
export function answerForKey(key: string, request: { boundary_crossed: string }): string | null {
  if (key === 'Enter' || key === 'Escape') return ApprovalDeny;
  return choicesFor(request).find((c) => c.key === key)?.decision ?? null;
}

/** The request a session waits on, from its flow; null when it waits on none. */
export function pendingRequest(view: SessionView | undefined): ApprovalRequest | null {
  const id = view?.pendingApprovalId;
  if (!id) return null;
  for (const e of view.entries) if (e.kind === 'approval' && e.request.approval_id === id) return e.request;
  return null;
}
