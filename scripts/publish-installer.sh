#!/usr/bin/env bash
#
# Carry the pinned installer back to main.
#
# The digests only exist once the artifacts are built and signed, so this runs
# after the release is public and the pinned installer is already attached to
# it. That ordering decides how every failure here must behave: failing now does
# not un-publish anything — it paints a successful release red, which is the one
# report nobody can act on.
#
# So every recoverable condition exits zero and says so loudly. What must never
# happen is the opposite: exiting zero having silently done nothing.
#
# Why main and not only the release asset: the URL the README publishes is
# raw.githubusercontent.com/.../main/install.sh, so an asset-only pin never
# reaches the people who install. It is also the entire security argument. A
# release asset can be replaced leaving no public trace; a line in a tracked
# file cannot, because changing it is a commit — visible in the log, in the
# diff, and attributed. Two routes, and the attacker needs both.
#
# The commit subject is exact and load-bearing: scripts/version.sh skips it, so
# that a release does not make every later version query answer "there are
# commits since the tag" when nothing human changed.
#
# Usage: publish-installer.sh <version> <pinned-install.sh>
# Env:   GH_TOKEN           the push credential; absent means "not configured"
#        INSTALLER_REMOTE   override the remote, for tests

set -euo pipefail

version="${1:?usage: publish-installer.sh <version> <pinned-install.sh>}"
pinned="${2:?usage: publish-installer.sh <version> <pinned-install.sh>}"

# The one condition that is NOT recoverable. Pushing on without the pinned file
# would leave main carrying the PREVIOUS release's digests while the release
# reports success. Every install would then fall back to checksums.txt — the
# correct behaviour for an unpinned installer, and therefore invisible.
if [ ! -f "$pinned" ]; then
  echo "::error::no pinned installer at $pinned; refusing to leave main on the previous release's digests" >&2
  exit 1
fi

if [ -z "${GH_TOKEN:-}" ] && [ -z "${INSTALLER_REMOTE:-}" ]; then
  echo "no GH_TOKEN: main keeps the previous digests. The pinned installer is" \
       "attached to the release and installable by URL — one channel fewer, not" \
       "a broken release."
  exit 0
fi

remote="${INSTALLER_REMOTE:-https://x-access-token:${GH_TOKEN}@github.com/aguinelo/dcode.git}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT INT TERM

# Shallow, and main specifically. This runs from a tag checkout, which is
# detached and carries the pre-pin file; cloning is simpler than reasoning about
# what the workspace happens to be pointing at.
if ! git clone --quiet --depth 1 --branch main "$remote" "$work/repo" 2>"$work/err"; then
  echo "::warning::could not reach the repository; the release is published and the" \
       "pinned installer is attached to it, but main still carries the previous" \
       "release's digests. $(tr -d '\n' < "$work/err")"
  exit 0
fi

cp "$pinned" "$work/repo/install.sh"
chmod +x "$work/repo/install.sh"

cd "$work/repo"
git config user.name  "dcode release"
git config user.email "noreply@github.com"
git add install.sh

# `git commit` exits non-zero with nothing staged, so the naive form turns a
# harmless re-run into a red job. The second half of that is worse: someone
# re-runs the workflow for another reason and reads this red as their fault.
if git diff --cached --quiet; then
  echo "main already carries this installer; unchanged, nothing to push."
  exit 0
fi

git commit --quiet -m "chore(release): pin the installer to v${version}"

# Someone may have pushed to main between the clone and here. Rebasing once and
# retrying costs a second; giving up would leave main stale for a whole release
# cycle. A second rejection is reported and not fought — at that point something
# is contending that this script should not be racing.
if ! git push --quiet origin main 2>"$work/perr"; then
  if git pull --quiet --rebase origin main && git push --quiet origin main; then
    echo "main updated to v${version}, after rebasing onto a concurrent push"
    exit 0
  fi
  echo "::warning::could not push the pinned installer to main; the release is" \
       "published and the pinned installer is attached to it, but main still" \
       "carries the previous release's digests. $(tr -d '\n' < "$work/perr")"
  exit 0
fi

echo "main updated to v${version}"
