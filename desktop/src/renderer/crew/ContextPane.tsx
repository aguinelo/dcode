import { ChevronDown, FileDiff } from 'lucide-react';
import { useMemo, useState } from 'react';
import type * as P from '../../protocol/generated';
import { filesTouched } from '../../state/context';
import { contextPercent } from '../../state/crew';
import type { GitState } from '../../state/git';
import type { History } from '../../state/history';
import type { Measure } from '../../state/models';
import type { PaneView } from '../../state/pane';
import type { SessionView } from '../../state/session';
import { tokens, turnTelemetry, usageFact } from '../../state/telemetry';
import type { Row } from '../../state/sidebar';
import { GitChanges } from '../context/GitChanges';
import { GitFacts } from '../context/GitFacts';
import { ModeChip } from '../ModeMenu';
import { LoopSection } from '../LoopRuler';
import { DoneRow } from '../Pane';

type Tab = 'mudancas' | 'pronto' | 'sessao';

/**
 * What this conversation's tools said they wrote (D8) — beside the folder's
 * git, never in place of it (D41). An ended one's come from its history (D42).
 */
function Wrote({ session, row, history }: { session: SessionView | undefined; row: Row; history?: History }) {
  const files = useMemo(() => (session ? filesTouched(session.entries) : []), [session]);
  if (!session) {
    // An ended conversation's files come from its history (D42): being read, or why it could not be.
    if (row.state === 'recorded' && history?.state === 'failed') {
      return <p className="crew-quiet tone-err">Os arquivos vêm do histórico desta conversa, que não pôde ser lido.</p>;
    }
    return (
      <p className="crew-quiet">
        {row.state === 'recorded' ? 'Lendo o histórico desta conversa para saber os arquivos que ela mudou…' : 'Os eventos desta conversa ainda estão chegando.'}
      </p>
    );
  }
  if (files.length === 0) return <p className="crew-quiet">As ferramentas desta sessão não escreveram nenhum arquivo ainda.</p>;
  const added = files.reduce((n, f) => n + f.added, 0);
  const removed = files.reduce((n, f) => n + f.removed, 0);
  return (
    <>
      <div className="crew-tab-sum">
        <span>
          Esta sessão escreveu {files.length} {files.length === 1 ? 'arquivo' : 'arquivos'}
        </span>
        <span className="mono">
          <span className="tone-ok">+{added}</span> <span className="tone-err">−{removed}</span>
        </span>
      </div>
      <ul className="crew-files">
        {files.map((f) => (
          <li key={f.path} className={f.failed ? 'failed' : undefined} title={f.path}>
            <FileDiff size={12} className="crew-file-icon" aria-hidden />
            <span className="crew-file mono">{f.path}</span>
            {f.failed ? (
              <span className="tone-err">falhou</span>
            ) : (
              <span className="crew-diff mono">
                <span className="tone-ok">+{f.added}</span> <span className="tone-err">−{f.removed}</span>
              </span>
            )}
          </li>
        ))}
      </ul>
      <p className="crew-quiet crew-foot-note">Lido das chamadas de escrita e das linhas que as ferramentas reportaram (D8).</p>
    </>
  );
}

function Done({ view, measure }: { view: PaneView; measure: Measure | null }) {
  return (
    <>
      <DoneRow view={view} measure={measure} />
      {view.touched.length > 0 && (
        <div className="crew-touched">
          <span className="tone-warn">Escreveu, no último turno, onde o trabalho é medido:</span>
          <ul className="crew-files">
            {view.touched.map((p) => (
              <li key={p}>
                <span className="crew-file mono">{p}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <p className="crew-quiet crew-foot-note">
        O selo é o que os critérios conferiram — a cada conferência do turno que roda, e ao fim do turno —, não o que o modelo disse (D13,
        D54).
</p>
    </>
  );
}

function Facts({
  row,
  session,
  onModel,
  onMode,
}: {
  row: Row;
  session: SessionView | undefined;
  onModel: (anchor: HTMLElement) => void;
  onMode: (anchor: HTMLElement) => void;
}) {
  const info = session?.info;
  const pct = contextPercent(session);
  const model = info?.model ?? row.model;
  return (
    <dl className="crew-facts">
      <dt>modelo</dt>
      <dd>
        {model ? (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => onModel(e.currentTarget)}>
            {model} <ChevronDown size={11} aria-hidden />
          </button>
        ) : (
          '—'
        )}
      </dd>
      {session?.info && (session.sandbox || session.mode) && (
        <>
          <dt>modo</dt>
          <dd>
            <ModeChip mode={session.mode} sandbox={session.sandbox} className="chip" onOpen={onMode} />
          </dd>
        </>
      )}
      <dt>turnos</dt>
      <dd>{session?.turns ?? row.turns ?? 0}</dd>
      <dt>contexto</dt>
      <dd>{pct !== null ? `${pct}%` : <span className="tone-faint">medido ao fim do primeiro turno</span>}</dd>
      <TurnFacts session={session} />
    </dl>
  );
}

/**
 * The running turn against its ceilings, and what the last one cost (D44).
 * The usage arrives only in turn.completed, so a running turn's is said to
 * come at its end rather than estimated here.
 */
function TurnFacts({ session }: { session: SessionView | undefined }) {
  if (!session) return null;
  const live = turnTelemetry(session);
  const tone = (c: { tone: string } | null) => (c ? `tone-${c.tone}` : undefined);
  return (
    <>
      {live && (
        <>
          <dt>turno</dt>
          <dd data-turn-telemetry>
            {session.rounds || session.inFlight ? (
              <>
                {session.rounds && (
                  <span className={tone(live.rounds)}>
                    rodada {session.rounds.done}
                    {session.rounds.total !== null && ` de ${session.rounds.total}`}
                  </span>
                )}
                {session.rounds && session.inFlight && ' · '}
                {session.inFlight && (
                  <span className={tone(live.inFlight)}>
                    {session.inFlight.done}
                    {session.inFlight.total !== null && `/${session.inFlight.total}`} em voo
                  </span>
                )}
              </>
            ) : (
              <span className="tone-faint">{live.detail}</span>
            )}
          </dd>
        </>
      )}
      {!live && session.usage ? (
        <Spent usage={session.usage} />
      ) : (
        <>
          <dt>tokens</dt>
          <dd data-turn-usage>
            {live ? (
              <span className="tone-faint">ditos pelo daemon ao fim do turno</span>
            ) : session.turns > 0 ? (
              <span className="tone-warn">o provedor não informou o uso do último turno</span>
            ) : (
              <span className="tone-faint">ditos ao fim do primeiro turno</span>
            )}
          </dd>
        </>
      )}
    </>
  );
}

/** The last turn's usage: the output, the input summed by its rounds, the cache. */
function Spent({ usage }: { usage: P.Usage }) {
  const fact = usageFact(usage);
  const cached = !!usage.cache_read_tokens || !!usage.cache_write_tokens;
  return (
    <>
      <dt>saída</dt>
      <dd data-turn-usage title={`Último turno: ${fact.detail}`}>
        {tokens(usage.output_tokens)} <span className="tone-faint">no último turno</span>
      </dd>
      <dt>entrada</dt>
      <dd title="Cada rodada reenvia o contexto: é o que o turno custou, não o quanto o contexto está cheio.">
        {tokens(usage.input_tokens)} <span className="tone-faint">somada pelas rodadas</span>
      </dd>
      {cached && (
        <>
          <dt>cache</dt>
          <dd>
            {tokens(usage.cache_read_tokens ?? 0)} lidos · {tokens(usage.cache_write_tokens ?? 0)} escritos
          </dd>
        </>
      )}
    </>
  );
}

/**
 * The right pane of the Painel: what the conversation in focus established,
 * in tabs — what changed in its folder and what it wrote, its seal and
 * criteria, and where it runs (D41). Only what the daemon carries; nothing
 * here stands in for a pull request.
 */
export function ContextPane({
  row,
  view,
  session,
  history,
  measure,
  git,
  moved,
  onModel,
  onMode,
}: {
  row: Row | null;
  view: PaneView | null;
  session: SessionView | undefined;
  /** For an ended conversation, its history as it is being read (D42); `session` is its folded view once read. */
  history?: History;
  measure: Measure | null;
  /** The folder's git, read once for the thread's header and this pane (D41). */
  git: { state: GitState; refresh: () => void };
  moved: unknown;
  onModel: (row: Row, anchor: HTMLElement) => void;
  onMode: (id: string, anchor: HTMLElement) => void;
}) {
  const [tab, setTab] = useState<Tab>('mudancas');
  const wrote = useMemo(() => (session ? filesTouched(session.entries) : []), [session]);
  const wrotePaths = useMemo(() => new Set(wrote.map((f) => f.path)), [wrote]);
  // The folder's count once git answered; what the tools wrote until then.
  const files = git.state.kind === 'ready' && git.state.git.repository ? git.state.git.files.length : wrote.length;
  const tabs: { id: Tab; label: string; count?: string }[] = [
    { id: 'mudancas', label: 'Mudanças', count: files > 0 ? String(files) : undefined },
    { id: 'pronto', label: 'Pronto' },
    { id: 'sessao', label: 'Sessão' },
  ];
  return (
    <aside className="crew-context" aria-label="Contexto da conversa">
      <div className="crew-tabs" role="tablist">
        {tabs.map((t) => (
          <button key={t.id} type="button" role="tab" aria-selected={t.id === tab} className={t.id === tab ? 'on' : ''} onClick={() => setTab(t.id)}>
            {t.label}
            {t.count && <span className="crew-count">{t.count}</span>}
          </button>
        ))}
      </div>
      <div className="crew-tab-body" role="tabpanel">
        {!row || !view ? (
          <p className="crew-quiet">Nenhuma conversa em foco.</p>
        ) : tab === 'mudancas' ? (
          <>
            <GitChanges conversationId={row.id} state={git.state} moved={moved} wrote={wrotePaths} onRefresh={git.refresh} />
            <div className="ctx-divider" />
            <Wrote session={session} row={row} history={history} />
          </>
        ) : tab === 'pronto' ? (
          <>
            <LoopSection sessionId={row.id} />
            <Done view={view} measure={measure} />
          </>
        ) : (
          <>
            <GitFacts row={row} sessionState={history ? 'recorded' : session?.state} startedOn={session?.info?.branch ?? row.branch} state={git.state} />
            <div className="ctx-divider" />
            <Facts row={row} session={session} onModel={(anchor) => onModel(row, anchor)} onMode={(anchor) => onMode(row.id, anchor)} />
          </>
        )}
      </div>
    </aside>
  );
}
