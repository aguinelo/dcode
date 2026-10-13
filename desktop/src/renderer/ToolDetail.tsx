import { useState } from 'react';
import { plural } from '../state/format';
import type { ToolDetail as Detail } from '../state/tooldetail';
import { DiffPre } from './context/GitChanges';

/**
 * What a model's tool call did, under its line once opened (D45): the diff of
 * a change, the output of anything else, a failure's error whole. Every cut is
 * said — the preview's, the tool's and the daemon's.
 */
export function ToolDetail({ id, detail, label }: { id: string; detail: Detail; label: string }) {
  const [all, setAll] = useState(false);
  const out = detail.output;
  return (
    <div id={id} className="tool-detail" role="region" aria-label={label}>
      {out && (
        <>
          <pre className={`tool-output${detail.failed ? ' failed' : ''}`}>{all || !out.long ? out.text : out.preview}</pre>
          {out.long && (
            <button type="button" className="tool-detail-more" aria-expanded={all} onClick={() => setAll(!all)}>
              {all ? 'mostrar menos' : `mostrar tudo · ${plural(out.totalLines, 'linha', 'linhas')}`}
            </button>
          )}
          {out.cutByTool && <p className="tool-detail-note tone-warn">A ferramenta cortou esta saída antes de devolvê-la: o que falta não chegou ao daemon.</p>}
        </>
      )}
      {detail.diff && (
        <>
          <div className="ctx-diff tool-diff" role="region" aria-label="Diff desta chamada">
            <DiffPre lines={detail.diff.lines} />
          </div>
          {detail.diff.cutAt !== null && (
            <p className="tool-detail-note tone-warn">Diff cortado em {detail.diff.cutAt} linhas: o que o arquivo tem agora, contra o último commit, está no painel do git.</p>
          )}
        </>
      )}
    </div>
  );
}
