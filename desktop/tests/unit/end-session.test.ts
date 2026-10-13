import { describe, expect, it } from 'vitest';
import { ENDED, endWarning } from '../../src/state/end';

// Ending a session from the ⋯ menu (D51): the daemon closes it (DELETE), the
// record stays, and the conversation stays in the list as ended. What would be
// cut short is said before, and asked once more.

describe('ending a session', () => {
  it('asks nothing of a session at rest', () => {
    expect(endWarning('idle')).toBeNull();
    expect(endWarning(undefined)).toBeNull();
  });

  it('says what a running turn or a waiting approval loses, so the person confirms', () => {
    expect(endWarning('running')).toBe('Um turno está rodando: encerrar o interrompe no meio e fecha a sessão.');
    expect(endWarning('blocked')).toBe('Uma aprovação espera resposta: encerrar interrompe o turno e fecha a sessão.');
  });

  it('says where the conversation went once it ended', () => {
    expect(ENDED).toBe('Sessão encerrada. A conversa fica na lista, terminada; escrever nela continua numa sessão nova.');
  });
});
