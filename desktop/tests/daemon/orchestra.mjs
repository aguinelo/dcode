// The orchestrator's live agent list, driven from the window against a real
// daemon (D64, D66; step 14 of the orchestration .i). The daemon conducts a
// project with orchestrator.enabled; one scripted model answers every session
// the project opens — the orchestrator, each child, each reviewer — from a
// script of its own; the window opens the orchestrator through "Nova", answers
// the merge through the question card in the conversation (D67), and is read for the rows, the queue and the
// grouped list — and every claim is compared with the daemon's own routes.
//
// Besides the attributes in docs/loop/tasks.md, these read what the
// orchestrator's window already carries: `data-orchestrate` on the toggle of
// "Nova", `data-child-id` and `data-child-state` on a row of the Agentes tab,
// `data-card-id`, `data-card-item`, `data-option` and `data-card-confirm` on
// the question card, and `data-depth` on a child's row in the list of Sessões.

import { spawnSync } from 'node:child_process';
import { writeFile } from 'node:fs/promises';
import http from 'node:http';
import path from 'node:path';
import { daemonState, eventually, visible } from './app.mjs';
import { text, toolCall } from './model.mjs';
import { fail } from './scenarios.mjs';
import { must } from './wire.mjs';

/** The toy project's gate: red while a file says BROKEN. */
const GATE = `#!/bin/sh
if grep -rqx BROKEN --exclude=gate.sh --exclude-dir=.git . ; then
  echo "gate: a file says BROKEN"
  exit 1
fi
echo "gate: green"
`;

/** How long a finished child stays in /children: shorter than the window's 1.5 s quiet read, so the read after the merge no longer has it. */
const RETENTION = '1s';

/** Git with no machine configuration and a fixed identity: the harness commits in the project. */
const GIT_ENV = {
  GIT_CONFIG_GLOBAL: '/dev/null',
  GIT_CONFIG_NOSYSTEM: '1',
  GIT_AUTHOR_NAME: 'check',
  GIT_AUTHOR_EMAIL: 'check@localhost',
  GIT_COMMITTER_NAME: 'check',
  GIT_COMMITTER_EMAIL: 'check@localhost',
};

function git(ws, ...args) {
  const r = spawnSync('git', args, { cwd: ws, encoding: 'utf8', env: { ...process.env, ...GIT_ENV } });
  if (r.status !== 0) throw new Error(`git ${args.join(' ')} failed in ${ws}: ${r.stderr || r.error?.message}`);
  return r.stdout.trim();
}

/**
 * The workspace as a project the orchestrator can merge into locally: a gate
 * committed on main, and the person's tree off main — a local merge moves main
 * by a conditional update-ref, refused while a worktree has it checked out.
 */
async function toyProject(ws) {
  await writeFile(path.join(ws, 'gate.sh'), GATE);
  git(ws, 'add', 'gate.sh');
  git(ws, 'commit', '-q', '-m', 'the gate');
  git(ws, 'checkout', '-q', '--detach');
}

/** Whose request this is, told apart by the tools it was offered: only an orchestrator has delegate, only a reviewer has verdict. */
function role(body) {
  if (body.includes('"name":"delegate"')) return 'orchestrator';
  if (body.includes('"name":"verdict"')) return 'reviewer';
  return 'child';
}

/**
 * One model for every session of the project, as the core's scripted
 * scenarios have it (internal/app/scenario_model_test.go): a child's task and
 * its review carry [[key]], which names the script that answers. Requests
 * arrive from several sessions at once, so a single queue would answer them
 * in whatever order they raced; one script per session is what keeps the
 * scenario deterministic. The orchestrator, past its script, acknowledges
 * each report; anything else past its script is recorded and fails the
 * scenario.
 */
async function startRoleModel(scripts) {
  const seen = {};
  const unexpected = [];
  const server = http.createServer((req, res) => {
    const chunks = [];
    req.on('data', (c) => chunks.push(c));
    req.on('end', () => {
      const body = Buffer.concat(chunks).toString('utf8');
      const who = role(body);
      const name = who === 'orchestrator' ? who : `${who}/${/\[\[([a-z0-9-]+)\]\]/.exec(body)?.[1] ?? '?'}`;
      const n = seen[name] ?? 0;
      seen[name] = n + 1;
      let frames = scripts[name]?.[n];
      if (!frames && who === 'orchestrator') frames = [text('Anotado; o pipeline segue.')];
      if (!frames) {
        unexpected.push(`${name}, answer ${n + 1}`);
        res.writeHead(500, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ error: { message: `the check's model has no answer ${n + 1} scripted for ${name}` } }));
        return;
      }
      res.writeHead(200, { 'content-type': 'text/event-stream' });
      for (const f of frames) res.write(`data: ${f}\n\n`);
      res.end('data: [DONE]\n\n');
    });
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  return {
    url: `http://127.0.0.1:${server.address().port}`,
    seen,
    unexpected,
    close() {
      server.closeAllConnections();
      return new Promise((resolve) => server.close(() => resolve()));
    },
  };
}

/** One entry of a delegate batch. */
function item(key, owns) {
  return { title: key, task: `[[${key}]] o trabalho de ${key}`, owns: [owns], pr_title: `${key}: o trabalho`, why: `${key} é um tema próprio` };
}

/** The row of a child in the Agentes tab. */
function agent(page, id) {
  return page.locator(`[data-child-id="${id}"]`);
}

async function agentState(page, id, state, timeoutMs) {
  await visible(page.locator(`[data-child-id="${id}"][data-child-state="${state}"]`), timeoutMs,
    `the row of child ${id} in the Agentes tab reading data-child-state="${state}"`);
}

/** Waits for a child in the daemon's /children to match, and returns it. */
async function child(socket, orq, pred, timeoutMs, what) {
  let seen = '';
  try {
    return await eventually(
      async () => {
        const all = (await must(socket, 'GET', `/v1/sessions/${orq}/children`))?.children ?? [];
        seen = all.map((c) => `${c.title} ${c.state}`).join(', ') || 'no child';
        return all.find(pred);
      },
      timeoutMs,
      '',
    );
  } catch {
    throw new Error(`${what} within ${timeoutMs} ms; /children reads ${seen}`);
  }
}

/** The row's tone class and words, as the person reads them. */
async function rowLooks(page, id) {
  return agent(page, id).first().evaluate((el) => {
    const s = el.querySelector('.agents-state');
    return { word: (s?.textContent ?? '').trim(), tone: [...(s?.classList ?? [])].find((c) => c.startsWith('tone-')) ?? '', all: (el.textContent ?? '').trim() };
  });
}

/**
 * Whether a child's row in Sessões sits under its parent's: in the same
 * group, after it, drawn in one level with "↳" before the folder (D66).
 */
async function underParent(page, parent, kid) {
  return page.evaluate(([p, k]) => {
    const pr = document.querySelector(`.crew-sessions [data-session-id="${p}"]`);
    const kr = document.querySelector(`.crew-sessions [data-session-id="${k}"]`);
    if (!pr || !kr) return false;
    const same = pr.closest('.crew-group') === kr.closest('.crew-group');
    const after = Boolean(pr.compareDocumentPosition(kr) & Node.DOCUMENT_POSITION_FOLLOWING);
    const folder = kr.querySelector('.crew-item-project');
    const mark = folder ? getComputedStyle(folder, '::before').content : '';
    return same && after && kr.getAttribute('data-depth') === '1' && mark.includes('↳');
  }, [parent, kid]);
}

export const orchestraScenarios = [
  {
    name: 'orchestra-lists',
    about: '◆ orquestradora in Nova opens the project’s orchestrator; its two children are listed under it, one waiting for the person, the other answered merge through the question card in the conversation, merged, and gone after the retention — as is the waiting one once the person closes it',
    async run(ctx) {
      const t = await ctx.fresh();
      const outside = path.join(t.dir, 'elsewhere.txt');
      await writeFile(outside, 'as notas da pessoa\n');
      const model = await startRoleModel({
        orchestrator: [
          [toolCall('o1', 'delegate', { items: [item('notes', 'notes'), item('docs', 'docs')] })],
          [text('Dois agentes despachados.')],
        ],
        // A read outside its worktree crosses a boundary: the child asks, and waits for the person.
        'child/notes': [[toolCall('n1', 'read', { path: outside })], [text('Parado.')]],
        'child/docs': [[toolCall('d1', 'write', { path: 'docs/a.md', content: 'docs\n' })], [text('Feito.')]],
        'reviewer/docs': [[toolCall('r1', 'verdict', { verdict: 'approve', why: 'o doc está bom' })], [text('Revisado.')]],
      });
      try {
        await toyProject(t.ws);
        const env = {
          ...ctx.env(t, model),
          ...GIT_ENV,
          DCODE_MEMORY_ENABLED: 'false',
          DCODE_RECORD_ENABLED: 'true',
          DCODE_ORCHESTRATOR_ENABLED: 'true',
          DCODE_ORCHESTRATOR_DIR: path.join(t.dir, 'orq'),
          DCODE_ORCHESTRATOR_FORGE: 'local',
          DCODE_ORCHESTRATOR_GATE_COMMAND: 'sh gate.sh',
          DCODE_ORCHESTRATOR_GATE_TIMEOUT: '1m',
          DCODE_ORCHESTRATOR_CI_TIMEOUT: '1m',
          DCODE_ORCHESTRATOR_REPORT_DEBOUNCE: '10ms',
          DCODE_ORCHESTRATOR_CLOSED_RETENTION: RETENTION,
        };
        await ctx.daemon(env, t.ws);
        // A conversation in the folder, so "Nova" offers it as a known project.
        const plain = await must(t.socket, 'POST', '/v1/sessions', { workspace: t.ws });
        const ui = await ctx.launch(env, t);
        const page = ui.page;
        await daemonState(page, 'connected', 15_000);
        await visible(page.locator(`[data-session-id="${plain.id}"]`), 10_000, `the row of session ${plain.id}`);

        // Nova → ◆ orquestradora → the project: the window opens the project's orchestrator in the middle.
        await page.getByRole('button', { name: /Nova sessão/ }).first().click();
        await (await visible(page.locator('[data-orchestrate]'), 5_000, 'the ◆ orquestradora toggle of Nova')).click();
        await visible(page.getByText('Abrir a orquestradora de'), 5_000, 'the title “Abrir a orquestradora de”');
        await (await visible(page.locator(`button.place[title="${t.ws}"]`), 5_000, `the project ${t.ws} in Nova`)).click();
        const orq = await eventually(
          async () => ((await must(t.socket, 'GET', '/v1/sessions')) ?? []).find((s) => s.role === 'orchestrator' && s.workspace === t.ws),
          10_000,
          `the daemon has no orchestrator in ${t.ws}`,
        );
        await eventually(async () => (await page.locator(`[data-session-id="${orq.id}"]`).first().getAttribute('aria-current')) === 'true', 10_000,
          `the window did not open the orchestrator ${orq.id}`);
        await visible(page.getByRole('tab', { name: /Agentes/, selected: true }), 5_000, 'the Agentes tab, open on the orchestrator');
        await visible(page.getByText('Esta orquestradora ainda não despachou nenhum agente.'), 5_000, 'the empty agent list');

        const box = await visible(page.locator('main textarea'), 5_000, 'the composer of the orchestrator');
        await box.fill('notas e docs');
        await box.press('Enter');

        // Two children: notes waits for the person, docs goes through to its merge question.
        const notes = await child(t.socket, orq.id, (c) => c.title === 'notes' && c.state === 'awaiting_person', 30_000,
          'child notes never waited for the person');
        if (notes.waiting !== 'approval') fail(`child notes waits on “${notes.waiting}”, want approval`);
        const docs = await child(t.socket, orq.id, (c) => c.title === 'docs', 15_000, 'child docs was never dispatched');
        await agentState(page, notes.id, 'awaiting_person', 10_000);
        const waiting = await rowLooks(page, notes.id);
        if (waiting.word !== 'esperando você' || waiting.tone !== 'tone-warn') {
          fail(`the waiting child reads “${waiting.word}” in ${waiting.tone || 'no tone'}, want “esperando você” in tone-warn`);
        }
        if (!waiting.all.includes('pede aprovação')) fail(`the waiting child's row does not say what it asks: “${waiting.all}”`);

        const question = await eventually(
          async () => ((await must(t.socket, 'GET', `/v1/sessions/${orq.id}/questions`))?.questions ?? [])
            .find((q) => q.child_id === docs.id && q.kind === 'merge' && !q.answer),
          45_000,
          `no merge question was asked about ${docs.id}`,
        );
        await agentState(page, docs.id, 'pr', 10_000);
        const asking = await rowLooks(page, docs.id);
        if (asking.tone !== 'tone-warn') fail(`the PR waiting for its merge answer reads ${asking.tone}, want tone-warn`);

        // The list of Sessões: docs, idle, sits under the orchestrator in its project's group; notes, waiting, is among the Ativas.
        await eventually(() => underParent(page, orq.id, docs.id), 15_000,
          `the row of child ${docs.id} is not under the orchestrator ${orq.id} with ↳ in Sessões`);
        await visible(page.locator(`.crew-sessions [data-session-id="${notes.id}"][data-state="blocked"]`), 10_000,
          `the row of the waiting child ${notes.id} in Sessões, blocked`);

        // The card in the conversation (D67): “◆ 1 pergunta” takes the person to it, and Merge, confirmed, is one POST of the batch.
        await (await visible(page.getByRole('button', { name: '◆ 1 pergunta' }), 10_000, 'the “◆ 1 pergunta” of the Agentes tab')).click();
        const card = await visible(page.locator('main [data-card-id]').filter({ has: page.locator(`[data-card-item="${question.id}"]`) }), 5_000,
          'the question card holding the merge question in the conversation');
        await eventually(() => card.evaluate((el) => el.contains(document.activeElement)), 5_000, '“◆ 1 pergunta” did not focus the card in the conversation');
        await card.locator('[data-option="merge"]').click();
        await card.locator('[data-card-confirm]').click();
        await eventually(
          async () => ((await must(t.socket, 'GET', `/v1/sessions/${orq.id}/questions`))?.questions ?? []).find((q) => q.id === question.id)?.answer === 'merge',
          10_000,
          'the daemon did not take the window’s merge answer',
        );

        // Merged, then gone: the window reads /children again once the stream is quiet, and the daemon no longer carries it.
        await agentState(page, docs.id, 'merged', 30_000);
        if (git(t.ws, 'show', 'main:docs/a.md') !== 'docs') fail('main does not hold the merged docs/a.md');
        await eventually(async () => (await agent(page, docs.id).count()) === 0, 10_000,
          `the merged child ${docs.id} is still in the Agentes tab after the retention`);
        await child(t.socket, orq.id, (c) => c.id === notes.id && c.state === 'awaiting_person', 1_000,
          'past the retention the waiting child is no longer waiting');
        await agentState(page, notes.id, 'awaiting_person', 1_000);
        const counted = await page.locator('.agents-asks').allTextContents();
        if (counted.length > 0) fail(`the answered question is still counted in the Agentes tab: “${counted.join(', ')}”`);

        // The person closes the waiting one, from another client: closed, and gone after the retention too.
        await must(t.socket, 'DELETE', `/v1/sessions/${notes.id}`);
        await eventually(async () => (await agent(page, notes.id).count()) === 0, 10_000,
          `the closed child ${notes.id} is still in the Agentes tab after the retention`);
        await visible(page.getByText('Esta orquestradora ainda não despachou nenhum agente.'), 5_000, 'the agent list, empty once both left');
      } finally {
        await model.close();
      }
      if (model.unexpected.length > 0) fail(`the model was asked what the scenario did not script: ${model.unexpected.join('; ')}`);
    },
  },
];
