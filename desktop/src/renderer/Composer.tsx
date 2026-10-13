import { useContext, useEffect, useLayoutEffect, useRef, useState, type DragEvent, type KeyboardEvent } from 'react';
import { imageSource, QUEUE_TAKES_TEXT, sizeOf } from '../state/attachments';
import { SheetContext } from './CommandSheet';
import { useSlashMenu } from './SlashMenu';
import type { SessionView } from '../state/session';
import { turnTelemetry, usageFact } from '../state/telemetry';
import { ModeChip } from './ModeMenu';
import { QueueChips } from './QueueChips';
import { useFieldKeys } from './useFieldKeys';
import { useAttached, useImages } from './useImages';
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

/** The pictures a paste carries: only images, since the text of a paste is the field's to take. */
function imageFiles(data: DataTransfer): File[] {
  return [...data.files].filter((f) => f.type.startsWith('image/'));
}

/**
 * The pictures attached to the next message (D46), each with its ×, and the
 * last refusal under them. Nothing is drawn while there are neither.
 */
function Attachments({ sessionId }: { sessionId: string }) {
  const images = useImages();
  const attached = useAttached();
  const draft = attached.draft[sessionId] ?? [];
  const refusal = attached.refusal[sessionId];
  if (!images || (draft.length === 0 && !refusal)) return null;
  return (
    <div className="composer-images">
      {draft.length > 0 && (
        <div className="composer-thumbs">
          {draft.map((img, i) => (
            <figure key={`${i}-${img.name}`} className="composer-thumb" title={`${img.name} · ${sizeOf(img.bytes)}`}>
              <img src={imageSource(img)} alt={img.name} />
              <button type="button" className="composer-thumb-remove" aria-label={`Tirar ${img.name}`} onClick={() => images.remove(sessionId, i)}>
                ×
              </button>
            </figure>
          ))}
        </div>
      )}
      {refusal && (
        <div className="composer-refusal tone-err" role="alert">
          {refusal}
        </div>
      )}
    </div>
  );
}

/**
 * The text field itself: grows with what is typed, ↵ sends and ⇧↵ breaks the
 * line; ↑/↓, ⌥↵ and ⌃X are the conversation's (D47). ⌘↵ is the window's, and Esc leaves the field so the window's keys —
 * the arrows between windows, an approval's answers — apply again. With
 * `commands`, a leading / opens the menu of commands, which owns the arrows
 * while open, and ? in an empty field opens the help (D48).
 */
export function Field({
  text,
  placeholder,
  disabled = false,
  commands = false,
  caret,
  onText,
  onSubmit,
  onKeys,
  onPasteImages,
}: {
  text: string;
  placeholder: string;
  disabled?: boolean;
  /** Whether the field takes the harness's commands: the / menu and ? for the help. */
  commands?: boolean;
  /** Changes when the caret is asked into this field. */
  caret?: number;
  onText: (text: string) => void;
  /** Sends what is in the field, or the text given, which the field already shows. */
  onSubmit: (text?: string) => void;
  /** The conversation's own keys — history, queue (D47) — before ↵ sends; one that acts prevents the default. */
  onKeys?: (e: KeyboardEvent<HTMLTextAreaElement>) => void;
  /** A paste that carries pictures (D46): they attach, and nothing is typed. */
  onPasteImages?: (files: File[]) => void;
}) {
  const input = useRef<HTMLTextAreaElement>(null);
  const sheet = useContext(SheetContext);
  const menu = useSlashMenu(text, commands && !disabled, onText, onSubmit);
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
      {menu.node}
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
        onChange={(e) => {
          menu.reset();
          onText(e.target.value);
        }}
        onPaste={(e) => {
          const files = imageFiles(e.clipboardData);
          if (!onPasteImages || files.length === 0) return;
          e.preventDefault();
          onPasteImages(files);
        }}
        onKeyDown={(e) => {
          menu.keys(e);
          if (e.defaultPrevented) return;
          if (commands && sheet && e.key === '?' && text === '') {
            e.preventDefault();
            sheet({ kind: 'help' });
            return;
          }
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
  const send = (t: string = text) => {
    if (blocked || !t.trim()) return;
    keys.reset();
    onSend(t);
  };
  // Pictures (D46): "+" opens the picker, a paste or a drop attaches.
  const images = useImages();
  const [dropping, setDropping] = useState(false);
  const attach = images ? (files: File[]) => void images.addFiles(session.id, files) : undefined;
  const carriesFiles = (e: DragEvent) => !!images && e.dataTransfer.types.includes('Files');
  // The queue (D47) carries text: a message with pictures does not wait in it, and says why.
  const withImages = (useAttached().draft[session.id]?.length ?? 0) > 0;
  const onKeys = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    const queueing = e.key === 'Enter' && e.altKey && !e.shiftKey && !e.metaKey && !e.ctrlKey && session.state === 'running';
    if (queueing && withImages && images) {
      e.preventDefault();
      images.refuse(session.id, QUEUE_TAKES_TEXT);
      return;
    }
    keys.onKeys(e);
  };
  return (
    <div
      className={`composer-wrap${compact ? ' compact' : ''}${dropping ? ' dropping' : ''}`}
      onDragOver={(e) => {
        if (!carriesFiles(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'copy';
        setDropping(true);
      }}
      onDragLeave={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDropping(false);
      }}
      onDrop={(e) => {
        if (!carriesFiles(e)) return;
        e.preventDefault();
        setDropping(false);
        attach?.([...e.dataTransfer.files]);
      }}
    >
      <div className="composer">
        {prompts && (
          <QueueChips queue={prompts.queue(session.id)} onDrop={(i) => prompts.dropAt(session.id, i)} onResume={() => prompts.resume(session.id)} />
        )}
        <Attachments sessionId={session.id} />
        <Field
          text={text}
          placeholder={placeholderOf(session.state)}
          disabled={blocked}
          commands
          caret={caret}
          onText={onText}
          onSubmit={send}
          onKeys={onKeys}
          onPasteImages={attach}
        />
        <div className="composer-bar">
          {!compact && (
            <button
              type="button"
              className="composer-attach"
              aria-label="Anexar imagens"
              onClick={() => (images ? void images.pick(session.id) : onMissing('Anexar'))}
            >
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
            <button type="button" className="composer-button send" aria-label="Enviar" disabled={blocked} onClick={() => send()}>
              ↑
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
