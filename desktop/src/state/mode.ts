// A session's behavioural mode as the window names and switches it (D36).
// The daemon maps each name to a pair of sandbox and approval policy; a pair
// that is none of the three has no name on the wire (`mode` empty), and is
// called custom here rather than the nearest mode — the TUI's reading.

import { ModeAssist, ModeAuto, ModePlan } from '../protocol/generated';

export interface ModeOption {
  mode: string;
  label: string;
  /** What choosing it does, in a line. */
  about: string;
}

/** The three modes, in the order ⇧⇥ cycles them. */
export const MODES: readonly ModeOption[] = [
  { mode: ModePlan, label: 'Plano', about: 'só leitura; não escreve no projeto' },
  { mode: ModeAssist, label: 'Assistir', about: 'escreve no projeto; pede antes de sair dele' },
  { mode: ModeAuto, label: 'Auto', about: 'acesso total à máquina' },
];

/** The mode ⇧⇥ goes to: plan → assist → auto → plan; a pair with no name enters at plan. */
export function nextMode(current: string): string {
  const at = MODES.findIndex((m) => m.mode === current);
  return MODES[(at + 1) % MODES.length]?.mode ?? ModePlan;
}

export interface ModeChip {
  text: string;
  /** auto is the one fact worth the warning colour: no boundary between the agent and the machine. */
  tone: 'quiet' | 'warn';
  /** Whether the mode is one of the three. */
  named: boolean;
  title: string;
}

const CUSTOM = 'personalizado';

/** The mode's chip: the sandbox and the mode, as the composer has always drawn them; null before either is known. */
export function modeChip(mode: string, sandbox: string): ModeChip | null {
  if (!mode && !sandbox) return null;
  const named = MODES.find((m) => m.mode === mode);
  if (!named) {
    return {
      text: [sandbox, CUSTOM].filter(Boolean).join(' · '),
      tone: 'quiet',
      named: false,
      title: `${CUSTOM} (sandbox · política): a sessão roda em ${sandbox || 'um sandbox não dito'} com uma política de aprovação que não é a de nenhum dos três modos. Trocar de modo`,
    };
  }
  return {
    text: [sandbox, mode].filter(Boolean).join(' · '),
    tone: mode === ModeAuto ? 'warn' : 'quiet',
    named: true,
    title: `${named.label}: ${named.about}. Trocar de modo (⇧⇥)`,
  };
}

/** The sandbox each mode maps to, as the daemon maps it (`modeToSandbox`): what a new session opens in. */
const SANDBOX: Readonly<Record<string, string>> = { plan: 'read-only', assist: 'workspace-write', auto: 'full-access' };

export function sandboxOf(mode: string): string | undefined {
  return SANDBOX[mode];
}

/**
 * The mode a session just opened still has to be switched to: the sandbox
 * gives the boundary, and the approval policy comes from the configuration,
 * so a session can open on the right sandbox and still be another mode — or
 * none of the three. Null when it opened in the one chosen, or none was.
 */
export function modeToSwitch(opened: string, chosen: string | undefined): string | null {
  return chosen && opened !== chosen ? chosen : null;
}
