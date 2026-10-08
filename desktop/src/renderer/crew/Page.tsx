import { useEffect, useState, type ReactNode } from 'react';
import { shortPath } from '../../state/sidebar';
import type { Outcome } from '../window';

/** A project the window knows, by its workspace path. */
export interface Place {
  id: string;
  label: string;
}

/** A section's page: its name, what it is, and what it holds. */
export function Page({ title, about, tools, children }: { title: string; about: string; tools?: ReactNode; children: ReactNode }) {
  return (
    <div className="crew-page">
      <div className="crew-page-inner">
        <header className="crew-page-head">
          <div>
            <h1>{title}</h1>
            <p>{about}</p>
          </div>
          {tools}
        </header>
        {children}
      </div>
    </div>
  );
}

/** What is not there yet, said as such: never a list that only looks empty. */
export function NotYet({ text }: { text: string }) {
  return (
    <div className="crew-not-yet" role="note" data-not-yet>
      <strong>Ainda não</strong>
      <span>{text}</span>
    </div>
  );
}

/** Which project a section reads: the known ones, the one in focus first. */
export function PlacePicker({ places, value, onChange }: { places: readonly Place[]; value: string | null; onChange: (id: string) => void }) {
  if (places.length === 0) return null;
  return (
    <label className="crew-picker">
      <span>projeto</span>
      <select value={value ?? ''} onChange={(e) => onChange(e.target.value)}>
        {places.map((p) => (
          <option key={p.id} value={p.id} title={p.id}>
            {p.label} — {shortPath(p.id, 28)}
          </option>
        ))}
      </select>
    </label>
  );
}

export type Loaded<T> = { state: 'loading' } | { state: 'ok'; value: T } | { state: 'no'; why: string };

/** Asks for one workspace's data each time the workspace changes; a late answer for another one is dropped. */
export function useWorkspaceData<T>(workspace: string | null, load: (ws: string) => Promise<Outcome<T>>): Loaded<T> {
  const [got, setGot] = useState<{ ws: string; out: Outcome<T> } | null>(null);
  useEffect(() => {
    if (!workspace) return;
    let live = true;
    load(workspace).then(
      (out) => live && setGot({ ws: workspace, out }),
      (err: unknown) => live && setGot({ ws: workspace, out: { ok: false, why: String((err as Error)?.message ?? err) } }),
    );
    return () => {
      live = false;
    };
  }, [workspace, load]);
  if (!got || got.ws !== workspace) return { state: 'loading' };
  return got.out.ok ? { state: 'ok', value: got.out.value } : { state: 'no', why: got.out.why };
}

/** The workspace a section starts on: the one in focus when it is known, else the first. */
export function usePlace(places: readonly Place[], preferred: string | null): [string | null, (id: string) => void] {
  const [chosen, setChosen] = useState<string | null>(null);
  const known = (id: string | null) => !!id && places.some((p) => p.id === id);
  const value = known(chosen) ? chosen : known(preferred) ? preferred : (places[0]?.id ?? null);
  return [value, setChosen];
}
