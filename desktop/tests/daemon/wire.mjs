// The check's own view of the daemon: HTTP over the Unix socket, and the event
// stream of a session. What the window claims is compared with what the wire
// carried — a decision the window says it sent is only sent if the daemon
// logged it.

import http from 'node:http';

/** One request to the daemon. Resolves with the status and the parsed body. */
export function call(socket, method, path, body) {
  return new Promise((resolve, reject) => {
    const payload = body === undefined ? undefined : JSON.stringify(body);
    const req = http.request(
      {
        socketPath: socket,
        method,
        path,
        headers: payload ? { 'content-type': 'application/json', 'content-length': Buffer.byteLength(payload) } : {},
      },
      (res) => {
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => {
          const raw = Buffer.concat(chunks).toString('utf8');
          let json;
          try {
            json = raw ? JSON.parse(raw) : null;
          } catch {
            json = { unparsed: raw };
          }
          resolve({ status: res.statusCode ?? 0, json });
        });
      },
    );
    req.on('error', reject);
    req.setTimeout(10_000, () => req.destroy(new Error(`${method} ${path} on ${socket} took more than 10 s`)));
    if (payload) req.write(payload);
    req.end();
  });
}

/** True when a daemon answers /health on the socket. */
export async function healthy(socket) {
  try {
    const r = await call(socket, 'GET', '/health');
    return r.status === 200;
  } catch {
    return false;
  }
}

/** A request that must succeed; anything else fails with what the daemon said. */
export async function must(socket, method, path, body) {
  const r = await call(socket, method, path, body);
  if (r.status < 200 || r.status >= 300) {
    throw new Error(`${method} ${path} answered ${r.status}: ${JSON.stringify(r.json)}`);
  }
  return r.json;
}

/**
 * Follows a session's events from the first. `until(pred)` resolves with the
 * first event, already seen or still to come, that matches — or fails after
 * the timeout, listing the types that did arrive.
 */
export function follow(socket, sessionId) {
  const events = [];
  let wake = () => {};
  let broken = null;
  const req = http.request({ socketPath: socket, method: 'GET', path: `/v1/sessions/${sessionId}/events?from=1` }, (res) => {
    if (res.statusCode !== 200) {
      broken = new Error(`the event stream of ${sessionId} answered ${res.statusCode}`);
      wake();
      return;
    }
    let buffer = '';
    res.setEncoding('utf8');
    res.on('data', (chunk) => {
      buffer += chunk;
      let cut;
      while ((cut = buffer.indexOf('\n\n')) >= 0) {
        const frame = buffer.slice(0, cut);
        buffer = buffer.slice(cut + 2);
        const data = frame
          .split('\n')
          .filter((l) => l.startsWith('data: '))
          .map((l) => l.slice(6))
          .join('\n');
        if (!data) continue;
        try {
          events.push(JSON.parse(data));
        } catch {
          events.push({ type: 'unparsed', payload: data });
        }
      }
      wake();
    });
    // A daemon that dies mid-stream closes without an end: close covers both.
    res.on('close', () => {
      broken ??= new Error(`the event stream of ${sessionId} closed`);
      wake();
    });
  });
  req.on('error', (err) => {
    broken ??= err;
    wake();
  });
  req.end();

  return {
    events,
    async until(pred, timeoutMs, what) {
      const deadline = Date.now() + timeoutMs;
      for (;;) {
        const hit = events.find(pred);
        if (hit) return hit;
        if (broken) throw new Error(`waiting for ${what}: ${broken.message}; seen: ${events.map((e) => e.type).join(', ') || 'nothing'}`);
        const left = deadline - Date.now();
        if (left <= 0) {
          throw new Error(`the daemon did not log ${what} within ${timeoutMs} ms; seen: ${events.map((e) => e.type).join(', ') || 'nothing'}`);
        }
        await new Promise((resolve) => {
          const t = setTimeout(resolve, Math.min(left, 250));
          wake = () => {
            clearTimeout(t);
            resolve();
          };
        });
      }
    },
    close() {
      req.destroy();
    },
  };
}
