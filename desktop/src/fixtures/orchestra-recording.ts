// The third recording: an orchestrator's conversation and its children, for
// the live agent list (orchestration RN-9). One child in each of the eight
// states, a merge, a design and a ceiling question waiting for the person, and
// the health of main after the first merge — every fact in the events the
// orchestrator's log carries (.p §5.3), nothing read from text (D8). The
// children that have a conversation of their own replay it too, so a row
// opens one; the dispatched, merged and closed ones have none listed, which
// the panel says.

import type * as P from '../protocol/generated';
import { MINUTE, secondsBefore, SessionWriter, type Recording } from './record';

export const ORCHESTRA_AT = '2026-10-17T15:00:00.000Z';

const DCODE = '/Users/aguinelo/work/dcode';
const CHILDREN = '/Users/aguinelo/work/dcode/.dcode/children';
const opus = { sandbox_mode: 'workspace-write', mode: 'assist', model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };
const minimax = { sandbox_mode: 'workspace-write', mode: 'assist', model: 'MiniMax-M3', family: 'minimax', context_window: 204800 };
const at = (secondsAgo: number) => secondsBefore(ORCHESTRA_AT, secondsAgo);
/** How long ago a child started, in seconds. */
const ago = (c: P.Child) => (Date.parse(ORCHESTRA_AT) - Date.parse(c.started)) / 1000;

type Spec = Pick<P.Child, 'id' | 'title' | 'owns'> & Partial<P.Child> & { startedAgo: number };

function child(s: Spec): P.Child {
  const { startedAgo, ...rest } = s;
  return {
    parent_id: 's-orq',
    profile: 'fast',
    model: 'MiniMax-M3',
    state: 'dispatched',
    branch: `dcode/${s.id.slice(2)}-${s.id.length}a1b3`,
    worktree: `${CHILDREN}/${s.id}`,
    base: '6ec45f1',
    usage: { input_tokens: 0, output_tokens: 0 },
    started: at(startedAgo),
    ...rest,
  };
}

const gate = (commit: string, ago: number): P.GateResult => ({
  commit,
  tree: '5a6b7c8',
  base: '6ec45f1',
  command: 'make check',
  exit_code: 0,
  duration_ms: 81000,
  result: 'green',
  at: at(ago),
});

const protocolo = child({ id: 'c-protocolo', title: 'Tipos do fio da orquestração', owns: ['internal/protocol/orchestrate.go', 'desktop/src/protocol/orchestrate.ts'], startedAgo: 52 * MINUTE });
const sessao = child({ id: 'c-sessao', title: 'Sessão com pai e reabertura', owns: ['internal/session/'], startedAgo: 50 * MINUTE });
const vcs = child({ id: 'c-vcs', title: 'Worktrees dos filhos', owns: ['internal/vcs/'], startedAgo: 48 * MINUTE });
const tui = child({ id: 'c-tui', title: 'A lista viva na TUI', owns: ['internal/tui/agents.go'], startedAgo: 46 * MINUTE });
const politica = child({ id: 'c-politica', title: 'A lista “sempre perguntar”', owns: ['internal/policy/'], profile: 'deep', model: 'Opus 5.5', startedAgo: 44 * MINUTE });
const decisoes = child({ id: 'c-decisoes', title: 'O log de decisões', owns: ['internal/orchestrate/decisions.go', 'cmd/dcode/decisions.go'], startedAgo: 40 * MINUTE });
const docs = child({ id: 'c-docs', title: 'O changelog da família', owns: ['docs/specs/architecture/orchestration/changelog/'], startedAgo: 38 * MINUTE });
const cliente = child({ id: 'c-cliente', title: 'O cliente das rotas novas', owns: ['pkg/client/orchestrate.go'], startedAgo: 2 * MINUTE });

const mergeQuestion: P.Question = {
  id: 'q-merge-521',
  kind: 'merge',
  text: 'Merge #521 (Sessão com pai e reabertura)?',
  recommended: 'merge',
  why: 'revisão aprovou 9a8b7c6 e o gate ficou verde nele',
  child_id: 'c-sessao',
  options: [
    { id: 'merge', label: 'Merge' },
    { id: 'return', label: 'Devolver ao filho' },
    { id: 'close', label: 'Fechar o filho' },
  ],
  sensitive: ['internal/session/reopen.go'],
  asked: at(3 * MINUTE),
};

const designQuestion: P.Question = {
  id: 'q-design-policy',
  kind: 'design',
  text: '“sempre perguntar” nega também no nível autônomo, ou só avisa?',
  recommended: 'deny',
  why: 'a RN-4 diz que nenhum nível desliga a lista',
  child_id: 'c-politica',
  options: [
    { id: 'deny', label: 'Nega sempre' },
    { id: 'warn', label: 'Só avisa' },
  ],
  asked: at(6 * MINUTE),
};

/** The token ceiling, which no child asks: the project's own (orchestrator.token_ceiling). */
const ceilingQuestion: P.Question = {
  id: 'q-ceiling-1',
  kind: 'ceiling',
  text: 'O projeto gastou 2.100.000 tokens, passou do orchestrator.token_ceiling (2.000.000). Continuar com uma janela nova, ou parar de despachar?',
  recommended: 'stop',
  why: 'os filhos que rodam terminam o que fazem; nenhum novo é despachado até a resposta',
  options: [
    { id: 'continue', label: 'Continuar: uma janela nova de tokens' },
    { id: 'stop', label: 'Parar de despachar' },
  ],
  asked: at(90),
};

function orchestrator(): SessionWriter {
  const w = new SessionWriter('s-orq', ORCHESTRA_AT)
    .created(55 * MINUTE, { ...opus, workspace: DCODE, branch: 'main', role: 'orchestrator', project_id: 'p-dcode' })
    .emit('turn.started', 54 * MINUTE, { turn_id: 't-1', text: 'Constrói a primeira fatia do orquestrador, passo a passo, um PR por tema.' })
    .emit('session.renamed', 54 * MINUTE - 5, { name: 'Orquestrar a fatia 1' })
    .emit('message.delta', 53 * MINUTE, { turn_id: 't-1', text: 'São oito temas com donos disjuntos. Despacho em lotes e trago cada um quando a esteira medir.' });
  const state = (ago: number, c: P.Child) => w.emit('child.state', ago, { child: c });
  for (const c of [protocolo, sessao, vcs, tui, politica, decisoes, docs]) w.emit('child.started', ago(c), { child: c });
  state(36 * MINUTE, { ...docs, state: 'closed', reason: 'cancelled', usage: { input_tokens: 3200, output_tokens: 180 }, ended: at(36 * MINUTE) });
  state(30 * MINUTE, { ...protocolo, state: 'in_review', usage: { input_tokens: 88_400, output_tokens: 6_100, cache_read_tokens: 61_000 } });
  w.emit('review.verdict', 28 * MINUTE, { child_id: 'c-protocolo', session_id: 's-rev-1', commit: '3c4d5e6', verdict: 'approve' });
  w.emit('gate.measured', 26 * MINUTE, { child_id: 'c-protocolo', ...gate('3c4d5e6', 26 * MINUTE) });
  state(25 * MINUTE, { ...protocolo, state: 'pr', pr: { number: 518, url: 'https://github.com/aguinelo/dcode/pull/518', ci: 'pending' }, usage: { input_tokens: 88_400, output_tokens: 6_100, cache_read_tokens: 61_000 } });
  w.emit('merge.completed', 20 * MINUTE, { child_id: 'c-protocolo', method: 'squash', pr: 518, merged: '8c82381', measured: '3c4d5e6' });
  const merged: P.Child = { ...protocolo, state: 'merged', head: '3c4d5e6', pr: { number: 518, url: 'https://github.com/aguinelo/dcode/pull/518', ci: 'pass' }, usage: { input_tokens: 88_400, output_tokens: 6_100, cache_read_tokens: 61_000 }, ended: at(20 * MINUTE) };
  state(20 * MINUTE - 2, merged);
  w.emit('main.health', 19 * MINUTE, { commit: '8c82381aa', health: 'green', checked: at(19 * MINUTE) });
  state(18 * MINUTE, { ...decisoes, state: 'working', usage: { input_tokens: 40_100, output_tokens: 3_900 } });
  state(14 * MINUTE, { ...decisoes, state: 'failed', reason: 'max_iterations', usage: { input_tokens: 152_300, output_tokens: 11_800, cache_read_tokens: 98_000 } });
  state(10 * MINUTE, { ...sessao, state: 'in_review', head: '9a8b7c6', usage: { input_tokens: 121_000, output_tokens: 9_400, cache_read_tokens: 80_500, cache_write_tokens: 12_000 } });
  w.emit('review.verdict', 8 * MINUTE, { child_id: 'c-sessao', session_id: 's-rev-2', commit: '9a8b7c6', verdict: 'approve' });
  state(7 * MINUTE, { ...politica, state: 'awaiting_person', waiting: 'question', usage: { input_tokens: 64_000, output_tokens: 4_200 } });
  w.emit('question.asked', 6 * MINUTE, designQuestion);
  state(5 * MINUTE, { ...vcs, state: 'in_review', head: '2b3c4d5', returned: 1, usage: { input_tokens: 97_500, output_tokens: 7_000, cache_read_tokens: 70_000 } });
  state(4 * MINUTE + 30, { ...tui, state: 'working', usage: { input_tokens: 55_800, output_tokens: 4_600, cache_read_tokens: 31_000 } });
  w.emit('gate.measured', 4 * MINUTE, { child_id: 'c-sessao', ...gate('9a8b7c6', 4 * MINUTE) });
  state(3 * MINUTE + 5, {
    ...sessao,
    state: 'pr',
    waiting: 'merge',
    head: '9a8b7c6',
    pr: { number: 521, url: 'https://github.com/aguinelo/dcode/pull/521', ci: 'pending' },
    usage: { input_tokens: 121_000, output_tokens: 9_400, cache_read_tokens: 80_500, cache_write_tokens: 12_000 },
  });
  w.emit('question.asked', 3 * MINUTE, mergeQuestion);
  w.emit('child.started', 2 * MINUTE, { child: cliente });
  w.emit('question.asked', 90, ceilingQuestion);
  return w.emit('progress', 60, { turn_id: 't-1', kind: 'in_flight', done: 6, total: 8 });
}

/** A child's own conversation, as its session's log holds it: asked by the harness, in its worktree. */
function childSession(c: P.Child, ask: string, said: string, running = false): SessionWriter {
  const started = ago(c);
  const w = new SessionWriter(c.id, ORCHESTRA_AT)
    .created(started, { ...(c.model === 'Opus 5.5' ? opus : minimax), workspace: c.worktree, branch: c.branch, role: 'child', parent_id: 's-orq', project_id: 'p-dcode', owns: c.owns })
    .emit('turn.started', started - 5, { turn_id: 't-1', text: ask })
    .emit('session.renamed', started - 6, { name: c.title })
    .emit('message.delta', started - 60, { turn_id: 't-1', text: said });
  return running ? w : w.emit('turn.completed', started - 120, { turn_id: 't-1', reason: 'done' });
}

function emptyOrchestrator(): SessionWriter {
  return new SessionWriter('s-orq-vazia', ORCHESTRA_AT)
    .created(4 * MINUTE, { ...opus, workspace: DCODE, branch: 'main', role: 'orchestrator', project_id: 'p-dcode' })
    .emit('turn.started', 3 * MINUTE, { turn_id: 't-1', text: 'O que falta para a fatia 1?' })
    .emit('session.renamed', 3 * MINUTE - 2, { name: 'Orquestradora do dcode' })
    .emit('message.delta', 2 * MINUTE, { turn_id: 't-1', text: 'Leio o plano antes de despachar alguém.' })
    .emit('turn.completed', 2 * MINUTE - 10, { turn_id: 't-1', reason: 'done' });
}

const sessions = [
  orchestrator(),
  childSession(sessao, 'Implementa o vínculo pai→filhos e a reabertura sob o próprio id.', 'Feito: Reopen e o vínculo, com teste vermelho antes.'),
  childSession(vcs, 'Cria e remove as worktrees dos filhos sem tocar a configuração git.', 'Corrigi o que a revisão apontou em worktree.go.'),
  childSession(tui, 'Mostra a lista viva como texto na TUI.', 'Escrevendo a coluna dos estados…', true),
  childSession(politica, 'Implementa a lista “sempre perguntar” como negação.', 'Preciso saber se o nível autônomo também nega.'),
  childSession(decisoes, 'Escreve o log de decisões em JSONL e o comando que o lê.', 'Cheguei ao teto de rodadas sem fechar o comando.'),
];

export const orchestraRecording: Recording = {
  recordedAt: ORCHESTRA_AT,
  daemonVersion: '0.21.1',
  events: sessions.flatMap((w) => w.events),
};

export const emptyOrchestraRecording: Recording = {
  recordedAt: ORCHESTRA_AT,
  daemonVersion: '0.21.1',
  events: emptyOrchestrator().events,
};
