// One row per orchestrated project (D69, replacing D66's grouping). The
// sidebar lists an orchestrator alone; its children are folded into it, and
// live in its plan (D68) and in its Agentes tab (D64), and a reviewer is never
// a conversation of the list. Pure, like the rest of state/.

import type * as P from '../protocol/generated';
import type { Row, Side } from './sidebar';

export interface Folded {
  /** The rows the sidebar lists: orchestrators, ordinary conversations, and a child no orchestrator is listed for. */
  top: Row[];
  /** The children folded into each orchestrator, by its id, in the order they came. */
  crew: Record<string, Row[]>;
}

/**
 * Folds the children into their orchestrator: the one up their parents, else
 * the orchestrator of their project_id. A reviewer is dropped. A child with
 * no orchestrator listed stays a row of its own, so it can still be opened.
 */
export function foldCrew(rows: readonly Row[]): Folded {
  const byId = new Map(rows.map((r) => [r.id, r]));
  const byProject = new Map<string, Row>();
  for (const r of rows) if (r.role === 'orchestrator' && r.projectId && !r.parentId && !byProject.has(r.projectId)) byProject.set(r.projectId, r);
  const top: Row[] = [];
  const crew: Record<string, Row[]> = {};
  for (const r of rows) {
    if (r.role === 'reviewer') continue;
    if (r.role !== 'child') {
      top.push(r);
      continue;
    }
    const orq = orchestratorOf(r, byId) ?? byProject.get(r.projectId ?? '');
    if (orq) (crew[orq.id] ??= []).push(r);
    else top.push(r);
  }
  return { top, crew };
}

function orchestratorOf(r: Row, byId: ReadonlyMap<string, Row>): Row | undefined {
  const seen = new Set<string>([r.id]);
  for (let up = byId.get(r.parentId ?? ''); up && !seen.has(up.id); up = byId.get(up.parentId ?? '')) {
    if (up.role === 'orchestrator') return up;
    seen.add(up.id);
  }
  return undefined;
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/** The phases the line counts, in the order a child moves through them. */
const PHASES = ['trabalhando', 'em revisão', 'em PR', 'esperando você', 'com falha'] as const;
type Phase = (typeof PHASES)[number];

function phaseOf(c: P.Child): Phase | null {
  if (c.state === 'merged' || c.state === 'closed') return null;
  if (c.waiting || c.state === 'awaiting_person') return 'esperando você';
  if (c.state === 'in_review') return 'em revisão';
  if (c.state === 'pr') return 'em PR';
  if (c.state === 'failed') return 'com falha';
  // dispatched, working, and a state the app does not know: at work.
  return 'trabalhando';
}

function line(agents: number, phases: readonly Phase[]): Side {
  if (agents === 0) return { text: 'nenhum agente em curso', tone: 'faint' };
  const parts = [plural(agents, 'agente', 'agentes')];
  for (const p of PHASES) {
    const n = phases.filter((x) => x === p).length;
    if (n > 0) parts.push(`${n} ${p}`);
  }
  return { text: parts.join(' · '), tone: phases.includes('esperando você') ? 'warn' : 'faint' };
}

/**
 * The orchestrator's line in the sidebar: its agents in course and how many
 * are in each phase, e.g. "3 agentes · 1 em revisão · 2 esperando você". From
 * the orchestrator's own list when the window has it (D64), else from the
 * children's rows — running is at work, blocked is waiting for you, and an
 * ended one is not counted. Null with no child at all.
 */
export function crewLine(children: readonly P.Child[] | undefined, rows: readonly Row[]): Side | null {
  if (children && children.length > 0) {
    const phases = children.flatMap((c) => phaseOf(c) ?? []);
    return line(phases.length, phases);
  }
  if (rows.length === 0) return null;
  // A row says less than the list: an idle child counts, with no phase named.
  const live = rows.filter((r) => r.state !== 'recorded');
  const phases = live.flatMap((r): Phase[] => (r.state === 'blocked' ? ['esperando você'] : r.state === 'running' ? ['trabalhando'] : []));
  return line(live.length, phases);
}
