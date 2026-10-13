// The model's text, drawn as the markdown it is written in (D34).
//
// `marked` only tokenises: its lexer turns the text into a tree, and this file
// maps the tree to React elements. Nothing here produces an HTML string, so
// the HTML a model writes is never parsed by the page — it is shown as the text
// it is, and React escapes it like any other string.

import { Lexer, type Token, type Tokens } from 'marked';
import { useMemo, type ReactNode } from 'react';

// `breaks`: a single newline in a reply is a line break, as the text showed
// before markdown was drawn; models write lines that way.
const OPTIONS = { gfm: true, breaks: true } as const;

/** Only a web address leaves the app, and the main process opens it in the browser (D2). */
function safeHref(href: string): string | null {
  if (!/^https:\/\//i.test(href)) return null;
  try {
    return new URL(href).protocol === 'https:' ? href : null;
  } catch {
    return null;
  }
}

const NAMED: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'", nbsp: ' ' };

/** The lexer leaves entities as written; React would escape them a second time. */
function decode(s: string): string {
  return s.replace(/&(#\d{1,7}|#x[0-9a-f]{1,6}|[a-z]+);/gi, (whole, name: string) => {
    if (name[0] !== '#') return NAMED[name.toLowerCase()] ?? whole;
    const code = name[1] === 'x' || name[1] === 'X' ? parseInt(name.slice(2), 16) : parseInt(name.slice(1), 10);
    return code > 0 && code <= 0x10ffff ? String.fromCodePoint(code) : whole;
  });
}

function inline(tokens: Token[] | undefined): ReactNode[] {
  return (tokens ?? []).map((t, i) => inlineToken(t, i));
}

function inlineToken(t: Token, key: number): ReactNode {
  switch (t.type) {
    case 'text': {
      const tt = t as Tokens.Text;
      return tt.tokens ? <span key={key}>{inline(tt.tokens)}</span> : decode(tt.text);
    }
    case 'escape':
      return (t as Tokens.Escape).text;
    case 'strong':
      return <strong key={key}>{inline((t as Tokens.Strong).tokens)}</strong>;
    case 'em':
      return <em key={key}>{inline((t as Tokens.Em).tokens)}</em>;
    case 'del':
      return <del key={key}>{inline((t as Tokens.Del).tokens)}</del>;
    case 'codespan':
      return (
        <code key={key} className="inline-code">
          {(t as Tokens.Codespan).text}
        </code>
      );
    case 'br':
      return <br key={key} />;
    case 'link': {
      const l = t as Tokens.Link;
      const href = safeHref(l.href);
      if (!href) {
        return (
          <span key={key} className="md-link-text" title={l.href}>
            {inline(l.tokens)}
          </span>
        );
      }
      return (
        <a key={key} className="md-link" href={href} target="_blank" rel="noreferrer noopener" title={href}>
          {inline(l.tokens)}
        </a>
      );
    }
    case 'image': {
      // Never fetched: the page loads nothing from where the model pointed.
      const img = t as Tokens.Image;
      return (
        <span key={key} className="md-link-text" title={img.href}>
          {decode(img.text)}
        </span>
      );
    }
    case 'checkbox':
      return (t as Tokens.Checkbox).checked ? '☑ ' : '☐ ';
    default:
      // Inline HTML, and anything this file does not know: the text as written.
      return t.raw;
  }
}

function Block({ t }: { t: Token }): ReactNode {
  switch (t.type) {
    case 'space':
    case 'def':
      return null;
    case 'paragraph':
      return <p className="rich-p">{inline((t as Tokens.Paragraph).tokens)}</p>;
    case 'text': {
      const tt = t as Tokens.Text;
      return <>{tt.tokens ? inline(tt.tokens) : decode(tt.text)}</>;
    }
    case 'heading': {
      const h = t as Tokens.Heading;
      const depth = Math.min(Math.max(h.depth, 1), 4);
      const Tag = `h${depth}` as 'h1' | 'h2' | 'h3' | 'h4';
      return <Tag className={`md-h md-h${depth}`}>{inline(h.tokens)}</Tag>;
    }
    case 'code': {
      const c = t as Tokens.Code;
      return (
        <pre className="md-pre">
          <code data-lang={c.lang || undefined}>{c.text}</code>
        </pre>
      );
    }
    case 'blockquote':
      return (
        <blockquote className="md-quote">
          <Blocks tokens={(t as Tokens.Blockquote).tokens} />
        </blockquote>
      );
    case 'hr':
      return <hr className="md-hr" />;
    case 'list': {
      const l = t as Tokens.List;
      const items = l.items.map((it, i) => (
        <li key={i}>
          <Blocks tokens={it.tokens} />
        </li>
      ));
      if (!l.ordered) return <ul className="md-list">{items}</ul>;
      const start = typeof l.start === 'number' && l.start !== 1 ? l.start : undefined;
      return (
        <ol className="md-list" start={start}>
          {items}
        </ol>
      );
    }
    case 'table': {
      const tb = t as Tokens.Table;
      const align = (a: Tokens.TableCell['align']) => (a ? { textAlign: a } : undefined);
      return (
        <div className="md-table-wrap">
          <table className="md-table">
            <thead>
              <tr>
                {tb.header.map((c, i) => (
                  <th key={i} style={align(c.align)}>
                    {inline(c.tokens)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {tb.rows.map((row, r) => (
                <tr key={r}>
                  {row.map((c, i) => (
                    <td key={i} style={align(c.align)}>
                      {inline(c.tokens)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    }
    default:
      // Block HTML, and anything this file does not know: the text as written.
      return <p className="rich-p">{t.raw.replace(/\n+$/, '')}</p>;
  }
}

function Blocks({ tokens }: { tokens: Token[] }) {
  return (
    <>
      {tokens.map((t, i) => (
        <Block key={i} t={t} />
      ))}
    </>
  );
}

/**
 * Model text. It arrives in fragments, so every prefix is drawn as what it is
 * so far: an unclosed `**` stays as typed until its partner arrives, and an
 * unclosed fence is code up to where the text has reached.
 */
export function RichText({ text }: { text: string }) {
  const tokens = useMemo(() => new Lexer(OPTIONS).lex(text), [text]);
  return <Blocks tokens={tokens} />;
}
