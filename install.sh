#!/usr/bin/env sh
# dcode installer.
#
# Verifies the artifact checksum always, and the release signature when cosign
# is here. A check that fails aborts and removes everything that was downloaded.
#
# When a release has pinned it, this file also carries the digests of that
# release's artifacts. Those arrived by a different route from the tarball — in
# git history rather than beside the download — and that is what covers a
# substituted release without asking the machine for anything.
#
# Nothing here requires a package to be installed first. Of rustup, bun, deno,
# nvm, k3s and uv, not one does, and a first install is the worst moment to ask.
# cosign is used when it happens to be there and is never mentioned when it is
# not — what is worth saying is whether a substituted release was covered, and
# either route covering it is enough.
#
# The two are independent and are treated as such. Requiring cosign made the
# weaker check conditional on the stronger one, so a machine without it got no
# binary AND no verification — which is the outcome this file exists to avoid.
# The line worth holding is not "verified or nothing", it is "never unverified
# in silence": what could not be checked is said out loud, and at the size of
# what was actually missed.
#
# Spec: docs/specs/architecture/distribution/202608072352-*.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/aguinelo/dcode/main/install.sh | sh
#
# Environment:
#   DCODE_VERSION       version to install (default: latest)
#   DCODE_INSTALL_DIR   destination (default: $HOME/.local/bin)
#   DCODE_SKIP_VERIFY   debugging the release pipeline only — never in real use
#
# Optional:
#   cosign              verifies the release signature when present. Absent, the
#                       checksum still runs and the installer says what it could
#                       not check.
set -eu

REPO="aguinelo/dcode"
VERSION="${DCODE_VERSION:-latest}"
INSTALL_DIR="${DCODE_INSTALL_DIR:-$HOME/.local/bin}"
SKIP_VERIFY="${DCODE_SKIP_VERIFY:-false}"
UNSIGNED=0
PINNED_OK=0
SUPPORTED="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

# BEGIN PINNED — gerado por scripts/installer.sh a partir do checksums.txt assinado.
# Nao edite a mao. Estes sao os digests dos artefatos que foram assinados, e a
# graca deles e viverem no historico do git, longe do host que serve o tarball.
PINNED_VERSION="0.13.0"
pinned_sum() {
  case "$1" in
    dcode_0.13.0_darwin_amd64.tar.gz) echo b984dfce2d1f515bddf5d10ea3140343e8ec694be857f1fdc1b717b58d0e333e ;;
    dcode_0.13.0_darwin_arm64.tar.gz) echo aecbb1258be5a03df92ecb344fa456235d0aa9ca4c1b6d0c6ad3431047792ce5 ;;
    dcode_0.13.0_linux_amd64.tar.gz) echo e2269e94a115d376d3634fe481c6b25178ecf26bbb15d76fbf6de35f79659178 ;;
    dcode_0.13.0_linux_arm64.tar.gz) echo 5e573a7937cfb0b733195c127aa49c4c95a769b7e439ba555405105f33d88014 ;;
  esac
}
# END PINNED

die() { printf 'dcode: %s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }

WORK=""
cleanup() { [ -n "$WORK" ] && rm -rf "$WORK"; }
trap cleanup EXIT INT TERM

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"; }

# 1. Detect the platform. An unsupported combination aborts with the list, so
#    the user learns what is available rather than what failed.
detect_platform() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin|linux) ;;
    *) die "$os is not supported. Supported: $SUPPORTED" ;;
  esac

  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "$arch is not supported. Supported: $SUPPORTED" ;;
  esac

  case " $SUPPORTED " in
    *" $os/$arch "*) ;;
    *) die "$os/$arch is not supported. Supported: $SUPPORTED" ;;
  esac

  PLATFORM_OS="$os"
  PLATFORM_ARCH="$arch"
}

# 2. Resolve the version. Pinning is the route to a reproducible install.
resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    return
  fi
  need curl
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  [ -n "$VERSION" ] || die "could not resolve the latest version"
}

download() {
  url="$1"; dest="$2"
  curl -fsSL "$url" -o "$dest" || die "could not download $url"
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    die "neither sha256sum nor shasum is installed, so nothing can be verified"
  fi
}

main() {
  need curl
  need tar
  detect_platform
  resolve_version

  bare="${VERSION#v}"
  artifact="dcode_${bare}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"
  base="https://github.com/$REPO/releases/download/$VERSION"

  WORK="$(mktemp -d)"
  info "dcode $VERSION for $PLATFORM_OS/$PLATFORM_ARCH"

  # 3. Download the artifact and everything needed to check it.
  download "$base/$artifact" "$WORK/$artifact"
  download "$base/checksums.txt" "$WORK/checksums.txt"

  if [ "$SKIP_VERIFY" = "true" ] || [ "$SKIP_VERIFY" = "1" ]; then
    printf '\n  !!  DCODE_SKIP_VERIFY is set. Nothing below is verified.\n'
    printf '  !!  This exists to debug the release pipeline. Using it for a\n'
    printf '  !!  real install turns this channel into a supply-chain vector.\n\n'
  else
    # 4. Verify the signature over the checksums file, when there is something
    #    here that can. One signature covers the whole release; the per-artifact
    #    check is the SHA-256 below, and that one runs either way.
    #
    #    Nothing is downloaded for a check that cannot happen: asking for a
    #    signature no tool here can read only adds a way to fail for a reason
    #    that is not the user's.
    if command -v cosign >/dev/null 2>&1; then
      download "$base/checksums.txt.sig" "$WORK/checksums.txt.sig"
      download "$base/checksums.txt.pem" "$WORK/checksums.txt.pem"
      cosign verify-blob \
        --certificate "$WORK/checksums.txt.pem" \
        --signature "$WORK/checksums.txt.sig" \
        --certificate-identity-regexp "^https://github.com/$REPO/" \
        --certificate-oidc-issuer https://token.actions.githubusercontent.com \
        "$WORK/checksums.txt" >/dev/null 2>&1 ||
        die "the release signature did not verify — nothing was installed"
    else
      # Noted, not printed. What the absence costs depends on whether the digest
      # below is carried, and that is not known yet.
      UNSIGNED=1
    fi

    got="$(sha256_of "$WORK/$artifact")"

    # 5. The digest this installer carries, when it carries one for this
    #    artifact. It matters because it did NOT travel with the tarball.
    #
    #    checksums.txt comes from the same host as the artifact, so whoever can
    #    replace one can replace the other and the pair stays self-consistent.
    #    This digest lives in the installer, and the installer lives in git
    #    history: a release asset can be swapped leaving no public trace, a line
    #    in a tracked file cannot. That is what lets the signature be optional
    #    without the checksum becoming decorative.
    pinned="$(pinned_sum "$artifact")"
    if [ -n "$pinned" ]; then
      [ "$pinned" = "$got" ] ||
        die "checksum mismatch for $artifact — this installer carries $pinned, the download is $got. Nothing was installed"
      PINNED_OK=1
    fi

    # 6. And against the list published with it, which catches the corruption
    #    the pinned digest would also catch, plus the case of no pin at all.
    want="$(grep " \*\{0,1\}$artifact\$" "$WORK/checksums.txt" | awk '{print $1}' | head -1)"
    [ -n "$want" ] || die "$artifact is not listed in checksums.txt"
    [ "$want" = "$got" ] ||
      die "checksum mismatch for $artifact — expected $want, got $got. Nothing was installed"

    # 7. Say what was left uncovered. Nothing else.
    #
    #    A substituted release is the only thing here that needs two routes, and
    #    two independent things cover it: the digest this installer carries, and
    #    the signature. EITHER is enough, so there is nothing to report unless
    #    both are missing. Telling someone their signature went unchecked while
    #    the check that matters passed by a route not depending on it is noise
    #    dressed as diligence.
    #
    #    And the way out is never "install a package". Nothing in this class of
    #    tool requires one — of rustup, bun, deno, nvm, k3s and uv, not one — and
    #    a first install is the worst moment to ask. An installer that answers a
    #    problem with "install something else first" has handed over a second
    #    problem. What gets printed is the installer that carries the digests.
    if [ "$PINNED_OK" != 1 ] && [ "$UNSIGNED" = 1 ]; then
      printf '\n  !   Nothing here covered a substituted release: this installer does\n'
      printf '  !   not carry the digests of %s. The SHA-256 published with the\n' "$bare"
      printf '  !   release matched, which catches a corrupted download.\n'
      printf '  !   The installer that carries them:\n'
      printf '  !     %s/install.sh\n\n' "$base"
    fi
  fi

  # 8. Extract and install.
  tar -xzf "$WORK/$artifact" -C "$WORK" || die "could not extract $artifact"
  [ -f "$WORK/dcode" ] || die "$artifact does not contain a dcode binary"
  chmod +x "$WORK/dcode"

  mkdir -p "$INSTALL_DIR" || die "could not create $INSTALL_DIR"
  mv "$WORK/dcode" "$INSTALL_DIR/dcode" || die "could not install into $INSTALL_DIR"

  # 9. Confirm the installed binary runs here, before claiming success.
  "$INSTALL_DIR/dcode" --version >/dev/null 2>&1 ||
    die "the installed binary does not run on this machine"

  info "Installed $("$INSTALL_DIR/dcode" --version) to $INSTALL_DIR/dcode"
  # Said twice on purpose. The warning above is several screens back by now, and
  # an install that ends on an unqualified success line is remembered as one.
  # Repeated because a `curl | sh` scroll buries whatever appeared at the top,
  # and this is the one case where something real was left uncovered.
  if [ "$PINNED_OK" != 1 ] && [ "$UNSIGNED" = 1 ]; then
    info "Nothing verified this against a substituted release. See above."
  fi
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) info "Add it to your PATH:  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
  esac
}

main "$@"
