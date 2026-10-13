import { describe, expect, it } from 'vitest';
import type { ImageAttachment } from '../../src/shared/api';
import {
  attach,
  detach,
  imageSource,
  imagesOfEntries,
  markSent,
  MAX_IMAGES,
  moveDraft,
  noAttachments,
  refuse,
  sizeOf,
} from '../../src/state/attachments';
import type { Entry } from '../../src/state/session';

const img = (name: string): ImageAttachment => ({ name, mediaType: 'image/png', data: 'iVBORw0KGgo=', bytes: 8 });
const user = (seq: number, steer = false): Entry => ({ kind: 'user', seq, at: '', text: `m${seq}`, steer });

describe('the pictures of a field (D46)', () => {
  it('attach per conversation, and come out one at a time', () => {
    let a = attach(noAttachments, 's1', [img('a'), img('b')]);
    a = attach(a, 's2', [img('c')]);
    expect(a.draft.s1?.map((i) => i.name)).toEqual(['a', 'b']);
    a = detach(a, 's1', 0);
    expect(a.draft.s1?.map((i) => i.name)).toEqual(['b']);
    expect(detach(a, 's1', 5)).toBe(a);
    expect(a.draft.s2?.map((i) => i.name)).toEqual(['c']);
  });

  it('say each refusal in the field, and past the count name the ones left out', () => {
    const full = attach(noAttachments, 's1', Array.from({ length: MAX_IMAGES - 1 }, (_, i) => img(`${i}`)));
    const a = attach(full, 's1', [img('cabe'), img('sobra')], ['leia.txt não é uma imagem que o modelo lê.']);
    expect(a.draft.s1).toHaveLength(MAX_IMAGES);
    expect(a.refusal.s1).toBe(`leia.txt não é uma imagem que o modelo lê. sobra ficou de fora: um turno leva até ${MAX_IMAGES} imagens.`);
    // A good attach clears what was said; taking one out does too.
    expect(attach(refuse(noAttachments, 's1', 'não'), 's1', [img('a')]).refusal.s1).toBeUndefined();
    expect(detach(attach(refuse(noAttachments, 's1', 'não'), 's1', [img('a')]), 's1', 0).refusal.s1).toBeUndefined();
    // Nothing attached and nothing refused keeps what was said.
    expect(attach(refuse(noAttachments, 's1', 'não'), 's1', []).refusal.s1).toBe('não');
  });

  it('leave the field once sent, all but one attached while the message was on its way', () => {
    const a1 = img('a');
    const late = img('tarde');
    let a = attach(noAttachments, 's1', [a1]);
    a = attach(a, 's1', [late]);
    a = markSent(a, 's1', 7, [a1]);
    expect(a.draft.s1).toEqual([late]);
    expect(a.sent.s1).toEqual([{ after: 7, images: [a1] }]);
    expect(markSent(a, 's1', 9, [])).toBe(a);
  });

  it('go with the text to the session that continues an ended conversation', () => {
    const a = moveDraft(attach(noAttachments, 'old', [img('a')]), 'old', 'new');
    expect(a.draft.old).toBeUndefined();
    expect(a.draft.new?.map((i) => i.name)).toEqual(['a']);
    expect(moveDraft(a, 'none', 'new')).toBe(a);
  });

  it('are shown on the first message after the seq they were sent at, never on a correction', () => {
    const entries = [user(2), user(5, true), user(6), user(9)];
    const sent = [
      { after: 4, images: [img('a')] },
      { after: 8, images: [img('b')] },
    ];
    const at = imagesOfEntries(entries, sent);
    expect([...at.keys()]).toEqual([6, 9]);
    expect(at.get(6)?.[0]?.name).toBe('a');
    // A message whose turn has not started yet has no entry: nothing is shown until it does.
    expect(imagesOfEntries([user(2)], sent).size).toBe(0);
    expect(imagesOfEntries(entries, undefined).size).toBe(0);
  });

  it('are drawn from data URLs, and their size reads short', () => {
    expect(imageSource(img('a'))).toBe('data:image/png;base64,iVBORw0KGgo=');
    expect(sizeOf(900)).toBe('900 B');
    expect(sizeOf(20 * 1024)).toBe('20 KB');
    expect(sizeOf(3.5 * 1024 * 1024)).toBe('3,5 MB');
  });
});
