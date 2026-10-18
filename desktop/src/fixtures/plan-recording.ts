// The fifth recording: an orchestrator's plan as its main view (D68). Three
// steps, each fulfilled by a child its delegate item named: one merged, one
// under review, one waiting for its merge, whose question card stands in the
// conversation below the pinned plan. The plan's statuses are the harness's,
// set from the children; every fact is an event the orchestrator's log
// carries, nothing read from text (D8).

import type * as P from '../protocol/generated';
import { MINUTE, secondsBefore, SessionWriter, type Recording } from './record';

export const PLAN_AT = '2026-10-18T18:00:00.000Z';

const DCODE = '/Users/aguinelo/work/dcode';
const at = (secondsAgo: number) => secondsBefore(PLAN_AT, secondsAgo);
const ago = (iso: string) => (Date.parse(PLAN_AT) - Date.parse(iso)) / 1000;
const opus = { sandbox_mode: 'workspace-write', mode: 'plan', model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };

function child(id: string, title: string, owns: string[], step: number, startedAgo: number, spent: number): P.Child {
  return {
    id,
    parent_id: 's-plano',
    title,
    profile: 'fast',
    model: 'MiniMax-M3',
    state: 'working',
    owns,
    branch: `dcode/${id.slice(2)}`,
    worktree: `${DCODE}/.dcode/children/${id}`,
    base: '6ec45f1',
    usage: { input_tokens: spent, output_tokens: Math.round(spent / 12) },
    started: at(startedAgo),
    plan_step: step,
  };
}

const contagem = child('c-contagem', 'Contar as palavras', ['internal/index/'], 1, 50 * MINUTE, 48_000);
const docs = child('c-docs', 'Documentar o índice', ['docs/index.md'], 2, 49 * MINUTE, 31_000);
const tui = child('c-tui', 'A contagem na TUI', ['internal/tui/count.go'], 3, 48 * MINUTE, 57_000);

const verdict = (c: P.Child): P.Verdict => ({ session_id: `s-rev-${c.id.slice(2)}`, commit: '5f2e3d4', verdict: 'approve' });
const gate = (): P.GateResult => ({
  commit: '5f2e3d4',
  tree: 'a1b2c3',
  base: '6ec45f1',
  command: 'make check',
  exit_code: 0,
  duration_ms: 84_000,
  result: 'green',
  at: at(9 * MINUTE),
});

const merged: P.Child = { ...contagem, state: 'merged', review: verdict(contagem), gate: gate(), pr: { number: 571, ci: 'pass' }, ended: at(20 * MINUTE) };
const reviewing: P.Child = { ...docs, state: 'in_review' };
const atMerge: P.Child = { ...tui, state: 'pr', waiting: 'merge', review: verdict(tui), gate: gate(), pr: { number: 573, url: 'https://github.com/aguinelo/dcode/pull/573', ci: 'pass' } };

const plan = (one: string, two: string, three: string): P.PlanItem[] => [
  { id: 1, text: 'Contar as palavras de cada spec', status: one },
  { id: 2, text: 'Documentar o índice', status: two },
  { id: 3, text: 'Mostrar a contagem na TUI', status: three },
];

const question: P.Question = {
  id: 'q-tui',
  header: tui.title,
  kind: 'merge',
  text: `Merge #573 (${tui.title})?`,
  recommended: 'merge',
  why: 'a revisão aprovou o commit e o gate ficou verde nele',
  child_id: tui.id,
  options: [
    { id: 'merge', label: 'Merge', description: 'Entra na main pelo PR, com o gate medido neste commit.' },
    { id: 'return', label: 'Devolver ao filho', description: 'Ele volta a trabalhar, com a sua nota como o que falta.' },
    { id: 'close', label: 'Fechar o filho', description: 'A branch e a worktree saem; nada entra na main.' },
  ],
  allow_other: true,
  summary: 'A TUI mostra quantas palavras cada spec tem, ao lado do nome.\nO teste de render cobre a lista vazia.',
  evidence: {
    verdict: 'approve',
    reviewer: 's-rev-tui',
    reviewed_commit: '5f2e3d4',
    review_seq: 31,
    gate: 'green',
    command: 'make check',
    commit: '5f2e3d4',
    base: '6ec45f1',
    exit_code: 0,
    duration_ms: 84_000,
    gate_seq: 32,
  },
  asked: at(6 * MINUTE),
};

const card: P.QuestionCard = {
  id: 'card-tui',
  kind: 'merge',
  title: 'Um merge pronto',
  items: [{ ...question, state: 'pending', selected: 'merge' }],
  pending: 1,
  created: at(6 * MINUTE),
  updated: at(6 * MINUTE),
};

function orchestrator(): SessionWriter {
  const w = new SessionWriter('s-plano', PLAN_AT)
    .created(55 * MINUTE, { ...opus, workspace: DCODE, branch: 'main', role: 'orchestrator', project_id: 'p-dcode' })
    .emit('turn.started', 54 * MINUTE, { turn_id: 't-1', text: 'Conta as palavras de cada spec, documenta e mostra na TUI.' })
    .emit('session.renamed', 54 * MINUTE - 5, { name: 'A contagem das specs' })
    .emit('plan.updated', 53 * MINUTE, { items: plan('active', 'pending', 'pending') })
    .emit('message.delta', 52 * MINUTE, { turn_id: 't-1', text: 'Três passos, um filho cada, com donos disjuntos. Despacho os três.' });
  for (const c of [contagem, docs, tui]) w.emit('child.started', ago(c.started), { child: c });
  w.emit('plan.updated', 47 * MINUTE, { items: plan('active', 'active', 'active') });
  w.emit('turn.completed', 47 * MINUTE - 10, { turn_id: 't-1', reason: 'done' });
  w.emit('child.state', 20 * MINUTE, { child: merged });
  w.emit('plan.updated', 20 * MINUTE - 1, { items: plan('done', 'active', 'active') });
  w.emit('child.state', 8 * MINUTE, { child: reviewing });
  w.emit('child.state', 6 * MINUTE + 2, { child: atMerge });
  w.emit('question.asked', 6 * MINUTE, question);
  return w.emit('question.card', 6 * MINUTE - 1, card);
}

export const planRecording: Recording = {
  recordedAt: PLAN_AT,
  daemonVersion: '0.22.0',
  events: orchestrator().events,
};
