import { Brain, Sparkles } from 'lucide-react';
import { useMemo } from 'react';
import { skillsSeen, type MemoryListView, type SkillsView } from '../../state/crew';
import type { SessionsState } from '../../state/sessions';
import { MEMORY_NOT_YET, SKILLS_NOT_YET } from '../text';
import type { WindowActions } from '../window';
import { NotYet, Page, PlacePicker, usePlace, useWorkspaceData, type Loaded, type Place } from './Page';

/** The answer that did not come: "not yet" when the daemon has no such list, a failure otherwise. */
function Missing({ why, notYet }: { why: string; notYet: string }) {
  if (why === notYet) return <NotYet text={why} />;
  return (
    <div className="crew-not-yet failed" role="alert">
      <strong>Não veio</strong>
      <span>{why}</span>
    </div>
  );
}

function Pending<T>({ got, notYet, children }: { got: Loaded<T>; notYet: string; children: (value: T) => React.ReactNode }) {
  if (got.state === 'loading') return <p className="crew-quiet">Perguntando ao daemon…</p>;
  if (got.state === 'no') return <Missing why={got.why} notYet={notYet} />;
  return <>{children(got.value)}</>;
}

function NoPlace() {
  return <p className="crew-quiet">Nenhum projeto conhecido ainda: abra uma sessão num projeto e ele aparece aqui.</p>;
}

function SkillList({ list }: { list: SkillsView }) {
  return (
    <>
      {!list.enabled && <p className="tone-warn">Skills desligadas neste projeto (`behavior.skills_enabled`): listadas, mas nenhuma entra num turno.</p>}
      {list.skills.length === 0 ? (
        <p className="crew-quiet">Nenhuma skill neste projeto nem nas do usuário.</p>
      ) : (
        <ul className="crew-cards">
          {list.skills.map((s) => (
            <li key={`${s.source}:${s.name}`} className="crew-card">
              <span className="crew-card-icon">
                <Sparkles size={15} aria-hidden />
              </span>
              <span className="crew-card-body">
                <strong>{s.name}</strong>
                <span>{s.whenToUse || '—'}</span>
                <span className="crew-card-meta">
                  {s.source === 'project' ? 'do projeto' : 'do usuário'}
                  {s.held && <span className="tone-warn"> · pede permissão antes de entrar</span>}
                </span>
              </span>
            </li>
          ))}
        </ul>
      )}
      {list.notices.map((n) => (
        <p key={n} className="tone-warn">
          {n}
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
      {ws ? (
        <Pending got={got} notYet={SKILLS_NOT_YET}>
          {(list) => <SkillList list={list} />}
        </Pending>
      ) : (
        <NoPlace />
      )}
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

function MemoryList({ list }: { list: MemoryListView }) {
  return (
    <>
      <p className="crew-quiet">
        <span className="mono">{list.path}</span>
        {!list.exists && ' — ainda não existe: nenhuma sessão aprendeu nada aqui.'}
        {list.exists && !list.enabled && ' — memória desligada neste projeto: listada, e nenhuma sessão a lê.'}
      </p>
      {list.entries.length > 0 && (
        <ul className="crew-cards two">
          {list.entries.map((m) => (
            <li key={`${m.kind}:${m.subject}`} className={`crew-card${m.shown ? '' : ' dim'}`}>
              <span className="crew-card-icon">
                <Brain size={15} aria-hidden />
              </span>
              <span className="crew-card-body">
                <strong>{m.subject}</strong>
                {m.body && <span>{m.body}</span>}
                <span className="crew-card-meta">
                  {m.kind}
                  {m.stale && <span className="tone-warn"> · o commit em que valia sumiu</span>}
                  {!m.shown && ' · fora do que a sessão lê'}
                </span>
              </span>
            </li>
          ))}
        </ul>
      )}
      {list.problems.map((p) => (
        <p key={p} className="tone-warn">
          {p}
        </p>
      ))}
    </>
  );
}

export function MemorySection({ places, preferred, actions }: { places: readonly Place[]; preferred: string | null; actions: WindowActions }) {
  const [ws, setWs] = usePlace(places, preferred);
  const got = useWorkspaceData(ws, actions.listMemory);
  return (
    <Page title="Memória" about="O que sessões anteriores aprenderam num projeto e as próximas leem." tools={<PlacePicker places={places} value={ws} onChange={setWs} />}>
      {ws ? (
        <Pending got={got} notYet={MEMORY_NOT_YET}>
          {(list) => <MemoryList list={list} />}
        </Pending>
      ) : (
        <NoPlace />
      )}
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
