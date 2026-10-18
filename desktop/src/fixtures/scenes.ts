// The reference states the visual check opens, by name (D18): which recording
// the window replays, the version of the screen it opens on when none was
// chosen, which conversation the grid maximizes at first, and the loops the
// recorded daemon conducts. What the person does after the window opens — a
// key, a click, a stored choice — is the check's, in tests/visual/states.json;
// here is only what the daemon would say.

import type { LoopsState } from '../state/loop';
import { emptyLoops } from '../state/loop';
import type { Look } from '../renderer/TopBar';
import { loopRecording, recordedLoops } from './loop-recording';
import { emptyOrchestraRecording, orchestraRecording } from './orchestra-recording';
import { orchestrasRecording } from './orchestras-recording';
import { planRecording } from './plan-recording';
import { questionsRecording } from './questions-recording';
import type { Recording } from './record';
import { recording, RECORDED_AT } from './recording';

export interface Scene {
  recording: Recording;
  /** The version of the screen when none was chosen: the Crew unless said (D63). */
  look?: Look;
  /** The conversation the grid maximizes as the window opens; null opens on the grid. */
  session: string | null;
  loops: LoopsState;
}

/** A daemon with no conversations at all: the first run. */
export const emptyRecording: Recording = { recordedAt: RECORDED_AT, daemonVersion: recording.daemonVersion, events: [] };

const crew = (rec: Recording, loops: LoopsState = emptyLoops): Scene => ({ recording: rec, session: null, loops });

export const referenceStates: Readonly<Record<string, Scene>> = {
  // The grid's maximized window, captured before the Crew became the main screen (D63).
  '02-janela-principal-rodando': { recording, look: 'grade', session: 's-catalogar', loops: emptyLoops },
  '03-janela-principal-aprovacao': { recording, look: 'grade', session: 's-tokens', loops: emptyLoops },
  // The Crew (D63), over the first recording, over none, over the second, and over the third.
  crew: crew(recording),
  'crew-vazia': crew(emptyRecording),
  'crew-loop': crew(loopRecording, recordedLoops),
  // An orchestrator's conversation with its live agent list (RN-9): busy, and before its first dispatch.
  'crew-orquestra': crew(orchestraRecording),
  'crew-orquestra-vazia': crew(emptyOrchestraRecording),
  // An orchestrator that asks in its conversation (D67): a card of three merges, and the card it proposed from the person's words.
  'crew-perguntas': crew(questionsRecording),
  // An orchestrator's plan as its main view (D68): each step with its agent's phase, and a merge card below.
  'crew-orquestra-plano': crew(planRecording),
  // Two orchestrated projects and an ordinary conversation (D69): one row per orchestrator, no child and no reviewer listed.
  'crew-orquestras': crew(orchestrasRecording),
};
