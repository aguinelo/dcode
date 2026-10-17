# Toda decisão vai ao log com o porquê

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.p` §9.1, §13, §14; `.config` §1; `.i`, passo 4
e fechamento).

## O que mudou

O passo 4 da primeira fatia: o pacote novo `internal/orchestrate/` começa pelo log de decisões.
`OpenDecisionLog(dir, projeto)` abre `<orchestrator.dir>/<project-id>/decisions.jsonl` (diretório
`0700`, arquivo `0600`), e `Record(Decision)` escreve uma linha, faz `fsync` e só então devolve o
`seq`. `orchestrate.Dir` lê `orchestrator.dir`, que entra em `KnownKeys`. Ninguém abre o log ainda:
a orquestradora que o abre por projeto é o passo 5, e o comando que o lê sem daemon, o 11.

## O que o código decide onde a `.p` deixava a forma em aberto

- **`v`, `seq`, `at` e `project` são do log.** `Record` os preenche; um `project` que não é o do
  log é recusado, não sobrescrito. Quem chama não tem como errar o `seq`, e é o `seq` que a
  invariante protege.
- **`kind` fora da lista da §9.1 é recusado**, como `why` vazio (ou só de espaços). Um `kind`
  digitado errado é uma decisão que nenhum filtro do `dcode decisions` acha; o teste
  `TestTheKindsAreTheSpecsList` lê a lista da `.p`, então acrescentar um `kind` é mudar a spec e o
  código juntos.
- **`measured` e `data` são bytes JSON** (`json.RawMessage`) que quem chama codifica — um
  `GateResult`, um `Verdict`. O log devolve exatamente o que recebeu, que é o que o `--json` do
  passo 11 promete ("as linhas como estão").
- **O corte de 64 KiB conta o fim de linha** e escolhe o maior entre `why`, `claim`, `measured` e
  `data`, depois o seguinte se ainda não couber. `measured` ou `data` cortados viram uma string
  JSON com o texto do que ficou: meio objeto não é objeto, e um leitor que tenta decodificar o
  objeto falharia na linha inteira. Os ids não são cortados; decisão que nem com os quatro cortados
  cabe é erro, nunca linha acima do teto.
- **Escrita que falha não gasta `seq`.** Se a escrita ou o `fsync` falha, a linha parcial é
  desfeita (`Truncate` ao tamanho de antes) e o erro volta; a transição que dependia dela não
  acontece (§8). Se nem desfazer der certo, o arquivo pode terminar em meia linha, e emendar a
  próxima depois dela juntaria duas decisões numa linha: o log recusa toda decisão seguinte, dizendo
  por quê, até ser reaberto.
- **A linha rasgada por uma queda é cortada ao reabrir, e dita.** Uma queda no meio do `Record`
  deixa a última linha sem `\n`. O `Record` dela não voltou, então nada agiu sobre ela, e o `seq`
  dela não foi dado a ninguém: cortá-la é o que mantém o `seq` sem buraco e o arquivo legível linha
  a linha. Cortar calado seria perder uma linha em silêncio; `Torn` devolve os bytes cortados, e o
  passo 5 os põe no `daemon.restart`. A alternativa — terminar a linha rasgada com `\n` e deixá-la
  no arquivo — faria todo `dcode decisions` sair 1 para sempre depois da primeira queda, por uma
  linha que nunca foi decisão.
- **Linha completa ilegível, versão desconhecida, ou `seq` que não segue o anterior impede o log de
  abrir**, com o número da linha. Uma escrita numa chamada só, seguida de `fsync`, não produz isso;
  é um arquivo que este escritor não deixou, e continuar levaria o buraco ou a repetição adiante. O
  projeto não abre, e é o `dcode decisions` (passo 11) que lê o arquivo e diz onde.
- **"Fora de qualquer projeto que o daemon conduz"** é conferido ao abrir o log de um projeto, por
  ser o único lugar que conhece a raiz dele: `orchestrator.dir` dentro de `Project.Root` é recusado.
  O id do projeto precisa ser 24 hex antes de virar caminho.
- **`orchestrator.dir` relativa é erro** com a chave e o valor, nunca resolvida contra o diretório
  onde o daemon nasceu; sem raiz de estado não há default, e isso também é erro.

## Invariantes e configuração

A `.p` §13 passa a ser "Invariantes verificáveis", com as cinco linhas do log que têm teste, e
`internal/orchestrate/invariants_test.go` nomeia `orchestration` no `specguard.Check`; as outras
viram a §14, "previstas", e sobem com o passo que lhes der teste. A `.config` §1 separa "Lidas",
que abre pela variável (é a linha que os guardas de configuração leem), de "Por ler", que continua
abrindo pela chave. `orchestrator.dir` é a primeira lida; o guarda de fiação a dispensa da tabela de
sessões porque quem a lê é a orquestradora, que conduz sessões e não é uma, e passa a procurar a
leitura em `internal/orchestrate/`.

`OpenDecisionLog` e `Torn` não têm chamador fora dos testes até o passo 5; ficam dispensados em
`internal/specguard/unused_orchestrate_test.go`, como os nomes do passo 2, e saem de lá quando o
passo 5 os chamar.
