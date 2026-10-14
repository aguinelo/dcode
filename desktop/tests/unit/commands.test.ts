import { describe, expect, it } from 'vitest';
import { compactText, parseInput, undoText } from '../../src/state/commands';

// What the field does with what was typed (D40): `!` runs a command, a few
// slash commands act on the session, and any other slash is said to be
// unknown — never sent to the model as if it were a message.

describe('what was typed in the field', () => {
  it('is a message when it is neither a command nor a slash', () => {
    expect(parseInput('conserte o parser')).toEqual({ kind: 'message', text: 'conserte o parser' });
    expect(parseInput('  use a/b no caminho')).toEqual({ kind: 'message', text: '  use a/b no caminho' });
  });

  it('runs what follows ! as a command, and refuses an empty one', () => {
    expect(parseInput('!go test ./...')).toEqual({ kind: 'exec', command: 'go test ./...' });
    expect(parseInput('  ! ls -la ')).toEqual({ kind: 'exec', command: 'ls -la' });
    expect(parseInput('!')).toMatchObject({ kind: 'bad' });
  });

  it('knows /undo, /compact and /mode with a mode', () => {
    expect(parseInput('/undo')).toEqual({ kind: 'undo' });
    expect(parseInput(' /compact ')).toEqual({ kind: 'compact' });
    expect(parseInput('/mode auto')).toEqual({ kind: 'mode', mode: 'auto' });
    expect(parseInput('/mode  plan')).toEqual({ kind: 'mode', mode: 'plan' });
  });

  it('says what /mode lacks, rather than guessing a mode', () => {
    const none = parseInput('/mode');
    expect(none).toMatchObject({ kind: 'bad' });
    expect(none.kind === 'bad' && none.why).toContain('plan, assist ou auto');
    const unknown = parseInput('/mode yolo');
    expect(unknown.kind === 'bad' && unknown.why).toContain('yolo');
  });

  it('says any other slash command is unknown, naming it and the ones that exist', () => {
    const got = parseInput('/model qwen');
    expect(got.kind).toBe('unknown');
    expect(got.kind === 'unknown' && got.why).toBe('Comando desconhecido: /model. Aqui há /help, /init, /plan, /undo, /compact, /mode <plan|assist|auto>, /loop <pasta|objetivo> e !comando.');
    expect(parseInput('/')).toMatchObject({ kind: 'unknown' });
  });
});

describe('what undo and compact answered', () => {
  it('says what went back, and what stayed because it changed after the turn', () => {
    expect(undoText({ restored: [], refused: [] })).toBe('O último turno não mexeu em arquivo nenhum.');
    expect(undoText({ restored: ['a.go', 'b.go'], refused: [] })).toBe('Desfeito. Restaurados: a.go, b.go.');
    expect(undoText({ restored: ['a.go'], refused: ['c.go'] })).toBe('Desfeito. Restaurados: a.go. Intocados, mudaram depois do turno: c.go.');
    expect(undoText({ restored: [], refused: ['c.go'] })).toBe('Nada voltou. Intocados, mudaram depois do turno: c.go.');
  });

  it('says when nothing was worth compacting', () => {
    expect(compactText({ compacted: true })).toBe('Contexto resumido.');
    expect(compactText({ compacted: false })).toBe('Nada que valesse resumir ainda: o contexto ficou como estava.');
  });
});
