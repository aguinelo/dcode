import { readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { CodeInvalidInput, MaxDocumentsPerTurn, MaxPDFBytes, type TurnDocument, type TurnImage } from '../protocol/generated';
import { isImage, type Attachment, type PickedAttachments, type Refusal } from '../shared/api';
import { documentFromBytes, turnDocuments } from './documents';
import { imageFromBytes, MAX_IMAGE_BYTES, MAX_IMAGES, megabytes, sniff, turnImages, type Checked } from './images';

// What a turn carries besides its text, whichever kind it is: a picture (D46)
// or a document (D59). The kind is the bytes', never the name's — a file
// called notes.png that is a PDF goes as a PDF — and each kind is then
// checked by its own rules.

/** The largest file read at all: the larger of the two kinds' limits, so neither is read past it. */
const MAX_FILE_BYTES = Math.max(MAX_IMAGE_BYTES, MaxPDFBytes);

function invalid(message: string): { ok: false; refusal: Refusal } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

/** A file's bytes, as a picture when they are one and as a document otherwise. */
export function attachmentFromBytes(name: unknown, bytes: unknown): Checked<Attachment> {
  if (bytes instanceof Uint8Array && sniff(bytes)) return imageFromBytes(name, bytes);
  return documentFromBytes(name, bytes);
}

/** A file the person picked, read here: its size known before it is read, so a huge one is never loaded. */
export async function readAttachmentFile(file: string): Promise<Checked<Attachment>> {
  const shown = path.basename(file);
  try {
    const info = await stat(file);
    if (!info.isFile()) return invalid(`${shown} não é um arquivo.`);
    if (info.size > MAX_FILE_BYTES) {
      return invalid(`${shown} tem ${megabytes(info.size)}, e um anexo vai ao modelo com até ${megabytes(MAX_FILE_BYTES)}.`);
    }
    return attachmentFromBytes(file, new Uint8Array(await readFile(file)));
  } catch (err) {
    return invalid(`${shown} não foi lido: ${err instanceof Error ? err.message : String(err)}`);
  }
}

/**
 * The files a picker gave, each read and checked: one refused is said by its
 * reason, and the others still attach. More of a kind than a turn carries is
 * refused by name rather than cut silently.
 */
export async function readAttachmentFiles(files: readonly string[]): Promise<PickedAttachments> {
  const out: PickedAttachments = { attachments: [], refused: [] };
  let images = 0;
  let documents = 0;
  for (const file of files) {
    const read = await readAttachmentFile(file);
    if (!read.ok) {
      out.refused.push(read.refusal.message);
      continue;
    }
    if (isImage(read.value)) {
      if (images >= MAX_IMAGES) {
        out.refused.push(`${read.value.name} ficou de fora: um turno leva até ${MAX_IMAGES} imagens.`);
        continue;
      }
      images++;
    } else {
      if (documents >= MaxDocumentsPerTurn) {
        out.refused.push(`${read.value.name} ficou de fora: um turno leva até ${MaxDocumentsPerTurn} documentos.`);
        continue;
      }
      documents++;
    }
    out.attachments.push(read.value);
  }
  return out;
}

/** What a turn is sent with, split by kind for the wire and each kind checked again by its own rules. */
export function turnAttachments(v: unknown): Checked<{ images: TurnImage[]; documents: TurnDocument[] }> {
  if (v === undefined) return { ok: true, value: { images: [], documents: [] } };
  if (!Array.isArray(v)) return invalid('Os anexos do turno precisam vir numa lista.');
  const pictures = v.filter((a) => typeof a === 'object' && a !== null && typeof a.mediaType === 'string' && isImage(a as Attachment));
  const rest = v.filter((a) => !pictures.includes(a));
  const images = turnImages(pictures);
  if (!images.ok) return images;
  const documents = turnDocuments(rest);
  if (!documents.ok) return documents;
  return { ok: true, value: { images: images.value, documents: documents.value } };
}
