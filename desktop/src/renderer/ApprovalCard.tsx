import { boundaryLabel, decisionLabel } from '../state/flow';
import { clock, since } from '../state/format';
import type { Entry } from '../state/session';

type ApprovalEntry = Extract<Entry, { kind: 'approval' }>;

/**
 * The three answers the design offers, and the keys that give them. Allowing
 * is always an explicit choice; `↵` and `esc` deny, as in the TUI, and deny is
 * the option shown selected (D22).
 */
export const CHOICES: ReadonlyArray<{ key: string; label: string; decision: string; hint: string }> = [
  { key: '1', label: 'Permitir uma vez', decision: 'allow', hint: '' },
  { key: '2', label: 'Permitir nesta sessão', decision: 'allow_session', hint: '' },
  { key: '3', label: 'Negar', decision: 'deny', hint: '↵ esc' },
];

const DENY = 'deny';

/** The decision a key gives while an approval waits, or null for a key that gives none. */
export function answerForKey(key: string): string | null {
  if (key === 'Enter' || key === 'Escape') return DENY;
  return CHOICES.find((c) => c.key === key)?.decision ?? null;
}

export function ApprovalCard({
  entry,
  now,
  onAnswer,
}: {
  entry: ApprovalEntry;
  now: number;
  onAnswer: (decision: string) => void;
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
        </div>
      )}
      <div className="approval-reason" title={r.reason || undefined}>
        Pede <span className="approval-resource">{boundaryLabel(r.boundary_crossed)}</span>
        {r.rule ? (
          <>
            {' · regra '}
            <span className="approval-rule">{r.rule}</span>
          </>
        ) : null}
        .
      </div>
      {!answered && (
        <div className="approval-options" role="group" aria-label="Responder">
          {CHOICES.map((c) => (
            <button
              key={c.key}
              type="button"
              className={`approval-option${c.decision === DENY ? ' preselected' : ''}`}
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
