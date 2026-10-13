import { useEffect, type RefObject } from 'react';

/**
 * A menu that opens from a button: a click away or Esc closes it, the arrows
 * move between its enabled items, and the first of them takes the focus once
 * there are items — so ↵ picks without the pointer. `items` is what changes
 * when the list arrives.
 */
export function usePopover(box: RefObject<HTMLDivElement | null>, anchor: HTMLElement, onClose: () => void, selector: string, items: unknown): void {
  useEffect(() => {
    const away = (e: MouseEvent) => {
      const at = e.target as Node;
      // A click on the button is the button's, not a click away.
      if (box.current?.contains(at) || anchor.contains(at)) return;
      onClose();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
      const all = [...(box.current?.querySelectorAll<HTMLButtonElement>(`${selector}:not(:disabled)`) ?? [])];
      if (all.length === 0) return;
      e.preventDefault();
      const at = all.indexOf(document.activeElement as HTMLButtonElement);
      const next = e.key === 'ArrowDown' ? (at + 1) % all.length : at <= 0 ? all.length - 1 : at - 1;
      all[next]?.focus();
    };
    window.addEventListener('mousedown', away);
    window.addEventListener('keydown', key, true);
    return () => {
      window.removeEventListener('mousedown', away);
      window.removeEventListener('keydown', key, true);
    };
  }, [box, anchor, onClose, selector]);
  useEffect(() => {
    box.current?.querySelector<HTMLButtonElement>(`${selector}:not(:disabled)`)?.focus();
  }, [box, selector, items]);
}

/**
 * Where a menu of this width goes: right-aligned to its button, below one in
 * the upper half of the window — a panel's chip —, above one in the lower half.
 */
export function placeBy(anchor: HTMLElement, width: number): { left: number; top?: number; bottom?: number } {
  const r = anchor.getBoundingClientRect();
  const left = Math.max(8, Math.min(r.right - width, window.innerWidth - width - 8));
  return r.top < window.innerHeight / 2 ? { left, top: r.bottom + 6 } : { left, bottom: Math.max(8, window.innerHeight - r.top + 6) };
}
