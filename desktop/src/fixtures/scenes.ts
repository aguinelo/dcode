// The reference states the visual check opens, by screenshot name (D18): which
// recording the window replays, which conversation is maximized at first (null
// is the grid), and the loops the recorded daemon conducts. What the person
// does after the window opens — a key, a click, a stored choice — is the
// check's, in tests/visual/states.json; here is only what the daemon would say.

import type { LoopsState } from '../state/loop';
import { emptyLoops } from '../state/loop';
import { loopRecording, recordedLoops } from './loop-recording';
import type { Recording } from './record';
import { recording, RECORDED_AT } from './recording';

export interface Scene {
  recording: Recording;
  /** The conversation maximized as the window opens; null opens on the grid. */
  session: string | null;
  loops: LoopsState;
}

/** A daemon with no conversations at all: the first run. */
export const emptyRecording: Recording = { recordedAt: RECORDED_AT, daemonVersion: recording.daemonVersion, events: [] };

const scene = (session: string | null, rec: Recording = recording, loops: LoopsState = emptyLoops): Scene => ({ recording: rec, session, loops });

export const referenceStates: Readonly<Record<string, Scene>> = {
  '02-janela-principal-rodando': scene('s-catalogar'),
  '03-janela-principal-aprovacao': scene('s-tokens'),
  '05-grade-em-repouso': scene(null),
  '06-grade-vazia': scene(null, emptyRecording),
  '11-plano-fixo-e-atividade': scene('s-frete', loopRecording, recordedLoops),
  '12-loop-ciclo-e-regua': scene('s-carrinho', loopRecording, recordedLoops),
  '13-loop-revisao-propostas': scene('s-qualificar', loopRecording, recordedLoops),
};
