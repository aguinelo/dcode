import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { forgetMissing, markSeen } from '../state/attention';
import { capacityFor, gridShape, planGrid } from '../state/grid';
import { measureOf } from '../state/models';
import { paneView } from '../state/pane';
import { emptyPrefs, dropBefore, moveProject, parsePrefs, PREFS_KEY, relabel, setAllCollapsed, setCollapsed, type Prefs } from '../state/prefs';
import { emptySession } from '../state/session';
import type { SessionsState } from '../state/sessions';
import { countStates, projectsOf, rowOfSession, type Row } from '../state/sidebar';
import { CrewView } from './CrewView';
import { Grid } from './Grid';
import type { Host } from './host';
import { useNow, useReducedMotion, useTick, useWindowSize } from './hooks';
import { ModelMenu } from './ModelMenu';
import { Search } from './Search';
import { SessionPanel, type Peer } from './SessionPanel';
import { Sidebar } from './Sidebar';
import { notYet } from './text';
import { TopBar, type Look } from './TopBar';
import { useAttention } from './useAttention';
import { useModelLists } from './useModelLists';
import { useModelMenu } from './useModelMenu';
import { useWindowKeys } from './useWindowKeys';
import { useWindows } from './useWindows';
import type { DaemonView, Outcome, WindowActions } from './window';

interface Toast {
  id: number;
  text: string;
}

const TOAST_MS = 6000;

/** In the history of the stage, the grid's place. */
const GRID = '';

/** Which version of the screen, per window, as a convenience: the grid when unknown (D32). */
const LOOK_KEY = 'dcode.desktop.look.v1';

function loadLook(): Look {
  try {
    return window.localStorage.getItem(LOOK_KEY) === 'crew' ? 'crew' : 'grade';
  } catch {
    return 'grade';
  }
}

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

function sameIds(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((x, i) => x === b[i]);
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
  /** The sessions whose events the window holds: the grid's and the maximized one. */
  sessions: SessionsState;
  daemon: DaemonView;
  actions: WindowActions;
  /** Registers what to call with each notice that arrives after the first frame. */
  notices?: (say: (text: string) => void) => () => void;
  /** The conversation maximized at first; null opens on the grid. */
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
  const [look, setLook] = useState<Look>(loadLook);
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
  const places = useMemo(() => projects.filter((p) => p.id).map((p) => ({ id: p.id, label: p.label })), [projects]);
  const counts = useMemo(() => countStates(rows), [rows]);

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

  /** Maximizes a conversation, or puts the grid on the stage with null. */
  const show = useCallback(
    (id: string | null) => {
      if (id === selectedId) return;
      // Leaving one counts as having seen what it did while it was maximized.
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

  const win = useWindows({ actions, rowById, sessions, updateAttention, once, say, setDrafts, maximized: selectedId, maximize: show });
  const models = useModelMenu(actions, sessions, once, win.replace);

  // The grid, in stable places: the plan reads the places as last drawn, and
  // they are kept the way React keeps a value from the previous render. The
  // windows asking where a session opens take their places from the same room.
  const [placed, setPlaced] = useState<string[]>([]);
  const capacity = Math.max(0, capacityFor(size.width, size.height) - win.choosers.length);
  const plan = useMemo(() => planGrid(rows, attention, placed, capacity), [rows, attention, placed, capacity]);
  if (!sameIds(plan.ids, placed)) setPlaced(plan.ids);
  const onGridIds = useMemo(() => new Set(plan.ids), [plan.ids]);
  const windowIds = useMemo(() => [...plan.ids, ...win.choosers], [plan.ids, win.choosers]);

  // A window shows its conversation live: the grid's sessions are followed.
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

  const newSession = useCallback(() => void win.newSession(places.length), [places.length, win]);
  // Crew has no grid to ask "where?" in: ⌘N goes to the folder picker, and the
  // sessions pane's "Nova" lists the known projects (D33).
  const crewNew = useCallback(() => void win.startIn(null), [win]);
  const isConversation = useCallback((id: string) => rowById.has(id) || id in sessions.byId, [rowById, sessions]);

  useWindowKeys({
    ids: windowIds,
    cols: gridShape(windowIds.length).cols,
    focusId: win.focusId,
    setFocusId: win.setFocusId,
    caretTo: win.caretTo,
    sessions,
    maximized: selected,
    busy: searching || !!models.menu,
    setSearching,
    newSession: look === 'crew' ? crewNew : newSession,
    show,
    answer: win.answerIn,
    isConversation,
  });

  const peers: Peer[] = plan.ids
    .filter((id) => id !== selectedId)
    .flatMap((id) => {
      const r = rowById.get(id);
      return r ? [{ row: r, view: paneView(r, sessions.byId[id], now) }] : [];
    });

  const chooseLook = (next: Look) => {
    setLook(next);
    try {
      window.localStorage.setItem(LOOK_KEY, next);
    } catch (err) {
      say(`Não foi possível guardar a versão da tela (${(err as Error).message}); vale só até fechar.`);
    }
  };
  // Crew shows one conversation in the middle: the one in focus, else the first that wants attention.
  const crewId = win.focusId && isConversation(win.focusId) ? win.focusId : (plan.ids[0] ?? null);
  const crewRow = crewId ? (rowById.get(crewId) ?? (sessions.byId[crewId] ? rowOfSession(sessions.byId[crewId]) : null)) : null;

  const missing = (what: string) => say(notYet(what));
  const draftIn = (id: string, text: string) => setDrafts((d) => ({ ...d, [id]: text }));

  return (
    <div className={`window look-${look}${searching ? ' searching' : ''}`}>
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
        look={look}
        onLook={chooseLook}
      />
      <div className="window-main">
        {look === 'crew' ? (
          <CrewView
            row={crewRow}
            session={crewId ? sessions.byId[crewId] : undefined}
            projects={projects}
            places={places}
            sessions={sessions}
            now={now}
            verbTick={verbTick}
            draft={crewId ? (drafts[crewId] ?? '') : ''}
            caret={crewId && win.caret?.id === crewId ? win.caret.n : undefined}
            measure={crewRow ? measureFor(crewRow) : null}
            waiting={counts.blocked}
            look={look}
            daemon={daemon}
            actions={actions}
            onSectionError={say}
            on={{
              pick: (row) => (row.state === 'recorded' ? win.focus(row.id) : void win.open(row.id)),
              draft: draftIn,
              send: (id, text) => void win.sendIn(id, text),
              stop: win.stopIn,
              answer: win.answerIn,
              continueIn: (row, text) => void win.continueIn(row, text),
              model: models.open,
              missing,
              newSession: crewNew,
              startIn: (workspace) => void win.startIn(workspace),
              look: chooseLook,
            }}
          />
        ) : (
          <>
            <Sidebar
              projects={projects}
              selectedId={selectedId ?? win.focusId}
              onGridIds={onGridIds}
              gridShown={!selected}
              gridCount={plan.ids.length}
              now={now}
              userName={userName}
              onToggleAll={() => updatePrefs((p) => setAllCollapsed(p, shown, projects.some((x) => !x.collapsed)))}
              onShowGrid={() => show(null)}
              onNewSession={newSession}
              onSearch={() => setSearching(true)}
              onMissing={missing}
              actions={{
                toggle: (id) => updatePrefs((p) => setCollapsed(p, id, !p.collapsed[id])),
                rename: (id, label) => updatePrefs((p) => relabel(p, id, label)),
                move: (id, to) => updatePrefs((p) => moveProject(p, shown, id, to)),
                dropBefore: (id, target) => updatePrefs((p) => dropBefore(p, shown, id, target)),
                select: (id) => void win.open(id),
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
                  caret={win.caret?.id === selected.id ? win.caret.n : undefined}
                  onDraft={(text) => draftIn(selected.id, text)}
                  onAnswer={(decision) => selected.pendingApprovalId && win.answerIn(selected.id, selected.pendingApprovalId, decision)}
                  onSend={(text) => void win.sendIn(selected.id, text)}
                  onStop={() => win.stopIn(selected.id)}
                  onMissing={missing}
                  onGrid={() => show(null)}
                  onOpenPeer={(id) => (rowById.get(id)?.state === 'recorded' ? void win.open(id) : show(id))}
                  onModel={(anchor) => models.open(selectedRow ?? rowOfSession(selected), anchor)}
                  measure={measureFor(selectedRow ?? rowOfSession(selected))}
                />
              ) : (
                <Grid
                  plan={plan}
                  choosers={win.choosers}
                  opening={win.opening}
                  places={places}
                  rows={rowById}
                  sessions={sessions}
                  focusId={win.focusId}
                  caret={win.caret}
                  drafts={drafts}
                  now={now}
                  verbTick={verbTick}
                  measureFor={measureFor}
                  on={{
                    focus: win.focus,
                    maximize: (id) => show(id),
                    closer: win.closer,
                    draft: draftIn,
                    send: (id, text) => void win.sendIn(id, text),
                    stop: win.stopIn,
                    answer: win.answerIn,
                    continueIn: (row, text) => void win.continueIn(row, text),
                    model: models.open,
                    missing,
                  }}
                  chooser={{
                    place: (key, workspace) => void win.openIn(key, workspace),
                    otherFolder: (key) => void win.openIn(key, null),
                    close: win.dropChooser,
                  }}
                  onNew={newSession}
                  onSearch={() => setSearching(true)}
                />
              )}
            </main>
          </>
        )}
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
            void win.open(id);
          }}
          onClose={() => setSearching(false)}
        />
      )}
    </div>
  );
}
