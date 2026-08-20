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

## Current state — 20 August 2026

**What it is.** An agentic coding harness in Go: a daemon, a terminal client and
the agent loop between them, as a single static binary, with no cgo outside the
isolated package.

**Where it stands.**

| | |
|---|---|
| spec families | 13, with 68 decision changelogs |
| behavioural contracts | 42 declared |
| **contracts measured against a model** | **3** |
| coverage | 95.0%, gate at 90% |
| CI | macOS + Linux matrix, gated on the **union** of the profiles |
| published version | **0.1.0** |

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
