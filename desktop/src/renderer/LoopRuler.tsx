import { useContext, useState } from 'react';
import { plural } from '../state/format';
import { loopRuler } from '../state/loop';
import { useNow } from './hooks';
import { ProposalReview } from './ProposalReview';
import { LoopContext } from './useLoops';

function phaseGlyph(phase: string): string {
  switch (phase) {
    case 'done':
      return '✓';
    case 'unfinished':
      return '✗';
    case 'pending':
    case 'skipped':
      return '○';
    default:
      return '↻';
  }
}

/**
 * The ruler at the top of a loop's window (D55, Figure 2 of the proposal): the
 * goal or the folder, the state, the folder it is in of how many, and — opened
 * — each folder with its phase, one click from its session. When the run
 * ends, its verdict; when it fails, why. Nothing at all for a conversation
 * that is not a loop's.
 */
export function LoopRuler({ sessionId, now, unfolded = false }: { sessionId: string; now: number; unfolded?: boolean }) {
  const loops = useContext(LoopContext);
  const [open, setOpen] = useState(unfolded);
  const run = loops?.of(sessionId) ?? null;
  if (!loops || !run) return null;
  const v = loopRuler(run, now);
  const review = loops.review(run);
  return (
    <div className="loop-ruler" data-loop-id={v.id} data-loop-state={run.state}>
      <div className="loop-line">
        <span className={`loop-glyph tone-${v.tone}`}>↻</span>
        <span className="loop-title" title={v.title}>
          {v.title}
        </span>
        <span className={`loop-state tone-${v.tone}`}>{v.state}</span>
        <span className="loop-elapsed mono">{v.elapsed}</span>
      </div>
      {v.folders.length > 0 && (
        <button type="button" className="loop-step" aria-expanded={open} onClick={() => setOpen((x) => !x)}>
          {v.step ?? plural(v.folders.length, 'pasta', 'pastas')} {open ? '▴' : '▾'}
        </button>
      )}
      {open && (
        <div className="loop-folders-run">
          {v.folders.map((f) => (
            <button
              key={f.path}
              type="button"
              className={`loop-folder-run${f.current ? ' current' : ''}`}
              disabled={!f.sessionId}
              title={f.sessionId ? 'Abrir a conversa desta pasta' : 'Esta pasta ainda não teve sessão'}
              onClick={() => f.sessionId && loops.openSession(f.sessionId)}
            >
              <span className="loop-folder-glyph">{phaseGlyph(f.phase)}</span>
              <span className="mono loop-folder-path">{f.path}</span>
              <span className="loop-folder-phase">{f.label}</span>
            </button>
          ))}
        </div>
      )}
      {review && <ProposalReview review={review} onOpen={loops.openProposal} onStop={loops.stopNow} onStart={loops.startWork} />}
      {v.verdict && <div className={`loop-note tone-${v.tone}`}>{v.verdict}</div>}
      {v.error && <div className="loop-note tone-err">{v.error}</div>}
      {v.warnings.map((w) => (
        <div key={w} className="loop-note tone-warn">
          ⚠ {w}
        </div>
      ))}
      {v.live && !review && (
        <div className="loop-stops">
          <button type="button" className="chip" title="Interrompe o turno e encerra o loop" onClick={() => loops.stopNow(v.id)}>
            ⏹ Parar agora
          </button>
          <button type="button" className="chip" title="Deixa a pasta corrente terminar e não abre a próxima" onClick={() => loops.stopAfter(v.id)}>
            ⏭ Parar depois desta pasta
          </button>
        </div>
      )}
    </div>
  );
}

/** The Crew's Pronto tab gets the loop of the conversation in focus (D56, Figure 4): nothing when it has none. */
export function LoopSection({ sessionId }: { sessionId: string }) {
  const loops = useContext(LoopContext);
  const now = useNow(1000);
  if (!loops?.of(sessionId)) return null;
  return (
    <section className="crew-loop" aria-label="Loop">
      <h3 className="crew-loop-title">Loop</h3>
      <LoopRuler sessionId={sessionId} now={now} unfolded />
    </section>
  );
}
