import { afterEach, describe, expect, it, vi } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { LiveStore } from '../../src/renderer/live';
import type { Answer, DaemonStatus, DcodeApi, ImageAttachment, SessionEvents, StreamEnd } from '../../src/shared/api';
import { STEER_TAKES_TEXT } from '../../src/state/attachments';
import { quietLoops, quietOrchestrate, quietPr } from './helpers';

const picture = (name: string): ImageAttachment => ({ name, mediaType: 'image/png', data: 'iVBORw0KGgo=', bytes: 8 });

/** A main process that answers what the test says, and records what was asked. */
function fakeApi() {
  const asked: string[] = [];
  const listeners = {
    daemon: [] as ((s: DaemonStatus) => void)[],
    list: [] as ((c: P.ConversationChange) => void)[],
    events: [] as ((b: SessionEvents) => void)[],
    end: [] as ((e: StreamEnd) => void)[],
  };
  const ok = <T,>(value: T): Answer<T> => ({ ok: true, value });
  // The mode a new session opens in, as the daemon would answer it.
  let opensIn = 'assist';
  const session = (id: string) => ({ id, workspace: '/w', mode: opensIn }) as P.Session;
  // What the daemon refuses, by the first word of what was asked.
  const refusals: Record<string, { code: string; message: string }> = {};
  const answer = (what: string): Promise<Answer<null>> => {
    asked.push(what);
    const refusal = refusals[what.split(' ')[0] ?? ''];
    return Promise.resolve(refusal ? { ok: false, refusal } : ok(null));
  };
  let resolveContinue: (a: Answer<P.Session>) => void = () => {};
  let models: Answer<P.ModelsResponse> = { ok: false, refusal: { code: 'unexpected_answer', message: 'O daemon respondeu 404 a GET /v1/models: 404 page not found.' } };
  const missing = (path: string): Answer<unknown> => ({
    ok: false,
    refusal: { code: 'unexpected_answer', message: `O daemon respondeu 404 a GET ${path}: 404 page not found.` },
  });
  const crew = {
    skills: missing('/v1/skills?workspace=%2Fw%2Fdcode'),
    memory: missing('/v1/memory?workspace=%2Fw%2Fdcode'),
    commands: missing('/v1/commands?workspace=%2Fw%2Fdcode'),
    undo: ok<unknown>({ restored: ['a.go'], refused: [] }),
    compact: ok<unknown>({ compacted: true }),
    history: missing('/v1/conversations/old/events?from=1'),
    git: missing('/v1/git?workspace=%2Fw%2Fdcode'),
    gitDiff: missing('/v1/git/diff?workspace=%2Fw%2Fdcode&path=a.go'),
  };
  const api: DcodeApi = {
    platform: 'test',
    user: () => Promise.resolve({ name: 'Ana' }),
    onDaemon: (l) => (listeners.daemon.push(l), () => {}),
    onConversations: (l) => (listeners.list.push(l), () => {}),
    onSessionEvents: (l) => (listeners.events.push(l), () => {}),
    onStreamEnd: (l) => (listeners.end.push(l), () => {}),
    follow: (id) => answer(`follow ${id}`),
    unfollow: () => Promise.resolve(),
    continueConversation: (c, model) => {
      asked.push(`continue ${c.id} in ${c.workspace}${model ? ` on ${model}` : ''}`);
      return new Promise((r) => (resolveContinue = r));
    },
    closeSession: (id) => answer(`close ${id}`),
    listModels: (ws) => (asked.push(`models of ${ws}`), Promise.resolve(models)),
    listSkills: (ws) => (asked.push(`skills of ${ws}`), Promise.resolve(crew.skills)),
    listMemory: (ws) => (asked.push(`memory of ${ws}`), Promise.resolve(crew.memory)),
    listCommands: (ws) => (asked.push(`commands of ${ws}`), Promise.resolve(crew.commands)),
    conversationEvents: (id, from) => (asked.push(`history of ${id} from ${from}`), Promise.resolve(crew.history)),
    readGit: (id) => (asked.push(`git of ${id}`), Promise.resolve(crew.git)),
    readGitDiff: (id, path) => (asked.push(`diff of ${path} in ${id}`), Promise.resolve(crew.gitDiff)),
    pickFolder: () => Promise.resolve('/w/novo'),
    createSession: (ws, o) => {
      const how = [o?.model && `on ${o.model}`, o?.sandboxMode && `in ${o.sandboxMode}`].filter(Boolean).join(' ');
      asked.push(`create in ${ws}${how ? ` ${how}` : ''}`);
      return Promise.resolve(ok(session('new')));
    },
    submitTurn: (id, text, images) => answer(`turn ${id}: ${text}${images?.length ? ` with ${images.map((i) => i.name).join(', ')}` : ''}`),
    pickAttachments: () => (asked.push('pick attachments'), Promise.resolve({ attachments: [picture('a.png')], refused: ['b.bin não é um PDF nem um arquivo de texto que o modelo lê.'] })),
    readAttachment: (name, bytes) => {
      asked.push(`read ${name} (${bytes.length} bytes)`);
      return Promise.resolve(
        name.endsWith('.png') ? ok(picture(name)) : { ok: false, refusal: { code: 'invalid_input', message: `${name} não é um PDF nem um arquivo de texto que o modelo lê.` } },
      );
    },
    steer: (id, text) => answer(`steer ${id}: ${text}`),
    interrupt: (id) => answer(`interrupt ${id}`),
    resolveApproval: (id, a, d) => answer(`approve ${id} ${a} ${d}`),
    setMode: (id, m) => answer(`mode ${id} ${m}`),
    renameConversation: (id, name) => answer(`name ${id}: ${name}`),
    // The terminal (D37) talks to its dock, not to the store: none of it is asked here.
    copy: (text) => answer(`copy ${text}`),
    openTerminal: () => Promise.resolve({ ok: false, refusal: { code: 'unexpected', message: 'not under test' } }),
    writeTerminal: () => {},
    resizeTerminal: () => {},
    killTerminal: () => {},
    onTerminal: () => () => {},
    exec: (id, c) => answer(`exec ${id}: ${c}`),
    undo: (id) => (asked.push(`undo ${id}`), Promise.resolve(crew.undo)),
    compact: (id) => (asked.push(`compact ${id}`), Promise.resolve(crew.compact)),
    loops: quietLoops(), pr: quietPr(), orchestrate: quietOrchestrate(), // the loops (D55), the PR (D60) and projects (D66) are not asked here
  };
  return {
    api,
    asked,
    listeners,
    refuse: (what: string, code: string, message: string) => (refusals[what] = { code, message }),
    continued: (id: string) => resolveContinue(ok(session(id))),
    listing: (a: Answer<P.ModelsResponse>) => (models = a),
    opensIn: (mode: string) => (opensIn = mode),
    crew,
  };
}

const ended = { id: 'old', title: 'conserte o parser', workspace: '/w/dcode', state: 'recorded', live: false, turns: 1, started: '2026-09-30T10:00:00Z', last_activity: '2026-09-30T10:05:00Z' };

describe('the window connected', () => {
  it('draws the daemon’s status and its list', () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    expect(store.getSnapshot().daemon).toEqual({ state: 'connecting' });
    f.listeners.daemon[0]?.({ state: 'connected', version: '0.0.0-check', socket: '/s', started: false });
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended] });
    const snap = store.getSnapshot();
    expect(snap.daemon.state).toBe('connected');
    expect(snap.rows.map((r) => [r.id, r.state, r.title])).toEqual([['old', 'recorded', 'conserte o parser']]);
  });

  it('continues an ended conversation once, however many times it is opened', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended] });
    const row = store.getSnapshot().rows[0]!;
    const first = store.actions.open(row);
    const second = store.actions.open(row);
    f.continued('new');
    expect(await first).toEqual({ ok: true, value: 'new' });
    expect(await second).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['continue old in /w/dcode', 'follow new']);

    // Once the list shows the continuation live, opening the ended one opens
    // it — already followed, so its stream is not asked for a second time.
    f.listeners.list[0]?.({ kind: 'changed', conversation: { ...ended, id: 'new', state: 'idle', live: true, continued_from: 'old' } });
    expect(await store.actions.open(row)).toEqual({ ok: true, value: 'new' });
    expect(f.asked.slice(2)).toEqual([]);
  });

  it('follows a panel’s conversation once, and an ended one not at all', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended, { ...ended, id: 'live', state: 'running', live: true }] });
    const byId = new Map(store.getSnapshot().rows.map((r) => [r.id, r]));
    store.actions.watch(byId.get('old')!);
    store.actions.watch(byId.get('live')!);
    store.actions.watch(byId.get('live')!);
    await Promise.resolve();
    expect(f.asked).toEqual(['follow live']);
    // A stream that ended for good is asked for again the next time.
    f.listeners.end[0]?.({ sessionId: 'live', reason: 'a sessão não existe mais' });
    store.actions.watch(byId.get('live')!);
    expect(f.asked).toEqual(['follow live', 'follow live']);
  });

  it('continues a live conversation on the model chosen, closes the one it left, and follows the new one', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [{ ...ended, id: 'live', state: 'idle', live: true }] });
    const switched = store.actions.switchModel('live', '/w/dcode', 'qwen-local');
    f.continued('on-qwen');
    expect(await switched).toEqual({ ok: true, value: 'on-qwen' });
    expect(f.asked).toEqual(['continue live in /w/dcode on qwen-local', 'close live', 'follow on-qwen']);
  });

  it('closes nothing when the conversation had already ended, and says so when the old one stays open', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended, { ...ended, id: 'live', state: 'idle', live: true }] });
    const fromEnded = store.actions.switchModel('old', '/w/dcode', 'gemini');
    f.continued('on-gemini');
    expect(await fromEnded).toEqual({ ok: true, value: 'on-gemini' });
    expect(f.asked).toEqual(['continue old in /w/dcode on gemini', 'follow on-gemini']);

    // The continuation opened: that it is shown matters more than the old one closing.
    f.refuse('close', 'session_not_found', 'no session live');
    const fromLive = store.actions.switchModel('live', '/w/dcode', 'gemini');
    f.continued('on-gemini-2');
    expect(await fromLive).toEqual({ ok: true, value: 'on-gemini-2' });
    expect(said).toEqual(['A conversa continuou em gemini, e a sessão anterior não fechou: no session live']);
  });

  it('reads the daemon’s list of models, and says why when there is none or it cannot be read', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.listModels('/w/dcode')).toEqual({
      ok: false,
      why: 'O daemon não disse quais modelos existem: O daemon respondeu 404 a GET /v1/models: 404 page not found.',
    });
    const menu = {
      default: { name: 'MiniMax-M3', model: 'MiniMax-M3', family: 'minimax-m3', transport: 'openai', measured: true },
      profiles: [{ name: 'local', model: 'qwen3.5-9b', family: 'generic', transport: 'openai', measured: false, notice: 'nobody measured this endpoint' }],
    };
    f.listing({ ok: true, value: menu });
    expect(await store.actions.listModels('/w/dcode')).toEqual({ ok: true, value: menu });
    f.listing({ ok: true, value: { default: { name: 'x' } } as unknown as P.ModelsResponse });
    const unreadable = await store.actions.listModels('/w/dcode');
    expect(unreadable.ok).toBe(false);
    expect(!unreadable.ok && unreadable.why).toContain('Resposta ilegível do daemon: lista de modelos: default.model');
    expect(f.asked).toEqual(['models of /w/dcode', 'models of /w/dcode', 'models of /w/dcode']);
  });

  it('says a daemon older than the skills and memory routes does not expose them, in its own words', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const skills = await store.actions.listSkills('/w/dcode');
    const memory = await store.actions.listMemory('/w/dcode');
    expect(skills).toEqual({
      ok: false,
      why: 'Este daemon ainda não expõe as skills de um projeto (GET /v1/skills) — atualize o dcode. O daemon disse: O daemon respondeu 404 a GET /v1/skills?workspace=%2Fw%2Fdcode: 404 page not found.',
    });
    expect(!memory.ok && memory.why).toMatch(/^Este daemon ainda não expõe a memória de um projeto/);
    expect(f.asked).toEqual(['skills of /w/dcode', 'memory of /w/dcode']);
  });

  it('reads an ended conversation’s history without continuing it, and says why when it cannot (D42)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const older = await store.actions.readHistory('old');
    expect(older.state === 'failed' && older.reason).toMatch(/^Este daemon ainda não expõe o histórico de uma conversa terminada/);
    f.crew.history = { ok: false, refusal: { code: 'session_not_found', message: 'no recorded conversation with the id "old"' } };
    expect(await store.actions.readHistory('old')).toEqual({
      state: 'failed',
      reason: 'O histórico desta conversa não foi lido: no recorded conversation with the id "old"',
    });
    f.crew.history = { ok: true, value: {
      events: [{ seq: 1, session_id: 'old', type: 'turn.started', at: '2026-09-30T10:00:00Z', payload: { turn_id: 't', text: 'conserte o parser' } }],
      total: 1,
    } };
    const read = await store.actions.readHistory('old');
    expect(read.state === 'loaded' && read.view.entries).toMatchObject([{ kind: 'user', text: 'conserte o parser' }]);
    // Read, and nothing else: no continuation, no stream.
    expect(f.asked).toEqual(['history of old from 1', 'history of old from 1', 'history of old from 1']);
  });

  it('reads a workspace’s commands, and says an older daemon, a refusal or an unreadable answer (D50)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.listCommands('/w/dcode')).toEqual({
      ok: false,
      why: 'Este daemon ainda não expõe os comandos de um projeto (GET /v1/commands) — atualize o dcode. O daemon disse: O daemon respondeu 404 a GET /v1/commands?workspace=%2Fw%2Fdcode: 404 page not found.',
    });
    const list = { commands: [{ name: 'review', description: '', source: 'project', path: 'review.md', body: 'Review it.' }], overrides: [] };
    f.crew.commands = { ok: true, value: list };
    expect(await store.actions.listCommands('/w/dcode')).toEqual({ ok: true, value: list });
    f.crew.commands = { ok: false, refusal: { code: 'internal', message: 'config: review.md has no body, so there is nothing to send' } };
    expect(await store.actions.listCommands('/w/dcode')).toEqual({
      ok: false,
      why: 'O daemon não disse quais comandos existem: config: review.md has no body, so there is nothing to send',
    });
    f.crew.commands = { ok: true, value: { commands: 'x' } };
    const bad = await store.actions.listCommands('/w/dcode');
    expect(!bad.ok && bad.why).toMatch(/^Resposta ilegível do daemon: lista de comandos/);
  });

  it('reads skills and memory, and says a refusal or an unreadable answer as a failure', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const skills = { enabled: true, skills: [{ name: 'release', when_to_use: 'ao publicar', source: 'project', path: 'release/SKILL.md', held: false }], notices: [] };
    f.crew.skills = { ok: true, value: skills };
    expect(await store.actions.listSkills('/w/dcode')).toEqual({ ok: true, value: skills });
    f.crew.memory = { ok: false, refusal: { code: 'workspace_invalid', message: 'workspace must be absolute' } };
    expect(await store.actions.listMemory('/w/dcode')).toEqual({ ok: false, why: 'O daemon não disse o que a memória tem: workspace must be absolute' });
    f.crew.memory = { ok: true, value: { path: '.dcode/memory.md' } };
    const unreadable = await store.actions.listMemory('/w/dcode');
    expect(!unreadable.ok && unreadable.why).toBe('Resposta ilegível do daemon: memória: exists deveria ser booleano');
  });

  it('says a daemon older than the git routes does not expose them, rather than an empty diff (D41)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const git = await store.actions.readGit('s1');
    expect(!git.ok && git.why).toMatch(/^Este daemon ainda não expõe o git da pasta \(GET \/v1\/git\) — atualize o dcode/);
    const diff = await store.actions.readGitDiff('s1', 'a.go');
    expect(!diff.ok && diff.why).toMatch(/^Este daemon ainda não expõe o diff de um arquivo/);
    expect(f.asked).toEqual(['git of s1', 'diff of a.go in s1']);
  });

  it('reads a folder’s git and a diff, and says a refusal or an unreadable answer as a failure (D41)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const tree = {
      repository: true,
      root: '/w/dcode',
      branch: 'main',
      detached: false,
      head: 'abc1234',
      worktree: true,
      main_checkout: '/src/dcode',
      files: [{ path: 'a.go', status: 'modified', added: 2, removed: 1, binary: false }],
      truncated: false,
    };
    f.crew.git = { ok: true, value: tree };
    expect(await store.actions.readGit('s1')).toEqual({ ok: true, value: tree });
    f.crew.git = { ok: false, refusal: { code: 'internal', message: 'git is not installed: it is not on the daemon’s PATH' } };
    expect(await store.actions.readGit('s1')).toEqual({
      ok: false,
      why: 'O daemon não leu o git da pasta: git is not installed: it is not on the daemon’s PATH',
    });
    f.crew.git = { ok: true, value: { repository: true, files: [] } };
    const unreadable = await store.actions.readGit('s1');
    expect(!unreadable.ok && unreadable.why).toBe('Resposta ilegível do daemon: git da pasta: detached deveria ser booleano');
    const d = { path: 'a.go', status: 'modified', diff: '@@ -1 +1 @@\n-a\n+b\n', binary: false, truncated: false, limit: 262144 };
    f.crew.gitDiff = { ok: true, value: d };
    expect(await store.actions.readGitDiff('s1', 'a.go')).toEqual({ ok: true, value: d });
  });

  it('steers a running turn, and says the daemon’s reason when it refuses', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.send('s1', 'na verdade, resuma', true)).toEqual({ ok: true, value: null });
    f.refuse('turn', 'session_not_found', 'session s1 not found');
    expect(await store.actions.send('s1', 'oi', false)).toEqual({ ok: false, why: 'A mensagem não foi enviada: session s1 not found' });
    expect(f.asked).toEqual(['steer s1: na verdade, resuma', 'turn s1: oi']);
  });

  it('sends the other way when the daemon knows better what the session is doing', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    // The turn started, and its event has not reached the window yet.
    f.refuse('turn', 'turn_already_active', 'a turn is already running');
    expect(await store.actions.send('s1', 'na verdade, resuma', false)).toEqual({ ok: true, value: null });
    // The turn ended, and the window still shows it running.
    f.refuse('steer', 'no_active_turn', 'no turn is running');
    expect(await store.actions.send('s2', 'e agora?', true)).toEqual({
      ok: false,
      why: 'A mensagem não foi enviada: a turn is already running',
    });
    expect(f.asked).toEqual(['turn s1: na verdade, resuma', 'steer s1: na verdade, resuma', 'steer s2: e agora?', 'turn s2: e agora?']);
  });

  it('sends pictures with a turn, and says the daemon’s reason when it refuses them (D46)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.send('s1', 'o que é isto?', false, [picture('a.png')])).toEqual({ ok: true, value: null });
    f.refuse('turn', 'internal', 'an image could not be decoded: illegal base64 data at input byte 4');
    expect(await store.actions.send('s1', 'e isto?', false, [picture('b.png')])).toEqual({
      ok: false,
      why: 'A mensagem não foi enviada: an image could not be decoded: illegal base64 data at input byte 4',
    });
    expect(f.asked).toEqual(['turn s1: o que é isto? with a.png', 'turn s1: e isto? with b.png']);
  });

  it('never turns a message with pictures into a correction, which carries text only (D46)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.send('s1', 'olhe isto', true, [picture('a.png')])).toEqual({ ok: false, why: STEER_TAKES_TEXT });
    expect(f.asked).toEqual([]);
    // The window thought the session idle; the daemon says a turn runs: nothing is steered.
    f.refuse('turn', 'turn_already_active', 'a turn is already running');
    expect(await store.actions.send('s1', 'olhe isto', false, [picture('a.png')])).toEqual({ ok: false, why: STEER_TAKES_TEXT });
    expect(f.asked).toEqual(['turn s1: olhe isto with a.png']);
  });

  it('asks the main process to pick and to read attachments, and says a refusal by its reason (D46, D59)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.pickAttachments()).toEqual({
      ok: true,
      value: { attachments: [picture('a.png')], refused: ['b.bin não é um PDF nem um arquivo de texto que o modelo lê.'] },
    });
    expect(await store.actions.readAttachment('colada.png', new Uint8Array(8))).toEqual({ ok: true, value: picture('colada.png') });
    expect(await store.actions.readAttachment('dados.bin', new Uint8Array(2))).toEqual({
      ok: false,
      why: 'dados.bin não foi anexado: dados.bin não é um PDF nem um arquivo de texto que o modelo lê.',
    });
    expect(f.asked).toEqual(['pick attachments', 'read colada.png (8 bytes)', 'read dados.bin (2 bytes)']);
  });

  it('asks the daemon to switch the mode, and says its reason when it refuses', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.setMode('s1', 'auto')).toEqual({ ok: true, value: null });
    f.refuse('mode', 'invalid_input', 'unknown mode "yolo" (want plan, assist or auto)');
    expect(await store.actions.setMode('s1', 'plan')).toEqual({
      ok: false,
      why: 'O modo não mudou: unknown mode "yolo" (want plan, assist or auto)',
    });
    expect(f.asked).toEqual(['mode s1 auto', 'mode s1 plan']);
  });

  it('asks the daemon to name a conversation, and says its reason when it refuses (D43)', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.rename('old', 'Parser novo')).toEqual({ ok: true, value: null });
    f.refuse('name', 'invalid_input', 'a name is at most 120 characters');
    expect(await store.actions.rename('old', 'x'.repeat(121))).toEqual({
      ok: false,
      why: 'A conversa não foi renomeada: a name is at most 120 characters',
    });
    expect(f.asked).toEqual(['name old: Parser novo', `name old: ${'x'.repeat(121)}`]);
  });

  it('undoes and compacts, reading the answers, and says an unreadable one or a refusal', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.undo('s1')).toEqual({ ok: true, value: { restored: ['a.go'], refused: [] } });
    expect(await store.actions.compact('s1')).toEqual({ ok: true, value: { compacted: true } });
    f.crew.undo = { ok: true, value: { restored: 'a.go' } };
    expect(await store.actions.undo('s1')).toEqual({ ok: false, why: 'Resposta ilegível do daemon: o desfazer não trouxe as listas restored e refused' });
    f.crew.compact = { ok: false, refusal: { code: 'turn_already_active', message: 'a turn is running' } };
    expect(await store.actions.compact('s1')).toEqual({ ok: false, why: 'O contexto não foi resumido: a turn is running' });
    expect(await store.actions.exec('s1', 'ls')).toEqual({ ok: true, value: null });
    f.refuse('exec', 'turn_already_active', 'a turn is running; wait for it to finish or interrupt it');
    expect(await store.actions.exec('s1', 'ls')).toEqual({
      ok: false,
      why: 'O comando não rodou: a turn is running; wait for it to finish or interrupt it',
    });
  });

  it('opens a new session in the folder chosen, and follows it', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.newSession()).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['create in /w/novo', 'follow new']);
  });

  it('opens a new session in a project it already knows, without asking for a folder', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.newSessionIn('/w/dcode')).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['create in /w/dcode', 'follow new']);
    expect(await store.actions.closeSession('new')).toEqual({ ok: true, value: null });
    f.refuse('close', 'session_not_found', 'no session new');
    expect(await store.actions.closeSession('new')).toEqual({ ok: false, why: 'A sessão não fechou: no session new' });
  });

  it('opens a new session on the model and mode chosen, switching the mode only when the sandbox did not give it', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.newSessionIn('/w/dcode', { model: 'qwen-local', mode: 'auto' })).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['create in /w/dcode on qwen-local in full-access', 'mode new auto', 'follow new']);
    f.asked.length = 0;
    f.opensIn('plan');
    await store.actions.newSessionIn('/w/b', { mode: 'plan' });
    // Opened in plan: no switch. (The fake answers the same id, already followed.)
    expect(f.asked).toEqual(['create in /w/b in read-only']);
  });

  it('opens the session even when the mode is refused after, and says so with the daemon’s reason', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    f.refuse('mode', 'internal', 'session has no engine');
    expect(await store.actions.newSessionIn('/w/dcode', { mode: 'auto' })).toEqual({ ok: true, value: 'new' });
    expect(said).toEqual(['A sessão abriu, mas não no modo auto: session has no engine']);
  });

  it('folds the events it follows, and says what it cannot read or what stopped', () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.events[0]?.({
      sessionId: 's1',
      events: [
        { seq: 1, session_id: 's1', type: 'turn.started', at: '2026-09-30T10:00:00Z', payload: { turn_id: 't1', text: 'oi' } },
        'not an event',
      ],
    });
    expect(store.getSnapshot().sessions.byId.s1?.state).toBe('running');
    vi.useFakeTimers();
    f.listeners.end[0]?.({ sessionId: 's1', reason: 'a sessão não existe mais' });
    vi.runAllTimers();
    expect(said).toEqual([
      'Evento ilegível, sem sessão para mostrá-lo: o evento não é um objeto',
      'Os eventos de “s1” pararam de chegar: a sessão não existe mais',
    ]);
  });

  it('says nothing when the stream that ended belongs to a conversation that ended', () => {
    vi.useFakeTimers();
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [{ ...ended, id: 'live', live: true, state: 'idle' }] });
    f.listeners.end[0]?.({ sessionId: 'live', reason: 'no session live' });
    // The list says it ended a moment after the stream did.
    f.listeners.list[0]?.({ kind: 'changed', conversation: { ...ended, id: 'live' } });
    vi.runAllTimers();
    expect(said).toEqual([]);
  });
});

afterEach(() => {
  vi.useRealTimers();
});
