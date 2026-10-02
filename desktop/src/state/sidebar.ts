// What the sidebar and the bottom bar show, derived from the conversations and
// the person's local preferences. Pure: `now` is passed in, never read.
//
// A row comes from the daemon's list of conversations when the window is
// connected, and from a recorded session's events in fixture mode; past this
// file the two read the same.

import type { Conversation } from '../protocol/generated';
import { age, firstLine, since } from './format';
import type { Prefs } from './prefs';
import type { SessionView } from './session';
import type { SessionsState } from './sessions';

/** How long a derived title may be, the same limit the daemon's listing uses. */
const TITLE_LIMIT = 72;

/** What a conversation without a title is called. */
export const UNTITLED = '(nada perguntado ainda)';

/**
 * A session's title: the name a person gave, else the first thing asked.
 * The same rule as the daemon's own listing (internal/session/browse.go).
 */
export function sessionTitle(v: SessionView): string {
  if (v.name.trim()) return v.name.trim();
  const derived = firstLine(v.firstQuestion, TITLE_LIMIT);
  return derived || UNTITLED;
}

/** One line of the sidebar: a conversation, whatever it was read from. */
export interface Row {
  id: string;
  title: string;
  /** The workspace path, which is the project it is listed under. */
  workspace: string;
  /** The wire's state: idle, running or blocked, or `recorded` for one that ended. */
  state: string;
  /** When it started, in ms, for the order within a project. */
  started: number;
  /** The instant anything last happened in it, for the project order and the age shown. */
  lastAt: string | null;
  /** What the search's preview shows, where the source says it. */
  branch?: string;
  model?: string;
  turns?: number;
  /** The seal of the last completed turn (D13). */
  verification?: string;
  /** The last event that moved it along. */
  lastEvent?: string;
}

function instant(s: string | null | undefined): number {
  const t = Date.parse(s ?? '');
  return Number.isNaN(t) ? 0 : t;
}

export function rowOfSession(v: SessionView): Row {
  return {
    id: v.id,
    title: sessionTitle(v),
    workspace: v.info?.workspace ?? '',
    state: v.state,
    started: instant(v.info?.created_at ?? v.lastAt),
    lastAt: v.lastAt,
    branch: v.info?.branch,
    model: v.info?.model,
    turns: v.turns,
  };
}

/** The rows of the sessions a recording holds, in the order their first event arrived. */
export function rowsOfSessions(state: SessionsState): Row[] {
  return state.order.flatMap((id) => {
    const v = state.byId[id];
    return v ? [rowOfSession(v)] : [];
  });
}

export function rowOfConversation(c: Conversation): Row {
  return {
    id: c.id,
    title: firstLine(c.title, TITLE_LIMIT) || UNTITLED,
    workspace: c.workspace,
    state: c.live ? c.state : 'recorded',
    started: instant(c.started),
    lastAt: c.last_activity || null,
    branch: c.branch,
    model: c.model,
    turns: c.turns,
    verification: c.verification,
    lastEvent: c.last_event,
  };
}

export type Mark = 'running' | 'blocked' | 'none';

/** The shape drawn beside a row: a circle while it works, a diamond while it waits for you. */
export function rowMark(r: Row): Mark {
  if (r.state === 'blocked') return 'blocked';
  if (r.state === 'running') return 'running';
  return 'none';
}

export interface Side {
  text: string;
  tone: 'warn' | 'faint';
}

/** What a row shows on its right: `aprovar`, nothing while running, or its age. */
export function rowSide(r: Row, now: number): Side | null {
  if (r.state === 'blocked') return { text: 'aprovar', tone: 'warn' };
  if (r.state === 'running') return null;
  if (!r.lastAt) return null;
  return { text: age(since(r.lastAt, now)), tone: 'faint' };
}

export function basename(path: string): string {
  const parts = path.split('/').filter(Boolean);
  return parts[parts.length - 1] ?? path;
}

export interface ProjectView {
  /** The workspace path. Labels, order and collapsing are keyed by it. */
  id: string;
  label: string;
  rows: Row[];
  collapsed: boolean;
  /** Shown only while collapsed: who is waiting, else who is running, else how many. */
  meta: Side | null;
}

function lastActivity(rows: Row[]): number {
  let latest = 0;
  for (const r of rows) latest = Math.max(latest, instant(r.lastAt));
  return latest;
}

export function countStates(rows: readonly Row[]): { running: number; blocked: number } {
  let running = 0;
  let blocked = 0;
  for (const r of rows) {
    if (r.state === 'running') running++;
    else if (r.state === 'blocked') blocked++;
  }
  return { running, blocked };
}

function projectMeta(rows: Row[], collapsed: boolean): Side | null {
  if (!collapsed) return null;
  const { running, blocked } = countStates(rows);
  if (blocked > 0) return { text: `${blocked} espera`, tone: 'warn' };
  if (running > 0) return { text: `${running} rodando`, tone: 'faint' };
  return { text: String(rows.length), tone: 'faint' };
}

/**
 * The projects in the sidebar: the person's order first, then the rest by most
 * recent activity; within a project, the newest first — by start, so rows do
 * not reshuffle under the pointer while sessions work.
 */
export function projectsOf(rows: readonly Row[], prefs: Prefs): ProjectView[] {
  const groups = new Map<string, Row[]>();
  for (const r of rows) {
    const list = groups.get(r.workspace);
    if (list) list.push(r);
    else groups.set(r.workspace, [r]);
  }
  const known = prefs.order.filter((id) => groups.has(id));
  const rest = [...groups.keys()]
    .filter((id) => !known.includes(id))
    .sort((a, b) => lastActivity(groups.get(b) ?? []) - lastActivity(groups.get(a) ?? []));
  return [...known, ...rest].map((id) => {
    const sorted = (groups.get(id) ?? []).slice().sort((a, b) => b.started - a.started);
    const collapsed = !!prefs.collapsed[id];
    const label = prefs.labels[id]?.trim() || (id ? basename(id) : '(sem projeto)');
    return { id, label, rows: sorted, collapsed, meta: projectMeta(sorted, collapsed) };
  });
}

/** The rows running or waiting for you, in sidebar order — what ⌘1…⌘9 open. */
export function activeRows(projects: ProjectView[]): Row[] {
  return projects.flatMap((p) => p.rows).filter((r) => r.state === 'running' || r.state === 'blocked');
}
