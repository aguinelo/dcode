#!/usr/bin/env bash
# Fixa no install.sh os digests do release que ele instala.
#
# Mesma razao do scripts/formula.sh, no outro canal: um SHA-256 digitado passa
# em todo teste local e um dia aponta para um binario que ninguem assinou. Aqui
# ha um motivo a mais.
#
# O checksums.txt vem do MESMO host que o tarball, entao sozinho ele pega
# download corrompido e nao pega release substituido. Fixado aqui, o valor
# esperado passa a viver no historico do git: um asset de release pode ser
# trocado sem deixar rastro publico, um digest na main so muda por commit —
# visivel no log, no diff e atribuido. E o que torna a assinatura opcional
# (#222) segura por construcao e nao por promessa.
#
# O install.sh NAO e gerado por inteiro. Ele continua um script real, editavel
# a mao, e este gerador reescreve apenas o bloco entre os marcadores. Um
# template dentro de heredoc seria uma segunda copia do script para manter em
# sincronia, e este repositorio ja sabe o que acontece com a segunda copia.
#
#   scripts/installer.sh <versao> <checksums.txt> [install.sh]
set -euo pipefail

VERSION="${1:?versao, ex. 0.0.2}"
SUMS="${2:?caminho do checksums.txt}"
SCRIPT="${3:-install.sh}"

BEGIN='# BEGIN PINNED'
END='# END PINNED'

grep -qF "$BEGIN" "$SCRIPT" || { echo "installer: $SCRIPT nao tem $BEGIN" >&2; exit 1; }
grep -qF "$END"   "$SCRIPT" || { echo "installer: $SCRIPT nao tem $END" >&2; exit 1; }

sum_for() {
  local name="$1" line
  line="$(grep -E "[[:space:]]\*?${name}\$" "$SUMS" || true)"
  [ -n "$line" ] || { echo "installer: $name nao esta em $SUMS" >&2; exit 1; }
  echo "$line" | awk '{print $1}'
}

# Resolvidos ANTES de montar o bloco, pelo motivo que o formula.sh ja documenta:
# substituicao de comando dentro de heredoc engole o codigo de saida, e uma
# plataforma faltando escreveria um digest vazio em vez de parar.
PLATFORMS="darwin_amd64 darwin_arm64 linux_amd64 linux_arm64"
CASES=""
for p in $PLATFORMS; do
  name="dcode_${VERSION}_${p}.tar.gz"
  CASES="${CASES}    ${name}) echo $(sum_for "$name") ;;"$'\n'
done

BLOCK="$BEGIN — gerado por scripts/installer.sh a partir do checksums.txt assinado.
# Nao edite a mao. Estes sao os digests dos artefatos que foram assinados, e a
# graca deles e viverem no historico do git, longe do host que serve o tarball.
PINNED_VERSION=\"$VERSION\"
pinned_sum() {
  case \"\$1\" in
${CASES}  esac
}
$END"

python3 - "$SCRIPT" "$BEGIN" "$END" "$BLOCK" <<'PY'
import re, sys
path, begin, end, block = sys.argv[1:5]
s = open(path).read()
pat = re.compile(re.escape(begin) + r".*?" + re.escape(end), re.S)
assert pat.search(s), "bloco nao encontrado"
open(path, "w").write(pat.sub(lambda _: block, s, count=1))
PY

echo "installer: $SCRIPT fixado em $VERSION" >&2
