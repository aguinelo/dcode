// The sixth recording: two orchestrated projects and an ordinary conversation,
// for the sidebar's one row per orchestrated project (D69). The dcode project
// is the third recording's; the listas project has an orchestrator with three
// children, titled by the short titles of their delegate items, and the
// reviewer of one of them — which the sidebar never lists — beside a
// conversation a person opened there.

import type * as P from '../protocol/generated';
import { MINUTE, secondsBefore, SessionWriter, type Recording } from './record';
import { orchestraRecording, ORCHESTRA_AT } from './orchestra-recording';

const LISTAS = '/Users/aguinelo/work/listas';
const opus = { sandbox_mode: 'workspace-write', mode: 'assist', model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };
const at = (secondsAgo: number) => secondsBefore(ORCHESTRA_AT, secondsAgo);

function child(id: string, title: string, state: string, startedAgo: number, over: Partial<P.Child> = {}): P.Child {
  return {
    id,
    parent_id: 's-listas',
    title,
    profile: 'fast',
    model: 'MiniMax-M3',
    state,
    owns: [`${id.slice(2)}.go`],
    branch: `dcode/${id.slice(2)}`,
    worktree: `${LISTAS}/.dcode/children/${id}`,
    base: '1a2b3c4',
    usage: { input_tokens: 12_000, output_tokens: 900 },
    started: at(startedAgo),
    ...over,
  };
}

const unicos = child('c-unicos', 'listas: Unicos', 'in_review', 30 * MINUTE);
const inverte = child('c-inverte', 'listas: Inverte', 'awaiting_person', 29 * MINUTE, { waiting: 'question' });
const agrupa = child('c-agrupa', 'listas: Agrupa', 'pr', 28 * MINUTE, { waiting: 'merge' });

function orchestrator(): SessionWriter {
  const w = new SessionWriter('s-listas', ORCHESTRA_AT)
    .created(32 * MINUTE, { ...opus, mode: 'plan', workspace: LISTAS, branch: 'main', role: 'orchestrator', project_id: 'p-listas' })
    .emit('turn.started', 31 * MINUTE, { turn_id: 't-1', text: 'Três funções novas no pacote listas, uma por PR.' })
    .emit('session.renamed', 31 * MINUTE - 5, { name: 'Funções do pacote listas' });
  for (const c of [unicos, inverte, agrupa]) w.emit('child.started', (Date.parse(ORCHESTRA_AT) - Date.parse(c.started)) / 1000, { child: c });
  return w.emit('turn.completed', 27 * MINUTE, { turn_id: 't-1', reason: 'done' });
}

/** A child's conversation, asked by the harness with its whole task, and named by its short title. */
function childSession(c: P.Child, task: string): SessionWriter {
  const started = (Date.parse(ORCHESTRA_AT) - Date.parse(c.started)) / 1000;
  return new SessionWriter(c.id, ORCHESTRA_AT)
    .created(started, { ...opus, workspace: c.worktree, branch: c.branch, role: 'child', parent_id: 's-listas', project_id: 'p-listas', owns: c.owns })
    .emit('session.renamed', started - 1, { name: c.title })
    .emit('turn.started', started - 5, { turn_id: 't-1', text: task })
    .emit('turn.completed', started - 300, { turn_id: 't-1', reason: 'done' });
}

const reviewer = new SessionWriter('s-rev-unicos', ORCHESTRA_AT)
  .created(10 * MINUTE, { ...opus, mode: 'plan', workspace: unicos.worktree, branch: unicos.branch, role: 'reviewer', parent_id: 's-listas', project_id: 'p-listas' })
  .emit('turn.started', 10 * MINUTE - 2, { turn_id: 't-1', text: 'Review commit 3aa49175 of listas: Unicos.' });

const ordinary = new SessionWriter('s-listas-doc', ORCHESTRA_AT)
  .created(6 * MINUTE, { ...opus, workspace: LISTAS, branch: 'main' })
  .emit('turn.started', 6 * MINUTE - 2, { turn_id: 't-1', text: 'Explica o README do pacote listas.' })
  .emit('message.delta', 5 * MINUTE, { turn_id: 't-1', text: 'O README descreve as funções públicas.' })
  .emit('turn.completed', 5 * MINUTE - 10, { turn_id: 't-1', reason: 'done' });

export const orchestrasRecording: Recording = {
  recordedAt: ORCHESTRA_AT,
  daemonVersion: orchestraRecording.daemonVersion,
  events: [
    ...orchestraRecording.events,
    ...orchestrator().events,
    ...childSession(unicos, 'Crie no pacote `listas` (Go, módulo github.com/x/listas) a função Unicos que remove duplicados.').events,
    ...childSession(inverte, 'Crie no pacote `listas` a função Inverte, que devolve a lista ao contrário.').events,
    ...childSession(agrupa, 'Crie no pacote `listas` a função Agrupa, que agrupa por uma chave.').events,
    ...reviewer.events,
    ...ordinary.events,
  ],
};
