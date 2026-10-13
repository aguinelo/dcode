import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import { forgetMissing, markSeen, togglePin } from '../state/attention';
import { capacityFor, gridShape, planGrid } from '../state/grid';
import { measureOf } from '../state/models';
import { paneView } from '../state/pane';
import { emptyPrefs, dropBefore, moveProject, parsePrefs, PREFS_KEY, relabel, setAllCollapsed, setCollapsed, type Prefs } from '../state/prefs';
import { emptySession } from '../state/session';
import type { SessionsState } from '../state/sessions';
import { countStates, projectsOf, rowOfSession, type Row } from '../state/sidebar';
import { answerForKey } from './ApprovalCard';
import { Grid } from './Grid';
import type { Host } from './host';
import { useNow, useReducedMotion, useTick, useWindowSize } from './hooks';
import { ModelMenu } from './ModelMenu';
import { Search } from './Search';
import { SessionPanel, type Peer } from './SessionPanel';
import { Sidebar } from './Sidebar';
import { notYet } from './text';
import { TopBar } from './TopBar';
import { useAttention } from './useAttention';
import { useModelLists } from './useModelLists';
import { useModelMenu } from './useModelMenu';
import type { DaemonView, Outcome, WindowActions } from './window';

interface Toast {
  id: number;
  text: string;
}

const TOAST_MS = 6000;

/** In the history of the stage, the grid's place. */
const GRID = '';

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

function sameIds(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((x, i) => x === b[i]);
}

/** Focus nowhere in particular: the window's own keys apply, not a control's. */
function focusIsNowhere(): boolean {
  const el = document.activeElement;
  return !el || el === document.body || (el instanceof HTMLTextAreaElement && el.disabled);
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
  /** The sessions whose events the window holds: the open one and the grid's. */
  sessions: SessionsState;
  daemon: DaemonView;
  actions: WindowActions;
  /** Registers what to call with each notice that arrives after the first frame. */
  notices?: (say: (text: string) => void) => () => void;
  /** The conversation open at first; null opens on the grid. */
  initialSelection: string | null;
}) {
  const now = useNow(1000);
  const reduced = useReducedMotion();
  const verbTick = useTick(2400, reduced);
  const size = useWindowSize();
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
  const [searching, setSearching] = useState(false);
  const [focus, setFocus] = useState(0);
  // Requests still on their way, so a second ↵ or click does not repeat one.
  const pending = useRef(new Set<string>());

  const say = useCallback((text: string) => {
    setToasts((t) => [...t.filter((x) => x.text !== text), { id: Date.now() + Math.random(), text }]);
  }, []);
  const [attention, updateAttention] = useAttention(say);

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

  const rowById = useMemo(() => new Map(rows.map((r) => [r.id, r])), [rows]);
  const projects = useMemo(() => projectsOf(rows, prefs), [rows, prefs]);
  const shown = useMemo(() => projects.map((p) => p.id), [projects]);
  const counts = useMemo(() => countStates(rows), [rows]);

  // The grid, in stable places: the plan reads the places as last drawn, and
  // they are kept the way React keeps a value from the previous render.
  const [placed, setPlaced] = useState<string[]>([]);
  const capacity = capacityFor(size.width, size.height);
  const plan = useMemo(() => planGrid(rows, attention, placed, capacity), [rows, attention, placed, capacity]);
  if (!sameIds(plan.ids, placed)) setPlaced(plan.ids);
  const onGridIds = useMemo(() => new Set(plan.ids), [plan.ids]);
  const focusAt = Math.min(focus, Math.max(plan.ids.length - 1, 0));

  // A panel shows its conversation live: the grid's sessions are followed.
  useEffect(() => {
    for (const id of plan.ids) {
      const r = rowById.get(id);
      if (r) actions.watch(r);
    }
  }, [plan.ids, rowById, actions]);

  // What the daemon no longer has, the grid stops remembering.
  useEffect(() => {
    if (rows.length > 0) updateAttention((a) => forgetMissing(a, new Set(rows.map((r) => r.id))));
  }, [rows, updateAttention]);

  // A session just opened has no events yet: it is drawn empty, and fills in
  // as its stream arrives.
  const selected = selectedId ? (sessions.byId[selectedId] ?? emptySession(selectedId)) : null;
  const selectedRow = selectedId ? (rowById.get(selectedId) ?? null) : null;

  /** Runs one request at a time per key, saying why when it is not done; null when one was on its way. */
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

  /** The person looked at a conversation: what it had done by now is seen. */
  const seeNow = useCallback(
    (id: string | null) => {
      const r = id ? rowById.get(id) : undefined;
      if (r) updateAttention((a) => markSeen(a, r.id, r.lastAt));
    },
    [rowById, updateAttention],
  );

  /** Puts a conversation on the stage, or the grid with null. */
  const show = useCallback(
    (id: string | null) => {
      if (id === selectedId) return;
      // Leaving one counts as having seen what it did while it was open.
      seeNow(selectedId);
      seeNow(id);
      setHistory((h) => ({ back: [...h.back, selectedId ?? GRID], forward: [] }));
      setSelectedId(id);
    },
    [selectedId, seeNow],
  );

  const travel = (to: string | undefined, push: (h: { back: string[]; forward: string[] }) => { back: string[]; forward: string[] }) => {
    if (to === undefined) return;
    seeNow(selectedId);
    setHistory(push);
    setSelectedId(to === GRID ? null : to);
  };

  const open = useCallback(
    async (id: string) => {
      if (id === selectedId) return;
      const row = rowById.get(id);
      if (!row) return;
      const opened = await once(`open:${id}`, () => actions.open(row));
      if (opened?.ok) show(opened.value);
    },
    [actions, once, rowById, selectedId, show],
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

  const answerIn = useCallback(
    (sessionId: string, approvalId: string, decision: string) => {
      if (approvalId) void once(`answer:${approvalId}`, () => actions.answer(sessionId, approvalId, decision));
    },
    [actions, once],
  );

  const answer = useCallback(
    (decision: string) => {
      if (selected?.pendingApprovalId) answerIn(selected.id, selected.pendingApprovalId, decision);
    },
    [answerIn, selected],
  );

  const models = useModelMenu(actions, sessions, once, show);

  // Whether each conversation on the stage runs on a measured family, as the daemon's list says.
  const stage = useMemo(
    () => [...new Set([...plan.ids, ...(selectedId ? [selectedId] : [])].map((id) => sessions.byId[id]?.info?.workspace ?? rowById.get(id)?.workspace ?? ''))],
    [plan.ids, selectedId, sessions, rowById],
  );
  const lists = useModelLists(actions, stage);
  const measureFor = useCallback(
    (row: Row) => {
      const info = sessions.byId[row.id]?.info;
      const running = info ? { model: info.model, family: info.family ?? null } : row.model ? { model: row.model, family: null } : null;
      return measureOf(lists.get(info?.workspace ?? row.workspace), running);
    },
    [lists, sessions],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      const key = e.key.toLowerCase();
      if (mod && key === 'k') {
        e.preventDefault();
        // Drawn and focused before this key's handling ends, so what is typed
        // right after ⌘K lands in the search, not in the window behind it.
        flushSync(() => setSearching((s) => !s));
        return;
      }
      // While the search or a menu is open, its keys are its own.
      if (searching || models.menu) return;
      if (mod && key === 'n') {
        e.preventDefault();
        void newSession();
        return;
      }
      if (mod && key === '0') {
        e.preventDefault();
        show(null);
        return;
      }
      if (mod && /^[1-9]$/.test(e.key)) {
        const id = plan.ids[Number(e.key) - 1];
        if (id) {
          e.preventDefault();
          void open(id);
        }
        return;
      }
      if (mod || isEditable(document.activeElement)) return;
      // ↵ answers only from nowhere in particular: on a focused control it is
      // that control's own ↵ (D22).
      const nowhere = focusIsNowhere();
      if (selected) {
        if (selected.pendingApprovalId) {
          if (e.key === 'Enter' && !nowhere) return;
          const decision = answerForKey(e.key);
          if (decision) {
            e.preventDefault();
            answer(decision);
          }
        } else if (e.key === 'Escape') {
          e.preventDefault();
          show(null);
        }
        return;
      }
      const n = plan.ids.length;
      if (n === 0) return;
      const cols = gridShape(n).cols;
      const moves: Record<string, number> = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -cols, ArrowDown: cols };
      const step = moves[e.key];
      if (step !== undefined) {
        e.preventDefault();
        setFocus(Math.max(0, Math.min(n - 1, focusAt + step)));
        return;
      }
      const id = plan.ids[focusAt];
      if (!id) return;
      const waiting = sessions.byId[id]?.pendingApprovalId;
      if (waiting) {
        if (e.key === 'Enter' && !nowhere) return;
        const decision = answerForKey(e.key);
        if (decision) {
          e.preventDefault();
          answerIn(id, waiting, decision);
        }
        return;
      }
      if (e.key === 'Enter' && nowhere) {
        e.preventDefault();
        void open(id);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [plan.ids, focusAt, sessions, open, newSession, selected, answer, answerIn, searching, models.menu, show]);

  const peers: Peer[] = plan.ids
    .filter((id) => id !== selectedId)
    .flatMap((id) => {
      const r = rowById.get(id);
      return r ? [{ row: r, view: paneView(r, sessions.byId[id], now) }] : [];
    });

  return (
    <div className={`window${searching ? ' searching' : ''}`}>
      <TopBar
        host={host}
        daemon={daemon}
        counts={counts}
        overflow={plan.overflow.length}
        onGrid={!selected}
        canGoBack={history.back.length > 0}
        canGoForward={history.forward.length > 0}
        onBack={() =>
          travel(history.back[history.back.length - 1], (h) => ({ back: h.back.slice(0, -1), forward: [selectedId ?? GRID, ...h.forward] }))
        }
        onForward={() => travel(history.forward[0], (h) => ({ back: [...h.back, selectedId ?? GRID], forward: h.forward.slice(1) }))}
      />
      <div className="window-main">
        <Sidebar
          projects={projects}
          selectedId={selectedId}
          onGridIds={onGridIds}
          gridShown={!selected}
          gridCount={plan.ids.length}
          now={now}
          userName={userName}
          onToggleAll={() => updatePrefs((p) => setAllCollapsed(p, shown, projects.some((x) => !x.collapsed)))}
          onShowGrid={() => show(null)}
          onNewSession={() => void newSession()}
          onSearch={() => setSearching(true)}
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
              row={selectedRow}
              peers={peers}
              now={now}
              verbTick={verbTick}
              draft={drafts[selected.id] ?? ''}
              onDraft={(text) => setDrafts((d) => ({ ...d, [selected.id]: text }))}
              onAnswer={answer}
              onSend={(text) => void send(text)}
              onStop={stop}
              onMissing={(what) => say(notYet(what))}
              onGrid={() => show(null)}
              onOpenPeer={(id) => void open(id)}
              onModel={(anchor) => models.open(selectedRow ?? rowOfSession(selected), anchor)}
              measure={measureFor(selectedRow ?? rowOfSession(selected))}
            />
          ) : (
            <Grid
              plan={plan}
              rows={rowById}
              sessions={sessions}
              attention={attention}
              focus={focusAt}
              now={now}
              onFocus={setFocus}
              onOpen={(id) => void open(id)}
              onPin={(id) => updateAttention((a) => togglePin(a, id))}
              onSeen={seeNow}
              onAnswer={answerIn}
              onModel={models.open}
              measureFor={measureFor}
              onNew={() => void newSession()}
              onSearch={() => setSearching(true)}
            />
          )}
        </main>
      </div>
      {models.menu && (
        <ModelMenu
          anchor={models.menu.anchor}
          options={models.menu.options}
          problem={models.menu.problem}
          busy={models.busy}
          onPick={(name) => void models.pick(name)}
          onClose={models.close}
        />
      )}
      {toasts.length > 0 && (
        <div className="toasts" role="status" aria-live="polite">
          {toasts.map((t) => (
            <button key={t.id} type="button" className="toast" onClick={() => setToasts((all) => all.filter((x) => x.id !== t.id))}>
              {t.text}
            </button>
          ))}
        </div>
      )}
      {searching && (
        <Search
          projects={projects}
          project={selected ? (selected.info?.workspace ?? selectedRow?.workspace ?? null) : null}
          now={now}
          onOpen={(id) => {
            setSearching(false);
            void open(id);
          }}
          onClose={() => setSearching(false)}
        />
      )}
    </div>
  );
}
