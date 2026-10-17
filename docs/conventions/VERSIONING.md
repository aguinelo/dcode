# Versioning convention

🇧🇷 [Versão em português](VERSIONING.pt-BR.md)

## 1. SemVer, and what it does not see

Versions are `MAJOR.MINOR.PATCH`, tagged `vX.Y.Z`, as [SemVer 2.0.0][semver]
defines them.

**Before 1.0, a breaking change raises MINOR rather than MAJOR.** `0.x` says the
shape is still moving, and spending the 1 on the first break is how projects
reach 7.0 with nothing stable in them.

[semver]: https://semver.org/

## 2. The public surface, named

Most projects adopt SemVer and skip this, then argue in review about whether
something was breaking. Here it is written down.

**Covered by the version:**

| Surface | Why it counts |
|---|---|
| CLI commands and flags | what a person types |
| configuration keys, `DCODE_*` and TOML | what a person sets, and what an admin locks |
| `pkg/client` | the package's own doc calls it the public API for daemon consumers |
| the client–server protocol | a client and a daemon at different versions have to agree |

**Not covered:** everything under `internal/`, the on-disk shape of session
records, and the eval fixtures. They change without a version saying so, and
nobody outside builds on them.

## 3. Behaviour is part of the contract

This is where an agentic product leaves the standard behind, and the gap is not
academic.

Changing one sentence of a tool description moved delegation from zero uses to
five in a measured run. No API changed. No signature moved. A user would have
felt it immediately.

So:

- **a declared behavioural contract that is removed, or whose threshold drops,
  is at least MINOR** — and the changelog names the contract;
- **a tool description, a reminder or the doctrine changing meaning is at least
  MINOR**, for the same reason;
- a threshold that rises, or a contract that is added, is MINOR as a feature.

The rule exists because SemVer reads signatures, and this product's surface is
partly made of sentences.

## 4. Deriving, rather than deciding

Commits here already follow [Conventional Commits][cc] — `feat:`, `fix:`,
`docs:` — and did so before anyone agreed to it. The version comes from them:

```bash
./scripts/version.sh      # the next version, from the commits since the last tag
./scripts/changelog.sh v0.0.1 en   # the pull requests that landed, in English
./scripts/changelog.sh v0.0.1 pt   # and in Portuguese
make changelog                     # Unreleased, assembled from the fragments
```

| commit | effect |
|---|---|
| `feat:` | MINOR |
| `fix:`, `chore:`, `docs:`, `test:`, `refactor:`, `perf:`, `build:`, `ci:` | PATCH |
| any `type!:`, or `BREAKING CHANGE:` in the body | MAJOR — or MINOR while below 1.0 |

[cc]: https://www.conventionalcommits.org/

**Both scripts refuse rather than guess.** A commit that does not match the
convention is an error, not a patch-by-default; a tag that is not `vX.Y.Z` is an
error, not an arithmetic attempt. A version chosen by a silent guess is worse
than a script that stops.

## 5. The changelog is assembled from fragments, never generated as prose

Each change is a fragment written on its own branch —
`changelog.d/<yyyymmddhhmm>-<slug>.md`, or `desktop/changelog.d/` for the
desktop app — carrying the entry in both editions. No pull request edits a line
another one edits, so none conflicts on the changelog. `Unreleased` is
assembled from them; `scripts/changelog-assemble.sh` renders it.

`scripts/changelog.sh` produces what mechanically exists: which pull requests
landed, grouped, numbered. At release it is the cross-check that every one of
them left a fragment.

It does not produce the **why**, and it must not. "The refusal was honest but it
instructed abandonment" is in no commit subject and comes out of no generator. A
tool that tried would write a plausible sentence about a decision nobody took,
which is worse than a list with no sentence at all.

So: the reason is written by hand, in the fragment, by whoever made the change.

## 6. Releasing

1. `make check` green, and CI green on `main`.
2. `./scripts/version.sh` for the number.
3. `./scripts/changelog-assemble.sh release X.Y.Z` folds the fragments into a
   `## X.Y.Z` section of **both editions**, deletes them, and recounts the state
   table from the tree. Compare with `./scripts/changelog.sh` that no pull
   request was left without a fragment, then rewrite the status heading and its
   prose. With no fragment pending, the state-table guard holds the table to the
   tree exactly, so this commit's CI is where a stale number fails. The
   changelog is bilingual like the README, English canonical — see
   `LANGUAGE.md`. The desktop app folds the same way, with `release X.Y.Z
   desktop`, at its own `desktop-v*` tags.
4. Merge that, then tag the merge commit: `git tag -a vX.Y.Z -m 'vX.Y.Z'`.
5. `git push origin vX.Y.Z` — this publishes. The workflow builds every
   platform, checksums, signs with cosign, and creates the GitHub release.

Step 5 is the one that does not come back. Everything before it is reversible;
a published tag is not, and it is the only step worth pausing on.
