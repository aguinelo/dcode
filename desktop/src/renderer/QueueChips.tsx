import type { Queue } from '../state/queue';

/**
 * The messages waiting for the turn to end (D47), above the field: in order,
 * each with its ×, the one on its way without. A queue a failed send stopped
 * says why and waits for the person to resume it.
 */
export function QueueChips({ queue, onDrop, onResume }: { queue: Queue | undefined; onDrop: (index: number) => void; onResume: () => void }) {
  if (!queue || queue.items.length === 0) return null;
  const halted = queue.phase === 'halted';
  return (
    <div className="composer-queue" data-queue-phase={queue.phase} aria-label="Mensagens na fila">
      {halted ? (
        <div className="composer-queue-head halted" role="alert">
          <span className="composer-queue-why">A fila parou: {queue.why}</span>
          <button type="button" className="composer-queue-resume" onClick={onResume}>
            Retomar
          </button>
        </div>
      ) : (
        <div className="composer-queue-head">Na fila · seguem quando o turno terminar · ⌃X tira a última</div>
      )}
      <div className="composer-queue-items">
        {queue.items.map((text, i) => {
          const going = queue.phase === 'sending' && i === 0;
          return (
            <span key={`${i}:${text}`} className={`composer-queued${going ? ' going' : ''}`} title={text}>
              <span className="composer-queued-text">{text}</span>
              {!going && (
                <button type="button" className="composer-queued-drop" aria-label="Tirar da fila" onClick={() => onDrop(i)}>
                  ×
                </button>
              )}
            </span>
          );
        })}
      </div>
    </div>
  );
}
