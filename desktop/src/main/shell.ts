import path from 'node:path';
import { CodeInvalidInput } from '../protocol/generated';
import type { Checked } from './inputs';

// The shell the embedded terminal runs (D37): the person's own, started as a
// login shell so it has the PATH a terminal of theirs would have — a GUI app
// on macOS inherits almost none. zsh and bash are taught to mark their
// prompts with the FinalTerm sequences (OSC 133), after the person's own rc
// files have run, so the window knows when a command ended and with what code.
// Any other shell runs as it is, unmarked: its result is sent by hand.

export type ShellKind = 'zsh' | 'bash' | 'other';

export interface Shell {
  file: string;
  kind: ShellKind;
}

export function pickShell(env: NodeJS.ProcessEnv, platform: string): Shell {
  const given = env.SHELL;
  const file = given && path.isAbsolute(given) ? given : platform === 'darwin' ? '/bin/zsh' : '/bin/bash';
  const name = path.basename(file);
  return { file, kind: name === 'zsh' ? 'zsh' : name === 'bash' ? 'bash' : 'other' };
}

/** What the app's process carries that a shell of the person's must not. */
const DROPPED = ['ELECTRON_RUN_AS_NODE', 'ELECTRON_NO_ATTACH_CONSOLE', 'NODE_OPTIONS', 'DCODE_DESKTOP_USER_DATA'];

export function launch(shell: Shell, dir: string, env: NodeJS.ProcessEnv): { file: string; args: string[]; env: Record<string, string> } {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(env)) if (v !== undefined && !DROPPED.includes(k)) out[k] = v;
  out.TERM = 'xterm-256color';
  out.COLORTERM = 'truecolor';
  out.TERM_PROGRAM = 'dcode';
  switch (shell.kind) {
    case 'zsh':
      out.DCODE_USER_ZDOTDIR = env.ZDOTDIR || env.HOME || '';
      out.ZDOTDIR = path.join(dir, 'zsh');
      return { file: shell.file, args: ['-l'], env: out };
    case 'bash':
      return { file: shell.file, args: ['--rcfile', path.join(dir, 'bash', 'rc'), '-i'], env: out };
    case 'other':
      return { file: shell.file, args: ['-l'], env: out };
  }
}

// Each zsh file reads the person's own under their ZDOTDIR, then points zsh
// back at the app's for the next one. After .zshrc, ZDOTDIR is theirs again,
// so .zlogin and everything after reads their files directly.
const SWAP_IN = `_dcode_zd="$ZDOTDIR"; ZDOTDIR="\${DCODE_USER_ZDOTDIR:-$HOME}"`;
const SWAP_OUT = `DCODE_USER_ZDOTDIR="$ZDOTDIR"; ZDOTDIR="$_dcode_zd"`;

function zshPassThrough(file: string): string {
  return `# dcode: reads your ${file}, then the app's next one (D37).\n${SWAP_IN}\n[[ -f "$ZDOTDIR/${file}" ]] && source "$ZDOTDIR/${file}"\n${SWAP_OUT}\n`;
}

const ZSHRC = `# dcode: reads your .zshrc, then marks prompts for the app's terminal (D37).
${SWAP_IN}
[[ -f "$ZDOTDIR/.zshrc" ]] && source "$ZDOTDIR/.zshrc"
unset _dcode_zd DCODE_USER_ZDOTDIR
_dcode_cmd=""
_dcode_b64() { print -rn -- "$1" | base64 | tr -d '\\n'; }
_dcode_preexec() { _dcode_cmd="$1"; printf '\\033]133;C;%s\\007' "$(_dcode_b64 "$1")"; }
_dcode_precmd() {
  local c=$?
  if [[ -n "$_dcode_cmd" ]]; then printf '\\033]133;D;%s;%s\\007' "$c" "$(_dcode_b64 "$_dcode_cmd")"; else printf '\\033]133;D\\007'; fi
  _dcode_cmd=""
}
precmd_functions=(_dcode_precmd $precmd_functions)
preexec_functions+=(_dcode_preexec)
`;

// bash has no preexec: the command comes from the history at the prompt, and
// one equal to the last is an empty line. PS0 marks the start on bash 4.4+
// and is ignored before it (macOS ships 3.2).
const BASHRC = `# dcode: reads your profile as a login shell would, then marks prompts for the app's terminal (D37).
[ -f /etc/profile ] && . /etc/profile
if [ -f ~/.bash_profile ]; then . ~/.bash_profile
elif [ -f ~/.bash_login ]; then . ~/.bash_login
elif [ -f ~/.profile ]; then . ~/.profile
elif [ -f ~/.bashrc ]; then . ~/.bashrc
fi
_dcode_last="$(HISTTIMEFORMAT= builtin history 1)"
_dcode_pc() {
  local c=$? n
  n="$(HISTTIMEFORMAT= builtin history 1)"
  if [ -n "$n" ] && [ "$n" != "$_dcode_last" ]; then
    _dcode_last="$n"
    n="$(printf '%s' "$n" | sed -e '1s/^ *[0-9]*[* ] *//')"
    printf '\\033]133;D;%s;%s\\007' "$c" "$(printf '%s' "$n" | base64 | tr -d '\\n')"
  else
    printf '\\033]133;D\\007'
  fi
}
PROMPT_COMMAND="_dcode_pc\${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
PS0=$'\\033]133;C\\007'
`;

/** The files that teach a shell to mark its prompts, relative to the app's shell folder. */
export function integration(kind: ShellKind): Array<{ name: string; content: string }> {
  switch (kind) {
    case 'zsh':
      return [
        { name: 'zsh/.zshenv', content: zshPassThrough('.zshenv') },
        { name: 'zsh/.zprofile', content: zshPassThrough('.zprofile') },
        { name: 'zsh/.zshrc', content: ZSHRC },
        // Never read while things work — ZDOTDIR is the person's by then — and harmless if it is.
        { name: 'zsh/.zlogin', content: zshPassThrough('.zlogin') },
      ];
    case 'bash':
      return [{ name: 'bash/rc', content: BASHRC }];
    case 'other':
      return [];
  }
}

const PASTE_START = '\x1b[200~';
const PASTE_END = '\x1b[201~';

function refuse(message: string): { ok: false; refusal: { code: string; message: string } } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

/**
 * The bytes that put a proposed command on the shell's line without running
 * it. zsh takes it as a bracketed paste, so a command of several lines waits
 * for Enter too; another shell takes one line only, since a line break there
 * would run it. Nothing that would run it, interrupt or end the paste is ever
 * typed: carriage return, escape, and the other control characters but tab.
 */
export function typed(kind: ShellKind, command: string): Checked<string> {
  if (command.trim() === '') return refuse('O comando está vazio: não há o que digitar no terminal.');
  // eslint-disable-next-line no-control-regex
  if (/[\x00-\x08\x0b-\x1f\x7f]/.test(command)) {
    return refuse('O comando tem caracteres de controle, que rodariam ou mudariam o que foi proposto: copie e cole à mão.');
  }
  if (kind === 'zsh') return { ok: true, value: `${PASTE_START}${command}${PASTE_END}` };
  if (command.includes('\n')) {
    return refuse('O comando tem várias linhas, e neste shell uma quebra de linha o rodaria: copie e cole à mão.');
  }
  return { ok: true, value: command };
}
