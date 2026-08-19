#!/usr/bin/env bash
# Une os perfis de cobertura da matriz de CI num só, escrito na saída padrão.
#
# A convenção de testes já declarava a exclusão — "pacote específico de SO, fora
# da plataforma do runner: não é executável ali; coberto na matriz de CI da
# plataforma correspondente" — e nada a cumpria. Cada job rodava o gate sozinho,
# então um ramo que só o macOS executa contava como descoberto no Ubuntu. É a
# forma que este repositório não para de encontrar em si mesmo: algo declarado
# que um lado lê e nenhum lado escreve.
#
# Unir é concatenar, e isso não é atalho: `go tool cover` soma as ocorrências do
# mesmo bloco. O repositório já depende disso — com -coverpkg=./... cada binário
# de teste emite um perfil de TODOS os pacotes, então o mesmo bloco já aparece
# várias vezes dentro de um único perfil. O awk por pacote de coverage.sh faz a
# mesma leitura: coberto se QUALQUER ocorrência o executou.
#
# O que não se pode misturar é modo. `set` grava 0 ou 1 e `atomic` grava
# contagem; somar os dois produz um número que não significa nada. Perfis de
# modos diferentes são recusados em vez de somados.
set -euo pipefail

[ "$#" -gt 0 ] || { echo "uso: merge-coverage.sh <perfil> [perfil...]" >&2; exit 1; }

MODE=""
for p in "$@"; do
  [ -f "$p" ] || { echo "cobertura: perfil '$p' nao encontrado" >&2; exit 1; }
  m="$(head -1 "$p")"
  case "$m" in
    mode:*) ;;
    *) echo "cobertura: '$p' nao comeca com uma linha de modo" >&2; exit 1 ;;
  esac
  if [ -z "$MODE" ]; then
    MODE="$m"
  elif [ "$m" != "$MODE" ]; then
    echo "cobertura: '$p' e '$m', mas os anteriores sao '$MODE'" >&2
    exit 1
  fi
done

printf '%s\n' "$MODE"
for p in "$@"; do
  tail -n +2 "$p"
done
