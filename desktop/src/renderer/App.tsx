import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { emptyPrefs, dropBefore, moveProject, parsePrefs, PREFS_KEY, relabel, setAllCollapsed, setCollapsed, type Prefs } from '../state/prefs';
import { emptySession } from '../state/session';
import type { SessionsState } from '../state/sessions';
import { activeRows, countStates, projectsOf, type Row } from '../state/sidebar';
import { answerForKey } from './ApprovalCard';
import { DaemonBar } from './DaemonBar';
import type { Host } from './host';
import { useNow, useReducedMotion, useTick } from './hooks';
import { SessionPanel } from './SessionPanel';
import { Sidebar } from './Sidebar';
import { notYet } from './text';
import type { DaemonView, Outcome, WindowActions } from './window';

interface Toast {
  id: number;
  text: string;
}

const TOAST_MS = 6000;

function loadPrefs(host: Host): { prefs: Prefs; problem: string | null } {
  let raw: string | null;
  try {
    raw = window.localStorage.getItem(PREFS_KEY);
  } catch (err) {
    return { prefs: host.initialPrefs, problem: `Não foi possível ler as preferências locais (${(err as Error).message}); a lateral começa do arranjo inicial.` };
  }
  const read = parsePrefs(raw);
  if (read.problem) return { prefs: host.initialPrefs, problem: `${read.problem}; a lateral começa do arranjo inicial.` };
  return { prefs: read.prefs ?? host.initialPrefs ?? emptyPrefs, problem: null };
}

/** Whether keys typed now go into text. A disabled field takes none. */
function isEditable(el: Element | null): boolean {
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) return !el.disabled;
  return !!el && (el as HTMLElement).isContentEditable;
}

export function App({
  host,
  rows,
  sessions,
  daemon,
  actions,
  notices,
  initialSelection,
}: {
  host: Host;
  /** The sidebar's conversations. */
  rows: Row[];
  /** The sessions whose events the window holds — the ones it opened. */
  sessions: SessionsState;
  daemon: DaemonView;
  actions: WindowActions;
  /** Registers what to call with each notice that arrives after the first frame. */
  notices?: (say: (text: string) => void) => () => void;
  initialSelection: string | null;
}) {
  const now = useNow(1000);
  const reduced = useReducedMotion();
  const verbTick = useTick(2400, reduced);
  // What went wrong before the first frame is said in it: stored preferences
  // that could not be read, events that name no session to be shown in.
  const [boot] = useState(() => {
    const loaded = loadPrefs(host);
    const notes = sessions.problems.map((p) => `Evento ilegível, sem sessão para mostrá-lo: ${p.reason}`);
    return { prefs: loaded.prefs, notes: loaded.problem ? [loaded.problem, ...notes] : notes };
  });
  const [prefs, setPrefs] = useState(boot.prefs);
  const [selectedId, setSelectedId] = useState(initialSelection);
  const [history, setHistory] = useState<{ back: string[]; forward: string[] }>({ back: [], forward: [] });
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [toasts, setToasts] = useState<Toast[]>(() => boot.notes.map((text, i) => ({ id: i + 1, text })));
  const [userName, setUserName] = useState<string | null>(null);
  // Requests still on their way, so a second ↵ or click does not repeat one.
  const pending = useRef(new Set<string>());

  const say = useCallback((text: string) => {
    setToasts((t) => [...t.filter((x) => x.text !== text), { id: Date.now() + Math.random(), text }]);
  }, []);

  useEffect(() => {
    host.user().then(
      (u) => setUserName(u.name),
      (err: unknown) => say(`Não foi possível saber o nome de quem usa: ${(err as Error).message ?? String(err)}`),
    );
  }, [host, say]);

  useEffect(() => notices?.(say), [notices, say]);

  useEffect(() => {
    if (toasts.length === 0) return;
    const t = window.setTimeout(() => setToasts((all) => all.slice(1)), TOAST_MS);
    return () => window.clearTimeout(t);
  }, [toasts]);

  const updatePrefs = useCallback(
    (fn: (p: Prefs) => Prefs) => {
      const next = fn(prefs);
      setPrefs(next);
      try {
        window.localStorage.setItem(PREFS_KEY, JSON.stringify(next));
      } catch (err) {
        say(`Não foi possível guardar as preferências locais (${(err as Error).message}); a mudança vale só até fechar.`);
      }
    },
    [prefs, say],
  );

  const projects = useMemo(() => projectsOf(rows, prefs), [rows, prefs]);
  const shown = useMemo(() => projects.map((p) => p.id), [projects]);
  const counts = useMemo(() => countStates(rows), [rows]);
  // A session just opened has no events yet: it is drawn empty, and fills in
  // as its stream arrives.
  const selected = selectedId ? (sessions.byId[selectedId] ?? emptySession(selectedId)) : null;

  /**
   * Runs one request at a time per key, saying why when it is not done.
   * Answers its outcome, or null when one with the same key was still on its
   * way — never the bare value, since a done request's value can be null too.
   */
  const once = useCallback(
    async <T,>(key: string, run: () => Promise<Outcome<T>>): Promise<Outcome<T> | null> => {
      if (pending.current.has(key)) return null;
      pending.current.add(key);
      try {
        const out = await run();
        if (!out.ok) say(out.why);
        return out;
      } finally {
        pending.current.delete(key);
      }
    },
    [say],
  );

  const show = useCallback(
    (id: string) => {
      if (id === selectedId) return;
      setHistory((h) => ({ back: selectedId ? [...h.back, selectedId] : h.back, forward: [] }));
      setSelectedId(id);
    },
    [selectedId],
  );

  const open = useCallback(
    async (id: string) => {
      if (id === selectedId) return;
      const row = rows.find((r) => r.id === id);
      if (!row) return;
      const opened = await once(`open:${id}`, () => actions.open(row));
      if (opened?.ok) show(opened.value);
    },
    [actions, once, rows, selectedId, show],
  );

  const newSession = useCallback(async () => {
    if (pending.current.has('new')) return;
    pending.current.add('new');
    try {
      const created = await actions.newSession();
      if (!created) return;
      if (created.ok) show(created.value);
      else say(created.why);
    } finally {
      pending.current.delete('new');
    }
  }, [actions, say, show]);

  const send = useCallback(
    async (text: string) => {
      if (!selected) return;
      const id = selected.id;
      const done = await once(`send:${id}`, () => actions.send(id, text, selected.state === 'running'));
      // Emptied only once the daemon took it, and only if what is in the field
      // is still what was sent: a refusal keeps the text, said why.
      if (done?.ok) setDrafts((d) => (d[id] === text ? { ...d, [id]: '' } : d));
    },
    [actions, once, selected],
  );

  const stop = useCallback(() => {
    if (selected) void once(`stop:${selected.id}`, () => actions.stop(selected.id));
  }, [actions, once, selected]);

  const answer = useCallback(
    (decision: string) => {
      const approval = selected?.pendingApprovalId;
      if (!selected || !approval) return;
      void once(`answer:${approval}`, () => actions.answer(selected.id, approval, decision));
    },
    [actions, once, selected],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key.toLowerCase() === 'n') {
        e.preventDefault();
        void newSession();
        return;
      }
      if (mod && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        say(notYet('Procurar'));
        return;
      }
      if (mod && /^[1-9]$/.test(e.key)) {
        const target = activeRows(projects)[Number(e.key) - 1];
        if (target) {
          e.preventDefault();
          void open(target.id);
        }
        return;
      }
      if (!selected?.pendingApprovalId || isEditable(document.activeElement) || mod) return;
      // ↵ denies only from nowhere in particular: on a focused control — an
      // answer, a row — it is that control's own ↵.
      const focused = document.activeElement;
      const nowhere = !focused || focused === document.body || (focused instanceof HTMLTextAreaElement && focused.disabled);
      if (e.key === 'Enter' && !nowhere) return;
      const decision = answerForKey(e.key);
      if (decision) {
        e.preventDefault();
        answer(decision);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [projects, open, newSession, selected, answer, say]);

  return (
    <div className="window">
      <div className="window-main">
        <Sidebar
          drawsWindowControls={host.drawsWindowControls}
          projects={projects}
          selectedId={selectedId}
          now={now}
          userName={userName}
          canGoBack={history.back.length > 0}
          canGoForward={history.forward.length > 0}
          onBack={() => {
            const prev = history.back[history.back.length - 1];
            if (!prev) return;
            setHistory((h) => ({ back: h.back.slice(0, -1), forward: selectedId ? [selectedId, ...h.forward] : h.forward }));
            setSelectedId(prev);
          }}
          onForward={() => {
            const next = history.forward[0];
            if (!next) return;
            setHistory((h) => ({ back: selectedId ? [...h.back, selectedId] : h.back, forward: h.forward.slice(1) }));
            setSelectedId(next);
          }}
          onToggleAll={() => updatePrefs((p) => setAllCollapsed(p, shown, projects.some((x) => !x.collapsed)))}
          onNewSession={() => void newSession()}
          onMissing={(what) => say(notYet(what))}
          actions={{
            toggle: (id) => updatePrefs((p) => setCollapsed(p, id, !p.collapsed[id])),
            rename: (id, label) => updatePrefs((p) => relabel(p, id, label)),
            move: (id, to) => updatePrefs((p) => moveProject(p, shown, id, to)),
            dropBefore: (id, target) => updatePrefs((p) => dropBefore(p, shown, id, target)),
            select: (id) => void open(id),
          }}
        />
        <main className="main">
          {selected ? (
            <SessionPanel
              key={selected.id}
              session={selected}
              now={now}
              verbTick={verbTick}
              draft={drafts[selected.id] ?? ''}
              onDraft={(text) => setDrafts((d) => ({ ...d, [selected.id]: text }))}
              onAnswer={answer}
              onSend={(text) => void send(text)}
              onStop={stop}
              onMissing={(what) => say(notYet(what))}
            />
          ) : (
            <section className="panel empty-panel">
              <p>Nenhuma sessão aberta.</p>
            </section>
          )}
        </main>
      </div>
      <footer className="bottombar">
        <DaemonBar daemon={daemon} />
        <span className="spacer" />
        {counts.running > 0 && <span>{counts.running} rodando</span>}
        {counts.blocked > 0 && <span className="tone-warn">{counts.blocked} esperando você</span>}
      </footer>
      {toasts.length > 0 && (
        <div className="toasts" role="status" aria-live="polite">
          {toasts.map((t) => (
            <button key={t.id} type="button" className="toast" onClick={() => setToasts((all) => all.filter((x) => x.id !== t.id))}>
              {t.text}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
