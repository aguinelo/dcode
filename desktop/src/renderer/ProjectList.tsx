import { useEffect, useRef, useState } from 'react';
import { rowMark, rowSide, type ProjectView } from '../state/sidebar';

export interface ProjectActions {
  toggle(id: string): void;
  rename(id: string, label: string): void;
  move(id: string, to: number): void;
  dropBefore(id: string, target: string): void;
  select(sessionId: string): void;
}

function RenameInput({ initial, onDone }: { initial: string; onDone: (label: string | null) => void }) {
  const [draft, setDraft] = useState(initial);
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    ref.current?.focus();
    ref.current?.select();
  }, []);
  return (
    <input
      ref={ref}
      className="project-rename"
      value={draft}
      aria-label="Novo nome do projeto"
      onChange={(e) => setDraft(e.target.value)}
      onBlur={() => onDone(draft)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') onDone(draft);
        if (e.key === 'Escape') onDone(null);
      }}
    />
  );
}

function Project({
  project,
  index,
  total,
  selectedId,
  onGridIds,
  now,
  dragging,
  dropTarget,
  setDragging,
  setDropTarget,
  actions,
}: {
  project: ProjectView;
  index: number;
  total: number;
  selectedId: string | null;
  onGridIds: ReadonlySet<string>;
  now: number;
  dragging: string | null;
  dropTarget: string | null;
  setDragging: (id: string | null) => void;
  setDropTarget: (id: string | null) => void;
  actions: ProjectActions;
}) {
  const [menu, setMenu] = useState(false);
  const [editing, setEditing] = useState(false);
  const p = project;
  const menuItems = [
    { label: 'Renomear', hint: '2× clique', run: () => setEditing(true) },
    { label: p.collapsed ? 'Expandir sessões' : 'Recolher sessões', hint: '', run: () => actions.toggle(p.id) },
    ...(index > 0 ? [{ label: 'Mover para cima', hint: '', run: () => actions.move(p.id, index - 1) }] : []),
    ...(index < total - 1 ? [{ label: 'Mover para baixo', hint: '', run: () => actions.move(p.id, index + 1) }] : []),
  ];
  return (
    <div
      className={`project${dropTarget === p.id && dragging !== p.id ? ' drop-target' : ''}${dragging === p.id ? ' dragging' : ''}${menu ? ' menu-open' : ''}`}
      draggable={!editing}
      onDragStart={(e) => {
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', p.id);
        setDragging(p.id);
        setMenu(false);
      }}
      onDragOver={(e) => {
        e.preventDefault();
        if (dropTarget !== p.id) setDropTarget(p.id);
      }}
      onDrop={(e) => {
        e.preventDefault();
        if (dragging && dragging !== p.id) actions.dropBefore(dragging, p.id);
        setDragging(null);
        setDropTarget(null);
      }}
      onDragEnd={() => {
        setDragging(null);
        setDropTarget(null);
      }}
      onMouseLeave={() => setMenu(false)}
    >
      {editing ? (
        <div className="project-row editing">
          <span className="chevron">{p.collapsed ? '▸' : '▾'}</span>
          <RenameInput
            initial={p.label}
            onDone={(label) => {
              setEditing(false);
              if (label !== null) actions.rename(p.id, label);
            }}
          />
        </div>
      ) : (
        <div
          className="project-row"
          role="button"
          tabIndex={0}
          aria-expanded={!p.collapsed}
          onClick={() => actions.toggle(p.id)}
          onDoubleClick={(e) => {
            e.stopPropagation();
            setEditing(true);
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') actions.toggle(p.id);
          }}
        >
          <span className="chevron">{p.collapsed ? '▸' : '▾'}</span>
          <span className="project-name" title={p.id}>
            {p.label}
          </span>
          {p.meta && <span className={`project-meta tone-${p.meta.tone}`}>{p.meta.text}</span>}
          <span
            className="project-dots"
            role="button"
            aria-label={`Ações de ${p.label}`}
            onClick={(e) => {
              e.stopPropagation();
              setMenu(!menu);
            }}
          >
            ⋯
          </span>
        </div>
      )}
      {menu && (
        <div className="popover project-menu" role="menu">
          {menuItems.map((m) => (
            <div
              key={m.label}
              className="popover-item"
              role="menuitem"
              onClick={(e) => {
                e.stopPropagation();
                setMenu(false);
                m.run();
              }}
            >
              <span className="popover-label">{m.label}</span>
              <span className="popover-hint">{m.hint}</span>
            </div>
          ))}
        </div>
      )}
      {!p.collapsed &&
        p.rows.map((r) => {
          const mark = rowMark(r);
          const side = rowSide(r, now);
          const selected = r.id === selectedId;
          return (
            <div
              key={r.id}
              className={`session-row${selected ? ' selected' : ''}${mark === 'none' ? ' idle' : ''}`}
              role="button"
              tabIndex={0}
              data-session-id={r.id}
              data-state={r.state}
              aria-current={selected ? 'true' : undefined}
              onClick={() => actions.select(r.id)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') actions.select(r.id);
              }}
            >
              <span className="mark-col">
                <span className={`mark mark-${mark}${mark === 'running' ? ' dc-breath' : ''}`} />
              </span>
              <span className="session-title">{r.title}</span>
              {onGridIds.has(r.id) && <span className="mark-grid" title="Na grade" />}
              {side && <span className={`session-side tone-${side.tone}`}>{side.text}</span>}
            </div>
          );
        })}
    </div>
  );
}

export function ProjectList({
  projects,
  selectedId,
  onGridIds,
  now,
  actions,
}: {
  projects: ProjectView[];
  selectedId: string | null;
  onGridIds: ReadonlySet<string>;
  now: number;
  actions: ProjectActions;
}) {
  const [dragging, setDragging] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  return (
    <>
      {projects.map((p, i) => (
        <Project
          key={p.id}
          project={p}
          index={i}
          total={projects.length}
          selectedId={selectedId}
          onGridIds={onGridIds}
          now={now}
          dragging={dragging}
          dropTarget={dropTarget}
          setDragging={setDragging}
          setDropTarget={setDropTarget}
          actions={actions}
        />
      ))}
    </>
  );
}
