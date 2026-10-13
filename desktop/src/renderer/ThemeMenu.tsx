import { useEffect, useRef, useState } from 'react';
import { THEME_CHOICES, type ThemeChoice } from '../state/theme';

/**
 * The grid's way to the theme (D35): the ⚙ at the foot of the sidebar opens
 * the three choices above it. Closed, it draws exactly the ⚙ it replaced, so
 * the grid looks as it did until someone asks.
 */
export function ThemeMenu({ theme, onTheme }: { theme: ThemeChoice; onTheme: (theme: ThemeChoice) => void }) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open) return;
    const away = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const esc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    window.addEventListener('mousedown', away);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', away);
      window.removeEventListener('keydown', esc);
    };
  }, [open]);

  return (
    <span className="theme-menu" ref={box}>
      <button type="button" className="settings" aria-label="Configurações" aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen(!open)}>
        ⚙
      </button>
      {open && (
        <div className="popover theme-popover" role="menu" aria-label="Tema">
          <div className="popover-head">Tema</div>
          {THEME_CHOICES.map((t) => (
            <div
              key={t.value}
              className={`popover-item${t.value === theme ? ' on' : ''}`}
              role="menuitemradio"
              aria-checked={t.value === theme}
              onClick={() => {
                setOpen(false);
                onTheme(t.value);
              }}
            >
              <span className="popover-label">{t.label}</span>
              <span className="popover-hint">{t.value === theme ? '✓' : t.value === 'escuro' ? 'padrão' : ''}</span>
            </div>
          ))}
        </div>
      )}
    </span>
  );
}
