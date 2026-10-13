import { createContext, useContext, useMemo, useState, useSyncExternalStore } from 'react';
import type { ImageAttachment } from '../shared/api';
import { attach, detach, MAX_IMAGE_BYTES, MAX_IMAGES, noAttachments, refuse, sizeOf, type Attached } from '../state/attachments';
import type { Outcome, WindowActions } from './window';

// The pictures of the window's fields (D46): one store for every
// conversation, kept by the window like the drafts are, so switching away
// loses nothing. Reading a file is the main process's: a picked one is read
// there, and a pasted or dropped one goes there as bytes to be checked.

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

export interface Images {
  store: AttachmentStore;
  /** Opens the system's picker and attaches what was picked to the conversation's next message. */
  pick(id: string): Promise<void>;
  /** Attaches pictures pasted or dropped: each checked by the main process, each refusal said. */
  addFiles(id: string, files: readonly File[]): Promise<void>;
  remove(id: string, index: number): void;
  /** Says why something was not done with the pictures: a notice, and in the field. */
  refuse(id: string, why: string): void;
}

const ImagesContext = createContext<Images | null>(null);
export const ImagesProvider = ImagesContext.Provider;

/** The window's pictures, with what attaches them. A refusal is said twice: a toast, and in the field. */
export function useImageStore(actions: Pick<WindowActions, 'pickImages' | 'readImage'>, say: (text: string) => void): Images {
  const [store] = useState(() => new AttachmentStore());
  return useMemo(() => {
    const refused = (id: string, why: string) => {
      say(why);
      store.update((a) => refuse(a, id, why));
    };
    /** Attaches, and says in a toast what was refused — the field says it too, until the next attach. */
    const attachSaying = (id: string, images: ImageAttachment[], why: string[]) => {
      const room = MAX_IMAGES - (store.get().draft[id]?.length ?? 0);
      store.update((a) => attach(a, id, images, why));
      const now = store.get().refusal[id];
      if (now && (why.length > 0 || images.length > room)) say(now);
    };
    return {
      store,
      pick: async (id) => {
        const got = await actions.pickImages();
        if (!got.ok) return refused(id, `Nenhuma imagem foi anexada: ${got.why}`);
        attachSaying(id, got.value.images, got.value.refused);
      },
      addFiles: async (id, files) => {
        const read: ImageAttachment[] = [];
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

/** A pasted or dropped file, to the main process as bytes. One too large is refused before it is read. */
async function readFile(actions: Pick<WindowActions, 'readImage'>, file: File): Promise<Outcome<ImageAttachment>> {
  const name = file.name || 'imagem';
  if (file.size > MAX_IMAGE_BYTES) {
    return { ok: false, why: `${name} tem ${sizeOf(file.size)}, e uma imagem vai ao modelo com até ${sizeOf(MAX_IMAGE_BYTES)}.` };
  }
  try {
    return await actions.readImage(name, new Uint8Array(await file.arrayBuffer()));
  } catch (err) {
    return { ok: false, why: `${name} não foi lida: ${err instanceof Error ? err.message : String(err)}` };
  }
}

/** The window's pictures; null outside a window that has them. */
export function useImages(): Images | null {
  return useContext(ImagesContext);
}

const none = () => () => {};
const empty = () => noAttachments;

/** What is attached and sent, kept current. */
export function useAttached(): Attached {
  const images = useImages();
  return useSyncExternalStore(images?.store.subscribe ?? none, images?.store.get ?? empty);
}
