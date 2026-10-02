import { app, BrowserWindow, dialog, ipcMain, nativeTheme, shell } from 'electron';
import path from 'node:path';
import { USER_CHANNEL } from '../shared/api';
import { userDataDir } from './paths';
import { currentUser } from './user';

// The renderer is a second client of the daemon, like the TUI, and it is kept
// away from everything that is not drawing: no Node, no filesystem, no socket.
// The main process will be the one that talks to the daemon (next version);
// the preload hands the renderer a narrow API and nothing else.

// Set before anything reads it, which is before `ready`.
const userData = userDataDir(process.env, app.getPath('appData'));
if (userData.ok) {
  app.setPath('userData', userData.dir);
} else {
  console.error(`dcode: ${userData.reason}`);
  dialog.showErrorBox('O DCode não abriu', userData.reason);
  app.exit(1);
}

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

ipcMain.handle(USER_CHANNEL, (event) => {
  // Answered for the app's own window only.
  if (!BrowserWindow.fromWebContents(event.sender)) throw new Error('dcode: user() asked by an unknown sender');
  return currentUser();
});

app.whenReady().then(
  () => {
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
