# Testing convention

🇧🇷 [Versão em português](TESTING.pt-BR.md)

Applies to all code in this repository. A pull request that violates any rule here is
blocked.

---

## 1. TDD

Mandatory cycle for new code: **red → green → refactor.**

1. Write the test first. It **must** fail.
2. Verify it fails **for the right reason** — a broken assertion, not a compile error or
   a typo in a filename. A test that fails for the wrong reason is not red, it is noise.
3. Write the minimum code to make it pass.
4. Refactor with the green test as a net.

**London School (mock-first)** for new code, per organization convention: the
dependencies of the unit under test are replaced by doubles, and what gets verified is
the interaction with collaborators. This keeps tests fast and module boundaries explicit.

Deliberate exception: code at the OS boundary — sandbox, socket, PTY — is tested against
the real resource in `t.TempDir()`, not against a mock. Mocking `syscall` tests the mock,
not the behavior.

---

## 2. A bug requires a reproducing test — before the fix

**Explicit rule, no exceptions:**

1. Reproduce the bug in a test. The test **fails**.
2. Confirm it fails for exactly the reported symptom.
3. Only then fix it.
4. The same test passes, unmodified.

The reproducing test ships in the **same commit or pull request** as the fix. A `fix:`
pull request with no new test is blocked.

### Why the order matters

A test written after the fix does not prove it reproduced the bug — only that the current
code passes it. If it was never seen red, there is no evidence it would catch the
regression. Writing it first is what turns the test into a safety net instead of
decoration.

### Regressions are permanent

A reproducing test is never removed, nor "simplified" during refactoring. Name it
traceably — `TestEventLog_NoGapUnderConcurrentAppend_Issue42` — so that years later it is
obvious that the case exists because something actually broke once.

---

## 3. Coverage gate: 95% aggregate, 90% per package

CI fails below **95%** line coverage over the whole denominator, and reports any
package under **90%** on its own.

Two numbers, because they answer different questions. The per-package floor is
what five `.i.spec.md` files require, and it exists so a weak package cannot
hide behind stronger ones. The aggregate is the one that ratchets: 90% was
reached and staying there turned the margin into slack where new code arrived
untested.

Raise the aggregate when it is comfortably clear, never to a number the tree
does not already meet — a gate that is red on arrival is a gate people learn to
ignore.

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### The denominator

A gate without a defined denominator is either unreachable or vacuous. Here it is
explicit.

**Counted:** all deterministic code in `internal/**` and `pkg/**`.

**Excluded, with justification:**

| Exclusion | Reason |
|---|---|
| Generated code | not authored; test the generator, not its output |
| `cmd/**` — `main` wiring | dependency assembly, no logic; covered by a smoke test |
| Model-mediated paths behind build tags | not verifiable by assertion — see section 4 |
| OS-specific code, off its platform | not executable there; covered in that platform's CI matrix |

### The gate measures the matrix, not one runner

The line above sat declared for months with nothing honouring it: the gate ran
inside each matrix job, so a branch reachable only on macOS counted as uncovered
on Ubuntu. It failed three pull requests in one night, always for that reason,
and always on code that was tested — on the other platform.

In CI the gate runs once, over the union of the profiles:

```bash
./scripts/merge-coverage.sh profiles/*/coverage.out > coverage.out
./scripts/coverage.sh coverage.out
```

`make check` still measures only the platform you are on. That is a stricter
approximation than CI's, not a looser one — whatever it rejects CI would reject
by another route, and the reverse is what this arrangement fixes.

A new exclusion requires justification in the pull request. "Hard to test" is not a
justification — it is usually a symptom of coupling, and the fix is the design, not the
exemption.

### What the gate does not prove

The gate is a **floor, not a target.** It catches files with no tests at all; it
does not prove correctness.

The classic way to game it is a test with no assertions — it exercises the line, verifies
nothing, and raises the number. In review, a test that calls a function and asserts
nothing about the result is a finding, even when coverage is green.

Line coverage is also not case coverage: 100% of lines through a single happy path
ignores every error branch. Table-driven tests with edge cases are worth more than the
percentage.

---

## 4. Model-mediated behavior

Stays out of the gate, for the reason recorded in the determinism boundary that every
`.r.spec.md` declares (see `SDD-HARNESS.md`).

Behavior that emerges from interaction with the LLM is not verifiable by assertion. It is
measured by a **threshold over fixtures**, declared in the behavioral contracts section
of the corresponding `.p.spec.md`.

- Sits behind a build tag or `testing.Short()` — it depends on a real model and costs
  money.
- A regression below the threshold is a blocker, same as a red test.
- Lowering a threshold in the same pull request that broke it requires a `changelog/`
  entry, because that is a rule change.

**The incentive is intentional:** because this code is outside the gate, there is pressure
to push behavior to the deterministic side — where it counts toward coverage and is
exactly verifiable. That is the same architectural goal described in `SDD-HARNESS.md`.

---

## 5. Pull request checklist

- [ ] New code came from a test that failed first.
- [ ] `fix:` ships with a reproducing test that failed before the fix.
- [ ] `go test -race ./...` is clean.
- [ ] Coverage ≥ 95% over the defined denominator, and no package under 90%.
- [ ] No new test without assertions.
- [ ] Any new coverage exclusion is justified in the pull request description.
- [ ] Spec kept in sync, if technical behavior changed.
- [ ] Both language versions updated, if a bilingual document changed.
