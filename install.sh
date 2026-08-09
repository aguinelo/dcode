#!/usr/bin/env sh
# dcode installer.
#
# Verifies the release signature and the artifact checksum before installing.
# Either check failing aborts and removes everything that was downloaded —
# "installed, but unverified" is the worst of both worlds, because the user ends
# up with a binary and the impression that it went fine.
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
set -eu

REPO="aguinelo/dcode"
VERSION="${DCODE_VERSION:-latest}"
INSTALL_DIR="${DCODE_INSTALL_DIR:-$HOME/.local/bin}"
SKIP_VERIFY="${DCODE_SKIP_VERIFY:-false}"
SUPPORTED="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

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
    download "$base/checksums.txt.sig" "$WORK/checksums.txt.sig"
    download "$base/checksums.txt.pem" "$WORK/checksums.txt.pem"

    # 4. Verify the signature over the checksums file. One signature covers the
    #    whole release; per-artifact verification is the SHA-256 below.
    need cosign
    cosign verify-blob \
      --certificate "$WORK/checksums.txt.pem" \
      --signature "$WORK/checksums.txt.sig" \
      --certificate-identity-regexp "^https://github.com/$REPO/" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      "$WORK/checksums.txt" >/dev/null 2>&1 ||
      die "the release signature did not verify — nothing was installed"

    # 5. Verify this artifact against the signed list.
    want="$(grep " \*\{0,1\}$artifact\$" "$WORK/checksums.txt" | awk '{print $1}' | head -1)"
    [ -n "$want" ] || die "$artifact is not listed in checksums.txt"
    got="$(sha256_of "$WORK/$artifact")"
    [ "$want" = "$got" ] ||
      die "checksum mismatch for $artifact — expected $want, got $got. Nothing was installed"
  fi

  # 6. Extract and install.
  tar -xzf "$WORK/$artifact" -C "$WORK" || die "could not extract $artifact"
  [ -f "$WORK/dcode" ] || die "$artifact does not contain a dcode binary"
  chmod +x "$WORK/dcode"

  mkdir -p "$INSTALL_DIR" || die "could not create $INSTALL_DIR"
  mv "$WORK/dcode" "$INSTALL_DIR/dcode" || die "could not install into $INSTALL_DIR"

  # 7. Confirm the installed binary runs here, before claiming success.
  "$INSTALL_DIR/dcode" --version >/dev/null 2>&1 ||
    die "the installed binary does not run on this machine"

  info "Installed $("$INSTALL_DIR/dcode" --version) to $INSTALL_DIR/dcode"
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) info "Add it to your PATH:  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
  esac
}

main "$@"
