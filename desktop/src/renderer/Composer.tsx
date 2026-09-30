import type { SessionView } from '../state/session';

/**
 * The placeholder by state. While a turn runs the design says the message
 * queues; the TUI's spec says a word typed during a turn steers it. The
 * design's text stays and neither behaviour is built here (docs/DECISIONS.md).
 */
function placeholderOf(state: string): string {
  if (state === 'running') return 'Mensagem entra na fila depois deste turno';
  if (state === 'blocked') return 'Ou diga o que fazer em vez disso';
  return 'Escreva uma mensagem';
}

/** The context meter: faint to 74%, warn to 89%, err from 90% (the brief's bands). */
function meterTone(pct: number): string {
  if (pct >= 90) return 'tone-err';
  if (pct >= 75) return 'tone-warn';
  return 'tone-faint';
}

export function Composer({
  session,
  text,
  onText,
  onSend,
  onStop,
  onMissing,
}: {
  session: SessionView;
  /** The draft, kept per session by the window so switching away loses nothing. */
  text: string;
  onText: (text: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
  onMissing: (what: string) => void;
}) {
  const running = session.state === 'running';
  const blocked = session.state === 'blocked';
  const empty = text.length === 0;
  const contextWindow = session.info?.context_window ?? 0;
  // Known only once a turn has ended: the daemon measures the context in
  // turn.completed, and a number derived here would be a second meter.
  const pct =
    session.contextTokens !== null && contextWindow > 0
      ? Math.min(100, Math.floor((100 * session.contextTokens) / contextWindow))
      : null;
  const chip = [session.sandbox, session.mode].filter(Boolean).join(' · ');
  const send = () => {
    if (text.trim()) onSend(text);
  };
  return (
    <div className="composer-wrap">
      <div className="composer">
        <div className={`composer-field${empty ? ' empty' : ''}${blocked ? ' blocked' : ''}`}>
          <div className="composer-sizer" aria-hidden="true">
            {empty ? (
              <>
                {placeholderOf(session.state)}
                {blocked && <span className="composer-caret dc-caret" />}
              </>
            ) : (
              `${text}\n`
            )}
          </div>
          <textarea
            className="composer-input"
            aria-label={placeholderOf(session.state)}
            rows={1}
            value={text}
            onChange={(e) => onText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                e.preventDefault();
                send();
              }
            }}
          />
        </div>
        <div className="composer-bar">
          <button type="button" className="composer-attach" onClick={() => onMissing('Anexar')}>
            +
          </button>
          {chip && <span className="composer-chip">{chip}</span>}
          <span className="spacer" />
          {session.info?.model && (
            <button type="button" className="composer-model" onClick={() => onMissing('Trocar de modelo')}>
              {session.info.model} <span className="faint">⌄</span>
            </button>
          )}
          {pct !== null && <span className={`composer-meter ${meterTone(pct)}`}>{pct}%</span>}
          {running ? (
            <button type="button" className="composer-button stop" aria-label="Parar o turno" onClick={onStop}>
              <span className="stop-square" />
            </button>
          ) : (
            <button type="button" className="composer-button send" aria-label="Enviar" onClick={send}>
              ↑
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
