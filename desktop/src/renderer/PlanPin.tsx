import type * as P from '../protocol/generated';
import type { PinnedPlan } from '../state/plan';
import { PHASES, childPhase, type StepAgents } from '../state/plan-agents';
import { tokens } from '../state/telemetry';

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

/** What a child spent, in one figure: in, out and cache read — never money in this slice. */
function spent(u: P.Usage): string {
  return `${tokens(u.input_tokens + u.output_tokens + (u.cache_read_tokens ?? 0))} tokens`;
}

/**
 * One agent under its step (D68): its name, which opens its conversation, the
 * pipeline with the phase it is in highlighted in its tone, the state in a
 * word, and its tokens.
 */
function PlanAgent({ child, canOpen, onOpen }: { child: P.Child; canOpen: boolean; onOpen?: (id: string) => void }) {
  const ph = childPhase(child);
  const current = PHASES[ph.at]!;
  return (
    <li className="plan-agent" data-plan-child-id={child.id} data-phase-current={current.key} data-child-state={child.state}>
      <button
        type="button"
        className="plan-agent-name"
        disabled={!canOpen || !onOpen}
        title={!onOpen ? undefined : canOpen ? `Abrir a conversa de ${child.title}` : 'sem conversa na lista desta janela'}
        onClick={() => onOpen?.(child.id)}
      >
        {child.title}
      </button>
      <ol className="phase-pipe" aria-label={`Fase: ${current.word} — ${ph.word}`}>
        {PHASES.map((p, i) => (
          <li
            key={p.key}
            data-phase={p.key}
            aria-current={i === ph.at ? 'step' : undefined}
            className={i < ph.at ? 'phase passed' : i === ph.at ? `phase current tone-${ph.tone}` : 'phase todo'}
          >
            {p.word}
          </li>
        ))}
      </ol>
      <span className={`plan-agent-word tone-${ph.tone}`}>{ph.word}</span>
      <span className="plan-agent-tokens mono">{spent(child.usage)}</span>
    </li>
  );
}

function AgentList({ kids, canOpen, onOpen, label }: { kids: P.Child[]; canOpen: (id: string) => boolean; onOpen?: (id: string) => void; label: string }) {
  return (
    <ul className="plan-agents" aria-label={label}>
      {kids.map((c) => (
        <PlanAgent key={c.id} child={c} canOpen={canOpen(c.id)} onOpen={onOpen} />
      ))}
    </ul>
  );
}

/**
 * The plan of the turn under way, pinned above the flow (D57). A window on the
 * grid gives it one line — the step and what it says —; an opened
 * conversation and the Crew, the whole checklist. An orchestrator's plan
 * stays pinned, and each step shows the agents that fulfil it (D68).
 */
export function PlanPin({
  pin,
  compact,
  agents,
  canOpen = () => false,
  onOpen,
}: {
  pin: PinnedPlan;
  compact: boolean;
  /** An orchestrator's children, by the step each names; absent, the plan is drawn as D57 drew it. */
  agents?: StepAgents;
  /** Whether a child's conversation is in the window's list, to be opened. */
  canOpen?: (id: string) => boolean;
  onOpen?: (id: string) => void;
}) {
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
  const outside = agents?.outside ?? [];
  return (
    <div className={agents ? 'plan-pin with-agents' : 'plan-pin'} role="status" aria-label={label}>
      <ol className="plan-pin-list">
        {pin.items.map((it) => {
          const kids = agents?.byStep[it.id] ?? [];
          return (
            <li key={it.id} className={`plan-pin-item status-${it.status}`}>
              <Glyph status={it.status} />
              <span className="plan-pin-text">{it.text}</span>
              {it.blocked && <span className="plan-pin-blocked">{it.blocked}</span>}
              {kids.length > 0 && <AgentList kids={kids} canOpen={canOpen} onOpen={onOpen} label={`Agentes do passo ${it.id}`} />}
            </li>
          );
        })}
      </ol>
      {outside.length > 0 && (
        <div className="plan-pin-outside">
          <span className="plan-pin-outside-head">Fora do plano</span>
          <AgentList kids={outside} canOpen={canOpen} onOpen={onOpen} label="Agentes fora do plano" />
        </div>
      )}
    </div>
  );
}
