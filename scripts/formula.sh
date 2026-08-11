#!/usr/bin/env bash
# Gera a formula do Homebrew a partir do checksums.txt do proprio release.
#
# Derivada, nunca escrita a mao. A RN-1 diz "um artefato para todos os canais",
# e uma formula com um SHA-256 digitado e a forma mais facil de quebrar isso:
# passa em todo teste local, instala em todo lugar, e um dia aponta para um
# binario que ninguem assinou.
#
#   scripts/formula.sh <versao> <checksums.txt> [url-base]
set -euo pipefail

VERSION="${1:?versao, ex. 0.4.0}"
SUMS="${2:?caminho do checksums.txt}"
BASE="${3:-https://github.com/aguinelo/dcode/releases/download/v$VERSION}"

sum_for() {
  local name="$1"
  local line
  line="$(grep -E "[[:space:]]${name}\$" "$SUMS" || true)"
  [ -n "$line" ] || { echo "formula: $name nao esta em $SUMS" >&2; exit 1; }
  echo "$line" | awk '{print $1}'
}

DARWIN_ARM="dcode_${VERSION}_darwin_arm64.tar.gz"
DARWIN_AMD="dcode_${VERSION}_darwin_amd64.tar.gz"
LINUX_ARM="dcode_${VERSION}_linux_arm64.tar.gz"
LINUX_AMD="dcode_${VERSION}_linux_amd64.tar.gz"

# Resolvidos ANTES do heredoc, de proposito. Substituicao de comando dentro de
# um heredoc engole o codigo de saida, entao uma plataforma faltando escreveria
# uma formula com um SHA vazio em vez de parar — e o erro so apareceria na
# instalacao de alguem.
SHA_DARWIN_ARM="$(sum_for "$DARWIN_ARM")"
SHA_DARWIN_AMD="$(sum_for "$DARWIN_AMD")"
SHA_LINUX_ARM="$(sum_for "$LINUX_ARM")"
SHA_LINUX_AMD="$(sum_for "$LINUX_AMD")"

cat <<EOF
# Gerado por scripts/formula.sh a partir do checksums.txt do release.
# Nao edite a mao: o SHA-256 aqui e o do artefato assinado, por construcao.
class Dcode < Formula
  desc "Agentic coding harness for the terminal"
  homepage "https://github.com/aguinelo/dcode"
  version "$VERSION"
  license "MIT"

  on_macos do
    on_arm do
      url "$BASE/$DARWIN_ARM"
      sha256 "${SHA_DARWIN_ARM}"
    end
    on_intel do
      url "$BASE/$DARWIN_AMD"
      sha256 "${SHA_DARWIN_AMD}"
    end
  end

  on_linux do
    on_arm do
      url "$BASE/$LINUX_ARM"
      sha256 "${SHA_LINUX_ARM}"
    end
    on_intel do
      url "$BASE/$LINUX_AMD"
      sha256 "${SHA_LINUX_AMD}"
    end
  end

  def install
    bin.install "dcode"
  end

  test do
    assert_match "dcode", shell_output("#{bin}/dcode --version")
  end
end
EOF
