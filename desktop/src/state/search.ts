// What ⌘K finds: every conversation in the sidebar, by its title and its
// project, grouped the way the design groups them. Pure: the query, the scope
// and `now` are arguments.

import { age, since } from './format';
import type { Tone } from './flow';
import { sealWord } from './pane';
import type { ProjectView, Row } from './sidebar';

/** Where the search looks: everything, what is running or waiting, or the open session's project. */
export type Scope = 'todas' | 'ativas' | 'projeto';

export const SCOPES: readonly Scope[] = ['todas', 'ativas', 'projeto'];

export interface SearchGroup {
  label: string;
  rows: Row[];
}

/** Case and accents do not count: "readme" finds "README", "sessao" finds "sessão". */
export function fold(s: string): string {
  return s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase();
}

function isActive(r: Row): boolean {
  return r.state === 'running' || r.state === 'blocked';
}

/**
 * The groups ⌘K shows: "Ativas" first — running or waiting for you — then one
 * per project, in the sidebar's order. Every word typed must be in the title or
 * in the project (its label or its path). `project` is the open session's
 * workspace, which the `projeto` scope keeps to.
 */
export function searchGroups(projects: readonly ProjectView[], query: string, scope: Scope, project: string | null): SearchGroup[] {
  const words = fold(query).split(/\s+/).filter(Boolean);
  const active: Row[] = [];
  const groups: SearchGroup[] = [];
  for (const p of projects) {
    if (scope === 'projeto' && p.id !== project) continue;
    const place = `${fold(p.label)} ${fold(p.id)}`;
    const rest: Row[] = [];
    for (const r of p.rows) {
      const hay = `${fold(r.title)} ${place}`;
      if (!words.every((w) => hay.includes(w))) continue;
      if (isActive(r)) active.push(r);
      else if (scope !== 'ativas') rest.push(r);
    }
    if (rest.length > 0) groups.push({ label: p.label, rows: rest });
  }
  return active.length > 0 ? [{ label: 'Ativas', rows: active }, ...groups] : groups;
}

/** What a row says on its right in the search: `rodando`, `aprovar`, or its age. */
export function searchSide(r: Row, now: number): { text: string; tone: Tone } {
  if (r.state === 'blocked') return { text: 'aprovar', tone: 'warn' };
  if (r.state === 'running') return { text: 'rodando', tone: 'dim' };
  return { text: r.lastAt ? age(since(r.lastAt, now)) : '', tone: 'faint' };
}

/** The preview's long state: what it is doing, else how its last turn was sealed (D13). */
export function longState(r: Row): { text: string; tone: Tone } {
  if (r.state === 'blocked') return { text: 'Esperando aprovação', tone: 'warn' };
  if (r.state === 'running') return { text: 'Rodando', tone: 'dim' };
  const seal = sealWord(r.verification);
  if (seal) return seal;
  if (r.state === 'recorded') return { text: 'Terminada — abrir continua numa sessão nova', tone: 'faint' };
  return { text: 'Parada, esperando mensagem', tone: 'faint' };
}

const LAST_EVENT: Readonly<Record<string, string>> = {
  'session.created': 'sessão aberta',
  'session.resumed': 'conversa continuada',
  'session.renamed': 'renomeada',
  'turn.started': 'turno começou',
  'turn.steered': 'turno redirecionado',
  'turn.completed': 'turno terminou',
  'tool.requested': 'ferramenta pedida',
  'tool.completed': 'ferramenta terminou',
  'tool.approval_required': 'pediu aprovação',
  'tool.approval_resolved': 'aprovação respondida',
  'session.error': 'erro',
};

/** The last event as the preview says it, with how long ago. An event it does not name is shown as it came. */
export function lastEventText(r: Row, now: number): string {
  if (!r.lastEvent) return '';
  const what = LAST_EVENT[r.lastEvent] ?? r.lastEvent;
  return r.lastAt ? `${what} · ${age(since(r.lastAt, now))}` : what;
}
