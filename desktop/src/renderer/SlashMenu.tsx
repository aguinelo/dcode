import { useState, type KeyboardEvent } from 'react';
import type { CommandInfo } from '../protocol/generated';
import { completions, type Command } from '../state/commands';

/**
 * The "/" menu of a field (D48): open while the field is a slash and a name
 * being typed. ↑ ↓ choose, ⇥ completes, ↵ completes and sends a command that
 * needs nothing more, esc closes it for what is typed now. While open it owns
 * those keys: `keys` prevents their default, and whatever handles the field's
 * keys after it leaves a prevented key alone.
 */
export function useSlashMenu(
  text: string,
  on: boolean,
  write: (text: string) => void,
  send: (text: string) => void,
  /** The commands the person and the project wrote (D50), after the field's own. */
  user: readonly CommandInfo[] = [],
  /** Why the project's commands are not in the list, said at its foot. */
  missing: string | null = null,
) {
  const [at, setAt] = useState(0);
  // The text the menu was closed on: it stays closed until the text changes.
  const [shut, setShut] = useState<string | null>(null);
  const list = on && shut !== text ? completions(text, user) : [];
  const cur = Math.min(at, Math.max(0, list.length - 1));

  const choose = (c: Command, submit: boolean) => {
    setAt(0);
    write(c.insert);
    if (submit && c.ready) send(c.insert);
  };

  const keys = (e: KeyboardEvent<HTMLTextAreaElement>): void => {
    if (list.length === 0 || e.nativeEvent.isComposing) return;
    const c = list[cur];
    if (!c) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      const n = list.length;
      setAt((cur + (e.key === 'ArrowDown' ? 1 : n - 1)) % n);
    } else if (e.key === 'Tab' && !e.shiftKey) {
      e.preventDefault();
      choose(c, false);
    } else if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey) {
      e.preventDefault();
      choose(c, true);
    } else if (e.key === 'Escape') {
      // Closes the menu only; a second esc leaves the field.
      e.preventDefault();
      e.stopPropagation();
      setShut(text);
    }
  };

  const node =
    list.length > 0 ? (
      <div className="slash-menu" role="listbox" aria-label="Comandos">
        {list.map((c, i) => (
          <div
            key={c.name}
            role="option"
            aria-selected={i === cur}
            className={`slash-option${i === cur ? ' current' : ''}`}
            // Down, not click: the field keeps the focus, and the caret with it.
            onMouseDown={(e) => {
              e.preventDefault();
              choose(c, true);
            }}
            onMouseEnter={() => setAt(i)}
          >
            <span className="slash-name">
              {c.name === '!' ? '!' : `/${c.name}`}
              {c.args && <span className="slash-args"> {c.args}</span>}
            </span>
            <span className="slash-detail">
              {c.detail}
              {c.origin && <span className="slash-origin"> · {c.origin}</span>}
            </span>
          </div>
        ))}
        {missing && <div className="slash-missing tone-err">{missing}</div>}
      </div>
    ) : null;

  return { keys, node, open: list.length > 0, reset: () => setAt(0) };
}
