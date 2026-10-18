import { ChevronLeft, ChevronRight } from 'lucide-react';
import { createContext, useContext, useState, type KeyboardEvent } from 'react';
import type * as P from '../protocol/generated';
import { AnswerOther } from '../protocol/generated';
import {
  blocked,
  choose,
  confirmBatch,
  currentItem,
  evidenceLine,
  goTo,
  isLast,
  isOpen,
  labelOf,
  needsPerson,
  NOTE_LIMIT,
  pickOf,
  recommendAll,
  resultsById,
  resultView,
  skip,
  startDraft,
  step,
  type CardDraft,
} from '../state/card';
import { answerRefusal } from '../state/questions';
import type { Outcome } from './window';

/** Sends a confirm of a card, and answers what became of each answer, or why the batch was not taken. */
export type AnswerCard = (batch: P.AnswerQuestionsRequest) => Promise<Outcome<P.AnswerResult[]>>;

/** What reaches the cards of a conversation without every component in between carrying it (D67). */
export const CardsContext = createContext<((sessionId: string, batch: P.AnswerQuestionsRequest) => Promise<Outcome<P.AnswerResult[]>>) | null>(null);

const KIND: Record<string, string> = { merge: 'merge', design: 'desenho', ceiling: 'teto' };

/** The evidence of a merge, compact: what the harness measured, never what the child said. */
function Evidence({ e }: { e: P.QuestionEvidence }) {
  const v = evidenceLine(e);
  const ok = (good: boolean) => (good ? 'tone-ok' : 'tone-err');
  return (
    <div className="qcard-evidence mono">
      <span className={ok(e.verdict === 'approve')}>{v.verdict}</span>
      <span className={ok(e.gate === 'green')}>{v.gate}</span>
      <span className="tone-dim">{v.command}</span>
      <span className="tone-faint">{v.commit}</span>
    </div>
  );
}

/** An item already answered, or a closed card's line: the question and the answer it has. */
function Answered({ item }: { item: P.CardItem }) {
  return (
    <li className="qcard-done">
      <span className="tone-ok" aria-hidden>
        ✓
      </span>
      <span className="qcard-done-q">{item.header || item.text}</span>
      <span className="qcard-done-a">{item.answer ? labelOf(item, item.answer) : 'respondida'}</span>
    </li>
  );
}

/**
 * The orchestrator's question card (.p §5.5, D67), inside the conversation:
 * one item at a time, "1 de N" and the arrows at the top right, the header
 * above the question, the options as radio rows — the selected one checked,
 * the recommended one said —, "Outro" for the person's words when the item
 * takes them, and "Pular" and "Próximo ⌘↵", which on the last item confirms
 * every answer in one POST. Nothing is optimistic: an item changes when the
 * daemon's card says so, and what became of each answer is said under it.
 */
export function QuestionCardView({
  card,
  answer,
  initial,
  results: firstResults,
}: {
  card: P.QuestionCard;
  /** Sends the confirm; absent, the card only shows (a recorded conversation, or no daemon). */
  answer?: AnswerCard;
  /** The draft to open on: for the fixture and the tests. */
  initial?: CardDraft;
  results?: Record<string, P.AnswerResult>;
}) {
  const [draft, setDraft] = useState<CardDraft>(() => initial ?? startDraft(card));
  const [results, setResults] = useState<Record<string, P.AnswerResult>>(firstResults ?? {});
  const [sending, setSending] = useState(false);
  const [why, setWhy] = useState<string | null>(null);

  const open = needsPerson(card);
  if (!open) {
    return (
      <section className="qcard closed" data-card-id={card.id} aria-label={card.title}>
        <header className="qcard-top">
          <span className="qcard-kind tone-faint">◆ {KIND[card.kind] ?? card.kind}</span>
          <span className="qcard-title-sm">{card.title}</span>
        </header>
        <ul className="qcard-done-list">{card.items.map((i) => (i.state === 'answered' ? <Answered key={i.id} item={i} /> : null))}</ul>
      </section>
    );
  }

  const item = currentItem(card, draft);
  if (!item) return null;
  const pick = pickOf(draft, item);
  const at = Math.min(draft.index, card.items.length - 1);
  const last = isLast(card, draft);
  const result = results[item.id];
  const said = result ? resultView(result, answerRefusal) : null;
  const canSend = !!answer && !sending;

  const send = (batch: P.AnswerQuestionsRequest) => {
    if (!answer || sending) return;
    setSending(true);
    setWhy(null);
    void answer(batch).then((out) => {
      setSending(false);
      if (!out.ok) {
        setWhy(out.why);
        return;
      }
      setResults(resultsById(out.value));
      // The daemon's card says what each item is now; the picks start again from its selections.
      setDraft((d) => ({ ...d, picks: {}, skipped: [] }));
    });
  };
  const confirm = () => {
    const stop = blocked(card, draft);
    if (stop) {
      setWhy(stop);
      return;
    }
    send(confirmBatch(card, draft));
  };
  const next = () => (last ? confirm() : setDraft((d) => goTo(card, d, d.index + 1)));
  const onSkip = () => {
    const skipped = skip(card, draft);
    if (!last) setDraft(skipped);
    else {
      setDraft(skipped);
      send(confirmBatch(card, skipped));
    }
  };
  const onKey = (e: KeyboardEvent<HTMLElement>) => {
    const inField = e.target instanceof HTMLInputElement;
    if (e.key === 'Escape') {
      // The card stays in the stream: Esc closes nothing, here or in the window behind it.
      e.stopPropagation();
      return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      e.stopPropagation();
      if (canSend || !last) next();
      return;
    }
    if (inField) return;
    const by = e.key === 'ArrowDown' ? 1 : e.key === 'ArrowUp' ? -1 : 0;
    if (by !== 0) {
      e.preventDefault();
      e.stopPropagation();
      setDraft((d) => step(card, d, by));
    }
  };

  const option = (id: string, label: string, description?: string) => {
    const checked = pick.answer === id;
    return (
      <li key={id}>
        <button
          type="button"
          role="radio"
          aria-checked={checked}
          className={checked ? 'qcard-option on' : 'qcard-option'}
          data-option={id}
          disabled={!isOpen(item) || sending}
          onClick={() => setDraft((d) => choose(card, d, id, id === AnswerOther ? pick.note : ''))}
        >
          <span className="qcard-radio" aria-hidden />
          <span className="qcard-option-text">
            <b>
              {label}
              {id === item.recommended ? ' (Recomendado)' : ''}
            </b>
            {description && <span className="qcard-option-desc">{description}</span>}
          </span>
        </button>
      </li>
    );
  };

  return (
    <div className="qcard-wrap">
      <section className="qcard" data-card-id={card.id} tabIndex={-1} aria-label={card.title} onKeyDown={onKey}>
        {card.proposed && card.from && (
          <div className="qcard-from">
            <span className="tone-faint">Você pediu:</span> {card.from}
            {card.why && <span className="qcard-from-why tone-dim">{card.why}</span>}
          </div>
        )}
        <header className="qcard-top">
          <span className="qcard-kind tone-warn">
            ◆ {KIND[card.kind] ?? card.kind}
            {item.header && <span className="qcard-header"> · {item.header}</span>}
          </span>
          <span className="qcard-pager">
            <button type="button" aria-label="Item anterior" disabled={at === 0} onClick={() => setDraft((d) => goTo(card, d, at - 1))}>
              <ChevronLeft size={13} aria-hidden />
            </button>
            <span className="qcard-count">
              {at + 1} de {card.items.length}
            </span>
            <button type="button" aria-label="Item seguinte" disabled={last} onClick={() => setDraft((d) => goTo(card, d, d.index + 1))}>
              <ChevronRight size={13} aria-hidden />
            </button>
          </span>
        </header>
        <div className="qcard-item" data-card-item={item.id} data-card-item-state={item.state}>
          <h3 className="qcard-title">{item.text}</h3>
          {item.summary && <p className="qcard-summary">{item.summary}</p>}
          {item.evidence && <Evidence e={item.evidence} />}
          {item.sensitive && item.sensitive.length > 0 && <p className="qcard-line tone-warn mono">sensível: {item.sensitive.join(', ')}</p>}
          {isOpen(item) ? (
            <ul className="qcard-options" role="radiogroup" aria-label={item.text}>
              {item.options.map((o) => option(o.id, o.label, o.description))}
              {item.allow_other && option(AnswerOther, 'Outro', 'Nas suas palavras: vão à orquestradora.')}
            </ul>
          ) : (
            <ul className="qcard-done-list">
              <Answered item={item} />
            </ul>
          )}
          {isOpen(item) && item.allow_other && pick.answer === AnswerOther && (
            <input
              className="qcard-other"
              type="text"
              value={pick.note}
              maxLength={NOTE_LIMIT}
              autoFocus
              placeholder="O que você quer que aconteça"
              aria-label={`Sua resposta a ${item.text}`}
              disabled={sending}
              onChange={(e) => setDraft((d) => choose(card, d, AnswerOther, e.target.value))}
            />
          )}
          {isOpen(item) && pick.answer !== AnswerOther && pick.note && <p className="qcard-line tone-dim">Nota que vai junto: {pick.note}</p>}
          {item.state === 'relayed' && <p className="qcard-line tone-dim">Suas palavras foram à orquestradora; a pergunta espera a proposta dela.</p>}
          {draft.skipped.includes(item.id) && <p className="qcard-line tone-faint">Pulada: continua esperando.</p>}
          {said && (
            <p className={`qcard-line tone-${said.tone}`} role={said.tone === 'err' ? 'alert' : undefined}>
              {said.text}
            </p>
          )}
        </div>
        {why && (
          <p className="qcard-line qcard-why tone-err" role="alert">
            {why}
          </p>
        )}
        <footer className="qcard-foot">
          <button type="button" className="qcard-all" disabled={!canSend} onClick={() => send(recommendAll(card))}>
            Aplicar a recomendação a todos
          </button>
          <span className="qcard-foot-right">
            <button type="button" className="qcard-skip" disabled={!isOpen(item) || sending || (last && !canSend)} onClick={onSkip}>
              Pular
            </button>
            {last ? (
              <button type="button" className="crew-primary qcard-next" data-card-confirm disabled={!canSend} onClick={confirm}>
                {sending ? 'enviando…' : 'Confirmar'} <span className="qcard-key">⌘↵</span>
              </button>
            ) : (
              <button type="button" className="crew-primary qcard-next" onClick={next}>
                Próximo <span className="qcard-key">⌘↵</span>
              </button>
            )}
          </span>
        </footer>
        {!answer && <p className="qcard-line tone-faint">Responder pede o daemon, e esta janela não fala com nenhum.</p>}
      </section>
      <span className="qcard-chip tone-warn">
        ◆ Precisa da sua contribuição · {card.pending} {card.pending === 1 ? 'pendente' : 'pendentes'}
      </span>
    </div>
  );
}

/** A card in a conversation's flow: it answers through the window, when the window answers (D67). */
export function CardEntry({ card, sessionId }: { card: P.QuestionCard; sessionId?: string }) {
  const send = useContext(CardsContext);
  return <QuestionCardView key={card.id} card={card} answer={send && sessionId ? (batch) => send(sessionId, batch) : undefined} />;
}
