import { useMemo, useState, type KeyboardEvent } from 'react';
import { notRecalling, olderPrompts, promptsOf, recallDown, recallUp, sourceOf, wantsDown, wantsUp, type Recall } from '../state/recall';
import type { SessionView } from '../state/session';
import type { Prompts } from './usePrompts';

/**
 * A conversation's own keys in its field (D47), as the TUI's prompt line has
 * them: ↑/↓ at the field's edges walk what was asked before, ⌥↵ during a turn
 * queues instead of steering, ⌃X drops the last queued. A key another handler
 * already took (the "/" menu, D48) is left alone.
 */
export function useFieldKeys(session: SessionView, text: string, onText: (text: string) => void, prompts: Prompts | null) {
  const [walk, setWalk] = useState<{ id: string; recall: Recall }>({ id: session.id, recall: notRecalling });
  const recall = walk.id === session.id ? walk.recall : notRecalling;
  const source = useMemo(() => sourceOf(session.entries), [session.entries]);
  const older = source && prompts ? prompts.older(source) : null;
  const list = useMemo(() => olderPrompts(older ?? [], promptsOf(session.entries)), [older, session.entries]);

  const put = (r: Recall, t: string) => {
    setWalk({ id: session.id, recall: r });
    onText(t);
  };
  const reset = () => setWalk({ id: session.id, recall: notRecalling });

  const onKeys = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.defaultPrevented || e.nativeEvent.isComposing) return;
    const el = e.currentTarget;
    const plain = !e.shiftKey && !e.altKey && !e.metaKey && !e.ctrlKey;
    if (e.key === 'ArrowUp' && plain && wantsUp(text, el.selectionStart, el.selectionEnd, recall)) {
      const up = recallUp(list, recall, text);
      if (up) {
        e.preventDefault();
        put(up.recall, up.text);
      } else if (source && prompts && !older) {
        // At the oldest of a continuation: the earlier conversation's are read, and the next ↑ reaches them.
        prompts.readOlder(source);
      }
      return;
    }
    if (e.key === 'ArrowDown' && plain && wantsDown(text, el.selectionStart, el.selectionEnd, recall)) {
      const down = recallDown(list, recall);
      if (down) {
        e.preventDefault();
        put(down.recall, down.text);
      }
      return;
    }
    if (e.key === 'Enter' && e.altKey && !e.shiftKey && !e.metaKey && !e.ctrlKey && session.state === 'running' && prompts) {
      e.preventDefault();
      if (text.trim() && prompts.enqueue(session.id, text)) {
        onText('');
        reset();
      }
      return;
    }
    const queued = prompts?.queue(session.id);
    if (e.key.toLowerCase() === 'x' && e.ctrlKey && !e.metaKey && !e.altKey && !e.shiftKey && queued && el.selectionStart === el.selectionEnd) {
      e.preventDefault();
      prompts?.dropLast(session.id);
    }
  };

  return { onKeys, reset };
}
