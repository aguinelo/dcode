// The recorded state the window shows until it talks to a daemon: the two
// sessions of the v2 handoff's screens 02 and 03 in one project, and the idle
// ones around them. Only fields the protocol carries, with the values a daemon
// would send — which is why some of what the mock shows is not here (see
// docs/DECISIONS.md, "Lacunas").

import type { Prefs } from '../state/prefs';
import { DAY, finishedTurn, HOUR, MINUTE, SessionWriter, type Recording } from './record';

export const RECORDED_AT = '2026-09-28T18:00:00.000Z';

const DCODE = '/Users/aguinelo/work/dcode';
const BIZPACK = '/Users/aguinelo/work/bizpack';
const EVA = '/Users/aguinelo/work/eva';

const base = { sandbox_mode: 'workspace-write', mode: 'assist', context_window: 204800 };
const minimax = { ...base, model: 'MiniMax-M3', family: 'minimax' };
const opus = { ...base, model: 'Opus 5.5', family: 'anthropic', context_window: 1000000 };

function catalogar(): SessionWriter {
  const w = new SessionWriter('s-catalogar', RECORDED_AT);
  const child = (id: string, dir: string) =>
    w.emit('tool.requested', 90, {
      turn_id: 't-1',
      tool_call_id: id,
      name: 'explore',
      input: {
        task: `Catalogar as chamadas de .Save em ${dir} e escrever ${dir}ARCH.md.`,
        path: dir,
        owns: [dir],
      },
    });
  w.created(180, { ...minimax, workspace: DCODE, branch: 'feat/catalogo' })
    .emit('turn.started', 102, {
      turn_id: 't-1',
      text: 'Cataloga toda chamada de .Save em internal/ e escreve um ARCH.md por pacote.',
    })
    .emit('session.renamed', 100, { name: 'Catalogar chamadas de .Save' })
    .emit('progress', 100, { turn_id: 't-1', kind: 'rounds', done: 1, total: 100 })
    .emit('tool.requested', 98, {
      turn_id: 't-1',
      tool_call_id: 'call-1',
      name: 'grep',
      input: { pattern: '\\.Save\\(', path: 'internal' },
    })
    .emit('tool.completed', 97, {
      tool_call_id: 'call-1',
      ok: true,
      output: 'internal/alpha/store.go:41: s.Save(ctx, rec)\n…',
      truncated: false,
      lines: 11,
      files: 4,
      duration_ms: 212,
    })
    .emit('progress', 95, { turn_id: 't-1', kind: 'rounds', done: 2, total: 100 })
    .emit('message.delta', 93, { turn_id: 't-1', text: 'São quatro pacotes independentes. ' })
    .emit('message.delta', 92, {
      turn_id: 't-1',
      text: 'Vou repartir: cada filho fica dono de um pacote e escreve só o ARCH.md dele.',
    });
  child('call-2', 'internal/alpha/');
  child('call-3', 'internal/bravo/');
  child('call-4', 'internal/store/');
  child('call-5', 'internal/tui/');
  return w
    .emit('progress', 89, { turn_id: 't-1', kind: 'in_flight', done: 4, total: 4 })
    .emit('tool.completed', 41, {
      tool_call_id: 'call-2',
      ok: true,
      output:
        'alpha chama .Save em três lugares, todos no caminho de escrita.\n\nlooked at: internal/alpha/store.go, …\nwrote: internal/alpha/ARCH.md',
      truncated: false,
      files: 9,
      duration_ms: 48900,
    })
    .emit('tool.completed', 22, {
      tool_call_id: 'call-3',
      ok: false,
      output: 'the delegated turn failed: context deadline exceeded (task: Catalogar as chamadas de .Save em internal/bravo/ e escrever internal/bravo/ARCH.md.)',
      truncated: false,
      duration_ms: 67800,
    })
    .emit('progress', 22, { turn_id: 't-1', kind: 'in_flight', done: 2, total: 4 });
}

function tokens(): SessionWriter {
  return new SessionWriter('s-tokens', RECORDED_AT)
    .created(5 * MINUTE, { ...minimax, workspace: DCODE, branch: 'feat/tokens' })
    .emit('turn.started', 172, { turn_id: 't-1', text: 'Gera as variáveis CSS a partir de docs/brand/, claro e escuro.' })
    .emit('session.renamed', 170, { name: 'Extrair tokens de cor' })
    .emit('tool.requested', 160, {
      turn_id: 't-1',
      tool_call_id: 'call-1',
      name: 'read',
      input: { path: 'docs/brand/PALETTE.md' },
    })
    .emit('tool.completed', 159, {
      tool_call_id: 'call-1',
      ok: true,
      output: '# Palette\n…',
      truncated: false,
      lines: 96,
      files: 1,
      duration_ms: 4,
    })
    .emit('tool.requested', 131, {
      turn_id: 't-1',
      tool_call_id: 'call-2',
      name: 'write',
      input: { path: 'desktop/src/tokens.json', content: '{ … }' },
    })
    .emit('tool.completed', 130, {
      tool_call_id: 'call-2',
      ok: true,
      output: 'created desktop/src/tokens.json',
      truncated: false,
      added: 54,
      files: 1,
      duration_ms: 3,
    })
    .emit('message.delta', 110, {
      turn_id: 't-1',
      text: 'Para virar variáveis CSS preciso de uma dependência de build — isso usa rede e escreve fora de `src/`.',
    })
    .emit('tool.requested', 96, {
      turn_id: 't-1',
      tool_call_id: 'call-3',
      name: 'bash',
      input: { command: 'npm i -D style-dictionary' },
    })
    .emit('tool.approval_required', 95, {
      approval_id: 't-1-2',
      turn_id: 't-1',
      tool_call_id: 'call-3',
      tool: 'bash',
      command: 'npm i -D style-dictionary',
      boundary_crossed: 'network',
      // What the daemon sends today: the deadline is set after the event goes
      // out, so the wire carries the zero time (docs/DECISIONS.md, "Lacunas").
      expires_at: '0001-01-01T00:00:00Z',
      reason: 'this would reach the network',
    });
}

function suite(): SessionWriter {
  return new SessionWriter('s-suite', RECORDED_AT)
    .created(16 * MINUTE, { ...minimax, workspace: DCODE, branch: 'fix/integration' })
    .emit('turn.started', 922, { turn_id: 't-1', text: 'Faz a suíte de integração passar.' })
    .emit('session.renamed', 920, { name: 'Fazer a suíte de integração passar' })
    .emit('tool.requested', 900, {
      turn_id: 't-1',
      tool_call_id: 'call-1',
      name: 'bash',
      input: { command: 'go test ./internal/integration/...' },
    })
    .emit('tool.completed', 862, {
      tool_call_id: 'call-1',
      ok: false,
      output: 'exit 1\n--- FAIL: TestReplayAfterCrash (0.41s)',
      truncated: false,
      exit_code: 1,
      has_exit: true,
      duration_ms: 38100,
    })
    .emit('message.delta', 850, { turn_id: 't-1', text: 'Falta TestReplayAfterCrash, que depende de fsync.' })
    .emit('tool.requested', 40, {
      turn_id: 't-1',
      tool_call_id: 'call-2',
      name: 'read',
      input: { path: 'internal/log/writer.go' },
    })
    .emit('tool.completed', 39, {
      tool_call_id: 'call-2',
      ok: true,
      output: 'package log\n…',
      truncated: false,
      lines: 233,
      files: 1,
    })
    .emit('tool.requested', 12, {
      turn_id: 't-1',
      tool_call_id: 'call-3',
      name: 'grep',
      input: { pattern: 'Sync\\(\\)', path: 'internal/log/' },
    });
}

function idle(
  id: string,
  workspace: string,
  model: typeof minimax,
  branch: string,
  title: string,
  createdAgo: number,
  endedAgo: number,
  ending: Parameters<typeof finishedTurn>[2]['completed'],
): SessionWriter {
  const w = new SessionWriter(id, RECORDED_AT).created(createdAgo, { ...model, workspace, branch });
  w.emit('session.renamed', createdAgo - 30, { name: title });
  return finishedTurn(w, endedAgo, {
    id: 't-1',
    ask: title,
    tools: [
      {
        id: 'call-1',
        name: 'read',
        input: { path: 'README.md' },
        result: { ok: true, output: '…', truncated: false, lines: 120, files: 1 },
      },
    ],
    answer: 'Feito.',
    completed: ending,
  });
}

const usage = (context: number) => ({ input_tokens: context * 3, output_tokens: 2400, context_tokens: context });

export const recording: Recording = {
  recordedAt: RECORDED_AT,
  daemonVersion: '0.21.1',
  events: [
    catalogar(),
    tokens(),
    suite(),
    idle('s-done-check', DCODE, minimax, 'fix/done-interrupt', 'Stop the done check after an interrupt',
      2 * HOUR + 40 * MINUTE, 2 * HOUR + 5 * MINUTE,
      { reason: 'done', usage: usage(61000), completion: { verification: 'passed', met: ['go test', 'go vet'] } }),
    idle('s-rebuild', DCODE, opus, 'fix/rebuild-batch', 'Fix Rebuild orphaning multi-call batch',
      31 * HOUR, 30 * HOUR,
      { reason: 'done', usage: usage(212000), completion: { verification: 'stale', met: ['go test'] } }),
    idle('s-specguard', DCODE, minimax, 'feat/specguard', 'Make specguard reject ambiguous invariants',
      36 * HOUR, 35 * HOUR, { reason: 'done', usage: usage(88000) }),
    idle('s-checkout', BIZPACK, minimax, 'main', 'Revisar fluxo de checkout',
      8 * DAY + 2 * HOUR, 8 * DAY, { reason: 'done', usage: usage(40000) }),
    idle('s-webhooks', BIZPACK, opus, 'feat/webhook-queue', 'Migrar webhooks para fila',
      15 * DAY + 3 * HOUR, 15 * DAY, { reason: 'interrupted', usage: usage(154000) }),
    idle('s-prompt-cache', EVA, minimax, 'feat/prompt-cache', 'Compartilhar cache do system prompt',
      22 * DAY + HOUR, 22 * DAY, { reason: 'done', usage: usage(71000) }),
    idle('s-ci', EVA, minimax, 'fix/ci-integration', 'Consertar a CI: testes de integração',
      35 * DAY + HOUR, 35 * DAY,
      { reason: 'done', usage: usage(52000), completion: { verification: 'passed', met: ['go test'] } }),
  ].flatMap((w) => w.events),
};

/**
 * The client-local side of the recorded state: what the person had arranged in
 * the sidebar (eva collapsed) and who they are. Neither travels on the wire.
 */
export const recordedClient: { prefs: Prefs; userName: string } = {
  prefs: { order: [], labels: {}, collapsed: { [EVA]: true } },
  userName: 'Aguinelo Koczkodai',
};

/** The reference states of the handoff, by screenshot name, and the session each one shows. */
export const referenceStates: Readonly<Record<string, string>> = {
  '02-janela-principal-rodando': 's-catalogar',
  '03-janela-principal-aprovacao': 's-tokens',
};
