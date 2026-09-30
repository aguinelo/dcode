import { contextBridge, ipcRenderer } from 'electron';
import { USER_CHANNEL, type DcodeApi, type UserInfo } from '../shared/api';

// The only door between the renderer and the rest of the app. Each entry is a
// single, named request: the renderer never gets ipcRenderer itself, so it
// cannot send on a channel nobody here chose.
const api: DcodeApi = {
  platform: process.platform,
  user: () => ipcRenderer.invoke(USER_CHANNEL) as Promise<UserInfo>,
};

contextBridge.exposeInMainWorld('dcode', api);
