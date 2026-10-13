// A recorded conversation's history, read from its record through
// `GET /v1/conversations/{id}/events` (D42) and folded the way a live session's
// events are — so a conversation that ended reads as it did while it ran.
//
// Pure, apart from the page fetcher it is handed: what is read, how it is
// folded and when it is cut are decided here, and tested without a daemon.

import { applyRaw, emptySessions } from './sessions';
import { emptySession, withNote, type SessionView } from './session';
import { isRecord } from '../protocol/validate';

/**
 * The most pages read of one conversation: at the daemon's 5000 events a page,
 * a hundred thousand events. Past it, what was read is shown and the rest is
 * said to be missing — never a thread that ends as if that were all.
 */
export const HISTORY_PAGES = 20;

/** The state the window state is "recorded" in, as the list says it. */
const RECORDED = 'recorded';

export type History =
  | { state: 'loading' }
  /** `cut` is how many events of the conversation were not read: zero when it was read whole. */
  | { state: 'loaded'; view: SessionView; total: number; cut: number }
  | { state: 'failed'; reason: string };

export interface HistoryPage {
  events: unknown[];
  total: number;
  /** Where the next page starts; zero when this one ends the conversation. */
  next: number;
}

/** One page as the daemon answered it, checked before anything reads it; the events themselves are decoded when folded. */
export function decodePage(raw: unknown): { ok: true; page: HistoryPage } | { ok: false; reason: string } {
  const fail = (reason: string) => ({ ok: false as const, reason: `página do histórico: ${reason}` });
  if (!isRecord(raw)) return fail('a resposta não é um objeto');
  const events = raw.events ?? [];
  if (!Array.isArray(events)) return fail('events deveria ser uma lista');
  const whole = (v: unknown): v is number => typeof v === 'number' && Number.isInteger(v) && v >= 0;
  if (!whole(raw.total)) return fail('total deveria ser um número inteiro');
  const next = raw.next ?? 0;
  if (!whole(next)) return fail('next deveria ser um número inteiro');
  return { ok: true, page: { events, total: raw.total, next } };
}

/**
 * Folds a recorded conversation's events into the view a live one has, in the
 * state the list gives it. An event that cannot be read is said in the flow,
 * where it would have been, never dropped.
 */
export function foldHistory(id: string, events: readonly unknown[]): SessionView {
  const folded = applyRaw(emptySessions, events);
  let view = folded.byId[id] ?? emptySession(id);
  for (const p of folded.problems) {
    view = withNote(view, p.seq ?? view.lastSeq, p.at ?? view.lastAt ?? new Date(0).toISOString(), { kind: 'protocol', reason: p.reason });
  }
  // Events of another id would be a daemon answering about a different
  // conversation: said, not merged.
  for (const other of folded.order) {
    if (other !== id) {
      view = withNote(view, view.lastSeq, view.lastAt ?? new Date(0).toISOString(), {
        kind: 'protocol',
        reason: `o histórico trouxe eventos de outra conversa (${other}), que ficaram de fora`,
      });
    }
  }
  return { ...view, state: RECORDED, turn: null, pendingApprovalId: null };
}

/** Asks the daemon for one page, from a position; a refusal is the reason, in the window's language. */
export type PageFetcher = (from: number) => Promise<{ ok: true; value: unknown } | { ok: false; why: string }>;

/**
 * Reads a recorded conversation page by page, up to `pages` of them. Any page
 * that fails fails the whole reading with its reason: half a conversation
 * shown as if it were the conversation is the failure this exists to end.
 */
export async function readHistory(id: string, fetch: PageFetcher, pages = HISTORY_PAGES): Promise<History> {
  const events: unknown[] = [];
  let from = 1;
  let total = 0;
  for (let i = 0; i < pages; i++) {
    const got = await fetch(from);
    if (!got.ok) return { state: 'failed', reason: got.why };
    const read = decodePage(got.value);
    if (!read.ok) return { state: 'failed', reason: `Resposta ilegível do daemon: ${read.reason}` };
    events.push(...read.page.events);
    total = read.page.total;
    if (read.page.next === 0) return { state: 'loaded', view: foldHistory(id, events), total, cut: 0 };
    // A next that does not move forward would read the same page forever.
    if (read.page.next <= from) {
      return { state: 'failed', reason: `Resposta ilegível do daemon: a página de ${from} diz que a próxima começa em ${read.page.next}` };
    }
    from = read.page.next;
  }
  return { state: 'loaded', view: foldHistory(id, events), total, cut: Math.max(0, total - events.length) };
}
