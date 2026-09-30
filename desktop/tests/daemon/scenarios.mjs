// What "the window talks to a daemon" means, one scenario at a time. Each runs
// against a fresh `dcode serve` and a fresh scripted model, and checks the
// window against the wire: what the daemon logged, not only what was drawn.

import { answerFolderPicker, daemonState, eventually, visible } from './app.mjs';
import { text, toolCall } from './model.mjs';
import { healthy, must } from './wire.mjs';

function fail(message) {
  throw new Error(message);
}

function row(page, id) {
  return page.locator(`[data-session-id="${id}"]`);
}

/** Opens the session in the window and returns its composer field. */
async function open(page, id) {
  await (await visible(row(page, id), 10_000, `the sidebar row of session ${id}`)).click();
  return visible(page.locator('main textarea'), 5_000, 'the composer of the opened session');
}

async function rowState(page, id, state, timeoutMs) {
  await eventually(async () => (await row(page, id).first().getAttribute('data-state')) === state, timeoutMs,
    `the row of session ${id} did not read data-state="${state}"`);
}

/** A daemon of the check with one session, and the window attached to it. */
async function attached(ctx, script) {
  const t = await ctx.fresh();
  const model = await ctx.model();
  script(model);
  const env = ctx.env(t, model);
  const daemon = await ctx.daemon(env, t.ws);
  const session = await must(t.socket, 'POST', '/v1/sessions', { workspace: t.ws });
  const wire = ctx.follow(t.socket, session.id);
  const ui = await ctx.launch(env, t);
  await daemonState(ui.page, 'connected', 15_000);
  return { t, model, env, daemon, session, wire, ui };
}

export const scenarios = [
  {
    name: 'attaches',
    about: 'a daemon already answering is used, its sessions are listed — including one another client opens later — and quitting leaves it running',
    async run(ctx) {
      const t = await ctx.fresh();
      const model = await ctx.model();
      const env = ctx.env(t, model);
      const daemon = await ctx.daemon(env, t.ws);
      const first = await must(t.socket, 'POST', '/v1/sessions', { workspace: t.ws });
      const version = await must(t.socket, 'GET', '/version');
      const ui = await ctx.launch(env, t);
      const status = await daemonState(ui.page, 'connected', 15_000);
      const said = (await status.textContent()) ?? '';
      if (!said.includes(version.version)) fail(`the status reads “${said.trim()}”, without the daemon's version ${version.version}`);
      await visible(row(ui.page, first.id), 10_000, `the row of session ${first.id}, live before the window opened`);
      const later = await must(t.socket, 'POST', '/v1/sessions', { workspace: t.ws });
      await visible(row(ui.page, later.id), 10_000, `the row of session ${later.id}, opened by another client after the window`);
      if ((await ui.page.getByText('gravação').count()) > 0) fail('connected to a daemon, the window still says gravação');
      await ui.close();
      if (!(await healthy(t.socket))) fail('quitting the app stopped a daemon the app did not start');
      await daemon.stop();
    },
  },
  {
    name: 'starts-its-own',
    about: 'with nothing on the socket the app starts dcode serve there, and quitting stops it',
    async run(ctx) {
      const t = await ctx.fresh();
      const model = await ctx.model();
      const env = ctx.env(t, model);
      const ui = await ctx.launch(env, t);
      await daemonState(ui.page, 'connected', 20_000);
      if (!(await healthy(t.socket))) fail(`the window says connected and nothing answers on ${t.socket}`);
      await ui.close();
      await eventually(async () => !(await healthy(t.socket)), 5_000, 'the daemon the app started still answers after the app quit');
    },
  },
  {
    name: 'says-why-not',
    about: 'a daemon that cannot be started is a failure on screen, naming what was tried — never a connection',
    async run(ctx) {
      const t = await ctx.fresh();
      const model = await ctx.model();
      const missing = `${t.dir}/missing/dcode`;
      const env = { ...ctx.env(t, model), DCODE_BIN: missing };
      const ui = await ctx.launch(env, t);
      const status = await daemonState(ui.page, 'failed', 15_000);
      const said = (await status.textContent()) ?? '';
      const told = (await ui.page.locator('body').textContent()) ?? '';
      if (!said.includes(missing) && !told.includes(missing)) fail(`the failure does not name ${missing}; the status reads “${said.trim()}”`);
      if ((await ui.page.locator('[data-daemon="connected"]').count()) > 0) fail('a daemon that never started reads as connected');
    },
  },
  {
    name: 'new-session',
    about: 'Nova sessão asks for a folder, opens a session there, and a message gets the model’s answer',
    async run(ctx) {
      const t = await ctx.fresh();
      const model = await ctx.model();
      model.reply(text('Oi, daqui é o modelo roteirizado.'));
      const env = ctx.env(t, model);
      await ctx.daemon(env, t.dir);
      const ui = await ctx.launch(env, t);
      await daemonState(ui.page, 'connected', 15_000);
      await answerFolderPicker(ui.app, t.ws);
      await ui.page.getByRole('button', { name: /Nova sessão/ }).first().click();
      const created = await eventually(
        async () => ((await must(t.socket, 'GET', '/v1/sessions')) ?? []).find((s) => s.workspace === t.ws),
        10_000,
        `no session was opened in the chosen folder ${t.ws}`,
      );
      await visible(row(ui.page, created.id), 10_000, `the row of the new session ${created.id}`);
      const wire = ctx.follow(t.socket, created.id);
      const box = await visible(ui.page.locator('main textarea'), 5_000, 'the composer of the new session');
      await box.fill('diga oi');
      await box.press('Enter');
      const started = await wire.until((e) => e.type === 'turn.started', 10_000, 'turn.started');
      if (started.payload?.text !== 'diga oi') fail(`the turn started with “${started.payload?.text}”, not what was typed`);
      await visible(ui.page.getByText('Oi, daqui é o modelo roteirizado.'), 15_000, 'the model’s answer in the flow');
      await wire.until((e) => e.type === 'turn.completed', 10_000, 'turn.completed');
      await rowState(ui.page, created.id, 'idle', 10_000);
    },
  },
  {
    name: 'enter-denies',
    about: 'a pending approval blocks the field, and ↵ answers deny',
    async run(ctx) {
      const { ui, session, wire } = await attached(ctx, (m) => {
        m.reply(toolCall('c1', 'bash', { command: 'echo oi' }));
        m.reply(text('Entendido, não rodo.'));
      });
      const box = await open(ui.page, session.id);
      await box.fill('rode echo');
      await box.press('Enter');
      const asked = await wire.until((e) => e.type === 'tool.approval_required', 15_000, 'tool.approval_required');
      const card = await visible(ui.page.locator(`[data-approval-id="${asked.payload.approval_id}"]`), 10_000, 'the approval card');
      await rowState(ui.page, session.id, 'blocked', 5_000);
      if (!(await ui.page.locator('main textarea').first().isDisabled())) fail('the field takes text while an approval waits');
      await ui.page.keyboard.press('Enter');
      const answered = await wire.until((e) => e.type === 'tool.approval_resolved', 10_000, 'tool.approval_resolved after ↵');
      if (answered.payload.decision !== 'deny') fail(`↵ answered ${answered.payload.decision}, not deny`);
      await eventually(async () => (await card.getAttribute('data-decision')) === 'deny', 5_000, 'the card did not read data-decision="deny"');
      await visible(ui.page.getByText('Entendido, não rodo.'), 15_000, 'the model’s answer after the denial');
    },
  },
  {
    name: 'one-allows',
    about: '1 allows once, and the command runs',
    async run(ctx) {
      const { ui, session, wire } = await attached(ctx, (m) => {
        m.reply(toolCall('c1', 'bash', { command: 'echo oi' }));
        m.reply(text('Rodou.'));
      });
      const box = await open(ui.page, session.id);
      await box.fill('rode echo');
      await box.press('Enter');
      const asked = await wire.until((e) => e.type === 'tool.approval_required', 15_000, 'tool.approval_required');
      await visible(ui.page.locator(`[data-approval-id="${asked.payload.approval_id}"]`), 10_000, 'the approval card');
      await ui.page.keyboard.press('1');
      const answered = await wire.until((e) => e.type === 'tool.approval_resolved', 10_000, 'tool.approval_resolved after 1');
      if (answered.payload.decision !== 'allow') fail(`1 answered ${answered.payload.decision}, not allow`);
      const ran = await wire.until((e) => e.type === 'tool.completed', 15_000, 'tool.completed');
      if (!ran.payload.ok) fail(`the allowed command did not run: ${ran.payload.output}`);
      await visible(ui.page.getByText('Rodou.'), 15_000, 'the model’s answer after the command');
    },
  },
  {
    name: 'steers',
    about: 'text sent while the turn runs corrects it — the model gets it in the same turn, and no second turn starts',
    async run(ctx) {
      let release = () => {};
      const { ui, session, wire, model } = await attached(ctx, (m) => {
        release = m.held(toolCall('c1', 'read', { path: 'README.md' }));
        m.reply(text('Resumo em uma linha.'));
      });
      const box = await open(ui.page, session.id);
      await box.fill('leia o README');
      await box.press('Enter');
      await model.waitForRequests(1, 15_000);
      await rowState(ui.page, session.id, 'running', 5_000);
      const correction = 'na verdade, resuma em uma linha';
      await box.fill(correction);
      await box.press('Enter');
      await eventually(async () => (await box.inputValue()) === '', 5_000, 'the field kept the correction: the daemon did not take it');
      release();
      const steered = await wire.until((e) => e.type === 'turn.steered', 15_000, 'turn.steered');
      if (steered.payload.text !== correction) fail(`the daemon was steered with “${steered.payload.text}”`);
      await model.waitForRequests(2, 15_000);
      if (!model.requests[1].body.includes(correction)) fail('the model’s next request did not carry the correction');
      await wire.until((e) => e.type === 'turn.completed', 15_000, 'turn.completed');
      const turns = wire.events.filter((e) => e.type === 'turn.started').length;
      if (turns !== 1) fail(`the correction started a turn of its own: ${turns} turns`);
      await visible(ui.page.getByText(correction), 5_000, 'the correction in the flow');
      await visible(ui.page.getByText('Resumo em uma linha.'), 10_000, 'the model’s answer after the correction');
    },
  },
  {
    name: 'daemon-dies',
    about: 'a daemon that dies is said, and nothing sent afterwards looks accepted',
    async run(ctx) {
      const { ui, session, daemon } = await attached(ctx, () => {});
      const box = await open(ui.page, session.id);
      await daemon.kill();
      const status = await daemonState(ui.page, 'lost', 10_000);
      if (!((await status.textContent()) ?? '').trim()) fail('the lost daemon is marked but not explained');
      if (!(await box.isDisabled())) {
        await box.fill('ainda aí?');
        await box.press('Enter');
        await new Promise((r) => setTimeout(r, 1500));
        if ((await box.inputValue()) !== 'ainda aí?') fail('a message sent to a dead daemon was cleared as if accepted');
      }
      if ((await ui.page.locator('[data-daemon="connected"]').count()) > 0) fail('a dead daemon still reads as connected');
    },
  },
];
