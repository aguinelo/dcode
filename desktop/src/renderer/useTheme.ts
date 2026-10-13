import { useCallback, useLayoutEffect, useState } from 'react';
import { parseTheme, resolveTheme, THEME_KEY, type ThemeChoice } from '../state/theme';

const DARK = '(prefers-color-scheme: dark)';

function systemDark(): boolean {
  return typeof window.matchMedia === 'function' ? window.matchMedia(DARK).matches : true;
}

function load(): { choice: ThemeChoice; problem: string | null } {
  try {
    const read = parseTheme(window.localStorage.getItem(THEME_KEY));
    return { choice: read.choice, problem: read.problem ? `${read.problem}; a janela fica no escuro.` : null };
  } catch (err) {
    return { choice: parseTheme(null).choice, problem: `Não foi possível ler o tema guardado (${(err as Error).message}); a janela fica no escuro.` };
  }
}

/**
 * The window's palette (D35): the choice, remembered per window, and its
 * result written as `data-theme` on the document root before the frame is
 * painted — the tokens do the rest. Under Sistema it follows the system as
 * it changes. A choice that cannot be read or kept is said, never swallowed.
 */
export function useTheme(say: (text: string) => void): [ThemeChoice, (next: ThemeChoice) => void] {
  const [boot] = useState(load);
  const [choice, setChoice] = useState<ThemeChoice>(boot.choice);
  const [dark, setDark] = useState(systemDark);

  useLayoutEffect(() => {
    if (boot.problem) say(boot.problem);
  }, [boot, say]);

  useLayoutEffect(() => {
    if (choice !== 'sistema' || typeof window.matchMedia !== 'function') return;
    const mq = window.matchMedia(DARK);
    const on = () => setDark(mq.matches);
    on();
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, [choice]);

  useLayoutEffect(() => {
    document.documentElement.dataset.theme = resolveTheme(choice, dark);
  }, [choice, dark]);

  const choose = useCallback(
    (next: ThemeChoice) => {
      setChoice(next);
      try {
        window.localStorage.setItem(THEME_KEY, next);
      } catch (err) {
        say(`Não foi possível guardar o tema (${(err as Error).message}); vale só até fechar.`);
      }
    },
    [say],
  );
  return [choice, choose];
}
