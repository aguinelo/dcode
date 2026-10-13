import { CodeInvalidInput } from '../protocol/generated';
import type { TerminalPush } from '../shared/api';
import type { Checked } from './inputs';
import { launch, pickShell, typed } from './shell';

// The PTYs the main process holds for the windows (D37). Electron-free, so it
// is tested with a fake PTY: ipc.ts hands it the asking window's id, and
// main.ts gives it node-pty.
//
// Each terminal belongs to the window that opened it — another window can
// neither write to it, resize it, kill it nor hear it — and opens only in a
// folder that exists. The command the agent proposed is typed once the shell
// shows its first prompt and its line editor is reading, and never with Enter: running it is the person's
// keystroke. What passes through is never logged or kept here: a password
// typed at sudo's prompt reaches the PTY and nothing else.

export interface Pty {
  onData(cb: (data: string) => void): void;
  onExit(cb: (e: { exitCode: number; signal?: number }) => void): void;
  write(data: string): void;
  resize(cols: number, rows: number): void;
  kill(): void;
}

export type Spawn = (file: string, args: string[], o: { cwd: string; env: Record<string, string>; cols: number; rows: number }) => Pty;

export type { TerminalPush };

export interface TerminalsOptions {
  spawn: Spawn;
  /** Writes the shell integration files for `kind` and answers their folder. */
  prepare: (kind: 'zsh' | 'bash' | 'other') => Promise<string>;
  env: NodeJS.ProcessEnv;
  platform: string;
  isDir: (dir: string) => Promise<boolean>;
  push: (owner: number, push: TerminalPush) => void;
  /** How long to wait for the first prompt before typing the command anyway. */
  typeAfterMs: number;
  /** How many terminals the app holds at once. */
  max: number;
}

interface Held {
  owner: number;
  pty: Pty;
  cancelTyping: () => void;
}

const PROMPT = '\x1b]133;D';
/**
 * The line editor taking the keyboard: zsh's ZLE and readline 8 switch
 * bracketed paste on when they start reading a line. Typed before that, the
 * terminal is still in cooked mode and echoes the bytes as they came — the
 * paste's own escapes printed above the prompt.
 */
const EDITING = '\x1b[?2004h';
/** After a prompt mark with no line editor announcing itself (bash 3.2), how long to let the prompt settle. */
const SETTLE_MS = 250;

function refuse(message: string): { ok: false; refusal: { code: string; message: string } } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

export class Terminals {
  private readonly held = new Map<string, Held>();
  private next = 1;

  constructor(private readonly o: TerminalsOptions) {}

  /** Opens a shell in `cwd` for window `owner`, with `command` typed on its line once it is ready. */
  async open(owner: number, cwd: string, command: string, cols: number, rows: number): Promise<Checked<{ id: string; cwd: string; shell: string }>> {
    if (this.held.size >= this.o.max) {
      return refuse(`O app já tem ${this.o.max} terminais abertos: feche um antes de abrir outro.`);
    }
    const shell = pickShell(this.o.env, this.o.platform);
    const line = typed(shell.kind, command);
    if (!line.ok) return line;
    if (!(await this.o.isDir(cwd))) return refuse(`A pasta da sessão, ${cwd}, não existe mais: o terminal não abriu.`);
    const dir = await this.o.prepare(shell.kind);
    const how = launch(shell, dir, this.o.env);
    const pty = this.o.spawn(how.file, how.args, { cwd, env: how.env, cols, rows });
    const id = `t${this.next++}`;

    let pending = true;
    let prompted = false;
    const type = (): void => {
      if (!pending) return;
      pending = false;
      clearTimeout(timer);
      pty.write(line.value);
    };
    let timer = setTimeout(type, this.o.typeAfterMs);
    this.held.set(id, { owner, pty, cancelTyping: () => ((pending = false), clearTimeout(timer)) });

    pty.onData((data) => {
      this.o.push(owner, { kind: 'data', id, data });
      if (!pending) return;
      if (data.includes(EDITING) && (prompted || data.includes(PROMPT))) type();
      else if (!prompted && data.includes(PROMPT)) {
        prompted = true;
        clearTimeout(timer);
        timer = setTimeout(type, SETTLE_MS);
      }
    });
    pty.onExit((e) => {
      const h = this.held.get(id);
      h?.cancelTyping();
      this.held.delete(id);
      this.o.push(owner, { kind: 'exit', id, code: e.exitCode });
    });
    return { ok: true, value: { id, cwd, shell: shell.file } };
  }

  write(owner: number, id: string, data: string): Checked<null> {
    const h = this.mine(owner, id);
    if (!h.ok) return h;
    h.value.pty.write(data);
    return { ok: true, value: null };
  }

  resize(owner: number, id: string, cols: number, rows: number): Checked<null> {
    const h = this.mine(owner, id);
    if (!h.ok) return h;
    h.value.pty.resize(cols, rows);
    return { ok: true, value: null };
  }

  kill(owner: number, id: string): Checked<null> {
    const h = this.mine(owner, id);
    if (!h.ok) return h;
    this.end(id, h.value);
    return { ok: true, value: null };
  }

  /** A window went away: what it opened goes with it. */
  closeOwner(owner: number): void {
    for (const [id, h] of this.held) if (h.owner === owner) this.end(id, h);
  }

  closeAll(): void {
    for (const [id, h] of this.held) this.end(id, h);
  }

  private end(id: string, h: Held): void {
    h.cancelTyping();
    this.held.delete(id);
    try {
      h.pty.kill();
    } catch (err) {
      console.warn(`dcode: terminal ${id} did not stop cleanly: ${err instanceof Error ? err.message : String(err)}`);
    }
  }

  private mine(owner: number, id: string): Checked<Held> {
    const h = this.held.get(id);
    // Another window's terminal is refused exactly as one that does not exist: nothing about it is said.
    if (!h || h.owner !== owner) return refuse(`O terminal ${id} não está aberto nesta janela.`);
    return { ok: true, value: h };
  }
}
