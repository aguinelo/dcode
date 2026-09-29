import '@fontsource-variable/geist';
import '@fontsource-variable/geist-mono';
import './styles/tokens.css';
import './styles/base.css';
import './styles/sidebar.css';
import './styles/session.css';
import './styles/composer.css';

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { asOf } from '../fixtures/record';
import { recording, referenceStates } from '../fixtures/recording';
import { applyRaw, emptySessions } from '../state/sessions';
import { activeSessions, projectsOf } from '../state/sidebar';
import { App } from './App';
import { currentHost } from './host';

const root = document.getElementById('root');
if (!root) throw new Error('index.html has no #root to draw into');

const host = currentHost();
// Until the main process talks to the daemon, the window shows a recording,
// replayed as if it had just happened. The bottom bar says so.
const sessions = applyRaw(emptySessions, asOf(recording, Date.now()));

// `?fixture=<state>` opens one of the handoff's reference states — what the
// visual check loads. An unknown name is an error on screen, never a default.
const wanted = new URLSearchParams(window.location.search).get('fixture');
const fixtureSession = wanted === null ? null : (referenceStates[wanted] ?? null);

if (wanted !== null && fixtureSession === null) {
  root.textContent = `Estado de referência desconhecido: ${wanted}. Os que existem: ${Object.keys(referenceStates).join(', ')}.`;
  document.documentElement.dataset.ready = 'error';
} else {
  const first = activeSessions(projectsOf(sessions, host.initialPrefs))[0]?.id ?? sessions.order[0] ?? null;
  createRoot(root).render(
    <StrictMode>
      <App host={host} sessions={sessions} daemonVersion={recording.daemonVersion} initialSelection={fixtureSession ?? first} />
    </StrictMode>,
  );
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
