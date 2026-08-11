#!/usr/bin/env bash
#
# Publish the Homebrew formula to the tap.
#
# This runs AFTER the release exists, and that ordering decides how every
# failure here must behave. By the time it runs, the binaries are signed, the
# release is public, and the formula is attached to it and installable by URL.
# Failing now does not un-publish anything — it paints a successful release red,
# which is the one report nobody can act on.
#
# So every recoverable condition exits zero and says so loudly. What must never
# happen is the opposite: exiting zero having silently done nothing. That is why
# the token check lives here rather than in the workflow's `if:` — a step's own
# `env` block feeding that step's own condition is a subtlety whose failure mode
# is invisible, because the tap would simply never update and the release would
# stay green.
#
# Usage: publish-tap.sh <version> <formula-path>
# Env:   TAP_TOKEN   the push credential; absent means "no tap configured"
#        TAP_REMOTE  override the remote, for tests

set -euo pipefail

version="${1:?usage: publish-tap.sh <version> <formula-path>}"
formula="${2:?usage: publish-tap.sh <version> <formula-path>}"

# The one condition that is NOT recoverable. Pushing on without a formula would
# leave the tap pointing at whatever it held before while the release reports
# success — an installer serving the previous version to everyone who runs
# `brew upgrade`, with nothing anywhere saying so.
if [ ! -f "$formula" ]; then
  echo "::error::no formula at $formula; refusing to leave the tap on the previous version" >&2
  exit 1
fi

if [ -z "${TAP_TOKEN:-}" ]; then
  echo "no TAP_TOKEN: the tap is not updated. The formula is attached to the" \
       "release and installable by URL — one channel fewer, not a broken release."
  exit 0
fi

remote="${TAP_REMOTE:-https://x-access-token:${TAP_TOKEN}@github.com/aguinelo/homebrew-dcode.git}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# A tap nobody created is a misconfiguration worth seeing, not a reason to
# redden a release that already succeeded. Warned where the warning shows on the
# run, and the job stays green.
if ! git clone --quiet "$remote" "$work/tap" 2>"$work/err"; then
  echo "::warning::could not reach the tap; the release is published and the formula" \
       "is attached to it, but the tap still points at the previous version." \
       "$(tr -d '\n' < "$work/err")"
  exit 0
fi

mkdir -p "$work/tap/Formula"
cp "$formula" "$work/tap/Formula/dcode.rb"

cd "$work/tap"
git config user.name  "dcode release"
git config user.email "noreply@github.com"
git add Formula/dcode.rb

# `git commit` exits non-zero with nothing staged, so the naive form turns a
# harmless re-run into a red job. The second half of that is worse: someone
# re-runs the workflow to fix something else and reads this red as their fault.
if git diff --cached --quiet; then
  echo "the tap already carries this formula; unchanged, nothing to push."
  exit 0
fi

git commit --quiet -m "dcode $version"
git push --quiet
echo "tap updated to $version"
