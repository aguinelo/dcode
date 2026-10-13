import { describe, expect, it } from 'vitest';
import type { Entry } from '../../src/state/session';
import { notRecalling, olderPrompts, promptsOf, recallDown, recallUp, sourceOf, wantsDown, wantsUp } from '../../src/state/recall';

// ↑/↓ in a conversation's field (D47): what the person asked before comes back,
// newest first, and leaving the walk gives back what was being written. The
// arrows are taken only at the field's edges, so a message of several lines is
// still edited the way a text field is.

const at = '2026-10-13T12:00:00Z';
const user = (seq: number, text: string, steer = false): Entry => ({ kind: 'user', seq, at, text, steer });

describe('the prompts a conversation remembers', () => {
  it('are what was asked and steered, oldest first, trimmed, consecutive repeats once', () => {
    const entries: Entry[] = [
      user(1, ' corrige o login '),
      { kind: 'model', seq: 2, at, turnId: 't', round: 1, text: 'feito' },
      user(3, 'e o logout', true),
      user(4, 'e o logout'),
      user(5, '   '),
      user(6, 'corrige o login'),
    ];
    expect(promptsOf(entries)).toEqual(['corrige o login', 'e o logout', 'corrige o login']);
  });

  it('come before a continuation’s own, the repeat at the seam once', () => {
    expect(olderPrompts(['a', 'b'], ['b', 'c'])).toEqual(['a', 'b', 'c']);
    expect(olderPrompts([], ['c'])).toEqual(['c']);
  });

  it('say which conversation a continuation came from', () => {
    const resumed: Entry = { kind: 'note', seq: 1, at, note: { kind: 'resumed', sourceId: 's-0', turns: 2 } };
    expect(sourceOf([resumed, user(2, 'oi')])).toBe('s-0');
    expect(sourceOf([user(2, 'oi')])).toBeNull();
  });
});

describe('walking the prompts with the arrows', () => {
  const list = ['primeiro', 'segundo', 'terceiro'];

  it('↑ goes back one at a time, keeping the draft, and stops at the oldest', () => {
    let r = notRecalling;
    const one = recallUp(list, r, 'rascunho');
    expect(one).toEqual({ recall: { back: 1, draft: 'rascunho' }, text: 'terceiro' });
    r = one!.recall;
    const two = recallUp(list, r, 'terceiro editado');
    // The draft is what was there when the walk began, not what a recalled prompt became.
    expect(two).toEqual({ recall: { back: 2, draft: 'rascunho' }, text: 'segundo' });
    r = recallUp(list, two!.recall, 'segundo')!.recall;
    expect(r.back).toBe(3);
    expect(recallUp(list, r, 'primeiro')).toBeNull();
  });

  it('↓ comes forward and, past the newest, gives the draft back', () => {
    const r = recallUp(list, recallUp(list, notRecalling, 'rascunho')!.recall, 'terceiro')!.recall;
    const fwd = recallDown(list, r);
    expect(fwd).toEqual({ recall: { back: 1, draft: 'rascunho' }, text: 'terceiro' });
    expect(recallDown(list, fwd!.recall)).toEqual({ recall: notRecalling, text: 'rascunho' });
  });

  it('does nothing with nothing to recall, or ↓ when not walking', () => {
    expect(recallUp([], notRecalling, '')).toBeNull();
    expect(recallDown(list, notRecalling)).toBeNull();
  });

  it('keeps its place when older prompts arrive in front of the list', () => {
    const r = recallUp(list, notRecalling, '')!.recall;
    expect(recallUp(['zero', ...list], r, 'terceiro')?.text).toBe('segundo');
  });
});

describe('when the arrows are the field’s and when they recall', () => {
  it('↑ recalls from an empty field, or with the caret at the start', () => {
    expect(wantsUp('', 0, 0, notRecalling)).toBe(true);
    expect(wantsUp('um texto', 0, 0, notRecalling)).toBe(true);
  });

  it('↑ in a draft is the field’s until the caret reaches the start', () => {
    expect(wantsUp('um texto', 4, 4, notRecalling)).toBe(false);
  });

  it('↑ below the first line moves between lines, even while walking', () => {
    const walking = { back: 1, draft: '' };
    expect(wantsUp('linha um\nlinha dois', 12, 12, walking)).toBe(false);
    expect(wantsUp('linha um\nlinha dois', 3, 3, walking)).toBe(true);
  });

  it('never with text selected', () => {
    expect(wantsUp('um texto', 0, 3, notRecalling)).toBe(false);
    expect(wantsDown('um texto', 0, 3, { back: 1, draft: '' })).toBe(false);
  });

  it('↓ comes forward only while walking, and only from the last line', () => {
    const walking = { back: 1, draft: '' };
    expect(wantsDown('um', 2, 2, notRecalling)).toBe(false);
    expect(wantsDown('um', 2, 2, walking)).toBe(true);
    expect(wantsDown('linha um\nlinha dois', 3, 3, walking)).toBe(false);
    expect(wantsDown('linha um\nlinha dois', 12, 12, walking)).toBe(true);
  });
});
