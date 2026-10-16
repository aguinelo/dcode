import type { PinnedPlan } from '../state/plan';

/** A step's glyph in the pinned checklist: done, blocked, still to come. The step under way is a turning ring, drawn. */
function glyphOf(status: string): string {
  if (status === 'done') return '✓';
  if (status === 'active') return '◌';
  if (status === 'blocked') return '⊘';
  return '○';
}

function Glyph({ status }: { status: string }) {
  if (status === 'active') return <span className="plan-pin-glyph status-active" aria-hidden="true"><span className="plan-pin-ring" /></span>;
  return <span className={`plan-pin-glyph status-${status}`} aria-hidden="true">{glyphOf(status)}</span>;
}

/**
 * The plan of the turn under way, pinned above the flow (D57). A window on the
 * grid gives it one line — the step and what it says —; an opened
 * conversation and the Crew, the whole checklist.
 */
export function PlanPin({ pin, compact }: { pin: PinnedPlan; compact: boolean }) {
  const label = `Plano do turno: passo ${pin.step} de ${pin.total}`;
  if (compact) {
    const status = pin.items[pin.step - 1]?.status ?? 'pending';
    return (
      <div className="plan-pin compact" role="status" aria-label={label} title={pin.items.map((i) => `${glyphOf(i.status)} ${i.text}`).join('\n')}>
        <Glyph status={status} />
        <span className="plan-pin-step">
          passo {pin.step} de {pin.total}
        </span>
        <span className="plan-pin-sep">·</span>
        <span className="plan-pin-current">{pin.current}</span>
      </div>
    );
  }
  return (
    <div className="plan-pin" role="status" aria-label={label}>
      <ol className="plan-pin-list">
        {pin.items.map((it) => (
          <li key={it.id} className={`plan-pin-item status-${it.status}`}>
            <Glyph status={it.status} />
            <span className="plan-pin-text">{it.text}</span>
            {it.blocked && <span className="plan-pin-blocked">{it.blocked}</span>}
          </li>
        ))}
      </ol>
    </div>
  );
}
