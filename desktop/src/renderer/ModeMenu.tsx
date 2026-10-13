import { useRef } from 'react';
import { MODES, modeChip } from '../state/mode';
import { placeBy, usePopover } from './popover';

const WIDTH = 320;

/**
 * The session's mode, as the composer has always drawn it — sandbox · mode —
 * and the button that opens the menu to switch it (D36). auto is drawn in the
 * warning tone, as the TUI's bar does: no boundary between the agent and the
 * machine is the one fact on the bar worth the colour.
 */
export function ModeChip({
  mode,
  sandbox,
  className = 'composer-chip',
  onOpen,
}: {
  mode: string;
  sandbox: string;
  className?: string;
  onOpen?: (anchor: HTMLElement) => void;
}) {
  const chip = modeChip(mode, sandbox);
  if (!chip) return null;
  const tone = chip.tone === 'warn' ? ' tone-warn mode-auto' : '';
  if (!onOpen) return <span className={`${className}${tone}`}>{chip.text}</span>;
  return (
    <button
      type="button"
      className={`${className} mode-chip${tone}`}
      title={chip.title}
      aria-haspopup="menu"
      data-mode={mode || 'custom'}
      onClick={(e) => onOpen(e.currentTarget)}
    >
      {chip.text}
    </button>
  );
}

/**
 * Switches the session between plan, assist and auto (D36). Nothing changes
 * here when a mode is picked: the chip changes when the session announces the
 * switch, and a refusal is said with the daemon's reason.
 */
export function ModeMenu({
  anchor,
  mode,
  sandbox,
  onPick,
  onClose,
}: {
  anchor: HTMLElement;
  /** The mode the session announced last; empty for a pair that is none of the three. */
  mode: string;
  sandbox: string;
  onPick: (mode: string) => void;
  onClose: () => void;
}) {
  const box = useRef<HTMLDivElement>(null);
  usePopover(box, anchor, onClose, '.model-option', mode);
  const chip = modeChip(mode, sandbox);
  return (
    <div ref={box} className="model-menu" style={{ width: WIDTH, ...placeBy(anchor, WIDTH) }} role="menu" aria-label="Modo da sessão">
      <div className="model-menu-head">Modo da sessão</div>
      {chip && !chip.named && <div className="model-menu-note">Agora: personalizado (sandbox · política) — {sandbox || 'sandbox não dito'}.</div>}
      {MODES.map((m) => {
        const current = m.mode === mode;
        return (
          <button
            key={m.mode}
            type="button"
            role="menuitem"
            className={`model-option${current ? ' current' : ''}`}
            disabled={current}
            data-mode={m.mode}
            onClick={() => onPick(m.mode)}
          >
            <span className={`model-name${m.mode === 'auto' ? ' tone-warn' : ''}`}>{m.label}</span>
            <span className="model-detail">{m.about}</span>
            {current ? <span className="model-badge">atual</span> : m.mode === 'auto' ? <span className="model-badge warn">sem fronteira</span> : null}
          </button>
        );
      })}
      <div className="model-menu-foot">⇧⇥ passa ao próximo. Um turno rodando continua, já no modo novo.</div>
    </div>
  );
}
