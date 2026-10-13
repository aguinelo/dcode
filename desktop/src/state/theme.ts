// Which palette the window wears (D35): the person's choice, kept per window
// like the version of the screen, and what it comes to under the system's
// own preference. Dark stays the default, so the rulers keep measuring the
// window they were made against.

/** What the person picked: follow the system, or one palette always. */
export type ThemeChoice = 'sistema' | 'claro' | 'escuro';

/** What the window wears: the value of `data-theme` on the document root. */
export type Theme = 'light' | 'dark';

export const THEME_KEY = 'dcode.desktop.theme.v1';

export const DEFAULT_THEME: ThemeChoice = 'escuro';

export const THEME_CHOICES: readonly { value: ThemeChoice; label: string }[] = [
  { value: 'sistema', label: 'Sistema' },
  { value: 'claro', label: 'Claro' },
  { value: 'escuro', label: 'Escuro' },
];

function isChoice(v: string): v is ThemeChoice {
  return THEME_CHOICES.some((c) => c.value === v);
}

/**
 * Reads what was stored. Nothing stored is the ordinary first run; a value
 * that is not one of the three is a problem, returned so it can be said —
 * the default is used either way, but never in silence.
 */
export function parseTheme(raw: string | null): { choice: ThemeChoice; problem: string | null } {
  if (raw === null) return { choice: DEFAULT_THEME, problem: null };
  if (isChoice(raw)) return { choice: raw, problem: null };
  return { choice: DEFAULT_THEME, problem: `tema guardado ilegível ("${raw.slice(0, 40)}")` };
}

/** What a choice comes to, given whether the system asks for dark. */
export function resolveTheme(choice: ThemeChoice, systemDark: boolean): Theme {
  if (choice === 'claro') return 'light';
  if (choice === 'escuro') return 'dark';
  return systemDark ? 'dark' : 'light';
}
