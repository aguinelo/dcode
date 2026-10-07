// What the session's flow shows, derived from its entries. Pure and
// Portuguese-facing: tool results are summarised from the fields the tool
// reported, never by reading numbers back out of its output text.

import type * as P from '../protocol/generated';
import { firstLine, plural } from './format';
import type { Entry, ToolCall } from './session';

export type Block =
  | { kind: 'tools'; calls: ToolCall[] }
  | { kind: 'delegation'; calls: ToolCall[] }
  | { kind: 'entry'; entry: Exclude<Entry, { kind: 'tool' }> };

function isDelegated(call: ToolCall): boolean {
  return call.name.toLowerCase() === 'explore';
}

/**
 * Groups the flow into what is drawn together.
 *
 * Adjacent tool calls are one block, drawn closer than the rest. Two or more
 * adjacent `explore` calls are one delegation — the events arrive in emission
 * order and a delegated batch is emitted together, the same reading the TUI
 * makes. A call whose crossing is put to the person is drawn by its approval
 * card while it waits, and after the card once answered, so the story reads in
 * the order it happened.
 */
export function flowBlocks(entries: readonly Entry[]): Block[] {
  const asked = new Map<string, Extract<Entry, { kind: 'approval' }>>();
  for (const e of entries) if (e.kind === 'approval') asked.set(e.request.tool_call_id, e);

  const ordered: Entry[] = [];
  const held = new Map<string, Entry>();
  for (const e of entries) {
    if (e.kind === 'tool' && asked.has(e.call.id)) {
      held.set(e.call.id, e);
      continue;
    }
    ordered.push(e);
    if (e.kind === 'approval' && e.decision !== null) {
      const call = held.get(e.request.tool_call_id);
      if (call) ordered.push(call);
    }
  }

  const blocks: Block[] = [];
  let run: ToolCall[] = [];
  const flush = () => {
    let plain: ToolCall[] = [];
    let i = 0;
    while (i < run.length) {
      let j = i;
      while (j < run.length && isDelegated(run[j] as ToolCall)) j++;
      if (j - i >= 2) {
        if (plain.length > 0) blocks.push({ kind: 'tools', calls: plain });
        plain = [];
        blocks.push({ kind: 'delegation', calls: run.slice(i, j) });
        i = j;
      } else {
        // One child is a call, not a delegation: a frame and a count around it
        // would repeat what its own line already says.
        plain.push(run[i] as ToolCall);
        i++;
      }
    }
    if (plain.length > 0) blocks.push({ kind: 'tools', calls: plain });
    run = [];
  };
  for (const e of ordered) {
    if (e.kind === 'tool') {
      run.push(e.call);
      continue;
    }
    flush();
    blocks.push({ kind: 'entry', entry: e });
  }
  flush();
  return blocks;
}

function field(input: unknown, name: string): string {
  if (typeof input !== 'object' || input === null) return '';
  const v = (input as Record<string, unknown>)[name];
  return typeof v === 'string' ? v.replace(/\s*\n\s*/g, ' ').trim() : '';
}

function list(input: unknown, name: string): string[] {
  if (typeof input !== 'object' || input === null) return [];
  const v = (input as Record<string, unknown>)[name];
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

/** What a call acts on, as its line names it: the pattern searched, the file, the command. */
export function toolTarget(call: ToolCall): string {
  const i = call.input;
  switch (call.name.toLowerCase()) {
    case 'grep':
    case 'glob':
      return field(i, 'pattern') || field(i, 'path');
    case 'bash':
      return field(i, 'command');
    case 'fetch':
      return field(i, 'url');
    default:
      return field(i, 'path') || field(i, 'pattern') || field(i, 'command') || field(i, 'url');
  }
}

export type Tone = 'ok' | 'err' | 'accent' | 'dim' | 'faint' | 'warn';

export interface ToolLineView {
  glyph: string;
  glyphTone: Tone;
  name: string;
  target: string;
  summary: string;
  summaryTone: Tone;
}

function signed(added: number, removed: number): string {
  return `+${added} −${removed}`;
}

/** The one-line result of a finished call, from what the tool reported. */
export function toolSummary(call: ToolCall): string {
  const d = call.result;
  if (!d) return runningSummary(call);
  if (!d.ok) return firstLine(d.output) || 'falhou';
  const added = d.added ?? 0;
  const removed = d.removed ?? 0;
  const lines = d.lines ?? 0;
  const files = d.files ?? 0;
  switch (call.name.toLowerCase()) {
    case 'read':
      if (lines > 0) return `${plural(lines, 'linha', 'linhas')}${d.truncated ? ' (cortado)' : ''}`;
      break;
    case 'edit':
      return signed(added, removed);
    case 'write':
      if (removed === 0 && added > 0) return `criado, ${plural(added, 'linha', 'linhas')}`;
      return signed(added, removed);
    case 'glob':
      return plural(files, 'arquivo', 'arquivos');
    case 'grep':
      if (lines === 0) return 'nenhum match';
      return `${plural(lines, 'match', 'matches')} · ${plural(files, 'arquivo', 'arquivos')}`;
    case 'bash':
      if (d.has_exit) return `exit ${d.exit_code ?? 0}`;
      break;
    case 'explore':
      if (files > 0) return `leu ${files}`;
      break;
  }
  return firstLine(d.output) || 'ok';
}

function runningSummary(call: ToolCall): string {
  const p = call.progress;
  if (p && p.kind === 'files') {
    return p.total ? `${p.done} de ${plural(p.total, 'arquivo', 'arquivos')}` : `${plural(p.done, 'arquivo', 'arquivos')}…`;
  }
  // Running with nothing counted: an ellipsis, never an invented number.
  return '…';
}

export function toolLine(call: ToolCall): ToolLineView {
  const summary = toolSummary(call);
  switch (call.status) {
    case 'ok':
      return { glyph: '✓', glyphTone: 'ok', name: call.name, target: toolTarget(call), summary, summaryTone: 'dim' };
    case 'failed':
      return { glyph: '⊘', glyphTone: 'err', name: call.name, target: toolTarget(call), summary, summaryTone: 'err' };
    default:
      return { glyph: '●', glyphTone: 'accent', name: call.name, target: toolTarget(call), summary, summaryTone: 'faint' };
  }
}

export interface ChildView {
  id: string;
  name: string;
  owns: string;
  status: 'ok' | 'failed' | 'running';
  /** How much of the bar is filled, 0–1, or null when nothing measures it. */
  bar: number | null;
  meta: string;
  /** Why a child did not answer, in the daemon's words, for a tooltip. */
  reason: string;
}

export interface DelegationView {
  count: number;
  /** Children that came back, answered or not. */
  finished: number;
  running: number;
  /** Every child declared what it may write, and no two declared the same path. */
  disjoint: boolean;
  children: ChildView[];
}

function lastSegment(path: string): string {
  const parts = path.split('/').filter((p) => p && p !== '.');
  return parts[parts.length - 1] ?? '';
}

function overlaps(a: string, b: string): boolean {
  const x = a.replace(/\/+$/, '');
  const y = b.replace(/\/+$/, '');
  return x === y || x.startsWith(`${y}/`) || y.startsWith(`${x}/`);
}

/**
 * The children of a delegation. The wire carries no name for a child — explore
 * takes a task, a path and the paths it owns — so the name is the last segment
 * of its path, the reading the TUI makes, and the owner column is what it may
 * write. How far along a running child is does not travel either: its bar
 * stays empty rather than show a guess.
 */
export function delegation(calls: readonly ToolCall[]): DelegationView {
  const owned = calls.map((c) => list(c.input, 'owns'));
  let disjoint = owned.every((o) => o.length > 0);
  for (let i = 0; disjoint && i < owned.length; i++) {
    for (let j = i + 1; disjoint && j < owned.length; j++) {
      if ((owned[i] ?? []).some((a) => (owned[j] ?? []).some((b) => overlaps(a, b)))) disjoint = false;
    }
  }
  const children = calls.map((c, i): ChildView => {
    const path = field(c.input, 'path');
    const owns = owned[i] ?? [];
    const name = lastSegment(path) || lastSegment(owns[0] ?? '') || `filho ${i + 1}`;
    const status = c.status === 'ok' ? 'ok' : c.status === 'failed' ? 'failed' : 'running';
    const files = c.result?.files ?? 0;
    return {
      id: c.id,
      name,
      owns: owns.length > 0 ? owns.join(', ') : path,
      status,
      bar: status === 'running' ? null : 1,
      meta: status === 'failed' ? 'não respondeu' : status === 'ok' ? (files > 0 ? `leu ${files}` : 'respondeu') : '…',
      reason: status === 'failed' ? firstLine(c.result?.output ?? '') : '',
    };
  });
  const running = children.filter((c) => c.status === 'running').length;
  return { count: calls.length, finished: calls.length - running, running, disjoint, children };
}

export type Phase = 'reading' | 'writing' | 'delegating' | 'running' | 'other';

/** Grouped by what is going on rather than by what the tool is called (internal/tui/activity.go). */
export function phaseOf(tool: string): Phase {
  switch (tool.toLowerCase()) {
    case 'read':
    case 'glob':
    case 'grep':
    case 'symbol':
    case 'fetch':
      return 'reading';
    case 'write':
    case 'edit':
      return 'writing';
    case 'explore':
      return 'delegating';
    case 'bash':
    case 'process':
      return 'running';
    default:
      return 'other';
  }
}

/** The TUI's Portuguese catalogue: synonyms per phase, rotated while a tool runs. */
export const VERBS: Readonly<Record<Phase, readonly string[]>> = {
  reading: ['lendo', 'varrendo', 'procurando'],
  writing: ['escrevendo', 'editando', 'aplicando'],
  delegating: ['delegando', 'repartindo', 'coordenando'],
  running: ['rodando', 'executando', 'conferindo'],
  other: ['processando', 'organizando', 'anotando'],
};

/** The plain word with no tool running. It never rotates. */
export const WORKING = 'trabalhando';

export type Fact =
  | { kind: 'none' }
  | { kind: 'tool'; name: string; target: string }
  | { kind: 'children'; names: string[] };

export interface Activity {
  phase: Phase | null;
  fact: Fact;
}

/**
 * What the activity line says while the turn runs: the phase of the latest
 * running call and the fact beside it. The verb never appears alone — with no
 * call running there is no phase, and the line says the plain word.
 */
export function activity(entries: readonly Entry[]): Activity {
  const running: ToolCall[] = [];
  for (const e of entries) {
    if (e.kind === 'tool' && (e.call.status === 'running' || e.call.status === 'arriving')) running.push(e.call);
  }
  const latest = running[running.length - 1];
  if (!latest) return { phase: null, fact: { kind: 'none' } };
  if (running.every(isDelegated)) {
    const names = delegation(running).children.map((c) => c.name);
    return { phase: 'delegating', fact: { kind: 'children', names } };
  }
  const last = [...running].reverse().find((c) => !isDelegated(c)) ?? latest;
  return { phase: phaseOf(last.name), fact: { kind: 'tool', name: last.name, target: toolTarget(last) } };
}

/** The crossing a request makes, in words; unknown codes are shown as they came. */
export function boundaryLabel(boundary: string): string {
  switch (boundary) {
    case 'network':
      return 'rede';
    case 'filesystem_write':
      return 'escrita fora do workspace';
    case 'filesystem_read':
      return 'leitura fora do workspace';
    case 'workspace_write':
      return 'escrita no workspace';
    case 'rule:write':
      return 'uma escrita coberta por regra';
    case 'rule:read':
      return 'uma leitura coberta por regra';
    case 'rule:command':
      return 'um comando coberto por regra';
    default:
      return boundary || 'uma fronteira sem nome';
  }
}

/** What an answered approval says it was answered with. */
export function decisionLabel(decision: string): { text: string; tone: Tone } {
  switch (decision) {
    case 'allow':
      return { text: 'Permitido uma vez', tone: 'ok' };
    case 'allow_session':
      return { text: 'Permitido nesta sessão', tone: 'ok' };
    case 'allow_project':
      return { text: 'Permitido neste projeto', tone: 'ok' };
    case 'allow_always':
      return { text: 'Permitido sempre', tone: 'ok' };
    case 'deny':
      return { text: 'Negado', tone: 'err' };
    default:
      return { text: `Respondido: ${decision}`, tone: 'dim' };
  }
}

/**
 * The seal a turn ended with. `clean` changed nothing and says nothing — unless
 * the turn wrote where the work is measured, which is never left out: a seal
 * checked with a ruler the turn changed is not a plain pass.
 */
export function sealOf(c: P.Completion): { text: string; tone: Tone } | null {
  const seal = verificationOf(c);
  const touched = c.touched_protected ?? [];
  if (touched.length === 0) return seal;
  const ruler = `tocou a régua: ${touched.join(', ')}`;
  if (!seal) return { text: `⚠ ${ruler}`, tone: 'warn' };
  return { text: `${seal.text} · ${ruler}`, tone: seal.tone === 'err' ? 'err' : 'warn' };
}

function verificationOf(c: P.Completion): { text: string; tone: Tone } | null {
  switch (c.verification) {
    case 'passed':
      return { text: `✓ verificado · ${plural((c.met ?? []).length, 'critério', 'critérios')}`, tone: 'ok' };
    case 'failed':
      return { text: `✗ não verificado · ${(c.unmet ?? []).join(', ')}`, tone: 'err' };
    case 'stale':
      return { text: '⚠ não conferido · mudou depois do check', tone: 'dim' };
    case 'unavailable':
      return { text: '⚠ não conferido · nada pôde ser conferido', tone: 'dim' };
    case 'clean':
      return null;
    default:
      return { text: c.verification, tone: 'dim' };
  }
}
