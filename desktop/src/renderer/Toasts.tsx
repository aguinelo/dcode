import { MascotGlyph } from './Mascot';

export type Toast = { id: number; text: string };

/** What the window says where the person looks, each dismissed by a click. */
export function Toasts({ toasts, onDismiss }: { toasts: readonly Toast[]; onDismiss: (id: number) => void }) {
  if (toasts.length === 0) return null;
  return (
    <div className="toasts" role="status" aria-live="polite">
      {toasts.map((t) => (
        <button key={t.id} type="button" className="toast" onClick={() => onDismiss(t.id)}>
          <MascotGlyph state="bug" size={16} className="toast-glyph tone-err" />
          <span>{t.text}</span>
        </button>
      ))}
    </div>
  );
}
