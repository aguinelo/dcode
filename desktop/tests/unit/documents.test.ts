import { describe, expect, it } from 'vitest';
import { MaxDocumentsPerTurn } from '../../src/protocol/generated';
import type { Attachment } from '../../src/shared/api';
import { isImage } from '../../src/shared/api';
import { attach, markSent, MAX_IMAGES, noAttachments, STEER_TAKES_TEXT } from '../../src/state/attachments';
import { log, created } from './helpers';

// The documents of a field (D59): attached beside the pictures, each kind to
// its own count, gone from the field once the daemon took them — and shown on
// the message from what turn.started says, never from what the window kept.

const img = (name: string): Attachment => ({ name, mediaType: 'image/png', data: 'iVBORw0KGgo=', bytes: 8 });
const pdf = (name: string): Attachment => ({ name, mediaType: 'application/pdf', data: 'JVBERi0=', bytes: 5 });

describe('the documents of a field (D59)', () => {
  it('attach beside the pictures, each kind up to its own count', () => {
    const pictures = Array.from({ length: MAX_IMAGES }, (_, i) => img(`${i}.png`));
    const docs = Array.from({ length: MaxDocumentsPerTurn + 1 }, (_, i) => pdf(`${i}.pdf`));
    const a = attach(noAttachments, 's1', [...pictures, ...docs]);
    expect(a.draft.s1?.filter(isImage)).toHaveLength(MAX_IMAGES);
    expect(a.draft.s1?.filter((x) => !isImage(x))).toHaveLength(MaxDocumentsPerTurn);
    expect(a.refusal.s1).toBe(`${MaxDocumentsPerTurn}.pdf ficou de fora: um turno leva até ${MaxDocumentsPerTurn} documentos.`);
  });

  it('leave the field once sent, and only the pictures are kept to be shown', () => {
    const p = img('a.png');
    const d = pdf('spec.pdf');
    let a = attach(noAttachments, 's1', [p, d]);
    a = markSent(a, 's1', 3, [p, d]);
    expect(a.draft.s1).toEqual([]);
    expect(a.sent.s1).toEqual([{ after: 3, images: [p] }]);
    // Documents alone leave too, with nothing kept: turn.started names them.
    const b = markSent(attach(noAttachments, 's1', [d]), 's1', 4, [d]);
    expect(b.draft.s1).toEqual([]);
    expect(b.sent.s1).toBeUndefined();
  });

  it('are said as attachments when a correction cannot carry them', () => {
    expect(STEER_TAKES_TEXT).toMatch(/os anexos ficam/);
  });
});

describe('a message with documents', () => {
  it('shows them from turn.started, live and read back alike', () => {
    const v = log()
      .add('session.created', created)
      .add('turn.started', {
        turn_id: 't-1',
        text: 'Compare os dois.',
        documents: [
          { name: 'spec.pdf', media_type: 'application/pdf', size: 2048 },
          { name: 'notas.md', media_type: 'text/markdown', size: 120 },
        ],
      })
      .fold();
    expect(v.entries).toMatchObject([
      {
        kind: 'user',
        text: 'Compare os dois.',
        documents: [
          { name: 'spec.pdf', media_type: 'application/pdf', size: 2048 },
          { name: 'notas.md', media_type: 'text/markdown', size: 120 },
        ],
      },
    ]);
  });
});
