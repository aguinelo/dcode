#!/usr/bin/env bash
# Assembles the "Unreleased" sections from fragments, and folds them into a
# version's section at release.
#
# A change is recorded as one file, changelog.d/<yyyymmddhhmm>-<slug>.md (the
# core) or desktop/changelog.d/... (the desktop app), carrying both editions:
#
#   ## en
#
#   - **What changed.** And why.
#
#   ## pt-BR
#
#   - **O que mudou.** E por quê.
#
# On 16 October 2026 fifteen pull requests in flight all conflicted on the top
# of "Unreleased" in both editions, and integration became serial. A file no
# other pull request writes cannot conflict, so no pull request edits a shared
# line any more: the shared lines are written here, once, at release.
#
#   changelog-assemble.sh show [root|desktop] [en|pt]
#       prints the assembled Unreleased entries, newest first
#   changelog-assemble.sh release <version> [root|desktop]
#       folds the fragments into a "## <version> — <date>" section of both
#       editions, deletes them, and for the core recounts the state table
#
# internal/evals/fragments_test.go holds every fragment to that shape and this
# script's output to the fragments, so neither can drift from the other.
set -euo pipefail

cd "$(dirname "$0")/.."

die() { echo "changelog-assemble: $*" >&2; exit 1; }

set_dir() {
  case "$1" in
    root) echo changelog.d ;;
    desktop) echo desktop/changelog.d ;;
    *) die "no changelog '$1'; use root or desktop" ;;
  esac
}

set_file() { # <set> <en|pt>
  local base=""
  [ "$1" = desktop ] && base="desktop/"
  case "$2" in
    en) echo "${base}CHANGELOG.md" ;;
    pt) echo "${base}CHANGELOG.pt-BR.md" ;;
    *) die "no edition '$2'; use en or pt" ;;
  esac
}

# Newest first, the order the section has always been read in. The names carry
# no spaces — the guard holds them to <yyyymmddhhmm>-<slug>.md.
fragments() {
  local dir
  dir="$(set_dir "$1")"
  [ -d "$dir" ] || return 0
  find "$dir" -maxdepth 1 -type f -name '*.md' | sort -r
}

# One edition of one fragment, without the blank lines around it. An empty
# edition stops the script: an entry missing in one language is not an entry
# to publish in the other alone.
edition() { # <file> <en|pt>
  local head="## en"
  [ "$2" = pt ] && head="## pt-BR"
  local out
  out="$(awk -v head="$head" '
    /^## / { inside = ($0 == head); next }
    inside { lines[++n] = $0 }
    END {
      first = 1; while (first <= n && lines[first] ~ /^[[:space:]]*$/) first++
      last = n; while (last >= first && lines[last] ~ /^[[:space:]]*$/) last--
      for (i = first; i <= last; i++) print lines[i]
    }' "$1")"
  [ -n "$out" ] || die "$1 has no '$head' edition"
  printf '%s\n' "$out"
}

show() { # <set> <en|pt>
  local f
  for f in $(fragments "$1"); do
    edition "$f" "$2"
  done
}

heading_date() { # <en|pt>
  local d m y
  d="$(date +%-d)"; m=$(( 10#$(date +%m) - 1 )); y="$(date +%Y)"
  if [ "$1" = pt ]; then
    local meses=(janeiro fevereiro março abril maio junho julho agosto setembro outubro novembro dezembro)
    printf '%s de %s de %s' "$d" "${meses[$m]}" "$y"
  else
    local months=(January February March April May June July August September October November December)
    printf '%s %s %s' "$d" "${months[$m]}" "$y"
  fi
}

release() { # <version> <set>
  local version="${1#v}" set="$2"
  printf '%s' "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$' \
    || die "'$1' is not MAJOR.MINOR.PATCH"
  [ -n "$(fragments "$set")" ] || die "no fragment under $(set_dir "$set"); nothing to release"

  local lang file heading section tmp
  for lang in en pt; do
    file="$(set_file "$set" "$lang")"
    heading="## Unreleased"; [ "$lang" = pt ] && heading="## Não publicado"
    section="$(mktemp)"; tmp="$(mktemp)"
    { printf '## %s — %s\n\n' "$version" "$(heading_date "$lang")"; show "$set" "$lang"; } >"$section"
    # The new section goes right under the Unreleased pointer, above the last
    # release. A file with no Unreleased heading is refused, not appended to.
    awk -v head="$heading" -v sect="$section" '
      state == 0 && $0 == head { state = 1; print; next }
      state == 1 && /^## / { while ((getline l < sect) > 0) print l; print ""; state = 2 }
      { print }
      END {
        if (state == 1) { print ""; while ((getline l < sect) > 0) print l }
        if (state == 0) exit 3
      }' "$file" >"$tmp" || die "$file has no '$heading' heading"
    mv "$tmp" "$file"
    rm -f "$section"
  done

  fragments "$set" | xargs rm -f
  rmdir "$(set_dir "$set")" 2>/dev/null || true
  echo "changelog-assemble: $set folded into $version; the fragments are deleted"

  if [ "$set" = root ]; then
    # With no fragment pending, the state-table guard holds the table to the
    # tree exactly, so the recount is part of the fold rather than a step to
    # remember after it.
    go test ./internal/evals -run '^TestTheStateTableIsCountedAndNotCarried$' -count=1 -args -write-state >/dev/null \
      || die "the state table could not be recounted; run the guard to see why"
    echo "changelog-assemble: state table recounted from the tree"
    echo "changelog-assemble: the 'Current state' heading and its prose are still yours to rewrite"
  fi
}

case "${1:-}" in
  show) show "${2:-root}" "${3:-en}" ;;
  release)
    [ -n "${2:-}" ] || die "release needs a version: release <version> [root|desktop]"
    release "$2" "${3:-root}"
    ;;
  *) die "usage: changelog-assemble.sh show [root|desktop] [en|pt] | release <version> [root|desktop]" ;;
esac
