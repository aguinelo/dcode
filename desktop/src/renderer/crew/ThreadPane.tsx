import { ChevronDown } from 'lucide-react';
import type { PaneView } from '../../state/pane';
import { emptySession, type SessionView } from '../../state/session';
import type { Row } from '../../state/sidebar';
import { Field } from '../Composer';
import { ConversationBody } from '../ConversationBody';
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
  on,
}: {
  row: Row | null;
  view: PaneView | null;
  session: SessionView | undefined;
  now: number;
  verbTick: number;
  draft: string;
  caret?: number;
  on: CrewHandlers;
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
          <span className="crew-title">{row.title}</span>
          {view.where && <span className="crew-where mono">{view.where}</span>}
        </span>
        {model && (
          <button type="button" className="chip" title="Continuar em outro modelo" onClick={(e) => on.model(row, e.currentTarget)}>
            {model} <ChevronDown size={11} aria-hidden />
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
  );
}
