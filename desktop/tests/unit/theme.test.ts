import { describe, expect, it } from 'vitest';
import { DEFAULT_THEME, parseTheme, resolveTheme, THEME_CHOICES } from '../../src/state/theme';

describe('parseTheme', () => {
  it('starts dark when nothing was stored, and says nothing', () => {
    expect(parseTheme(null)).toEqual({ choice: 'escuro', problem: null });
    expect(DEFAULT_THEME).toBe('escuro');
  });

  it('reads each of the three choices back', () => {
    for (const c of THEME_CHOICES) expect(parseTheme(c.value)).toEqual({ choice: c.value, problem: null });
  });

  it('falls back to dark on a stored value it does not know, and says so', () => {
    const got = parseTheme('light');
    expect(got.choice).toBe('escuro');
    expect(got.problem).toContain('light');
  });
});

describe('resolveTheme', () => {
  it('keeps a fixed choice whatever the system says', () => {
    expect(resolveTheme('claro', true)).toBe('light');
    expect(resolveTheme('claro', false)).toBe('light');
    expect(resolveTheme('escuro', true)).toBe('dark');
    expect(resolveTheme('escuro', false)).toBe('dark');
  });

  it('follows the system under Sistema', () => {
    expect(resolveTheme('sistema', true)).toBe('dark');
    expect(resolveTheme('sistema', false)).toBe('light');
  });
});
