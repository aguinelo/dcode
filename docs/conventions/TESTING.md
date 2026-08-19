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

## 3. Coverage: a 90% floor, and critical scenarios tested

CI fails below **90%** line coverage over the whole denominator, **and** below
**90%** in any single package on its own.

The two numbers are equal on purpose. The aggregate answers "is there a package
with no tests at all?"; the per-package floor answers "is there a weak package
hiding behind a strong one?". While the aggregate sat at 95 it did the second
job by accident, and the floor could be printed and ignored. With the aggregate
at 90 the floor is the only one that bites, so it now fails rather than reports.

### Why 90 and not 95

The aggregate sat at 95, at the exact value the tree measured. A gate pinned to
the measured value fails on rounding and on platform geography, not on untested
code — and that is what happened: three pull requests in one night, none of them
for a missing test. 90 is the number the six `.i.spec.md` files already ask for
in their own package, and the number the tree clears comfortably.

Raise the aggregate when it is comfortably clear, never to a number the tree
does not already meet — a gate that is red on arrival is a gate people learn to
ignore. That is the sentence the move to 95 broke, on the page that writes it.

### Critical scenarios are tested, whatever the percentage says

A percentage measures how much code ran, never what was asserted. These have
tests because they are what the product promises, and no good number excuses
any of them:

- **every crossing of a security boundary** — a policy decision, sandbox
  containment, reading or writing a credential;
- **every path where the user's data can disappear** — session recording, the
  event log, context compaction;
- **every bug seen once**, with the reproducing test from section 2;
- **every invariant declared** under `## N. Invariantes verificáveis` in an
  `.i.spec.md`.

The last is the only one a machine enforces, and that is why it exists:
`specguard` fails an invariant that names no existing test, and fails a family
that declares invariants without a guard. A critical scenario that never becomes
an invariant is worth what a sentence is worth — and this repository knows that
price. When you find one, write the invariant.

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
- [ ] Coverage ≥ 90% over the defined denominator, and no package under 90%.
- [ ] Any critical scenario the change touches is asserted, and written as an invariant.
- [ ] No new test without assertions.
- [ ] Any new coverage exclusion is justified in the pull request description.
- [ ] Spec kept in sync, if technical behavior changed.
- [ ] Both language versions updated, if a bilingual document changed.
