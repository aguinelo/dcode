import path from 'node:path';
import { CodeInvalidInput, type TurnImage } from '../protocol/generated';
import type { ImageAttachment, Refusal } from '../shared/api';

// The pictures a turn carries (D46), checked here — in the main process, at
// the boundary — before any of them reaches the daemon. The renderer never
// names a file: a picked one is read in `attachments.ts`, and a pasted or
// dropped one arrives as bytes that are checked exactly as a read one is.
//
// The media type comes from the bytes, never from the name, as the daemon's
// own reader decides it (`session.ReadImage`): a shot.png that is a JPEG is
// common, and trusting the name would turn into a provider error about a
// mismatch nobody can connect to what they did.

export type Checked<T> = { ok: true; value: T } | { ok: false; refusal: Refusal };

/** What the providers take, and nothing else: the daemon's `supportedImage`. */
export const IMAGE_TYPES: readonly string[] = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

/** The extensions the picker offers. Only an offer: what is accepted is decided by the bytes. */
export const IMAGE_EXTENSIONS: readonly string[] = ['png', 'jpg', 'jpeg', 'gif', 'webp'];

/** The largest picture that goes to the model: the terminal client's `MaxImageBytes`, 10 MiB. */
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024;

/**
 * How many pictures one turn carries. The daemon sets no count; this is the
 * app's own, so a turn the provider would refuse for its size is refused here
 * while the person can still take some out.
 */
export const MAX_IMAGES = 20;

function invalid(message: string): { ok: false; refusal: Refusal } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

export function megabytes(n: number): string {
  return `${(n / (1024 * 1024)).toFixed(1).replace('.', ',')} MB`;
}

/** The type the first bytes say, among the ones accepted; null for anything else. */
export function sniff(bytes: Uint8Array): string | null {
  const at = (i: number, ...want: number[]) => want.every((b, k) => bytes[i + k] === b);
  if (at(0, 0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a)) return 'image/png';
  if (at(0, 0xff, 0xd8, 0xff)) return 'image/jpeg';
  if (at(0, 0x47, 0x49, 0x46, 0x38) && (at(4, 0x37, 0x61) || at(4, 0x39, 0x61))) return 'image/gif';
  if (at(0, 0x52, 0x49, 0x46, 0x46) && at(8, 0x57, 0x45, 0x42, 0x50)) return 'image/webp';
  return null;
}

/** A name as the window shows it: the file's, or what the clipboard called it. */
function nameOf(v: unknown): string {
  if (typeof v !== 'string') return 'imagem';
  const base = path.basename(v.trim());
  return base === '' ? 'imagem' : base.length > 120 ? `${base.slice(0, 120)}…` : base;
}

/** A picture's bytes, checked: in size, and an image the model can read by what the bytes are. */
export function imageFromBytes(name: unknown, bytes: unknown): Checked<ImageAttachment> {
  const shown = nameOf(name);
  if (!(bytes instanceof Uint8Array)) return invalid(`${shown} não chegou como bytes de uma imagem.`);
  if (bytes.length === 0) return invalid(`${shown} está vazia.`);
  if (bytes.length > MAX_IMAGE_BYTES) {
    return invalid(`${shown} tem ${megabytes(bytes.length)}, e uma imagem vai ao modelo com até ${megabytes(MAX_IMAGE_BYTES)}.`);
  }
  const mediaType = sniff(bytes);
  if (!mediaType) return invalid(`${shown} não é uma imagem que o modelo lê: só PNG, JPEG, GIF e WebP.`);
  return { ok: true, value: { name: shown, mediaType, data: Buffer.from(bytes).toString('base64'), bytes: bytes.length } };
}

/** Base64 as the daemon decodes it: the standard alphabet, padded. */
export const BASE64 = /^[A-Za-z0-9+/]*={0,2}$/;

/**
 * The pictures a turn is sent with, as the window holds them: checked again
 * here, since what crosses from the renderer is never taken on its word — the
 * count, each type against the bytes it names, and each size.
 */
export function turnImages(v: unknown): Checked<TurnImage[]> {
  if (v === undefined) return { ok: true, value: [] };
  if (!Array.isArray(v)) return invalid('As imagens do turno precisam vir numa lista.');
  if (v.length > MAX_IMAGES) return invalid(`Vieram ${v.length} imagens, e um turno leva até ${MAX_IMAGES}.`);
  const out: TurnImage[] = [];
  for (const [i, img] of v.entries()) {
    const which = `A imagem ${i + 1}`;
    const rec = img as Partial<ImageAttachment> | null;
    if (typeof rec !== 'object' || rec === null || typeof rec.mediaType !== 'string' || typeof rec.data !== 'string') {
      return invalid(`${which} precisa vir com o tipo e os dados.`);
    }
    if (!IMAGE_TYPES.includes(rec.mediaType)) return invalid(`${which} é ${rec.mediaType}, e o modelo lê só ${IMAGE_TYPES.join(', ')}.`);
    if (rec.data.length % 4 !== 0 || !BASE64.test(rec.data)) return invalid(`${which} não está em base64.`);
    const bytes = Buffer.from(rec.data, 'base64');
    if (bytes.length > MAX_IMAGE_BYTES) return invalid(`${which} tem ${megabytes(bytes.length)}, e o limite é ${megabytes(MAX_IMAGE_BYTES)}.`);
    const real = sniff(bytes);
    if (real !== rec.mediaType) return invalid(`${which} diz ser ${rec.mediaType}, e os bytes são ${real ?? 'de outra coisa'}.`);
    out.push({ media_type: rec.mediaType, data: rec.data });
  }
  return { ok: true, value: out };
}
