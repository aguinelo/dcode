import { describe, expect, it } from 'vitest';
import child from '../../../internal/protocol/testdata/child.json?raw';
import childMinimal from '../../../internal/protocol/testdata/child_minimal.json?raw';
import question from '../../../internal/protocol/testdata/question.json?raw';
import reviewVerdict from '../../../internal/protocol/testdata/review_verdict.json?raw';
import gateMeasured from '../../../internal/protocol/testdata/gate_measured.json?raw';
import sessionChild from '../../../internal/protocol/testdata/session_child.json?raw';
import { decodeEvent } from '../../src/protocol/events';

const envelope = (type: string, payload: unknown) => ({
  seq: 1,
  session_id: 's-1',
  type,
  at: '2026-10-17T12:00:00Z',
  payload,
});

// The payloads are the Go goldens themselves, so a field the core renames
// fails here and not in front of someone.
describe('the orchestrator events', () => {
  it('reads what the core writes', () => {
    for (const [type, payload] of [
      ['child.started', { child: JSON.parse(childMinimal) }],
      ['child.state', { child: JSON.parse(child) }],
      ['question.asked', JSON.parse(question)],
      ['review.verdict', JSON.parse(reviewVerdict)],
      ['gate.measured', JSON.parse(gateMeasured)],
      ['session.created', JSON.parse(sessionChild)],
    ] as const) {
      const d = decodeEvent(envelope(type, payload));
      expect(d.ok ? null : d.problem.reason, type).toBeNull();
    }
  });

  it('says what is wrong with a child it cannot read', () => {
    const broken = { ...JSON.parse(childMinimal), owns: 'internal/index/' };
    const d = decodeEvent(envelope('child.state', { child: broken }));
    expect(d.ok).toBe(false);
    if (!d.ok) expect(d.problem.reason).toContain('owns');
  });

  it('refuses a session whose owns is not a list', () => {
    const s = { ...JSON.parse(sessionChild), owns: 3 };
    expect(decodeEvent(envelope('session.created', s)).ok).toBe(false);
  });
});
