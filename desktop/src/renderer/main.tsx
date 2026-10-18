import '@fontsource-variable/geist-mono';
import '@fontsource-variable/geist';
import '@fontsource/fragment-mono';
import './styles/tokens.css';
import './styles/base.css';
import './styles/sidebar.css';
import './styles/session.css';
import './styles/markdown.css';
import './styles/composer.css';
import './styles/grid.css';
import './styles/search.css';
import './styles/crew.css';
import './styles/crew-board.css';
import './styles/crew-pages.css';
import './styles/start.css';
import './styles/terminal.css';
import './styles/actions.css';
import './styles/history.css';
import './styles/context.css';
import './styles/commands.css';
import './styles/loop.css';
import './styles/turn.css';
import './styles/pr.css';
import './styles/agents.css';
import './styles/question-card.css';

import { StrictMode, useSyncExternalStore } from 'react';
import { createRoot } from 'react-dom/client';
import { asOf } from '../fixtures/record';
import { recording } from '../fixtures/recording';
import { referenceStates } from '../fixtures/scenes';
import { applyRaw, emptySessions } from '../state/sessions';
import { rowsOfSessions } from '../state/sidebar';
import { App } from './App';
import { fixtureActions } from './fixture';
import { currentHost, type Host } from './host';
import { LiveStore } from './live';
import { GitReaderContext } from './useGit';
import { CardsContext } from './QuestionCard';

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
      loops={snap.loops}
    />
  );
}

const host = currentHost();
// `?fixture=<state>` opens one of the handoff's reference states — what the
// visual check loads. An unknown name is an error on screen, never a default.
const wanted = new URLSearchParams(window.location.search).get('fixture');
const scene = wanted === null ? null : (referenceStates[wanted] ?? null);
const api = window.dcode;

if (wanted !== null && scene === null) {
  root.textContent = `Estado de referência desconhecido: ${wanted}. Os que existem: ${Object.keys(referenceStates).join(', ')}.`;
  document.documentElement.dataset.ready = 'error';
} else {
  if (api && wanted === null) {
    // Inside Electron the window is a client of the daemon (D2, D19).
    const store = new LiveStore(api);
    store.start();
    createRoot(root).render(
      <StrictMode>
        <GitReaderContext.Provider value={store.actions}>
          <CardsContext.Provider value={store.actions.answerQuestions}>
            <LiveWindow host={host} store={store} />
          </CardsContext.Provider>
        </GitReaderContext.Provider>
      </StrictMode>,
    );
  } else {
    // In a browser — fixture mode — the window shows a recording, replayed as
    // if it had just happened. The bottom bar says so (D5).
    const shown = scene?.recording ?? recording;
    const sessions = applyRaw(emptySessions, asOf(shown, Date.now()));
    const rows = rowsOfSessions(sessions);
    createRoot(root).render(
      <StrictMode>
        <GitReaderContext.Provider value={fixtureActions}>
          <CardsContext.Provider value={fixtureActions.answerQuestions}>
            <App
              host={host}
              rows={rows}
              sessions={sessions}
              daemon={{ state: 'recording', version: shown.daemonVersion }}
              actions={fixtureActions}
              initialSelection={scene?.session ?? null}
              // 02 and 03 are the grid's maximized window, captured before the Crew became the main screen (D63).
              initialLook={scene?.look}
              loops={scene?.loops}
            />
          </CardsContext.Provider>
        </GitReaderContext.Provider>
      </StrictMode>,
    );
  }
  // Ready once the bundled fonts are in: a screenshot before that would
  // measure the fallback font, not the design. A face that does not load is a
  // build defect, and it is said rather than drawn around.
  // The text face is the system's (D29); the bundled one is the monospace,
  // which Linux also reads text in.
  const faces = ['400 12px "Geist Mono Variable"'];
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
