import { GitPullRequest, RotateCw } from 'lucide-react';
import { useState } from 'react';
import { createPortal } from 'react-dom';
import type { GitPRResponse, GitResponse } from '../../protocol/generated';
import type { PrDraft } from '../../shared/api';
import type { GitState } from '../../state/git';
import { cannotOpen, checksView, draftFor, draftProblem, hasOpenPR, prStateView, uncommittedNote, type PrState } from '../../state/pr';
import type { Entry } from '../../state/session';
import { SheetView } from '../CommandSheet';
import { useGitPR, type GitReader } from '../useGit';

/**
 * "Criar PR" (D60): every field the daemon will send, shown and editable —
 * title, body, base — beside the branch that is pushed, and nothing leaves
 * until the person presses Confirmar. The daemon pushes the branch to origin,
 * never by force, and opens the PR; it never merges.
 */
export function PrSheet({
  conversationId,
  start,
  head,
  note,
  reader,
  onOpened,
  onClose,
}: {
  conversationId: string;
  start: PrDraft;
  head: string;
  note: string | null;
  reader: GitReader;
  onOpened: (pr: GitPRResponse) => void;
  onClose: () => void;
}) {
  const [d, setD] = useState<PrDraft>(start);
  const [why, setWhy] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const problem = draftProblem(d);
  const confirm = async () => {
    if (problem || sending) return setWhy(problem);
    setSending(true);
    setWhy(null);
    const got = await reader.openGitPR(conversationId, d);
    setSending(false);
    if (!got.ok) return setWhy(got.why);
    onOpened(got.value);
    onClose();
  };
  return (
    <SheetView title="Criar PR" note={`${head} → ${d.base || '…'}`} onClose={sending ? () => {} : onClose}>
      <div className="start-form loop-sheet pr-sheet">
        <label className="start-field">
          <span className="start-label">Título</span>
          <input className="loop-input" value={d.title} autoFocus maxLength={256} onChange={(e) => setD({ ...d, title: e.target.value })} />
        </label>
        <label className="start-field">
          <span className="start-label">Corpo</span>
          <textarea className="loop-input pr-body" value={d.body} rows={9} onChange={(e) => setD({ ...d, body: e.target.value })} />
        </label>
        <div className="pr-branches">
          <label className="start-field">
            <span className="start-label">Base · para onde o PR aponta</span>
            <input className="loop-input mono" value={d.base} onChange={(e) => setD({ ...d, base: e.target.value })} />
          </label>
          <div className="start-field">
            <span className="start-label">Branch · empurrada para origin</span>
            <span className="loop-input mono pr-head">{head}</span>
          </div>
        </div>
        <label className="pr-draft">
          <input type="checkbox" checked={d.draft} onChange={(e) => setD({ ...d, draft: e.target.checked })} /> abrir como rascunho
        </label>
        <p className="sheet-empty">
          Confirmar faz <span className="mono">git push -u origin {head}</span> — nunca com força — e abre o PR com o gh. O app nunca faz merge nem liga
          auto-merge.
        </p>
        {note && <p className="sheet-empty tone-warn">{note}</p>}
        {(why ?? problem) && <p className={`sheet-empty ${why ? 'tone-err' : 'tone-dim'}`}>{why ?? problem}</p>}
        <div className="loop-sheet-actions">
          <button type="button" className="sheet-close" disabled={sending} onClick={onClose}>
            Cancelar
          </button>
          <button type="button" className="start-open" disabled={problem !== null || sending} onClick={() => void confirm()}>
            {sending ? 'Empurrando e abrindo…' : 'Confirmar'}
          </button>
        </div>
      </div>
    </SheetView>
  );
}

function Refresh({ onClick }: { onClick: () => void }) {
  return (
    <button type="button" className="ctx-refresh" title="Ler o PR de novo no GitHub" aria-label="Ler o PR de novo no GitHub" onClick={onClick}>
      <RotateCw size={11} aria-hidden />
    </button>
  );
}

/** The branch's PR as a line: number and title, linked; how it stands; its CI as a dot, a word and counts. */
function PrLine({ got, onRefresh }: { got: GitPRResponse; onRefresh: () => void }) {
  const pr = got.pr!;
  const st = prStateView(pr);
  const ci = checksView(pr.checks);
  return (
    <div className="pr-line" data-pr={pr.number} data-ci={pr.checks.state}>
      <a className="pr-link" href={pr.url} target="_blank" rel="noreferrer" title={`${pr.url} — abre no navegador`}>
        <GitPullRequest size={12} aria-hidden /> <span className="mono">#{pr.number}</span> <span className="pr-title">{pr.title}</span>
      </a>
      <Refresh onClick={onRefresh} />
      <span className="pr-meta">
        <span className="pr-ci" title={ci.detail}>
          <span className={`pr-dot tone-${ci.tone}`} aria-hidden>
            ●
          </span>{' '}
          <span className={`tone-${ci.tone}`}>{ci.word}</span>
          <span className="tone-faint"> · {ci.detail}</span>
        </span>
        <span className="tone-faint"> · </span>
        <span className={`tone-${st.tone}`}>{st.word}</span>
      </span>
    </div>
  );
}

/**
 * The branch's PR, beside what changed (D60): the PR with its CI when there is
 * one, else "Criar PR", which opens the confirmation sheet. What cannot be a
 * PR is said, with the daemon's reason, in the tone of whose it is.
 */
export function PullRequest({
  conversationId,
  title,
  entries,
  git,
}: {
  conversationId: string;
  /** The conversation's title: the draft's. */
  title: string;
  /** The conversation's events, folded: the draft's body. Undefined while they are not here. */
  entries: readonly Entry[] | undefined;
  git: GitState;
}) {
  const repo: GitResponse | null = git.kind === 'ready' ? git.git : null;
  const enabled = repo !== null && repo.repository;
  const { state, refresh, opened, reader } = useGitPR(conversationId, enabled);
  const [sheet, setSheet] = useState<PrDraft | null>(null);
  if (!enabled) return null;
  return (
    <div className="pr-box" aria-label="PR da branch">
      <PrBody state={state} repo={repo} onRefresh={refresh} onCreate={(got) => setSheet(draftFor(title, entries, got))} />
      {sheet &&
        reader &&
        state.kind === 'ready' &&
        createPortal(
          <PrSheet
          conversationId={conversationId}
          start={sheet}
          head={state.pr.branch}
          note={uncommittedNote(repo)}
          reader={reader}
          onOpened={opened}
          onClose={() => setSheet(null)}
          />,
          document.body,
        )}
    </div>
  );
}

export function PrBody({
  state,
  repo,
  onRefresh,
  onCreate,
}: {
  state: PrState;
  repo: GitResponse | null;
  onRefresh: () => void;
  onCreate: (got: GitPRResponse) => void;
}) {
  if (state.kind === 'idle' || state.kind === 'loading') return <p className="crew-quiet">Perguntando ao GitHub pelo PR da branch…</p>;
  if (state.kind === 'failed') {
    return (
      <div className="ctx-sum">
        <span className={`tone-${state.tone} ctx-wrap`}>PR: {state.why}</span>
        <Refresh onClick={onRefresh} />
      </div>
    );
  }
  const got = state.pr;
  const blocked = cannotOpen(repo, got);
  return (
    <>
      {got.pr && <PrLine got={got} onRefresh={onRefresh} />}
      {!hasOpenPR(got) && (
        <div className="ctx-sum">
          <span className="tone-dim ctx-wrap">{blocked ?? (got.pr ? 'O PR desta branch não está aberto; um novo pode ser aberto.' : `A branch ${got.branch} não tem PR.`)}</span>
          <span className="pr-actions">
            <button type="button" className="start-open pr-create" disabled={blocked !== null} title={blocked ?? undefined} onClick={() => onCreate(got)}>
              Criar PR
            </button>
            {!got.pr && <Refresh onClick={onRefresh} />}
          </span>
        </div>
      )}
    </>
  );
}
