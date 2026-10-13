import { useEffect, useState } from 'react';

/**
 * Copies through the main process inside the app — the renderer's own
 * clipboard API needs a permission the window never asks for — and through
 * the browser's in fixture mode. Answers why not, when it did not copy.
 */
export async function copyText(text: string): Promise<string | null> {
  const api = window.dcode;
  if (api) {
    const done = await api.copy(text);
    return done.ok ? null : done.refusal.message;
  }
  try {
    await navigator.clipboard.writeText(text);
    return null;
  } catch (err) {
    return err instanceof Error ? err.message : String(err);
  }
}

/**
 * Copiar and Rodar no terminal, on a command the agent proposed (D37). Shown
 * when the line is hovered or a button has the focus; out of the way, and out
 * of the layout, otherwise.
 */
export function CommandActions({ command, onRun }: { command: string; onRun?: (command: string) => void }) {
  const [copied, setCopied] = useState<{ ok: true } | { ok: false; why: string } | null>(null);
  useEffect(() => {
    if (!copied) return;
    const t = setTimeout(() => setCopied(null), 1600);
    return () => clearTimeout(t);
  }, [copied]);
  return (
    <span className="cmd-actions" onMouseDown={(e) => e.stopPropagation()}>
      <button
        type="button"
        className={`cmd-action${copied && !copied.ok ? ' tone-err' : ''}`}
        title={copied && !copied.ok ? `Não copiou: ${copied.why}` : 'Copiar o comando'}
        onClick={() => void copyText(command).then((why) => setCopied(why === null ? { ok: true } : { ok: false, why }))}
      >
        {copied ? (copied.ok ? 'Copiado' : 'Não copiou') : 'Copiar'}
      </button>
      {onRun && (
        <button
          type="button"
          className="cmd-action"
          title="Abre um terminal na pasta da sessão com o comando digitado — você revisa, edita e aperta Enter"
          onClick={() => onRun(command)}
        >
          Rodar no terminal
        </button>
      )}
    </span>
  );
}
