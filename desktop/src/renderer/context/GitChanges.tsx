import { ChevronDown, ChevronRight, RotateCw } from 'lucide-react';
import { useState } from 'react';
import type { GitFile } from '../../protocol/generated';
import { diffLines, statusOf, summarize, summaryText, type DiffLine, type GitState } from '../../state/git';
import { useGitDiff } from '../useGit';

/** A unified diff's lines, coloured by what each one is; shared by the git panel and a tool call's detail. */
export function DiffPre({ lines }: { lines: readonly DiffLine[] }) {
  return (
    <pre className="mono">
      {lines.map((l, i) => (
        <span key={i} className={`ctx-dl ctx-dl-${l.kind}`}>
          {l.text || ' '}
          {'\n'}
        </span>
      ))}
    </pre>
  );
}

/** One file's unified diff, read when it is opened, cut where the daemon cut it and saying so. */
function DiffView({ conversationId, file, moved }: { conversationId: string; file: GitFile; moved: unknown }) {
  const got = useGitDiff(conversationId, file.path, moved);
  if (!got) return <p className="ctx-diff-note tone-faint">lendo o diff…</p>;
  if (!got.ok) return <p className="ctx-diff-note tone-err">Não foi possível ler o diff: {got.why}</p>;
  const d = got.value;
  if (d.binary) return <p className="ctx-diff-note tone-dim">Arquivo binário: o git não mostra linhas dele.</p>;
  if (d.diff === '') return <p className="ctx-diff-note tone-dim">O git não mostrou diferença neste arquivo agora.</p>;
  return (
    <div className="ctx-diff" role="region" aria-label={`Diff de ${d.path}`}>
      <DiffPre lines={diffLines(d.diff)} />
      {d.truncated && (
        <p className="ctx-diff-note tone-warn">Cortado em {Math.round(d.limit / 1024)} KiB: o diff inteiro está no git da pasta.</p>
      )}
    </div>
  );
}

/**
 * What the folder holds that the last commit does not (D41), file by file with
 * its lines; opening one shows its diff. The folder's, not only this
 * conversation's: an edit by hand or another conversation in the same folder is
 * here too, and the files this conversation wrote are marked.
 */
export function GitChanges({
  conversationId,
  state,
  moved,
  wrote,
  onRefresh,
}: {
  conversationId: string;
  state: GitState;
  /** Changes when the conversation moves along: an open diff is read again. */
  moved: unknown;
  /** The paths this conversation's tools wrote, as they named them. */
  wrote: ReadonlySet<string>;
  onRefresh: () => void;
}) {
  const [open, setOpen] = useState<string | null>(null);
  const refresh = (
    <button type="button" className="ctx-refresh" title="Ler o git da pasta de novo" aria-label="Ler o git da pasta de novo" onClick={onRefresh}>
      <RotateCw size={11} aria-hidden />
    </button>
  );
  if (state.kind === 'idle' || state.kind === 'loading') return <p className="crew-quiet">Lendo o git da pasta…</p>;
  if (state.kind === 'failed') {
    return (
      <div className="ctx-sum">
        <span className="tone-err ctx-wrap">Não foi possível ler o git da pasta: {state.why}</span>
        {refresh}
      </div>
    );
  }
  const git = state.git;
  if (!git.repository) {
    return (
      <div className="ctx-sum">
        <span className="tone-warn ctx-wrap">A pasta não está num repositório git: não há diff a mostrar, só o que as ferramentas disseram ter escrito.</span>
        {refresh}
      </div>
    );
  }
  const sum = summarize(git.files);
  const isWrote = (p: string) => wrote.has(p) || [...wrote].some((w) => w.endsWith(`/${p}`) || p.endsWith(`/${w}`));
  return (
    <>
      <div className="ctx-sum">
        <span>
          <span className="ctx-sum-label">Na pasta, desde o último commit</span>
          <span className="mono ctx-sum-nums">{summaryText(sum)}</span>
        </span>
        {refresh}
      </div>
      {git.files.length > 0 && (
        <ul className="ctx-files">
          {git.files.map((f) => {
            const st = statusOf(f.status);
            const on = open === f.path;
            return (
              <li key={f.path} className={on ? 'on' : undefined}>
                <button type="button" className="ctx-file-row" aria-expanded={on} title={`${f.path} — ${st.word}`} onClick={() => setOpen(on ? null : f.path)}>
                  {on ? <ChevronDown size={11} aria-hidden /> : <ChevronRight size={11} aria-hidden />}
                  <span className={`ctx-status mono tone-${st.tone}`} aria-label={st.word}>
                    {st.letter}
                  </span>
                  <span className="ctx-path mono">{f.path}</span>
                  {isWrote(f.path) && (
                    <span className="ctx-wrote" title="Esta conversa escreveu neste arquivo">
                      ●
                    </span>
                  )}
                  <span className="ctx-nums mono">
                    {f.binary ? (
                      <span className="tone-faint">binário</span>
                    ) : (
                      <>
                        <span className="tone-ok">+{f.added}</span> <span className="tone-err">−{f.removed}</span>
                      </>
                    )}
                  </span>
                </button>
                {on && <DiffView conversationId={conversationId} file={f} moved={moved} />}
              </li>
            );
          })}
        </ul>
      )}
      {git.truncated && <p className="crew-quiet tone-warn">A lista foi cortada em {git.files.length} arquivos: há mais mudanças na pasta.</p>}
    </>
  );
}
