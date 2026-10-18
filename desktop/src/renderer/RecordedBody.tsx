import { useLayoutEffect, useMemo, useRef } from 'react';
import { flowBlocks } from '../state/flow';
import type { History } from '../state/history';
import { ENDED_AGENT, openPathOf, type OpenPath } from '../state/questions';
import type { Row } from '../state/sidebar';
import { Field } from './Composer';
import { FlowBlocks } from './Flow';

const nothing = () => {};

/**
 * A conversation that ended, read from its record (D42): its flow as it ran,
 * read-only, and below it the field that continues it in a new session. While
 * the record is read it says so; when it cannot be read it says why, with a
 * way to try again — never an empty thread that looks like nothing happened.
 */
export function RecordedBody({
  row,
  history,
  compact,
  now,
  draft,
  caret,
  onDraft,
  onContinue,
  onRetry,
}: {
  row: Row;
  /** Undefined until it is asked for: drawn as being read. */
  history: History | undefined;
  /** A window on the grid: the smaller hand. */
  compact: boolean;
  now: number;
  draft: string;
  caret?: number;
  onDraft: (text: string) => void;
  onContinue: (text: string) => void;
  onRetry: () => void;
}) {
  const path = endedPath(row);
  const view = history?.state === 'loaded' ? history.view : null;
  const blocks = useMemo(() => (view ? flowBlocks(view.entries) : []), [view]);
  const scroller = useRef<HTMLDivElement>(null);
  // Opens at its end, where the conversation stopped.
  useLayoutEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [blocks]);

  let flow: React.ReactNode;
  if (!history || history.state === 'loading') {
    flow = (
      <p className="history-note" role="status">
        Lendo o histórico desta conversa…
      </p>
    );
  } else if (history.state === 'failed') {
    flow = (
      <div className="history-note failed" role="alert">
        <span>{history.reason}</span>
        <button type="button" className="history-retry" onClick={onRetry}>
          Tentar de novo
        </button>
      </div>
    );
  } else {
    flow = (
      <>
        {blocks.length === 0 && <p className="history-note">O registro desta conversa não tem nada além do começo dela.</p>}
        <FlowBlocks blocks={blocks} now={now} onAnswer={nothing} />
        {history.cut > 0 && (
          <p className="history-note warn" role="note">
            Mostrando {history.total - history.cut} de {history.total} eventos: o fim desta conversa não foi lido.
          </p>
        )}
      </>
    );
  }

  return (
    <>
      <div className="flow-scroll" ref={scroller}>
        <div className="flow" aria-label="Histórico da conversa">
          {flow}
        </div>
      </div>
      <p className="recorded-ended">
        {path === 'orchestrator' ? 'Orquestradora do projeto' : 'Terminada'} · {row.turns ?? 0} {row.turns === 1 ? 'turno' : 'turnos'}. {ENDED_LINE[path]}
      </p>
      {path !== 'ended-agent' && (
        <div className={`composer-wrap${compact ? ' compact' : ''}`}>
          <div className="composer">
            <Field
              text={draft}
              placeholder={path === 'orchestrator' ? 'Escreva para a orquestradora' : 'Escreva para continuar esta conversa'}
              caret={caret}
              onText={onDraft}
              onSubmit={() => draft.trim() && onContinue(draft)}
            />
          </div>
        </div>
      )}
    </>
  );
}

/**
 * What writing below an ended conversation does. An ordinary one continues in
 * a new session (D42); an orchestrator is never continued — writing opens the
 * live one of its project and sends it there; a child or a reviewer has no
 * field at all, because its work goes on through the orchestrator.
 */
const ENDED_LINE: Record<Exclude<OpenPath, 'follow'>, string> = {
  continue: 'Escrever continua a conversa numa sessão nova, com o histórico.',
  orchestrator: 'Escrever abre a orquestradora viva do projeto e manda a mensagem a ela — uma orquestradora nunca continua numa sessão nova.',
  'ended-agent': ENDED_AGENT,
};

function endedPath(row: Row): Exclude<OpenPath, 'follow'> {
  const path = openPathOf({ ...row, state: 'recorded' });
  return path === 'follow' ? 'continue' : path;
}
