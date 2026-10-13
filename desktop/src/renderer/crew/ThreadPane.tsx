import { ChevronDown } from 'lucide-react';
import type { History } from '../../state/history';
import type { GitState } from '../../state/git';
import type { PaneView } from '../../state/pane';
import { ContextLine } from '../context/ContextLine';
import { emptySession, type SessionView } from '../../state/session';
import type { Row } from '../../state/sidebar';
import { ConversationBody } from '../ConversationBody';
import { RecordedBody } from '../RecordedBody';
import { ConversationTitle } from '../RenameTitle';
import { ActionsButton } from '../SessionActions';
import type { CrewHandlers } from './handlers';

/**
 * The middle of the Painel: the conversation in focus, whole (D31) — its flow,
 * the approval it waits on, its own field, steering while it runs and stop.
 * An ended one continues in a new session when written to (D21).
 */
export function ThreadPane({
  row,
  view,
  session,
  now,
  verbTick,
  draft,
  caret,
  history,
  onRetryHistory,
  on,
  git,
}: {
  row: Row | null;
  view: PaneView | null;
  session: SessionView | undefined;
  now: number;
  verbTick: number;
  draft: string;
  caret?: number;
  /** An ended conversation's history, read from its record (D42); undefined until asked for. */
  history: History | undefined;
  onRetryHistory: () => void;
  on: CrewHandlers;
  /** The folder's git, for the line under the title (D41). */
  git: GitState;
}) {
  if (!row || !view) {
    return (
      <section className="crew-thread crew-empty">
        <strong>Nenhuma conversa em foco.</strong>
        <span>Escolha uma na lista de sessões, ou abra uma nova.</span>
        <button type="button" className="crew-primary" onClick={on.newSession}>
          + Nova sessão <span className="nav-key">⌘N</span>
        </button>
      </section>
    );
  }
  const ended = row.state === 'recorded' && !session;
  const model = session?.info?.model ?? row.model;
  return (
    <section className="crew-thread" aria-label={row.title}>
      <header className="crew-head">
        <span className={`pane-glyph tone-${view.tone}${view.live ? ' dc-breath' : ''}`}>{view.glyph}</span>
        <span className="crew-head-text">
          <ConversationTitle id={row.id} title={row.title} className="crew-title" />
          <span className="crew-where">
            <ContextLine row={row} state={git} />
          </span>
        </span>
        {model && (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => on.model(row, e.currentTarget)}>
            {model} <ChevronDown size={11} aria-hidden />
          </button>
        )}
        {session?.info && <ActionsButton title={row.title} onOpen={(anchor) => on.actions(row.id, anchor)} />}
      </header>
      {ended ? (
        <RecordedBody
          row={row}
          history={history}
          compact={false}
          now={now}
          draft={draft}
          caret={caret}
          onDraft={(t) => on.draft(row.id, t)}
          onContinue={(t) => on.continueIn(row, t)}
          onRetry={onRetryHistory}
        />
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
          onMode={session?.info ? (anchor) => on.mode(row.id, anchor) : undefined}
        />
      )}
    </section>
  );
}
