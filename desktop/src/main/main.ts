import { app, BrowserWindow, dialog, nativeTheme, shell } from 'electron';
import os from 'node:os';
import path from 'node:path';
import { CHANNELS } from '../shared/api';
import { Connection } from './connection';
import { broadcast, serveTheWindows } from './ipc';
import { systemLookup } from './locate';
import { userDataDir } from './paths';
import { quitQuestion } from './quit';
import { startServe } from './serve';

// The renderer is a second client of the daemon, like the TUI, and it is kept
// away from everything that is not drawing: no Node, no filesystem, no socket.
// The main process is the one that talks to the daemon (D2, D19); the preload
// hands the renderer a narrow API and nothing else.

// Set before anything reads it, which is before `ready`.
const userData = userDataDir(process.env, app.getPath('appData'));
if (userData.ok) {
  app.setPath('userData', userData.dir);
} else {
  console.error(`dcode: ${userData.reason}`);
  dialog.showErrorBox('O DCode não abriu', userData.reason);
  app.exit(1);
}

const daemon = new Connection(
  {
    lookup: systemLookup(process.env, os.homedir(), process.cwd()),
    // Started outside any project, with the app's environment as it came: each
    // session resolves its own project's configuration (D19, D20).
    serve: (bin, socket) =>
      startServe(bin, socket, {
        cwd: os.homedir(),
        env: process.env,
        echo: (stream, line) => process[stream].write(`dcode serve: ${line}\n`),
      }),
    log: (line) => console.log(`dcode: ${line}`),
  },
  {
    status: (status) => broadcast(CHANNELS.daemon, status),
    conversations: (frame) => broadcast(CHANNELS.conversations, frame),
    sessionEvents: (batch) => broadcast(CHANNELS.sessionEvents, batch),
    streamEnd: (end) => broadcast(CHANNELS.streamEnd, end),
  },
);

serveTheWindows(daemon);

function rendererURL(): { url: string } | { file: string } {
  if (MAIN_WINDOW_VITE_DEV_SERVER_URL) return { url: MAIN_WINDOW_VITE_DEV_SERVER_URL };
  return { file: path.join(__dirname, `../renderer/${MAIN_WINDOW_VITE_NAME}/index.html`) };
}

function createWindow(): void {
  const mac = process.platform === 'darwin';
  const win = new BrowserWindow({
    width: 1440,
    height: 900,
    minWidth: 900,
    minHeight: 560,
    title: 'DCode',
    // Painted before the first frame, so opening does not flash the other theme.
    backgroundColor: nativeTheme.shouldUseDarkColors ? '#161616' : '#f1f0ed',
    // The window draws its own title bar and keeps the native controls: on
    // macOS the traffic lights sit in the sidebar's 48px strip, centred on it.
    ...(mac ? { titleBarStyle: 'hiddenInset' as const, trafficLightPosition: { x: 16, y: 18 } } : {}),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      spellcheck: false,
    },
  });

  const target = rendererURL();
  if ('url' in target) void win.loadURL(target.url);
  else void win.loadFile(target.file);
}

// Everything the renderer could use to leave the app is closed: it may not
// navigate away, open windows, or attach webviews. Links go to the browser.
app.on('web-contents-created', (_event, contents) => {
  contents.on('will-navigate', (event, url) => {
    if (url !== contents.getURL()) event.preventDefault();
  });
  contents.on('will-attach-webview', (event) => event.preventDefault());
  contents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith('https://')) void shell.openExternal(url);
    return { action: 'deny' };
  });
});

app.whenReady().then(
  () => {
    daemon.start();
    createWindow();
    app.on('activate', () => {
      if (BrowserWindow.getAllWindows().length === 0) createWindow();
    });
  },
  (err: unknown) => {
    console.error('dcode: the app could not start:', err);
    app.exit(1);
  },
);

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

// Quitting stops the daemon the app started — asking first when sessions are
// at work, since they stop with it — and never one it attached to (D19).
let quitting: 'no' | 'deciding' | 'yes' = 'no';

/** Asks whether to stop `working` sessions; true for Fechar. */
async function confirmQuit(working: number): Promise<boolean> {
  const q = quitQuestion(working);
  const options = { type: 'question' as const, message: q.message, detail: q.detail, buttons: q.buttons, defaultId: 1, cancelId: 1, noLink: true };
  const win = BrowserWindow.getFocusedWindow() ?? BrowserWindow.getAllWindows()[0];
  const { response } = win ? await dialog.showMessageBox(win, options) : await dialog.showMessageBox(options);
  return response === 0;
}

async function stopAndQuit(): Promise<void> {
  try {
    const working = daemon.working();
    if (working > 0 && !(await confirmQuit(working))) {
      quitting = 'no';
      // Closing the last window is what asked, off macOS: staying open needs a window to stay in.
      if (BrowserWindow.getAllWindows().length === 0) createWindow();
      return;
    }
  } catch (err) {
    // The person asked to quit; a question that cannot be put must not keep the app open.
    console.error('dcode: could not ask before quitting, quitting anyway:', err);
  }
  await daemon.close().catch((err: unknown) => console.error('dcode: stopping the daemon failed:', err));
  quitting = 'yes';
  app.quit();
}

app.on('before-quit', (event) => {
  if (quitting === 'yes' || !daemon.startedByApp()) return;
  event.preventDefault();
  if (quitting === 'deciding') return;
  quitting = 'deciding';
  void stopAndQuit();
});

// Whatever quits the app, its streams close with it. A daemon it attached to
// keeps running, and one it started was stopped above.
app.on('will-quit', () => {
  void daemon.close();
});
