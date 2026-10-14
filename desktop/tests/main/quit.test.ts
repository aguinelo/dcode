import { describe, expect, it } from 'vitest';
import { quitQuestion } from '../../src/main/quit';

describe('quitting with sessions at work', () => {
  it('asks in the window’s language how many sessions stop', () => {
    expect(quitQuestion(1).message).toBe('Fechar o DCode Box e parar 1 sessão?');
    expect(quitQuestion(1).detail).toContain('Uma sessão está rodando ou esperando aprovação');
    expect(quitQuestion(3).message).toBe('Fechar o DCode Box e parar 3 sessões?');
    expect(quitQuestion(3).detail).toContain('3 sessões estão rodando ou esperando aprovação');
  });

  it('offers Fechar and Cancelar, in that order', () => {
    expect(quitQuestion(2).buttons).toEqual(['Fechar', 'Cancelar']);
  });
});

describe('quitting with loops alive (D19, D55)', () => {
  it('says the loops end with the daemon, alone', () => {
    const q = quitQuestion(0, 1);
    expect(q.message).toBe('Fechar o DCode Box e encerrar 1 loop?');
    expect(q.detail).toBe(
      'Um loop está vivo. O daemon foi aberto pelo DCode Box e fecha com ele, e um loop não sobrevive ao daemon: ele termina junto, e não volta quando o app abrir de novo.',
    );
    expect(quitQuestion(0, 2).message).toBe('Fechar o DCode Box e encerrar 2 loops?');
    expect(quitQuestion(0, 2).detail).toContain('2 loops estão vivos.');
  });

  it('counts sessions and loops together', () => {
    const q = quitQuestion(2, 1);
    expect(q.message).toBe('Fechar o DCode Box, parar 2 sessões e encerrar 1 loop?');
    expect(q.detail).toContain('2 sessões estão rodando ou esperando aprovação, e um loop está vivo.');
    expect(q.detail).toContain('não sobrevive ao daemon');
  });
});
