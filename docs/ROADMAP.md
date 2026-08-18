# Roadmap

What is not built and why it would be worth building. Each item carries the
reason, the shape, what already exists to build on, and what would break.

Nothing here is a commitment to a date. The order at the end is a
recommendation, not a schedule.

---

## Delivered

The five items this file opened with are all shipped, and the reasoning behind
each lives in its family's changelog rather than here.

| | shipped in |
|---|---|
| Undo a turn | #162 |
| The model can see a screenshot | #164 |
| Reaching the network, through the permission that already exists | #163 |
| Continue the last session | #160, #161 |
| The input box holds more than one line | #158 |

Since then, and from the same question — what does a mature harness have that
this one does not:

| | shipped in |
|---|---|
| The prompt says where the agent is (git state) | #171 |
| One edit call, many changes | #172 |
| Steer a turn without killing it | #173 |
| Learned memory: read, write, and the Layer 2 counterweight | #175, #176, #178 |

---

## Found by looking at the loop

Two questions decide whether a harness is any good, and most get them wrong:
**when to stop**, and **what to do with failure**. dcode answers the second well.
The first it answers on one axis and is blind on the other.

---

## 1. The model never learns it is running out of rounds

**The gap.** `MaxIterations` cuts the turn dead at `StopMaxIterations`. Nothing
warns the model on the way there — no reminder mentions rounds at all.

There IS an equivalent for context: budget bands cross upward and a reminder
fires, and `warns-when-task-exceeds-budget` measures that the model says so and
stops starting things. **Nothing does that for rounds.** The same class of
problem, solved on one axis and untouched on the other.

**The evidence, from this session.** Measuring the memory contracts, five runs in
six burned all twelve rounds looking for a fix that could not exist. Not one of
them knew it was on round ten of twelve. A model that knew would have done the
right thing — say what it found and stop — which is exactly what the context
version already asks for.

**Already there.** `CeilingReached(round, rounds, calls)` since #146, the
reminder channel, and the band pattern the budget notice already uses. This is
wiring, not invention.

**What would break.** Nothing in the loop. The risk is in the wording: a warning
that fires too early turns into a model that gives up with rounds to spare, which
is worse than the silence it replaces. The budget notice took two attempts to get
this right and the second is measured.

**Verified by.** A contract in the shape of `warns-when-task-exceeds-budget`: a
task too large for its ceiling, and the model saying so rather than being cut
mid-sentence.

---

## 2. Nothing tells a retryable failure from a hopeless one

**The gap.** A `read` on a path that does not exist can be retried until the
ceiling. Provider errors have `Backoff.Wait` and a class; tool errors have a
code, a hint, and no notion of whether trying again could ever work.

**What makes this cheap now.** #178 built a detector that sees exactly this —
same tool, same error code, same path, twice in one turn. It uses that to ask the
model to record a gotcha, and **for nothing else**. The information is already
being collected and the most obvious decision does not read it.

**The shape.** The second hit is already known. It could carry a different
sentence: not "remember this" but "this will not start working; do something
else or say you cannot". Same trigger, second consumer.

**What would break.** The line between "will never work" and "has not worked
yet" is a judgement, and getting it wrong stops a model that was one call from
succeeding. Start narrow: the same path, the same code, twice — which is already
what the detector requires.

---

## 3. `does-not-remember-activity` passes vacuously

**The finding.** It reads 100%, and it reads 100% because the model never calls
`remember` at all — not because it exercises restraint. It measures the same
absence as `remembers-what-cost-time`, from the other side.

It starts meaning something the day the first one rises. Until then its number is
decorative, and this entry exists so nobody reads it as a guarantee.

**What to do.** Nothing yet. Revisit after the Layer 2 reminder is measured — if
writing starts happening, this contract becomes the one that keeps it honest.

---

## 4. `remembers-what-cost-time` has no honest number

**Four fixture designs, zero calls, and three of the four failures were the
fixture's fault:** a Makefile removed with the shell while the README still
pointed at it; `package example` against the shared workspace's `package stats`;
and the trap sitting beside the task rather than inside it, which the model read
correctly as a pre-existing error outside its work.

The fourth put the discovery on the critical path and the model routed around it
by a simpler path — which is what a competent agent should do.

**The open question is whether a scenario can force a discovery at all.** It may
be that this contract cannot be built as a single scripted task, and that the
honest measurement is longer and messier than a fixture.

**Do not keep redesigning the fixture until the number moves.** That is tuning
the instrument against the result, and it is how the `init` family got a 100%
threshold legitimised by a check that did not exist.

---

## 5. The coverage gate has no margin

**What happened.** The gate was raised to 95% at exactly the value the tree
measured. It then went red on a **documentation-only PR**, because Linux and
macOS drift by about a tenth of a point.

The convention this repository wrote says: *never raise to a number the tree does
not clear comfortably — a gate that is red on arrival is a gate people learn to
ignore.* It was raised to the observed value with no headroom, which is the same
mistake in the same sentence that warns against it.

**What to do.** Either earn real margin, or accept that a hard threshold at the
observed value is the wrong instrument and give it a band. Both are decisions
about the gate rather than about coverage.

---

## 6. The rule the eval suite was missing

**A contract has to be honourable with the tools the harness actually permits, or
it measures the harness.**

Two memory contracts were written requiring a shell in a harness that refuses
shells by design. One of them read 5% while the evidence showed the model
honouring the contract and naming the memory out loud. Changing the judge took it
to 100%.

**The work.** The other 35 contracts have not been checked against this rule.
Any contract whose judge names `bash` deserves the same reading: is the shell the
measurement, or is it the thing that makes the measurement impossible?

---

## 7. Smaller, and each with its evidence

**A threshold of zero used to print no evidence.** Fixed in the same session:
zero means "measure and tell me", and a number with no transcript behind it is
half of what was asked for. Left here because the same shape may exist elsewhere
— a correct decision whose side effect nobody checked.

**`VerifyTools` still flags `gofmt` in `Testing tools: go test, gofmt`.**
Separating that from `Key tools: memory_store, agent_spawn` — a real finding — is
semantic rather than syntactic, and deserves its own decision rather than a guess.

**The embedded daemon leaks empty temp directories.** `startEmbedded` removes its
directory in a `defer`; a TUI killed outside that path leaves it. Seven of them in
four days of use, zero bytes each. A leak of count rather than of space, and
unlike the session record it has no pruning policy at all.

**Learned memory has no user-level scope.** Deliberately out of scope in the
spec: a gotcha from one project applied to another is worse than none. Revisit
only with evidence from project scope.

---

## Not doing, and why

**MCP.** A large surface with its own lifecycle, auth and failure modes.
Staying out was right and still is.

**Project hooks.** Commands run on tool events. Powerful, and exactly how
configuration becomes undergrowth: the session that produced this roadmap ran
under another product's hooks injecting noise into every turn.

**Sub-agents that write.** Three of the four known problems with delegation
exist *because* the child writes — conflict between children, inherited
approval, and undoing. Read-only makes all three disappear, and that reasoning
has not weakened.

---

## Suggested order

| | why here |
|---|---|
| **1** — the round ceiling is invisible | The only one with measured evidence of harm, and the machinery already exists. Wiring, not invention. |
| **6** — audit the contracts against the shell rule | Cheap, and it decides whether other numbers in this suite mean what they claim. One contract read 5% while doing the right thing. |
| **5** — the gate's margin | Small, and it is costing red CI on unrelated PRs right now. |
| **2** — retryable versus hopeless | The detector already sees it; this is a second consumer of information already collected. Narrow start. |
| **4** — a scenario that forces a discovery | Genuinely hard, possibly not solvable as a scripted fixture. Do it when there is an idea, not on a schedule. |
| **3** — the vacuous contract | Nothing to do until 4 moves. |
| **7** — the small ones | Whenever they are in the way. |

**Do not start 4 by redesigning the fixture again.** Four designs have been tried
and each redesign was pushing the model toward a behaviour rather than measuring
it. The next attempt needs a different idea, not another workspace.
