import path from 'node:path';
import {
  CodeInvalidInput,
  DocumentCSV,
  DocumentMarkdown,
  DocumentPDF,
  DocumentText,
  MaxDocumentNameBytes,
  MaxDocumentsPerTurn,
  MaxPDFBytes,
  MaxTextDocumentBytes,
  type TurnDocument,
} from '../protocol/generated';
import type { DocumentAttachment, Refusal } from '../shared/api';
import { BASE64, megabytes, type Checked } from './images';

// The documents a turn carries (D59), checked here — in the main process, at
// the boundary — by the rules the daemon holds them to (`decodeDocuments`), so
// a file is refused while the person can still take it out rather than when
// the turn is sent. The kinds, the sizes and the count are the protocol's
// constants: one number in two places is two numbers.
//
// A PDF is known by its first bytes, never by its name. Text has nothing to
// sniff, so it has to be UTF-8 without a NUL — what a binary renamed .txt is
// not — and only then does the extension say which text it is.

/** The extensions the picker offers besides pictures. Only an offer: what is accepted is decided by the bytes. */
export const DOCUMENT_EXTENSIONS: readonly string[] = [
  'pdf', 'txt', 'md', 'markdown', 'csv', 'json', 'log', 'yaml', 'yml', 'toml', 'xml', 'html', 'css',
  'go', 'ts', 'tsx', 'js', 'jsx', 'py', 'rs', 'java', 'c', 'h', 'sh', 'sql',
];

const TEXT_TYPES: readonly string[] = [DocumentText, DocumentMarkdown, DocumentCSV];

function invalid(message: string): { ok: false; refusal: Refusal } {
  return { ok: false, refusal: { code: CodeInvalidInput, message } };
}

function kilobytes(n: number): string {
  return `${Math.round(n / 1024)} KB`;
}

/** Whether the bytes are a PDF's: what `%PDF-` at the start says, as the daemon checks it. */
export function isPDF(bytes: Uint8Array): boolean {
  return bytes.length >= 5 && Buffer.from(bytes.subarray(0, 5)).toString('latin1') === '%PDF-';
}

const utf8 = new TextDecoder('utf-8', { fatal: true });

/** Whether the bytes are text the daemon takes: valid UTF-8, with no NUL in it. */
export function isText(bytes: Uint8Array): boolean {
  if (bytes.includes(0)) return false;
  try {
    utf8.decode(bytes);
    return true;
  } catch {
    return false;
  }
}

/** Which text a file is, by its extension — asked only once the bytes said it is text. */
function textTypeOf(name: string): string {
  const ext = path.extname(name).toLowerCase();
  if (ext === '.md' || ext === '.markdown') return DocumentMarkdown;
  if (ext === '.csv') return DocumentCSV;
  return DocumentText;
}

/** A name the daemon takes: a file's, without a path, a control character, or more bytes than it allows. */
function goodName(name: string): boolean {
  // eslint-disable-next-line no-control-regex
  return name !== '' && Buffer.byteLength(name) <= MaxDocumentNameBytes && !/[/\\\u0000-\u001f\u007f-\u009f]/.test(name);
}

/** The name as the daemon will see it: the file's, cut from any path that came with it. */
function nameOf(v: unknown): string {
  if (typeof v !== 'string') return 'documento';
  const base = path.basename(v.trim());
  return base === '' ? 'documento' : base;
}

/** A document's bytes, checked: a PDF or text, within its size, under a name the daemon takes. */
export function documentFromBytes(name: unknown, bytes: unknown): Checked<DocumentAttachment> {
  const shown = nameOf(name);
  if (!(bytes instanceof Uint8Array)) return invalid(`${shown} não chegou como bytes de um arquivo.`);
  if (bytes.length === 0) return invalid(`${shown} está vazio.`);
  if (!goodName(shown)) {
    return invalid(`O nome ${JSON.stringify(shown)} não serve a um anexo: até ${MaxDocumentNameBytes} bytes, sem caminho nem caractere de controle.`);
  }
  let mediaType: string;
  if (isPDF(bytes)) {
    if (bytes.length > MaxPDFBytes) return invalid(`${shown} tem ${megabytes(bytes.length)}, e um PDF vai ao modelo com até ${megabytes(MaxPDFBytes)}.`);
    mediaType = DocumentPDF;
  } else if (isText(bytes)) {
    if (bytes.length > MaxTextDocumentBytes) {
      return invalid(`${shown} tem ${kilobytes(bytes.length)}, e um arquivo de texto vai ao modelo com até ${kilobytes(MaxTextDocumentBytes)}.`);
    }
    mediaType = textTypeOf(shown);
  } else {
    return invalid(`${shown} não é um PDF nem um arquivo de texto que o modelo lê.`);
  }
  return { ok: true, value: { name: shown, mediaType, data: Buffer.from(bytes).toString('base64'), bytes: bytes.length } };
}

/**
 * The documents a turn is sent with, as the window holds them: checked again
 * here, since what crosses from the renderer is never taken on its word — the
 * count, each name, each type against the bytes it names, each size.
 */
export function turnDocuments(v: unknown): Checked<TurnDocument[]> {
  if (v === undefined) return { ok: true, value: [] };
  if (!Array.isArray(v)) return invalid('Os documentos do turno precisam vir numa lista.');
  if (v.length > MaxDocumentsPerTurn) return invalid(`Vieram ${v.length} documentos, e um turno leva até ${MaxDocumentsPerTurn}.`);
  const out: TurnDocument[] = [];
  for (const [i, doc] of v.entries()) {
    const which = `O documento ${i + 1}`;
    const rec = doc as Partial<DocumentAttachment> | null;
    if (typeof rec !== 'object' || rec === null || typeof rec.name !== 'string' || typeof rec.mediaType !== 'string' || typeof rec.data !== 'string') {
      return invalid(`${which} precisa vir com o nome, o tipo e os dados.`);
    }
    if (!goodName(rec.name)) return invalid(`${which} precisa de um nome de arquivo, sem caminho nem caractere de controle.`);
    const pdf = rec.mediaType === DocumentPDF;
    if (!pdf && !TEXT_TYPES.includes(rec.mediaType)) return invalid(`${which} é ${rec.mediaType}, e um turno leva PDF ou texto.`);
    if (rec.data.length % 4 !== 0 || !BASE64.test(rec.data)) return invalid(`${which} não está em base64.`);
    const bytes = new Uint8Array(Buffer.from(rec.data, 'base64'));
    const limit = pdf ? MaxPDFBytes : MaxTextDocumentBytes;
    if (bytes.length > limit) return invalid(`${which} tem ${megabytes(bytes.length)}, e o limite é ${megabytes(limit)}.`);
    if (pdf && !isPDF(bytes)) return invalid(`${which} diz ser um PDF, e os bytes não são.`);
    if (!pdf && !isText(bytes)) return invalid(`${which} diz ser texto, e os bytes não são UTF-8 sem NUL.`);
    out.push({ name: rec.name, media_type: rec.mediaType, data: rec.data });
  }
  return { ok: true, value: out };
}
