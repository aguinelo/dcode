// The window in fixture mode — a plain browser, or `?fixture=` — showing a
// recording. Opening a row shows it; everything that would reach a daemon
// says that nothing was sent (D5). Reading is answered from the recording:
// a folder's git, as the recorded daemon would give it (D41).

import { recordedDiffs, recordedGit, recordedPRs } from '../fixtures/git';
import { loopRecording } from '../fixtures/loop-recording';
import { recording } from '../fixtures/recording';
import { NOT_CONNECTED } from './text';
import type { Outcome, WindowActions } from './window';

const unsent = async (): Promise<Outcome<null>> => ({ ok: false, why: NOT_CONNECTED });

/** The folder a recorded conversation runs in, from its `session.created`. */
function recordedWorkspace(id: string): string | undefined {
  for (const e of [...recording.events, ...loopRecording.events]) {
    if (e.session_id === id && e.type === 'session.created') return e.payload.workspace;
  }
  return undefined;
}

const NOT_RECORDED = 'A gravação não tem o git desta pasta, e nada foi enviado.';

export const fixtureActions: WindowActions = {
  open: async (row) => ({ ok: true, value: row.id }),
  send: unsent,
  pickAttachments: async () => ({ ok: false, why: NOT_CONNECTED }),
  readAttachment: async () => ({ ok: false, why: NOT_CONNECTED }),
  stop: unsent,
  answer: unsent,
  newSession: async () => ({ ok: false, why: NOT_CONNECTED }),
  newSessionIn: async () => ({ ok: false, why: NOT_CONNECTED }),
  pickFolder: async () => null,
  closeSession: unsent,
  watch: () => {},
  listModels: async () => ({ ok: false, why: NOT_CONNECTED }),
  switchModel: async () => ({ ok: false, why: NOT_CONNECTED }),
  setMode: unsent,
  rename: unsent,
  exec: unsent,
  undo: async () => ({ ok: false, why: NOT_CONNECTED }),
  compact: async () => ({ ok: false, why: NOT_CONNECTED }),
  listSkills: async () => ({ ok: false, why: NOT_CONNECTED }),
  // A command the recording's project wrote, so the / menu shows one of the person's own (D50).
  listCommands: async () => ({
    ok: true,
    value: {
      commands: [{ name: 'review', description: 'revisa o que está preparado contra o AGENTS.md', source: 'project', path: 'review.md', body: 'Review the staged change against AGENTS.md. $ARGUMENTS' }],
      overrides: [],
    },
  }),
  listMemory: async () => ({ ok: false, why: NOT_CONNECTED }),
  readHistory: async () => ({ state: 'failed', reason: NOT_CONNECTED }),
  readGit: async (id) => {
    const git = recordedGit[recordedWorkspace(id) ?? ''];
    return git ? { ok: true, value: git } : { ok: false, why: NOT_RECORDED };
  },
  // A refusal is the recorded daemon's own words, as the panel would get them.
  readGitPR: async (id) => {
    const pr = recordedPRs[recordedWorkspace(id) ?? ''];
    if (pr === undefined) return { ok: false, why: NOT_RECORDED, kind: 'failed' };
    return typeof pr === 'string' ? { ok: false, why: pr, kind: 'refused' } : { ok: true, value: pr };
  },
  openGitPR: async () => ({ ok: false, why: NOT_CONNECTED, kind: 'failed' }),
  startLoop: async () => ({ ok: false, why: NOT_CONNECTED }),
  stopLoop: async () => ({ ok: false, why: NOT_CONNECTED }),
  listSpecs: async () => ({ ok: false, why: NOT_CONNECTED }),
  continueLoop: async () => ({ ok: false, why: NOT_CONNECTED }),
  openProposal: unsent,
  readGitDiff: async (id, path) => {
    const git = recordedGit[recordedWorkspace(id) ?? ''];
    const diff = git?.files.some((f) => f.path === path) ? recordedDiffs[path] : undefined;
    return diff ? { ok: true, value: diff } : { ok: false, why: `A gravação não tem o diff de ${path}, e nada foi enviado.` };
  },
};
