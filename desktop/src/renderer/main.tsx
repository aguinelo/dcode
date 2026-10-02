import '@fontsource-variable/geist';
import '@fontsource-variable/geist-mono';
import './styles/tokens.css';
import './styles/base.css';
import './styles/sidebar.css';
import './styles/session.css';
import './styles/composer.css';

import { StrictMode, useSyncExternalStore } from 'react';
import { createRoot } from 'react-dom/client';
import { asOf } from '../fixtures/record';
import { recording, referenceStates } from '../fixtures/recording';
import { applyRaw, emptySessions } from '../state/sessions';
import { activeRows, projectsOf, rowsOfSessions } from '../state/sidebar';
import { App } from './App';
import { fixtureActions } from './fixture';
import { currentHost, type Host } from './host';
import { LiveStore } from './live';

const root = document.getElementById('root');
if (!root) throw new Error('index.html has no #root to draw into');

/** The window connected: drawn from the store the main process feeds. */
function LiveWindow({ host, store }: { host: Host; store: LiveStore }) {
  const snap = useSyncExternalStore(store.subscribe, store.getSnapshot);
  return (
    <App
      host={host}
      rows={snap.rows}
      sessions={snap.sessions}
      daemon={snap.daemon}
      actions={store.actions}
      notices={store.onNotice}
      initialSelection={null}
    />
  );
}

const host = currentHost();
// `?fixture=<state>` opens one of the handoff's reference states — what the
// visual check loads. An unknown name is an error on screen, never a default.
const wanted = new URLSearchParams(window.location.search).get('fixture');
const fixtureSession = wanted === null ? null : (referenceStates[wanted] ?? null);
const api = window.dcode;

if (wanted !== null && fixtureSession === null) {
  root.textContent = `Estado de referência desconhecido: ${wanted}. Os que existem: ${Object.keys(referenceStates).join(', ')}.`;
  document.documentElement.dataset.ready = 'error';
} else {
  if (api && wanted === null) {
    // Inside Electron the window is a client of the daemon (D2, D19).
    const store = new LiveStore(api);
    store.start();
    createRoot(root).render(
      <StrictMode>
        <LiveWindow host={host} store={store} />
      </StrictMode>,
    );
  } else {
    // In a browser — fixture mode — the window shows a recording, replayed as
    // if it had just happened. The bottom bar says so (D5).
    const sessions = applyRaw(emptySessions, asOf(recording, Date.now()));
    const rows = rowsOfSessions(sessions);
    const first = activeRows(projectsOf(rows, host.initialPrefs))[0]?.id ?? sessions.order[0] ?? null;
    createRoot(root).render(
      <StrictMode>
        <App
          host={host}
          rows={rows}
          sessions={sessions}
          daemon={{ state: 'recording', version: recording.daemonVersion }}
          actions={fixtureActions}
          initialSelection={fixtureSession ?? first}
        />
      </StrictMode>,
    );
  }
  // Ready once the bundled fonts are in: a screenshot before that would
  // measure the fallback font, not the design. A face that does not load is a
  // build defect, and it is said rather than drawn around.
  const faces = ['400 14px "Geist Variable"', '500 14px "Geist Variable"', '600 10.5px "Geist Variable"', '400 12.5px "Geist Mono Variable"'];
  void Promise.all(faces.map((f) => document.fonts.load(f)))
    .then((loaded) => {
      const missing = faces.filter((_, i) => (loaded[i] ?? []).length === 0);
      if (missing.length > 0) throw new Error(`fonte não carregou: ${missing.join(', ')}`);
      return document.fonts.ready;
    })
    .then(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))))
    .then(
      () => {
        document.documentElement.dataset.ready = 'true';
      },
      (err: unknown) => {
        console.error(err);
        document.documentElement.dataset.ready = 'error';
        document.documentElement.dataset.error = String(err);
      },
    );
}
