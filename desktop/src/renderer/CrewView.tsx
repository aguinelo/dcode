import { useMemo } from 'react';
import { filesTouched } from '../state/context';
import type { Measure } from '../state/models';
import { paneView } from '../state/pane';
import { emptySession, type SessionView } from '../state/session';
import type { SessionsState } from '../state/sessions';
import { shortPath, type Row } from '../state/sidebar';
import { Field } from './Composer';
import { ConversationBody } from './ConversationBody';
import { DoneRow } from './Pane';

/** What the Crew view does, by the conversation in focus. */
export interface CrewHandlers {
  focus: (id: string) => void;
  draft: (id: string, text: string) => void;
  send: (id: string, text: string) => void;
  stop: (id: string) => void;
  answer: (id: string, approvalId: string, decision: string) => void;
  continueIn: (row: Row, text: string) => void;
  model: (row: Row, anchor: HTMLElement) => void;
  missing: (what: string) => void;
  newSession: () => void;
}

function stateWord(state: string): string {
  if (state === 'running') return 'rodando';
  if (state === 'blocked') return 'esperando você';
  if (state === 'recorded') return 'terminada';
  return 'parada';
}

function Section({ title, count, children }: { title: string; count?: number; children: React.ReactNode }) {
  return (
    <section className="crew-section">
      <h3 className="crew-section-title">
        {title}
        {count !== undefined && <span className="crew-count">{count}</span>}
      </h3>
      {children}
    </section>
  );
}

/**
 * The second version of the screen (D32), after Kiro Crew: the list of
 * conversations on the left, the one in focus in the middle, and on the right
 * what it established — its seal and criteria, the files it changed, where it
 * runs — with the rest of the crew one click away.
 */
export function CrewView({
  row,
  session,
  crew,
  rows,
  sessions,
  now,
  verbTick,
  draft,
  caret,
  measure,
  on,
}: {
  /** The conversation in focus, if any. */
  row: Row | null;
  session: SessionView | undefined;
  /** The conversations working or wanting attention, in the grid's order. */
  crew: readonly string[];
  rows: ReadonlyMap<string, Row>;
  sessions: SessionsState;
  now: number;
  verbTick: number;
  draft: string;
  caret?: number;
  measure: Measure | null;
  on: CrewHandlers;
}) {
  const files = useMemo(() => (session ? filesTouched(session.entries) : []), [session]);
  if (!row) {
    return (
      <div className="crew crew-empty">
        <strong>Nenhuma conversa em foco.</strong>
        <span>Escolha uma na lista, ou abra uma sessão.</span>
        <button type="button" className="nav-item" onClick={on.newSession}>
          + Nova sessão <span className="nav-key">⌘N</span>
        </button>
      </div>
    );
  }
  const view = paneView(row, session, now);
  const ended = row.state === 'recorded' && !session;
  const info = session?.info;
  const contextWindow = info?.context_window ?? 0;
  const pct = session && session.contextTokens !== null && contextWindow > 0 ? Math.min(100, Math.floor((100 * session.contextTokens) / contextWindow)) : null;
  const others = crew.filter((id) => id !== row.id);
  return (
    <div className="crew">
      <section className="crew-thread" aria-label={row.title}>
        <header className="crew-head">
          <span className={`pane-glyph tone-${view.tone}${view.live ? ' dc-breath' : ''}`}>{view.glyph}</span>
          <span className="crew-title">{row.title}</span>
          {(info?.model ?? row.model) && (
            <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => on.model(row, e.currentTarget)}>
              {info?.model ?? row.model} ⌄
            </button>
          )}
        </header>
        {ended ? (
          <div className="crew-ended">
            <p>
              Terminada · {row.turns ?? 0} {row.turns === 1 ? 'turno' : 'turnos'}. Escrever continua a conversa numa sessão nova, com o histórico.
            </p>
            <div className="composer-wrap">
              <div className="composer">
                <Field
                  text={draft}
                  placeholder="Escreva para continuar esta conversa"
                  caret={caret}
                  onText={(t) => on.draft(row.id, t)}
                  onSubmit={() => draft.trim() && on.continueIn(row, draft)}
                />
              </div>
            </div>
          </div>
        ) : (
          <ConversationBody
            session={session ?? emptySession(row.id)}
            now={now}
            verbTick={verbTick}
            draft={draft}
            compact={false}
            caret={caret}
            onDraft={(t) => on.draft(row.id, t)}
            onAnswer={(decision) => session?.pendingApprovalId && on.answer(row.id, session.pendingApprovalId, decision)}
            onSend={(t) => on.send(row.id, t)}
            onStop={() => on.stop(row.id)}
            onMissing={on.missing}
          />
        )}
      </section>
      <aside className="crew-context" aria-label="Contexto da conversa">
        <Section title="Pronto">
          <DoneRow view={view} measure={measure} />
        </Section>
        <Section title="Arquivos" count={files.length}>
          {files.length === 0 ? (
            <p className="crew-quiet">Nenhum arquivo escrito ainda.</p>
          ) : (
            <ul className="crew-files">
              {files.map((f) => (
                <li key={f.path} className={f.failed ? 'failed' : undefined} title={f.path}>
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
          )}
        </Section>
        <Section title="Sessão">
          <dl className="crew-facts">
            <dt>projeto</dt>
            <dd className="mono" title={row.workspace}>
              {shortPath(row.workspace, 26)}
            </dd>
            {(info?.branch ?? row.branch) && (
              <>
                <dt>branch</dt>
                <dd className="mono">{info?.branch ?? row.branch}</dd>
              </>
            )}
            <dt>modelo</dt>
            <dd className="mono">{info?.model ?? row.model ?? '—'}</dd>
            {session?.sandbox && (
              <>
                <dt>modo</dt>
                <dd className="mono">
                  {session.sandbox} · {session.mode}
                </dd>
              </>
            )}
            <dt>turnos</dt>
            <dd>{session?.turns ?? row.turns ?? 0}</dd>
            {pct !== null && (
              <>
                <dt>contexto</dt>
                <dd>{pct}%</dd>
              </>
            )}
          </dl>
        </Section>
        <Section title="Crew" count={others.length}>
          {others.length === 0 ? (
            <p className="crew-quiet">Nada mais rodando ou esperando você.</p>
          ) : (
            <ul className="crew-mates">
              {others.map((id) => {
                const r = rows.get(id);
                if (!r) return null;
                const v = paneView(r, sessions.byId[id], now);
                return (
                  <li key={id}>
                    <button type="button" data-crew-id={id} onClick={() => on.focus(id)}>
                      <span className={`tone-${v.tone}${v.live ? ' dc-breath' : ''}`}>{v.glyph}</span>
                      <span className="crew-mate-title">{r.title}</span>
                      <span className="crew-mate-state">{stateWord(sessions.byId[id]?.state ?? r.state)}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </Section>
      </aside>
    </div>
  );
}
