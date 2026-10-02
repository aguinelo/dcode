import type { DaemonView } from './window';

/**
 * The daemon's part of the bottom bar. `data-daemon` is the state the check
 * reads (docs/loop/tasks.md, "O que a régua lê"); the text beside it is the
 * version when connected and the reason otherwise — a status without its why
 * would be a mark nobody can act on.
 */
export function DaemonBar({ daemon }: { daemon: DaemonView }) {
  switch (daemon.state) {
    case 'recording':
      // A recording is neither a live daemon nor a failure: neutral, and named
      // — never "daemon", which would look like a connection (D5).
      return (
        <>
          <span className="status" title={`Eventos gravados de um daemon v${daemon.version}, reproduzidos sem conexão: nada é enviado.`}>
            <span className="status-dot replay" />
            gravação
          </span>
          <span className="tone-faint">v{daemon.version}</span>
        </>
      );
    case 'connecting':
      return (
        <span className="daemon" data-daemon="connecting">
          <span className="status">
            <span className="status-dot connecting" />
            conectando ao daemon…
          </span>
        </span>
      );
    case 'connected':
      return (
        <span
          className="daemon"
          data-daemon="connected"
          title={`${daemon.started ? 'Subido pelo app, que o encerra ao fechar' : 'Já estava rodando; continua quando o app fechar'} · ${daemon.socket}`}
        >
          <span className="status">
            <span className="status-dot connected" />
            daemon
          </span>
          <span className="tone-faint">v{daemon.version}</span>
        </span>
      );
    case 'lost':
      return (
        <span className="daemon" data-daemon="lost" title={daemon.reason}>
          <span className="status tone-err">
            <span className="status-dot failed" />
            daemon caiu
          </span>
          <span className="daemon-reason">{daemon.reason}</span>
        </span>
      );
    case 'failed':
      return (
        <span className="daemon" data-daemon="failed" title={daemon.reason}>
          <span className="status tone-err">
            <span className="status-dot failed" />
            daemon não subiu
          </span>
          <span className="daemon-reason">{daemon.reason}</span>
        </span>
      );
  }
}
