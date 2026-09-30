// A model that answers what the check tells it to.
//
// The daemon speaks to it as to any OpenAI-compatible endpoint: a POST, a
// stream of `data:` frames, then `data: [DONE]`. Everything between the window
// and this server is real — the daemon, its policy, its sandbox, its log — and
// only the model is scripted, because the model is the one part that answers
// differently every time. Same shape as the core's end-to-end fake
// (internal/app/e2e_test.go, frameText and frameToolCall).

import http from 'node:http';

/** A frame of streamed answer text. */
export function text(s) {
  return JSON.stringify({ choices: [{ delta: { content: s } }] });
}

/** A frame asking for one tool call. */
export function toolCall(id, name, args) {
  return JSON.stringify({ choices: [{ delta: { tool_calls: [{ id, function: { name, arguments: JSON.stringify(args) } }] } }] });
}

/**
 * Starts the model on a free local port. Replies are consumed in order, one
 * per request; a request with nothing scripted gets an error the daemon will
 * report, and is recorded, so the check can say which request was unexpected
 * instead of hanging on it.
 */
export async function startModel() {
  const replies = [];
  const requests = [];
  const unexpected = [];
  let notify = () => {};

  const server = http.createServer((req, res) => {
    const chunks = [];
    req.on('data', (c) => chunks.push(c));
    req.on('end', () => {
      const body = Buffer.concat(chunks).toString('utf8');
      if (req.method !== 'POST') {
        unexpected.push(`${req.method} ${req.url}`);
        res.writeHead(404, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ error: { message: `the check's model answers POST only, got ${req.method} ${req.url}` } }));
        return;
      }
      requests.push({ url: req.url, body });
      notify();
      const reply = replies.shift();
      if (!reply) {
        unexpected.push(`request ${requests.length} (POST ${req.url}) had no reply scripted`);
        res.writeHead(500, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ error: { message: `the check's model has no reply scripted for request ${requests.length}` } }));
        return;
      }
      void reply.gate.then(() => {
        res.writeHead(200, { 'content-type': 'text/event-stream' });
        for (const f of reply.frames) res.write(`data: ${f}\n\n`);
        res.end('data: [DONE]\n\n');
      });
    });
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });

  return {
    url: `http://127.0.0.1:${server.address().port}`,
    requests,
    unexpected,
    /** Scripts the next reply. */
    reply(...frames) {
      replies.push({ frames, gate: Promise.resolve() });
    },
    /** Scripts the next reply, held until the returned function is called. */
    held(...frames) {
      let release = () => {};
      const gate = new Promise((resolve) => {
        release = resolve;
      });
      replies.push({ frames, gate });
      return release;
    },
    /** Resolves once `n` requests have arrived, or fails naming how many did. */
    async waitForRequests(n, timeoutMs) {
      const deadline = Date.now() + timeoutMs;
      while (requests.length < n) {
        const left = deadline - Date.now();
        if (left <= 0) throw new Error(`the model expected request ${n} within ${timeoutMs} ms and got ${requests.length}`);
        await new Promise((resolve) => {
          const t = setTimeout(resolve, Math.min(left, 200));
          notify = () => {
            clearTimeout(t);
            resolve();
          };
        });
      }
    },
    close() {
      server.closeAllConnections();
      return new Promise((resolve) => server.close(() => resolve()));
    },
  };
}
