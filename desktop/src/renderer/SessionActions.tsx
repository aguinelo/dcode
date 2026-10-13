import { useCallback, useRef, useState, type ReactNode } from 'react';
import type { SessionsState } from '../state/sessions';
import { placeBy, usePopover } from './popover';

const WIDTH = 300;

/** The ⋯ that opens a conversation's actions (D40). */
export function ActionsButton({ title, onOpen }: { title: string; onOpen: (anchor: HTMLElement) => void }) {
  return (
    <button type="button" className="icon-btn" aria-label={`Ações de ${title}`} aria-haspopup="menu" title="Desfazer, resumir, rodar um comando, renomear" onClick={(e) => onOpen(e.currentTarget)}>
      ⋯
    </button>
  );
}

/**
 * What acts on a live session besides a message (D40): undo the last turn,
 * compact the context, and run a command — the same as /undo, /compact and !
 * in the field — and rename it (D43). While a turn runs or waits the daemon refuses the first two,
 * and the menu says so instead of offering them; renaming stays offered.
 */
function ActionsMenu({
  anchor,
  busy,
  onUndo,
  onCompact,
  onCommand,
  onRename,
  onClose,
}: {
  anchor: HTMLElement;
  busy: boolean;
  onUndo: () => void;
  onCompact: () => void;
  onCommand: () => void;
  onRename: () => void;
  onClose: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  usePopover(box, anchor, onClose, '.actions-option', busy);
  const items = [
    { label: 'Desfazer o último turno', detail: 'põe de volta os arquivos que ele mudou', key: '/undo', run: onUndo, idle: true },
    { label: 'Resumir o contexto', detail: 'compacta agora o que o modelo carrega', key: '/compact', run: onCompact, idle: true },
    { label: 'Rodar um comando…', detail: 'no campo, depois de !; a saída aparece aberta', key: '!', run: onCommand, idle: true },
    // Naming works in the middle of a turn too: the daemon takes it at any time (D43).
    { label: 'Renomear', detail: 'o título desta conversa; vazio volta ao automático', key: '2× clique', run: onRename, idle: false },
  ];
  return (
    <div ref={box} className="model-menu" style={{ width: WIDTH, ...placeBy(anchor, WIDTH) }} role="menu" aria-label="Ações da conversa">
      <div className="model-menu-head">Ações da conversa</div>
      {busy && <div className="model-menu-note">Pare o turno ou espere ele terminar: o daemon não desfaz, resume nem roda nada no meio de um.</div>}
      {items.map((i) => (
        <button
          key={i.key}
          type="button"
          role="menuitem"
          className="actions-option"
          disabled={busy && i.idle}
          onClick={() => {
            onClose();
            i.run();
          }}
        >
          <span className="actions-label">{i.label}</span>
          <span className="actions-detail">{i.detail}</span>
          <span className="actions-key">{i.key}</span>
        </button>
      ))}
    </div>
  );
}

/** The actions menu of one conversation at a time, and what each item does. */
export function useSessionActions(
  sessions: SessionsState,
  run: { undo: (id: string) => unknown; compact: (id: string) => unknown; command: (id: string) => void; rename: (id: string) => void },
): { open: (id: string, anchor: HTMLElement) => void; menu: boolean; node: ReactNode } {
  const [menu, setMenu] = useState<{ id: string; anchor: HTMLElement } | null>(null);
  const open = useCallback((id: string, anchor: HTMLElement) => setMenu((m) => (m?.id === id ? null : { id, anchor })), []);
  const close = useCallback(() => setMenu(null), []);
  const state = menu ? sessions.byId[menu.id]?.state : undefined;
  const node = menu ? (
    <ActionsMenu
      anchor={menu.anchor}
      busy={state === 'running' || state === 'blocked'}
      onUndo={() => void run.undo(menu.id)}
      onCompact={() => void run.compact(menu.id)}
      onCommand={() => run.command(menu.id)}
      onRename={() => run.rename(menu.id)}
      onClose={close}
    />
  ) : null;
  return { open, menu: !!menu, node };
}
