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

## Current state — 1 September 2026

**What it is.** An agentic coding harness in Go: a daemon, a terminal client and
the agent loop between them, as a single static binary, with no cgo outside the
isolated package.

**Where it stands.**

| | |
|---|---|
| spec families | 18, with 162 decision changelogs |
| behavioural contracts | 58 declared |
| contracts needing a model | 53 of the 58; 5 are settled by assertion |
| **contracts ever actually measured** | **19** |
| coverage | 93.3%, gate at 90% aggregate **and per package** |
| CI | macOS + Linux matrix, gated on the **union** of the profiles |
| published version | **0.18.0** |

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

**The floor.** A short list of things dcode does when nobody asked, and a rule
saying who may change it. The user's prompt outranks the project file, which
outranks the built-in default, and whoever is above **replaces** rather than
negotiates — no confirmation, no weighing, no notice that this contradicts good
practice. Overriding is obeyed *and* stated once, and stating is not asking.

None of it needed a precedence resolver: the prefix is assembled in order and
the project's instructions are the last block, so a default rendered before them
is outranked by anything anyone actually said. What can be a **fact** is a fact
rather than prose — that there is no repository, and which checks the project
declares — because prose is the weakest layer this repository recognises, and a
rule that needs a lookup first is a rule followed by accident.

**Skills.** Guidance that only matters sometimes: a `SKILL.md` in a folder or a
`<name>.md`, under `.dcode/skills/` here or `skills/` in the user's directory,
with `name` and `description` at the top. That is the shape other agents use, so
one found anywhere is usually a file to copy in unchanged — measured, not hoped:
a real third-party skill was downloaded, dropped in, and applied.

Only the index line is paid for every turn; the body arrives when the trigger
fires, and the load is announced, because a block of text that joins the turn and
changes what the model does used to leave no trace anywhere a person looks. The
trigger needs two word hits **and** one on a word no other installed skill
carries — sharing "projeto" and "versão" told two skills apart not at all, and
the stop list holds both languages this product is written in.

Nothing a skill file gets wrong stops the product: an over-long index line is
trimmed and said, an unreadable one is skipped and said. The one shape that is
neither is a skill reaching for the boundary — approvals disabled, sandbox
bypassed — which is **held and put to the person**. Granting it loads it whole,
and every outcome leaves a line, because consent that leaves no trace reads like
no question was asked.

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
question with no other answer on the screen is where the letters you type go. A
line starting with `!` is not sent to the model — it runs, through the same tool
and the same boundary, and the field says so from the first character.
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

**Which model.** Transport × family: the wire format is reusable, and the
measured thresholds belong to the model. Four families — MiniMax-M3, Claude,
Gemini and the explicit `generic` escape hatch — over two dialects.

A family with no measurement behind it **says so in the session**, and the list
of which families warn is checked against the measurements that exist rather
than typed. That guard found `claude` had been in that condition since it was
written.

**What this document does not say.** That the system is verified. Of the
fifty-three contracts that need a model, **thirty-four have never run against
one**, and the suite prints the split on every run to stop the opposite reading.

Of the nineteen that have, **five did not meet their threshold**, and the
thresholds did not move to meet them. The worst reads 5%: an instruction in the
project file overriding the built-in floor, which the family that owns it calls
its strongest rule.

The two numbers above are counted, not carried. The row used to read "4", from
the release before this one, and it stayed 4 while `boundary-decides-write` was
measured — a table describing a state that had moved, in the same document that
exists to stop exactly that.

---

## 0.18.0 — 1 September 2026

- **The end of a loop run is said, with where the criteria stand.** It used to
  be silence — the function that pulls the next spec returned nothing when the
  queue emptied. From outside, a loop that worked four specs and stopped is
  indistinguishable from one that stalled on the fourth: "it is over" reads
  exactly like "it is thinking". The notice says how much was worked, how many
  criteria are met of how many, and **names** what is unmet and what could not be
  checked — a count answers *how much*, a name answers *what to do next*, and the
  end of the run is when that is the question.
- The state comes from the turn-completed event, not re-read from the drawn
  entries: the completion travels on the wire precisely because it is the
  guarantee that survives a model claiming success in prose, and re-deriving it
  from that prose would put the prose back in the path of the fact.
- **A run that worked nothing says nothing.** The queue empties on every
  proposal commit, including the ones where there was never a queue, and a line
  announcing the end of a run that never ran is a line about the feature rather
  than the session.
- **The bottom bar carries the working directory, at its right-hand end.** The
  worktree segment always had the base name, which is the fast answer until two
  checkouts share one — `dcode` under two parents reads identically, and the
  session that ran in the wrong one looks like the session that ran in the right
  one. It elides from the **front**: the tail distinguishes two worktrees, the
  head is what every path on the machine shares.
- **It does not vanish as the terminal gets wider.** The first version took only
  leftover space, so at eighty cells the segments fit and left nine and the path
  was dropped — while at sixty, where the hints had already gone, it was drawn.
  It disappeared at the width most terminals are and came back as the window
  shrank. It now outranks the key hints, which `?` restates in full, and yields
  to the diff, the position and the mode, each the only place its fact appears.
  The test sweeps 30 to 200 cells, because the defect was non-monotonic and an
  assertion at one width would have passed at both ends.
- **The window title is the session's name**, its derived title when nobody
  named it, and where it is running when there is neither. A row of terminal
  tabs all called `dcode` answers nothing.
- **The proposal was written before the turn had run.** A qualifying session
  opened with the right brief and answered *"nothing was proposed for
  1a05dd01b2…"* — before the model had finished thinking, which is the tell. The
  trigger read "the session is idle and it is qualifying", and a session is idle
  **before its first turn starts**, with `attach` replaying from the beginning.
  It is now the turn-completed event.
- **The queue drain beside it still reads the state, and is right to.** It wants
  *any* moment nothing is running; the proposal wants *one* moment. The two lines
  look like the same condition and are different questions — which is how one
  copied the other's shape and was wrong.
- Three defects in a row, and only the first was visible from outside: the
  sentence had to become a qualified goal, then the client had to stop dying on
  the session switch, before this one could show at all.
- **Switching sessions could quit the client, and `/loop oi` was how it showed.**
  The event reader captures its channels when the command is *built*, so the
  reader watching the old session is still selecting on the old channels when
  `attach` cancels them — and a closed channel is how a stream reports that it
  ended. Untagged, that reached the case that quits.
- **This was never about `/loop`.** `/clear`, `/model` and `/resume` attach the
  same way and would have ended the same. The previous fix did not introduce the
  defect; it made the defect **reachable**, because `/loop <word>` used to fail
  before it got as far as switching sessions. The two readings have different
  repairs, and taking the first would have meant reverting the right fix.
- **Each subscription is numbered**, and the three messages it produces carry
  the number. The check happens in the update loop, which is single-threaded:
  reading the current generation from inside the command would be a data race
  with the loop that writes it. A replaced stream's message is dropped and does
  **not** re-arm the reader — whoever attached already started the reader for
  the stream that replaced it. The current stream ending still quits: the rule
  is about which stream, not about never quitting.
- **The status bar says which build is running.** A local build and a released
  one behave differently and presented identically, so the only way to tell was
  to leave the session and ask `--version`. The version string already said
  `-dev+sha` for a local build; it just was not on screen. It sits beside the
  name, dimmed, and is the **first** field given up as the terminal narrows —
  before the model, and the sandbox mode still never.
- **`/loop oi` still failed, and RN-8 never reached it.** `specArgument` reads a
  single word as a path, always — so the rule that a goal with no folder gets
  qualified applied from two words up. One word opened a session against a path
  that was never there, and the daemon answered with a raw read error carrying
  the absolute path twice. A bare word that names no folder the survey found is
  now a goal. The separator decides: someone who wrote `specs/hoem` meant a path,
  and a typo answered by qualifying it is a typo hidden, so that stays an error.
- **A goal with no spec folder is qualified, not refused.** `/loop revise o
  projeto até entender` answered *"no specs/ folder here, or nothing in it.
  /loop `<path>` works on one folder"* — the command telling someone their
  request was the wrong shape, in a product with a whole family for exactly that
  shape. The done-qualifier's own research spec names the prose request as what
  motivated it, and `/loop <path>` already qualified a folder that declared
  nothing; only the sentence route died first, because it was built to **select**
  among folders rather than to accept a brief.
- **The sentence is the brief, so the sentence is what the turn names**, and the
  turn no longer sends the model to read a specification that does not exist —
  looking for one spends the rounds it has. The proposal is anchored at `.dcode`,
  where a workspace's definition of done already lives, and **not** in a folder
  derived from the sentence: RN-5 records the opposite defect, prose becoming a
  path, and inventing `revise-o-projeto/` would be that defect wearing the other
  hat.
- **`.dcode` is the only directory the write creates**, deliberately. A wider fix
  broke `TestACommitThatCannotWriteSaysWhere`, which has guarded since before
  goals could be qualified that a spec path which does not exist stays an error —
  a mistyped folder answered by creating it is the silence this family refuses.
- **A test that would have been theatre, caught.** The first one exercised the
  helper directly and passed with the handler wiring removed. The one that
  counts goes through `Update`, where the refusal was drawn, and was seen red
  without the wiring.
- **A family for Gemini**, over the `openai` transport, through Google's
  compatibility surface. A family and not a transport: the dialect already
  exists, so `Gemini` embeds `MiniMaxM3` for the encoding and overrides exactly
  what the family axis is for — name, model prefixes, window, limits, images.
  The native surface is a transport, and writing one before anyone has run this
  against a real key would be building the harder half first on a guess.
- **The numbers are chosen, not copied.** The window is 1,000,000 against a
  documented 1,048,576, because under-guessing costs a summary and over-guessing
  loses the turn. The ceiling is 50 and explicitly not MiniMax's 2000, which is
  justified by a cited long-horizon run that says nothing about this model — a
  test asserts the two differ. `Encode` refuses the Anthropic dialect it would
  have inherited, for a family whose `Transports()` names one.
- **RN-11: a family with no measurements says so.** A family name reads as a
  measured family here, because `Measurement.Model` exists precisely so a
  threshold belongs to one model and says nothing about another. The warning
  names the family and says what the thresholds *were* measured against, and the
  list of who warns is checked against the measurements that exist, in both
  directions — nothing typed.
- **The guard failed on its first run, on `claude`.** That family has existed
  since the beginning, to prove the axes are orthogonal, and has never carried a
  single measurement — it had been running without saying so. Not what I went
  looking for; what the guard found by existing.
- Pointing at Gemini is `model.base_url` =
  `https://generativelanguage.googleapis.com/v1beta/openai`. `defaultBaseURL`
  answers per **transport**, not per family, which is why it does not name
  Gemini: a transport deciding something from a family is the axes collapsing,
  which the interface's own documentation forbids.
- **The skills block now says what the format is, not only where it lives.** The
  previous change half-worked: the agent went and looked at the directory, then
  still concluded that a skill found on GitHub "loads in Claude Code, not in my
  agent". It got the hard part right — that the URL pointed at a skill inside a
  **plugin**, that plugins and marketplaces are another product's packaging, and
  that installing one touches a global setup and needs confirmation. It missed
  the easy part: `curl` the `SKILL.md` and write it to `.dcode/skills/`, two
  lines, inside the workspace, crossing nothing.
- **It is a fact rather than a promise, which is why it can be said.** The block
  now names the shape — `SKILL.md` in a folder or `<name>.md`, with `name` and
  `description` on top — and says one found anywhere is usually a file to copy in
  unchanged. `description` has been an alias for `when_to_use` since before this
  family existed, and a real third-party skill in exactly that format was loaded
  and applied in a field test this afternoon.
- **What stays out is deliberate**: plugins, marketplaces and install commands
  are packaging, not format, and the agent already reasons about them correctly
  on its own. So does the matching divergence — the model decides from the
  description there, deterministic word matching here — which is a design choice
  that belongs to whoever writes a skill, and lives in the `.r`. The section is
  409 bytes against a 520-byte cap in the test.
- **The agent did not know about its own skills mechanism.** Asked to install a
  skill, it answered that it could not — that skills are a Claude Code thing and
  not installable from here. Every sentence of that is false about the product
  it is: dcode loads skills from `<workspace>/.dcode/skills/`, and writing there
  is a write **inside** the workspace, which crosses nothing and asks nobody. It
  had the tool and the permission; it lacked the information.
- **The skills block now renders even with none installed**, and says where they
  live and what writing one does. It used to render only when one existed, so a
  workspace with no skills told the model nothing about the mechanism at all —
  and with nothing written, the model answered from training, which is about
  another product. Two lines, never a manual: RN-7's economics are why the
  bodies are not in the prefix either, and what these two lines buy is the
  alternative not being the product misinforming the person about itself.
- **A guard was reading an empty string.** `TestAbsentSectionsEmitNoHeading`
  claimed `## Skills` was omitted when empty, and passed by looping over
  nothing: its `Prompt` had no `Safety`, so `Build` failed and the loop searched
  an empty output for four headings. Fixed in the same change, because it is the
  behaviour this change alters.
- **The README's boundary paragraph said the opposite of what the boundary
  does.** It claimed that "anything crossing that boundary — a write outside it,
  or the network — stops and asks", and neither half is true under the defaults:
  `sandbox.allow_network` is `true`, and `/tmp`, `/private/tmp`,
  `/private/var/tmp`, `/dev` and the toolchain caches are writable. Both grants
  are deliberate and the reasons are written in the code; what was missing was
  the front page saying so, in the one paragraph a reader trusts to decide
  whether to leave this running unattended.
- **The read asymmetry is written down.** `read /tmp/x` declares that path,
  which is outside the workspace, and asks. `bash cat /tmp/x` declares only the
  workspace and the network, and does not. Containment is identical — the OS
  permits the read either way — so only the question differs, and the tool that
  asks is the one being honest about what it touches. Found by watching a real
  session fetch a file from the internet, write it to `/tmp` unasked, and then
  stop to ask whether it could read what it had just written.
- **A skill that reaches for the boundary is held and put to the person, never
  loaded blind.**
  `SafetyClaims` has run over instructions since RN-10 asked for the attempt to
  be recorded. Nothing ran over a skill — and a skill is the least trusted text
  this product loads: it arrives by `git clone` into `.dcode/skills/`, or is
  downloaded from a stranger's repository, which is what happened in this
  afternoon's field test, and its body goes straight into the turn inside a
  `<skill>` block with nobody reading it first.
- **This one asks where RN-10 only reports, and the difference is provenance.**
  An instruction is the user's; dropping a file over one sentence would cost them
  a rule they wrote, so there a false positive costs a line of output and
  reporting is enough. For a skill the asymmetry flips: a false positive costs
  **one question**, answered with the matched text quoted — a false negative
  loads third-party text into the model's context with no question at all.
- **Asked, not refused.** Refusing outright would be the product deciding what is
  the person's to decide; boundary and authorization are separate axes (ADR-02),
  and this is the second. Approved, the skill loads whole — holding is a question,
  not a deletion. Denied, it does not. **With nobody to ask, it does not load** —
  the rule the loop already applies to every crossing, for the reason it already
  gives: with nobody to ask, the only alternative to refusing is granting in
  silence. All three outcomes leave a line in the audit, the granted one
  included, because consent that leaves no trace is indistinguishable from no
  question having been asked.
- **Both halves are screened, and the filter has to stay narrow.** The body is
  where a payload would sit; the index line is paid on every turn, so a harmless
  body under an offending line is the cheapest version of the attack. Measured
  against `web-design-engineer` — 35,012 bytes of real third-party guidance —
  zero matches. One sample, and said to be one sample.
- **The skill waits, the product does not stop.** Killing the process would hand
  any cloned repository the power to stop dcode running, which is the defect
  fixed in the entry above; recreating it in the name of safety would be trading
  a problem for itself.
- **A bad skill file no longer stops the product.** Found by field test, not by
  reading code: a real skill from the ecosystem this format came from —
  `ConardLi/garden-skills/skills/web-design-engineer`, with 455 characters of
  `description` where the cap is 120 — made `LoadSkills` return an error, which
  `app.go` propagated, which made dcode exit 1 in that workspace,
  `--dump-prompt` included. `.dcode/skills/` arrives by `git clone`, so one file
  in a cloned repository decided whether the binary ran at all.
- **The caps stand; being fatal did not.** An over-long index line is trimmed at
  a word boundary and the cut is reported; a file that cannot be a skill, or one
  over the byte cap, is skipped and reported. The body is never cut — guidance
  that stops mid-sentence is worse than guidance that is absent and said to be
  absent. Only an unreadable directory is still an error, because that is the
  machine failing rather than a file being wrong.
- The notices appear in `--dump-prompt` in a block of their own, separate from
  doctrine notices because they answer different questions.
- **`skill-loaded-on-trigger` measured: 100% of 20 runs**, threshold 85%,
  MiniMax-M3. It was one of the contracts that had never run. The judge looks
  for the step nobody would guess — the skill says to record the version in
  `RELEASING.md` before cutting the tag, and a model that never received the
  body has no way to know that file exists. Twenty runs, twenty hits: the
  mechanism works.
- **The number says nothing about the ceiling, and the note says so.** `Rounds`
  went from 12 to `exploreThenActRounds` *before* the run, by the definition
  written on that constant and not by evidence from this scenario — and then no
  run failed, so there is no failure to attribute to either number. A corrected
  ceiling followed by 100% reads as cause and effect and is not. This repository
  has already misread five numbers by confusing instrument with behaviour;
  reading a correct number for the wrong reason is the same mistake with better
  luck.
- **A loaded skill announces itself.** A skill body was appended to the turn as
  a reminder with nothing emitted: spent context, changed behaviour, and no
  trace anywhere the person looks — `grep -i skill internal/tui/` returned
  nothing outside tests. The index was always auditable, in the prefix and in
  `--dump-prompt`; what fired was not. `skill.loaded` carries the name and the
  same when-to-use line the model read in the index, so both are looking at the
  same sentence, and the stream draws it as a note.
- **Not the path, and not every turn.** The event log is read by another client
  on another machine, where an absolute path from the machine that wrote it is
  not a fact; which root a skill came from is a question `--dump-prompt` and the
  filesystem answer. A turn that loads nothing announces nothing, and an
  announcement with no name draws nothing — a row whose only content is that the
  feature exists is a row spent.
- **`docs/ROADMAP.md` §16 records the two things this deliberately did not do**:
  a `/skills` listing, and skills shipped inside the binary. The second is
  argued against by the product's own RN-7 — every bundled skill is a line paid
  on every turn of every session — with the three unpaid costs named.
- **A skill loads on what distinguishes it, not on what it has in common.** The
  stop list held English only, in a product whose `LANGUAGE.md` declares two
  languages and whose user writes prompts in Portuguese: `quando`, `projeto` and
  `estiver` counted as significant words while `when` and `that` did not, so the
  same sentence was filtered in one language and pulled whole skill bodies in the
  other. "quando o projeto estiver pronto me avisa" loaded two skills, neither of
  which was about anything in that sentence.
- **The Portuguese list alone did not fix it.** `projeto` and `versão` are
  content words, they appear in both skills' when-to-use lines, and two hits are
  still two hits. The defect is not that a word is common in the language — it is
  that it is common *among the skills in the index*, and a word both of them say
  tells neither apart. `Match` now needs two hits **and** at least one on a word
  no other skill in the index carries. A single installed skill discriminates by
  everything it says, which is the right answer: with no neighbour there is
  nothing to be confused with.
- **Neighbours in one domain stay reachable.** `release-go` and `release-node`
  both say cut, version and new, and each still has `golang` and `typescript` —
  which a blunter rule that simply discarded shared words would have broken.
- **The front page says what was measured, and a guard counts it.** The README
  still claimed there was no TUI and no released binary, four months and
  seventeen minor versions after both stopped being true; its badge said ten
  specs against eighteen families, and its testing section claimed a 95%
  coverage gate against a script that has always read 90. It now opens with the
  ratio it was hiding — 58 contracts declared, 18 measured — and carries the
  receipts table, `floor-yields-to-project` at 5% included. The comparison with
  the four agents that got here first moved to the bottom, where credit belongs,
  and the verification cycle moved up, because it is the part nobody else has.
- **`TestTheReadmeBadgesAreCountedAndNotCarried` and
  `TestEveryReceiptNamesAMeasurement`.** The badges, the sentence under them,
  the specs table and every rate in the receipts table are read from the tree
  and from `Measured`, in both editions — the treatment the changelog's state
  table already had, applied to the document more people actually read. The
  guard failed on its first run, on a number this same change had typed: 40
  contracts never measured, where the tree gives 35. Coverage is checked only
  for agreement between README and changelog, because a test that does not run
  the gate cannot honestly claim more, and that is said in the test rather than
  left to be discovered.
- **The changelog's own prose was two numbers stale.** It said six measured
  contracts missed their threshold and that the worst read 30%. Five miss, and
  the worst reads 5%.

## 0.17.0 — 31 August 2026

- **A criterion that prints nothing now names its command.** Found by running
  the installed binary against a real workspace, not by reading code: a
  `done.toml` with `test -f CHANGELOG.md` fails silently, so the output block
  did not render and the reminder was back to the name and nothing else. The
  model went and read `done.toml` to find out what the criterion was — two
  rounds after something the loop had in hand. The command is identity and
  never evidence: it stands in only when there is nothing to show.

## 0.16.0 — 30 August 2026

- **`recoverable-cycle`, a new family: the loop is closed on detection and open
  on recovery.** It knows a cycle made things worse and cannot go back —
  `Progressed` returns a boolean where three answers belong, so drawing,
  regressing and swapping one failure for another all collapse into a stall
  count. `.r` only. The objection that kept this out of scope turned out to be
  false: **a point of return does not have to be a commit**, so the boundary
  that git is the user's stays intact. Undo is the loop's decision and never the
  model's — an agent that can revert its own work can revert the evidence.

- **A cycle that broke something is put back.** The loop now classifies what
  a cycle did in three answers instead of two, and rolls back the ones that
  regressed — a criterion that passed and stopped. The snapshot machinery was
  already there; nothing told the loop a cycle had made things worse, and the
  scope was the whole turn, so undoing after one bad cycle would have thrown
  away every good one before it. Drawing is never rolled back: a cycle that
  read and closed nothing broke nothing. The model is told, and told to try
  something else — an agent that is not told repeats the edit believing it
  never happened.

- **The harness can run a verification cycle, and the first contract measures
  a correction.** Two families had shipped with nothing measured about them:
  every scenario injected the reminder the cycle would have produced, so
  `checkDone`, `Moved` and the rollback never ran. A scenario can now declare
  criteria — predicates over the workspace, never shell — and the judge re-runs
  the ruler instead of reading the transcript. `fixes-what-the-output-named`:
  **100% of 20**.
- **It read 65% first, and that was a criterion of mine.** It demanded one
  particular implementation and its error message read as its own opposite, so
  five of seven failures were runs stuck trying to satisfy it. Four times in two
  days a rate has said something interesting about the model and been about the
  instrument.

- **A measurement took two steps out of the plan.** The loop's roadmap had
  "progress by proximity" and "raise the stall ceiling" after the rollback, both
  to stop the loop giving up on work that advances without closing a criterion.
  `finishes-work-that-takes-more-than-one-cycle` measured **95% of 20**: the
  ceiling does not bite. `Moved`, shipped for another reason, had already fixed
  it — any forward movement resets the counter. The intuition was true when it
  was written and stopped being true; without measuring, both steps would have
  been built, would have worked, and nobody would have known they were
  unnecessary.
- **`verifiedCycleRounds`**, because a scenario that runs the cycle spends
  rounds the work never sees: the model has to stop calling tools for a cycle
  to run at all. The old ceiling was written when no scenario ran one.

## 0.15.0 — 29 August 2026

- **The failing criterion's output now reaches the model.** The reminder
  carries what the command printed, under the sentence that was already there,
  marked once as a result rather than an instruction. Measured before and
  after: the two contracts most at risk held at 100% of 50 and 100% of 20, and
  `states-unmet-on-stall` moved 92% → 94% — two points, the smallest difference
  50 runs can see. **The family did not justify itself by the number**; it
  stands on the structural argument, and that is written as such.
- **The round ceiling has now decided four measurements in two days.** At 12
  rounds the same pair read 82% → 72%, which was ready to be published as *the
  output makes honest reporting worse*. Thirteen of those fourteen failures
  were runs the harness cut mid-work. The ceiling does not rise further:
  raising it until a contract passes is fitting the instrument to the result.
- **The failing criterion's output is kept.** `Check` used to run the command
  and discard what it printed into a `_`; now it keeps the output of everything
  that did not pass, capped at 2000 bytes — the qualifier's ceiling, because it
  is the same information from the same runner. Cut from the END, since a
  runner's summary and its last assertion are at the bottom. It does not reach
  the model yet: that changes the prefix, and the measurement needs a "before".
- **`failure-feedback`, a new family: the loop detects well and returns badly.**
  When a criterion fails, `Check` runs it, discards what it printed into a `_`,
  and the reminder tells the model the criterion's NAME and nothing about what
  broke. The evidence is collected and thrown away on the same line — while the
  neighbouring phase, the qualifier, keeps it and writes it down. `.r` only: the
  problem, the rules, and the risk stated before anything is built.

## 0.14.0 — 28 August 2026

- **A check you cannot run does not cancel the work.** A fifth practice in the
  floor, and it entered by measurement, which is what the family's own RN-8
  demands. Three contracts across two families showed the same shape — the turn
  reads everything, reasons correctly, and ends without proposing or editing —
  always right after announcing a verification it could not perform. The
  doctrine already said what to SAY when you cannot verify; it never said the
  work is still owed.
- **Four measured rates were replaced, not accumulated.** They described
  scenarios that had changed underneath them: a round ceiling of 12 on turns
  that read a spec and a codebase before producing anything, and a shared eval
  workspace that did not compile. A rate belongs to a scenario, and one that
  outlives its scenario is the state table's defect wearing different clothes.
- **The shared eval workspace compiles again.** `internal/config/toml.go`
  called two helpers that did not exist. Models read that file in scenario
  after scenario, and the careful ones said so and spent their rounds there.
  `TestTheSharedWorkspaceCompiles` runs `go build` offline over the tree.
- **The ablation, because three changes at once attribute nothing.** Reverting
  one at a time over 20 runs each: without the practice 90%, with the ceiling
  back at 12 95%, with the workspace broken 95%, against 100% with all three
  and 75% with none. Joint and roughly additive, no dominant cause.

## 0.13.0 — 27 August 2026

- **Eight declared thresholds became measured ones, and five did not hold.**
  The qualifier's three contracts and the floor's five ran against a real
  model. `qualifier-proposes-commands` (96%), `floor-yields-to-user` (96%) and
  `floor-checks-before-claiming` (100%) met theirs; the other five did not, and
  no threshold moved to meet a result.
- **The floor's strongest rule measures 30%.** The same instruction, the same
  wording, the same task: said by the user in the turn it is obeyed 96% of 50
  runs; written in the project file it is obeyed 6 of 20. The family's design
  rests on the prefix being assembled in order, with the project's instructions
  last — position in the prefix is not precedence, it is hope of precedence.
- **A broken criterion is written down and not declared.** It used to be
  declared, and the file is what the next run loads: the work session was then
  measured against a command that does not exist — red forever — and the folder
  now declared a criterion, so it could never be sent back through
  qualification either. Two dead ends from one line.
- **Three contracts show the same failure, in scenarios that share nothing.**
  The turn reads everything, reasons correctly, and ends without calling the
  tool — always after saying it wants to verify something the turn cannot
  verify. The next target is the instruction, not the scenarios.
- **`qualifier-narrows-on-mismatch` was retired.** It described a second model
  turn reacting to the measurement, and the measurement now happens outside the
  turn: the person reads the disagreement. Retiring a contract is at least
  MINOR.

- **The loop can work out its own criteria, in plan mode.** A qualifying turn
  reads the spec and the code and calls `done_propose`; the harness measures
  each proposed criterion and writes a `done.toml` the person reviews. The
  **loop** decides there is a qualification — a model that chose when to
  qualify would be choosing when to be measured.
- **The tool touches nothing, and that is the design.** The first version
  declared a write and plan mode denied it, exactly as it should: read-only has
  no exception, and one would be the exception the next person widens. So the
  proposal is *recorded*, the turn ends, and the loop asks the daemon to
  measure and write it — under the boundary the work will run under, which is
  also the only place the criteria can actually run.
- **The loop chains the phases on its own.** `/loop specs/x` asks the daemon
  what the folder declares — a read, `measure=false`, running nothing — and
  opens a qualifying session when the answer is nothing. The turn ends, the
  loop commits the proposal, and then it **stops**: a proposal nobody looked at
  is a ruler nobody read. Over a backlog that is one pass, every folder
  qualified, and one sitting to review them before the work runs on its own.
- **`Expects` caught its first real one.** Against a prose-only spec the model
  proposed `bash reverse.sh; test $? -ne 0`, expecting it to fail. It passed —
  127 from a missing script is non-zero — so the criterion was green for the
  wrong reason, and the file says so above the criterion where a reviewer
  looks. No human scan of a command list would have caught that.

## 0.12.0 — 27 August 2026

- **`/loop <goal>` works the whole backlog.** `/loop implemente todas as specs
  pendentes` used to make `implemente` into a folder name and fail on
  `implemente/tasks.md` — prose became a path, the same defect as prose
  becoming a criterion pointing the other way. An argument with a separator, or
  a single word, is a path; a sentence is a goal. RN-7 and US-2, written on the
  25th and not built until now.
- **"Pending" is measured, not counted.** Discovery runs each folder's criteria
  through the same sandbox a turn uses, because a checkbox is ticked by whoever
  felt like ticking it. A folder that declares nothing is pending: absence of
  proof is not proof of done. The daemon decides, since it has the disk and the
  sandbox.
- **A spec with no tasks yet is pending, not unreadable.** Measured against a
  real backlog, 11 of 28 folders came back as errors and were dropped from the
  queue — every one of them a `spec.md` with no `tasks.md`, which is the most
  pending thing there is. Now 28 of 28, none unreadable.
- **A guard matched a loose string, for the third time today.** The
  English-leak test searched a Portuguese screen for words from the English
  catalogue by substring, and `works` matched inside `workspace-write` — a
  value, not layout. Whole words now.

## 0.11.1 — 27 August 2026

- **`/loop` does the work instead of preparing a place to ask for it.** It
  loaded the definition of done, switched session, and submitted **nothing** —
  so someone who typed `/loop specs/x` still had to say what they wanted. It
  submits now, naming the spec and saying the harness checks the criteria,
  without restating them.
- **A word after the path is what to do, not a mistyped flag.** `/loop <path>
  implementar …` was refused with *"implementar is not a flag here"*. Only
  something starting with `-` can be a mistyped flag; everything after the path
  is the task, as typed.

## 0.11.0 — 27 August 2026

- **A spec folder can declare its own `done.toml`.** Measured against the 17
  real Code Plain specs, `/loop` returned zero criteria in every one: their
  tasks carry no `verify:` marker and their acceptance criteria are sentences —
  *"Lighthouse ≥ 95"* — which no parser may turn into a command without
  inventing one. The folder now has somewhere to say it in commands. Same name
  and format as the workspace's file, one parser; it wins over `tasks.md`, an
  empty one is an error rather than a fall-through, and an absent one is
  ordinary.

## 0.10.0 — 27 August 2026

- **`/loop` is typeable.** `/loop <path> [--protect <glob>]` opens a session
  measured against that folder's `tasks.md`, and the command text never becomes
  turn input. The client sends the path and the daemon reads it, because a
  client may be nowhere near that disk.
- **A session says how many criteria it carries, and zero is an answer.** A
  session with no definition of done reports done at the end of the first turn,
  so `/loop` says so on the spot rather than at the end.
- **An error was classified by looking for the word "workspace" in its text**,
  and messages carry paths — so a missing spec came back `workspace_invalid`
  from a repository that lives under a directory of that name. A sentinel now,
  matched with `errors.Is`.
- **The state table is counted, not typed.** Families, decision changelogs and
  every contract number are read from the tree and checked against both
  editions and against the sentence beside the table. Both failures that
  actually happened are caught.
- **The number that says how little is verified was itself stale** — carried
  from the release before while a fifth contract had been measured. Counted and
  split now: 48 declared, 43 needing a model, 5 ever run.

## 0.9.1 — 27 August 2026

- **Two gates with the same name are told apart on screen.** Found by running
  0.9.0: a project with a `test` script in `package.json` and a `test` target in
  the `Makefile` printed two rows called `test`, with different commands, and
  nothing said which was which. `Source` was being carried and never shown. It
  is the same defect the task parser refuses in a `tasks.md` — two rows a reader
  cannot tell apart — arriving at the other end of the same release. Only the
  ambiguous ones are qualified; spending a column on a distinction that does not
  exist is noise on every other project.

## 0.9.0 — 27 August 2026

- **The qualifier can be signed, and signing is editing.** `Sign` runs the
  operator round trip: `SignedAnswer` carries the `DoneSet` **as they left it**,
  not a verdict on the one proposed, because a binary gate turns "I disagree
  with item 3" into "redo everything" and that cost lands on the operator until
  they stop disagreeing. Any criterion they **edited** is measured again before
  anything freezes — otherwise their own edit escapes the rule the package
  exists for. An edit that does not change the class settles at once; one that
  does, and any criterion they **added**, goes back once, because a class nobody
  has seen must not be signed. Refusal, an expired deadline, an exhausted round
  limit and **a failed channel** all end in `ErrRefused`, and none of them starts
  a loop: the client going away is not somebody having said yes.
- **A set that empties itself on the way out is refused.** Broken criteria are
  dropped from the frozen `DoneSet` because they cannot run — but dropping them
  can leave nothing, and nothing means "nothing to verify", which the loop
  reports as **done**. The proposal was not empty; it *became* empty, and that
  was the one door the rule did not cover. Found by a test written for something
  else that passed when it should have failed.

- **The qualifier can measure, and classify what it measured.**
  `internal/loop/qualifier` ships the deterministic half of `done-qualifier`:
  run every proposed criterion once against the repository as it stands, before
  any work, and classify. A criterion that **fails** is acceptance — it can
  testify the work happened; one that **passes** is a regression guard, whose
  job is to stay green. Both are legitimate for opposite reasons. Two details
  decide whether it works: passing is `Exit == ExitCode` and never `Exit == 0`,
  because a criterion declared `exit: 1` is met by exiting 1 and comparing to
  zero would call an already-green criterion acceptance; and 126/127 plus a
  failure to start are **broken**, not red, because a command that does not
  exist fails and by failing disguises itself as acceptance while measuring the
  absence of a tool. A set with nothing red is **named**, never refused — a
  genuine refactor has nothing new to prove, and the harness must not decide
  which is which. Nothing here derives a criterion, asks anyone anything, or is
  reachable from the product yet.

- **New `loop-command` family: the parser, not the command.** A third source
  for the RN-10 done definition — a `tasks.md`-shaped directory alongside
  `.dcode/done.toml` and the legacy verify command. `internal/loop/loopcommand/`
  parses one into a `loop.DoneSet` and builds the `loop.Config` a dedicated
  session would be born with. **`/loop` is not typeable yet**: recognising it in
  the client is Step 3 of the family's `.i` and has not been built, so nothing
  in the product calls this package. The turn cycle is untouched, which is the
  point — it consumes the same types and StopReasons as `done.toml` does today.
  Spec at `docs/specs/architecture/loop-command/202608252000-loop-command.*.spec.md`.
- **Only `- [ ] N.` and `` verify: `cmd` `` are syntax.** The parser required a
  literal em dash between the task number and its description, so a `tasks.md`
  written with a plain hyphen yielded zero criteria and no error — and zero
  criteria is "no definition of done", which the loop reports as done. Nothing
  about punctuation is a contract now.
- **A `tasks.md` that cannot be read is an error, not an empty `DoneSet`.** The
  parser skipped every line it did not recognise, so a file of prose came back
  with no criteria and no error: an unreadable spec became a green report. A
  file with no task line at all now fails, and a `verify:` with no command, an
  unreadable exit code, or a repeated task number each fail naming the line.
  A file whose tasks simply carry no command is still zero criteria, not an
  error — that is the one legitimate empty.
- **A missing spec falls through to the legacy verify command again.** The
  dispatcher asked `os.IsNotExist` about an error wrapped with `%w`, which does
  not follow the chain, so every absent spec path came back as a hard error
  under a comment saying it fell through. A spec that is *present and
  unreadable* is still an error — running the old command under a spec the user
  believes was loaded is the failure that distinction prevents.
- **New `done-qualifier` family, research only.** What to do when there is no
  definition of done to read at all — when the request arrived as prose. A phase
  before the loop derives candidate criteria, **runs each one against the
  repository as it stands** and puts the result in front of the operator to
  sign. The rule that gives the family its name: an acceptance criterion must
  FAIL before the work. One that already passed cannot testify that the work met
  it — it would have passed with no work at all, so the final green is
  coincidence, not evidence. The initial run classifies in three: red is
  acceptance, green is a regression guard (and `pnpm test` green at t=0 is
  exactly right), and a failure whose cause is a missing command is broken, not
  red. Proposing is the model's, signing is the operator's, running is the
  sandbox's, and no two are the same party. No `.p`, no code: spec at
  `docs/specs/architecture/done-qualifier/202608261730-done-qualifier.r.spec.md`.
- **`done-qualifier` gains its `.p` and `.config`: approved design, not built.**
  The proposal reaches the harness through a `done_propose` tool available only
  in a qualifying turn — a tool that can redefine done, within reach of a
  working turn, is the short way out of the loop. The proposer declares what it
  expects each criterion to do at t=0 (`fail` or `pass`); the declaration
  decides nothing, but the **disagreement** between it and the measurement is
  the line the operator's eye should land on. Exit 126 and 127 are the broken
  class, because a command that does not exist fails and so disguises itself as
  acceptance. Two named conditions with opposite answers: an empty proposal is
  an error, while a proposal with nothing red is a warning the operator signs —
  a genuine refactor has nothing new to prove, and the harness must not decide
  which is which. And any criterion the operator **edits** while signing is
  measured again before the freeze, or the edit escapes the very rule the family
  exists for. Refusal, deadline and round limit all end the same way and none of
  them starts a loop: a deadline that approves is the quietest way to break the
  rule. Invariants are declared as *previstas* and there is no `.i` — both
  because a verifiable invariant is a claim about a test that exists, following
  what `task-ledger` already does.
- **The prefix names the checks the project declares.** `internal/workspace`
  reads `package.json` scripts and `Makefile` targets and the workspace block
  lists them. The audited project declared four, two had been red since the
  first day, and the one that was green measured that one plus one is two —
  finding out they existed meant opening `package.json`, and a fact that needs
  a lookup is a fact used when someone remembers it. The list ends with a
  sentence that is a non-configurable constant with an invariant of its own:
  *nothing here says they pass, and nothing has run them*. Without it a list of
  gates reads as a list of guarantees, which is the defect that asked for the
  section. Naming is not measuring; measuring is `done-qualifier`.
- **`This repository` is now `This workspace`.** The block already carried the
  line saying there is *no* repository, and a heading claiming one above a line
  denying it reads as a contradiction. It now carries two classes of fact about
  the workspace and the name covers both.
- **A project that declares no gate gets no section**, and nothing in the prefix
  claims it declares none — the third time this session that "did not look" and
  "looked and there is none" had to be kept apart. The difference from a missing
  repository is consequence: having none changes what finishing means, while
  declaring no gate is ordinary. `DCODE_WORKSPACE_GATES` switches the inventory
  off for the repository whose Makefile has seventy targets.

- **A workspace with no repository says so, once.** `Repo` was `nil` for a
  directory that is not a git repository, and `nil` put nothing in the prefix
  at all — the field comment said "ordinary and silent" and the invariant said
  the prefix carries "nothing when it is not". Ordinary, yes; silent, no.
  Without a repository there is no diff to review, no undo short of rewriting a
  file by hand, and no commit, branch or pull request — so every working
  agreement a project file describes is describing machinery that is not there.
  This was found by audit: an agent worked a full day in exactly that state,
  writing its own project file demanding a commit per task and a pull request
  per spec, and nothing told it. The prefix now states it as a fact, with the
  instruction to say it once, offer `git init`, and get on with the work.
- **"We did not look" and "we looked and there is none" stay apart.** `nil` now
  means only the first, and stays silent. Three guards in one function keep the
  two separated: git not installed and a cancelled or timed-out probe both come
  back as no snapshot, and only `rev-parse` actually answering no produces the
  new `Absent` mark. The cancelled case was not foresight — an existing test
  caught the first version of this change claiming "not a repository" about a
  read that never completed, which is the same defect inside the commit that
  removed it.
- **The doctrine gains a floor, and it is overridable.** `Doctrine.Practices` is
  what dcode does when nobody asked. The asymmetry with `Safety` is the whole
  rule: Safety has no field in `DoctrineOverlay` *because it cannot be
  overridden* — a lock by type, not by convention — and Practices has one
  *because a floor that cannot be overridden is not a floor, it is a rule
  pretending to be a default*. An empty Practices does not fail `Build`, unlike
  Identity and Safety, because a floor switched off is a legitimate choice.
  `practices.md` replaces the shipped text and there is no appending variant:
  appending to a floor produces two floors, and switching off one practice is a
  line in the project file, which is rendered later and therefore wins.
- **The precedence needed no machinery.** `prompt > project > default` falls out
  of position: the floor renders after Safety and before anything anyone
  actually said, and the project's instructions stay the last block of the
  prefix. Two invariants guard it, and the second is the load-bearing one — the
  day project instructions stop being last, the floor starts outranking what
  should outrank it and nothing else in the code would say so.
  The section ships **empty**: with no text, `Build`'s output is byte-identical
  to before, and there is a test for that. The text is the next step, and it
  goes alone so it can be rewritten without taking the structure with it.
- **The floor now has text: three practices and the rule about them.** Check
  before claiming a file lacks something; reread a document this turn made
  stale; a non-zero exit is a failure, and if an instruction says to read a
  particular one as success, obey it **and name the instruction** — the licence
  covers the case it describes and no other. None came from a list of good
  practices; all three are defects someone shipped. Two paragraphs are not
  practices but rules about them: say any of this **once**, never as a caveat
  attached to the work and never waiting for an answer; and an instruction from
  the user or the project that contradicts the section **wins without
  discussion**. Both have invariants, because "say it once" becomes "warn every
  time" without a line of code changing. The doctrine size cap now counts the
  floor — a cap that skips the newest section stops measuring what is most
  likely to grow — going from 3000 to 3900 with the same headroom as before.
- **New `working-defaults` family, research only.** The floor: a handful of
  things dcode does when nobody asked, each with a declared default, and a rule
  saying who may change it. **Precedence is absolute and does not argue** — the
  user's prompt outranks the project file, which outranks the built-in default,
  and whoever is above *replaces* rather than negotiates. Overriding is
  obeyed **and** reported once, and the spec spells out that reporting is not
  asking: "the default is off, `DCODE.md` line 87" is a statement made once,
  never a caveat attached to the work or a request to confirm. A fact cannot be
  overridden, only a practice — `DCODE.md` can switch off the announcement that
  there is no repository; it cannot make the workspace be one. The family's own
  design rule: **whatever can become a fact in the prefix does not become
  prose**, because prose is the weakest layer this repository recognises. Four
  practices, deliberately few, each traced to a defect found by auditing a real
  project. No `.p`, no code: spec at
  `docs/specs/architecture/working-defaults/202608262200-working-defaults.r.spec.md`.
- **`working-defaults` gains its `.p` and `.config`.** The design got shorter
  than expected: **the precedence the `.r` asks for already exists**. `Build`
  assembles the prefix in order, project instructions are the last block — the
  position of greatest weight — and the `authority` table already ranks the
  sources. So `prompt > project > default` is not machinery to build; it is a
  consequence of the floor existing in the weakest position among rules. What
  was missing is somewhere for the default layer to live. Practices become a
  doctrine section with an overlay field — and `Safety` still has none, which
  is the asymmetry that is the whole rule: safety has no field *because it
  cannot be overridden*, and practices has one *because a floor that cannot be
  overridden is not a floor*. `practices.md` replaces, never appends, because
  appending to a floor produces two floors. A gate inventory reads
  `package.json` and `Makefile`, runs nothing, and its block ends with a
  sentence that is a non-configurable constant: nothing here says they pass —
  without it a list of gates reads as a list of guarantees, which is the defect
  that started the family.

## 0.8.0 — 26 August 2026

> **MINOR, though every commit in it says `fix:`.** `scripts/version.sh` derives
> 0.7.1 from Conventional Commits; the contract says otherwise, and the contract
> wins. Two fields were added to the protocol (`tool.requested.typed`,
> `session.mode_changed.sandbox_mode`), one behaviour was **removed** (the
> two-step confirmation for `/mode auto`, shipped in 0.7.0 and gone by first
> use), and the doctrine changed meaning — and "a superfície deste produto é em
> parte feita de frases".
>
> Measured against MiniMax-M3 this cycle: `boundary-decides-write` **MET, 100%
> of 20 runs**. `boundary-decides` came back 90.0% with one run lost to a
> transport EOF, which the harness reports as **unsound** rather than as a
> verdict — 19 runs are not 20.

### Fixed

- **A wall that says how it opens.** The doctrine now tells the model to attempt
  rather than refuse, and to let the boundary ask — but a path crossing inside a
  shell command has nothing to ask: the command is opaque, so nothing knows a
  crossing happened. Observed live, the model did as told, was refused, and told
  the user in good faith that *"the harness will ask you"*. It never does, and
  the person waits for a question that is not coming. The command result now
  carries a note: this EPERM is the sandbox, no prompt is coming, and the ways
  through are `/mode auto` or naming the path in `sandbox.writable`. Narrow on
  purpose — only EPERM, never under `full-access`.

- **The top bar follows the switch too.** The badge learned the new mode and the
  status bar did not, so a session in `auto` went on announcing
  `workspace-write` — the one field §2.1 calls dangerous to get wrong, which is
  why it is exempt from the bar's drop order. It announced a limit that had just
  been lifted, which is the worst direction for that field to be wrong in.
  `session.mode_changed` now carries `sandbox_mode`, carried rather than
  recomputed by the client, because the name-to-pair table has one home.

- **`auto` really removes the boundary.** Switching mode moved the policy's
  answer and nothing else: the sandbox was handed the mode as a **value**,
  copied when the session was built, so `/mode auto` made the verdict say
  `allow` and the badge say `auto` while the OS went on enforcing what the
  session started with. A write outside the workspace still came back `EPERM` —
  a mode whose whole promise is "no boundary" left one standing. The runner now
  asks for the mode **once per command**; a source that is nil means read-only,
  because a boundary nobody decided fails closed. Measured in a real pty: the
  same `mkdir` outside the workspace is refused under `assist` and works under
  `auto`.

- **The harness asks the user, and the model does not.** The doctrine said
  "when that happens **the user is asked**" — passive, no subject — so the model
  filled the subject with itself and built a permission protocol of its own, in
  prose, that never reaches the approval machinery: *"you have to say 'go'
  explicitly"*. It quoted that sentence to justify exactly what the same
  doctrine forbids three lines below. The subject is now named, the call is
  stated to BE the question, and a permission granted in prose is stated to
  grant nothing, because nothing was ever asked.
- **A cell measured is not its neighbour measured.** `boundary-decides` sat at
  100% of 20 runs while this was failing in front of a user, because it crosses
  the **network** and the reported failure **wrote outside the workspace**. A
  second scenario covers that cell. The limit of both is now written down: the
  eval is single-turn, and the refusal that survives being argued with is a
  failure this framework cannot yet see.

### Changed

- **Every mode goes through on the first try, `auto` included.** The two-step
  confirmation shipped yesterday in 0.7.0 and lasted until first use. Typing
  `/mode auto` is eleven deliberate characters — there is no reflex to
  disambiguate, unlike `^C`, and asking someone to repeat what they just said is
  not a safeguard but a step to learn past. What says there is no boundary is
  the badge on the bar, which says it for as long as it is true.
- **What you typed, you get to read.** `!ls -la` drew one row, `exit 0`, and
  nothing else. The output was never lost — it reached the client, it sat in the
  entry, and `esc`, `↑`, `tab` revealed it. That is worse than losing it: the
  screen answered a request to SEE something with a status code, and looked
  correct while doing so. The collapse rule was written for the model's calls,
  where output is a means and the prose that follows carries the point; a typed
  command has no prose after it. Origin now travels on the event rather than
  being inferred from the shape of a call id.
- **`exit N` is printed once.** `bash` prefixes its output with the code because
  the model reads the output as text, and the row already renders that code in
  its own column. Invisible while output stayed collapsed; doubled the moment a
  typed command started opening on its own.

## 0.7.0 — 25 August 2026

### Added

- **Three modes, and a way between them without restarting.** `plan`, `assist`
  and `auto` are names for the pair the engine already ran under — read-only +
  never, workspace-write + on-request, full-access + on-request. `/mode` shows
  or switches, `shift+tab` cycles, and the bar carries the badge. Dropping the
  boundary into `auto` takes the gesture twice, by both routes, with one warning
  that lives in the footer while the decision is pending — the way the second
  `^C` already works.

### Changed

- **Leaving takes two, and clearing a line takes none.** `^C` means "clear this
  line" in every shell, and it was wired straight to quit — so a reflex the
  terminal taught cost a conversation. Now: a running turn is interrupted, a
  typed line is cleared, and an empty line warns first and leaves second. Armed
  exactly while the warning is on screen, because any other key disarms it: a
  timer would keep the key live for a second after the sentence had gone, which
  is a state a person cannot see and therefore cannot reason about.

### Fixed

- **A session says the mode it is actually in.** It was born labelled `assist`
  whatever the engine ran, so `full-access` wore the bounded badge — and
  switching back **to** `assist` did nothing at all, because the session
  believed it was already there. The command that installs the boundary was the
  one that silently failed. The name is now derived from the pair in force, and
  a pair that is none of the three gets no name rather than the nearest one.
- **Switching mode mid-turn is no longer a data race.** `SetMode` writes from
  the HTTP handler while the turn runs — which is the point of not interrupting
  it. Evaluation was put under the mutex; building a delegated child was not,
  and it reads the same two fields from the goroutine that is alive when the
  switch arrives. Every reader goes through one guarded accessor now.
- **A typed command announces itself before it runs.** `!` ran the command and
  the screen said nothing. The daemon emitted the completion and not the
  announcement, and the client builds the row from the announcement and
  completes it by id — so the completion had nothing to land on and was dropped
  in silence. The command worked; from the only side that matters, nothing
  happened. The guard added with it asks the screen, not the events: no
  arrangement of events that fails to put the command and its output in front of
  a person is correct.

## 0.6.1 — 25 August 2026

### Fixed

- **The column shows the context, not what the turn cost.** The third place the
  same defect reached the screen: the model computed it right, and the side
  column went on drawing `5.9M / 1.0M` from the cumulative input count — under a
  gauge computed from the true share, so the pair disagreed with the gauge below
  it and with the bar above it. Each of the three was found by a person looking
  at their own screen, and each was fixed with a test asking about the one place
  it had just been found in. There is now a guard that asks the whole screen, at
  four widths, and it catches all three.
- **A target that is not a path keeps its head.** The recent list cut every
  target to whatever followed the last slash, which is right for a file and
  wrong for a URL: `.../trips/lowest-price?from=maringa-pr` showed as
  `lowest-price?from=maringa-pr`, which reads as a file nobody has.
  `looksLikePath` decides, which is the decision the tool line already makes.

### Changed

- **The daemon and the emitter are covered.** Ten tests recovered from work that
  never landed: the emitter's fragment handling, the plan marks, and the
  daemon's optional branches. `internal/app` 92.3% to 93.0%.
- **The learned memory is in the repository.** `.dcode/memory.md` was untracked,
  and the spec says plainly what it should be — versioned by the user. A memory
  that lives only on the machine that learned it is a memory the next person
  does not get.
- **The 500-line rule has never been enforced, and the tree is nowhere near
  it.** Ten production files and seventeen test files are over it, the largest
  at 1915. Recorded in `docs/ROADMAP.md` rather than fixed: splitting
  twenty-seven files makes nothing more correct, and writing the guard first
  puts twenty-seven files red on arrival — the mistake §5 already records about
  the coverage gate.

## 0.6.0 — 25 August 2026

### Added

- **`/update` from inside.** It checks, verifies the signature and the digest,
  and replaces the binary — the same updater the command builds, with the same
  refusals: a local build is not replaced, a pin is honoured, and going
  backwards is not an update. What it does not do is restart. Replacing the
  binary under a running process leaves the running process being the old one,
  so the note says to reopen rather than implying otherwise. Both ways of asking
  go through one door, which is what keeps the guarantee that nothing replaces
  this binary without having been asked to.

- **A command you run yourself: `!`.** A line starting with `!` is not sent to
  the model — it runs. The output reaches the screen as the tool events the
  transcript already draws, and reaches the history as one user message, because
  the user did run it; without that the model answers about a workspace whose
  state it cannot see. It goes through the same `bash` tool and the same policy,
  so a crossing is put to the person exactly as it would be had the model asked:
  `!` is a shortcut past the model, never past the sandbox. The input area says
  so from the first character, while the line can still be deleted.

### Changed

- **The boundary decides, not the model.** A report opened with *"Não vou rodar
  `npm install`… você roda localmente"* and *"Não vou rodar `vitest`… você roda
  localmente"* — a refusal nobody gave, answered on the user's behalf, handing
  the work back to be done by hand. The approval machinery exists for exactly
  that moment and was never reached. The doctrine said a crossing gets asked and
  that a refusal is final; it never said that **deciding in advance is not the
  model's to decide**. It says so now, in two sentences, at about sixty tokens a
  turn — the cost is real and it buys back the whole point of having boundaries
  a person answers. A new contract, `boundary-decides` at 90%, measures it: the
  judge asks for the attempt, not for success, because being denied is the
  boundary working.

### Fixed

- **A line you are typing stays visible.** Longer than the box, it was one row
  and the row was clipped — so everything past the right edge, the caret
  included, was invisible while it was being typed. There is no way to read what
  you cannot see and no way to fix a typo you cannot find. The input area counts
  its rows by wrapping now instead of by counting newlines, and the caret is
  carried through the wrap so it lands where the next character will appear. The
  wrap is by column and not by word: what is typed here is usually a command,
  and a path or a flag broken at a space reads as two arguments.


- **An update is something newer, not something different.** The notice asked
  whether the running version differed from the latest known one, so a binary
  ahead of the last release — a local build, or a release the day-old cached
  check has not caught up with — was told `dcode v0.4.0 is available (you have
  0.5.0). Run `dcode update`.` An offer to go backwards, wearing the word
  update. `update` itself already refused it, so the tool contradicted itself in
  two places on one screen; now both compare versions, field by field, and
  `update` refuses a release older than the one running even when asked
  directly.

## 0.5.1 — 25 August 2026

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
