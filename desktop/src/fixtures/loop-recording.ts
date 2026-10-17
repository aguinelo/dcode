// The second recording, behind the reference states the first one cannot
// reach without changing 02 and 03: a turn with its plan, the files it wrote
// and the documents it was asked with (D57, D58, D59); a /loop working a
// folder, its criteria moving check by check (D54, D55); and a /loop waiting
// for the review of a definition of done the qualifying turn proposed three
// times (D56, D62). Only fields the protocol carries, as in recording.ts.

import type { DoneCriterion, LoopRun, ProposedCriterion } from '../protocol/generated';
import { emptyLoops, type LoopsState } from '../state/loop';
import { MINUTE, secondsBefore, SessionWriter, type Recording } from './record';
import { RECORDED_AT } from './recording';

const LOJA = '/Users/aguinelo/work/loja';
const BIZPACK = '/Users/aguinelo/work/bizpack';

const opus = { sandbox_mode: 'workspace-write', mode: 'assist', model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };

/** A running turn that planned, wrote seven files, and was asked with a PDF and a Markdown file. */
function frete(): SessionWriter {
  const w = new SessionWriter('s-frete', RECORDED_AT)
    .created(9 * MINUTE, { ...opus, workspace: LOJA, branch: 'feat/frete' })
    .emit('turn.started', 300, {
      turn_id: 't-1',
      text: 'Implementa a tabela de frete do contrato anexo; as faixas estão no PDF e as regras de arredondamento no faixas.md.',
      documents: [
        { name: 'contrato-transportadora.pdf', media_type: 'application/pdf', size: 482_133 },
        { name: 'faixas.md', media_type: 'text/markdown', size: 3_912 },
      ],
    })
    .emit('session.renamed', 298, { name: 'Tabela de frete do contrato' })
    .emit('plan.updated', 290, {
      items: [
        { id: 1, text: 'Ler o contrato e as faixas', status: 'done' },
        { id: 2, text: 'Escrever a tabela e o cálculo', status: 'active' },
        { id: 3, text: 'Cobrir as faixas com testes', status: 'pending' },
      ],
    });
  const files = [
    ['internal/frete/tabela.go', 88],
    ['internal/frete/faixa.go', 41],
    ['internal/frete/arredonda.go', 27],
    ['internal/frete/doc.go', 9],
    ['internal/checkout/total.go', 12],
    ['internal/frete/tabela_test.go', 64],
    ['docs/frete.md', 18],
  ] as const;
  files.forEach(([path, added], i) => {
    const at = 240 - i * 25;
    w.emit('tool.requested', at, { turn_id: 't-1', tool_call_id: `call-${i + 1}`, name: 'write', input: { path, content: '…' } });
    w.emit('tool.completed', at - 2, { tool_call_id: `call-${i + 1}`, ok: true, output: `created ${path}`, truncated: false, added, files: 1, duration_ms: 3 });
  });
  return w
    .emit('progress', 60, { turn_id: 't-1', kind: 'rounds', done: 9, total: 100 })
    .emit('message.delta', 50, { turn_id: 't-1', text: 'A tabela está escrita; falta cobrir as faixas de borda do contrato.' });
}

/** The session a /loop works its second folder in: the criteria checked four times so far. */
function carrinho(): SessionWriter {
  const met = (n: string[], unmet: string[]) => ({ met: n, unmet });
  return new SessionWriter('s-carrinho', RECORDED_AT)
    .created(14 * MINUTE, { ...opus, workspace: LOJA, branch: 'loop/carrinho' })
    .emit('turn.started', 700, { turn_id: 't-1', text: 'Trabalhe specs/carrinho até a definição de pronto passar.' })
    .emit('session.renamed', 698, { name: 'loop · specs/carrinho' })
    .emit('tool.requested', 650, { turn_id: 't-1', tool_call_id: 'call-1', name: 'edit', input: { path: 'src/cart.ts' } })
    .emit('tool.completed', 648, { tool_call_id: 'call-1', ok: true, output: 'edited src/cart.ts', truncated: false, added: 14, removed: 3, files: 1 })
    .emit('done.checked', 600, { turn_id: 't-1', cycle: 1, ...met(['tsc'], ['vitest', 'e2e', 'lint']), moved: 'first', stall: 0, stall_limit: 3 })
    .emit('tool.requested', 520, { turn_id: 't-1', tool_call_id: 'call-2', name: 'edit', input: { path: 'src/cart.ts' } })
    .emit('tool.completed', 518, { tool_call_id: 'call-2', ok: true, output: 'edited src/cart.ts', truncated: false, added: 6, removed: 9, files: 1 })
    .emit('done.checked', 470, {
      turn_id: 't-1',
      cycle: 2,
      ...met([], ['tsc', 'vitest', 'e2e', 'lint']),
      moved: 'backward',
      stall: 0,
      stall_limit: 3,
      undone: ['src/cart.ts'],
    })
    .emit('tool.requested', 380, { turn_id: 't-1', tool_call_id: 'call-3', name: 'edit', input: { path: 'src/cart-total.ts' } })
    .emit('tool.completed', 378, { tool_call_id: 'call-3', ok: true, output: 'edited src/cart-total.ts', truncated: false, added: 22, removed: 4, files: 1 })
    .emit('done.checked', 300, { turn_id: 't-1', cycle: 3, ...met(['tsc', 'vitest'], ['e2e', 'lint']), moved: 'forward', stall: 0, stall_limit: 3 })
    .emit('tool.requested', 200, { turn_id: 't-1', tool_call_id: 'call-4', name: 'bash', input: { command: 'npm run e2e -- cart' } })
    .emit('tool.completed', 150, { tool_call_id: 'call-4', ok: false, output: 'exit 1', truncated: false, exit_code: 1, has_exit: true, duration_ms: 48_000 })
    .emit('done.checked', 120, { turn_id: 't-1', cycle: 4, ...met(['tsc', 'vitest'], ['e2e', 'lint']), moved: 'stalled', stall: 1, stall_limit: 3 })
    .emit('message.delta', 100, { turn_id: 't-1', text: 'O e2e do carrinho ainda falha no cupom expirado; vou olhar o fixture dele.' });
}

const proposed = (name: string, command: string, exit_code: number, why: string): ProposedCriterion => ({ name, command, exit_code, why });

/** The qualifying session of a /loop on bizpack: it proposed three times, and the last was measured and written. */
function qualificar(): SessionWriter {
  const criteria: DoneCriterion[] = [
    { ...proposed('checkout-e2e', 'npm run e2e -- checkout', 1, 'o fluxo novo ainda não existe'), class: 'acceptance', exit: 1, output: '1 failed' },
    { ...proposed('unit', 'npm test', 0, 'o que passa hoje tem de continuar passando'), class: 'regression', exit: 0 },
    {
      ...proposed('pix-webhook', 'npm run test:pix', 1, 'o webhook do Pix ainda não é tratado'),
      class: 'regression',
      exit: 0,
      mismatch: true,
    },
  ];
  return new SessionWriter('s-qualificar', RECORDED_AT)
    .created(12 * MINUTE, { ...opus, workspace: BIZPACK, branch: 'loop/checkout-v2' })
    .emit('turn.started', 640, { turn_id: 't-1', text: 'Proponha a definição de pronto de specs/checkout-v2.' })
    .emit('session.renamed', 638, { name: 'loop · qualificar specs/checkout-v2' })
    .emit('message.delta', 200, { turn_id: 't-1', text: 'Três critérios: o e2e do checkout novo, a suíte unitária e o webhook do Pix.' })
    .emit('done.proposed', 130, {
      proposal_id: 'p-3',
      sequence: 3,
      proposed_at: secondsBefore(RECORDED_AT, 150),
      path: 'specs/checkout-v2/done.toml',
      criteria,
      protected: ['specs/checkout-v2/**', 'e2e/**'],
      earlier: [
        {
          sequence: 1,
          proposed_at: secondsBefore(RECORDED_AT, 390),
          criteria: [
            proposed('checkout-e2e', 'npm run e2e -- checkout', 1, 'o fluxo novo ainda não existe'),
            proposed('unit', 'npm test', 0, 'o que passa hoje tem de continuar passando'),
          ],
        },
        {
          sequence: 2,
          proposed_at: secondsBefore(RECORDED_AT, 270),
          criteria: [
            proposed('checkout-e2e', 'npm run e2e -- checkout', 1, 'o fluxo novo ainda não existe'),
            proposed('unit', 'npm test', 0, 'o que passa hoje tem de continuar passando'),
            proposed('lint', 'npm run lint', 0, 'sem aviso novo'),
          ],
          protected: ['specs/checkout-v2/**'],
        },
      ],
    })
    .emit('turn.completed', 128, {
      turn_id: 't-1',
      reason: 'done',
      usage: { input_tokens: 96_000, output_tokens: 3_100, context_tokens: 41_000 },
    });
}

export const loopRecording: Recording = {
  recordedAt: RECORDED_AT,
  daemonVersion: '0.21.1',
  events: [frete(), carrinho(), qualificar()].flatMap((w) => w.events),
};

const working: LoopRun = {
  id: 'loop-carrinho',
  workspace: LOJA,
  args: 'fechar o backlog do carrinho',
  goal: 'fechar o backlog do carrinho',
  state: 'working',
  specs: [
    { path: 'specs/cupom', phase: 'done', session_id: 's-cupom', criteria: 3 },
    { path: 'specs/carrinho', phase: 'working', session_id: 's-carrinho', criteria: 4 },
    { path: 'specs/frete-gratis', phase: 'pending', criteria: 2 },
  ],
  current: 1,
  session_id: 's-carrinho',
  started: secondsBefore(RECORDED_AT, 31 * MINUTE),
};

const reviewing: LoopRun = {
  id: 'loop-checkout',
  workspace: BIZPACK,
  args: 'specs/checkout-v2',
  state: 'awaiting_review',
  specs: [
    {
      path: 'specs/checkout-v2',
      phase: 'proposed',
      session_id: 's-qualificar',
      proposal: 'specs/checkout-v2/done.toml',
      proposal_criteria: 3,
    },
  ],
  current: 0,
  session_id: 's-qualificar',
  started: secondsBefore(RECORDED_AT, 12 * MINUTE),
};

/** The loops the second recording's daemon conducts, as its first snapshot would bring them. */
export const recordedLoops: LoopsState = {
  ...emptyLoops,
  byId: { [working.id]: working, [reviewing.id]: reviewing },
  order: [working.id, reviewing.id],
  loaded: true,
};
