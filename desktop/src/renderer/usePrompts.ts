import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { parseInput } from '../state/commands';
import { dropAt, dropLast, enqueue, failed, noQueues, resume, sent, stepAll, type Queue, type Queues } from '../state/queue';
import { olderPrompts, promptsOf, sourceOf } from '../state/recall';
import type { SessionsState } from '../state/sessions';
import type { Outcome, WindowActions } from './window';

type Once = <T>(key: string, run: () => Promise<Outcome<T>>) => Promise<Outcome<T> | null>;

/** How many continuations back the field reads for what was asked before (D47). */
const SOURCE_DEPTH = 8;

/** A continued conversation's earlier prompts, read from the record of the one it continues. */
type Older = { state: 'loading' } | { state: 'loaded'; prompts: readonly string[] } | { state: 'failed' };

/** What the fields share and keep while windows come and go: the queues, and earlier conversations' prompts (D47). */
export interface Prompts {
  queue(id: string): Queue | undefined;
  /** ⌥↵ during a turn. False, said why, when the text cannot wait in a queue. */
  enqueue(id: string, text: string): boolean;
  dropLast(id: string): void;
  dropAt(id: string, index: number): void;
  resume(id: string): void;
  /** The prompts of the conversations this one continues, oldest first; null until read. */
  older(sourceId: string): readonly string[] | null;
  /** Reads them, once; a failure is said, and the next ↑ that wants them asks again. */
  readOlder(sourceId: string): void;
}

const PromptsContext = createContext<Prompts | null>(null);
export const PromptsProvider = PromptsContext.Provider;
export const usePromptsContext = () => useContext(PromptsContext);

export function usePrompts(
  actions: Pick<WindowActions, 'send' | 'readHistory'>,
  sessions: SessionsState,
  once: Once,
  say: (text: string) => void,
): Prompts {
  const [queues, setQueues] = useState<Queues>(noQueues);
  const [olders, setOlders] = useState<Readonly<Record<string, Older>>>({});
  const asked = useRef(new Set<string>());

  // A queue moves when its conversation does — idle again, the oldest is
  // marked on its way — decided while drawing, as the grid's places are.
  const moved = stepAll(queues, (id) => sessions.byId[id]);
  if (moved !== queues) setQueues(moved);

  // What is marked on its way goes to the daemon, once.
  const going = useRef(new Set<string>());
  useEffect(() => {
    for (const [id, q] of Object.entries(queues)) {
      const text = q.items[0];
      if (q.phase !== 'sending' || text === undefined || going.current.has(id)) continue;
      going.current.add(id);
      const halt = (why: string, said: boolean) => {
        if (!said) say(`A fila parou: ${why}`);
        setQueues((all) => failed(all, id, why));
      };
      once(`send:${id}`, () => actions.send(id, text, false))
        .then(
          (out) => {
            if (out?.ok) setQueues((all) => sent(all, id));
            // `once` already said a refusal; a send of this conversation still on its way it did not.
            else halt(out ? out.why : 'outra mensagem desta conversa ainda estava a caminho', !!out);
          },
          (err: unknown) => halt((err as Error).message ?? String(err), false),
        )
        .finally(() => going.current.delete(id));
    }
  }, [queues, actions, once, say]);

  const readChain = useCallback(
    async (id: string): Promise<string[]> => {
      const chain: string[][] = [];
      let at: string | null = id;
      for (let depth = 0; at && depth < SOURCE_DEPTH; depth++) {
        const h = await actions.readHistory(at);
        if (h.state !== 'loaded') throw new Error(h.state === 'failed' ? h.reason : 'histórico ainda carregando');
        chain.unshift(promptsOf(h.view.entries));
        at = sourceOf(h.view.entries);
      }
      return chain.reduce((all, own) => olderPrompts(all, own), [] as string[]);
    },
    [actions],
  );

  const older = useCallback((sourceId: string) => {
    const o = olders[sourceId];
    return o?.state === 'loaded' ? o.prompts : null;
  }, [olders]);

  const load = useCallback(
    (sourceId: string) => {
      if (asked.current.has(sourceId)) return;
      asked.current.add(sourceId);
      setOlders((all) => ({ ...all, [sourceId]: { state: 'loading' } }));
      readChain(sourceId).then(
        (prompts) => setOlders((all) => ({ ...all, [sourceId]: { state: 'loaded', prompts } })),
        (err: unknown) => {
          asked.current.delete(sourceId);
          say(`As mensagens da conversa continuada não foram lidas: ${(err as Error).message ?? String(err)}`);
          setOlders((all) => ({ ...all, [sourceId]: { state: 'failed' } }));
        },
      );
    },
    [readChain, say],
  );

  const add = useCallback(
    (id: string, text: string) => {
      const input = parseInput(text);
      if (input.kind === 'unknown' || input.kind === 'bad') {
        say(input.why);
        return false;
      }
      // A command acts on the session as it is now; waiting it would run it on another.
      if (input.kind !== 'message') {
        say('Só mensagens entram na fila: um comando vale quando é dado. Envie-o quando o turno terminar.');
        return false;
      }
      setQueues((q) => enqueue(q, id, text));
      return true;
    },
    [say],
  );

  return useMemo(
    () => ({
      queue: (id: string) => queues[id],
      enqueue: add,
      dropLast: (id: string) => setQueues((q) => dropLast(q, id)),
      dropAt: (id: string, index: number) => setQueues((q) => dropAt(q, id, index)),
      resume: (id: string) => setQueues((q) => resume(q, id)),
      older,
      readOlder: load,
    }),
    [queues, add, older, load],
  );
}
