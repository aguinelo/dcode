import { describe, expect, it } from 'vitest';
import { beginRename, commitRename, editRename, refusedRename, sentRename } from '../../src/state/rename';

// Renaming a conversation from its title (D43): the field opens on the title
// as it is, sends only what changed, and a refusal keeps what was typed. The
// title itself is never changed here — it comes back from the daemon.

describe('renaming a conversation from its title', () => {
  it('opens on the title as it is, for that conversation only', () => {
    expect(beginRename('s-1', 'corrige o login')).toEqual({ id: 's-1', draft: 'corrige o login', saving: false, refused: null });
  });

  it('closes without sending when nothing changed, spaces around included', () => {
    const r = editRename(beginRename('s-1', 'corrige o login'), '  corrige o login ');
    expect(commitRename(r, 'corrige o login', 'enter')).toEqual({ kind: 'close' });
    expect(commitRename(r, 'corrige o login', 'blur')).toEqual({ kind: 'close' });
  });

  it('sends the name trimmed, on Enter and on leaving the field', () => {
    const r = editRename(beginRename('s-1', 'corrige o login'), '  Login do SSO ');
    expect(commitRename(r, 'corrige o login', 'enter')).toEqual({ kind: 'send', name: 'Login do SSO' });
    expect(commitRename(r, 'corrige o login', 'blur')).toEqual({ kind: 'send', name: 'Login do SSO' });
  });

  it('sends an empty name, which gives the automatic title back', () => {
    const r = editRename(beginRename('s-1', 'Login do SSO'), '   ');
    expect(commitRename(r, 'Login do SSO', 'enter')).toEqual({ kind: 'send', name: '' });
  });

  it('sends once: while the daemon has not answered, the field stays as it is', () => {
    const r = sentRename(editRename(beginRename('s-1', 'a'), 'b'));
    expect(r.saving).toBe(true);
    expect(commitRename(r, 'a', 'enter')).toEqual({ kind: 'stay' });
    expect(commitRename(r, 'a', 'blur')).toEqual({ kind: 'stay' });
  });

  it('keeps the text after a refusal, and leaving the field does not send it again', () => {
    const r = refusedRename(sentRename(editRename(beginRename('s-1', 'a'), 'nome longo')));
    expect(r).toEqual({ id: 's-1', draft: 'nome longo', saving: false, refused: 'nome longo' });
    expect(commitRename(r, 'a', 'blur')).toEqual({ kind: 'stay' });
    // Enter is asking again, on purpose.
    expect(commitRename(r, 'a', 'enter')).toEqual({ kind: 'send', name: 'nome longo' });
    // What was typed after the refusal is new, and leaving the field sends it.
    expect(commitRename(editRename(r, 'nome'), 'a', 'blur')).toEqual({ kind: 'send', name: 'nome' });
  });
});
