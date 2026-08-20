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

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT INT TERM

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

{
  printf '%s' "$BEGIN"
  cat <<'HDR'
 — gerado por scripts/installer.sh a partir do checksums.txt assinado.
# Nao edite a mao. Estes sao os digests dos artefatos que foram assinados, e a
# graca deles e viverem no historico do git, longe do host que serve o tarball.
HDR
  echo "PINNED_VERSION=\"$VERSION\""
  echo 'pinned_sum() {'
  echo '  case "$1" in'
  printf '%s' "$CASES"
  echo '  esac'
  echo '}'
  echo "$END"
} > "$WORK/block"

# awk, nao sed nem python. `sed -i` diverge entre GNU e BSD, e este repositorio
# ja perdeu uma noite com um `sed ... t;` que so falhava numa das duas
# plataformas da matriz. python3 seria uma dependencia nova do pipeline de
# release — o scripts/formula.sh e bash puro — e no macOS ele nem sempre esta la.
awk -v block="$WORK/block" -v begin="$BEGIN" -v end="$END" '
  index($0, begin) == 1 && !done { while ((getline line < block) > 0) print line; skip = 1; done = 1; next }
  skip && index($0, end) == 1 { skip = 0; next }
  !skip { print }
' "$SCRIPT" > "$WORK/out"

# O bloco tem de continuar la depois da troca. Um gerador que escreve um arquivo
# sem pino nenhum produz um instalador que cai no fallback em silencio — que e
# exatamente o comportamento correto do NAO fixado, e por isso passaria despercebido.
grep -qF "$BEGIN" "$WORK/out" || { echo "installer: a troca do bloco perdeu o marcador" >&2; exit 1; }
grep -qF "PINNED_VERSION=\"$VERSION\"" "$WORK/out" ||
  { echo "installer: a troca do bloco nao fixou $VERSION" >&2; exit 1; }
cat "$WORK/out" > "$SCRIPT"

echo "installer: $SCRIPT fixado em $VERSION" >&2
