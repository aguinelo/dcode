// The embedded terminal's output, read back for the conversation (D37).
//
// Pure: the PTY's bytes come in as they arrived, and what comes out is the
// command the shell said it ran, how it ended and what it printed — as text,
// cut to a size a message can carry, and never with a password prompt's line.
//
// The shell says where a command starts and ends with the FinalTerm marks the
// main process teaches it (src/main/shell.ts): OSC 133;C[;<base64 command>]
// when a command starts, and OSC 133;D[;<code>;<base64 command>] at each
// prompt. A shell that never marks still works as a terminal; its result is
// sent by hand, without a code.

/** How many of the last lines go to the agent. */
export const MAX_LINES = 200;
/** And how many characters, whatever the lines: a few enormous lines are cut too. */
export const MAX_CHARS = 16_000;
/** How much output the fold keeps of a command still running: the tail is what is sent. */
const KEEP_CHARS = 256_000;
/** A mark longer than this is not one: it is let through as output. */
const MAX_MARK = 4096;

export interface Run {
  /** As the shell ran it — what the person typed, edits and sudo included. */
  command: string;
  /** Its exit code, or null when the shell did not say. */
  code: number | null;
  /** What it printed, raw: escapes and carriage returns still in. */
  output: string;
}

export interface Transcript {
  /** A mark cut in half at the end of a chunk, waiting for the rest. */
  partial: string;
  /** Output since the command started, or since the last prompt when none started. */
  current: string;
  /** A command started (OSC 133;C) and no prompt came back yet. */
  running: { command: string } | null;
  /** The last command that ended at a prompt. */
  last: Run | null;
  /** Prompts marked so far: zero means the shell does not mark, and the result goes by hand. */
  prompts: number;
}

export const emptyTranscript: Transcript = { partial: '', current: '', running: null, last: null, prompts: 0 };

function decode(b64: string | undefined): string {
  if (!b64) return '';
  try {
    const bin = atob(b64);
    return new TextDecoder().decode(Uint8Array.from(bin, (c) => c.charCodeAt(0)));
  } catch {
    return '';
  }
}

function keep(text: string): string {
  return text.length > KEEP_CHARS ? text.slice(text.length - KEEP_CHARS) : text;
}

/** Drops through the first line break: the prompt and the echoed command, when no mark said where output began. */
function afterFirstLine(text: string): string {
  const nl = text.indexOf('\n');
  return nl < 0 ? '' : text.slice(nl + 1);
}

// eslint-disable-next-line no-control-regex
const MARK = /\x1b\]133;([A-D])((?:;[^\x07\x1b]*)?)(?:\x07|\x1b\\)/g;

function mark(t: Transcript, kind: string, params: string[]): Transcript {
  if (kind === 'C') return { ...t, current: '', running: { command: decode(params[0]) } };
  if (kind !== 'D') return t;
  const prompts = t.prompts + 1;
  const [code, cmd] = params;
  const command = decode(cmd) || t.running?.command || '';
  // The first prompt is the shell starting; a bare D is an empty line.
  if (t.prompts === 0 || code === undefined || code === '' || command === '') {
    return { ...t, current: '', running: null, prompts };
  }
  const parsed = Number.parseInt(code, 10);
  const output = t.running ? t.current : afterFirstLine(t.current);
  return { ...t, current: '', running: null, prompts, last: { command, code: Number.isFinite(parsed) ? parsed : null, output } };
}

const OPEN = '\x1b]133;';

/** Where a mark that has not ended begins — or the start of one cut mid-way — else -1. */
function unfinished(text: string): number {
  const open = text.lastIndexOf(OPEN);
  // eslint-disable-next-line no-control-regex
  if (open >= 0 && !/\x07|\x1b\\/.test(text.slice(open)) && text.length - open < MAX_MARK) return open;
  for (let k = OPEN.length - 1; k > 0; k--) if (text.endsWith(OPEN.slice(0, k))) return text.length - k;
  return -1;
}

/** Folds a chunk of the PTY's output in. */
export function feed(t: Transcript, chunk: string): Transcript {
  let text = t.partial + chunk;
  let next: Transcript = { ...t, partial: '' };
  // A mark begun and not ended waits for the next chunk.
  const open = unfinished(text);
  if (open >= 0) {
    next = { ...next, partial: text.slice(open) };
    text = text.slice(0, open);
  }
  let at = 0;
  for (const m of text.matchAll(MARK)) {
    next = { ...next, current: keep(next.current + text.slice(at, m.index)) };
    next = mark(next, m[1] ?? '', (m[2] ?? '').split(';').slice(1));
    at = m.index + m[0].length;
  }
  return { ...next, current: keep(next.current + text.slice(at)) };
}

/**
 * What "Enviar resultado ao agente" sends when pressed: the command running
 * and its output so far; else the last that ended, when nothing came after
 * it; else the proposed command and whatever is on the screen since the prompt.
 */
export function pendingResult(t: Transcript, proposed: string): Run {
  if (t.running) return { command: t.running.command || proposed, code: null, output: t.current };
  if (t.last && afterFirstLine(t.current).trim() === '') return t.last;
  return { command: proposed, code: null, output: t.prompts > 0 ? afterFirstLine(t.current) : t.current };
}

// eslint-disable-next-line no-control-regex
const ESCAPES = /\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-?]*[ -/]*[@-~]|\x1b[()*+][0-9A-Za-z]|\x1b[@-Z\\-_=>78]/g;
// eslint-disable-next-line no-control-regex
const CONTROLS = /[\x00-\x07\x0b\x0c\x0e-\x1f\x7f]/g;

/** The text without colours, cursor moves, titles or other escapes. Backspace, tab, CR and LF stay. */
export function stripAnsi(text: string): string {
  return text.replace(ESCAPES, '').replace(CONTROLS, '');
}

/** A password prompt's line: sudo's, su's, ssh's, in English or Portuguese. Dropped whole. */
const PASSWORD = /(\[sudo\] (password|senha) |^\s*(password|senha)( for [^:]*)?:|'s password:)/i;

function apply(line: string): string {
  // What a carriage return left: the last overwrite wins, as on screen.
  const parts = line.split('\r');
  let shown = '';
  for (const p of parts) shown = p + shown.slice(p.length);
  let out = '';
  for (const c of shown) out = c === '\b' ? out.slice(0, -1) : out + c;
  return out.replace(CONTROLS, '');
}

/** The output as a terminal shows it, without the lines a password prompt printed. */
export function cleanOutput(raw: string): string {
  const lines = stripAnsi(raw)
    .replace(/\r+\n/g, '\n')
    .split('\n')
    .map(apply)
    .filter((l) => !PASSWORD.test(l));
  while (lines.length > 0 && lines[lines.length - 1]?.trim() === '') lines.pop();
  return lines.join('\n');
}

/** The last MAX_LINES lines, and no more than MAX_CHARS; how many lines there were. */
export function tailLines(text: string): { text: string; truncated: boolean; total: number } {
  const lines = text.split('\n');
  const total = lines.length;
  let kept = lines.slice(-MAX_LINES).join('\n');
  let truncated = total > MAX_LINES;
  if (kept.length > MAX_CHARS) {
    kept = kept.slice(kept.length - MAX_CHARS);
    truncated = true;
  }
  return { text: kept, truncated, total };
}

/** A fence longer than any run of backticks in the text. */
function fence(text: string): string {
  const longest = Math.max(2, ...Array.from(text.matchAll(/`+/g), (m) => m[0].length));
  return '`'.repeat(longest + 1);
}

function inline(command: string): string {
  return command.includes('`') ? `\`\` ${command} \`\`` : `\`${command}\``;
}

/** The message that tells the agent what the person ran in the terminal and what came of it. */
export function resultMessage(run: Run): string {
  const out = cleanOutput(run.output);
  const how = run.code === null ? 'ainda rodando, ou sem código de saída' : `código ${run.code}`;
  const head = run.command.includes('\n')
    ? `Rodei no terminal:\n${fence(run.command)}sh\n${run.command}\n${fence(run.command)}\n`
    : `Rodei no terminal: ${inline(run.command)} — `;
  if (out.trim() === '') return `${head}sem saída (${how}).`;
  const t = tailLines(out);
  const cut = t.truncated ? `(mostrando as últimas ${t.text.split('\n').length} de ${t.total} linhas)\n` : '';
  const f = fence(t.text);
  return `${head}saída (${how}):\n${cut}${f}\n${t.text}\n${f}`;
}
