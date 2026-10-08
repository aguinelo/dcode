import { menuOf } from '../../state/models';
import type { Look } from '../TopBar';
import type { DaemonView, WindowActions } from '../window';
import { Page, PlacePicker, usePlace, useWorkspaceData, type Place } from './Page';

function DaemonFacts({ daemon }: { daemon: DaemonView }) {
  switch (daemon.state) {
    case 'recording':
      return <p className="crew-quiet">Sem daemon: esta janela reproduz eventos gravados de um daemon v{daemon.version}, e nada é enviado.</p>;
    case 'connecting':
      return <p className="crew-quiet">Conectando ao daemon…</p>;
    case 'connected':
      return (
        <dl className="crew-facts">
          <dt>estado</dt>
          <dd className="tone-ok">conectado</dd>
          <dt>versão</dt>
          <dd className="mono">{daemon.version}</dd>
          <dt>socket</dt>
          <dd className="mono" title={daemon.socket}>
            {daemon.socket}
          </dd>
          <dt>quem subiu</dt>
          <dd>{daemon.started ? 'este app, que o encerra ao fechar (D19)' : 'já estava rodando; continua quando o app fechar'}</dd>
        </dl>
      );
    case 'lost':
    case 'failed':
      return (
        <dl className="crew-facts">
          <dt>estado</dt>
          <dd className="tone-err">{daemon.state === 'lost' ? 'caiu' : 'não subiu'}</dd>
          <dt>por quê</dt>
          <dd className="wrap">{daemon.reason}</dd>
        </dl>
      );
  }
}

function Models({ places, preferred, actions }: { places: readonly Place[]; preferred: string | null; actions: WindowActions }) {
  const [ws, setWs] = usePlace(places, preferred);
  const got = useWorkspaceData(ws, actions.listModels);
  const options = got.state === 'ok' ? menuOf(got.value, null) : [];
  return (
    <section className="crew-card-block">
      <header className="crew-block-head">
        <h2>Modelos</h2>
        <PlacePicker places={places} value={ws} onChange={setWs} />
      </header>
      <p className="crew-quiet">O que uma sessão neste projeto pode pedir, como o daemon resolve (GET /v1/models, D28).</p>
      {!ws ? (
        <p className="crew-quiet">Nenhum projeto conhecido ainda: abra uma sessão num projeto e os modelos dele aparecem aqui.</p>
      ) : got.state === 'loading' ? (
        <p className="crew-quiet">Perguntando ao daemon…</p>
      ) : got.state === 'no' ? (
        <p className="tone-err">{got.why}</p>
      ) : (
        <ul className="crew-models">
          {options.map((o) => (
            <li key={o.name} title={o.notice ?? undefined}>
              <span className="mono crew-model-name">{o.name}</span>
              <span className="crew-model-detail mono">{o.detail}</span>
              <span className={o.measured ? 'tone-ok' : 'tone-faint'}>{o.measured ? 'medido' : '◌ sem medição'}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

/**
 * Configurações: only what is real now — which version of the screen, where
 * the daemon stands, and what models each project can ask for. Nothing here
 * pretends to save a setting the daemon does not have.
 */
export function SettingsSection({
  look,
  onLook,
  daemon,
  places,
  preferred,
  actions,
}: {
  look: Look;
  onLook: (look: Look) => void;
  daemon: DaemonView;
  places: readonly Place[];
  preferred: string | null;
  actions: WindowActions;
}) {
  return (
    <Page title="Configurações" about="O que esta janela e o daemon dizem agora. O resto da configuração continua nos arquivos do dcode.">
      <section className="crew-card-block">
        <h2>Versão da tela</h2>
        <p className="crew-quiet">Lembrada nesta janela. A grade é o padrão (D32).</p>
        <div className="crew-choice" role="group" aria-label="Versão da tela">
          {(['grade', 'crew'] as const).map((l) => (
            <button key={l} type="button" className={l === look ? 'on' : ''} aria-pressed={l === look} onClick={() => onLook(l)}>
              {l === 'grade' ? 'Grade' : 'Crew'}
            </button>
          ))}
        </div>
      </section>
      <section className="crew-card-block">
        <h2>Daemon</h2>
        <DaemonFacts daemon={daemon} />
      </section>
      <Models places={places} preferred={preferred} actions={actions} />
    </Page>
  );
}
