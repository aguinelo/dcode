// What the field does with what was typed (D40). `!` runs a command in the
// session, as the TUI's `!`; `/undo`, `/compact` and `/mode <m>` act on it; any
// other slash is said to be unknown. A slash never goes to the model as a
// message: it is the person asking the harness, not the model, for something.

import type { CompactResult, UndoResult } from '../protocol/generated';
import { MODES } from './mode';

export type Input =
  | { kind: 'message'; text: string }
  | { kind: 'exec'; command: string }
  | { kind: 'undo' }
  | { kind: 'compact' }
  | { kind: 'mode'; mode: string }
  | { kind: 'unknown'; why: string }
  | { kind: 'bad'; why: string };

const NAMES = MODES.map((m) => m.mode);

export const KNOWN = 'Aqui há /undo, /compact, /mode <plan|assist|auto> e !comando.';

export function parseInput(text: string): Input {
  const t = text.trim();
  if (t.startsWith('!')) {
    const command = t.slice(1).trim();
    return command ? { kind: 'exec', command } : { kind: 'bad', why: 'Escreva o comando depois do !, como em !go test ./...' };
  }
  if (!t.startsWith('/')) return { kind: 'message', text };
  const [name = '', ...args] = t.slice(1).split(/\s+/);
  switch (name) {
    case 'undo':
      return { kind: 'undo' };
    case 'compact':
      return { kind: 'compact' };
    case 'mode': {
      const mode = args[0] ?? '';
      if (!mode) return { kind: 'bad', why: `/mode precisa de um modo: ${NAMES.slice(0, -1).join(', ')} ou ${NAMES[NAMES.length - 1]}.` };
      if (!NAMES.includes(mode)) return { kind: 'bad', why: `Modo desconhecido: ${mode}. Os que existem: ${NAMES.join(', ')}.` };
      return { kind: 'mode', mode };
    }
    default:
      return { kind: 'unknown', why: `Comando desconhecido: /${name}. ${KNOWN}` };
  }
}

/** What undo did, both halves: the file that did not go back is the one most worth knowing about. */
export function undoText(r: UndoResult): string {
  const restored = r.restored ?? [];
  const refused = r.refused ?? [];
  if (restored.length === 0 && refused.length === 0) return 'O último turno não mexeu em arquivo nenhum.';
  const kept = refused.length > 0 ? ` Intocados, mudaram depois do turno: ${refused.join(', ')}.` : '';
  if (restored.length === 0) return `Nada voltou.${kept}`;
  return `Desfeito. Restaurados: ${restored.join(', ')}.${kept}`;
}

/** What a forced compaction did; the daemon emits nothing when there was nothing to compact, so this is where it is said. */
export function compactText(r: CompactResult): string {
  return r.compacted ? 'Contexto resumido.' : 'Nada que valesse resumir ainda: o contexto ficou como estava.';
}
