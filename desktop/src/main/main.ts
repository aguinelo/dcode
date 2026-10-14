import { app, BrowserWindow, dialog, shell, webContents } from 'electron';
import { mkdir, stat, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import * as pty from 'node-pty';
import { CHANNELS } from '../shared/api';
import { Connection } from './connection';
import { broadcast, serveTheWindows } from './ipc';
import { systemLookup } from './locate';
import { userDataDir } from './paths';
import { quitQuestion } from './quit';
import { startServe } from './serve';
import { integration } from './shell';
import { Terminals } from './terminals';

// The renderer is a second client of the daemon, like the TUI, and it is kept
// away from everything that is not drawing: no Node, no filesystem, no socket.
// The main process is the one that talks to the daemon (D2, D19); the preload
// hands the renderer a narrow API and nothing else.

// The product's name, as the menu, the windows and the system show it. Set
// before `ready`, so nothing reads Electron's own name first.
const APP_NAME = 'DCode Box';
app.setName(APP_NAME);

// The mascot icon (D49). A packaged app takes it from its bundle; running from
// the sources, the binary is Electron's own, so the icon is set at runtime or
// the Dock and the taskbar show Electron's.
const DEV_ICON = path.join(app.getAppPath(), 'assets', 'icon', 'icon.png');

// Set before anything reads it, which is before `ready`.
const userData = userDataDir(process.env, app.getPath('appData'));
if (userData.ok) {
  app.setPath('userData', userData.dir);
} else {
  console.error(`dcode: ${userData.reason}`);
  dialog.showErrorBox('O DCode Box não abriu', userData.reason);
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
    loops: (push) => broadcast(CHANNELS.loops, push),
  },
);

// The terminals the windows open (D37): real PTYs, so sudo can ask for its
// password inside them. What passes through goes to the window that opened
// each one and nowhere else — never to a log.
const terminals = new Terminals({
  spawn: (file, args, o) => pty.spawn(file, args, { name: 'xterm-256color', cwd: o.cwd, env: o.env, cols: o.cols, rows: o.rows }),
  prepare: async (kind) => {
    const dir = path.join(app.getPath('userData'), 'shell');
    for (const f of integration(kind)) {
      const file = path.join(dir, f.name);
      await mkdir(path.dirname(file), { recursive: true });
      await writeFile(file, f.content, { mode: 0o644 });
    }
    return dir;
  },
  env: process.env,
  platform: process.platform,
  isDir: async (dir) => (await stat(dir).catch(() => null))?.isDirectory() ?? false,
  push: (owner, push) => {
    const to = webContents.fromId(owner);
    if (!to || to.isDestroyed()) return;
    to.send(push.kind === 'data' ? CHANNELS.terminalData : CHANNELS.terminalExit, push);
  },
  typeAfterMs: 3000,
  max: 8,
});

serveTheWindows(daemon, terminals);

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
    title: APP_NAME,
    ...(app.isPackaged ? {} : { icon: DEV_ICON }),
    // The theme is the renderer's to know (D35: localStorage, or the system's
    // under Sistema), so the window stays hidden until its first frame is
    // painted, and neither palette flashes the other on launch. This colour
    // only shows behind a resize; it is the default dark ground.
    show: false,
    backgroundColor: '#0a0a0a',
    // The window draws its own title bar and keeps the native controls: on
    // macOS the traffic lights sit at the left of the 40px top bar, centred on it.
    ...(mac ? { titleBarStyle: 'hiddenInset' as const, trafficLightPosition: { x: 14, y: 13 } } : {}),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      spellcheck: false,
    },
  });

  win.once('ready-to-show', () => win.show());
  // A renderer that never paints must not leave an invisible app behind: the
  // window is shown anyway, and whatever it failed at is on screen.
  setTimeout(() => {
    if (!win.isDestroyed() && !win.isVisible()) {
      console.error('dcode: the window did not paint within 5 s; showing it anyway');
      win.show();
    }
  }, 5000);

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
  // A window that reloads or goes away leaves no shell running behind it.
  const id = contents.id;
  contents.on('did-start-navigation', (details) => {
    if (details.isMainFrame && !details.isSameDocument) terminals.closeOwner(id);
  });
  contents.once('destroyed', () => terminals.closeOwner(id));
  contents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith('https://')) void shell.openExternal(url);
    return { action: 'deny' };
  });
});

app.whenReady().then(
  () => {
    if (!app.isPackaged && process.platform === 'darwin') {
      try {
        app.dock?.setIcon(DEV_ICON);
      } catch (err) {
        console.error(`dcode: the Dock icon could not be set from ${DEV_ICON}; the Dock shows Electron's:`, err);
      }
    }
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
  terminals.closeAll();
  void daemon.close();
});
