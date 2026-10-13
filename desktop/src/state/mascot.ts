// The mascot's state (D49): which pose the window draws for a conversation.
// Pure, and read only from what the window already folds — whether it waits
// for you, the phase of the call running now (D14), and the seal of the last
// measured turn (D13). Nothing here is a field the protocol does not carry.

import type { Phase } from './flow';

export type MascotState = 'planejando' | 'codando' | 'verificando' | 'deu-certo' | 'bug' | 'esperando' | 'hum' | 'parado';

/** What each pose says, as a label and a tooltip; the sheet's words where it has them. */
export const MASCOT_WORDS: Readonly<Record<MascotState, string>> = {
  planejando: 'Planejando…',
  codando: 'Codando…',
  verificando: 'Verificando…',
  'deu-certo': 'Deu certo!',
  bug: 'Bug encontrado…',
  esperando: 'Esperando você',
  hum: 'Hum…',
  parado: 'Parado',
};

/**
 * A conversation that waits for you says so before anything else. One that
 * runs plans while the model thinks or reads, codes while it writes or
 * delegates, and verifies while a command runs — the TUI's `conferindo`.
 * One that stopped shows how its last turn was measured, and only that.
 */
export function mascotOf(state: string, verification: string | undefined, phase: Phase | null): MascotState {
  if (state === 'blocked') return 'esperando';
  if (state === 'running') {
    if (phase === null || phase === 'reading') return 'planejando';
    if (phase === 'running') return 'verificando';
    return 'codando';
  }
  if (verification === 'passed') return 'deu-certo';
  if (verification === 'failed') return 'bug';
  if (verification === 'stale' || verification === 'unavailable') return 'hum';
  return 'parado';
}
