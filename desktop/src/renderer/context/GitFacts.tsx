import { GitBranch } from 'lucide-react';
import { stateWord } from '../../state/crew';
import { branchNow, rootIfElsewhere, shortId, startedElsewhere, type GitState } from '../../state/git';
import { shortPath, type Row } from '../../state/sidebar';
import { CopyButton } from './CopyButton';

/** The git half of the facts, in each state a read can be in — never a blank that reads as "no branch". */
function GitRows({ state, workspace, startedOn }: { state: GitState; workspace: string; startedOn: string | undefined }) {
  if (state.kind === 'idle' || state.kind === 'loading') {
    return (
      <>
        <dt>branch</dt>
        <dd className="tone-faint">lendo o git da pasta…</dd>
      </>
    );
  }
  if (state.kind === 'failed') {
    return (
      <>
        <dt>git</dt>
        <dd className="tone-err ctx-wrap">{state.why}</dd>
      </>
    );
  }
  const git = state.git;
  if (!git.repository) {
    return (
      <>
        <dt>git</dt>
        <dd className="tone-warn ctx-wrap">A pasta não está num repositório git: não há branch, nem diff, nem desfazer pelo git.</dd>
      </>
    );
  }
  const before = startedElsewhere(git, startedOn);
  const root = rootIfElsewhere(git, workspace);
  return (
    <>
      <dt>branch</dt>
      <dd className="mono">
        <GitBranch size={11} aria-hidden /> {branchNow(git)}
        {before && (
          <span className="tone-faint ctx-note" title="A sessão lê a branch uma vez, ao começar; a pasta mudou de branch depois.">
            {' '}
            · a sessão começou em {before}
          </span>
        )}
      </dd>
      <dt>worktree</dt>
      <dd className="mono" title={git.main_checkout}>
        {git.worktree ? (
          <>
            sim, de {git.main_checkout ? shortPath(git.main_checkout, 34) : 'outro checkout'}
            {git.main_checkout && <CopyButton text={git.main_checkout} what="a pasta do checkout principal" />}
          </>
        ) : (
          <span className="tone-dim">não — é o checkout principal</span>
        )}
      </dd>
      {root && (
        <>
          <dt>repositório</dt>
          <dd className="mono" title={root}>
            {shortPath(root, 34)}
          </dd>
        </>
      )}
      <dt>commit</dt>
      <dd className="mono">{git.head ?? <span className="tone-faint">nenhum ainda</span>}</dd>
    </>
  );
}

/**
 * Which conversation this is and where it works (D41): the session's id, live
 * or ended; the folder, shortened, whole on hover and on copy; and, from the
 * daemon, the branch now, whether the folder is a worktree and of which
 * checkout, and the commit.
 */
export function GitFacts({
  row,
  sessionState,
  startedOn,
  state,
}: {
  row: Row;
  /** The session's own state, when its events are here; else the list's. */
  sessionState: string | undefined;
  /** The branch the session read at its start (`Session.branch`). */
  startedOn: string | undefined;
  state: GitState;
}) {
  const now = sessionState ?? row.state;
  const live = now !== 'recorded';
  const liveWord = now === 'idle' ? 'esperando mensagem' : stateWord(now);
  return (
    <dl className="ctx-facts">
      <dt>sessão</dt>
      <dd className="mono" title={row.id}>
        <span className={live ? 'tone-accent' : 'tone-faint'} aria-hidden>
          {live ? '●' : '○'}
        </span>{' '}
        #{shortId(row.id)} <span className="tone-dim">· {live ? `viva, ${liveWord}` : 'terminada'}</span>
        <CopyButton text={row.id} what="o id da sessão" />
      </dd>
      <dt>pasta</dt>
      <dd className="mono" title={row.workspace}>
        {shortPath(row.workspace, 34)}
        <CopyButton text={row.workspace} what="o caminho da pasta" />
      </dd>
      <GitRows state={state} workspace={row.workspace} startedOn={startedOn} />
    </dl>
  );
}
