// The window in fixture mode — a plain browser, or `?fixture=` — showing a
// recording. Opening a row shows it; everything that would reach a daemon
// says that nothing was sent (D5).

import { NOT_CONNECTED } from './text';
import type { Outcome, WindowActions } from './window';

const unsent = async (): Promise<Outcome<null>> => ({ ok: false, why: NOT_CONNECTED });

export const fixtureActions: WindowActions = {
  open: async (row) => ({ ok: true, value: row.id }),
  send: unsent,
  stop: unsent,
  answer: unsent,
  newSession: async () => ({ ok: false, why: NOT_CONNECTED }),
  listModels: async () => ({ ok: false, why: NOT_CONNECTED }),
  switchModel: async () => ({ ok: false, why: NOT_CONNECTED }),
};
