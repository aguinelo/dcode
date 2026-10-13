import { GitBranch } from 'lucide-react';
import { branchNow, shortId, summarize, worktreeWord, type GitState } from '../../state/git';
import { shortPath, type Row } from '../../state/sidebar';

/**
 * One line under a conversation's title (D41): its id, its folder, the branch
 * now, whether the folder is a worktree, and how much changed in it — each
 * whole on hover. Git that could not be read says so here too, short, with the
 * reason on hover.
 */
export function ContextLine({ row, state }: { row: Row; state: GitState }) {
  const parts: React.ReactNode[] = [
    <span key="id" title={`Sessão ${row.id}`}>
      #{shortId(row.id)}
    </span>,
    <span key="dir" title={row.workspace}>
      {shortPath(row.workspace, 28)}
    </span>,
  ];
  if (state.kind === 'failed') {
    parts.push(
      <span key="git" className="tone-err" title={state.why}>
        git ilegível
      </span>,
    );
  } else if (state.kind === 'ready' && !state.git.repository) {
    parts.push(
      <span key="git" className="tone-warn" title="A pasta não está num repositório git">
        sem git
      </span>,
    );
  } else if (state.kind === 'ready') {
    const git = state.git;
    parts.push(
      <span key="branch" title={git.detached ? 'HEAD destacado' : `branch ${git.branch ?? ''}`}>
        <GitBranch size={10} aria-hidden /> {branchNow(git)}
      </span>,
    );
    const wt = worktreeWord(git);
    if (wt) {
      parts.push(
        <span key="wt" title={git.main_checkout ? `worktree ligada de ${git.main_checkout}` : 'worktree ligada'}>
          {wt}
        </span>,
      );
    }
    const sum = summarize(git.files);
    parts.push(
      sum.files === 0 ? (
        <span key="sum" className="tone-faint">
          sem mudanças
        </span>
      ) : (
        <span key="sum" title={`${sum.files} ${sum.files === 1 ? 'arquivo mudado' : 'arquivos mudados'} desde o último commit`}>
          <span className="tone-ok">+{sum.added}</span> <span className="tone-err">−{sum.removed}</span>
        </span>
      ),
    );
  }
  return (
    <span className="ctx-line mono">
      {parts.map((p, i) => (
        <span key={i} className="ctx-line-part">
          {i > 0 && <span className="ctx-sep">·</span>}
          {p}
        </span>
      ))}
    </span>
  );
}
