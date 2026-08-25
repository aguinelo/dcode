# Changelog

🇧🇷 [Versão em português](CHANGELOG.pt-BR.md)

A living record of what changes in dcode. **Every change lands here**, at the
top, on the branch that makes it — not afterwards, not in batches.

The current state is the first section and is rewritten alongside each entry.
One file rather than two on purpose: a status that lives apart from the log is a
status that ages on its own, and this repository has scar tissue from declared
things nobody maintains.

The fine detail of a decision stays in the per-family changelogs, under
`docs/specs/architecture/<family>/changelog/`. What lives here is what changed
and why, one line each.

---

## Current state — 24 August 2026

**What it is.** An agentic coding harness in Go: a daemon, a terminal client and
the agent loop between them, as a single static binary, with no cgo outside the
isolated package.

**Where it stands.**

| | |
|---|---|
| spec families | 13, with 106 decision changelogs |
| behavioural contracts | 42 declared |
| **contracts measured against a model** | **3** |
| coverage | 94.0%, gate at 90% |
| CI | macOS + Linux matrix, gated on the **union** of the profiles |
| published version | **0.5.0** |

**Getting it.** `curl … install.sh | sh`, or `go install`. Nothing else has to be
installed first — of rustup, bun, deno, nvm, k3s and uv, not one requires an
external verification tool, and a first install is the worst moment to ask.

The SHA-256 always runs. What says the digest should be that value arrives by two
independent routes, and **either covers a substituted release**: the digest the
installer carries, committed to `main` where changing a line is a visible commit,
and the cosign signature, used when it happens to be on PATH. `dcode update`
applies the same rule by reading the digests from the installer on `main`.

Homebrew is not a channel yet — the tap was published to for one release and had
never been created. Removed rather than left running; `docs/ROADMAP.md` §9 says
what creating it would take.

**The interface.** The conversation gets the terminal. The file column starts
hidden and `^B` summons it; the conversation list is an overlay on `^R`, which
is what that key means in the shell it was borrowed from; the panel opens at its
floor and grows out of the surplus. Every question opens with a rule, so a
screen of scrollback has a boundary in it. Delegation is one card with its
children inside, and the child that did not answer is named there with its
reason. A tool call appears the moment it begins arriving from the model, and a
boundary crossing is asked in the stream, in its own lane, keeping its place
with the answer once it has one.

That shape came from a measurement rather than a preference. Replaying a real
recorded session at four widths, the column and the panel took 61 of 132 columns
and left 71 for the conversation, while the same session at 99 columns — where
both disappeared — gave it 99. **Widening the terminal made the text narrower**,
and the crossing was a single column, because two thresholds sat at the same
hundred. What the column held was a second copy of what the stream had just
said.

**Where the keyboard is.** The input area is a framed field, because the one
question with no other answer on the screen is where the letters you type go.
Nothing on that frame carries state: an earlier version dimmed it while the
stream had the keyboard, and its own test asked whether that distinction
survived without colour. It did not.

Copy mode is `^O`. It was `v` twice, and the second time is the instructive one:
the first fix required the stream cursor to be in the stream, which **narrowed
the rule instead of applying it**. The input line is always a line where you
type, so no condition could satisfy "a letter is not a shortcut" — only giving
the letter back could.

**What the guards could not see.** Eight of the defects fixed on 24 August had
guards written for exactly them, and every guard was asking about a set it
already knew. The box-drawing guard derived its forbidden glyphs from the two
glyph tables, and the approval screen — drawn from literals, in English, the one
screen that asks whether a boundary may be crossed — was outside both, in two
different ways, found twice in one day. The width guard split on newlines before
measuring, so a row broken in two measured as two short rows. The blank-row
guard trimmed correctly and had never been shown prose. Each is now asked as a
question about the whole screen rather than about a list.

**Security, on two axes.** Containment is the sandbox — Seatbelt on macOS,
bubblewrap on Linux, with the boundary tested against the kernel and exercised
in CI. Authorisation is the approval policy plus the rules. The two are
orthogonal, and that separation is what makes it possible to be permissive
without being unsafe.

The sandbox today: hides the credential stores by default (`~/.aws`, `~/.gnupg`,
`~/.kube`, `gcloud`, `~/.netrc`, `~/.docker/config.json` and dcode's own key);
keeps a container runtime's socket out of reach; grants a socket or a writable
path **by name**; and hides `~/.ssh` as soon as the `ssh-agent` socket is
granted — because then ssh signs without reading the key and hiding costs
nothing.

**Delegation.** A delegated child writes, inside what it declared it owns, with
the parent's containment narrowed to that set. Ownership is a boundary, not an
agreement.

**What this document does not say.** That the system is verified. Thirty-nine of
the forty-two contracts have never run against a model, and the suite prints
that on every run to stop the opposite reading.

---

## Unreleased

### Fixed

- **The meter on the screen is the one that was fixed.** The context meter was
  corrected in the model and the bar went on drawing from the cumulative input
  count beside it, so it read `ctx 591%` — in a colour computed from the true
  percentage, so the number disagreed with its own colour. Replaying a real
  record of 3163 events: `input_tokens 5917178` against a million-token window
  is the 591; `context_tokens 363500` is the 36 the bar now shows. The cap moved
  into the function that turns the number into text, because it had been written
  down only on a field the screen did not pass through — and the guard was
  extended to ask the screen instead of the field it had just watched change.

## 0.5.0 — 24 August 2026

### Changed

- **The approval is in the stream, and stays there.** The modal is gone: the
  question is drawn where it was asked, in a fourth lane, and once answered it
  keeps its place with the answer in place of the keys. The box was read as
  being what enforced RN-6, and it never was — what owns the keyboard while a
  crossing is pending is the client refusing to hand the keystroke to the input,
  which is unchanged. What the box did do was hide the work being judged and
  then delete itself, taking with it the most durable record a session
  produces. Answers now land on the request by `ApprovalID`, because with two
  crossings in flight "the last one" writes a decision nobody made.

### Fixed

- **Continuing a long conversation opens it.** `dcode -c` painted the splash
  screen and exited, leaving the terminal's answers to its own startup queries
  typed into the next shell prompt. Three failures in a line, each hiding the
  next. The response to creating a session was built *before* the continued
  conversation was put in it, so it said the session held nothing; the client
  believed it and asked for events from 1, which retention had already dropped
  from a conversation of eighteen thousand; the refusal was then written to the
  error channel and lost in a race with that channel closing, so the client
  quit saying nothing. The session now describes itself after the conversation
  is in it and reports the earliest event it still holds, the client asks from
  there, and a reason that arrives with a closed stream is read before the
  closure is.
- **A fatal outlives the screen it was drawn on.** It was written into the last
  frame, and the alternate screen takes the last frame with it — so the one
  message the person needed was the one guaranteed to be wiped. Failing looked
  exactly like doing nothing.
- **Only a release tag names a build.** A backup tag left beside the branch —
  `tui-v1`, a restore point and not a version — shadowed the last release simply
  by being newer, and every build then called itself `tui-v1-dev+411c237`.
  Naming a build after something that is not a version is the same defect as
  naming it after the version it has already left, which is what the derivation
  exists to prevent. Both the script and the Makefile now match `v[0-9]*`.
- **The version reads what the history actually holds.** `scripts/version.sh`
  refused to derive anything the moment a merge or a revert appeared: a merge
  commit's subject is not a change — the changes are its parents', already in
  the range — and `Revert "feat: …"` matches no convention because it quotes
  one. A merge is skipped; a revert is classified by what it undid, since
  removing a feature is a change of the same class the addition was. The
  refusal itself also printed one word per line, which is how a message written
  to be read by whoever will fix it arrives illegible.

- **A record stops copying what it continues.** Continuing copied the whole
  previous record into the new one, so a session that continued a session that
  continued a session held three copies of the first — the largest record on this
  machine is 3.6 MB and 18,410 events, most of it itself, repeated. The carried
  conversation goes to the log and not to the record; the record keeps the
  marker, and reading one follows the chain back. Growth is linear now.
- **Resuming paints once.** Continuing writes the whole of the old log into the
  new session, so attaching replayed every event of it — 3544 on a real session
  — and Bubble Tea paints after every message, so the screen redrew 3544 times
  with the window following its own end. It shows one line while it reads, with
  a count, and the conversation once when it catches up. The line moves: a
  session reading history is IDLE, and a still spinner under the word "reading"
  is how a stuck screen looks.
- **The context meter measures the context.** It read `ctx 175%`, which is not
  a context that is 175% full — it is a turn that spent 1.75 windows of input.
  `InputTokens` is cumulative across a turn's rounds, and every round re-sends
  the context. The daemon now states what the assembled context costs, using the
  same estimate the compaction trigger reads so the meter and the threshold
  agree by construction. Providers could not answer this: the two families
  disagree about whether their input count already includes the cached prefix.
- **A Chromium reaches its first frame inside the sandbox.** Without
  `mach-register` and a scoped `iokit-open`, any Chromium died with SIGSEGV
  before drawing anything — Playwright, Puppeteer, Lighthouse, an Electron app
  under test. It hid because it was a SIGNAL and not a denial: a refusal says
  `Operation not permitted` somewhere a person can read it, and a crash says a
  stack trace with nothing anywhere naming the boundary. What the screen showed
  was a browser breaking, and it was read as the model being timid. It is a
  pair — neither alone gets past the crash — and `iokit-open` is scoped to one
  user client, which is what makes it affordable.

### Added

- **The preserved tail has two floors.** `KeepTurns` counted turns, and a count
  is the wrong unit: turns vary by an order of magnitude, so four short ones
  protected almost nothing — the summary ate a forty-tool investigation and kept
  four "ok"s. `KeepFraction` (0.30) is a floor in tokens of the window, measured
  with the same estimate the trigger uses, and whichever protects more wins. The
  rule that the current task never gets compacted still sits above both.
- **The context says it is filling before it is cut.** The bands were computed
  and announced to the model, and nobody announced them to the reader — so the
  summary arrived as one line saying it had happened, after the fact, with no
  chance to finish a thought first. The crossing now reaches the client at the
  same moment it reaches the model, and the cut says how many messages went and
  how many stayed instead of only that something did.
- **A mode where a letter is a key.** `esc` from an empty line steps into the
  transcript, and inside it `j/k` move, `↵` opens, `t` cycles the theme and `/`
  goes back to writing. Every key the mode does not name is swallowed, which is
  what makes a letter safe there — the design's footer offers those letters and
  puts a NAV badge beside them, and a badge is the name of a mode. `↑` at the
  border now scrolls instead of walking into the transcript, which removes the
  state the `v` defect kept coming back through.
- **Four themes: neon, ashes, ember, mono.** The design's own values, on one
  shared role mapping — change what colour a heading is and all four change. The
  contrast test runs on every one of them.
- **The side column is the diff pane over the session pane, on the right.** It
  replaces the file list, and the difference is what got the file list hidden
  this morning: that column repeated what the stream had just said, and these
  two do not — a bar of the change, a context gauge, how much of what was asked
  the person allowed, the last calls by the clock. The default is reversed
  again, and the width test has now been written three ways in one day, which is
  said in its comment rather than edited quietly for the third time.
- **A lane legend, and a nav bar.** The legend appears once at the top and only
  when the screen is making more than one lane. The nav bar names the keys that
  are keys — the design also offers `j/k` and `t`, which are letters, and those
  belong to a mode that owns the keyboard.
- **The interface has a palette of its own.** Neon: a violet ground, a magenta
  mark, teal for what worked, amber for the person. Until now the roles mapped
  to ANSI codes chosen to sit politely inside whatever the terminal's theme was,
  and that politeness is what made the screen read as grey. A theme carries its
  own ground, which is the decision and is not free: the interface stops
  inheriting the terminal's colours and starts owning them. Colour switched off
  gets none of it — no escape reaches the screen, the ground included.
- **The plan moves into the stream.** It was a column of its own; it is a block
  where the model made it, always showing the current plan, updated in place. The
  panel is dissolved with it and its ceiling readout rides the status bar, so
  `-no-panel` and `^P` are gone — a contract removed from a stable surface.
- **The stream has lanes.** Every row says which of three things it is — what
  you asked, what the model did on the way, what it says — marked by a character
  in the first column. On a long turn prose and tool calls alternated with
  nothing structural between them, so catching up meant reading every row to
  find out which rows were worth reading; now the eye runs down the answer lane
  and skips the work. It costs no columns: every row already reserved two, and
  the lane takes the first while the selection marker keeps the second. From the
  `Coding Agent TUI v2` design; what did not come from it, and why, is in
  `docs/ROADMAP.md` §11.

## 0.4.0 — 24 August 2026

Two reports from the person using it, and both were the same shape as the
release before: a rule that had been narrowed instead of applied, and a state
the screen never showed.

### Changed

- **Copy mode is `^O`, and `v` is a letter.** It was `v` twice. The first time a
  bare `v` on an empty line ate the first character of anything starting with
  one; the fix required the stream cursor to be in the stream, which narrowed the
  rule rather than applying it — and the same report came back, by a path a test
  now walks: `↑` on a session with no history walks into the stream, and the next
  `v` typed there was a shortcut again. The input line is always a line where you
  type, so no condition could satisfy the rule; only giving the letter back
  could. Typing also returns the focus to the line being typed on, so browsing
  and writing stop being two states at once.

### Added

- **The input area is delimited on all four sides.** A frame here and not around
  a tool call, which is what a box is for: the input is a *field* — a fixed
  region that does not scroll, that you return to, and that has to be findable
  without reading — while a tool call is content, and a frame around content is a
  frame around what you were already reading. The frame carries no state: an
  earlier version dimmed it while the stream had the keyboard, and its own test
  asked whether that survived without colour. It did not.

## 0.3.0 — 24 August 2026

The release the interface actually needed, and the first one where the defects
were found by **replaying a real recorded session through the reducer and
rendering it**, rather than by a state I chose. Every entry below was found that
way or by a guard that had to be rewritten to ask a different question.

The shape of the whole release: *a rule with one exception has more*, and *a
guard that asks about a set only ever finds what is already in the set*.

### Changed

- **Text has a hierarchy instead of one dim.** `StyleDim` meant five different
  things at forty-seven call sites; there are six roles now, and the mapping is
  one decision in one table. The first thing that decision changed: the model's
  prose is no longer drawn faint. Dimming a sentence does put the eye on the file
  name inside it, and dims the answer to do it — so the contrast is bought with
  the term instead, which is one word rather than a paragraph. A terminal has
  three weights that survive an unknown background, and the invariant says so.

- **The panel pays for the width it takes.** It arrived owing a quarter of the
  screen the instant it was allowed to appear, so crossing from 99 to 100 columns
  cost the stream twenty-five of them in one step; it opens at its floor now and
  grows out of the surplus beyond that threshold. And the TURN section, which
  exists to warn that a ceiling is coming, was drawn from the first event of
  every session — spending thirty-three columns to say `iteração 0/2000`. It
  appears from half the ceiling, and whenever every in-flight slot is taken.

- **Every conversation row says when and how much.** The overlay fixed the width
  and left the real problem showing: four rows read the same because four
  conversations began with the same question. The meta takes its width before the
  title — the opposite of the rule the file rows follow, because when the titles
  collide the date is the only thing that tells them apart. `relativeDay` takes
  the clock as an argument now: the picker could read one, the overlay is inside
  a render that is pure over the model. And `%d turn(s)` became a real plural.
- **The conversation list is summoned, not resident.** `^R` in readline is a
  search you summon — it appears, you choose, it goes — and borrowing the key
  while making it twenty-six permanent columns contradicted the convention that
  justified borrowing it. It is an overlay now, the way the approval modal
  already was, with sixty-four columns to show a title in instead of twenty-six.
  `RailNav` did not move: the cursor, the filter, the naming mode and every test
  over them are unchanged, and only the drawing changed place.
- **The file column starts hidden.** Measured on a real session: at 132 columns
  the column and the panel took 61 of them and left 71 for the conversation,
  while the same session at 99 columns — where both disappear — gave it 99.
  Widening the terminal made the text *narrower*, and the crossing was a single
  column: 99 gave the stream 99, and 100 gave it 53. What the column held was
  also a second copy of what the stream had just said. `^B` summons it and it
  stays as it was left, which is what the key means in the editor it came from.
  A contract change on a `stable` surface, so MINOR at least.

### Added

- **A turn begins with a visible boundary.** A question used to be a mark in the
  same weight as the prose around it, so a screen of scrollback had no boundary
  anywhere. Every question now opens with a rule, inset to the same gutter the
  rest of the stream uses. A rule and not a colour: a question picked out by
  colour alone is not picked out at all on a monochrome terminal, and this is the
  landmark the eye scrolls to. It costs one row per turn and no columns.

### Fixed

- **A marker still arriving is not drawn.** Every emphasised word reaches the
  screen as `**` first and its partner some deltas later, so `1. **` sat alone as
  the last line of the stream before each heading the model wrote. Dropped only
  when it is at the end of the text and unpaired: a marker somebody opened and
  left mid-sentence was written on purpose, and a text ending in `**` because a
  pair closed there is a finished pair.

- **Every screen speaks the interface language.** Nine English literals sat in
  the drawing code, the whole approval modal among them — the one screen that
  asks whether a boundary may be crossed, in a language the reader may not have,
  and consent given to a sentence somebody could not read is not consent. The
  existing guard asks whether every declared string has a translation and cannot
  ask whether the renderer uses them; the new one derives its forbidden set from
  the catalogue itself, so it grows as the catalogue does.

- **Prose leaves one blank row between paragraphs, and it is empty.** Splitting
  `"a\n\n"` yields three parts and the last is the end of the text, not a
  paragraph; a run boundary at a `**` marker split the same text twice, so a
  block came out with three blank rows between two sentences. They were also
  indented, making them two spaces rather than empty — and every rule about blank
  rows here compares against `""` or trims, so both readings passed over them.
  The invariant existed and its guard already trimmed; what it had never seen was
  prose, because the fixture was made only of tool calls.

- **A tool line stays on one line.** A shell command wrapped over several lines
  was written into the frame as several lines, and everything after it — the
  sidebar, the divider, the panel — was out of alignment for the rest of the
  screen. Flattened at `clipStyled`, which every line of every column passes
  through, so the guarantee holds for the next one-line field too.
- **A command keeps its beginning, a path keeps its end.** The elision kept the
  tail for both, so four different searches drew four identical rows reading
  `… | sort -u | head -40`. What it keeps is now decided by the value rather
  than by the tool, reusing the one definition of "is this a path" the package
  already had.
- **A line that was cut says so.** The sidebar states the rule for a
  conversation title and the panel answered it the other way — `✓ 6 CLI sob
  demanda com contr` just ended — while the sidebar itself did not apply it to a
  file name, where `client.py` and `client.pyi` differ by what is missing. Both
  columns mark the cut now, and elide before styling, which is the order the
  palette's contract asks for.
- **ASCII reaches the approval modal.** The modal was drawn entirely from
  literals with no fallback, so the one screen that asks whether a boundary may
  be crossed was the one screen a terminal in ASCII could not read. Seven other
  leaks went with it. The guard asked whether a *known set* of glyphs escaped,
  which only ever covers what the glyph tables already know; it now asks whether
  every rune is ASCII, over a model built entirely from ASCII.
- **The sidebar counts a file once.** The same file arrived under two spellings
  — `DCODE.md` from one call, `/Users/…/craw/DCODE.md` from the next — and drew
  two rows, two line counters and a header claiming fifteen files were touched
  when eleven were. Normalised against the workspace where the target enters the
  model, so the tool line gets the short path too; a path outside the workspace
  keeps the spelling that finds it rather than becoming a `../..` ladder.

## 0.2.0 — 23 August 2026

The release the interface needed, and the one every entry below was found by
somebody **using** it rather than by a test.

The v5 design is built: a sidebar with the files this turn touched and the
conversations this workspace has recorded, delegation drawn as one card with its
children, a verb on the activity line, and a tool call that appears the moment it
starts arriving instead of after it has finished.

Four defects in it were found the same way — by opening the product and saying
what was wrong — and none of them by the tests that were supposed to cover the
same ground. That is recorded here because it is the most useful thing this
release taught.

### CLI

- **A dev build is named for the version it is heading to.** It took the last
  tag, so every build between two releases reported the **older** one: a binary
  carrying two days of work called itself `0.1.0`, and the only thing saying
  otherwise was a commit hash nobody reads. Somebody watched that number sit
  still and reasonably concluded nothing had been installed. It now derives from
  `scripts/version.sh`, so the same build reads `0.2.0-dev+7b27519`.

- **`-v` prints the version.** It answered *"flag provided but not defined: -v"*
  and then printed the entire usage, burying the one line that says what went
  wrong under twenty that do not. `-h` was already there beside it, and a pair of
  one-letter flags where only one exists is a pair somebody gets wrong every
  time.

### Protocol

- **A tool call appears while it is still arriving.** On a write of a few hundred
  lines the model streamed the whole file and **the screen showed nothing** — not
  even the tool's name — until the call was fully assembled, then it appeared
  already complete. Silence through exactly the part of the turn where the work
  happens, which is what makes a live interface read as a dead one.

  Both facts were already known and both were thrown away: the name and id at
  `content_block_start`, the byte count on every fragment. The provider now emits
  `tool_call_opened` and `tool_call_progress`, and the loop turns them into
  `progress` with `kind: "arguments"`.

  **Not a new protocol event** — the existing one, with a `Name` field, because a
  subject that does not exist yet has to name itself: `tool.requested` carries
  the name and only arrives once the call has finished assembling.

  Bytes rather than lines, since what has landed is a fragment of JSON and
  counting lines inside half an escaped string counts something that is not there
  yet. No total, because the model does not say how long the call will be and a
  denominator nobody sent is one somebody would trust.

  Throttled at half a kilobyte, in the **loop** rather than the provider: the
  provider reports what it sees, the protocol decides what is worth saying. The
  first report is always sent — it is the one that puts the line on screen.

  A consumer that ignores both new events sees exactly the sequence it saw
  before, and there is a test for that.

### Client TUI

- **A column that hides itself says so.** The sidebar disappears below a hundred
  columns — which is most terminals — and said **nothing at all**, so a column
  that had been built read as a column that had not, and the key that brings it
  back (`^B`) was documented only inside the column that was not on screen.

  Found the only way it could be: by somebody opening the product and saying the
  interface was not what they designed. Every verification behind it had been a
  Go test calling `Render` in memory, at widths I chose — never the binary in a
  terminal.

  The threshold stays at a hundred, because the reasoning holds: a 20-column
  sidebar on an 80-column terminal leaves 59 for the stream, and a diff in 59
  columns is bad. The defect was the silence. The plan panel had already paid
  this debt and already carried a hint; the sidebar inherited the behaviour
  without it.

### Protocol

- **A conversation can be given a name, stored in its own record.** Three places
  were considered: a sidecar per session, one index per workspace, and the
  record. The record wins on the thing that decides it — **a name for a
  conversation that no longer exists is worse than no name.** Pruning removes
  transcripts, so a sidecar is orphaned and an index keeps titles for sessions
  nobody can open. Here the name dies with what it named.

  It also keeps the count at one: a store beside the log is a second thing that
  can disagree with it. And it costs nothing to read — `Browse` already scans
  every line of every record to count turns.

  The sequence is **read before appending, never assumed**: putting a number
  already in the file would leave a duplicate in a log whose whole contract is
  that there are none. Renaming twice is somebody changing their mind, so the
  last one wins.

  An empty name restores the derived title and is not an error — one operation
  with a meaningful zero value is one thing to get right. Control characters
  never reach the record, because it is read back line by line and a newline
  inside a name would make one line look like two. A name too long is
  **refused rather than trimmed**: silently keeping half of what was typed is how
  somebody ends up with a name they did not choose.

  It writes to the record rather than to the live session, because the rail
  lists what a workspace has recorded and almost none of it is loaded — a rename
  that only worked on the open conversation would work on the one row nobody
  needs it for.

### Client TUI

- **`r` and `F2` name the conversation under the cursor.** Naming is its own mode
  inside the list, because it is the one thing there that changes something:
  while it is open **every key belongs to the name**, so nothing else is
  reachable by accident halfway through.

  The draft starts from the **name**, never from the derived title. Offering the
  title would turn *give this a name* into *confirm the one you were given*, and
  the first Enter would promote a derived title into a chosen one with nobody
  deciding. `esc` cancels and keeps what was there.

  A given name carries a `·`. Without the mark the column shows two kinds of
  claim — derived and chosen — and nothing tells them apart.


- **A scan says how far it has got, and a result lands on its own call.**
  `kind: "files"` joins the declared set: `grep` says `n of N` because it has the
  list before it starts, `glob` sends the count alone because it is still
  discovering, and the card shows `150/184` where an ellipsis used to be.

  The reporter travels **on the call's context, not on `State`**. State is per
  session and shared, so two scans running in parallel would write their counts
  through one field and the screen would show one's progress under the other's
  name. `Progress(ctx)` never returns nil: a tool should say how far it has got
  without first asking whether anybody is listening.

  Still no kind for lines or tests, and that is a finding rather than an
  omission. `read` takes the whole file and splits it, so it learns the total at
  the same moment it learns the content — there is no point at which "n of 240"
  is true. Counting passing tests would mean parsing `bash` output, which
  `ToolCompleted`'s own comment forbids. **A kind that could only be filled
  dishonestly does not get declared.**

  Reports go out every twenty-five files rather than every file: one per file
  would put ten thousand lines nobody reads into the record of a ten-thousand
  file scan.

  **A latent defect surfaced on the way.** `ToolCompleted` matched the *last
  running* entry, which is right exactly while one call runs at a time — with
  two in flight, the first result landed on the second call's line. Real numbers
  on the wrong row. `Entry.CallID` fixes the routing of both the result and the
  progress, and it was found because progress needed the same addressing, not
  because anybody noticed the wrong screen.


- **`progress`: one event for "how far along".** A tool counting files and a turn
  counting rounds are the same question asked of different subjects, so it is one
  event with a `tool_call_id` that is empty when the subject is the turn.
  Adding a versioned surface twice for one kind of question is how it comes out
  crooked — the second one always answers slightly differently.

  `kind` is a **closed set rather than a word to print**: the daemon's language
  is not the reader's. Only what something actually emits is declared, so
  `rounds` and `in_flight` are there and `files`/`lines` arrive when tools emit
  them. `tests` probably never will — counting passing tests means parsing
  `bash` output, which `ToolCompleted`'s own comment forbids.

  **It joins the sequence**, and that was the hard call. Leaving it out of `Seq`
  would have put a gap in the one property the record is built on, and a record
  with a hole is a record whose replay cannot be trusted about anything else
  either. `message.delta` is already chatty and already in there; this follows
  it rather than inventing an exception.

  The ceiling travels with the count, because a count without its limit answers
  *how many* when the question is *how close*. And a turn that answered in one
  pass reports no round at all: there is no ceiling approaching, and `0/100` on
  screen is a figure that means nothing is happening.

### Client TUI

- **The panel shows where the turn stands.** `iteração 2/100` and
  `em vôo 2·4`, and the count changes style as it nears the ceiling — that
  ceiling is roadmap item 1, the one item with measured evidence of harm, and
  what it lacked was anything saying it was coming.

  The panel now opens on those numbers alone. Most turns have no plan, so the
  ceiling was hiding in a panel that only opened when something else was already
  there. The numbers outlive the turn that produced them, so it opens on the
  first turn and stays rather than appearing and leaving with every one.


- **`^R` gives the sidebar the keyboard.** `↑↓` move, a letter filters, `enter`
  continues the conversation under the cursor, `esc` clears the filter and then
  closes. The second of the design's three rail modes; the third, naming, still
  has nowhere to be stored.

  Owning the keyboard is not a flourish: a list you move through with keys that
  also type into the input line is a list where every keystroke does two things,
  and the one time it does the wrong one it opens somebody else's afternoon. The
  block sits **above** the completion-menu guard for the reason the copy-mode
  changelog records — placed inside it, the mode would never have run at all,
  and nothing would have said so.

  Each small decision carries its reason: the cursor is a character rather than
  a colour and wins over the open-conversation mark, because with the keyboard
  here the question is *which one am I about to open*; `↑↓` do not wrap, reusing
  the picker's own argument; `esc` backs out one thing at a time; typing returns
  the cursor to the top, since the list on screen is now a different one; a
  filter matching nothing chooses nothing **and says so** rather than going
  blank; choosing the conversation already open does nothing.

  A fifth hard-coded Unicode rune slipped in — the filter caret — and #243's
  guard did not catch it, because it enumerated the runes by hand. It now
  **derives** them from the glyph sets by reflection, so a new mark joins the
  prohibition on its own.

### Documentation

- **What the v5 design asks for and the client still does not show.** Found by
  the person running the interface and saying it looked poorer than the drawing,
  which is the only way a gap like this gets found.

  Delegation was the large one and is **built** (see below). What remains is the
  panel's turn numbers — `iteração 2/100`, `em vôo 2 · teto 4` — and that is a
  **data gap rather than a forgotten render**: the protocol carries
  `StopMaxIterations` as a reason a turn *ended* and nothing at all while one
  runs. Recorded as an omission first and corrected after checking, because the
  two have different fixes and only one is cheap.

  It is the client half of roadmap item 1. That item is about the *model* never
  learning it is running out of rounds; this is the *person* not learning it
  either, in the place they are already looking. Whatever event answers one
  should answer both.

### Client TUI

- **No box-drawing rune reaches a terminal that cannot draw one.** Four separate
  literals in this package assumed Unicode — the column divider, the diff
  gutter, the running marker and the path ellipsis — and each was found by
  looking at an ASCII render, **after** the previous one had been fixed. The
  divider went in #241; the other three went here.

  So the guard is over the whole screen rather than over one glyph at a time. A
  fifth would otherwise wait for a fifth pair of eyes.

- **The sidebar lists this workspace's conversations.** Under the files, with the
  open one marked by a character rather than only by colour. It is `dcode -r`
  promoted to a permanent column, in the mode the design calls *passive*.

  Same source and same filter as `-r` — `session.Browse` through `choicesFrom`,
  read once at start by the edge, because two ways of listing a workspace's
  conversations would eventually disagree about which exist. Conversations
  nobody asked anything in stay out; that is most of what a record directory
  holds. The client still reads no disk: the list arrives through `Options`, like
  the language.

  **Naming a conversation could not ship, and that is written down rather than
  approximated.** A name the person gives has to outlive the session, and a
  record directory holds transcripts, not titles — the title is derived from the
  first question every time it is read. That is a change in `internal/session`,
  and `docs/ROADMAP.md` now carries it with the three places it could live and
  why only one survives pruning. `^R` navigation waits on the same decision in
  part; `/resume` already does the continuing.

  Details the screen decided: conversations alone are enough to open the column,
  because asking only about files emptied it for the first minute of every
  session; a cut title says it was cut, and is cut in **cells** rather than
  bytes, since a rune is not a column and a title with an accent is where that
  goes wrong.

- **A sidebar shows what the turn touched.** Files, their state, and the line
  count of the ones that finished. `^B` folds and unfolds it. It is
  `clamp(20, w/5, 30)` wide, gone below a hundred columns, and an explicit
  choice wins at any width in both directions — the panel's manners exactly,
  because answering one question two ways would give two columns different
  behaviour on one terminal.

  **Derived, not stored.** The handoff puts a `tree` field on the model; this is
  a pure function over `Entries`, which are already the reduction of the log. A
  field would be a *second* reduction of the same events, and two reductions can
  disagree — deriving makes "the same session reopened reproduces the same tree"
  true by construction rather than by care. `Entry` gained `Added`/`Removed` as
  numbers, because reading a count back out of the summary sentence is what the
  protocol comment forbids in as many words.

  **Two levels, not a full tree.** The column is twenty to thirty characters and
  every level of indentation takes two of them from the only part that
  identifies a file. The folder row carries its whole path instead.

  Four defects only the screen showed: indentation drifting a level after the
  first file (path depth is not visual depth once a folder is compacted), the
  count printed twice, `+38` sitting against the divider and reading as frame,
  and the divider itself hard-coded to `│` — **which the plan panel already
  did**, visible only once a second column repeated it. Both now follow
  `g.Unicode`.

- **The expansion hint speaks the interface language.** Under a collapsed body it
  read `⋯ 42 lines · Tab expande` — one line, two languages, in **both**
  interfaces: the count was hard-coded English and the verb hard-coded
  Portuguese, and neither followed what the user had chosen.

  Same family as #238 and found the same way, by reading the output rather than
  the diff.

- **A tool call that carries a body reads as a block.** One blank line separates
  it from what is around it; a call with no body stays a single line, because
  most calls are one line and a card around one line is a box around nothing.

  It is this small because almost all of the design's §3 was already built —
  the `…` while running, the duration only past 500ms, and the whole finished
  meta column (`240 lines`, `created, 38 lines`, `+24 −2`, matches in files,
  `exit 0`) in `summariseResult`. And the card itself already existed in the
  units a terminal has: `detailLines` draws a `│` down the left of every body
  line, which is the spine tying body to header. What was missing was the
  breathing room the handoff asks for, not a frame.

  The rune border stays a recorded preference with its price — two columns and
  two rows per call, an ASCII variant, and the border joining what copy mode
  selects — and it would do nothing the gutter does not.

  **The gap goes before, never after**, and that was measured rather than
  guessed: put after, it pushed the changed line of a diff off a 40-row
  terminal, because the window is anchored to the end and a trailing blank costs
  a row of what happened to show nothing.

- **The activity line speaks one language, the way out included.** It said
  `^C interrupts` in English under a Portuguese interface. The verb made it
  obvious by sitting right beside it — `lendo grep … ^C interrupts` — and half a
  sentence in each language reads as a bug in the product rather than as a
  missing translation.

  Its own catalogue entry rather than the existing `Interrupt`, which is the
  `esc` hint somewhere else: two keys sharing one sentence is how a hint ends up
  naming the wrong key in one of the languages.

- **The activity line carries a verb, and the verb never appears alone.** A
  short gerund now rides beside the running tool — `⏺ reading grep \.Save\(` —
  drawn from the phase's set and changing every 20 frames, which is the design's
  2.4 seconds at the 120ms tick. `DCODE_ACTIVITY_VERBS=0` turns it off, taking
  the word and leaving the facts.

  Alone it would be motion pretending to be information: the screen looks alive
  and the reader learns nothing. So it is only drawn next to a running tool, dim
  against the fact's bold — what moves is the accompaniment, what is true is the
  emphasis — and with no tool the line says its one plain word and stays still.

  Found while building it: `working` was both the no-tool word **and** a verb in
  the `other` set. With one string in both roles, nobody can tell a rotating verb
  from a still one — not the reader, not a test. The fallback word also joined
  the language catalogue, where it should have been all along.

- **The tick stops when the session is idle.** It already refused to advance the
  frame, and the comment said why — *"an idle screen that keeps repainting burns
  a laptop battery for no information"* — while rescheduling anyway, so the
  screen repainted eight times a second for a number that never moved. The
  sentence was right; it just was not being kept.

  It restarts when a turn begins, with a guard so exactly one comes back:
  without it every event would add a tick and the frame counter would sprint,
  which is motion claiming the machine is busier than it is. Nothing is lost by
  stopping — `Now` is refreshed on every event.

### Sandbox

- **A boundary decision follows the mode, not the filesystem.** `canonical()`
  returned a path unresolved when it did not exist yet — `EvalSymlinks` only
  resolves what does — so the same directory canonicalised two different ways
  according to *when* it was asked about: `/tmp/ws` before creation,
  `/private/tmp/ws` after. The `/tmp` comparisons then tested it against the
  literal `"/tmp"`, and whether the workspace was remounted over the tmpfs came
  down to what happened to be on disk.

  It now resolves the deepest existing ancestor and puts the rest back, and the
  comparisons run against `/tmp` as `canonical()` itself reports it.

  This was found as a test that passed or failed by machine, hidden for a day
  behind Go's test cache. But the same function feeds the **seatbelt** profile,
  and the comment above it already named the danger it was causing: on macOS a
  profile naming an unresolved path *"grants nothing and every write fails with
  no explanation"*. Production `bubblewrap` is Linux-only, where both spellings
  already agree, so no argument list changes there.

  `TestCanonicalFallsBackToTheInput` asserted the old contract by name and was
  rewritten to the new one rather than weakened; three assertions that hard-coded
  a fixture's raw spelling now compare against `canonical(...)`, which is what
  `args()` actually mounts.

### Documentation

- **What the v5 design asks for and the product does not have is written down.**
  A new `docs/ROADMAP.md` section names `refs/design/HANDOFF.md` as its source,
  so the specification that comes next knows where the request came from.

  Three items. A tool reports **nothing while it runs** — the protocol has
  `tool.requested` and `tool.completed` and nothing between, so the design's
  running counts have no origin; that is four layers and MINOR at minimum, and
  it blocks the card's progress bar and nothing else. The session rail **reads
  the disk**, which is a premise rather than a fact, and the day a client
  attaches to a daemon elsewhere the rail lists nothing — silence that reads as
  a bug unless it was written down first. And the full card border stays
  recorded as a visual preference with its price named, rather than as a change
  of mind waiting to happen.

  A fourth, not from the design: **a boundary test passes or fails according to
  what is on the machine.** `TestKeepingTheWorkspaceVisibleDoesNotMakeItWritable`
  depends on whether `/tmp/ws` exists, because `EvalSymlinks` only resolves paths
  that do. It hid for a day behind Go's test cache — the same trap as the cached
  coverage number, by another road.

### Client TUI

- **The v5 keyboard is decided by convention, and nothing is declared yet.**
  The design handoff proposes five keys and three of them collide: `^E` is
  line-end, `^N` is already "down" in the picker, and `^Z` is SIGTSTP in every
  terminal that exists.

  Decided instead: `^B` for the sidebar (what VS Code made the word mean), `^R`
  for the session rail (readline's reverse-i-search — a session **is** history,
  so borrowing the chord reinforces its meaning), and `r`/`F2` to rename while
  the rail owns the keyboard. The FILES section gets **no key at all**: no
  editor gives every sidebar section a global chord, so `^B` opens the column
  and both sections are simply there. That deletes a key instead of adding one.

  `^Z` is refused twice over — rebinding suspend is hostile, and `/undo` already
  exists deliberately, already reaching delegated work through the state the
  parent adopts.

  The section 7 table is **unchanged**. Each key joins it in the PR that
  implements it, with the test that holds it: a key declared in a spec that no
  code executes is the same defect the copy-mode changelog records against
  itself.

### Documentation

- **Design references live in `refs/design/`, and there is one of them.** The
  v2 handoff sat committed in `docs/design_handoff_dcode_tui/`, a byte-identical
  copy of it sat beside it as `docs/Interface TUI com Tea Bubble.zip`, and the
  new v3/v4/v5 handoff arrived untracked in `refs/design/`. Three copies of the
  same material, two of them versioned, none of them saying which one governs.

  All of it is now in `refs/design/`, tracked, with the zip removed and the two
  prose references that named the old path updated so no link is dead. The
  README there is an index: what each file is, which one governs, and that the
  spec wins wherever a handoff and the spec disagree.

  The handoff itself is **verbatim, as delivered**. Five divergences found by
  checking its claims against the code are recorded beside it rather than edited
  into it — chief among them that tool progress does not exist in the protocol
  at all (`tool.requested` → `tool.completed`, nothing between), which makes
  "task as a card with progress" a four-layer change rather than a TUI one.
  Rewriting the handoff would erase the difference between what was designed and
  what was found out afterwards.

## 0.1.0 — 20 August 2026

The release the install path actually needed. 0.0.1 was published and then found
to be uninstallable by its own documented command, and everything below follows
from pulling that thread — including the discovery that the premise was wrong
twice before it was right.

The short version: **nothing has to be installed first**, and the digest that
says what the download should be now travels by a route the download cannot
touch.

> **Upgrading from 0.0.1 needs the install script, not `dcode update`.**
>
> ```
> curl -fsSL https://raw.githubusercontent.com/aguinelo/dcode/main/install.sh | sh
> ```
>
> A 0.0.1 binary carries the code from before the fix, so it still demands cosign
> and stops:
>
> ```
> Updating dcode 0.0.1 → v0.1.0
> dcode: cannot verify the release signature: cosign is not installed…
> ```
>
> The repair travels **inside** 0.1.0, which is the release the broken code would
> have to fetch — a bootstrap problem with no fix available from that side. Found
> by running the upgrade rather than assuming it, and written here because a
> migration note that is not recorded the day it is discovered is one nobody
> writes at all. From 0.1.0 onward `dcode update` takes the new path.

### Distribution

- **The release stops publishing to a tap that does not exist.**
  `aguinelo/homebrew-dcode` was never created, and `TAP_TOKEN` was never set, so
  the step exited zero and warned on every release — correct by design, since
  failing there would redden a release that had already succeeded, and the effect
  was that v0.0.1 reported success with a channel that never existed.

  Machinery that runs and delivers nothing is worse than absence: it occupies the
  place of a decision nobody has taken and makes the release look complete.
  `scripts/publish-tap.sh`, its tests and the workflow step are gone.

  The formula stays — generated from the signed checksums, attached to the
  release, and the artefact a tap would consume. What went with the step is the
  documented command, because `brew install aguinelo/tap/dcode` pointed at a tap
  that does not exist and, had it existed, under the wrong name: the script
  pushed to `homebrew-dcode`, whose brew shorthand is `aguinelo/dcode/dcode`. The
  two never agreed, and nothing held them together.

  Recorded in `docs/ROADMAP.md` with what creating it would take, and the naming
  trap to avoid.

- **`dcode update` does not require cosign either.** It was the last place that
  demanded a package — of a machine that already has a working dcode, which
  makes the demand harder to justify, not easier.

  The binary now reads the digests from the installer on `main`, the same file
  the install script carries in itself, so it has the same second route: a
  digest that did not travel with the artifact. The rule is the installer's
  rule — the carried digest **or** the signature, either is enough.

  One difference, deliberate: where the install script warns, `update`
  **refuses**. There is a working binary on the machine, so stopping costs a
  version and keeps everything. A signature that *fails* still aborts whatever
  the carried digest said; making a check optional must not make it decorative.

  `ErrNoVerifier` stopped being a verdict and became what it always was — one
  route unavailable. Its message no longer says "dcode will not install
  something it could not check", because that sentence was the requirement.

  `DCODE_UPDATE_INSTALLER_URL` overrides where the second route is read from,
  and a mirror that overrides `DCODE_UPDATE_URL` must override this too or its
  independent digest still comes from upstream. The wiring is asserted by
  reading the command as data: a field the updater reads and no command sets is
  the build-stamp defect again.

- **The installer never asks for another package.** Nobody installs an extra
  tool in order to install a binary, and the survey behind #223 already carried
  the proof — of rustup, bun, deno, nvm, k3s and uv, **not one** requires an
  external verification tool, and four verify nothing at all. I had the data and
  did not draw the whole conclusion.

  So cosign stops being something the installer talks about. What needs two
  routes is a **substituted release**, and two independent things cover it: the
  carried digest and the signature. Either is enough, so a covered install says
  nothing — reporting an unchecked signature while the check that matters passed
  by a route that does not depend on it is noise dressed as diligence. When
  neither covered it, the notice points at the installer that carries this
  release's digests, never at a package to install: answering a problem with
  "install something else first" hands over a second problem.

  This does not loosen "never unverified in silence" — it is why the rule could
  shrink. An install whose carried digest matched **is** verified. The SHA-256
  always runs, cosign is still used when it happens to be on PATH, and a
  signature that fails still aborts.

  Three tests asserting the previous rule were replaced rather than weakened,
  and one lost an assertion; the family changelog names each and why.

- **The notice is as large as what went unchecked.** Asked whether the cosign
  warning could go away. It cannot — "never unverified in silence" is the line
  four changes were spent establishing — but it was oversized, and about to
  become false.

  It claimed the checksum *"catches a corrupted download but not a substituted
  release"*. True with no pin. The moment a release pins the installer, the
  carried digest covers substitution exactly, which is the entire reason it
  exists. So with a digest carried and matched, the notice drops to two lines,
  loses that claim, and is no longer repeated at the end; with no pin it keeps
  every word and the repetition, because a `curl | sh` scroll buries whatever
  appeared at the top.

  A warning that overstates is one people learn to skip, including on the run
  where it finally means something.

### Documentation

- **The README describes the install that ships.** It still claimed the script
  *"verifies the release signature and the checksum, and installs nothing if
  either fails"* — untrue since cosign became optional, and silent about the
  digest the installer now carries. It now says what each of the three sources
  covers and what each needs, and why the carried digest is the one that catches
  a substituted release.

  Homebrew was going to be added at the same time, since every release publishes
  a formula. Checking first: `aguinelo/homebrew-dcode` **does not exist** — 404.
  `publish-tap.sh` exits zero and warns when it cannot reach the tap, by design,
  so v0.0.1 reported success with that channel never having been created. The
  README documents the three channels that work, and the tap is a decision to
  take rather than a line to write.

### Distribution

- **The release pins the installer it publishes.** The pipeline now verifies the
  signature it just produced, *then* fills the `PINNED` block from that
  checksums file, *then* publishes — with the pinned installer among the
  artifacts — *then* carries it to `main`.

  Each ordering carries weight. Pinning before verifying would take digests
  nobody vouched for, which is the failure the feature exists to prevent,
  reproduced inside the pipeline that implements it. Publishing before pinning
  would attach the unpinned installer. Writing to `main` before publishing would
  let a failure there redden a release that already succeeded — so that step,
  like the tap, exits zero and warns loudly on every recoverable condition, with
  one exception: a missing pinned file stops, because leaving `main` on the
  previous release's digests would make every install fall back **silently**,
  which is the correct behaviour for an unpinned installer and therefore
  invisible.

  `main` matters and the asset alone does not: the URL the README publishes is
  `main/install.sh`, and a pin that never reaches it never reaches anyone.

- **`scripts/version.sh` ignores the pipeline's own pin commit.** That commit
  lands after the tag, since the digests do not exist until the artifacts are
  built. Counting it would make every post-release query answer "there are
  commits since the tag" with nothing human changed, and the derivation would
  start raising PATCH on its own — automation leaving a trace another mechanism
  reads as a signal, which is a shape this repository keeps finding.

  The exemption is for the exact subject, never the prefix: exempting
  `chore(release):` wholesale would give anyone a way not to be counted. Tested
  both ways. `scripts/version.sh` had no test at all until now.

- **The installer verifies against a digest it carries.** `install.sh` gained a
  `PINNED` block that `scripts/installer.sh` fills from the *already signed*
  `checksums.txt`. When the downloaded artifact has a pinned digest, that is
  what it is checked against — and a mismatch aborts even when the release's own
  `checksums.txt` agrees with the download.

  This is the structural half of the entry below. `checksums.txt` travels from
  the same host as the tarball, so on its own it catches a corrupted download
  and not a substituted release: whoever can replace one can replace the other,
  and the pair stays self-consistent. Making the signature optional was right,
  but optional must not mean decorative — and what stops that is the expected
  value living in **git history**, where a release asset can be swapped with no
  public trace and a line in a tracked file cannot.

  The block starts empty, and empty is silent: an unpinned installer falls back
  to `checksums.txt` without complaining, because warning about a pin nobody
  applied trains people to ignore the line that matters. Pinned to one release
  and asked for another, it says which it carries and names the installer that
  carries the right ones.

  Taken from `uv`, the only one of six installers examined (rustup, bun, deno,
  nvm, k3s, uv) stricter than this one — four of them verify nothing at all, and
  **none requires an external verification tool.** uv's separation is better
  than ours: its installer comes from one host and its artifacts from another.
  Without a domain of our own, git history against release asset is the best
  available, and saying so is part of having it.

  The pipeline that fills the block is the next change; this one is the
  mechanism, with the generator and six tests.

- **A missing cosign no longer cancels the checksum.** `install.sh` required
  cosign and put that requirement immediately before the signature check, with
  the SHA-256 comparison after it. On a machine without cosign — every ordinary
  one — the install aborted having verified nothing: no binary **and** no check,
  the worst outcome available, produced by the rule written to prevent it. Found
  by the first person to run the documented command.

  The two checks are independent and are now treated as such: the checksum
  always runs, the signature runs when cosign is here, and its absence is said
  out loud at the point of skipping and again on the last line. The line worth
  holding was never "verified or nothing" — it is **never unverified in
  silence**. Signature present and failing still aborts, and that is asserted
  next to the degradation so the two cannot drift apart.

  It also stops downloading the `.sig` and `.pem` it cannot read, which was one
  more way to fail for a reason that is not the user's.

  Every existing test stubbed cosign into existence, so the one configuration
  every user is in was the one never exercised. Four reproducing tests now cover
  it, all four red on the reported symptom first.

## 0.0.1 — 20 August 2026

The first tagged release. It does not open a stable surface: `0.x` says the
shape is still moving, and this one is the point from which changes start being
counted rather than the point at which they stop.

The entries below are the work of the days leading to it. Everything before that
lives in the per-family changelogs, where it was written when the decision was
taken.


### Measurement instrument

- **Measured against the fixed harness.** Three contracts, fifty runs each,
  ninety-two minutes:

  | contract | before | after |
  |---|---|---|
  | `keeps-writing-that-must-cohere` | 96.0% | **100.0%** |
  | `names-the-child-that-did-not-answer` | 98.0% | **100.0%** |
  | `delegates-writing-when-disjoint` | 50.0% | **52.0%** |

  **The author's prediction was wrong.** #216 was written claiming the harness's
  refusal was talking the model out of delegating, and that fixing it would raise
  the third number. At n=50 the spread is about seven points: **two points is
  noise.** Why the work goes undivided is open again.

  The fix earned its keep in the other two: the model now has a delegation option
  that **works** and still declines to split work that has to cohere. Before, it
  declined in a world where delegating was impossible, which measured far less.
- **The eval harness runs a delegated turn** (#216). The old refusal was honest
  but said "do the reading yourself", which instructs abandonment. Still the
  right behaviour to fix; what did not hold up was the prediction about its
  effect.
- **Three contracts for divided work** (#214, #215). The third's threshold fell
  from 80% to 25% after four measurements spread twenty-five points — a floor
  against regression, not a certificate of quality.
- **The release reaches a mirror that answers** (#218). The release pipeline was
  a copy of CI that stopped being updated: no deadline on `apt`, no
  `apparmor_restrict_unprivileged_userns`, no probe — so **every boundary test
  skipped in silence** in the pipeline that decides whether to publish.

### Coordinating machines

- **A command that leaves the machine asks** (#212). `ssh`, `scp`, `rsync` to a
  host, `kubectl exec`, `ansible`, `aws ssm`, `docker -H`. `git push` does not.
- **An outside resource can be granted by name** (#211). `DCODE_SANDBOX_SOCKETS`
  and `DCODE_SANDBOX_WRITABLE`; the literal `ssh-agent` stands for
  `$SSH_AUTH_SOCK`.
- **A credential store can be put out of reach** (#210). `DCODE_SANDBOX_UNREADABLE`,
  with a default that hides without being asked.

### Sandbox

- **A socket is reachable where writing already is** (#199). Fixes the regression
  from #196, which closed port binding and took half the suite down with it.
- **A granted network is not a privileged socket** (#196). dcode found its own
  escape: it ran `docker run` from inside `workspace-write`, and it worked.
- **A nested sandbox is detected, not guessed** (#189).
- **A toolchain can reach its own cache** (#188).

### Delegation that writes

- **A refused write says it was a write** (#206).
- **A child says what it wrote** (#205). `Wrote` in the report, and the parent's
  turn undo reaches what the child did.
- **A delegated child writes only what it owns** (#204). `owns` is a request that
  can only narrow, and containment answers for it.
- **Research and planning** (#201, #202).

### Loop and configuration

- **The backstop matches the model's horizon** (#195). A ceiling of 200 became
  2,000 — the citation justifying it spoke of 1,959 calls.
- **The project instructions describe this project** (#194). 76% of the prompt
  described a Node project; it fell from 16,904 to 8,757 bytes.
- **The tool describes what it can do** (#207, #208). The description denied the
  writing its schema offered, and the model would not delegate because of it.

### CI and coverage

- **CI names a mirror that answers** (#203). The `apt` step went from six minutes
  of timeouts to thirteen seconds.
- **Coverage relaxes to the floor the specs ask for** (#192). The aggregate goes
  to 90%, and the per-package floor starts failing rather than only printing.
- **The coverage gate reads the whole matrix** (#190).
