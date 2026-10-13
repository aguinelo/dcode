import { choicesFor } from '../state/approval';
import { boundaryLabel, decisionLabel } from '../state/flow';
import { clock, since } from '../state/format';
import type { Entry } from '../state/session';
import { CommandActions } from './CommandActions';

type ApprovalEntry = Extract<Entry, { kind: 'approval' }>;

export function ApprovalCard({
  entry,
  now,
  onAnswer,
  onRun,
}: {
  entry: ApprovalEntry;
  now: number;
  onAnswer: (decision: string) => void;
  /** Runs the command in the conversation's terminal instead, by the person's own hand (D37). */
  onRun?: (command: string) => void;
}) {
  const r = entry.request;
  const answered = entry.decision !== null ? decisionLabel(entry.decision) : null;
  return (
    <div
      className={`approval${answered ? ' answered' : ''}`}
      data-approval-id={r.approval_id}
      data-decision={entry.decision ?? undefined}
    >
      <div className="approval-head">
        <span className="diamond approval-diamond" />
        <span className="approval-title">{r.command ? 'Rodar este comando?' : `Permitir ${r.tool}?`}</span>
        <span className="spacer" />
        {answered ? (
          <span className={`approval-answer tone-${answered.tone}`}>{answered.text}</span>
        ) : (
          // Counted from the instant the question was put — the envelope's
          // `at`, not the request's expires_at (D11).
          <span className="approval-wait">esperando · {clock(since(entry.at, now))}</span>
        )}
      </div>
      {r.command && (
        <div className="approval-command">
          <span className="approval-prompt">$</span> {r.command}
          <CommandActions command={r.command} onRun={onRun} />
        </div>
      )}
      <div className="approval-reason">
        Pede <span className="approval-resource">{boundaryLabel(r.boundary_crossed)}</span>
        {r.rule ? (
          <>
            {' · regra '}
            <span className="approval-rule">{r.rule}</span>
          </>
        ) : null}
        .{r.reason ? <span className="approval-why">{r.reason}</span> : null}
      </div>
      {!answered && (
        <div className="approval-options" role="group" aria-label="Responder">
          {choicesFor(r).map((c) => (
            <button
              key={c.key}
              type="button"
              className={`approval-option${c.decision === 'deny' ? ' preselected' : ''}${c.standing ? ' standing' : ''}`}
              title={c.standing ? 'Fica gravado na sua configuração, fora do projeto, e vale para as próximas sessões.' : undefined}
              onClick={() => onAnswer(c.decision)}
            >
              <span className="approval-key">{c.key}</span>
              <span className="approval-label">{c.label}</span>
              <span className="approval-hint">{c.hint}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
