import { Fragment, type ReactNode } from 'react';
import { delegation, toolLine, VERBS, WORKING, type Activity, type Block, type Tone } from '../state/flow';
import { elapsed, since } from '../state/format';
import type { Entry, ToolCall } from '../state/session';
import { ApprovalCard } from './ApprovalCard';
import { noteText } from './text';

/** Model text: paragraphs, and `code` between backticks. An unpaired backtick stays text. */
export function RichText({ text }: { text: string }) {
  const paragraphs = text.split(/\n{2,}/);
  return (
    <>
      {paragraphs.map((p, i) => {
        const parts = p.split('`');
        // An even count of parts means one backtick has no partner: the last.
        const stray = parts.length % 2 === 0 ? parts.length - 1 : -1;
        return (
          <p key={i} className="rich-p">
            {parts.map((part, j) =>
              j === stray ? (
                <Fragment key={j}>{`\`${part}`}</Fragment>
              ) : j % 2 === 1 ? (
                <code key={j} className="inline-code">
                  {part}
                </code>
              ) : (
                <Fragment key={j}>{part}</Fragment>
              ),
            )}
          </p>
        );
      })}
    </>
  );
}

function ToolLine({ call }: { call: ToolCall }) {
  const v = toolLine(call);
  return (
    <div className="tool-line" title={call.result && !call.result.ok ? call.result.output : undefined}>
      <span className={`tool-glyph tone-${v.glyphTone}${call.result ? '' : ' dc-breath'}`}>{v.glyph}</span>
      <span className="tool-name">{v.name}</span>
      {v.target && <span className="tool-target">{v.target}</span>}
      <span className="spacer" />
      <span className={`tool-summary tone-${v.summaryTone}`}>{v.summary}</span>
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
}: {
  activity: Activity;
  startedAt: string;
  now: number;
  tick: number;
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
    </div>
  );
}

const toneClass = (t: Tone) => `tone-${t}`;

function EntryView({ entry, now, onAnswer }: { entry: Exclude<Entry, { kind: 'tool' }>; now: number; onAnswer: (d: string) => void }) {
  switch (entry.kind) {
    case 'user':
      return <div className={`bubble${entry.steer ? ' steer' : ''}`}>{entry.text}</div>;
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
      return <ApprovalCard entry={entry} now={now} onAnswer={onAnswer} />;
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

export function FlowBlocks({ blocks, now, onAnswer }: { blocks: Block[]; now: number; onAnswer: (d: string) => void }) {
  return (
    <>
      {blocks.map((b, i) => {
        if (b.kind === 'tools') {
          return (
            <div key={`t-${b.calls[0]?.id}`} className="tool-group">
              {b.calls.map((c) => (
                <ToolLine key={c.id} call={c} />
              ))}
            </div>
          );
        }
        if (b.kind === 'delegation') return <Delegation key={`d-${b.calls[0]?.id}`} calls={b.calls} />;
        return <EntryView key={`e-${i}-${b.entry.kind}`} entry={b.entry} now={now} onAnswer={onAnswer} />;
      })}
    </>
  );
}
