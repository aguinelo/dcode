import { Brain, Sparkles } from 'lucide-react';
import { useMemo } from 'react';
import type { MemoryResponse, SkillsResponse } from '../../protocol/crew';
import { memoryGroups, skillsSeen } from '../../state/crew';
import type { SessionsState } from '../../state/sessions';
import { NOT_EXPOSED } from '../text';
import type { WindowActions } from '../window';
import { NotYet, Page, PlacePicker, usePlace, useWorkspaceData, type Loaded, type Place } from './Page';

/** The answer that did not come: "not yet" from a daemon older than the route, a failure otherwise. */
function Missing({ why }: { why: string }) {
  if (why.startsWith(NOT_EXPOSED)) return <NotYet text={why} />;
  return (
    <div className="crew-not-yet failed" role="alert">
      <strong>Não veio</strong>
      <span>{why}</span>
    </div>
  );
}

function Pending<T>({ got, children }: { got: Loaded<T>; children: (value: T) => React.ReactNode }) {
  if (got.state === 'loading') return <p className="crew-quiet">Perguntando ao daemon…</p>;
  if (got.state === 'no') return <Missing why={got.why} />;
  return <>{children(got.value)}</>;
}

function NoPlace() {
  return <p className="crew-quiet">Nenhum projeto conhecido ainda: abra uma sessão num projeto e ele aparece aqui.</p>;
}

function SkillList({ list }: { list: SkillsResponse }) {
  return (
    <>
      {!list.enabled && (
        <p className="tone-warn">Skills desligadas neste projeto (behavior.skills_enabled): listadas, mas nenhuma entra num turno.</p>
      )}
      {list.skills.length === 0 ? (
        <p className="crew-quiet">Nenhuma skill neste projeto nem nas do usuário.</p>
      ) : (
        <ul className="crew-cards">
          {list.skills.map((s) => (
            <li key={`${s.source}:${s.name}`} className="crew-card" data-skill={s.name}>
              <span className="crew-card-icon">
                <Sparkles size={15} aria-hidden />
              </span>
              <span className="crew-card-body">
                <strong>{s.name}</strong>
                <span>{s.when_to_use || '—'}</span>
                <span className="crew-card-meta">
                  {s.source === 'project' ? 'do projeto' : 'do usuário'} · <span className="mono">{s.path}</span>
                  {s.triggers && s.triggers.length > 0 && <> · gatilhos: {s.triggers.join(', ')}</>}
                </span>
                {s.held && (
                  <span className="tone-warn crew-card-meta">
                    Pede permissão antes de entrar{s.claims && s.claims.length > 0 ? `: ${s.claims.join('; ')}` : ''}
                  </span>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
      {list.notices.map((n) => (
        <p key={`${n.source}:${n.path}`} className="tone-warn crew-notice">
          <span className="mono">{n.path}</span> ({n.source === 'project' ? 'do projeto' : 'do usuário'}): {n.reason}
        </p>
      ))}
    </>
  );
}

export function SkillsSection({ places, preferred, sessions, actions }: { places: readonly Place[]; preferred: string | null; sessions: SessionsState; actions: WindowActions }) {
  const [ws, setWs] = usePlace(places, preferred);
  const got = useWorkspaceData(ws, actions.listSkills);
  const seen = useMemo(() => skillsSeen(sessions), [sessions]);
  return (
    <Page title="Skills" about="Arquivos que ensinam o agente a trabalhar, do projeto e do usuário." tools={<PlacePicker places={places} value={ws} onChange={setWs} />}>
      {ws ? <Pending got={got}>{(list) => <SkillList list={list} />}</Pending> : <NoPlace />}
      <h2 className="crew-page-sub">Entraram nas conversas abertas</h2>
      {seen.length === 0 ? (
        <p className="crew-quiet">Nenhuma skill entrou num turno das conversas que esta janela acompanha.</p>
      ) : (
        <ul className="crew-cards">
          {seen.map((s) => (
            <li key={s.name} className="crew-card">
              <span className="crew-card-icon">
                <Sparkles size={15} aria-hidden />
              </span>
              <span className="crew-card-body">
                <strong>{s.name}</strong>
                <span>{s.whenToUse || '—'}</span>
                <span className="crew-card-meta">
                  em {s.sessions} {s.sessions === 1 ? 'conversa' : 'conversas'}
                </span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </Page>
  );
}

function MemoryList({ list }: { list: MemoryResponse }) {
  const groups = memoryGroups(list.entries);
  return (
    <>
      <p className="crew-quiet">
        <span className="mono">{list.path}</span>
        {!list.exists
          ? ' — ainda não existe: nenhuma sessão aprendeu nada aqui.'
          : list.enabled
            ? ` — memória ligada; até ${list.max_entries} entram numa sessão, as mais recentes.`
            : ' — memória desligada neste projeto (memory.enabled): listada, e nenhuma sessão a lê.'}
      </p>
      {list.unreadable && <p className="tone-err">O arquivo não pôde ser lido, e as sessões abrem sem memória: {list.unreadable}</p>}
      {groups.map((g) => (
        <section key={g.kind} className="crew-memory-group">
          <h2 className="crew-page-sub">
            {g.label} <span className="crew-count">{g.entries.length}</span>
          </h2>
          <ul className="crew-cards two">
            {g.entries.map((m, i) => (
              <li key={`${m.subject}:${i}`} className={`crew-card${m.shown ? '' : ' dim'}`} data-memory={m.subject}>
                <span className="crew-card-icon">
                  <Brain size={15} aria-hidden />
                </span>
                <span className="crew-card-body">
                  <strong>{m.subject}</strong>
                  {m.body && <span>{m.body}</span>}
                  <span className="crew-card-meta">
                    {[m.learned && `aprendida em ${m.learned}`, m.commit && `no ${m.commit.slice(0, 7)}`].filter(Boolean).join(' · ') || 'escrita à mão'}
                    {!m.shown && ' · fora do que a sessão lê'}
                  </span>
                  {m.stale && <span className="tone-warn crew-card-meta">O commit em que valia não está mais no repositório.</span>}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ))}
      {list.exists && list.entries.length === 0 && !list.unreadable && <p className="crew-quiet">O arquivo existe e não tem nenhuma memória.</p>}
      {list.malformed.length > 0 && (
        <>
          <h2 className="crew-page-sub tone-warn">Blocos que não são memória</h2>
          {list.malformed.map((b, i) => (
            <p key={i} className="tone-warn crew-notice">
              <span className="mono">{b.line}</span> — {b.reason}
            </p>
          ))}
        </>
      )}
    </>
  );
}

export function MemorySection({ places, preferred, actions }: { places: readonly Place[]; preferred: string | null; actions: WindowActions }) {
  const [ws, setWs] = usePlace(places, preferred);
  const got = useWorkspaceData(ws, actions.listMemory);
  return (
    <Page title="Memória" about="O que sessões anteriores aprenderam num projeto e as próximas leem." tools={<PlacePicker places={places} value={ws} onChange={setWs} />}>
      {ws ? <Pending got={got}>{(list) => <MemoryList list={list} />}</Pending> : <NoPlace />}
    </Page>
  );
}

/** A section the daemon has nothing for yet: its name, and why it is empty. */
export function NotYetSection({ title, about, text }: { title: string; about: string; text: string }) {
  return (
    <Page title={title} about={about}>
      <NotYet text={text} />
    </Page>
  );
}
