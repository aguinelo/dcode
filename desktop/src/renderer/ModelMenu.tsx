import { useEffect, useRef } from 'react';
import type { ModelOption } from '../state/models';

const WIDTH = 320;

/**
 * Continues a conversation on another model (D28). Nothing changes model in
 * place: the daemon continues the conversation in a new session, history and
 * all, on what was chosen, and the one it leaves closes — which is what the
 * menu says it does.
 */
export function ModelMenu({
  anchor,
  options,
  problem,
  busy,
  onPick,
  onClose,
}: {
  /** The button it opens from; a click on it is the button's, not a click away. */
  anchor: HTMLElement;
  /** Null while the daemon has not answered. */
  options: ModelOption[] | null;
  /** Why there is no list, in the window's words. */
  problem: string | null;
  /** A turn runs or waits for an answer: switching now would cut it. */
  busy: boolean;
  onPick: (name: string) => void;
  onClose: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const away = (e: MouseEvent) => {
      const at = e.target as Node;
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
      const items = [...(box.current?.querySelectorAll<HTMLButtonElement>('.model-option:not(:disabled)') ?? [])];
      if (items.length === 0) return;
      e.preventDefault();
      const at = items.indexOf(document.activeElement as HTMLButtonElement);
      const next = e.key === 'ArrowDown' ? (at + 1) % items.length : at <= 0 ? items.length - 1 : at - 1;
      items[next]?.focus();
    };
    window.addEventListener('mousedown', away);
    window.addEventListener('keydown', key, true);
    return () => {
      window.removeEventListener('mousedown', away);
      window.removeEventListener('keydown', key, true);
    };
  }, [anchor, onClose]);
  // The first choice takes the focus once there are choices: the arrows move
  // it and ↵ picks, without the pointer.
  useEffect(() => {
    box.current?.querySelector<HTMLButtonElement>('.model-option:not(:disabled)')?.focus();
  }, [options]);

  // Above the button, right-aligned to it: the composer is at the bottom.
  const r = anchor.getBoundingClientRect();
  const left = Math.max(8, Math.min(r.right - WIDTH, window.innerWidth - WIDTH - 8));
  const bottom = Math.max(8, window.innerHeight - r.top + 6);
  return (
    <div ref={box} className="model-menu" style={{ left, bottom, width: WIDTH }} role="menu" aria-label="Continuar em outro modelo">
      <div className="model-menu-head">Continuar em outro modelo</div>
      {busy ? (
        <div className="model-menu-note">Pare o turno ou espere ele terminar para trocar de modelo.</div>
      ) : problem ? (
        <div className="model-menu-note tone-warn">{problem}</div>
      ) : options === null ? (
        <div className="model-menu-note">Perguntando ao daemon…</div>
      ) : (
        options.map((o) => (
          <button
            key={`${o.isDefault ? 'default' : 'profile'}:${o.name}`}
            type="button"
            role="menuitem"
            className={`model-option${o.current ? ' current' : ''}`}
            disabled={o.current}
            title={o.notice ?? undefined}
            onClick={() => onPick(o.name)}
          >
            <span className="model-name">{o.name}</span>
            <span className={`model-detail${o.isDefault ? '' : ' mono'}`}>{o.detail}</span>
            {o.current ? (
              <span className="model-badge">atual</span>
            ) : o.measured ? (
              <span className="model-badge ok" title="Os contratos de comportamento do dcode já foram medidos contra esta família.">
                medido
              </span>
            ) : (
              <span className="model-badge warn">sem medição</span>
            )}
          </button>
        ))
      )}
      <div className="model-menu-foot">A conversa continua numa sessão nova, com todo o histórico, e esta se encerra.</div>
    </div>
  );
}
