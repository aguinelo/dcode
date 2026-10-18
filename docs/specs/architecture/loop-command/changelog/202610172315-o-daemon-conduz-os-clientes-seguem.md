# O daemon conduz; a TUI e o desktop seguem

**Data:** 2026-10-17
**Specs afetadas:** `202608252000-loop-command` (`.r` estado, RN-2, RN-3; `.p` §1, §2, §8; `.i` estado, pré-requisitos, Passos 3 e 4, marco; índices de changelog dos quatro)

Correção de documentação. Nenhum comportamento muda: o texto alcança o que os
changelogs `202610141200` e `202610141700` já tinham feito no código.

## O que mudou

- **Os blocos de estado.** O `.r` abria com *"Estado, 2026-08-26. Nada no
  produto chama esta família… `/loop` ainda não é digitável"*, e o `.p` §1 com
  *"Nada no produto chama este pacote"* — enquanto o `.p` §2, logo abaixo,
  descrevia o `Runner` do daemon conduzindo o loop. Os dois passam a dizer o que
  é: **o daemon conduz; a TUI e o desktop seguem.** O cliente reconhece `/loop`
  e manda o argumento como foi digitado (`POST /v1/loops`); o `Runner` faz a
  sequência inteira; os clientes seguem `GET /v1/loops/events`.
- **RN-2.** Dizia que a sessão dedicada ganha o nome `loop-{basename}-{timestamp}`
  e é encerrada no fim. O daemon abre uma sessão por pasta com o caminho da spec
  no pedido; ela tem o identificador de qualquer sessão, fica na lista, e o
  loop guarda qual é de qual pasta.
- **RN-3.** Dizia que o cliente traduz o argumento em `LoopSpec{Path}` e popula
  `Config.Done`. O cliente só reconhece o prefixo e manda o argumento; a
  gramática e o `Config.Done` são do daemon. A regra de fundo — o texto
  `/loop ...` nunca entra no histórico — não muda.
- **`.i`.** O pré-requisito citava um evento `loop.session.created` "emitido no
  Passo 4", que nunca existiu: a sessão que o loop abre chega no próprio
  `LoopRun`. O Passo 3 tinha caixas abertas para o que está feito, e apontava
  `internal/client/`, que não existe. As invariantes "que nascem com o Passo 3"
  descreviam a sessão como o cliente a montaria; ficam como desenho registrado,
  porque o `Runner` não encerra a sessão da pasta. O marco executável foi
  atingido.
- **A tecla que para o loop.** O `.p` §2 dizia que interromper o turno do loop
  na TUI o para, sem dizer como. É `ctrl+c`, a tecla que interrompe qualquer
  turno ali (`internal/tui/program.go`), e o que ela pede ao daemon é
  `stop now`. Não é `Esc`.
- **Índices de changelog.** O `.r` e o `.i` listavam só a entrada de criação; o
  `.p` não listava as quatro de 2026-09-01 nem a `202610161800`.

## O que a correção achou e não conserta

A TUI declara um texto `Interrupt` — *"esc interrupts"* / *"esc interrompe"*,
em `internal/tui/lang.go` — que nada lê, e que nomeia a tecla errada: o que
interrompe um turno, e o loop, é `ctrl+c`, e o texto que o produto mostra é o
`WorkingInterrupt` (*"^C interrompe"*). Fica registrado como defeito de código.
