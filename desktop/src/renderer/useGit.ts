import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import type { GitDiffResponse, GitResponse } from '../protocol/generated';
import type { GitState } from '../state/git';
import type { Outcome } from './window';

/** What reads a conversation's git: the daemon through the main process, or the recording (D41). */
export interface GitReader {
  readGit(conversationId: string): Promise<Outcome<GitResponse>>;
  readGitDiff(conversationId: string, path: string): Promise<Outcome<GitDiffResponse>>;
}

export const GitReaderContext = createContext<GitReader | null>(null);

const NO_READER = 'esta janela não tem como perguntar ao daemon pelo git.';

/** How long after the conversation moves the folder is read again: one read per burst of tool calls. */
const SETTLE_MS = 700;

/**
 * The git of a conversation's folder, read when the panel shows it (D41), again
 * when the conversation moves (`moved` changes) or the window comes back to the
 * front, and on demand. While a new read is on its way the last answer stays —
 * a list that blinks empty would read as "nothing changed" for an instant.
 */
export function useGit(conversationId: string | null, moved: unknown, enabled = true): { state: GitState; refresh: () => void } {
  const reader = useContext(GitReaderContext);
  // The last answer, kept with the conversation it answers for.
  const [got, setGot] = useState<{ id: string; state: GitState } | null>(null);
  const [tick, setTick] = useState(0);
  const refresh = useCallback(() => setTick((t) => t + 1), []);
  const active = enabled && conversationId !== null;

  useEffect(() => {
    if (!active) return;
    const onFocus = () => setTick((t) => t + 1);
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [active]);

  const answered = got !== null && got.id === conversationId;
  useEffect(() => {
    if (!active || !reader || conversationId === null) return;
    let live = true;
    const timer = setTimeout(
      () => {
        void reader.readGit(conversationId).then((r) => {
          if (live) setGot({ id: conversationId, state: r.ok ? { kind: 'ready', git: r.value } : { kind: 'failed', why: r.why } });
        });
      },
      answered ? SETTLE_MS : 0,
    );
    return () => {
      live = false;
      clearTimeout(timer);
    };
    // `answered` only picks the delay; a change in it is not a reason to read again.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reader, conversationId, moved, tick, active]);

  let state: GitState;
  if (!active) state = { kind: 'idle' };
  else if (!reader) state = { kind: 'failed', why: NO_READER };
  else state = answered ? got.state : { kind: 'loading' };
  return { state, refresh };
}

/** One file's diff, read when it is opened; null while it is on its way. */
export function useGitDiff(conversationId: string, path: string | null, moved: unknown): Outcome<GitDiffResponse> | null {
  const reader = useContext(GitReaderContext);
  // Kept with the file it is the diff of: another file's is never shown under this one's name.
  const [got, setGot] = useState<{ path: string; diff: Outcome<GitDiffResponse> } | null>(null);
  useEffect(() => {
    if (!path || !reader) return;
    let live = true;
    void reader.readGitDiff(conversationId, path).then((d) => {
      if (live) setGot({ path, diff: d });
    });
    return () => {
      live = false;
    };
  }, [reader, conversationId, path, moved]);
  if (!path) return null;
  if (!reader) return { ok: false, why: NO_READER };
  return got && got.path === path ? got.diff : null;
}
