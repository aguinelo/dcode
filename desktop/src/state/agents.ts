// The live agent list of an orchestrator's conversation (orchestration RN-9,
// .p §11): its children, what waits for the person, and the health of main,
// folded from the orchestrator's own events over the daemon's answer to
// /children. Pure, like the rest of state/: no clock and no text read (D8).
// A closed child leaves when the daemon's answer no longer carries it — the
// retention is the daemon's (orchestrator.closed_retention), never a timer
// here — and a failed one never leaves on its own.

import type * as P from '../protocol/generated';
import type { ProtocolEvent } from '../protocol/events';
import { tokens } from './telemetry';

/** The eight states of a child (.p §6), in the order a child moves through them. */
export const CHILD_STATES = ['dispatched', 'working', 'awaiting_person', 'in_review', 'pr', 'merged', 'closed', 'failed'] as const;
export type ChildState = (typeof CHILD_STATES)[number];

export interface AgentList {
  /** The children, in the order they started. */
  children: P.Child[];
  /** What waits for the person, in the order it was asked; an answered one leaves. */
  questions: P.Question[];
  /** The last refusal to merge, per child, until a merge of it completes. */
  refused: Record<string, P.MergeRefused>;
  /** What the forge last said of main; null before the first merge. */
  main: P.MainHealth | null;
}

export const emptyAgents: AgentList = { children: [], questions: [], refused: {}, main: null };

const byStart = (a: P.Child, b: P.Child) => Date.parse(a.started) - Date.parse(b.started) || a.id.localeCompare(b.id);

function upsert(children: P.Child[], c: P.Child): P.Child[] {
  const i = children.findIndex((x) => x.id === c.id);
  if (i < 0) return [...children, c].sort(byStart);
  const next = children.slice();
  next[i] = c;
  return next;
}

/** Changes one known child; an event about a child this list never saw changes nothing. */
function update(list: AgentList, id: string, change: (c: P.Child) => P.Child): AgentList {
  const i = list.children.findIndex((x) => x.id === id);
  if (i < 0) return list;
  const children = list.children.slice();
  children[i] = change(children[i]!);
  return { ...list, children };
}

/** The daemon's answer to GET /sessions/{id}/children: what it does not carry has left. */
export function withChildren(list: AgentList, children: P.Child[]): AgentList {
  return { ...list, children: children.slice().sort(byStart) };
}

/** The daemon's answer to GET /sessions/{id}/questions: only those still unanswered wait. */
export function withQuestions(list: AgentList, questions: P.Question[]): AgentList {
  return { ...list, questions: questions.filter((q) => !q.answered) };
}

/** Folds one orchestrator event; every other event leaves the list as it was. */
export function applyAgentEvent(list: AgentList, ev: ProtocolEvent): AgentList {
  switch (ev.type) {
    case 'child.started':
    case 'child.state':
      return { ...list, children: upsert(list.children, ev.payload.child) };
    case 'review.verdict': {
      const { child_id, ...verdict } = ev.payload;
      return update(list, child_id, (c) => ({ ...c, review: verdict }));
    }
    case 'gate.measured': {
      const { child_id, ...gate } = ev.payload;
      return update(list, child_id, (c) => ({ ...c, gate }));
    }
    case 'question.asked':
      if (ev.payload.answered || list.questions.some((q) => q.id === ev.payload.id)) return list;
      return { ...list, questions: [...list.questions, ev.payload] };
    case 'question.answered':
      return { ...list, questions: list.questions.filter((q) => q.id !== ev.payload.id) };
    case 'merge.refused':
      return { ...list, refused: { ...list.refused, [ev.payload.child_id]: ev.payload } };
    case 'merge.completed': {
      if (!(ev.payload.child_id in list.refused)) return list;
      const refused = { ...list.refused };
      delete refused[ev.payload.child_id];
      return { ...list, refused };
    }
    case 'main.health':
      return { ...list, main: ev.payload };
    default:
      // child.completed carries a turn's end; the child's own state, usage
      // included, arrives in the child.state that follows it.
      return list;
  }
}

export interface AgentNode {
  child: P.Child;
  depth: number;
}

/**
 * The children as a tree, flattened in reading order: those of the
 * orchestrator first, each followed by its own. A child whose parent is
 * neither the orchestrator nor a listed child is drawn at the top rather than
 * hidden.
 */
export function agentTree(list: AgentList, rootId: string): AgentNode[] {
  const ids = new Set(list.children.map((c) => c.id));
  const under = new Map<string, P.Child[]>();
  for (const c of list.children) {
    const parent = c.parent_id !== rootId && ids.has(c.parent_id) && c.parent_id !== c.id ? c.parent_id : rootId;
    under.set(parent, [...(under.get(parent) ?? []), c]);
  }
  const out: AgentNode[] = [];
  const seen = new Set<string>();
  const walk = (id: string, depth: number) => {
    for (const c of under.get(id) ?? []) {
      if (seen.has(c.id)) continue;
      seen.add(c.id);
      out.push({ child: c, depth });
      walk(c.id, depth + 1);
    }
  };
  walk(rootId, 0);
  // A cycle of parents leaves its children unreached: drawn at the top, never dropped.
  for (const c of list.children) if (!seen.has(c.id)) out.push({ child: c, depth: 0 });
  return out;
}

/** The questions waiting for the person about one child. */
export function questionsOf(list: AgentList, childId: string): P.Question[] {
  return list.questions.filter((q) => q.child_id === childId);
}

/** Children that are still at work or waiting: neither merged nor closed. */
export function liveCount(list: AgentList): number {
  return list.children.filter((c) => c.state !== 'merged' && c.state !== 'closed').length;
}

const WAITING: Record<string, string> = {
  approval: 'pede aprovação',
  question: 'tem uma pergunta',
  merge: 'pergunta o merge',
};

/**
 * A child's state as a word and a tone, by D52 and D61: the accent for what
 * is at work, the yellow for what waits for you, red for a failure, grey for
 * what is over. A state the desktop does not know is said in red.
 */
export function childStateView(c: P.Child): { word: string; tone: string; detail: string } {
  const waiting = c.waiting ? (WAITING[c.waiting] ?? `espera: ${c.waiting}`) : '';
  switch (c.state) {
    case 'dispatched':
      return { word: 'despachado', tone: 'dim', detail: 'abrindo a worktree e a sessão' };
    case 'working':
      return { word: 'trabalhando', tone: 'accent', detail: c.returned ? `devolvido ${c.returned}×` : '' };
    case 'awaiting_person':
      return { word: 'esperando você', tone: 'warn', detail: waiting };
    case 'in_review':
      return { word: 'em revisão', tone: 'accent', detail: !c.review ? 'revisando' : !c.gate ? 'medindo o gate' : `gate ${c.gate.result}` };
    case 'pr':
      return { word: 'PR', tone: c.waiting ? 'warn' : 'accent', detail: waiting };
    case 'merged':
      return { word: 'merged', tone: 'dim', detail: '' };
    case 'closed':
      return { word: 'fechado', tone: 'dim', detail: c.reason === 'cancelled' ? 'cancelado' : '' };
    case 'failed':
      return { word: 'falhou', tone: 'err', detail: c.reason ? `o turno terminou em ${c.reason}` : '' };
    default:
      return { word: `estado "${c.state}" desconhecido`, tone: 'err', detail: 'o daemon disse um estado que o app não conhece' };
  }
}

/**
 * A child's PR's CI by D61. The core writes it in GitChecks' grammar
 * (pass, fail, pending, none); its golden says "passing", so both are read.
 */
export function ciView(ci: string): { tone: string; word: string } {
  switch (ci) {
    case 'pass':
    case 'passing':
      return { tone: 'ok', word: 'CI verde' };
    case 'fail':
    case 'failing':
      return { tone: 'err', word: 'CI vermelho' };
    case 'pending':
      return { tone: 'dim', word: 'CI rodando' };
    case '':
    case 'none':
      return { tone: 'faint', word: 'sem CI' };
    default:
      return { tone: 'err', word: `CI "${ci}" desconhecido` };
  }
}

/** The tokens a child spent, summed over its turns: in, out and cache — never money in this slice. */
export function usageLine(u: P.Usage): string {
  const cache = (u.cache_read_tokens ?? 0) + (u.cache_write_tokens ?? 0);
  if (u.input_tokens === 0 && u.output_tokens === 0 && cache === 0) return 'nenhum token ainda';
  const parts = [`${tokens(u.input_tokens)} entrada`, `${tokens(u.output_tokens)} saída`];
  if (cache > 0) parts.push(`${tokens(cache)} cache`);
  return parts.join(' · ');
}

/** What a child owns, in a line: the first path and how many more. */
export function ownsSummary(owns: readonly string[]): string {
  if (owns.length === 0) return 'sem owns';
  return owns.length === 1 ? owns[0]! : `${owns[0]} +${owns.length - 1}`;
}

/** What the forge last said of main, by D61: measured green or red, running grey, unknown faint. */
export function mainHealthView(m: P.MainHealth): { tone: string; word: string } {
  const at = m.commit.slice(0, 7);
  switch (m.health) {
    case 'green':
      return { tone: 'ok', word: `main verde em ${at}` };
    case 'red':
      return { tone: 'err', word: `main vermelha em ${at}` };
    case 'pending':
      return { tone: 'dim', word: `CI da main rodando em ${at}` };
    case 'unknown':
      return { tone: 'faint', word: `main sem CI conhecido em ${at}` };
    default:
      return { tone: 'err', word: `main em estado "${m.health}" desconhecido` };
  }
}
