---
name: release
when_to_use: preparar ou publicar uma release, cortar uma tag de versão
triggers: release, tag, publicar
---

# Preparar uma release

A ordem importa, e o segundo passo é o que ninguém adivinha.

1. Atualize `internal/version/version.go` com a nova versão.
2. **Antes de cortar a tag**, registre a versão em `RELEASING.md` na raiz,
   acrescentando uma linha `## <versão>` no topo. O pipeline lê esse arquivo
   para decidir o que publicar, e uma tag sem a linha correspondente é
   publicada como vazia.
3. Só então corte a tag.
