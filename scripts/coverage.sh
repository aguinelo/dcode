#!/usr/bin/env bash
# Gate de cobertura — 202608072337 TESTING.md secao 3.
# Denominador: codigo deterministico em internal/ e pkg/.
# Excluidos, com justificativa na convencao:
#   cmd/**        montagem de dependencia; a LOGICA dali tem teste proprio
#   **/evals/**   caminhos mediados por modelo, atras de build tag
#   *_gen.go      codigo gerado
set -euo pipefail

PROFILE="${1:-coverage.out}"
# Agregado e por pacote sao 90, e sao o mesmo numero de proposito.
#
# O agregado esteve em 95 por um tempo, no valor exato que a arvore media. Um
# gate colado no medido reprova por geografia e por arredondamento, nao por
# codigo sem teste, e foi o que aconteceu: tres PRs numa noite, nenhum deles por
# falta de teste. Volta para 90, que e o que os seis .i.spec.md pedem.
#
# O que muda de natureza e o por pacote. Ele era impresso e ignorado — regra
# declarada que um lado le e nenhum lado escreve, a forma que este repositorio
# nao para de encontrar em si mesmo. Com o agregado em 90 ele passa a ser o
# unico piso que morde: media boa escondendo pacote fraco e exatamente o que
# ninguem procura.
#
# Percentual nao cobra cenario critico. Isso e invariante em spec, e quem cobra
# e o specguard — ver docs/conventions/TESTING.md secao 3.
MIN="${DCODE_COVERAGE_MIN:-90}"
MIN_PKG="${DCODE_COVERAGE_MIN_PKG:-90}"

[ -f "$PROFILE" ] || { echo "cobertura: perfil '$PROFILE' nao encontrado"; exit 1; }

FILTERED="$(mktemp)"
trap 'rm -f "$FILTERED"' EXIT
head -1 "$PROFILE" > "$FILTERED"
grep -v -E '/cmd/|/evals/|_gen\.go:' "$PROFILE" | tail -n +2 >> "$FILTERED" || true

TOTAL="$(go tool cover -func="$FILTERED" | tail -1 | awk '{print $3}' | tr -d '%')"

printf 'cobertura: %s%% (minimo %s%%)\n' "$TOTAL" "$MIN"

# Por pacote, alem do agregado. Cinco .i.spec.md exigem >= 90% no PROPRIO
# pacote, e a media escondia tres abaixo disso — internal/loop, internal/tools
# e internal/credential — atras de outros em 95%+.
#
# A media nao e a regra que as specs escrevem, e pacote fraco atras de media boa
# e exatamente o que ninguem procura. Contado do perfil: cada linha traz o
# numero de statements e quantas vezes rodaram.
# Com -coverpkg=./... cada binario de teste emite um perfil de TODOS os
# pacotes, entao o mesmo bloco aparece varias vezes e quase sempre zerado nos
# binarios que nao o exercitaram. Somar as ocorrencias dava 5%. Cada bloco conta
# uma vez, coberto se QUALQUER ocorrencia o executou.
PER_PKG="$(tail -n +2 "$FILTERED" | awk '
  { if ($3 > 0) hit[$1] = 1; stmt[$1] = $2 }
  END {
    for (b in stmt) {
      split(b, a, ":"); path = a[1]
      sub(/\/[^\/]*$/, "", path)
      total[path] += stmt[b]
      if (b in hit) done[path] += stmt[b]
    }
    for (p in total) if (total[p] > 0) printf "%s %.1f\n", p, 100 * done[p] / total[p]
  }
' | sort)"

BELOW="$(printf '%s\n' "$PER_PKG" | awk -v m="$MIN_PKG" '$2+0 < m+0 { printf "  %s %s%%\n", $1, $2 }')"
if [ -n "$BELOW" ]; then
  printf 'FALHA: pacotes abaixo de %s%% por conta propria:\n%s\n' "$MIN_PKG" "$BELOW"
  go tool cover -func="$FILTERED" | awk -v m="$MIN_PKG" '$3+0 < m+0 && $1 != "total:" { print "  " $0 }'
  exit 1
fi
awk -v t="$TOTAL" -v m="$MIN" 'BEGIN { exit !(t+0 >= m+0) }' || {
  echo "FALHA: cobertura agregada abaixo do gate"
  go tool cover -func="$FILTERED" | awk -v m="$MIN_PKG" '$3+0 < m+0 && $1 != "total:" { print "  " $0 }'
  exit 1
}
