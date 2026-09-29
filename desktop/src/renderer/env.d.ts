import type { DcodeApi } from '../shared/api';

declare global {
  interface Window {
    /** Present inside Electron, from the preload; absent in a plain browser. */
    dcode?: DcodeApi;
  }
}
