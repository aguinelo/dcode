import { useEffect, useLayoutEffect, useRef } from 'react';
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

/**
 * The text field itself: grows with what is typed, ↵ sends and ⇧↵ breaks the
 * line. ⌘↵ is the window's, and Esc leaves the field so the window's keys —
 * the arrows between windows, an approval's answers — apply again.
 */
export function Field({
  text,
  placeholder,
  disabled = false,
  caret,
  onText,
  onSubmit,
}: {
  text: string;
  placeholder: string;
  disabled?: boolean;
  /** Changes when the caret is asked into this field. */
  caret?: number;
  onText: (text: string) => void;
  onSubmit: () => void;
}) {
  const input = useRef<HTMLTextAreaElement>(null);
  // A field that stops taking text gives the keys back to the window, where
  // the approval's answers are (↵ among them).
  useLayoutEffect(() => {
    if (disabled && document.activeElement === input.current) input.current?.blur();
  }, [disabled]);
  useEffect(() => {
    if (caret) input.current?.focus();
  }, [caret]);
  const empty = text.length === 0;
  return (
    <div className={`composer-field${empty ? ' empty' : ''}${disabled ? ' blocked' : ''}`}>
      <div className="composer-sizer" aria-hidden="true">
        {empty ? placeholder : `${text}\n`}
      </div>
      <textarea
        ref={input}
        className="composer-input"
        aria-label={placeholder}
        rows={1}
        value={text}
        disabled={disabled}
        onChange={(e) => onText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            // Only out of the field: a second Esc is the window's.
            e.stopPropagation();
            e.currentTarget.blur();
            return;
          }
          if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey && !e.nativeEvent.isComposing) {
            e.preventDefault();
            onSubmit();
          }
        }}
      />
    </div>
  );
}

export function Composer({
  session,
  text,
  compact = false,
  caret,
  onText,
  onSend,
  onStop,
  onMissing,
}: {
  session: SessionView;
  /** The draft, kept per session by the window so switching away loses nothing. */
  text: string;
  /** In a grid window: the field and its button, without the bar's extras. */
  compact?: boolean;
  caret?: number;
  onText: (text: string) => void;
  onSend: (text: string) => void;
  onStop: () => void;
  onMissing: (what: string) => void;
}) {
  const running = session.state === 'running';
  const blocked = session.state === 'blocked';
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
    <div className={`composer-wrap${compact ? ' compact' : ''}`}>
      <div className="composer">
        <Field text={text} placeholder={placeholderOf(session.state)} disabled={blocked} caret={caret} onText={onText} onSubmit={send} />
        <div className="composer-bar">
          {!compact && (
            <button type="button" className="composer-attach" onClick={() => onMissing('Anexar')}>
              +
            </button>
          )}
          {!compact && chip && <span className="composer-chip">{chip}</span>}
          <span className="spacer" />
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
