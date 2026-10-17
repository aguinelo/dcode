import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { recordedPRs } from '../../src/fixtures/git';
import { decodeGitPR } from '../../src/protocol/git';
import { PrBody, PrSheet } from '../../src/renderer/context/PullRequest';
import { fixtureActions } from '../../src/renderer/fixture';
import { prOutcome } from '../../src/renderer/live';
import type { GitReader } from '../../src/renderer/useGit';
import { NOT_CONNECTED } from '../../src/renderer/text';
import { cannotOpen, checksView, draftFor, draftProblem, failureTone, prStateView, uncommittedNote, PR_BODY_LIMIT, PR_TITLE_LIMIT } from '../../src/state/pr';
import { log, done } from './helpers';

// A conversation's branch on GitHub (D60): its PR and CI, said in words, and
// "Criar PR", which only ever sends what a person confirmed in the sheet.

const checks = (state: string, passed = 0, failed = 0, pending = 0, skipped = 0): P.GitChecks => ({ state, passed, failed, pending, skipped });
const pr = (over: Partial<P.GitPR> = {}): P.GitPR => ({
  number: 7,
  url: 'https://github.com/o/r/pull/7',
  title: 'Add b',
  state: 'open',
  draft: false,
  base: 'main',
  head: 'feature',
  checks: checks('pass', 3),
  ...over,
});
const tree = (over: Partial<P.GitResponse> = {}): P.GitResponse => ({
  repository: true,
  branch: 'feature',
  detached: false,
  worktree: false,
  files: [],
  truncated: false,
  ...over,
});

describe('a PR’s checks, in words', () => {
  it('is never green over a red check, and says the counts behind the word', () => {
    expect(checksView(checks('pass', 3, 0, 0, 1))).toEqual({ tone: 'ok', word: 'CI verde', detail: '3 de 4 passaram · 1 pulada' });
    expect(checksView(checks('fail', 2, 1, 1))).toMatchObject({ tone: 'err', word: 'CI vermelho', detail: '1 de 4 falhou' });
    expect(checksView(checks('pending', 2, 0, 1))).toMatchObject({ tone: 'dim', word: 'CI rodando', detail: '1 de 3 ainda roda' });
    expect(checksView(checks('none'))).toMatchObject({ tone: 'faint', word: 'sem CI', detail: 'nenhuma checagem neste PR' });
    expect(checksView(checks('purple'))).toMatchObject({ tone: 'err' });
  });

  it('says how the PR stands', () => {
    expect(prStateView(pr())).toEqual({ tone: 'accent', word: 'aberto' });
    expect(prStateView(pr({ draft: true }))).toEqual({ tone: 'dim', word: 'rascunho' });
    expect(prStateView(pr({ state: 'merged' }))).toEqual({ tone: 'dim', word: 'merged' });
    expect(prStateView(pr({ state: 'closed' }))).toEqual({ tone: 'dim', word: 'fechado' });
    expect(prStateView(pr({ state: 'locked' })).tone).toBe('err');
  });

  // D61, after D52: the waiting tone is for what waits for you, and green and
  // red are for what was measured. CI that still runs waits for no one, and a
  // merged PR is history, not a measurement.
  it('keeps the waiting tone and green off what is neither waiting for you nor measured', () => {
    expect(checksView(checks('pending', 0, 0, 2)).tone).not.toBe('warn');
    expect(checksView(checks('pass', 2)).tone).toBe('ok');
    expect(checksView(checks('fail', 1, 1)).tone).toBe('err');
    expect(prStateView(pr({ state: 'merged' })).tone).not.toBe('ok');
  });
});

describe('what "Criar PR" opens with', () => {
  const got: P.GitPRResponse = { branch: 'feature', base: 'main' };

  it('starts from the conversation: its title, its last answer and the files it wrote', () => {
    const l = log()
      .add('turn.started', { turn_id: 't1', text: 'antes' })
      .add('message.delta', { turn_id: 't1', text: 'velho' })
      .add('turn.completed', { turn_id: 't1', reason: 'done' })
      .add('turn.started', { turn_id: 't2', text: 'faça' })
      .add('tool.requested', { turn_id: 't2', tool_call_id: 'c1', name: 'write', input: { path: 'b.txt', content: 'x' } })
      .add('tool.completed', done('c1', { added: 2, removed: 0 }))
      .add('message.delta', { turn_id: 't2', text: 'Adicionei ' })
      .add('message.delta', { turn_id: 't2', text: 'o b.' })
      .add('turn.completed', { turn_id: 't2', reason: 'done' });
    const d = draftFor('Adicionar b\nsegunda linha', l.fold().entries, got);
    expect(d.title).toBe('Adicionar b');
    expect(d.body).toBe('Adicionei o b.\n\nArquivos que esta conversa mudou:\n\n- `b.txt` (+2 −0)');
    expect(d).toMatchObject({ base: 'main', draft: false });
  });

  it('keeps within GitHub’s caps', () => {
    const d = draftFor('x'.repeat(PR_TITLE_LIMIT + 10), undefined, got);
    expect([...d.title]).toHaveLength(PR_TITLE_LIMIT);
    expect(d.body).toBe('');
  });

  it('refuses what the daemon would refuse, and says why', () => {
    const ok = { title: 'a', body: '', base: 'main', draft: false };
    expect(draftProblem(ok)).toBeNull();
    expect(draftProblem({ ...ok, title: '  ' })).toContain('título');
    expect(draftProblem({ ...ok, title: 'a\nb' })).toContain('uma linha');
    expect(draftProblem({ ...ok, body: 'x'.repeat(PR_BODY_LIMIT + 1) })).toContain('bytes');
    expect(draftProblem({ ...ok, base: '' })).toContain('base');
    expect(draftProblem({ ...ok, base: '-x' })).toContain('não é nome de branch');
  });

  it('cannot be pressed on a detached head or on the base itself', () => {
    expect(cannotOpen(tree(), got)).toBeNull();
    expect(cannotOpen(tree({ detached: true }), got)).toContain('destacado');
    expect(cannotOpen(tree(), { branch: 'main', base: 'main' })).toContain('própria base');
  });

  it('says what stays out: changes in no commit', () => {
    expect(uncommittedNote(tree())).toBeNull();
    const files = [{ path: 'a', status: 'modified', added: 1, removed: 0, binary: false }];
    expect(uncommittedNote(tree({ files }))).toContain('1 arquivo mudado na pasta fica fora do PR');
  });
});

describe('the confirmation sheet', () => {
  const reader = {} as GitReader;
  it('shows every field it sends, the branch pushed, and that nothing merges, before Confirmar', () => {
    const html = renderToStaticMarkup(
      <PrSheet
        conversationId="s1"
        start={{ title: 'Add b', body: 'It adds b.', base: 'main', draft: false }}
        head="feature"
        note="2 arquivos mudados na pasta ficam fora do PR: ele leva só o que já está em commits."
        reader={reader}
        onOpened={() => {}}
        onClose={() => {}}
      />,
    );
    expect(html).toContain('value="Add b"');
    expect(html).toContain('It adds b.</textarea>');
    expect(html).toContain('value="main"');
    expect(html).toContain('feature</span>');
    expect(html).toContain('git push -u origin feature');
    expect(html).toContain('nunca com força');
    expect(html).toContain('nunca faz merge nem liga');
    expect(html).toContain('ficam fora do PR');
    expect(html).toMatch(/<button[^>]*class="start-open"[^>]*>Confirmar<\/button>/);
  });
});

describe('the PR line', () => {
  const body = (state: Parameters<typeof PrBody>[0]['state'], repo: P.GitResponse | null = tree()) =>
    renderToStaticMarkup(<PrBody state={state} repo={repo} onRefresh={() => {}} onCreate={() => {}} />);

  it('links the PR, says how it stands and its CI, and offers no second one', () => {
    const html = body({ kind: 'ready', pr: { branch: 'feature', base: 'main', pr: pr({ checks: checks('fail', 2, 1) }) } });
    expect(html).toContain('href="https://github.com/o/r/pull/7"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain('#7');
    expect(html).toContain('aberto');
    expect(html).toContain('CI vermelho');
    expect(html).toContain('data-ci="fail"');
    expect(html).not.toContain('Criar PR');
  });

  it('offers "Criar PR" for a branch with none, and holds it on the base', () => {
    expect(body({ kind: 'ready', pr: { branch: 'feature', base: 'main' } })).toMatch(/<button[^>]*class="start-open pr-create"[^>]*>Criar PR/);
    expect(body({ kind: 'ready', pr: { branch: 'main', base: 'main' } }, tree({ branch: 'main' }))).toMatch(/disabled=""[^>]*>Criar PR/);
  });

  it('offers a new one beside a PR that was merged', () => {
    const html = body({ kind: 'ready', pr: { branch: 'feature', base: 'main', pr: pr({ state: 'merged' }) } });
    expect(html).toContain('merged');
    expect(html).toContain('Criar PR');
  });

  it('says a failure in the tone of whose it is, never as "no PR"', () => {
    expect(body({ kind: 'failed', why: 'gh is not logged in', tone: failureTone('gh') })).toContain('tone-warn');
    expect(body({ kind: 'failed', why: 'HEAD is detached', tone: failureTone('refused') })).toContain('tone-dim');
    expect(body({ kind: 'failed', why: 'boom', tone: failureTone('failed') })).toContain('tone-err');
    expect(body({ kind: 'loading' })).toContain('Perguntando ao GitHub');
  });
});

describe('the PR at the boundary', () => {
  it('reads the daemon’s answer, with or without a PR', () => {
    expect(decodeGitPR({ branch: 'f', base: 'main' })).toEqual({ ok: true, value: { branch: 'f', base: 'main' } });
    const full = { branch: 'feature', base: 'main', pr: pr() };
    expect(decodeGitPR(full)).toEqual({ ok: true, value: full });
    expect(decodeGitPR({ branch: 'f' })).toMatchObject({ ok: false });
    expect(decodeGitPR({ branch: 'f', base: 'main', pr: { ...pr(), checks: { state: 'pass' } } })).toMatchObject({ ok: false });
  });

  it('opens nothing but https', () => {
    const got = decodeGitPR({ branch: 'f', base: 'main', pr: pr({ url: 'file:///etc/passwd' }) });
    expect(got).toMatchObject({ ok: false });
  });

  it('keeps whose a refusal is', () => {
    const refused = (code: string) => prOutcome({ ok: false, refusal: { code, message: 'why' } }, 'x', 'Falhou');
    expect(refused('gh_unavailable')).toEqual({ ok: false, why: 'why', kind: 'gh' });
    expect(refused('pr_refused')).toEqual({ ok: false, why: 'why', kind: 'refused' });
    expect(refused('internal')).toEqual({ ok: false, why: 'Falhou: why', kind: 'failed' });
    expect(prOutcome({ ok: true, value: { branch: 'f', base: 'm' } }, 'x', 'y')).toEqual({ ok: true, value: { branch: 'f', base: 'm' } });
  });
});

describe('the recording’s PRs', () => {
  it('answers each folder as its daemon would, and opens nothing', async () => {
    const dcode = await fixtureActions.readGitPR('s-catalogar');
    expect(dcode.ok && dcode.value.pr?.number).toBe(212);
    const main = await fixtureActions.readGitPR('s-checkout');
    expect(main.ok && main.value.pr).toBeUndefined();
    expect(await fixtureActions.readGitPR('s-ci')).toMatchObject({ ok: false, kind: 'refused' });
    for (const p of Object.values(recordedPRs)) if (typeof p !== 'string') expect(decodeGitPR(p).ok).toBe(true);
    expect(await fixtureActions.openGitPR('s-catalogar', { title: 'x', body: '', base: 'main', draft: false })).toEqual({
      ok: false,
      why: NOT_CONNECTED,
      kind: 'failed',
    });
  });
});
