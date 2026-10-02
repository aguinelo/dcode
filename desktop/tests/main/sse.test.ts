import { describe, expect, it } from 'vitest';
import { SseParser, decodeData } from '../../src/main/sse';

/** Feeds the chunks in order and returns every frame dispatched. */
function feed(...chunks: string[]) {
  const p = new SseParser();
  return chunks.flatMap((c) => p.push(c));
}

describe('the event stream parser', () => {
  it('reads a session frame: its id and its data', () => {
    expect(feed('id: 7\ndata: {"seq":7}\n\n')).toEqual([{ id: '7', data: '{"seq":7}' }]);
  });

  it('reads a list frame, which has no id', () => {
    expect(feed('data: {"kind":"snapshot"}\n\n')).toEqual([{ data: '{"kind":"snapshot"}' }]);
  });

  it('puts a frame together across chunks, wherever they cut it', () => {
    const whole = 'id: 12\ndata: {"seq":12,"type":"turn.started"}\n\n';
    for (let cut = 1; cut < whole.length; cut++) {
      expect(feed(whole.slice(0, cut), whole.slice(cut)), `cut at ${cut}`).toEqual([
        { id: '12', data: '{"seq":12,"type":"turn.started"}' },
      ]);
    }
  });

  it('dispatches nothing until the blank line comes', () => {
    const p = new SseParser();
    expect(p.push('id: 1\ndata: {"a":1}\n')).toEqual([]);
    expect(p.push('\n')).toEqual([{ id: '1', data: '{"a":1}' }]);
  });

  it('reads many frames from one chunk, in order', () => {
    const chunk = 'id: 1\ndata: "a"\n\nid: 2\ndata: "b"\n\n: ping\n\nid: 3\ndata: "c"\n\n';
    expect(feed(chunk).map((f) => f.id)).toEqual(['1', '2', '3']);
  });

  it('skips comments, so a ping is no frame', () => {
    expect(feed(': ping\n\n', ': ping\n\n')).toEqual([]);
    expect(feed('id: 4\n: a comment inside\ndata: x\n\n')).toEqual([{ id: '4', data: 'x' }]);
  });

  it('tolerates \\r\\n, even split between \\r and \\n', () => {
    expect(feed('id: 5\r\ndata: {"seq":5}\r\n\r\n')).toEqual([{ id: '5', data: '{"seq":5}' }]);
    expect(feed('id: 5\r', '\ndata: {"seq":5}\r', '\n\r', '\n')).toEqual([{ id: '5', data: '{"seq":5}' }]);
  });

  it('tolerates a lone \\r as the line end', () => {
    expect(feed('data: a\r\rdata: b\r\r')).toEqual([{ data: 'a' }, { data: 'b' }]);
  });

  it('joins a frame’s data lines with \\n, and takes the value without its one leading space', () => {
    expect(feed('data: first\ndata:second\ndata:  third\n\n')).toEqual([{ data: 'first\nsecond\n third' }]);
  });

  it('does not carry an id from one frame into the next', () => {
    expect(feed('id: 9\ndata: a\n\ndata: b\n\n')).toEqual([{ id: '9', data: 'a' }, { data: 'b' }]);
  });

  it('ignores fields it has no use for', () => {
    expect(feed('event: x\nretry: 100\ndata: y\n\n')).toEqual([{ data: 'y' }]);
  });
});

describe('a frame’s data', () => {
  it('is the JSON it carries', () => {
    expect(decodeData('{"seq":1,"type":"session.created"}')).toEqual({ seq: 1, type: 'session.created' });
  });

  it('is its own text when it is not JSON, never dropped', () => {
    expect(decodeData('{"seq":1,')).toBe('{"seq":1,');
    expect(decodeData('not json')).toBe('not json');
  });
});
