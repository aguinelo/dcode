import { createContext, useContext, useMemo, useState, useSyncExternalStore } from 'react';
import type { Attachment } from '../shared/api';
import { attach, detach, MAX_IMAGE_BYTES, noAttachments, refuse, sizeOf, type Attached } from '../state/attachments';
import type { Outcome, WindowActions } from './window';

// The attachments of the window's fields — pictures (D46) and documents (D59):
// one store for every conversation, kept by the window like the drafts are, so
// switching away loses nothing. Reading a file is the main process's: a picked
// one is read there, and a pasted or dropped one goes there as bytes to be
// checked.

export class AttachmentStore {
  private state: Attached = noAttachments;
  private readonly listeners = new Set<() => void>();

  readonly subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  readonly get = (): Attached => this.state;

  update(change: (a: Attached) => Attached): void {
    const next = change(this.state);
    if (next === this.state) return;
    this.state = next;
    for (const l of this.listeners) l();
  }
}

export interface Attachments {
  store: AttachmentStore;
  /** Opens the system's picker and attaches what was picked to the conversation's next message. */
  pick(id: string): Promise<void>;
  /** Attaches files pasted or dropped: each checked by the main process, each refusal said. */
  addFiles(id: string, files: readonly File[]): Promise<void>;
  remove(id: string, index: number): void;
  /** Says why something was not done with the attachments: a notice, and in the field. */
  refuse(id: string, why: string): void;
}

const AttachmentsContext = createContext<Attachments | null>(null);
export const AttachmentsProvider = AttachmentsContext.Provider;

/** The window's attachments, with what attaches them. A refusal is said twice: a toast, and in the field. */
export function useAttachmentStore(actions: Pick<WindowActions, 'pickAttachments' | 'readAttachment'>, say: (text: string) => void): Attachments {
  const [store] = useState(() => new AttachmentStore());
  return useMemo(() => {
    const refused = (id: string, why: string) => {
      say(why);
      store.update((a) => refuse(a, id, why));
    };
    /** Attaches, and says in a toast what was refused — the field says it too, until the next attach. */
    const attachSaying = (id: string, files: Attachment[], why: string[]) => {
      const before = store.get().draft[id]?.length ?? 0;
      store.update((a) => attach(a, id, files, why));
      const now = store.get().refusal[id];
      const left = before + files.length - (store.get().draft[id]?.length ?? 0);
      if (now && (why.length > 0 || left > 0)) say(now);
    };
    return {
      store,
      pick: async (id) => {
        const got = await actions.pickAttachments();
        if (!got.ok) return refused(id, `Nada foi anexado: ${got.why}`);
        attachSaying(id, got.value.attachments, got.value.refused);
      },
      addFiles: async (id, files) => {
        const read: Attachment[] = [];
        const why: string[] = [];
        for (const file of files) {
          const got = await readFile(actions, file);
          if (got.ok) read.push(got.value);
          else why.push(got.why);
        }
        attachSaying(id, read, why);
      },
      remove: (id, index) => store.update((a) => detach(a, id, index)),
      refuse: refused,
    };
  }, [actions, say, store]);
}

/**
 * A pasted or dropped file, to the main process as bytes. One larger than any
 * kind carries is refused before it is read; the main process holds each kind
 * to its own size.
 */
async function readFile(actions: Pick<WindowActions, 'readAttachment'>, file: File): Promise<Outcome<Attachment>> {
  const name = file.name || 'anexo';
  if (file.size > MAX_IMAGE_BYTES) {
    return { ok: false, why: `${name} tem ${sizeOf(file.size)}, e um anexo vai ao modelo com até ${sizeOf(MAX_IMAGE_BYTES)}.` };
  }
  try {
    return await actions.readAttachment(name, new Uint8Array(await file.arrayBuffer()));
  } catch (err) {
    return { ok: false, why: `${name} não foi lido: ${err instanceof Error ? err.message : String(err)}` };
  }
}

/** The window's attachments; null outside a window that has them. */
export function useAttachments(): Attachments | null {
  return useContext(AttachmentsContext);
}

const none = () => () => {};
const empty = () => noAttachments;

/** What is attached and sent, kept current. */
export function useAttached(): Attached {
  const attachments = useAttachments();
  return useSyncExternalStore(attachments?.store.subscribe ?? none, attachments?.store.get ?? empty);
}
