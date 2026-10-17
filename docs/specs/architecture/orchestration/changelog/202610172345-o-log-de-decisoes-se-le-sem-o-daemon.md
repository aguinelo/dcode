# O log de decisões se lê sem o daemon

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.p` §9.2, §13, §14; `.i`, passo 11).

## O que mudou

O passo 11 da primeira fatia: `dcode decisions` imprime o `decisions.jsonl` de um projeto, lendo o
arquivo direto. `--project` (padrão: o diretório corrente) é resolvido por
`orchestrate.ResolveProject`, o diretório por `orchestrate.Dir` sobre a configuração que o daemon lê,
e o caminho é `<orchestrator.dir>/<project-id>/decisions.jsonl`. Filtros `--child`, `--kind` (lista
separada por vírgula), `--since` (duração de Go, `90m`, `24h`); `--follow` espera linhas novas até a
interrupção; `--json` devolve as linhas byte a byte. Texto: uma linha por decisão, como a `.p` §9.2
mostra, com a data na frente quando não é de hoje, e `claim:` ao lado do medido quando há.

## O que o código decide onde a `.p` deixava a forma em aberto

- **`ResolveProject` é a regra única do `Project.ID`.** `git rev-parse --git-common-dir` a partir
  da pasta dada; a raiz é o pai do `.git` comum (ou o próprio diretório comum, num repositório
  bare), com links resolvidos; o id são os 24 primeiros hex do sha256 dela. Raiz, subpasta, link e
  worktree ligada dão o mesmo projeto. O `POST /v1/projects` do passo 5 resolve pela mesma função:
  dois lugares que calculam o id são dois ids que um dia divergem, e o leitor abriria um log que
  ninguém escreve. `GIT_DIR` e afins do ambiente são ignorados — quem decide é a pasta.
- **O leitor é `orchestrate.LogReader`, não o `scan` do `OpenDecisionLog`.** O `scan` recusa o log
  inteiro na primeira linha ruim, que é o certo para quem vai escrever depois dela; quem lê para
  entender o que deu errado precisa das linhas depois da ruim. O `LogReader` devolve toda linha
  completa com o número, o erro dela quando não lê (JSON inválido ou `v` diferente de 1) e se o
  `seq` segue o da última linha legível. É o leitor que os cenários do passo 12 têm à mão para conferir os `kind`.
- **Toda falha é dita e dá saída 1, depois de imprimir o resto.** Linha ilegível, `seq` com buraco
  ou repetido, e última linha sem fim de linha (escrita em curso, ou cortada por uma queda) vão a
  stderr com `caminho:linha`, e o comando termina com "o log tem N problema(s)". Log que não existe
  é erro com o caminho onde foi procurado, nunca uma lista vazia que se lê como "nenhuma decisão".
- **O comando nunca escreve no log.** Nem para cortar a linha rasgada: cortar é da orquestradora ao
  reabrir (`Torn`), que é quem sabe que nenhuma escrita está em curso. Um leitor que cortasse
  poderia cortar a linha que o daemon está escrevendo agora.
- **`--follow` só imprime linha completa.** A linha pela metade espera o fim de linha; o leitor
  olha o arquivo a cada 250 ms.

## Invariantes

Sobe da §14 para a §13, reescrita para dizer também que o comando não escreve:
"`dcode decisions` lê o log sem daemon e nunca o escreve; linha ilegível é dita com o número, e o
resto é impresso" — `TestAnUnreadableLineIsSaidWithItsNumberAndTheRestIsPrinted`, em `cmd/dcode/`,
que o `invariants_test.go` do pacote passa a ler junto.
