#!/usr/bin/env bash
# Gate de cobertura — 202608072337 TESTING.md secao 3.
# Denominador: codigo deterministico em internal/ e pkg/.
# Excluidos, com justificativa na convencao:
#   cmd/**        wiring de main, sem logica
#   **/evals/**   caminhos mediados por modelo, atras de build tag
#   *_gen.go      codigo gerado
set -euo pipefail

PROFILE="${1:-coverage.out}"
MIN="${DCODE_COVERAGE_MIN:-90}"

[ -f "$PROFILE" ] || { echo "cobertura: perfil '$PROFILE' nao encontrado"; exit 1; }

FILTERED="$(mktemp)"
trap 'rm -f "$FILTERED"' EXIT
head -1 "$PROFILE" > "$FILTERED"
grep -v -E '/cmd/|/evals/|_gen\.go:' "$PROFILE" | tail -n +2 >> "$FILTERED" || true

TOTAL="$(go tool cover -func="$FILTERED" | tail -1 | awk '{print $3}' | tr -d '%')"

printf 'cobertura: %s%% (minimo %s%%)\n' "$TOTAL" "$MIN"
awk -v t="$TOTAL" -v m="$MIN" 'BEGIN { exit !(t+0 >= m+0) }' || {
  echo "FALHA: cobertura abaixo do gate"
  go tool cover -func="$FILTERED" | awk '$3+0 < 90 && $1 != "total:" { print "  " $0 }'
  exit 1
}
