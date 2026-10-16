import { describe, expect, it } from 'vitest';
import { PR_BODY_LIMIT, PR_TITLE_LIMIT, prDraft } from '../../src/main/inputs';
import { PR_CREATE_TIMEOUT_MS, PR_READ_TIMEOUT_MS } from '../../src/main/requests';

// A new PR as the confirmation sheet sends it (D60), checked before it reaches
// the daemon: what the person saw goes as they saw it, or is refused with why.

function refused(got: { ok: boolean; refusal?: { code: string; message: string } }): string {
  expect(got.ok).toBe(false);
  expect(got.refusal?.code).toBe('invalid_input');
  return got.refusal?.message ?? '';
}

const ok = { title: 'Add b', body: 'It adds b.\n', base: 'main', draft: true };

describe('a PR the sheet confirmed', () => {
  it('goes as it was shown, untrimmed', () => {
    expect(prDraft(ok)).toEqual({ ok: true, value: ok });
    expect(prDraft({ ...ok, base: '' })).toEqual({ ok: true, value: { ...ok, base: '' } });
    expect(prDraft({ ...ok, title: ' Add b ' })).toEqual({ ok: true, value: { ...ok, title: ' Add b ' } });
  });

  it('is refused when it is not one GitHub or the daemon would take', () => {
    expect(refused(prDraft(null))).toContain('título, corpo, base e rascunho');
    expect(refused(prDraft({ ...ok, title: '  ' }))).toContain('título');
    expect(refused(prDraft({ ...ok, title: 'a\nb' }))).toContain('uma linha');
    expect(refused(prDraft({ ...ok, title: 'x'.repeat(PR_TITLE_LIMIT + 1) }))).toContain(String(PR_TITLE_LIMIT));
    expect(refused(prDraft({ ...ok, body: 'é'.repeat(PR_BODY_LIMIT / 2 + 1) }))).toContain('bytes');
    expect(refused(prDraft({ ...ok, body: 3 }))).toContain('corpo');
    for (const base of ['-x', 'a b', 'a:b', 7]) expect(refused(prDraft({ ...ok, base }))).toContain('base');
    expect(refused(prDraft({ ...ok, draft: 'yes' }))).toContain('Rascunho');
  });

  it('waits for the daemon’s own deadlines, a little longer', () => {
    expect(PR_READ_TIMEOUT_MS).toBeGreaterThan(20_000);
    expect(PR_CREATE_TIMEOUT_MS).toBeGreaterThan(120_000);
  });
});
