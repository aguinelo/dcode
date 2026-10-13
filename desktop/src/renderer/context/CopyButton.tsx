import { Check, Copy } from 'lucide-react';
import { useEffect, useState } from 'react';
import { copyText } from '../CommandActions';

/**
 * A small copy button beside a fact that is shown shortened — a session's id,
 * a folder's path. It says when it did not copy, and why, on the button itself.
 */
export function CopyButton({ text, what }: { text: string; what: string }) {
  const [done, setDone] = useState<{ ok: true } | { ok: false; why: string } | null>(null);
  useEffect(() => {
    if (!done) return;
    const t = setTimeout(() => setDone(null), 1600);
    return () => clearTimeout(t);
  }, [done]);
  const failed = done && !done.ok;
  return (
    <button
      type="button"
      className={`ctx-copy${failed ? ' tone-err' : ''}`}
      aria-label={`Copiar ${what}`}
      title={failed ? `Não copiou: ${done.why}` : done ? 'Copiado' : `Copiar ${what}: ${text}`}
      onClick={(e) => {
        e.stopPropagation();
        void copyText(text).then((why) => setDone(why === null ? { ok: true } : { ok: false, why }));
      }}
    >
      {done?.ok ? <Check size={11} aria-hidden /> : <Copy size={11} aria-hidden />}
    </button>
  );
}
