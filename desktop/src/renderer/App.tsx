import { useCallback, useEffect, useMemo, useState } from 'react';
import { emptyPrefs, dropBefore, moveProject, parsePrefs, PREFS_KEY, relabel, setAllCollapsed, setCollapsed, type Prefs } from '../state/prefs';
import type { SessionsState } from '../state/sessions';
import { activeSessions, countStates, projectsOf } from '../state/sidebar';
import { CHOICES } from './ApprovalCard';
import type { Host } from './host';
import { useNow, useReducedMotion, useTick } from './hooks';
import { SessionPanel } from './SessionPanel';
import { Sidebar } from './Sidebar';
import { NOT_CONNECTED, notYet } from './text';

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

function isEditable(el: Element | null): boolean {
  return !!el && (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement || (el as HTMLElement).isContentEditable);
}

export function App({
  host,
  sessions,
  daemonVersion,
  initialSelection,
}: {
  host: Host;
  sessions: SessionsState;
  daemonVersion: string;
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

  const say = useCallback((text: string) => {
    setToasts((t) => [...t.filter((x) => x.text !== text), { id: Date.now() + Math.random(), text }]);
  }, []);

  useEffect(() => {
    host.user().then(
      (u) => setUserName(u.name),
      (err: unknown) => say(`Não foi possível saber o nome de quem usa: ${(err as Error).message ?? String(err)}`),
    );
  }, [host, say]);

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

  const projects = useMemo(() => projectsOf(sessions, prefs), [sessions, prefs]);
  const shown = useMemo(() => projects.map((p) => p.id), [projects]);
  const counts = useMemo(() => countStates(Object.values(sessions.byId)), [sessions]);
  const selected = selectedId ? sessions.byId[selectedId] ?? null : null;

  const select = useCallback(
    (id: string) => {
      if (id === selectedId) return;
      setHistory((h) => ({ back: selectedId ? [...h.back, selectedId] : h.back, forward: [] }));
      setSelectedId(id);
    },
    [selectedId],
  );

  const answer = useCallback(() => say(NOT_CONNECTED), [say]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && (e.key === 'n' || e.key === 'k')) {
        e.preventDefault();
        say(notYet(e.key === 'n' ? 'Nova sessão' : 'Procurar'));
        return;
      }
      if (mod && /^[1-9]$/.test(e.key)) {
        const target = activeSessions(projects)[Number(e.key) - 1];
        if (target) {
          e.preventDefault();
          select(target.id);
        }
        return;
      }
      if (!selected?.pendingApprovalId || isEditable(document.activeElement) || mod) return;
      const choice = CHOICES.find((c) => c.key === e.key) ?? (e.key === 'Escape' ? CHOICES[2] : undefined);
      if (choice) {
        e.preventDefault();
        answer();
      } else if (e.key === 'Enter') {
        e.preventDefault();
        say('↵ não responde à aprovação nesta versão: use 1, 2, 3 ou esc.');
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [projects, select, selected, answer, say]);

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
          onMissing={(what) => say(notYet(what))}
          actions={{
            toggle: (id) => updatePrefs((p) => setCollapsed(p, id, !p.collapsed[id])),
            rename: (id, label) => updatePrefs((p) => relabel(p, id, label)),
            move: (id, to) => updatePrefs((p) => moveProject(p, shown, id, to)),
            dropBefore: (id, target) => updatePrefs((p) => dropBefore(p, shown, id, target)),
            select,
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
              onSend={() => say(NOT_CONNECTED)}
              onStop={() => say(NOT_CONNECTED)}
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
        <span
          className="status"
          title={`Eventos gravados de um daemon v${daemonVersion}. Esta versão ainda não se conecta ao daemon.`}
        >
          <span className="status-dot replay" />
          gravação
        </span>
        <span className="tone-faint">v{daemonVersion}</span>
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
