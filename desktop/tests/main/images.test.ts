import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { readAttachmentFile, readAttachmentFiles } from '../../src/main/attachments';
import { imageFromBytes, MAX_IMAGE_BYTES, MAX_IMAGES, sniff, turnImages } from '../../src/main/images';

// The pictures a turn carries (D46), checked at the main process's boundary:
// the type from the bytes, the size before the read, the count before the send.

const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 13]);
const JPEG = new Uint8Array([0xff, 0xd8, 0xff, 0xe0, 0, 16]);
const GIF = new Uint8Array([...Buffer.from('GIF89a'), 1, 0]);
const WEBP = new Uint8Array([...Buffer.from('RIFF'), 0, 0, 0, 0, ...Buffer.from('WEBPVP8 ')]);
const b64 = (b: Uint8Array) => Buffer.from(b).toString('base64');

const dirs: string[] = [];
function temp(): string {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'dcimg-'));
  dirs.push(dir);
  return dir;
}
afterEach(() => {
  for (const d of dirs.splice(0)) rmSync(d, { recursive: true, force: true });
});

describe('a picture’s type', () => {
  it('comes from its first bytes, for the four the providers take', () => {
    expect(sniff(PNG)).toBe('image/png');
    expect(sniff(JPEG)).toBe('image/jpeg');
    expect(sniff(GIF)).toBe('image/gif');
    expect(sniff(WEBP)).toBe('image/webp');
    expect(sniff(new Uint8Array(Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>')))).toBeNull();
    expect(sniff(new Uint8Array([0x49, 0x49, 0x2a, 0x00]))).toBeNull(); // TIFF
    expect(sniff(new Uint8Array())).toBeNull();
  });
});

describe('a pasted or dropped picture', () => {
  it('is read from its bytes, named as the clipboard named it', () => {
    expect(imageFromBytes('captura.png', PNG)).toEqual({
      ok: true,
      value: { name: 'captura.png', mediaType: 'image/png', data: b64(PNG), bytes: PNG.length },
    });
  });

  it('is refused, said why, when it is not an image the model reads, is empty, too big, or not bytes', () => {
    expect(imageFromBytes('notas.txt', new Uint8Array(Buffer.from('oi')))).toEqual({
      ok: false,
      refusal: { code: 'invalid_input', message: 'notas.txt não é uma imagem que o modelo lê: só PNG, JPEG, GIF e WebP.' },
    });
    expect(imageFromBytes('a.png', new Uint8Array())).toMatchObject({ ok: false, refusal: { message: 'a.png está vazia.' } });
    const big = new Uint8Array(MAX_IMAGE_BYTES + 1);
    big.set(PNG);
    expect(imageFromBytes('grande.png', big)).toMatchObject({
      ok: false,
      refusal: { message: 'grande.png tem 10,0 MB, e uma imagem vai ao modelo com até 10,0 MB.' },
    });
    expect(imageFromBytes('a.png', 'iVBORw0KGgo=')).toMatchObject({ ok: false, refusal: { message: 'a.png não chegou como bytes de uma imagem.' } });
    // A name is only shown: a path in it is cut to the file's name.
    expect(imageFromBytes('/etc/../x/shot.png', PNG)).toMatchObject({ ok: true, value: { name: 'shot.png' } });
    expect(imageFromBytes(undefined, PNG)).toMatchObject({ ok: true, value: { name: 'imagem' } });
  });
});

describe('a picked picture', () => {
  it('is read by what its bytes are, not by its name', async () => {
    const file = path.join(temp(), 'foto.jpg');
    writeFileSync(file, PNG);
    expect(await readAttachmentFile(file)).toMatchObject({ ok: true, value: { name: 'foto.jpg', mediaType: 'image/png', bytes: PNG.length } });
  });

  it('attaches the good ones of a pick and says each one refused, past the count too', async () => {
    const dir = temp();
    const files = [];
    for (let i = 0; i < MAX_IMAGES + 1; i++) {
      const f = path.join(dir, `${i}.png`);
      writeFileSync(f, PNG);
      files.push(f);
    }
    const bin = path.join(dir, 'leia.bin');
    writeFileSync(bin, new Uint8Array([0xff, 0xfe, 0x00]));
    const got = await readAttachmentFiles([bin, ...files]);
    expect(got.attachments).toHaveLength(MAX_IMAGES);
    expect(got.refused).toEqual([
      'leia.bin não é um PDF nem um arquivo de texto que o modelo lê.',
      `${MAX_IMAGES}.png ficou de fora: um turno leva até ${MAX_IMAGES} imagens.`,
    ]);
  });
});

describe('the pictures a turn is sent with', () => {
  const attached = (bytes: Uint8Array, mediaType = sniff(bytes) ?? 'image/png') => ({ name: 'x', mediaType, data: b64(bytes), bytes: bytes.length });

  it('go on the wire as the protocol carries them, and none is an empty list', () => {
    expect(turnImages(undefined)).toEqual({ ok: true, value: [] });
    expect(turnImages([attached(PNG), attached(JPEG)])).toEqual({
      ok: true,
      value: [
        { media_type: 'image/png', data: b64(PNG) },
        { media_type: 'image/jpeg', data: b64(JPEG) },
      ],
    });
  });

  it('are checked again, never taken on the window’s word', () => {
    const msg = (v: unknown) => {
      const got = turnImages(v);
      return got.ok ? 'ok' : got.refusal.message;
    };
    expect(msg('uma')).toBe('As imagens do turno precisam vir numa lista.');
    expect(msg(Array.from({ length: MAX_IMAGES + 1 }, () => attached(PNG)))).toBe(`Vieram ${MAX_IMAGES + 1} imagens, e um turno leva até ${MAX_IMAGES}.`);
    expect(msg([null])).toBe('A imagem 1 precisa vir com o tipo e os dados.');
    expect(msg([attached(PNG, 'image/svg+xml')])).toBe('A imagem 1 é image/svg+xml, e o modelo lê só image/png, image/jpeg, image/gif, image/webp.');
    expect(msg([attached(PNG), { name: 'x', mediaType: 'image/png', data: 'não é base64' }])).toBe('A imagem 2 não está em base64.');
    expect(msg([attached(PNG, 'image/jpeg')])).toBe('A imagem 1 diz ser image/jpeg, e os bytes são image/png.');
    expect(msg([{ name: 'x', mediaType: 'image/png', data: b64(new Uint8Array(Buffer.from('oi!'))) }])).toBe(
      'A imagem 1 diz ser image/png, e os bytes são de outra coisa.',
    );
  });
});
