import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { decodeModels } from '../../src/protocol/models';
import { measureOf, menuOf } from '../../src/state/models';

const minimax: P.ModelChoice = { name: 'MiniMax-M3', model: 'MiniMax-M3', family: 'minimax-m3', transport: 'openai', window: 1000000, measured: true };
const local: P.ModelChoice = {
  name: 'qwen-local',
  model: 'qwen3.5-9b',
  family: 'generic',
  transport: 'openai',
  base_url: 'http://127.0.0.1:1234/v1',
  window: 32000,
  measured: false,
  notice: 'using --family generic: nobody measured this endpoint',
};
const gemini: P.ModelChoice = { name: 'gemini', model: 'gemini-2.5-pro', family: 'gemini', transport: 'openai', measured: true };

describe('the answer of GET /v1/models', () => {
  it('reads the default and the profiles, and an empty or null list as none', () => {
    expect(decodeModels({ default: minimax, profiles: [local] })).toEqual({ ok: true, models: { default: minimax, profiles: [local] } });
    expect(decodeModels({ default: minimax, profiles: null })).toEqual({ ok: true, models: { default: minimax, profiles: [] } });
    expect(decodeModels({ default: minimax })).toEqual({ ok: true, models: { default: minimax, profiles: [] } });
  });

  it('says what does not match, and never guesses a menu from it', () => {
    const bad = (raw: unknown) => {
      const d = decodeModels(raw);
      return d.ok ? 'read' : d.reason;
    };
    expect(bad('menu')).toBe('lista de modelos: a resposta não é um objeto');
    expect(bad({ profiles: [] })).toContain('default');
    expect(bad({ default: { ...minimax, measured: 'yes' } })).toContain('default.measured');
    expect(bad({ default: minimax, profiles: {} })).toBe('lista de modelos: profiles deveria ser uma lista');
    expect(bad({ default: minimax, profiles: [minimax, { name: 'x' }] })).toContain('profiles[1]');
  });
});

describe('the model menu', () => {
  const models: P.ModelsResponse = { default: minimax, profiles: [gemini, local] };

  it('offers the project’s default first, then each profile, saying what each runs and where', () => {
    const menu = menuOf(models, null);
    expect(menu.map((o) => [o.name, o.detail, o.isDefault])).toEqual([
      ['MiniMax-M3', 'padrão do projeto', true],
      ['gemini', 'gemini-2.5-pro', false],
      ['qwen-local', 'qwen3.5-9b · 127.0.0.1:1234', false],
    ]);
  });

  it('says which were measured, and carries the daemon’s words only for those that were not', () => {
    const menu = menuOf(models, null);
    expect(menu.map((o) => [o.measured, o.notice])).toEqual([
      [true, null],
      [true, null],
      [false, 'using --family generic: nobody measured this endpoint'],
    ]);
  });

  it('marks what the conversation runs on, by model and endpoint when the endpoint is known', () => {
    const current = (running: Parameters<typeof menuOf>[1]) => menuOf(models, running).flatMap((o) => (o.current ? [o.name] : []));
    expect(current({ model: 'qwen3.5-9b', baseUrl: 'http://127.0.0.1:1234/v1' })).toEqual(['qwen-local']);
    // The same model on another endpoint is another place to run.
    expect(current({ model: 'qwen3.5-9b', baseUrl: '' })).toEqual([]);
    // Only the model known: it is enough to say.
    expect(current({ model: 'MiniMax-M3', baseUrl: null })).toEqual(['MiniMax-M3']);
    expect(current(null)).toEqual([]);
  });
});

describe('the measure of a conversation', () => {
  const models: P.ModelsResponse = { default: minimax, profiles: [gemini, local] };

  it('reads the family the session announced, else the model the list carries', () => {
    expect(measureOf(models, { model: 'qwen3.5-9b', family: 'generic' })).toEqual({ measured: false, family: 'generic' });
    expect(measureOf(models, { model: 'anything', family: 'gemini' })).toEqual({ measured: true, family: 'gemini' });
    expect(measureOf(models, { model: 'qwen3.5-9b', family: null })).toEqual({ measured: false, family: 'generic' });
  });

  it('says nothing when the list does not say: not known is never measured', () => {
    expect(measureOf(undefined, { model: 'qwen3.5-9b', family: 'generic' })).toBeNull();
    expect(measureOf(models, { model: 'llama3', family: 'ollama' })).toBeNull();
    expect(measureOf(models, null)).toBeNull();
  });

  it('takes a family as measured when any choice of it builds a measured session', () => {
    const broken: P.ModelChoice = { ...gemini, name: 'typo', transport: 'nope', measured: false, notice: 'no transport named "nope"' };
    expect(measureOf({ default: minimax, profiles: [broken, gemini] }, { model: 'x', family: 'gemini' })).toEqual({ measured: true, family: 'gemini' });
  });
});
