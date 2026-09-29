import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { RichText } from '../../src/renderer/Flow';

const html = (text: string) => renderToStaticMarkup(<RichText text={text} />);

describe('model text', () => {
  it('draws code between backticks, and leaves a backtick with no partner as text', () => {
    expect(html('fora de `src/`.')).toBe('<p class="rich-p">fora de <code class="inline-code">src/</code>.</p>');
    expect(html('a `b` c `d')).toBe('<p class="rich-p">a <code class="inline-code">b</code> c `d</p>');
  });

  it('keeps paragraphs apart', () => {
    expect(html('um\n\ndois')).toBe('<p class="rich-p">um</p><p class="rich-p">dois</p>');
  });
});
