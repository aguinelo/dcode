import { useMemo, useState, type KeyboardEvent, type MouseEvent, type ReactNode } from 'react';
import { delegation, toolLine, VERBS, WORKING, type Activity, type Block, type Tone } from '../state/flow';
import { elapsed, since } from '../state/format';
import type { ImageAttachment } from '../shared/api';
import { imageSource, sizeOf } from '../state/attachments';
import type { Entry, ToolCall } from '../state/session';
import { toolDetail } from '../state/tooldetail';
import { ApprovalCard } from './ApprovalCard';
import { CommandActions } from './CommandActions';
import { RichText } from './Markdown';
import { noteText } from './text';
import { ToolDetail } from './ToolDetail';

export { RichText };

/** The command of a shell call, as the agent wrote it; empty for any other call. */
function shellCommand(call: ToolCall): string {
  if (call.name.toLowerCase() !== 'bash') return '';
  const c = (call.input as { command?: unknown } | null)?.command;
  return typeof c === 'string' ? c : '';
}

function ToolLine({ call, onRun }: { call: ToolCall; onRun?: (command: string) => void }) {
  const [open, setOpen] = useState(false);
  const v = toolLine(call);
  const command = shellCommand(call);
  // A command the person typed (`!`, D40) is shown open: they ran it to read what it printed.
  const output = call.typed && call.result ? call.result.output : '';
  // A model's call opens onto what it did (D45), closed until asked: the line is the summary.
  const detail = useMemo(() => toolDetail(call), [call]);
  const detailId = `tool-detail-${call.id}`;
  const toggle = () => setOpen((o) => !o);
  const opener = detail
    ? {
        role: 'button',
        tabIndex: 0,
        'aria-expanded': open,
        'aria-controls': open ? detailId : undefined,
        onClick: (e: MouseEvent<HTMLDivElement>) => {
          // Copiar and Rodar no terminal sit on the line; pressing them is not opening it.
          if ((e.target as HTMLElement).closest('button')) return;
          toggle();
        },
        onKeyDown: (e: KeyboardEvent<HTMLDivElement>) => {
          if (e.target !== e.currentTarget || (e.key !== 'Enter' && e.key !== ' ')) return;
          e.preventDefault();
          toggle();
        },
      }
    : {};
  const line = (
    <div
      className={`tool-line${detail ? ' opens' : ''}`}
      title={call.result && !call.result.ok && !output ? call.result.output : undefined}
      {...opener}
    >
      <span className={`tool-glyph tone-${v.glyphTone}${call.result ? '' : ' dc-breath'}`}>{v.glyph}</span>
      <span className="tool-name">{v.name}</span>
      {v.target && <span className="tool-target">{v.target}</span>}
      <span className="spacer" />
      <span className={`tool-summary tone-${v.summaryTone}`}>{v.summary}</span>
      {command && <CommandActions command={command} onRun={onRun} />}
    </div>
  );
  if (call.typed) {
    return (
      <div className="tool-typed" data-typed-call={call.id}>
        {line}
        {output && <pre className={`tool-output${call.result?.ok ? '' : ' failed'}`}>{output}</pre>}
      </div>
    );
  }
  return (
    <div className={`tool-call${open ? ' open' : ''}`} data-tool-call={call.id}>
      {line}
      {open && detail && <ToolDetail id={detailId} detail={detail} label={`O que ${v.name} fez`} />}
    </div>
  );
}

function Delegation({ calls }: { calls: ToolCall[] }) {
  const d = delegation(calls);
  const readOnly = calls.every((c) => {
    const owns = (c.input as { owns?: unknown } | null)?.owns;
    return !Array.isArray(owns) || owns.length === 0;
  });
  return (
    <div className="delegation">
      <div className="delegation-head">
        <span className={`delegation-dot${d.running > 0 ? ' running dc-breath' : ''}`} />
        <span className="delegation-title">Delegou a {d.count} filhos</span>
        {d.disjoint && <span className="delegation-sub">propriedade disjunta</span>}
        {!d.disjoint && readOnly && <span className="delegation-sub">só leitura</span>}
        <span className="spacer" />
        <span className="delegation-count">
          {d.finished} de {d.count}
        </span>
      </div>
      <div className="delegation-rows">
        {d.children.map((c) => (
          <div key={c.id} className="delegation-row" title={c.reason || undefined}>
            <span className={`child-glyph tone-${c.status === 'ok' ? 'ok' : c.status === 'failed' ? 'err' : 'accent'}`}>
              {c.status === 'ok' ? '✓' : c.status === 'failed' ? '⊘' : '●'}
            </span>
            <span className="child-name">{c.name}</span>
            <span className="child-owns">{c.owns}</span>
            <span className="child-bar">
              <span
                className={`child-bar-fill dc-bar-fill tone-bg-${c.status === 'ok' ? 'ok' : c.status === 'failed' ? 'err' : 'accent'}`}
                style={{ width: c.bar === null ? '0%' : `${Math.round(c.bar * 100)}%` }}
              />
            </span>
            <span className={`child-meta tone-${c.status === 'failed' ? 'err' : 'dim'}`}>{c.meta}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function joinNames(names: string[]): string {
  if (names.length <= 1) return names[0] ?? '';
  return `${names.slice(0, -1).join(', ')} e ${names[names.length - 1]}`;
}

export function ActivityLine({
  activity,
  startedAt,
  now,
  tick,
  touched,
}: {
  activity: Activity;
  startedAt: string;
  now: number;
  tick: number;
  /** What the turn touched, on the right of the elapsed time (D58). */
  touched?: ReactNode;
}) {
  const verbs = activity.phase ? VERBS[activity.phase] : null;
  // The verb never appears alone: with nothing running it is the plain word.
  const word = verbs ? (verbs[tick % verbs.length] ?? WORKING) : WORKING;
  const f = activity.fact;
  let fact: ReactNode = null;
  if (f.kind === 'children') fact = `${joinNames(f.names)} ainda rodando`;
  if (f.kind === 'tool')
    fact = (
      <>
        {f.name} <span className="mono-12">{f.target}</span>
      </>
    );
  return (
    <div className="activity">
      <span className="activity-dot dc-breath">●</span>
      <span className="activity-verb">{word.charAt(0).toUpperCase() + word.slice(1)}…</span>
      <span className="activity-fact">{fact}</span>
      <span className="activity-meta">{elapsed(since(startedAt, now))}</span>
      {touched}
    </div>
  );
}

const toneClass = (t: Tone) => `tone-${t}`;

function EntryView({
  entry,
  now,
  onAnswer,
  onRun,
  images,
}: {
  entry: Exclude<Entry, { kind: 'tool' }>;
  now: number;
  onAnswer: (d: string) => void;
  onRun?: (command: string) => void;
  /** The pictures this window sent with the message (D46), when it is one. */
  images?: ImageAttachment[];
}) {
  switch (entry.kind) {
    case 'user': {
      const bubble = <div className={`bubble${entry.steer ? ' steer' : ''}`}>{entry.text}</div>;
      if (!images || images.length === 0) return bubble;
      return (
        <div className="bubble-with-images">
          <div className="bubble-images">
            {images.map((img, i) => (
              <img key={`${i}-${img.name}`} className="bubble-image" src={imageSource(img)} alt={img.name} title={`${img.name} · ${sizeOf(img.bytes)}`} />
            ))}
          </div>
          {bubble}
        </div>
      );
    }
    case 'model':
      return (
        <div className="model-text">
          <RichText text={entry.text} />
        </div>
      );
    case 'reasoning':
      return (
        <details className="reasoning">
          <summary>Raciocínio</summary>
          <div className="reasoning-text">{entry.text}</div>
        </details>
      );
    case 'approval':
      return <ApprovalCard entry={entry} now={now} onAnswer={onAnswer} onRun={onRun} />;
    case 'plan':
      return (
        <div className="plan">
          {entry.items.map((it) => (
            <div key={it.id} className={`plan-item status-${it.status}`}>
              <span className="plan-glyph">{it.status === 'done' ? '✓' : it.status === 'active' ? '●' : it.status === 'blocked' ? '⊘' : '·'}</span>
              <span className="plan-text">{it.text}</span>
              {it.blocked && <span className="plan-blocked">{it.blocked}</span>}
            </div>
          ))}
        </div>
      );
    case 'note': {
      const n = noteText(entry.note);
      return n.text ? <div className={`note ${toneClass(n.tone)}`}>{n.text}</div> : null;
    }
    case 'error':
      return (
        <div className="error-entry">
          <span className="tone-err">Erro na sessão</span> {entry.message}
          {entry.code && <span className="error-code"> · {entry.code}</span>}
        </div>
      );
  }
}

export function FlowBlocks({
  blocks,
  now,
  onAnswer,
  onRun,
  imagesAt,
}: {
  blocks: Block[];
  now: number;
  onAnswer: (d: string) => void;
  /** Opens the conversation's terminal with a command the agent proposed typed in (D37). */
  onRun?: (command: string) => void;
  /** The pictures sent with each message, by its seq (D46). */
  imagesAt?: ReadonlyMap<number, ImageAttachment[]>;
}) {
  return (
    <>
      {blocks.map((b, i) => {
        if (b.kind === 'tools') {
          return (
            <div key={`t-${b.calls[0]?.id}`} className="tool-group">
              {b.calls.map((c) => (
                <ToolLine key={c.id} call={c} onRun={onRun} />
              ))}
            </div>
          );
        }
        if (b.kind === 'delegation') return <Delegation key={`d-${b.calls[0]?.id}`} calls={b.calls} />;
        return (
          <EntryView
            key={`e-${i}-${b.entry.kind}`}
            entry={b.entry}
            now={now}
            onAnswer={onAnswer}
            onRun={onRun}
            images={b.entry.kind === 'user' ? imagesAt?.get(b.entry.seq) : undefined}
          />
        );
      })}
    </>
  );
}
