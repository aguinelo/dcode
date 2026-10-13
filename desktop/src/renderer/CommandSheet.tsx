import { createContext, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { COMMANDS, planSummary, userEntries } from '../state/commands';
import { HELP_KEYS } from '../state/help';
import type { SessionsState } from '../state/sessions';
import { useUserCommands, type UserCommands } from './useUserCommands';
import type { WindowActions } from './window';

/** What opens over the window from a command (D48): the help, or one conversation's plan. */
export type Sheet = { kind: 'help'; workspace?: string } | { kind: 'plan'; id: string };

/** What a field reaches without every window in between carrying it: the help (`?`), and its project's commands (D50). */
export interface FieldCommands {
  show: (s: Sheet) => void;
  of: (workspace: string) => UserCommands;
}

export const SheetContext = createContext<FieldCommands | null>(null);

function glyph(status: string): string {
  return status === 'done' ? '✓' : status === 'active' ? '●' : status === 'blocked' ? '⊘' : '·';
}

/** The commands a project's people wrote, as the help lists them under the field's own (D50). */
function Mine({ mine }: { mine: UserCommands }) {
  const entries = mine.state === 'ok' ? userEntries(mine.list) : [];
  return (
    <section className="sheet-group">
      <h3 className="sheet-group-title">Deste projeto</h3>
      {mine.state === 'loading' && <p className="sheet-empty">Perguntando ao daemon quais comandos este projeto tem…</p>}
      {mine.state === 'failed' && <p className="sheet-empty tone-err">Os comandos deste projeto não vieram: {mine.why}</p>}
      {mine.state === 'ok' && entries.length === 0 && (
        <p className="sheet-empty">Nenhum. Um arquivo .md em .dcode/commands/ do projeto, ou em commands/ da sua pasta do dcode, vira /nome.</p>
      )}
      {entries.map((c) => (
        <div key={c.name} className="sheet-row">
          <span className="sheet-key">
            /{c.name}
            {c.args && <span className="sheet-args"> {c.args}</span>}
          </span>
          <span className="sheet-what">
            {c.detail} <span className="sheet-args">· {c.origin}</span>
          </span>
        </div>
      ))}
    </section>
  );
}

function Help({ mine }: { mine: UserCommands | null }) {
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
      {mine && <Mine mine={mine} />}
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
export function SheetView({ title, note, onClose, children }: { title: string; note?: string; onClose: () => void; children: ReactNode }) {
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

/** The sheet the window has open, if any, how commands open one, and the commands each project wrote. */
export function useCommandSheet(
  sessions: SessionsState,
  actions: WindowActions,
): { show: (s: Sheet) => void; open: boolean; node: ReactNode; of: (workspace: string) => UserCommands; context: FieldCommands } {
  const [sheet, setSheet] = useState<Sheet | null>(null);
  const of = useUserCommands(actions, sessions);
  const context = useMemo(() => ({ show: setSheet, of }), [of]);
  const close = useCallback(() => setSheet(null), []);
  let node: ReactNode = null;
  if (sheet?.kind === 'help') {
    node = (
      <SheetView title="Ajuda" note="comandos e atalhos" onClose={close}>
        <Help mine={sheet.workspace ? of(sheet.workspace) : null} />
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
  return { show: setSheet, open: sheet !== null, node, of, context };
}
