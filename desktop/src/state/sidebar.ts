// What the sidebar and the bottom bar show, derived from the sessions and the
// person's local preferences. Pure: `now` is passed in, never read.

import { age, firstLine, since } from './format';
import type { Prefs } from './prefs';
import type { SessionView } from './session';
import type { SessionsState } from './sessions';

/** How long a derived title may be, the same limit the daemon's listing uses. */
const TITLE_LIMIT = 72;

/**
 * A session's title: the name a person gave, else the first thing asked.
 * The same rule as the daemon's own listing (internal/session/browse.go).
 */
export function sessionTitle(v: SessionView): string {
  if (v.name.trim()) return v.name.trim();
  const derived = firstLine(v.firstQuestion, TITLE_LIMIT);
  return derived || '(nada perguntado ainda)';
}

export type Mark = 'running' | 'blocked' | 'none';

/** The shape drawn beside a session: a circle while it works, a diamond while it waits for you. */
export function sessionMark(v: SessionView): Mark {
  if (v.state === 'blocked') return 'blocked';
  if (v.state === 'running') return 'running';
  return 'none';
}

export interface Side {
  text: string;
  tone: 'warn' | 'faint';
}

/** What a session row shows on its right: `aprovar`, nothing while running, or its age. */
export function sessionSide(v: SessionView, now: number): Side | null {
  if (v.state === 'blocked') return { text: 'aprovar', tone: 'warn' };
  if (v.state === 'running') return null;
  if (!v.lastAt) return null;
  return { text: age(since(v.lastAt, now)), tone: 'faint' };
}

export function workspaceOf(v: SessionView): string {
  return v.info?.workspace ?? '';
}

export function basename(path: string): string {
  const parts = path.split('/').filter(Boolean);
  return parts[parts.length - 1] ?? path;
}

export interface ProjectView {
  /** The workspace path. Labels, order and collapsing are keyed by it. */
  id: string;
  label: string;
  sessions: SessionView[];
  collapsed: boolean;
  /** Shown only while collapsed: who is waiting, else who is running, else how many. */
  meta: Side | null;
}

function createdAt(v: SessionView): number {
  const t = Date.parse(v.info?.created_at ?? v.lastAt ?? '');
  return Number.isNaN(t) ? 0 : t;
}

function lastActivity(sessions: SessionView[]): number {
  let latest = 0;
  for (const s of sessions) {
    const t = Date.parse(s.lastAt ?? '');
    if (!Number.isNaN(t) && t > latest) latest = t;
  }
  return latest;
}

export function countStates(sessions: readonly SessionView[]): { running: number; blocked: number } {
  let running = 0;
  let blocked = 0;
  for (const s of sessions) {
    if (s.state === 'running') running++;
    else if (s.state === 'blocked') blocked++;
  }
  return { running, blocked };
}

function projectMeta(sessions: SessionView[], collapsed: boolean): Side | null {
  if (!collapsed) return null;
  const { running, blocked } = countStates(sessions);
  if (blocked > 0) return { text: `${blocked} espera`, tone: 'warn' };
  if (running > 0) return { text: `${running} rodando`, tone: 'faint' };
  return { text: String(sessions.length), tone: 'faint' };
}

/**
 * The projects in the sidebar: the person's order first, then the rest by most
 * recent activity; within a project, the newest session first — by creation,
 * so rows do not reshuffle under the pointer while sessions work.
 */
export function projectsOf(state: SessionsState, prefs: Prefs): ProjectView[] {
  const groups = new Map<string, SessionView[]>();
  for (const id of state.order) {
    const v = state.byId[id];
    if (!v) continue;
    const ws = workspaceOf(v);
    const list = groups.get(ws);
    if (list) list.push(v);
    else groups.set(ws, [v]);
  }
  const known = prefs.order.filter((id) => groups.has(id));
  const rest = [...groups.keys()]
    .filter((id) => !known.includes(id))
    .sort((a, b) => lastActivity(groups.get(b) ?? []) - lastActivity(groups.get(a) ?? []));
  return [...known, ...rest].map((id) => {
    const sessions = (groups.get(id) ?? []).slice().sort((a, b) => createdAt(b) - createdAt(a));
    const collapsed = !!prefs.collapsed[id];
    const label = prefs.labels[id]?.trim() || (id ? basename(id) : '(sem projeto)');
    return { id, label, sessions, collapsed, meta: projectMeta(sessions, collapsed) };
  });
}

/** The sessions running or waiting for you, in sidebar order — what ⌘1…⌘9 open. */
export function activeSessions(projects: ProjectView[]): SessionView[] {
  return projects.flatMap((p) => p.sessions).filter((s) => s.state === 'running' || s.state === 'blocked');
}
