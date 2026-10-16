// A loop, driven from the window against a real daemon (D55, D56; step 9 of
// desktop/docs/proposals/loop.md). The daemon conducts the run; the scripted
// model qualifies, proposes and works one spec folder; the window is read for
// the loop's ruler, the proposal waiting for review, the criteria lights of
// each check, and the verdict — and every claim is compared with the wire.
//
// Besides the attributes in docs/loop/tasks.md, these read what the loop's
// window already carries: `data-pane-id` on a window of the grid,
// `data-loop-id` and `data-loop-state` on the loop's ruler, `data-loop-review`
// on the proposal under review, and the criteria lights by their text.

import { spawnSync } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { eventually, visible } from './app.mjs';
import { text, toolCall } from './model.mjs';
import { attached, fail, open } from './scenarios.mjs';
import { must } from './wire.mjs';

const FOLDER = 'specs/ola';

/** A spec folder in the workspace, committed: spec.md, and done.toml when given. */
async function specFolder(ws, done) {
  const dir = path.join(ws, FOLDER);
  await mkdir(dir, { recursive: true });
  await writeFile(path.join(dir, 'spec.md'), '# Olá\n\nO projeto diz olá e tchau.\n');
  const files = [path.join(FOLDER, 'spec.md')];
  if (done) {
    await writeFile(path.join(dir, 'done.toml'), done);
    files.push(path.join(FOLDER, 'done.toml'));
  }
  for (const args of [['add', ...files], ['-c', 'user.name=check', '-c', 'user.email=check@localhost', 'commit', '-q', '-m', 'spec']]) {
    const r = spawnSync('git', args, { cwd: ws, encoding: 'utf8' });
    if (r.status !== 0) throw new Error(`git ${args.join(' ')} failed in ${ws}: ${r.stderr || r.error?.message}`);
  }
}

/** The run the daemon conducts in the workspace, once there is one. */
async function theLoop(socket, ws, timeoutMs) {
  return eventually(
    async () => ((await must(socket, 'GET', `/v1/loops?workspace=${encodeURIComponent(ws)}`))?.loops ?? [])[0],
    timeoutMs,
    `the daemon lists no loop in ${ws}`,
  );
}

/** Waits for the daemon to say the run is in `state`, and returns it. */
async function loopState(socket, ws, id, state, timeoutMs) {
  let seen = '';
  try {
    return await eventually(
      async () => {
        const run = ((await must(socket, 'GET', `/v1/loops?workspace=${encodeURIComponent(ws)}`))?.loops ?? []).find((l) => l.id === id);
        seen = run?.state ?? 'not listed';
        return run?.state === state ? run : null;
      },
      timeoutMs,
      '',
    );
  } catch {
    throw new Error(`the daemon did not put loop ${id} in ${state} within ${timeoutMs} ms; it reads ${seen}`);
  }
}

/** The loop's window: the one of the grid whose ruler is this run's. */
function loopPane(page, id) {
  return page.locator('[data-pane-id]', { has: page.locator(`[data-loop-id="${id}"]`) });
}

async function windowState(page, id, state, timeoutMs) {
  await visible(page.locator(`[data-loop-id="${id}"][data-loop-state="${state}"]`), timeoutMs, `the ruler of loop ${id} reading data-loop-state="${state}"`);
}

/** The light of a criterion in the loop's window: ✓ met, ✗ unmet. */
function light(page, id, name, met) {
  return loopPane(page, id).getByText(`${met ? '✓' : '✗'} ${name}`, { exact: true });
}

export const loopScenarios = [
  {
    name: 'loop-qualifies',
    about: 'Começar loop on a folder with no criteria qualifies it, the proposal waits for review, Começar o trabalho works it, and the lights move with each check up to the verdict',
    async run(ctx) {
      let proposing = () => {};
      let second = () => {};
      const { t, ui, model } = await attached(ctx, (m) => {
        proposing = m.held(
          toolCall('p1', 'done_propose', {
            criteria: [
              { name: 'ola', command: 'test -f ola.txt', expects: 'fail', why: 'o trabalho cria ola.txt' },
              { name: 'tchau', command: 'test -f tchau.txt', expects: 'fail', why: 'o trabalho cria tchau.txt' },
              { name: 'readme', command: 'test -f README.md', expects: 'pass', why: 'o README continua lá' },
            ],
          }),
        );
        m.reply(text('Proposta registrada.'));
        m.reply(toolCall('w1', 'write', { path: 'ola.txt', content: 'olá\n' }));
        m.reply(text('Escrevi ola.txt.'));
        second = m.held(toolCall('w2', 'write', { path: 'tchau.txt', content: 'tchau\n' }));
        m.reply(text('Pronto: ola.txt e tchau.txt.'));
      });
      await specFolder(t.ws);
      const page = ui.page;

      // Nova sessão → ↻ loop → the project → the sheet, where the folder says
      // what the daemon declares for it.
      await page.getByRole('button', { name: /Nova sessão/ }).first().click();
      await (await visible(page.getByRole('button', { name: '↻ loop' }), 10_000, 'the ↻ loop toggle of Nova sessão')).click();
      await (await visible(page.locator(`button.place[title="${t.ws}"]`), 5_000, `the project ${t.ws} in Nova sessão`)).click();
      const folder = await visible(page.getByRole('radio', { name: new RegExp(`${FOLDER}.*0 · qualifica antes`) }), 10_000,
        `${FOLDER} in Começar loop, declaring “0 · qualifica antes”`);
      await folder.click();
      await page.getByRole('button', { name: /^Começar ⌘↵$/ }).click();

      const started = await theLoop(t.socket, t.ws, 10_000);
      if (started.args !== FOLDER) fail(`the daemon was asked /loop ${started.args}, not /loop ${FOLDER}`);
      const id = started.id;
      await model.waitForRequests(1, 15_000);
      const qualifying = await loopState(t.socket, t.ws, id, 'qualifying', 5_000);
      await windowState(page, id, 'qualifying', 15_000);
      const qwire = ctx.follow(t.socket, qualifying.session_id);
      proposing();

      // The proposal, measured and written, waits for a person.
      const proposed = await qwire.until((e) => e.type === 'done.proposed', 20_000, 'done.proposed');
      const file = proposed.payload.path;
      if (!file.endsWith(path.join(FOLDER, 'done.toml'))) fail(`the proposal was written to ${file}, not into ${FOLDER}`);
      await loopState(t.socket, t.ws, id, 'awaiting_review', 10_000);
      await windowState(page, id, 'awaiting_review', 10_000);
      const review = await visible(page.locator(`[data-loop-review="${id}"]`), 10_000, 'the proposal under review');
      for (const name of ['ola', 'tchau', 'readme']) {
        await visible(review.getByText(name, { exact: true }), 5_000, `criterion ${name} in the review`);
      }
      if (model.requests.length !== 2) fail(`the work started before anyone said so: the model had ${model.requests.length} requests`);
      await review.getByRole('button', { name: 'Começar o trabalho' }).click();

      // The work: the window follows the run into the work session, and each
      // check of the criteria moves its lights before the turn ends.
      // Working is said as the button is answered, before the session opens.
      await loopState(t.socket, t.ws, id, 'working', 10_000);
      const working = await eventually(
        async () => ((await must(t.socket, 'GET', `/v1/loops?workspace=${encodeURIComponent(t.ws)}`))?.loops ?? [])
          .find((l) => l.id === id && l.session_id && l.session_id !== qualifying.session_id),
        10_000,
        'the work did not open a session of its own',
      );
      const wwire = ctx.follow(t.socket, working.session_id);
      const first = await wwire.until((e) => e.type === 'done.checked', 20_000, 'the first done.checked');
      if (!(first.payload.unmet ?? []).includes('tchau') || !(first.payload.met ?? []).includes('ola')) {
        fail(`the first check read met ${first.payload.met} and unmet ${first.payload.unmet}`);
      }
      await model.waitForRequests(5, 15_000);
      await visible(page.locator(`[data-pane-id="${working.session_id}"] [data-loop-id="${id}"]`), 10_000,
        `the loop's window following it into the work session ${working.session_id}`);
      await visible(light(page, id, 'ola', true), 10_000, 'the light ✓ ola after the first check');
      await visible(light(page, id, 'tchau', false), 5_000, 'the light ✗ tchau after the first check');
      second();

      const last = await wwire.until((e) => e.type === 'done.checked' && e.payload.cycle === 2, 20_000, 'the second done.checked');
      if ((last.payload.unmet ?? []).length > 0 || last.payload.moved !== 'forward') fail(`the second check read unmet ${last.payload.unmet}, moved ${last.payload.moved}`);
      const done = await wwire.until((e) => e.type === 'turn.completed', 15_000, 'turn.completed of the work');
      if (done.payload.reason !== 'done') fail(`the work ended ${done.payload.reason}, not done`);
      const finished = await loopState(t.socket, t.ws, id, 'finished', 10_000);
      if (finished.verdict?.met !== 3 || finished.verdict?.total !== 3) fail(`the daemon's verdict is ${JSON.stringify(finished.verdict)}`);
      await windowState(page, id, 'finished', 10_000);
      await visible(loopPane(page, id).getByText('1 pasta trabalhada · 3 de 3 critérios'), 10_000, 'the verdict in the loop’s window');
      await visible(light(page, id, 'tchau', true), 5_000, 'the light ✓ tchau at the end');
    },
  },
  {
    name: 'loop-stops',
    about: '/loop in the field works a folder, and Parar agora interrupts the turn and ends the run as stopped',
    async run(ctx) {
      const { t, ui, model, session, wire: typedIn } = await attached(ctx, (m) => {
        m.held(toolCall('w1', 'write', { path: 'ola.txt', content: 'olá\n' }));
      });
      await specFolder(t.ws, '[ola]\ncommand = "test -f ola.txt"\n');
      const box = await open(ui.page, session.id);
      await box.fill(`/loop ${FOLDER}`);
      await box.press('Enter');
      const started = await theLoop(t.socket, t.ws, 10_000);
      const id = started.id;
      await model.waitForRequests(1, 15_000);
      if (typedIn.events.some((e) => e.type === 'turn.started')) fail('/loop went to the model as a turn of the session it was typed in');
      const working = await loopState(t.socket, t.ws, id, 'working', 5_000);
      const wire = ctx.follow(t.socket, working.session_id);
      await windowState(ui.page, id, 'working', 15_000);
      await (await visible(loopPane(ui.page, id).getByRole('button', { name: /Parar agora/ }), 5_000, 'Parar agora in the loop’s window')).click();

      const stopped = await loopState(t.socket, t.ws, id, 'stopped', 10_000);
      const ended = await wire.until((e) => e.type === 'turn.completed', 10_000, 'turn.completed of the stopped work');
      if (ended.payload.reason !== 'interrupted') fail(`the stopped turn ended ${ended.payload.reason}, not interrupted`);
      if (stopped.error) fail(`the run stopped with an error: ${stopped.error}`);
      await windowState(ui.page, id, 'stopped', 10_000);
      if ((await loopPane(ui.page, id).getByRole('button', { name: /Parar agora/ }).count()) > 0) fail('a stopped run still offers Parar agora');
      if (existsSync(path.join(t.ws, 'ola.txt'))) fail('the interrupted turn wrote ola.txt');
    },
  },
];
