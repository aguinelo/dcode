import { mkdirSync, mkdtempSync, rmSync, truncateSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { attachmentFromBytes, readAttachmentFile, readAttachmentFiles, turnAttachments } from '../../src/main/attachments';
import { documentFromBytes, turnDocuments } from '../../src/main/documents';
import { MAX_IMAGES } from '../../src/main/images';
import { MaxDocumentsPerTurn, MaxPDFBytes, MaxTextDocumentBytes } from '../../src/protocol/generated';

// The documents a turn carries (D59), checked at the main process's boundary
// by the rules the daemon holds them to: a PDF by its bytes, text as UTF-8
// without NUL, a name that is a file's, the protocol's sizes and count.

const PDF = new Uint8Array(Buffer.from('%PDF-1.7\n1 0 obj\n'));
const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 13]);
const text = (s: string) => new Uint8Array(Buffer.from(s));
const b64 = (b: Uint8Array) => Buffer.from(b).toString('base64');

const dirs: string[] = [];
function temp(): string {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'dcdoc-'));
  dirs.push(dir);
  return dir;
}
afterEach(() => {
  for (const d of dirs.splice(0)) rmSync(d, { recursive: true, force: true });
});

describe('a document’s bytes', () => {
  it('make a PDF by their first bytes, whatever the name says', () => {
    expect(documentFromBytes('relatorio.txt', PDF)).toEqual({
      ok: true,
      value: { name: 'relatorio.txt', mediaType: 'application/pdf', data: b64(PDF), bytes: PDF.length },
    });
  });

  it('make text when they are UTF-8 without NUL, typed by the extension only among the text kinds', () => {
    expect(documentFromBytes('notas.md', text('# título'))).toMatchObject({ ok: true, value: { mediaType: 'text/markdown' } });
    expect(documentFromBytes('dados.csv', text('a,b'))).toMatchObject({ ok: true, value: { mediaType: 'text/csv' } });
    expect(documentFromBytes('main.go', text('package main'))).toMatchObject({ ok: true, value: { mediaType: 'text/plain' } });
    expect(documentFromBytes('LEIAME', text('oi'))).toMatchObject({ ok: true, value: { mediaType: 'text/plain' } });
  });

  it('are refused, said why, when they are neither, empty, too large, or not bytes', () => {
    const msg = (name: unknown, bytes: unknown) => {
      const got = documentFromBytes(name, bytes);
      return got.ok ? 'ok' : got.refusal.message;
    };
    expect(msg('a.bin', new Uint8Array([0xff, 0xfe, 0x00]))).toBe('a.bin não é um PDF nem um arquivo de texto que o modelo lê.');
    expect(msg('a.txt', text('a\u0000b'))).toBe('a.txt não é um PDF nem um arquivo de texto que o modelo lê.');
    expect(msg('a.txt', new Uint8Array())).toBe('a.txt está vazio.');
    expect(msg('a.txt', 'oi')).toBe('a.txt não chegou como bytes de um arquivo.');
    expect(msg('grande.txt', new Uint8Array(MaxTextDocumentBytes + 1024).fill(0x61))).toBe(
      'grande.txt tem 257 KB, e um arquivo de texto vai ao modelo com até 256 KB.',
    );
    const big = new Uint8Array(MaxPDFBytes + 1);
    big.set(PDF);
    expect(msg('grande.pdf', big)).toBe('grande.pdf tem 10,0 MB, e um PDF vai ao modelo com até 10,0 MB.');
  });

  it('carry the file’s name, never a path, and refuse one the daemon would', () => {
    expect(documentFromBytes('/Users/x/../docs/spec.pdf', PDF)).toMatchObject({ ok: true, value: { name: 'spec.pdf' } });
    expect(documentFromBytes(undefined, PDF)).toMatchObject({ ok: true, value: { name: 'documento' } });
    expect(documentFromBytes('a\nb.txt', text('x'))).toMatchObject({ ok: false, refusal: { code: 'invalid_input' } });
    expect(documentFromBytes(`${'a'.repeat(300)}.txt`, text('x'))).toMatchObject({ ok: false, refusal: { code: 'invalid_input' } });
  });
});

describe('a pasted, dropped or picked file', () => {
  it('is an image or a document by its bytes', () => {
    expect(attachmentFromBytes('x.pdf', PNG)).toMatchObject({ ok: true, value: { mediaType: 'image/png' } });
    expect(attachmentFromBytes('x.png', PDF)).toMatchObject({ ok: true, value: { mediaType: 'application/pdf' } });
    expect(attachmentFromBytes('x.png', text('oi'))).toMatchObject({ ok: true, value: { mediaType: 'text/plain' } });
  });

  it('is refused by its size before it is read, and when it is missing or a folder', async () => {
    const dir = temp();
    const big = path.join(dir, 'grande.pdf');
    writeFileSync(big, PDF);
    truncateSync(big, MaxPDFBytes + 1024 * 1024);
    expect(await readAttachmentFile(big)).toMatchObject({
      ok: false,
      refusal: { code: 'invalid_input', message: 'grande.pdf tem 11,0 MB, e um anexo vai ao modelo com até 10,0 MB.' },
    });
    const missing = await readAttachmentFile(path.join(dir, 'nada.pdf'));
    expect(missing.ok).toBe(false);
    if (!missing.ok) expect(missing.refusal.message).toMatch(/^nada\.pdf não foi lido: ENOENT/);
    mkdirSync(path.join(dir, 'pasta.pdf'));
    expect(await readAttachmentFile(path.join(dir, 'pasta.pdf'))).toMatchObject({ ok: false, refusal: { message: 'pasta.pdf não é um arquivo.' } });
  });

  it('of a pick attaches the good ones and says each one refused, past each count too', async () => {
    const dir = temp();
    const files: string[] = [];
    for (let i = 0; i < MaxDocumentsPerTurn + 1; i++) {
      const f = path.join(dir, `${i}.pdf`);
      writeFileSync(f, PDF);
      files.push(f);
    }
    for (let i = 0; i < MAX_IMAGES + 1; i++) {
      const f = path.join(dir, `${i}.png`);
      writeFileSync(f, PNG);
      files.push(f);
    }
    const bin = path.join(dir, 'a.bin');
    writeFileSync(bin, new Uint8Array([0xff, 0x00]));
    const got = await readAttachmentFiles([bin, ...files]);
    expect(got.attachments.filter((a) => a.mediaType === 'application/pdf')).toHaveLength(MaxDocumentsPerTurn);
    expect(got.attachments.filter((a) => a.mediaType === 'image/png')).toHaveLength(MAX_IMAGES);
    expect(got.refused).toEqual([
      'a.bin não é um PDF nem um arquivo de texto que o modelo lê.',
      `${MaxDocumentsPerTurn}.pdf ficou de fora: um turno leva até ${MaxDocumentsPerTurn} documentos.`,
      `${MAX_IMAGES}.png ficou de fora: um turno leva até ${MAX_IMAGES} imagens.`,
    ]);
  });
});

describe('the documents a turn is sent with', () => {
  const doc = (name: string, mediaType: string, bytes: Uint8Array) => ({ name, mediaType, data: b64(bytes), bytes: bytes.length });

  it('go on the wire as the protocol carries them, apart from the pictures', () => {
    expect(turnAttachments(undefined)).toEqual({ ok: true, value: { images: [], documents: [] } });
    expect(
      turnAttachments([doc('a.png', 'image/png', PNG), doc('spec.pdf', 'application/pdf', PDF), doc('n.md', 'text/markdown', text('# x'))]),
    ).toEqual({
      ok: true,
      value: {
        images: [{ media_type: 'image/png', data: b64(PNG) }],
        documents: [
          { name: 'spec.pdf', media_type: 'application/pdf', data: b64(PDF) },
          { name: 'n.md', media_type: 'text/markdown', data: b64(text('# x')) },
        ],
      },
    });
  });

  it('are checked again, never taken on the window’s word', () => {
    const msg = (v: unknown) => {
      const got = turnDocuments(v);
      return got.ok ? 'ok' : got.refusal.message;
    };
    expect(msg('um')).toBe('Os documentos do turno precisam vir numa lista.');
    expect(msg(Array.from({ length: MaxDocumentsPerTurn + 1 }, () => doc('a.pdf', 'application/pdf', PDF)))).toBe(
      `Vieram ${MaxDocumentsPerTurn + 1} documentos, e um turno leva até ${MaxDocumentsPerTurn}.`,
    );
    expect(msg([null])).toBe('O documento 1 precisa vir com o nome, o tipo e os dados.');
    expect(msg([doc('a.docx', 'application/msword', PDF)])).toBe('O documento 1 é application/msword, e um turno leva PDF ou texto.');
    expect(msg([{ name: 'a.pdf', mediaType: 'application/pdf', data: 'não é base64' }])).toBe('O documento 1 não está em base64.');
    expect(msg([doc('a.pdf', 'application/pdf', text('oi'))])).toBe('O documento 1 diz ser um PDF, e os bytes não são.');
    expect(msg([doc('a.txt', 'text/plain', new Uint8Array([0xff]))])).toBe('O documento 1 diz ser texto, e os bytes não são UTF-8 sem NUL.');
    expect(msg([doc('../a.txt', 'text/plain', text('x'))])).toBe('O documento 1 precisa de um nome de arquivo, sem caminho nem caractere de controle.');
  });
});
