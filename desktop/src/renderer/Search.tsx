import { useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react';
import { lastEventText, longState, SCOPES, searchGroups, searchSide, type Scope } from '../state/search';
import { activeRows, rowMark, type ProjectView, type Row } from '../state/sidebar';

function Preview({ row, now }: { row: Row | null; now: number }) {
  if (!row) return <div className="search-preview" />;
  const state = longState(row);
  const facts: [string, string, boolean][] = [
    ['workspace', row.workspace, true],
    ['branch', row.branch ?? '', true],
    ['modelo', row.model ?? '', false],
    ['turnos', row.turns !== undefined ? String(row.turns) : '', false],
  ];
  const last = lastEventText(row, now);
  return (
    <div className="search-preview">
      <div className="search-preview-title">{row.title}</div>
      <div className={`search-preview-state tone-${state.tone}`}>{state.text}</div>
      <dl className="search-facts">
        {facts
          .filter(([, value]) => value !== '')
          .map(([label, value, mono]) => (
            <div key={label} className="search-fact">
              <dt>{label}</dt>
              <dd className={mono ? 'mono' : undefined}>{value}</dd>
            </div>
          ))}
      </dl>
      {last && <div className="search-preview-last">{last}</div>}
    </div>
  );
}

/**
 * ⌘K: every conversation, live or ended, found by its title and its project.
 * ↵ opens the one under the cursor by the same rules as a click on its row —
 * an ended one is continued in a new session.
 */
export function Search({
  projects,
  project,
  now,
  onOpen,
  onClose,
}: {
  projects: ProjectView[];
  /** The open session's workspace, which the `projeto` scope keeps to; null with none open. */
  project: string | null;
  now: number;
  onOpen: (rowId: string) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState('');
  const [scope, setScope] = useState<Scope>('todas');
  const [cursor, setCursor] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  useLayoutEffect(() => input.current?.focus(), []);

  const scopes = useMemo(() => SCOPES.filter((s) => s !== 'projeto' || project !== null), [project]);
  const groups = useMemo(() => searchGroups(projects, query, scope, project), [projects, query, scope, project]);
  const flat = useMemo(() => groups.flatMap((g) => g.rows), [groups]);
  const keys = useMemo(() => activeRows(projects), [projects]);
  const at = Math.min(cursor, Math.max(flat.length - 1, 0));
  const current = flat[at] ?? null;
  const projectLabel = projects.find((p) => p.id === project)?.label ?? '';

  const move = (to: number) => setCursor(Math.max(0, Math.min(to, flat.length - 1)));
  const onKey = (e: KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        move(at + 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        move(at - 1);
        break;
      case 'Tab': {
        e.preventDefault();
        const i = scopes.indexOf(scope);
        setScope(scopes[(i + (e.shiftKey ? scopes.length - 1 : 1)) % scopes.length] ?? 'todas');
        setCursor(0);
        break;
      }
      case 'Enter':
        e.preventDefault();
        if (current) onOpen(current.id);
        break;
      case 'Escape':
        // Clears what was typed first; with nothing typed, closes.
        e.preventDefault();
        if (query) {
          setQuery('');
          setCursor(0);
        } else onClose();
        break;
      default:
    }
  };

  return (
    <div
      className="search-veil"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="search" role="dialog" aria-modal="true" aria-label="Ir para sessão" onKeyDown={onKey}>
        <div className="search-top">
          <span className="search-glyph">⌕</span>
          <input
            ref={input}
            className="search-input"
            placeholder="Ir para sessão…"
            aria-label="Ir para sessão"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setCursor(0);
            }}
          />
          <div className="search-scopes">
            {scopes.map((s) => (
              <button
                key={s}
                type="button"
                tabIndex={-1}
                className={`search-scope${s === scope ? ' active' : ''}`}
                onClick={() => {
                  setScope(s);
                  setCursor(0);
                  input.current?.focus();
                }}
              >
                {s === 'projeto' ? projectLabel : s}
              </button>
            ))}
          </div>
        </div>
        <div className="search-body">
          <div className="search-list" role="listbox" aria-label="Conversas">
            {flat.length === 0 ? (
              <p className="search-empty">{query ? `Nenhuma conversa com “${query}”.` : 'Nenhuma conversa aqui.'}</p>
            ) : (
              groups.map((g) => (
                <div key={g.label} className="search-group">
                  <div className="search-group-head">
                    <span>{g.label}</span>
                    <span>{g.rows.length}</span>
                  </div>
                  {g.rows.map((r) => {
                    const i = flat.indexOf(r);
                    const side = searchSide(r, now);
                    const key = keys.indexOf(r);
                    const mark = rowMark(r);
                    return (
                      <div
                        key={r.id}
                        role="option"
                        aria-selected={i === at}
                        className={`search-row${i === at ? ' selected' : ''}`}
                        onMouseEnter={() => setCursor(i)}
                        onClick={() => onOpen(r.id)}
                      >
                        <span className="mark-col">
                          <span className={`mark mark-${mark}`} />
                        </span>
                        <span className="search-title">{r.title}</span>
                        <span className={`search-side tone-${side.tone}`}>{side.text}</span>
                        {key >= 0 && key < 9 && <span className="search-key">⌘{key + 1}</span>}
                      </div>
                    );
                  })}
                </div>
              ))
            )}
          </div>
          <Preview row={current} now={now} />
        </div>
        <div className="search-foot">
          <span>↑↓ navegar</span>
          <span>↵ abrir</span>
          <span>tab escopo</span>
          <span className="spacer" />
          <span>esc fechar</span>
        </div>
      </div>
    </div>
  );
}
