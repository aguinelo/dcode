// The fourth recording: an orchestrator that asks in its conversation (.p
// §5.5, D67). Three children reach their merge question together, and the
// harness groups them into one card, each item with the orchestrator's
// summary and the evidence read from the decision log; the person answers in
// their own words, and the orchestrator proposes a card from them — its
// selections checked, an answer only once the person confirms it. Every fact
// is an event the orchestrator's log carries, nothing read from text (D8).

import type * as P from '../protocol/generated';
import { MINUTE, secondsBefore, SessionWriter, type Recording } from './record';

export const QUESTIONS_AT = '2026-10-18T16:00:00.000Z';

const DCODE = '/Users/aguinelo/work/dcode';
const at = (secondsAgo: number) => secondsBefore(QUESTIONS_AT, secondsAgo);
const opus = { sandbox_mode: 'workspace-write', mode: 'assist', model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };

const MERGE: P.QuestionOption[] = [
  { id: 'merge', label: 'Merge', description: 'Entra na main pelo PR, com o gate medido neste commit.' },
  { id: 'return', label: 'Devolver ao filho', description: 'Ele volta a trabalhar, com a sua nota como o que falta.' },
  { id: 'close', label: 'Fechar o filho', description: 'A branch e a worktree saem; nada entra na main.' },
];

function child(id: string, title: string, owns: string[], pr: number, startedAgo: number): P.Child {
  return {
    id,
    parent_id: 's-perg',
    title,
    profile: 'fast',
    model: 'MiniMax-M3',
    state: 'working',
    owns,
    branch: `dcode/${id.slice(2)}`,
    worktree: `${DCODE}/.dcode/children/${id}`,
    base: '6ec45f1',
    head: `${pr}a1b2c3d`,
    usage: { input_tokens: 61_000 + pr, output_tokens: 4_800 },
    started: at(startedAgo),
  };
}

/** The child at its merge question: its PR open, CI green, waiting for the person. */
const atMerge = (c: P.Child, pr: number): P.Child => ({ ...c, state: 'pr', waiting: 'merge', pr: { number: pr, url: `https://github.com/aguinelo/dcode/pull/${pr}`, ci: 'pass' } });

const indice = child('c-indice', 'O índice das specs', ['docs/specs/INDEX.md', 'scripts/index.sh'], 561, 40 * MINUTE);
const notas = child('c-notas', 'Notas da release', ['docs/release/'], 562, 38 * MINUTE);
const fila = child('c-fila', 'A fila da TUI', ['internal/tui/queue.go'], 563, 36 * MINUTE);

function evidence(c: P.Child, seq: number, gate = 'green', findings = 0): P.QuestionEvidence {
  return {
    verdict: 'approve',
    reviewer: `s-rev-${c.id.slice(2)}`,
    reviewed_commit: c.head ?? '',
    ...(findings > 0 ? { findings } : {}),
    review_seq: seq,
    gate,
    command: 'make check',
    commit: c.head ?? '',
    base: '6ec45f1',
    exit_code: gate === 'green' ? 0 : 1,
    duration_ms: 84_000,
    gate_seq: seq + 1,
  };
}

function merge(c: P.Child, pr: number, summary: string, ev: P.QuestionEvidence, askedAgo: number): P.Question {
  return {
    id: `q-${c.id.slice(2)}`,
    header: c.title,
    kind: 'merge',
    text: `Merge #${pr} (${c.title})?`,
    recommended: 'merge',
    why: 'a revisão aprovou o commit e o gate ficou verde nele',
    child_id: c.id,
    options: MERGE,
    allow_other: true,
    summary,
    evidence: ev,
    asked: at(askedAgo),
  };
}

const questions = [
  merge(indice, 561, 'Gera docs/specs/INDEX.md a partir das pastas de spec.\nO script falha se uma pasta não tiver .r.', evidence(indice, 41), 6 * MINUTE),
  merge(notas, 562, 'As notas da 0.22 em pt-BR e en, lidas do CHANGELOG.\nSem mudança de código.', evidence(notas, 47, 'green', 1), 5 * MINUTE + 40),
  merge(fila, 563, 'A fila da TUI mostra as perguntas com a recomendada.\nO teste de render cobre a fila vazia.', evidence(fila, 53), 5 * MINUTE + 20),
];

const item = (q: P.Question, extra: Partial<P.CardItem> = {}): P.CardItem => ({ ...q, state: 'pending', selected: q.recommended, ...extra });

const mergesCard = (ago: number): P.QuestionCard => ({
  id: 'card-merges',
  kind: 'merge',
  title: 'Três merges prontos',
  items: questions.map((q) => item(q)),
  pending: 3,
  created: at(6 * MINUTE),
  updated: at(ago),
});

const WORDS = 'manda o índice e as notas; a fila volta, quero o teste da fila cheia também';

const proposedCard: P.QuestionCard = {
  id: 'card-proposta',
  kind: 'merge',
  title: 'O que entendi do que você pediu',
  proposed: true,
  from: WORDS,
  why: 'Li merge no índice e nas notas, e devolver na fila, com o teste que falta como nota.',
  items: [
    item(questions[0]!),
    item(questions[1]!),
    item(questions[2]!, { selected: 'return', selected_note: 'falta o teste da fila cheia' }),
  ],
  pending: 3,
  created: at(2 * MINUTE),
  updated: at(2 * MINUTE),
};

function orchestrator(): SessionWriter {
  const w = new SessionWriter('s-perg', QUESTIONS_AT)
    .created(45 * MINUTE, { ...opus, workspace: DCODE, branch: 'main', role: 'orchestrator', project_id: 'p-dcode' })
    .emit('turn.started', 44 * MINUTE, { turn_id: 't-1', text: 'Fecha o índice, as notas e a fila da TUI, um PR cada.' })
    .emit('session.renamed', 44 * MINUTE - 5, { name: 'Três temas pequenos' })
    .emit('message.delta', 43 * MINUTE, { turn_id: 't-1', text: 'Três temas com donos disjuntos: despacho os três e trago cada merge quando a esteira medir.' });
  for (const c of [indice, notas, fila]) w.emit('child.started', (Date.parse(QUESTIONS_AT) - Date.parse(c.started)) / 1000, { child: c });
  w.emit('turn.completed', 42 * MINUTE, { turn_id: 't-1', reason: 'done' });
  questions.forEach((q, i) => {
    const c = [indice, notas, fila][i]!;
    w.emit('child.state', (Date.parse(QUESTIONS_AT) - Date.parse(q.asked)) / 1000 + 2, { child: atMerge(c, 561 + i) });
    w.emit('question.asked', (Date.parse(QUESTIONS_AT) - Date.parse(q.asked)) / 1000, q);
  });
  w.emit('question.card', 5 * MINUTE + 20, mergesCard(5 * MINUTE + 20));
  w.emit('turn.started', 3 * MINUTE, { turn_id: 't-2', text: WORDS });
  w.emit('message.delta', 2 * MINUTE + 30, { turn_id: 't-2', text: 'Entendi assim — confirme no cartão, e só a sua confirmação vira resposta:' });
  w.emit('question.card', 2 * MINUTE, proposedCard);
  return w.emit('turn.completed', 2 * MINUTE - 5, { turn_id: 't-2', reason: 'done' });
}

export const questionsRecording: Recording = {
  recordedAt: QUESTIONS_AT,
  daemonVersion: '0.22.0',
  events: orchestrator().events,
};
