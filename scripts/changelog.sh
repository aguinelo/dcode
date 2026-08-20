#!/usr/bin/env bash
# Gera o esqueleto da seção de uma versão, a partir dos commits desde a última
# tag.
#
# Esqueleto, e não o texto. O que este repositório guarda de mais valioso nos
# changelogs é o **porquê** — "a recusa era honesta mas instruía o abandono" não
# está em nenhum assunto de commit e não sai de nenhum gerador. O que sai é a
# parte mecânica: quais PRs entraram, agrupados, com o número ao lado.
#
# Um gerador que tentasse escrever o porquê produziria uma frase plausível sobre
# uma decisão que ninguém tomou, que é pior que uma lista sem frase nenhuma.
set -euo pipefail

version="${1:-}"
if [ -z "$version" ]; then
  version="$(./scripts/version.sh)" || exit 1
fi

last="$(git describe --tags --abbrev=0 2>/dev/null || true)"
range="${last:+$last..}HEAD"

section() {
  local prefix="$1" title="$2"
  local body
  body="$(git log --format='%s' "$range" \
    | grep -E "^${prefix}(\([^)]+\))?!?: " \
    | sed -E "s/^${prefix}(\([^)]+\))?!?: //" \
    | sed -E 's/ \(#([0-9]+)\)$/ (#\1)/' \
    | sed 's/^/- **/; s/ (#/** (#/; s/$//' || true)"
  [ -z "$body" ] && return 0
  printf '\n### %s\n\n%s\n' "$title" "$body"
}

# O mes por extenso, em portugues, sem depender de locale: `date +%B` devolve
# "August" numa maquina em ingles e "agosto" noutra, e um changelog cujo idioma
# muda com a maquina de quem gera nao e um changelog, sao dois.
meses=(janeiro fevereiro marco abril maio junho julho agosto setembro outubro novembro dezembro)
mes="${meses[$(( 10#$(date +%m) - 1 ))]}"
printf '## %s — %s de %s de %s\n' "${version#v}" "$(date +%-d)" "$mes" "$(date +%Y)"

section feat     'Adicionado'
section fix      'Corrigido'
section refactor 'Mudado'
section perf     'Mudado'
section docs     'Documentação'
section test     'Testes'
section chore    'Manutenção'
section build    'Manutenção'
section ci       'Manutenção'

printf '\n> Esqueleto gerado por `scripts/changelog.sh`. **Cada linha ainda precisa\n'
printf '> do porquê** — o que mudou sai do commit, a razão de ter mudado não sai de\n'
printf '> lugar nenhum automaticamente.\n'
