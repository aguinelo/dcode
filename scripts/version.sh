#!/usr/bin/env bash
# Deriva a próxima versão dos commits desde a última tag.
#
# Os commits deste repositório já seguem Conventional Commits — `feat:`, `fix:`,
# `docs:` — desde antes de alguém combinar isso. Derivar a versão deles é
# transformar uma convenção que já é praticada em regra que se executa, que é
# sempre melhor que a mesma regra escrita em prosa.
#
# Recusa quando não consegue derivar. Um commit que não casa com a convenção não
# vira "patch por segurança": vira erro, porque a alternativa é uma versão
# escolhida por um palpite silencioso.
set -euo pipefail

MAJOR_ZERO_NOTE='Antes de 1.0 uma quebra sobe MINOR, nao MAJOR: 0.x diz "ainda muda", e gastar o 1 na primeira quebra e o que faz projetos chegarem em 7.0 sem nada estavel.'

# Only release tags count. A tag that is not a version — a restore point, a
# marker — used to shadow the last release simply by being newer, and the build
# then named itself after it: `tui-v1-dev+411c237`. Naming a build after
# something that is not a version is the same defect as naming it after the
# version it left, which this script exists to prevent.
last="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"
if [ -z "$last" ]; then
  echo "versao: nao ha tag neste repositorio." >&2
  echo "A primeira e escolhida, nao derivada — nao ha de onde derivar." >&2
  echo "Escolha e crie: git tag -a v0.0.1 -m 'v0.0.1'" >&2
  exit 1
fi

# A tag tem de ter a forma que a derivacao pressupoe. Sem isto, uma tag como
# `v1.2` ou `release-3` passa pela leitura e produz uma aritmetica que parece ter
# funcionado — testado, e o resultado foi uma versao inventada com cara de certa.
if ! printf '%s' "$last" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "versao: a ultima tag e '$last', que nao tem a forma vMAJOR.MINOR.PATCH." >&2
  echo "Nao da para derivar de um numero que nao se sabe ler." >&2
  exit 1
fi

range="$last..HEAD"

# O pipeline de release deixa um commit na main DEPOIS de criar a tag: ele
# reescreve o bloco de digests do install.sh, e esses digests so existem depois
# dos artefatos estarem construidos e assinados.
#
# Conta-lo faria toda consulta pos-release responder "ha mudancas desde a tag"
# quando nada humano mudou, e a derivacao passaria a subir PATCH sozinha. E a
# forma que este repositorio nao para de encontrar: automacao deixando um rastro
# que outro mecanismo le como sinal.
#
# A isencao e do assunto EXATO que o pipeline escreve, nunca do prefixo. Isentar
# `chore(release):` inteiro daria a qualquer pessoa uma forma de nao ser contada.
PIN_SUBJECT='^chore\(release\): pin the installer to v[0-9]+\.[0-9]+\.[0-9]+$'
subjects="$(git log --format='%s' "$range" | grep -vE "$PIN_SUBJECT" || true)"
if [ -z "$subjects" ]; then
  echo "$last" # nada mudou; a versao e a que ja existe
  exit 0
fi

# Toda linha tem de casar, ou nao ha derivacao possivel.
bad="$(printf '%s\n' "$subjects" | grep -vE '^(feat|fix|chore|docs|refactor|test|perf|build|ci)(\([^)]+\))?!?: .+' || true)"
if [ -n "$bad" ]; then
  echo "versao: estes commits nao seguem a convencao, entao nao ha o que derivar:" >&2
  printf '  %s\n' $bad >&2
  exit 1
fi

kind=none
printf '%s\n' "$subjects" | grep -qE '^feat(\([^)]+\))?: ' && kind=minor
printf '%s\n' "$subjects" | grep -qE '^(feat|fix|chore|docs|refactor|test|perf|build|ci)(\([^)]+\))?!: ' && kind=breaking
git log --format='%B' "$range" | grep -q '^BREAKING CHANGE:' && kind=breaking
[ "$kind" = none ] && kind=patch

v="${last#v}"
IFS=. read -r ma mi pa <<<"$v"

case "$kind" in
  breaking)
    if [ "$ma" -eq 0 ]; then mi=$((mi + 1)); pa=0; else ma=$((ma + 1)); mi=0; pa=0; fi
    ;;
  minor) mi=$((mi + 1)); pa=0 ;;
  patch) pa=$((pa + 1)) ;;
esac

echo "v${ma}.${mi}.${pa}"
[ "$kind" = breaking ] && [ "$ma" -eq 0 ] && echo "$MAJOR_ZERO_NOTE" >&2
exit 0
