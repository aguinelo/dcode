// What a model's tool call opens onto (D45): its output and, when it changed a
// file, its diff. The line says what the call did in a few words; opening it
// shows what it did in full, the way the TUI's Tab does. Pure.

import { diffLines, type DiffLine } from './git';
import type { ToolCall } from './session';

/** How many lines of a long output show before "mostrar tudo" is asked for. */
export const OUTPUT_PREVIEW_LINES = 40;

const ESC = String.fromCharCode(27);
const BEL = String.fromCharCode(7);
// CSI (colours, cursor moves), OSC (titles, links) and the two-byte escapes.
const ANSI = new RegExp(`${ESC}\\[[0-?]*[ -/]*[@-~]|${ESC}\\][^${BEL}${ESC}]*(?:${BEL}|${ESC}\\\\)|${ESC}[@-Z\\\\-_]`, 'g');

/**
 * The text a terminal would have shown: escape sequences taken out, and of a
 * line a carriage return rewrote (a progress bar), only what was left on it.
 */
export function stripAnsi(text: string): string {
  return text
    .replace(ANSI, '')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((l) => {
      const at = l.lastIndexOf('\r');
      return at < 0 ? l : l.slice(at + 1);
    })
    .join('\n');
}

export interface OutputDetail {
  /** All of it, as the tool returned it, without escape codes. */
  text: string;
  /** What shows before it is asked for: all of it, unless it is long. */
  preview: string;
  totalLines: number;
  /** More lines than the preview holds. A failure is never long: it is shown whole. */
  long: boolean;
  /** The tool cut what it returned before it reached the daemon. */
  cutByTool: boolean;
}

export interface DiffDetail {
  lines: DiffLine[];
  /** The daemon cut the diff at this many lines (tools.DiffMaxLines); null when whole. */
  cutAt: number | null;
}

export interface ToolDetail {
  failed: boolean;
  output: OutputDetail | null;
  diff: DiffDetail | null;
}

// internal/tools/diff.go writes this as the diff's last line when it cuts.
const DIFF_CUT = /^⋯ diff truncated at (\d+) lines$/;

function diffOf(text: string): DiffDetail | null {
  if (!text.trim()) return null;
  const all = diffLines(text);
  const last = all[all.length - 1];
  const m = last ? DIFF_CUT.exec(last.text) : null;
  if (m) return { lines: all.slice(0, -1), cutAt: Number(m[1]) };
  return { lines: all, cutAt: null };
}

function outputOf(raw: string, truncated: boolean, failed: boolean): OutputDetail | null {
  const text = stripAnsi(raw).replace(/\n+$/, '');
  if (!text.trim()) return null;
  const all = text.split('\n');
  const long = !failed && all.length > OUTPUT_PREVIEW_LINES;
  return {
    text,
    preview: long ? all.slice(0, OUTPUT_PREVIEW_LINES).join('\n') : text,
    totalLines: all.length,
    long,
    cutByTool: truncated,
  };
}

/**
 * What opening a call's line shows; null when there is nothing to open — a
 * call still running, one that returned nothing, or a command the person typed,
 * which is shown open already (D40).
 *
 * A change shows its diff and not its output: the diff is what gets reviewed,
 * and the output of an edit is the model's echo of it. A failure shows its
 * error whole, beside any diff, because the error is the point.
 */
export function toolDetail(call: ToolCall): ToolDetail | null {
  const r = call.result;
  if (!r || call.typed) return null;
  const failed = !r.ok;
  const diff = diffOf(r.diff ?? '');
  const output = diff && !failed ? null : outputOf(r.output, r.truncated, failed);
  if (!diff && !output) return null;
  return { failed, output, diff };
}
