# dcode

An agentic coding harness written in Go: a daemon, a terminal client, and the
agent loop between them, shipped as one static binary. No cgo outside the
isolated package.

These are the instructions for **any** agent working in this repository —
`dcode` itself reads this file, and so does Claude Code through `CLAUDE.md`.
Keep it about the repository. Tool-specific configuration belongs in the file
of the tool that has it.

## Rules

- Do what has been asked; nothing more, nothing less
- NEVER create files unless absolutely necessary — prefer editing existing files
- NEVER create documentation files unless explicitly requested
- NEVER save working files or tests to the root
- ALWAYS read a file before editing it
- NEVER commit secrets, credentials, or `.env` files
- NEVER add a `Co-Authored-By` trailer to a commit. It is authorship attribution
  under git and GitHub convention; a tool is the facilitator, not a co-author.
- Keep files under 500 lines
- Validate input at system boundaries

## Working agreement — branches, staging, CI

Every rule below exists because it was broken, and each names the failure so it
can be recognised before it repeats rather than after.

### One theme, one branch, one PR

- A theme is what fits in a PR title **without the word "and"**. If the title
  needs an "and", it is two branches.
- Before starting work whose title would not fit the open PR, cut a new branch
  from an updated `main`. Continuing on the open branch because it is already
  checked out is how a PR reaches 11 commits and 17k lines — unreviewable, and
  merged on trust rather than on reading.
- A defect found while doing something else gets **its own branch**, unless
  fixing it is what unblocks the current work. "It was right there" is not a
  reason to widen a PR.
- Repeated approval to continue (`segue`, `pode ir`) grants **the work**, never
  the branch. Ask which branch, or start a new one.

### Stage explicitly

- **Never `git add -A` or `git add .`.** Stage named paths. A 41KB HTML mock
  entered a PR unnoticed exactly this way.
- Run `git status` before committing and account for **every** listed file. A
  file you cannot explain is a file that does not belong in the commit.

### A push is not finished until CI is read

- After every push, check CI (`gh pr checks`). A push whose result was never
  read is a push that was never finished.
- **Never report green from a local run alone.** `make check` passing locally
  and CI passing are different claims. When they disagree it is usually because
  the working tree and the repository differ — which is precisely the failure
  local runs cannot see.
- Never merge a red PR, and never merge without having read the checks.

### What the build depends on must be in the repository

- After creating a file the build, the gate or CI depends on, verify it is
  actually tracked: `git check-ignore -v <path>` and `git ls-files <path>`.
- `.gitignore` patterns without a leading `/` match at **any depth**.
  `coverage.*` swallowed `scripts/coverage.sh`, so the coverage gate never ran
  in CI while every local run reported it green.

## Build and test

```bash
make check   # lint, race, coverage gate, build — the whole gate
make test    # the suite alone
make cover   # the suite with the coverage gate
make install # build and install into ~/.local/bin
```

`make check` is the local approximation of CI, and CI is the claim that counts.

Behavioural contracts live behind the `eval` build tag and are **not** in
`make check`: each scenario runs against a real model and costs money. Run them
with `make eval`, and run `make eval-build` in any pull request that touches
`internal/evals` — nothing in `make check` compiles code behind that tag, so it
rots in silence.

## Where the rules are written

These four are the source of truth. Do not restate them here; a fifth copy is a
copy that drifts.

| Document | What it governs |
|---|---|
| `docs/conventions/SDD-HARNESS.md` | the spec protocol: `.r` → `.p` → `.config` / `.i`, and `changelog/` |
| `docs/conventions/TESTING.md` | TDD, reproducing tests, the coverage gate, critical scenarios |
| `docs/conventions/GO-CODE-REVIEW.md` | what Go code in this repository looks like |
| `docs/conventions/LANGUAGE.md` | which language each artefact is written in |

The short version of the last one, because it is the one most easily broken:
**specs and changelogs in Portuguese; code, comments, commit messages and pull
request titles in English.**

`docs/ROADMAP.md` is what is known and not yet done. `docs/DECISIONS.md` records
why the specs say what they say where the reason came from the code.
