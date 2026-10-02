import { useLayoutEffect, useRef } from 'react';
import type { SessionView } from '../state/session';

/**
 * The placeholder by state. During a turn, a message corrects it (D23); while
 * an approval waits, the answer comes first and the field takes nothing (D22).
 */
function placeholderOf(state: string): string {
  if (state === 'running') return 'Escreva para redirecionar este turno';
  if (state === 'blocked') return 'Responda à aprovação acima';
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
  const input = useRef<HTMLTextAreaElement>(null);
  // A field that stops taking text gives the keys back to the window, where
  // the approval's answers are (↵ among them).
  useLayoutEffect(() => {
    if (blocked && document.activeElement === input.current) input.current?.blur();
  }, [blocked]);
  const contextWindow = session.info?.context_window ?? 0;
  // Known only once a turn has ended: the daemon measures the context in
  // turn.completed, and a number derived here would be a second meter.
  const pct =
    session.contextTokens !== null && contextWindow > 0
      ? Math.min(100, Math.floor((100 * session.contextTokens) / contextWindow))
      : null;
  const chip = [session.sandbox, session.mode].filter(Boolean).join(' · ');
  const send = () => {
    if (!blocked && text.trim()) onSend(text);
  };
  return (
    <div className="composer-wrap">
      <div className="composer">
        <div className={`composer-field${empty ? ' empty' : ''}${blocked ? ' blocked' : ''}`}>
          <div className="composer-sizer" aria-hidden="true">
            {empty ? placeholderOf(session.state) : `${text}\n`}
          </div>
          <textarea
            ref={input}
            className="composer-input"
            aria-label={placeholderOf(session.state)}
            rows={1}
            value={text}
            disabled={blocked}
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
            <button type="button" className="composer-button send" aria-label="Enviar" disabled={blocked} onClick={send}>
              ↑
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
