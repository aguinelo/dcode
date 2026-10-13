import { FitAddon } from '@xterm/addon-fit';
import { Terminal, type ITheme } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { useEffect, useRef, useState } from 'react';
import { emptyTranscript, feed, pendingResult, resultMessage, type Run, type Transcript } from '../state/terminal';

// The conversation's terminal (D37): a real shell in the session's folder, in
// a PTY the main process holds, with the command the agent proposed typed and
// not run. The person reads it, edits it — adds sudo — and presses Enter.
// When the shell says the command ended, the window offers to tell the agent
// what came of it; nothing goes without the person's click, and what would
// go is shown first.

/** The terminal's colours, from the window's tokens: it follows the theme the window has. */
function themeOf(el: Element): ITheme {
  const css = getComputedStyle(el);
  const v = (name: string) => css.getPropertyValue(name).trim();
  return {
    background: v('--dc-sunken'),
    foreground: v('--dc-text'),
    cursor: v('--dc-accent'),
    cursorAccent: v('--dc-sunken'),
    selectionBackground: v('--dc-term-selection'),
    black: v('--dc-term-black'),
    red: v('--dc-err'),
    green: v('--dc-ok'),
    yellow: v('--dc-warn'),
    blue: v('--dc-term-blue'),
    magenta: v('--dc-term-magenta'),
    cyan: v('--dc-term-cyan'),
    white: v('--dc-dim'),
    brightBlack: v('--dc-faint'),
    brightRed: v('--dc-err'),
    brightGreen: v('--dc-ok'),
    brightYellow: v('--dc-warn'),
    brightBlue: v('--dc-term-blue'),
    brightMagenta: v('--dc-term-magenta'),
    brightCyan: v('--dc-term-cyan'),
    brightWhite: v('--dc-emph'),
  };
}

type Phase = { kind: 'opening' } | { kind: 'open'; id: string; cwd: string } | { kind: 'failed'; why: string } | { kind: 'ended'; code: number };

export function TerminalDock({
  sessionId,
  command,
  compact,
  onSend,
  onClose,
}: {
  sessionId: string;
  /** What the agent proposed, typed on the shell's line when it opens. */
  command: string;
  compact: boolean;
  /** Sends a message to the conversation: a turn when idle, a correction while it runs. */
  onSend: (text: string) => void;
  onClose: () => void;
}) {
  const host = useRef<HTMLDivElement>(null);
  const [phase, setPhase] = useState<Phase>({ kind: 'opening' });
  const transcript = useRef<Transcript>(emptyTranscript);
  /** A result offered to the agent, shown as it would go. */
  const [offer, setOffer] = useState<{ run: Run; text: string; sent: boolean } | null>(null);

  useEffect(() => {
    const api = window.dcode;
    const el = host.current;
    if (!api || !el) return;
    const term = new Terminal({
      fontFamily: getComputedStyle(el).getPropertyValue('--dc-mono').trim() || 'monospace',
      fontSize: compact ? 11 : 12,
      cursorBlink: true,
      scrollback: 5000,
      theme: themeOf(el),
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(el);
    fit.fit();

    let id: string | null = null;
    let lastSeen: Run | null = null;
    const early: Array<Parameters<Parameters<typeof api.onTerminal>[0]>[0]> = [];
    const take = (push: (typeof early)[number]): void => {
      if (push.kind === 'exit') {
        setPhase({ kind: 'ended', code: push.code });
        return;
      }
      term.write(push.data);
      transcript.current = feed(transcript.current, push.data);
      const last = transcript.current.last;
      if (last && last !== lastSeen) {
        lastSeen = last;
        setOffer({ run: last, text: resultMessage(last), sent: false });
      }
    };
    const off = api.onTerminal((push) => {
      if (id === null) early.push(push);
      else if (push.id === id) take(push);
    });
    const typing = term.onData((data) => {
      if (id !== null) api.writeTerminal(id, data);
    });
    void api.openTerminal(sessionId, command, term.cols, term.rows).then((opened) => {
      if (!opened.ok) {
        setPhase({ kind: 'failed', why: opened.refusal.message });
        return;
      }
      id = opened.value.id;
      setPhase({ kind: 'open', id, cwd: opened.value.cwd });
      for (const push of early.splice(0)) if (push.id === id) take(push);
      term.focus();
    });

    const sized = new ResizeObserver(() => {
      fit.fit();
      if (id !== null) api.resizeTerminal(id, term.cols, term.rows);
    });
    sized.observe(el);
    // The theme is the window's: when it changes, so does the terminal.
    const themed = new MutationObserver(() => {
      term.options.theme = themeOf(el);
    });
    themed.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme', 'class'] });

    return () => {
      sized.disconnect();
      themed.disconnect();
      typing.dispose();
      off();
      if (id !== null) api.killTerminal(id);
      term.dispose();
    };
    // One terminal per command: a new command is a new dock (keyed by the caller).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const offerNow = () => {
    const run = pendingResult(transcript.current, command);
    setOffer({ run, text: resultMessage(run), sent: false });
  };

  const where = phase.kind === 'open' ? phase.cwd : '';
  return (
    <div className={`term-dock${compact ? ' compact' : ''}`} data-terminal={phase.kind === 'open' ? phase.id : undefined}>
      <div className="term-head">
        <span className="term-title">Terminal</span>
        {where && (
          <span className="term-where" title={where}>
            {where}
          </span>
        )}
        {phase.kind === 'opening' && <span className="term-state">abrindo…</span>}
        {phase.kind === 'ended' && <span className="term-state">o shell terminou (código {phase.code})</span>}
        <span className="spacer" />
        {(phase.kind === 'open' || phase.kind === 'ended') && (
          <button type="button" className="term-btn" onClick={offerNow} title="Mostra o que vai ao agente antes de enviar">
            Enviar resultado ao agente
          </button>
        )}
        <button type="button" className="icon-btn" aria-label="Fechar o terminal" title="Fechar o terminal (encerra o shell)" onClick={onClose}>
          ×
        </button>
      </div>
      {!window.dcode && <div className="term-failed">O terminal só existe dentro do app: aqui não há shell para abrir.</div>}
      {phase.kind === 'failed' && <div className="term-failed">O terminal não abriu: {phase.why}</div>}
      {offer && (
        <div className="term-offer" data-offer-code={offer.run.code ?? 'none'}>
          <div className="term-offer-head">
            {offer.sent ? (
              <span className="tone-ok">Mandado à conversa: aparece no fluxo quando o daemon o recebe.</span>
            ) : (
              <span>
                {offer.run.code === null ? 'Enviar ao agente o que o terminal mostra até agora?' : `O comando terminou (código ${offer.run.code}). Enviar ao agente?`}
              </span>
            )}
            <span className="spacer" />
            {!offer.sent && (
              <button
                type="button"
                className="term-btn primary"
                onClick={() => {
                  onSend(offer.text);
                  setOffer({ ...offer, sent: true });
                }}
              >
                Enviar
              </button>
            )}
            <button type="button" className="term-btn" onClick={() => setOffer(null)}>
              {offer.sent ? 'Ok' : 'Descartar'}
            </button>
          </div>
          {!offer.sent && <pre className="term-preview">{offer.text}</pre>}
        </div>
      )}
      <div className="term-screen" ref={host} />
    </div>
  );
}
