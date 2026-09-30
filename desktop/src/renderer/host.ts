// Where the window runs, and what that decides: inside Electron the preload
// answers who the person is and the OS draws the window controls; in a plain
// browser — fixture mode, which the visual check uses — the recording answers
// and the window draws placeholders where the controls would be.

import { recordedClient } from '../fixtures/recording';
import type { Prefs } from '../state/prefs';
import type { UserInfo } from '../shared/api';

export interface Host {
  kind: 'electron' | 'fixture';
  platform: string;
  /** Draw the three window controls ourselves: only where no native ones exist. */
  drawsWindowControls: boolean;
  user(): Promise<UserInfo>;
  /** What the sidebar starts from when nothing is stored locally yet. */
  initialPrefs: Prefs;
}

export function currentHost(): Host {
  const api = window.dcode;
  if (api) {
    return {
      kind: 'electron',
      platform: api.platform,
      drawsWindowControls: false,
      user: () => api.user(),
      // The sessions this version shows are the recording's, so the recording's
      // arrangement of them is where the sidebar starts.
      initialPrefs: recordedClient.prefs,
    };
  }
  return {
    kind: 'fixture',
    platform: 'browser',
    drawsWindowControls: true,
    user: () => Promise.resolve({ name: recordedClient.userName }),
    initialPrefs: recordedClient.prefs,
  };
}

/** The first name and the initials, as the sidebar footer shows them. */
export function nameParts(full: string): { first: string; initials: string } {
  const words = full.trim().split(/\s+/).filter(Boolean);
  const first = words[0] ?? '';
  const last = words.length > 1 ? words[words.length - 1] ?? '' : '';
  const initials = (first.slice(0, 1) + (last ? last.slice(0, 1) : first.slice(1, 2))).toUpperCase();
  return { first, initials };
}
