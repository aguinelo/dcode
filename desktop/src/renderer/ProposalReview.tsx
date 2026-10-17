import type { ReviewView } from '../state/review';

/**
 * The proposed definition of done, waiting for a person before any work (D56,
 * Figure 3 of the proposal): each criterion with its class and what running it
 * did, the proposer's reason, where the eye should land, what is protected —
 * and the file, one click away in the editor. The work starts only when the
 * person says; nothing here starts on its own (done-qualifier RN-6).
 */
export function ProposalReview({
  review,
  onOpen,
  onStop,
  onStart,
}: {
  review: ReviewView;
  onOpen: (loopId: string, file: string) => void;
  onStop: (loopId: string) => void;
  onStart: (loopId: string) => void;
}) {
  const id = review.loopId;
  return (
    <div className="loop-review" data-loop-review={id}>
      {review.proposals.length === 0 && <p className="loop-note tone-warn">O loop espera uma revisão, e o daemon não listou proposta escrita.</p>}
      {review.proposals.map((p) => (
        <section key={p.file} className="loop-proposal" aria-label={`Proposta de ${p.folder}`}>
          <div className="loop-proposal-head">
            <span className="mono loop-proposal-file" title={p.file}>
              {p.file}
            </span>
            <span className="loop-proposal-count">{p.heading}</span>
            <button type="button" className="chip" title="Abre o done.toml no editor do sistema" onClick={() => onOpen(id, p.file)}>
              Abrir no editor
            </button>
          </div>
          {p.missing && <p className="loop-note tone-warn">{p.missing}</p>}
          {p.criteria.map((c) => (
            <div key={c.name} className="loop-criterion">
              <span className={`loop-class class-${c.klass}`}>{c.label}</span>
              <span className={`mono ${c.passed ? 'tone-ok' : 'tone-err'}`}>
                {c.passed ? '✓' : '✗'} {c.ran}
              </span>
              <span className="loop-criterion-name">{c.name}</span>
              <span className="mono loop-criterion-command" title={c.command}>
                {c.command}
              </span>
              {c.why && <span className="loop-criterion-why">{c.why}</span>}
              {c.warning && <span className="loop-criterion-warn tone-warn">⚠ {c.warning}</span>}
              {c.output && (
                <details className="loop-criterion-output">
                  <summary>saída</summary>
                  <pre className="mono">{c.output}</pre>
                </details>
              )}
            </div>
          ))}
          {p.noAcceptance && <p className="loop-note tone-warn">⚠ {p.noAcceptance}</p>}
          {p.earlierLabel && (
            <details className="loop-proposal-earlier" data-earlier-proposals={p.earlier.length}>
              <summary>
                {p.earlierLabel} · o turno repropôs; só a última foi medida e escrita
              </summary>
              {p.earlier.map((e) => (
                <div key={e.sequence} className="loop-earlier">
                  <span className="loop-earlier-title">{e.title}</span>
                  {e.criteria.map((c, i) => (
                    <div key={i} className="loop-criterion">
                      <span className="loop-criterion-name">{c.name}</span>
                      <span className="mono loop-criterion-command" title={c.command}>
                        {c.command}
                      </span>
                      {c.said && <span className="loop-criterion-why">{c.said}</span>}
                      {c.why && <span className="loop-criterion-why">{c.why}</span>}
                    </div>
                  ))}
                  {e.protected.length > 0 && (
                    <p className="loop-note">
                      Protegido: <span className="mono">{e.protected.join(', ')}</span>
                    </p>
                  )}
                </div>
              ))}
            </details>
          )}
          {p.protected.length > 0 && (
            <p className="loop-note">
              Protegido: <span className="mono">{p.protected.join(', ')}</span>
            </p>
          )}
        </section>
      ))}
      <p className="loop-note">O trabalho começa quando você disser. Edite o arquivo, se quiser: o daemon mede de novo antes de começar.</p>
      <div className="loop-review-actions">
        <button type="button" className="chip" onClick={() => onStop(id)}>
          ⏹ Parar o loop
        </button>
        <button type="button" className="start-open" onClick={() => onStart(id)}>
          Começar o trabalho
        </button>
      </div>
    </div>
  );
}
