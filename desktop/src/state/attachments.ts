import type { ImageAttachment } from '../shared/api';
import type { Entry } from './session';

// The pictures of each conversation's field (D46): what is attached to the
// message being written, why the last one was refused, and what went with
// each message the daemon took. The events carry no pictures — `turn.started`
// has the text only — so what the flow shows next to a message is what this
// window sent with it, and a conversation read back from its record shows the
// text alone.

/** How many pictures a turn carries: the main process's `MAX_IMAGES`, said here so the field refuses before it asks. */
export const MAX_IMAGES = 20;

/** The largest picture, as the main process checks it: read past this, a file is never loaded into the window. */
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024;

/** Why a message with pictures does not go to a running turn: `SteerRequest` carries text only. */
export const STEER_TAKES_TEXT =
  'O turno está rodando, e uma correção leva só texto: as imagens ficam anexadas e vão com a próxima mensagem, quando o turno terminar.';

/** Why a message with pictures does not wait in the queue (D47): the queue keeps text, and pictures go with a turn. */
export const QUEUE_TAKES_TEXT =
  'A fila guarda só texto: uma mensagem com imagens não entra nela. As imagens ficam anexadas; envie com ↵ quando o turno terminar.';

/** A message the daemon took with pictures, and the seq the session was at when it was sent. */
export interface Sent {
  after: number;
  images: ImageAttachment[];
}

export interface Attached {
  /** Per session: what goes with the next message. */
  draft: Readonly<Record<string, ImageAttachment[]>>;
  /** Per session: the last refusal, said in the field until the next attach or send. */
  refusal: Readonly<Record<string, string>>;
  /** Per session: the messages sent with pictures, in order. */
  sent: Readonly<Record<string, Sent[]>>;
}

export const noAttachments: Attached = { draft: {}, refusal: {}, sent: {} };

function withRefusal(a: Attached, id: string, why: string | null): Attached {
  const refusal = { ...a.refusal };
  if (why) refusal[id] = why;
  else delete refusal[id];
  return { ...a, refusal };
}

/**
 * Attaches pictures to a session's next message, up to what a turn carries.
 * The ones past the limit are refused by name, never dropped in silence; the
 * refusals given (by the main process's checks) are said with them.
 */
export function attach(a: Attached, id: string, images: readonly ImageAttachment[], refused: readonly string[] = []): Attached {
  const had = a.draft[id] ?? [];
  const room = Math.max(0, MAX_IMAGES - had.length);
  const kept = images.slice(0, room);
  const over = images.slice(room).map((img) => `${img.name} ficou de fora: um turno leva até ${MAX_IMAGES} imagens.`);
  const why = [...refused, ...over];
  const next = kept.length > 0 ? { ...a, draft: { ...a.draft, [id]: [...had, ...kept] } } : a;
  return withRefusal(next, id, why.length > 0 ? why.join(' ') : kept.length > 0 ? null : (a.refusal[id] ?? null));
}

/** A refusal for this session's field, from a paste or a drop the main process refused. */
export function refuse(a: Attached, id: string, why: string): Attached {
  return withRefusal(a, id, why);
}

/** Takes one picture out of the next message. */
export function detach(a: Attached, id: string, index: number): Attached {
  const had = a.draft[id] ?? [];
  if (index < 0 || index >= had.length) return a;
  return withRefusal({ ...a, draft: { ...a.draft, [id]: had.filter((_, i) => i !== index) } }, id, null);
}

/** An ended conversation continued in a new session: what was attached in its field goes with the text. */
export function moveDraft(a: Attached, from: string, to: string): Attached {
  const moving = a.draft[from] ?? [];
  if (moving.length === 0 || from === to) return a;
  const draft = { ...a.draft, [to]: [...(a.draft[to] ?? []), ...moving] };
  delete draft[from];
  return { ...a, draft };
}

/**
 * The daemon took a message with these pictures: they leave the field and are
 * kept against the seq the session was at, so the flow can show them on the
 * message that starts after it. Only the ones sent leave: one attached while
 * the message was on its way stays for the next.
 */
export function markSent(a: Attached, id: string, after: number, images: readonly ImageAttachment[]): Attached {
  if (images.length === 0) return a;
  const left = (a.draft[id] ?? []).filter((img) => !images.includes(img));
  const sent = [...(a.sent[id] ?? []), { after, images: [...images] }];
  return withRefusal({ ...a, draft: { ...a.draft, [id]: left }, sent: { ...a.sent, [id]: sent } }, id, null);
}

/**
 * Which message each sent set of pictures goes on: the first message the
 * person sent after the seq the session was at — a turn, never a correction,
 * since a correction carries text only. Keyed by the message's seq.
 */
export function imagesOfEntries(entries: readonly Entry[], sent: readonly Sent[] | undefined): ReadonlyMap<number, ImageAttachment[]> {
  const out = new Map<number, ImageAttachment[]>();
  if (!sent || sent.length === 0) return out;
  for (const s of sent) {
    const at = entries.find((e) => e.kind === 'user' && !e.steer && e.seq > s.after && !out.has(e.seq));
    if (at) out.set(at.seq, s.images);
  }
  return out;
}

/** How a picture's size reads under its thumbnail. */
export function sizeOf(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1).replace('.', ',')} MB`;
}

/** What a picture is drawn from: a data URL, which the renderer's CSP allows (img-src 'self' data:). */
export function imageSource(img: ImageAttachment): string {
  return `data:${img.mediaType};base64,${img.data}`;
}
