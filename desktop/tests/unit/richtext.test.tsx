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

  it('draws bold and italic, not the asterisks', () => {
    expect(html('**Resumo** e *nota* e _outra_')).toBe(
      '<p class="rich-p"><strong>Resumo</strong> e <em>nota</em> e <em>outra</em></p>',
    );
  });

  it('draws a bullet list, and a list inside a list', () => {
    const out = html('- um\n- dois\n  - dois.a\n  - dois.b\n- três');
    expect(out).toBe(
      '<ul class="md-list"><li>um</li><li>dois<ul class="md-list"><li>dois.a</li><li>dois.b</li></ul></li><li>três</li></ul>',
    );
    expect(out).not.toContain('- ');
  });

  it('draws a numbered list from the number it starts at', () => {
    expect(html('1. a\n2. b')).toBe('<ol class="md-list"><li>a</li><li>b</li></ol>');
    expect(html('3. c\n4. d')).toBe('<ol class="md-list" start="3"><li>c</li><li>d</li></ol>');
  });

  it('draws headings, the bold heading of a reply, and keeps them modest', () => {
    expect(html('## Plano\n\ntexto')).toBe('<h2 class="md-h md-h2">Plano</h2><p class="rich-p">texto</p>');
    expect(html('#### Detalhe')).toBe('<h4 class="md-h md-h4">Detalhe</h4>');
    // Levels past four are drawn as four: they live inside a chat flow.
    expect(html('###### fundo')).toBe('<h4 class="md-h md-h4">fundo</h4>');
  });

  it('draws a fenced block as code, with its text as it was', () => {
    expect(html('```go\nif a < b && c {\n}\n```')).toBe(
      '<pre class="md-pre"><code data-lang="go">if a &lt; b &amp;&amp; c {\n}</code></pre>',
    );
  });

  it('draws a quote, a rule and a table', () => {
    expect(html('> citado')).toBe('<blockquote class="md-quote"><p class="rich-p">citado</p></blockquote>');
    expect(html('a\n\n---\n\nb')).toContain('<hr class="md-hr"/>');
    const table = html('| a | b |\n|---|--:|\n| 1 | 2 |');
    expect(table).toContain('<table class="md-table">');
    expect(table).toContain('<th>a</th>');
    expect(table).toContain('<td style="text-align:right">2</td>');
  });

  it('opens a web link outside the app, and leaves any other link as text', () => {
    expect(html('[docs](https://example.com/x)')).toBe(
      '<p class="rich-p"><a class="md-link" href="https://example.com/x" target="_blank" rel="noreferrer noopener" title="https://example.com/x">docs</a></p>',
    );
    for (const url of ['javascript:alert(1)', 'file:///etc/passwd', 'http://example.com', '/relative']) {
      const out = html(`[x](${url})`);
      expect(out, url).not.toContain('<a');
      expect(out, url).toContain('class="md-link-text"');
    }
  });

  it('never injects the HTML the model wrote: it is shown as text', () => {
    const out = html('<script>alert(1)</script>\n\nantes <img src=x onerror="alert(1)"> depois <b>b</b>');
    expect(out).not.toMatch(/<script|<img|<b>/);
    expect(out).toContain('&lt;script&gt;');
    expect(out).toContain('&lt;img src=x onerror=&quot;alert(1)&quot;&gt;');
    // An image is its words: nothing is fetched from where the model pointed.
    expect(html('![logo](https://example.com/a.png)')).not.toContain('<img');
  });

  it('shows an entity as the character it names, once', () => {
    const out = html('a &amp; b &lt; c');
    expect(out).toContain('a &amp; b &lt; c');
    expect(out).not.toContain('&amp;amp;');
  });

  it('draws text that is still arriving sanely, and right once it is whole', () => {
    // An unclosed ** stays as it came until its partner arrives.
    expect(html('Isto é **impor')).toBe('<p class="rich-p">Isto é **impor</p>');
    expect(html('Isto é **importante**')).toBe('<p class="rich-p">Isto é <strong>importante</strong></p>');
    // An unclosed fence is code up to where the text has arrived.
    expect(html('veja:\n\n```ts\nconst a = 1')).toBe(
      '<p class="rich-p">veja:</p><pre class="md-pre"><code data-lang="ts">const a = 1</code></pre>',
    );
    // Every prefix of a reply renders, as it would while streaming.
    const reply = '## Plano\n\n1. **Ler** o `main.go`\n   - ver *flags*\n2. Testar\n\n```sh\ngo test ./...\n```\n\n> pronto | a |\n';
    for (let i = 0; i <= reply.length; i++) expect(() => html(reply.slice(0, i))).not.toThrow();
  });
});
