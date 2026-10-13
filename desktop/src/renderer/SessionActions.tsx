import { useCallback, useRef, useState, type ReactNode } from 'react';
import { endWarning } from '../state/end';
import type { SessionsState } from '../state/sessions';
import { placeBy, usePopover } from './popover';

const WIDTH = 300;

/** The ⋯ that opens a conversation's actions (D40). */
export function ActionsButton({ title, onOpen }: { title: string; onOpen: (anchor: HTMLElement) => void }) {
  return (
    <button type="button" className="icon-btn" aria-label={`Ações de ${title}`} aria-haspopup="menu" title="Desfazer, resumir, rodar um comando, renomear, encerrar" onClick={(e) => onOpen(e.currentTarget)}>
      ⋯
    </button>
  );
}

/**
 * What acts on a live session besides a message (D40): undo the last turn,
 * compact the context, and run a command — the same as /undo, /compact and !
 * in the field — rename it (D43), open its project's memory and skills (D53),
 * and end it (D51). While a turn runs or waits the daemon refuses the first two,
 * and the menu says so instead of offering them; renaming and the project's
 * context stay offered, and ending asks once more, saying what it would cut short.
 */
function ActionsMenu({
  anchor,
  state,
  onUndo,
  onCompact,
  onCommand,
  onRename,
  onContext,
  onEnd,
  onClose,
}: {
  anchor: HTMLElement;
  state: string | undefined;
  onUndo: () => void;
  onCompact: () => void;
  onCommand: () => void;
  onRename: () => void;
  onContext: () => void;
  onEnd: () => void;
  onClose: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  const busy = state === 'running' || state === 'blocked';
  // What ending would cut short, once "Encerrar" was asked for in the middle of a turn.
  const [confirming, setConfirming] = useState<string | null>(null);
  usePopover(box, anchor, onClose, '.actions-option', `${busy}${confirming ?? ''}`);
  const items = [
    { label: 'Desfazer o último turno', detail: 'põe de volta os arquivos que ele mudou', key: '/undo', run: onUndo, idle: true },
    { label: 'Resumir o contexto', detail: 'compacta agora o que o modelo carrega', key: '/compact', run: onCompact, idle: true },
    { label: 'Rodar um comando…', detail: 'no campo, depois de !; a saída aparece aberta', key: '!', run: onCommand, idle: true },
    // Naming works in the middle of a turn too: the daemon takes it at any time (D43).
    { label: 'Renomear', detail: 'o título desta conversa; vazio volta ao automático', key: '2× clique', run: onRename, idle: false },
    // Reading, so offered during a turn too: the Crew's own sections, over the grid (D53).
    { label: 'Memória e skills…', detail: 'o que as sessões deste projeto leem', key: 'projeto', run: onContext, idle: false },
    // Offered during a turn too, as the stop button is: asked once more, saying what stops (D51).
    { label: 'Encerrar sessão', detail: 'fecha a sessão; a conversa fica na lista, terminada', key: 'encerrar', run: onEnd, idle: false },
  ];
  if (confirming) {
    return (
      <div ref={box} className="model-menu" style={{ width: WIDTH, ...placeBy(anchor, WIDTH) }} role="menu" aria-label="Encerrar a sessão">
        <div className="model-menu-head">Encerrar a sessão?</div>
        <div className="model-menu-note">{confirming}</div>
        {/* Cancel first: the focus lands on it, so a stray ↵ keeps the turn. */}
        <button type="button" role="menuitem" className="actions-option" onClick={onClose}>
          <span className="actions-label">Cancelar</span>
          <span className="actions-detail">a sessão segue como está</span>
          <span className="actions-key">esc</span>
        </button>
        <button
          type="button"
          role="menuitem"
          className="actions-option"
          onClick={() => {
            onClose();
            onEnd();
          }}
        >
          <span className="actions-label">Encerrar mesmo assim</span>
          <span className="actions-detail">a conversa fica na lista, terminada</span>
        </button>
      </div>
    );
  }
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
            const warn = i.run === onEnd ? endWarning(state) : null;
            if (warn) return setConfirming(warn);
            onClose();
            i.run();
          }}
        >
          <span className="actions-label">{i.label}</span>
          <span className="actions-detail">{i.detail}</span>
          <span className="actions-key">{i.run === onEnd ? '' : i.key}</span>
        </button>
      ))}
    </div>
  );
}

/** The actions menu of one conversation at a time, and what each item does. */
export function useSessionActions(
  sessions: SessionsState,
  run: { undo: (id: string) => unknown; compact: (id: string) => unknown; command: (id: string) => void; rename: (id: string) => void; context: (id: string) => void; end: (id: string) => unknown },
): { open: (id: string, anchor: HTMLElement) => void; menu: boolean; node: ReactNode } {
  const [menu, setMenu] = useState<{ id: string; anchor: HTMLElement } | null>(null);
  const open = useCallback((id: string, anchor: HTMLElement) => setMenu((m) => (m?.id === id ? null : { id, anchor })), []);
  const close = useCallback(() => setMenu(null), []);
  const state = menu ? sessions.byId[menu.id]?.state : undefined;
  const node = menu ? (
    <ActionsMenu
      key={menu.id}
      anchor={menu.anchor}
      state={state}
      onUndo={() => void run.undo(menu.id)}
      onCompact={() => void run.compact(menu.id)}
      onCommand={() => run.command(menu.id)}
      onRename={() => run.rename(menu.id)}
      onContext={() => run.context(menu.id)}
      onEnd={() => void run.end(menu.id)}
      onClose={close}
    />
  ) : null;
  return { open, menu: !!menu, node };
}
