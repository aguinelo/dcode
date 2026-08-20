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
| spec families | 13, with 63 decision changelogs |
| behavioural contracts | 42 declared |
| **contracts measured against a model** | **3** |
| coverage | 95.0%, gate at 90% |
| CI | macOS + Linux matrix, gated on the **union** of the profiles |
| published version | **0.0.1** |

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

### Distribution

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
