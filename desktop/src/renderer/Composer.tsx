import { useEffect, useLayoutEffect, useRef, type KeyboardEvent } from 'react';
import type { SessionView } from '../state/session';
import { turnTelemetry, usageFact } from '../state/telemetry';
import { ModeChip } from './ModeMenu';
import { QueueChips } from './QueueChips';
import { useFieldKeys } from './useFieldKeys';
import { usePromptsContext } from './usePrompts';

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
 * line; ↑/↓, ⌥↵ and ⌃X are the conversation's (D47). ⌘↵ is the window's, and Esc leaves the field so the window's keys —
 * the arrows between windows, an approval's answers — apply again.
 */
export function Field({
  text,
  placeholder,
  disabled = false,
  caret,
  onText,
  onSubmit,
  onKeys,
}: {
  text: string;
  placeholder: string;
  disabled?: boolean;
  /** Changes when the caret is asked into this field. */
  caret?: number;
  onText: (text: string) => void;
  onSubmit: () => void;
  /** The conversation's own keys — history, queue (D47) — before ↵ sends; one that acts prevents the default. */
  onKeys?: (e: KeyboardEvent<HTMLTextAreaElement>) => void;
}) {
  const input = useRef<HTMLTextAreaElement>(null);
  // A field that stops taking text gives the keys back to the window, where
  // the approval's answers are (↵ among them).
  useLayoutEffect(() => {
    if (disabled && document.activeElement === input.current) input.current?.blur();
  }, [disabled]);
  useEffect(() => {
    const el = input.current;
    if (!caret || !el) return;
    el.focus();
    // After what is already there — a draft, or the ! a command starts with (D40).
    el.setSelectionRange(el.value.length, el.value.length);
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
          onKeys?.(e);
          if (e.defaultPrevented) return;
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
  onMode,
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
  /** Opens the mode menu from the chip (D36). */
  onMode?: (anchor: HTMLElement) => void;
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
  // The turn against its ceilings (D44): drawn only near one, said whole on hover.
  const live = turnTelemetry(session);
  const ceilings = live ? [live.rounds, live.inFlight].filter((c) => c !== null) : [];
  const spent = session.usage ? usageFact(session.usage) : null;
  const prompts = usePromptsContext();
  const keys = useFieldKeys(session, text, onText, prompts);
  const send = () => {
    if (blocked || !text.trim()) return;
    keys.reset();
    onSend(text);
  };
  return (
    <div className={`composer-wrap${compact ? ' compact' : ''}`}>
      <div className="composer">
        {prompts && (
          <QueueChips queue={prompts.queue(session.id)} onDrop={(i) => prompts.dropAt(session.id, i)} onResume={() => prompts.resume(session.id)} />
        )}
        <Field
          text={text}
          placeholder={placeholderOf(session.state)}
          disabled={blocked}
          caret={caret}
          onText={onText}
          onSubmit={send}
          onKeys={keys.onKeys}
        />
        <div className="composer-bar">
          {!compact && (
            <button type="button" className="composer-attach" onClick={() => onMissing('Anexar')}>
              +
            </button>
          )}
          <ModeChip mode={session.mode} sandbox={session.sandbox} onOpen={onMode} />
          <span className="spacer" />
          {ceilings.length > 0 && (
            <span className="composer-ceiling" data-ceiling title={live?.detail}>
              {ceilings.map((c) => (
                <span key={c.text} className={`tone-${c.tone}`}>
                  {c.text}
                </span>
              ))}
            </span>
          )}
          {pct !== null && (
            <span className={`composer-meter ${meterTone(pct)}`} title={spent ? `Último turno: ${spent.detail}` : undefined}>
              {pct}%
            </span>
          )}
          {running ? (
            <button type="button" className="composer-button stop" aria-label="Parar o turno" title={live?.detail} onClick={onStop}>
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
