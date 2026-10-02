// The daemon's event streams are text/event-stream, read here as the bytes
// arrive. A frame may be split across chunks and one chunk may hold many, so
// nothing assumes where a chunk ends — nor which line ending the other side
// chose.

/** One dispatched frame: its `id:` when it had one, and its data lines joined. */
export interface SseFrame {
  id?: string;
  data: string;
}

const LF = 10;
const CR = 13;

/**
 * Parses text/event-stream incrementally. Lines end in \n, \r\n or \r;
 * comments (`: ping`) and the fields this client has no use for are skipped,
 * and a frame without a data line is not dispatched.
 *
 * Each chunk is scanned once: a line still open at the end of a chunk is kept
 * in pieces and joined when it closes, so one large event arriving in many
 * chunks costs its size, not its size times the chunks.
 */
export class SseParser {
  private open: string[] = [];
  private data: string[] = [];
  private id: string | undefined;
  /** The last chunk ended in \r: a \n opening the next one is the rest of that line end. */
  private afterCR = false;

  push(chunk: string): SseFrame[] {
    const frames: SseFrame[] = [];
    let start = 0;
    if (this.afterCR && chunk.charCodeAt(0) === LF) start = 1;
    this.afterCR = false;
    for (let i = start; i < chunk.length; i++) {
      const c = chunk.charCodeAt(i);
      if (c !== LF && c !== CR) continue;
      const piece = chunk.slice(start, i);
      const line = this.open.length > 0 ? this.open.join('') + piece : piece;
      this.open = [];
      this.line(line, frames);
      if (c === CR) {
        if (i + 1 === chunk.length) this.afterCR = true;
        else if (chunk.charCodeAt(i + 1) === LF) i++;
      }
      start = i + 1;
    }
    if (start < chunk.length) this.open.push(chunk.slice(start));
    return frames;
  }

  private line(line: string, frames: SseFrame[]): void {
    if (line === '') {
      if (this.data.length > 0) {
        const data = this.data.join('\n');
        frames.push(this.id === undefined ? { data } : { id: this.id, data });
      }
      this.data = [];
      this.id = undefined;
      return;
    }
    if (line.startsWith(':')) return;
    const colon = line.indexOf(':');
    const field = colon < 0 ? line : line.slice(0, colon);
    let value = colon < 0 ? '' : line.slice(colon + 1);
    if (value.startsWith(' ')) value = value.slice(1);
    if (field === 'data') this.data.push(value);
    else if (field === 'id') this.id = value;
  }
}

/**
 * A frame's data as the JSON it should be, or as the text it is when it is
 * not. Never dropped: the renderer's decoder is what says a frame is
 * unreadable, and it can only say so about a frame it was given.
 */
export function decodeData(data: string): unknown {
  try {
    return JSON.parse(data) as unknown;
  } catch {
    return data;
  }
}
