import { createContext, useCallback, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react';
import { COMMANDS, planSummary } from '../state/commands';
import { HELP_KEYS } from '../state/help';
import type { SessionsState } from '../state/sessions';

/** What opens over the window from a command (D48): the help, or one conversation's plan. */
export type Sheet = { kind: 'help' } | { kind: 'plan'; id: string };

/** Lets any field open the help with `?`, without every window in between carrying it. */
export const SheetContext = createContext<((s: Sheet) => void) | null>(null);

function glyph(status: string): string {
  return status === 'done' ? '✓' : status === 'active' ? '●' : status === 'blocked' ? '⊘' : '·';
}

function Help() {
  return (
    <>
      <section className="sheet-group">
        <h3 className="sheet-group-title">Comandos</h3>
        {COMMANDS.map((c) => (
          <div key={c.name} className="sheet-row">
            <span className="sheet-key">
              {c.name === '!' ? '!' : `/${c.name}`}
              {c.args && <span className="sheet-args"> {c.args}</span>}
            </span>
            <span className="sheet-what">{c.detail}</span>
          </div>
        ))}
      </section>
      {HELP_KEYS.map((g) => (
        <section key={g.title} className="sheet-group">
          <h3 className="sheet-group-title">{g.title}</h3>
          {g.keys.map((k) => (
            <div key={k.key} className="sheet-row">
              <span className="sheet-key">{k.key}</span>
              <span className="sheet-what">{k.what}</span>
            </div>
          ))}
        </section>
      ))}
    </>
  );
}

/**
 * The help or a plan, over the window: Esc, a click outside or "Fechar" closes
 * it, and what was in focus before gets the focus back.
 */
function SheetView({ title, note, onClose, children }: { title: string; note?: string; onClose: () => void; children: ReactNode }) {
  const box = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    const before = document.activeElement as HTMLElement | null;
    box.current?.focus();
    return () => before?.focus?.();
  }, []);
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      e.stopPropagation();
      onClose();
    };
    window.addEventListener('keydown', key, true);
    return () => window.removeEventListener('keydown', key, true);
  }, [onClose]);
  return (
    <div className="sheet-veil" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div ref={box} className="sheet" role="dialog" aria-modal="true" aria-label={title} tabIndex={-1}>
        <div className="sheet-head">
          <h2 className="sheet-title">{title}</h2>
          {note && <span className="sheet-note">{note}</span>}
          <span className="spacer" />
          <button type="button" className="sheet-close" onClick={onClose}>
            Fechar <span className="sheet-key">esc</span>
          </button>
        </div>
        <div className="sheet-body">{children}</div>
      </div>
    </div>
  );
}

/** The sheet the window has open, if any, and how commands open one. */
export function useCommandSheet(sessions: SessionsState): { show: (s: Sheet) => void; open: boolean; node: ReactNode } {
  const [sheet, setSheet] = useState<Sheet | null>(null);
  const close = useCallback(() => setSheet(null), []);
  let node: ReactNode = null;
  if (sheet?.kind === 'help') {
    node = (
      <SheetView title="Ajuda" note="comandos e atalhos" onClose={close}>
        <Help />
      </SheetView>
    );
  } else if (sheet?.kind === 'plan') {
    // The plan as it is now: the last one the session announced (plan.updated).
    const plan = sessions.byId[sheet.id]?.entries.findLast((e) => e.kind === 'plan');
    const items = plan?.kind === 'plan' ? plan.items : [];
    node = (
      <SheetView title="Plano" note={planSummary(items)} onClose={close}>
        {items.length > 0 ? (
          <div className="plan sheet-plan">
            {items.map((it) => (
              <div key={it.id} className={`plan-item status-${it.status}`}>
                <span className="plan-glyph">{glyph(it.status)}</span>
                <span className="plan-text">{it.text}</span>
                {it.blocked && <span className="plan-blocked">{it.blocked}</span>}
              </div>
            ))}
          </div>
        ) : (
          <p className="sheet-empty">O agente monta um plano quando a tarefa pede vários passos. /plan com um texto pede que ele refaça o plano levando isso em conta.</p>
        )}
      </SheetView>
    );
  }
  return { show: setSheet, open: sheet !== null, node };
}
