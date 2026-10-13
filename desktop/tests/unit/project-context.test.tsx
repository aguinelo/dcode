import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ProjectContextBody } from '../../src/renderer/ProjectContext';
import type { WindowActions } from '../../src/renderer/window';
import { emptySessions } from '../../src/state/sessions';

// Memory and skills from the Grade (D53): the Crew's own sections, opened over
// the grid for the project of the conversation they were asked from.

const never = new Promise<never>(() => {});
const actions = { listMemory: () => never, listSkills: () => never } as unknown as WindowActions;
const places = [
  { id: '/w/dcode', label: 'dcode' },
  { id: '/w/eva', label: 'eva' },
];

const html = (tab: 'memoria' | 'skills') =>
  renderToStaticMarkup(<ProjectContextBody tab={tab} onTab={() => {}} places={places} preferred="/w/eva" sessions={emptySessions} actions={actions} />);

describe('memory and skills over the grid', () => {
  it('offers both, with the one shown marked', () => {
    const memory = html('memoria');
    expect(memory).toContain('aria-selected="true"');
    expect(memory).toMatch(/role="tab"[^>]*aria-selected="true"[^>]*>Memória/);
    expect(memory).toMatch(/role="tab"[^>]*aria-selected="false"[^>]*>Skills/);
  });

  it('draws the Crew’s own section, asking about the conversation’s project first', () => {
    const memory = html('memoria');
    expect(memory).toContain('<h1>Memória</h1>');
    expect(memory).toMatch(/<option value="\/w\/eva" title="\/w\/eva" selected="">/);
    expect(memory).toContain('Perguntando ao daemon…');
    expect(html('skills')).toContain('<h1>Skills</h1>');
  });
});
