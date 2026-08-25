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

**The evidence, from the fourth self-development run.** It wrote the fix it was
asked for — reproducing test first, both excuses restored with their reasons,
`make check` green at 95.1% — and then ended at `max_iterations` on round 200
without ever saying it was done. **The work survived; the answer did not.** Read
from the outside, a run that hits the ceiling is indistinguishable from a run
that failed.

The ceiling has since gone to 2,000 (#195), which makes the encounter rarer
without making it legible. Raising a ceiling is not the same as telling someone
where it is.

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

**What was done** (#190). The drift was not drift. It was 29 blocks that one
platform executes and the other cannot, counted as uncovered on the runner that
could never reach them — while the testing convention already excluded exactly
that, in a line nothing honoured. The gate now runs once, over the union of the
matrix profiles. Measured on the run that failed: Ubuntu 94.9%, macOS 95.0%,
union 95.0%.

That removes the cause of every failure on record here. It does not create
margin: the union sits at 95.0% against a gate of 95, so the slack is a handful
of statements.

**What is left.** 371 statements are uncovered, and almost all of them are
one-statement `if err != nil` returns spread across every package. Covering ~75
of them would buy a point. Whether to do that is a real question and the answer
is not obvious: a point of headroom bought by writing tests for the number is
the gate becoming a target, which the convention two paragraphs up forbids.

**Decided.** The threshold drops to 90% aggregate, which is the number the six
`.i.spec.md` files already ask for per package, and the per-package floor of 90%
stops being printed and starts failing. The counterweight to a looser percentage
is not a tighter percentage: it is naming what must be asserted regardless — a
security-boundary crossing, a path where the user's data can disappear, a bug
seen once, an invariant in a spec. See `TESTING.md` section 3.

That leaves one thin number where there used to be another: `internal/credential`
clears the per-package floor by 0.9 points. Earning margin there is worth doing
for its own sake — those are the failure paths of reading and writing a
credential — and not because of the gate.

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

## 7. Nobody has ever run a genuinely long turn

**The gap is evidence, not design.** Contracts run 12 to 20 rounds. v16 measured
1,270 short runs. A turn of 200 rounds, with compaction firing several times and
a summary merging into itself repeatedly, is a path **no test walks**.

That is different from knowing it works. The compaction boundary logic is careful
and reasoned — it walks back to a point where no assistant message is still
waiting on a tool result — but reasoning is not measurement, and every defect
this project has found lived in the gap between the two.

**What it would take.** Not a contract: this is not about the model. A test that
drives a scripted provider through hundreds of rounds with a small window, and
asserts what has to survive — no orphaned tool call, the summary accumulating
rather than replacing, the protected turns intact, and the whole thing still
reproducible.

**Why it is worth doing before the long-running story is claimed.** "Ready for
long sessions" is a claim about behaviour at a scale nobody has observed. Saying
it without the test is the same shape as a threshold declared before measuring.

---

## 8. How to index a codebase that does not fit — the analysis, before the code

**The question.** dcode's answer today is `glob`, `grep`, `symbol` and `explore`
— a read-only sub-agent that reads the repository and returns the answer without
spending the parent's window. That is a good answer for a large repository. It is
an untested answer for a **very** large one, and nobody has drawn the line
between the two.

**What this item is.** An analysis, not an implementation. What the options
actually are, what each costs, and which of them survives this project's
constraints — written down before anyone writes code, the way the memory spec
was.

**What has to be in it.**

The constraint that eliminates most of the market: `CGO_ENABLED=0`, a single
static binary, and a CI step that fails on any `import "C"` outside the isolated
package. `sqlite-vec` is the serious answer in this space and it is C. That is
not a preference — it is what gives a binary that cross-compiles — so any
embedded index has to be pure Go or it is not a candidate.

The argument against embeddings for code, which needs testing rather than
repeating: vector search returns semantically *similar* code, and what a person
needs is usually the *exact* thing. `grep` and `symbol` are precise. The failure
mode of an index is confident wrongness.

The staleness problem, which is the strongest argument and the one least often
made: **the agent edits files constantly.** An index needs invalidation on every
write, and a stale index answers with confidence about code that no longer
exists. That is the same shape as the sandbox defect fixed in #168 — something
that describes a state that has moved.

What the reference systems in this space concluded, and it agrees with this
project's doctrine: FTS5 plus link expansion handles most queries, with
embeddings as optional re-ranking. Simple storage, simple index, no vector
database.

**And the question that comes first.** "Large codebase" can mean at least three
different problems: `glob` returning too much, `grep` being slow, or the model
not knowing where to start. Three problems with three answers, and **only the
third comes anywhere near embeddings.** The analysis should establish which one
is actually being felt before comparing solutions to it.

**What would break.** Nothing yet — this is reading and writing. The risk is in
skipping it: building an index because indexes are what large codebases get, and
discovering afterwards that the felt problem was `glob` with no depth limit.

---

## 9. Smaller, and each with its evidence

**The Homebrew tap does not exist, and the release stopped pretending it might.**
Every release generates `dcode.rb` from the signed checksums, and a step pushed it
to `aguinelo/homebrew-dcode` — a repository nobody ever created. The step was
correct by design: without `TAP_TOKEN` it exited zero and warned, so as not to
redden a release that had already succeeded. The effect was that v0.0.1 reported
success with a channel that never existed, and no one noticed until the README was
about to document it.

Machinery that runs and delivers nothing is worse than absence: it occupies the
place of a decision nobody has taken, and makes the release look complete. The
push was removed; the formula is still generated and attached, because it is the
artefact the tap will consume and it is derived from the signed checksums rather
than typed.

What it would take: create `aguinelo/homebrew-dcode`, add a `TAP_TOKEN` secret,
restore one workflow step. `scripts/formula.sh` and its test stay, so nothing has
to be rebuilt. Note the naming trap that was already there — the spec advertised
`brew install aguinelo/tap/dcode` while the script pushed to `homebrew-dcode`,
whose brew shorthand is `aguinelo/dcode/dcode`. Whichever name is chosen, the two
have to agree, and a test should hold them together.

**A boundary test passed or failed according to what was on the machine.**
Fixed in #235, and left here because the shape is the point.
`TestKeepingTheWorkspaceVisibleDoesNotMakeItWritable` asked for
`--ro-bind /tmp/ws` and got it — unless `/tmp/ws` happened to exist, because
`canonical()` resolved symlinks with `EvalSymlinks`, which only resolves a path
that EXISTS. The same directory canonicalised two ways depending on when it was
asked about, and the `/tmp` comparison tested that against a literal string.

Two lessons generalise. **A function that answers differently for the same input
depending on the filesystem cannot sit under a boundary decision** — and this one
also fed the seatbelt profile, where the comment above it already named the
damage it was doing. And it stayed invisible for a day behind Go's test cache:
every `make check` said green until `go clean -testcache`, which is the same trap
as the cached coverage number two paragraphs down, arriving by a different road.
Worth asking of any test that has never been seen red: has it ever run?

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

**The unused-name guard cannot tell a second declaration from a user.** It counts
identifier occurrences and calls a name used when the count exceeds one. Adding a
second bubbletea model made `Init` and `Update` look used — nothing in the
repository calls either; the count rose because two types now declare them. The
excuses had to be removed to stay green, so those two names are now unguarded and
the map no longer records why they were exempt. Counting declarations separately
from uses would fix it, and until then any interface method implemented twice
loses its excuse silently.

**A test cache written inside the sandbox poisons a measurement taken outside.**
Go caches a passing test by its inputs, not by the confinement it ran under. An
agent that runs `make check` inside the sandbox leaves `ok` entries recorded
under a narrower environment, and the next run outside reuses them — reporting a
coverage number that no run actually produced. Measured on the fourth
self-development run: the same tree read 93.5% with the cache warm and 95.1%
after `go clean -testcache`, and the difference was read as a failing change
until the baseline was measured. Whatever the fix is, the trap is that the wrong
number looks like an ordinary one.

**O harness de eval não responde uma chamada delegada, e isso muda o que ele
mede.** A recusa é honesta — ele não finge que um filho rodou —, mas a frase
*"Do the reading yourself with the tools you have"* **instrui o abandono**: um
`explore` de reconhecimento devolve isso, o modelo acredita, e não delega mais na
execução inteira. Medido: `delegates-writing-when-disjoint` em 75% de 12, com as
três falhas todas nessa forma, e as nove aprovações sendo as que emitiram os
filhos numa mensagem só antes de a recusa chegar. Enquanto isso não mudar, todo
limiar de delegação é piso sobre a taxa do produto e não a taxa. O conserto é um
delegador de mentira que responda alguma coisa plausível sem rodar turno filho —
e ele tem a armadilha do `shellRefusal` pela frente, porque fingir que rodou foi
justamente o que fez um modelo queimar rodadas numa mentira.

**Learned memory has no user-level scope.** Deliberately out of scope in the
spec: a gotcha from one project applied to another is worse than none. Revisit
only with evidence from project scope.

---

**The 500-line rule has never been enforced, and the tree is nowhere near it.**
`AGENTS.md` says "Keep files under 500 lines". Measured on `main` at v0.6.0:
**ten production files** and **seventeen test files** are over it, the largest
being `internal/tui/render.go` at 1817, `internal/tui/program_test.go` at 1915,
`internal/tui/program.go` at 1407 and `internal/loop/turn.go` at 1294.

This is the shape `AGENTS.md` itself names two sections earlier, about the
changelog rule: *prose is the weakest layer this repository recognises, and
until there is a guard covering it, it is worth whatever the discipline of
whoever reads it is worth.* Here the discipline has been worth about a third of
the number.

Two ways out, and they are not equivalent. Splitting twenty-seven files is a
large change that makes nothing more correct — the seams in `render.go` are
real (the approval block, the input box, the status fields) but the ones in
`turn.go` are not obviously there at all. Writing the guard first would put
twenty-seven files red on arrival, which is the exact mistake §5 records about
the coverage gate.

What is missing before either is a decision nobody has taken: whether 500 is the
number this repository actually wants, measured against what its files are, or
whether it was copied in from somewhere and never tested against the code. The
number is not evidence until something has been measured against it.

## 10. What the v5 design asks for and the product does not have

**Source: `refs/design/HANDOFF.md` (v5).** These are the parts of the design that
could not be built as client work, kept here so the specification that comes next
knows where the request came from. Everything else in the design is either built
or scheduled in the phases that follow #233.

**A tool reports nothing while it runs.** The protocol has `tool.requested` and
`tool.completed` and nothing between them, so the design's running column —
`184/184 varridos`, `7/12 testes`, `n de 240 lines` — has no origin. A card can
show what a call DID (`ToolCompleted` already carries `Lines`, `Files`, `Added`,
`Removed`, `ExitCode`, `DurationMS`, `Diff`, and its own comment already forbids
parsing them back out of the output) but not what it is doing.

The shape is a `tool.progress` event, and the cost is that it is four layers, not
one: the event, the tools emitting it, the server forwarding it, the client
drawing it. It is a versioned surface, so it is MINOR at minimum and needs a
changelog in `client-server-protocol`. It blocks the card's progress bar and
nothing else — the card itself ships without it, showing `…` where there is no
count, which is what the design says to do.

Worth deciding deliberately rather than by default: a progress event is a stream
of messages nobody stores, on a protocol whose every other event is a fact worth
replaying. Whether it joins the log or travels beside it is the first question,
not the last.

**The panel cannot show the turn's own numbers.** The design puts
`iteração 2/100` and `em vôo 2 · teto 4` under the plan, and the client has no
way to know either: the protocol carries `StopMaxIterations` as a REASON a turn
ended and nothing at all while one runs, and nothing about concurrency in
flight.

Recorded here as an omission first, which was wrong — it was checked afterwards
and it is a data gap, not a forgotten render. Correcting it matters because the
two have different fixes and only one of them is cheap.

It is the client half of item 1 of this roadmap. That item is about the MODEL
never learning it is running out of rounds; this is the person not learning it
either, in the one place they are already looking. Whatever event answers the
first should answer both, and designing it for only one of them is how a
versioned surface gets added twice.

**Naming a conversation had nowhere to be stored.** Built: the name is an event
in the conversation's own record. Left here because the reasoning generalises —
three places were considered, and the one that decided it was that **pruning
deletes transcripts**, so a name kept anywhere else outlives the thing it names.
A sidecar is orphaned; a per-workspace index keeps titles for sessions nobody can
open. The same question will come up for anything else somebody wants to attach
to a session, and the answer will be the same.

**The session rail reads the disk, and one day that will stop being true.** The
rail lists recorded conversations, and it takes them from `recordDir` the way
`dcode -r` already does — a decision taken with the design, on the grounds that
it changes no protocol and that co-location is a premise the product already has
rather than a new one.

It is still a premise. `dcode serve` and `dcode tui` are separable, so the day
someone attaches a client to a daemon on another machine, the rail simply lists
nothing. That silence is the trigger for specifying an endpoint for recorded
sessions — and the reason to write it down now is that a rail that lists nothing
reads as a broken rail, not as an architectural boundary being reached.

**The column's state is not remembered between runs.** It starts hidden, and a
`^B` lasts as long as the process. Remembering it would mean the client writing
a preference, which it has never done — `internal/tui` reads no disk and no
environment by design, and the edge injects what it needs. So the store belongs
at the edge, keyed by workspace, and it is a small config surface with a real
question inside it: whether "I opened the column for this repository" is a
preference or a habit. Recorded here rather than guessed at.

**The full border was the road not taken, and is now rejected.** Spacing
plus a `─` under the header was chosen over `┌ ┐ └ ┘ │ ─` around every tool call:
it costs no columns, survives `NO_COLOR` and ASCII without a special case, and
stays out of what copy mode selects — which the spec treats as surface.

It stays rejected, and now with evidence rather than argument. Rendering a real
recorded session at four widths, the frame that read best was the one at 80
columns — no sidebar, no panel, and not one box-drawing character in it. The
price of the border was always known: two columns and two lines per card, an
ASCII variant, and the border joining the selection when someone copies. What
was missing was a measurement, and the measurement went the other way.

---

## 11. What the v2 TUI design asks for and the product does not have

**Source: the `Coding Agent TUI v2` design in the Claude Design project
"Projeto de terminal user interface"
(`fbbd32a7-28b3-4646-9497-aa948789ccb2`).** Its central idea — three lanes down
the stream — shipped. These are the parts that did not, each with what stops
them, so the specification that comes next knows where the request came from.

**The chosen theme does not persist.** `t` cycles four of them and a restart
comes back to neon. Remembering it needs a preference on disk, and
`internal/tui` reads none by design — the edge injects. Same shape as
remembering the column's state in §10, and the same open question inside it:
whether "I switched theme" is a preference or a gesture.

**The RESULT block.** The design ends a turn with a marked block: a badge, one
sentence of outcome, the diff totals and the file list. Every piece exists in
the model already (`Verification`, `DiffAdded/Removed/Files`, the entries) —
what does not exist is the fact that says WHICH assistant block concludes a
turn. `KindAssistant` covers both mid-turn narration and the final answer, and
nothing in the event stream tells them apart. Same shape as the `tool.progress`
gap in §10: a client cannot derive it, so it is a protocol question first.

**The diff pane and its proportional bars.** Per file, a bar of added, removed
and untouched — a real improvement on the sidebar's `+188`, and NOT a
repetition of the stream, which is what got the file column hidden by default
on 24 August. It belongs in the column rather than in a third pane; the
measurement that hid the column stands, and this would be a reason to open it.

**The WATCH list.** Background processes with their state — `tsc --watch ok`,
`vitest 1 fail`, `eslint idle`. `internal/tools` has a `process` tool, so the
daemon knows; nothing reports the set of live ones to the client. A protocol
question again.

**The session pane.** Context gauge, edits accepted, tests passed and failed, a
timestamped log of recent tool calls. The context percentage and the diff
totals are already on the bar; the rest is new. A third resident pane is the
thing the 24 August measurement argues hardest against — if it ships it is a
summoned overlay, like the conversation list.

**Themes, cycled from the keyboard.** The design carries four (neon, ashes,
ember, mono) and switches with `t`. The palette gained semantic roles on 24
August, which is the foundation this needs; what it still needs is a config
surface, and a KEY THAT IS NOT A LETTER — `t` on a line where you type is the
defect fixed twice already, most recently that same day.

**`j`/`k` navigation and a `3 / 7` readout.** Same objection and the same
answer: letters need a mode that owns the keyboard, and this product has one
already (the conversation overlay). The position readout is worth having on its
own — the stream cursor moves with nothing saying where it is.

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
| **7** — a genuinely long turn | Before "ready for long sessions" is said out loud. It is a test, not a feature, and it closes a gap that is pure absence of evidence. |
| **5** — the gate's margin | Small, and it is costing red CI on unrelated PRs right now. |
| **8** — the indexing analysis | Reading and writing, no code. Do it before anyone is tempted to build an index, and start by establishing which of the three problems is actually being felt. |
| **2** — retryable versus hopeless | The detector already sees it; this is a second consumer of information already collected. Narrow start. |
| **4** — a scenario that forces a discovery | Genuinely hard, possibly not solvable as a scripted fixture. Do it when there is an idea, not on a schedule. |
| **3** — the vacuous contract | Nothing to do until 4 moves. |
| **10** — what v5 asks for and we do not have | After the client phases land. The card ships without progress, so the protocol event is not blocking anything visible — and deciding it under pressure from a half-built card is how a versioned surface gets the wrong shape. |
| **9** — the small ones | Whenever they are in the way. |

**Do not start 4 by redesigning the fixture again.** Four designs have been tried
and each redesign was pushing the model toward a behaviour rather than measuring
it. The next attempt needs a different idea, not another workspace.

**Do not start 8 by choosing a library.** The first question is which problem is
being felt, and two of the three answers have nothing to do with indexing.
