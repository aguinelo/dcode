import { describe, expect, it } from 'vitest';
import { quitQuestion } from '../../src/main/quit';

describe('quitting with sessions at work', () => {
  it('asks in the window’s language how many sessions stop', () => {
    expect(quitQuestion(1).message).toBe('Fechar o DCode e parar 1 sessão?');
    expect(quitQuestion(1).detail).toContain('Uma sessão está rodando ou esperando aprovação');
    expect(quitQuestion(3).message).toBe('Fechar o DCode e parar 3 sessões?');
    expect(quitQuestion(3).detail).toContain('3 sessões estão rodando ou esperando aprovação');
  });

  it('offers Fechar and Cancelar, in that order', () => {
    expect(quitQuestion(2).buttons).toEqual(['Fechar', 'Cancelar']);
  });
});
