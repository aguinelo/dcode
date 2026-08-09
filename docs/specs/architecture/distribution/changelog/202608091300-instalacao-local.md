# Instalação local e origem do binário

**Data:** 2026-08-09
**Specs afetadas:** `202608072352-distribution` (`.p`, `.config`)

## O que mudou

Duas coisas que a spec não previa porque só tratava do caminho publicado.

**`make install`** instala o build local em `DCODE_INSTALL_DIR`
(default `$HOME/.local/bin`, o mesmo do `install.sh`). Depende do `check`:
instalar algo que não passou no gate é como um defeito local vira "o dcode
quebrou". `make install-fast` pula o gate, para o laço de edição.

**O binário declara a própria origem**, em `version.Source`, injetada no link.
Só o pipeline de release injeta `release`; `make install` injeta `local`;
qualquer outra coisa — `go install`, por exemplo — fica sem valor e é tratada
como local.

## Por que mudou

**Não havia alvo de instalação.** Rodar era `./bin/dcode` ou um script de
scratchpad, que é gambiarra com outro nome.

**E um build local se apresentava idêntico a um release.** Duas consequências,
ambas caras:

1. Relato de bug contra "0.1.0" que nunca foi o 0.1.0 publicado.
2. `dcode update` substituiria um build da árvore de trabalho — normalmente **à
   frente** da última tag — pelo release mais recente. Downgrade vestindo a
   palavra "atualizar", e a única pista seria o número diminuindo.

## Impacto

- `version.Source` e `version.IsRelease()`.
- `--version` marca `local build`.
- `update.Apply` devolve `ErrLocalBuild` sem `--force`, e o comando recusa
  **antes** de consultar a rede, para não reportar "nenhum release encontrado"
  quando o motivo verdadeiro é outro.
- O workflow de release passa a injetar `Source=release`. **Sem isso todo
  release publicado se recusaria a atualizar** — o guard teria trancado
  exatamente o caminho que existe para funcionar.

## Alternativa descartada

Inferir "é local" do formato da versão, procurando `-dev`. Descartada porque
`Version` é injetada no link e passável na linha de comando: a decisão de
sobrescrever ou não um binário não deve depender de casar texto que qualquer um
escolhe.
