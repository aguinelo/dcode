// What the Crew look's panel of sessions and its other sections show (D33),
// derived from the conversations the daemon lists and the sessions the window
// follows. Pure: the query and `now` are arguments.

import { fold } from './search';
import type { SessionView } from './session';
import type { SessionsState } from './sessions';
import type { ProjectView, Row } from './sidebar';

/** The sections of the icon rail, in the rail's order. */
export type CrewSection = 'painel' | 'agenda' | 'memoria' | 'skills' | 'apps' | 'conhecimento' | 'config';

export const CREW_SECTIONS: readonly CrewSection[] = ['painel', 'agenda', 'memoria', 'skills', 'apps', 'conhecimento', 'config'];

export function isCrewSection(s: string | null): s is CrewSection {
  return s !== null && (CREW_SECTIONS as readonly string[]).includes(s);
}

/** A conversation with nothing for longer than this goes under "Mais antigas". */
export const OLDER_AFTER_MS = 7 * 24 * 60 * 60 * 1000;

export interface SessionGroup {
  /** Stable across renders: `ativas`, `mais-antigas`, or `p:` and the workspace. */
  key: string;
  label: string;
  rows: Row[];
  /** "Mais antigas" opens on demand; the others are always open. */
  collapsible: boolean;
}

function isActive(r: Row): boolean {
  return r.state === 'running' || r.state === 'blocked';
}

function instant(s: string | null): number {
  const t = Date.parse(s ?? '');
  return Number.isNaN(t) ? 0 : t;
}

/**
 * The groups of the sessions panel: "Ativas" first — running or waiting for
 * you, the waiting ones on top —, then one per project with what moved in the
 * last week, in the sidebar's order, then "Mais antigas", the most recent
 * first. A conversation with no instant at all is old: nothing says it is not.
 * Every word typed must be in the title or the project, as in ⌘K (D27).
 */
export function sessionGroups(projects: readonly ProjectView[], query: string, now: number): SessionGroup[] {
  const words = fold(query).split(/\s+/).filter(Boolean);
  const active: Row[] = [];
  const older: Row[] = [];
  const groups: SessionGroup[] = [];
  for (const p of projects) {
    const place = `${fold(p.label)} ${fold(p.id)}`;
    const recent: Row[] = [];
    for (const r of p.rows) {
      if (!words.every((w) => `${fold(r.title)} ${place}`.includes(w))) continue;
      if (isActive(r)) active.push(r);
      else if (r.lastAt && now - instant(r.lastAt) <= OLDER_AFTER_MS) recent.push(r);
      else older.push(r);
    }
    if (recent.length > 0) groups.push({ key: `p:${p.id}`, label: p.label, rows: recent, collapsible: false });
  }
  // Stable: within each state the sidebar's order stays.
  active.sort((a, b) => Number(b.state === 'blocked') - Number(a.state === 'blocked'));
  older.sort((a, b) => instant(b.lastAt) - instant(a.lastAt));
  return [
    ...(active.length > 0 ? [{ key: 'ativas', label: 'Ativas', rows: active, collapsible: false }] : []),
    ...groups,
    ...(older.length > 0 ? [{ key: 'mais-antigas', label: 'Mais antigas', rows: older, collapsible: true }] : []),
  ];
}

/** How many rows the groups hold, which the search field shows. */
export function groupCount(groups: readonly SessionGroup[]): number {
  return groups.reduce((n, g) => n + g.rows.length, 0);
}

/** The state of a row in a word, as the seal words go (D13): what it is doing, else nothing. */
export function stateWord(state: string): string {
  if (state === 'running') return 'rodando';
  if (state === 'blocked') return 'esperando você';
  if (state === 'recorded') return 'terminada';
  return 'parada';
}

/**
 * How full the context is, in whole percent, once a turn has reported it
 * (D15); null before, never a zero that was not measured.
 */
export function contextPercent(v: SessionView | undefined): number | null {
  const window = v?.info?.context_window ?? 0;
  if (!v || v.contextTokens === null || window <= 0) return null;
  return Math.min(100, Math.floor((100 * v.contextTokens) / window));
}

export interface SkillSeen {
  name: string;
  /** What the skill said it is for, from the last time it entered a turn. */
  whenToUse: string;
  /** The conversations, among those the window follows, it entered. */
  sessions: number;
}

/**
 * The skills that entered a turn in the conversations this window follows
 * (`skill.loaded`), by name. Only what the events said: a skill that exists
 * and never entered a followed turn is not here — that list is the daemon's.
 */
export function skillsSeen(state: SessionsState): SkillSeen[] {
  const byName = new Map<string, { whenToUse: string; ids: Set<string> }>();
  for (const id of state.order) {
    const v = state.byId[id];
    if (!v) continue;
    for (const e of v.entries) {
      if (e.kind !== 'note' || e.note.kind !== 'skill') continue;
      const seen = byName.get(e.note.name) ?? { whenToUse: '', ids: new Set<string>() };
      seen.ids.add(id);
      if (e.note.whenToUse) seen.whenToUse = e.note.whenToUse;
      byName.set(e.note.name, seen);
    }
  }
  return [...byName.entries()]
    .map(([name, s]) => ({ name, whenToUse: s.whenToUse, sessions: s.ids.size }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * A skill a session in the workspace has, as the Skills section draws it —
 * what `GET /v1/skills` carries (N6), and only that: never the skill's body.
 */
export interface SkillView {
  name: string;
  whenToUse: string;
  /** `user` or `project`: where its file lives. */
  source: string;
  /** Asks a person before loading, because it reaches for the boundary. */
  held: boolean;
}

export interface SkillsView {
  /** Sessions in the workspace index skills at all. */
  enabled: boolean;
  skills: SkillView[];
  /** Files trimmed or not loaded, with why. */
  notices: string[];
}

/** One memory, as the Memória section draws it — what `GET /v1/memory` carries (N6). */
export interface MemoryView {
  kind: string;
  subject: string;
  body: string;
  /** The commit it was true at is gone: marked, never dropped. */
  stale: boolean;
  /** A session reads it. */
  shown: boolean;
}

export interface MemoryListView {
  /** The file, relative to the workspace. */
  path: string;
  exists: boolean;
  enabled: boolean;
  entries: MemoryView[];
  /** Blocks that looked like a memory and were not, and a file that could not be read. */
  problems: string[];
}
